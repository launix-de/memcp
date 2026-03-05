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
			var d9 scm.JITValueDesc
			_ = d9
			var r5 unsafe.Pointer
			_ = r5
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
			var d495 scm.JITValueDesc
			_ = d495
			var d496 scm.JITValueDesc
			_ = d496
			var d497 scm.JITValueDesc
			_ = d497
			var d498 scm.JITValueDesc
			_ = d498
			var d499 scm.JITValueDesc
			_ = d499
			var d500 scm.JITValueDesc
			_ = d500
			var d501 scm.JITValueDesc
			_ = d501
			var d502 scm.JITValueDesc
			_ = d502
			var d503 scm.JITValueDesc
			_ = d503
			var d504 scm.JITValueDesc
			_ = d504
			var d505 scm.JITValueDesc
			_ = d505
			var d506 scm.JITValueDesc
			_ = d506
			var d507 scm.JITValueDesc
			_ = d507
			var d508 scm.JITValueDesc
			_ = d508
			var d509 scm.JITValueDesc
			_ = d509
			var d510 scm.JITValueDesc
			_ = d510
			var d511 scm.JITValueDesc
			_ = d511
			var d512 scm.JITValueDesc
			_ = d512
			var d513 scm.JITValueDesc
			_ = d513
			var d514 scm.JITValueDesc
			_ = d514
			var d515 scm.JITValueDesc
			_ = d515
			var d516 scm.JITValueDesc
			_ = d516
			var d517 scm.JITValueDesc
			_ = d517
			var d518 scm.JITValueDesc
			_ = d518
			var d519 scm.JITValueDesc
			_ = d519
			var d520 scm.JITValueDesc
			_ = d520
			var d521 scm.JITValueDesc
			_ = d521
			var d522 scm.JITValueDesc
			_ = d522
			var d523 scm.JITValueDesc
			_ = d523
			var d524 scm.JITValueDesc
			_ = d524
			var d525 scm.JITValueDesc
			_ = d525
			var d526 scm.JITValueDesc
			_ = d526
			var d527 scm.JITValueDesc
			_ = d527
			var d528 scm.JITValueDesc
			_ = d528
			var d529 scm.JITValueDesc
			_ = d529
			var d530 scm.JITValueDesc
			_ = d530
			var d531 scm.JITValueDesc
			_ = d531
			var d532 scm.JITValueDesc
			_ = d532
			var d533 scm.JITValueDesc
			_ = d533
			var d534 scm.JITValueDesc
			_ = d534
			var d535 scm.JITValueDesc
			_ = d535
			var d536 scm.JITValueDesc
			_ = d536
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
			var bbs [9]scm.BBDescriptor
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
			ctx.ReclaimUntrackedRegs()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
				r2 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r2, thisptr.Reg, off)
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
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d1.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl10)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl3)
			ps4 := scm.PhiState{General: true}
			ps4.OverlayValues = make([]scm.JITValueDesc, 2)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps5 := scm.PhiState{General: true}
			ps5.OverlayValues = make([]scm.JITValueDesc, 2)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			snap6 := d0
			snap7 := d1
			alloc8 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps5)
			}
			ctx.RestoreAllocState(alloc8)
			d0 = snap6
			d1 = snap7
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
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d9 = idxInt
			_ = d9
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 = ctx.EmitSubRSP32Fixup()
			_ = r5
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			lbl12 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d9.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.EmitMovRegReg(r6, d9.Reg)
				ctx.EmitShlRegImm8(r6, 32)
				ctx.EmitShrRegImm8(r6, 32)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d11)
			}
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r7 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r7, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d12)
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.EmitMovRegReg(r8, d12.Reg)
				ctx.EmitShlRegImm8(r8, 56)
				ctx.EmitShrRegImm8(r8, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d13)
			}
			ctx.FreeDesc(&d12)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			var d14 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() * d13.Imm.Int())}
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.EmitImulInt64(scratch, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(scratch, d11.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r9 := ctx.AllocRegExcept(d11.Reg, d13.Reg)
				ctx.EmitMovRegReg(r9, d11.Reg)
				ctx.EmitImulInt64(r9, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d14)
			}
			if d14.Loc == scm.LocReg && d11.Loc == scm.LocReg && d14.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d13)
			var d15 scm.JITValueDesc
			r10 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r10, uint64(dataPtr))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10, StackOff: int32(sliceLen)}
				ctx.BindReg(r10, &d15)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.EmitMovRegMem(r10, thisptr.Reg, off)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d15)
			}
			ctx.BindReg(r10, &d15)
			ctx.EnsureDesc(&d14)
			var d16 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r11, d14.Reg)
				ctx.EmitShrRegImm8(r11, 6)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d16)
			}
			if d16.Loc == scm.LocReg && d14.Loc == scm.LocReg && d16.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d16)
			r12 := ctx.AllocReg()
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d15)
			if d16.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r12, uint64(d16.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r12, d16.Reg)
				ctx.EmitShlRegImm8(r12, 3)
			}
			if d15.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
				ctx.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r12, d15.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d17)
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d14)
			var d18 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r14, d14.Reg)
				ctx.EmitAndRegImm32(r14, 63)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d18)
			}
			if d18.Loc == scm.LocReg && d14.Loc == scm.LocReg && d18.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) << uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r15 := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(r15, d17.Reg)
				ctx.EmitShlRegImm8(r15, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d19)
			} else {
				{
					shiftSrc := d17.Reg
					r16 := ctx.AllocRegExcept(d17.Reg)
					ctx.EmitMovRegReg(r16, d17.Reg)
					shiftSrc = r16
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d17.Loc == scm.LocReg && d19.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d18)
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r17 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r17, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d20)
			}
			d21 = d20
			ctx.EnsureDesc(&d21)
			if d21.Loc != scm.LocImm && d21.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			if d21.Loc == scm.LocImm {
				if d21.Imm.Bool() {
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl13)
				} else {
					ctx.MarkLabel(lbl16)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d22 = d19
			if d22.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, int32(bbs[2].PhiBase)+int32(0))
			if d19.Loc == scm.LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
					ctx.EmitJmp(lbl14)
				}
			} else {
				ctx.EmitCmpRegImm32(d21.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl15)
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl15)
				ctx.EmitJmp(lbl13)
				ctx.MarkLabel(lbl16)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d23 = d19
			if d23.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			ctx.EmitStoreToStack(d23, int32(bbs[2].PhiBase)+int32(0))
			if d19.Loc == scm.LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
				ctx.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d20)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl14)
			ctx.ResolveFixups()
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r18 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r18, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.EmitMovRegReg(r19, d24.Reg)
				ctx.EmitShlRegImm8(r19, 56)
				ctx.EmitShrRegImm8(r19, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d25)
			}
			ctx.FreeDesc(&d24)
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d25)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(r20, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d27)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.EmitSubInt64(scratch, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(scratch, d26.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else {
				r21 := ctx.AllocRegExcept(d26.Reg, d25.Reg)
				ctx.EmitMovRegReg(r21, d26.Reg)
				ctx.EmitSubInt64(r21, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r22, d10.Reg)
				ctx.EmitShrRegImm8(r22, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d28)
			} else {
				{
					shiftSrc := d10.Reg
					r23 := ctx.AllocRegExcept(d10.Reg)
					ctx.EmitMovRegReg(r23, d10.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d27.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d27.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d27.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d28)
				}
			}
			if d28.Loc == scm.LocReg && d10.Loc == scm.LocReg && d28.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d27)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			if d28.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r24, d28)
			}
			ctx.EmitJmp(lbl12)
			bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl13)
			ctx.ResolveFixups()
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d14)
			var d29 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r25, d14.Reg)
				ctx.EmitAndRegImm32(r25, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d29)
			}
			if d29.Loc == scm.LocReg && d14.Loc == scm.LocReg && d29.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r26 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d30)
			}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d30.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.EmitMovRegReg(r27, d30.Reg)
				ctx.EmitShlRegImm8(r27, 56)
				ctx.EmitShrRegImm8(r27, 56)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d31)
			}
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(r28, d29.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d32)
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
				ctx.BindReg(d31.Reg, &d32)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.EmitAddInt64(scratch, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegReg(scratch, d29.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else {
				r29 := ctx.AllocRegExcept(d29.Reg, d31.Reg)
				ctx.EmitMovRegReg(r29, d29.Reg)
				ctx.EmitAddInt64(r29, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d32)
			}
			if d32.Loc == scm.LocReg && d29.Loc == scm.LocReg && d32.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d32.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitCmpRegImm32(d32.Reg, 64)
				ctx.EmitSetcc(r30, scm.CcA)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d33)
			}
			ctx.FreeDesc(&d32)
			d34 = d33
			ctx.EnsureDesc(&d34)
			if d34.Loc != scm.LocImm && d34.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			lbl19 := ctx.ReserveLabel()
			if d34.Loc == scm.LocImm {
				if d34.Imm.Bool() {
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl17)
				} else {
					ctx.MarkLabel(lbl19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d35 = d19
			if d35.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			ctx.EmitStoreToStack(d35, int32(bbs[2].PhiBase)+int32(0))
			if d19.Loc == scm.LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
					ctx.EmitJmp(lbl14)
				}
			} else {
				ctx.EmitCmpRegImm32(d34.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl18)
				ctx.EmitJmp(lbl19)
				ctx.MarkLabel(lbl18)
				ctx.EmitJmp(lbl17)
				ctx.MarkLabel(lbl19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocReg {
				ctx.ProtectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.ProtectReg(d19.Reg)
				ctx.ProtectReg(d19.Reg2)
			}
			d36 = d19
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, int32(bbs[2].PhiBase)+int32(0))
			if d19.Loc == scm.LocReg {
				ctx.UnprotectReg(d19.Reg)
			} else if d19.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d19.Reg)
				ctx.UnprotectReg(d19.Reg2)
			}
				ctx.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d33)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl17)
			ctx.ResolveFixups()
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d14)
			var d37 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r31, d14.Reg)
				ctx.EmitShrRegImm8(r31, 6)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d37)
			}
			if d37.Loc == scm.LocReg && d14.Loc == scm.LocReg && d37.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitMovRegReg(scratch, d37.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			}
			if d38.Loc == scm.LocReg && d37.Loc == scm.LocReg && d38.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d38)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d15)
			if d38.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r32, uint64(d38.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r32, d38.Reg)
				ctx.EmitShlRegImm8(r32, 3)
			}
			if d15.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
				ctx.EmitAddInt64(r32, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r32, d15.Reg)
			}
			r33 := ctx.AllocRegExcept(r32)
			ctx.EmitMovRegMem(r33, r32, 0)
			ctx.FreeReg(r32)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d39)
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d14)
			var d40 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitMovRegReg(r34, d14.Reg)
				ctx.EmitAndRegImm32(r34, 63)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d40)
			}
			if d40.Loc == scm.LocReg && d14.Loc == scm.LocReg && d40.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() - d40.Imm.Int())}
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(r35, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d42)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.EmitSubInt64(scratch, d40.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegReg(scratch, d41.Reg)
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d40.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else {
				r36 := ctx.AllocRegExcept(d41.Reg, d40.Reg)
				ctx.EmitMovRegReg(r36, d41.Reg)
				ctx.EmitSubInt64(r36, d40.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d42)
			}
			if d42.Loc == scm.LocReg && d41.Loc == scm.LocReg && d42.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d39.Imm.Int()) >> uint64(d42.Imm.Int())))}
			} else if d42.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitMovRegReg(r37, d39.Reg)
				ctx.EmitShrRegImm8(r37, uint8(d42.Imm.Int()))
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d43)
			} else {
				{
					shiftSrc := d39.Reg
					r38 := ctx.AllocRegExcept(d39.Reg)
					ctx.EmitMovRegReg(r38, d39.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d42.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d42.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d42.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d43)
				}
			}
			if d43.Loc == scm.LocReg && d39.Loc == scm.LocReg && d43.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() | d43.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d43.Reg}
				ctx.BindReg(d43.Reg, &d44)
			} else if d43.Loc == scm.LocImm && d43.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegReg(r39, d19.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d44)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.EmitOrInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegReg(r40, d19.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r40, int32(d43.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.EmitOrInt64(r40, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d44)
			} else {
				r41 := ctx.AllocRegExcept(d19.Reg, d43.Reg)
				ctx.EmitMovRegReg(r41, d19.Reg)
				ctx.EmitOrInt64(r41, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d44)
			}
			if d44.Loc == scm.LocReg && d19.Loc == scm.LocReg && d44.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d43)
			ctx.EnsureDesc(&d44)
			if d44.Loc == scm.LocReg {
				ctx.ProtectReg(d44.Reg)
			} else if d44.Loc == scm.LocRegPair {
				ctx.ProtectReg(d44.Reg)
				ctx.ProtectReg(d44.Reg2)
			}
			d45 = d44
			if d45.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d45)
			ctx.EmitStoreToStack(d45, int32(bbs[2].PhiBase)+int32(0))
			if d44.Loc == scm.LocReg {
				ctx.UnprotectReg(d44.Reg)
			} else if d44.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d44.Reg)
				ctx.UnprotectReg(d44.Reg2)
			}
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl12)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d46)
			ctx.BindReg(r24, &d46)
			if r3 { ctx.UnprotectReg(r4) }
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d46.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.EmitMovRegReg(r42, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d47)
			}
			ctx.FreeDesc(&d46)
			var d48 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r43 := ctx.AllocReg()
				ctx.EmitMovRegMem(r43, thisptr.Reg, off)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d48)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() + d48.Imm.Int())}
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r44, d47.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d49)
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d48.Reg}
				ctx.BindReg(d48.Reg, &d49)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.EmitAddInt64(scratch, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(scratch, d47.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else {
				r45 := ctx.AllocRegExcept(d47.Reg, d48.Reg)
				ctx.EmitMovRegReg(r45, d47.Reg)
				ctx.EmitAddInt64(r45, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d49)
			}
			if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d49.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.EmitMovRegReg(r46, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d50)
			}
			ctx.FreeDesc(&d49)
			var d51 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r47, thisptr.Reg, off)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d51)
			}
			d52 = d51
			ctx.EnsureDesc(&d52)
			if d52.Loc != scm.LocImm && d52.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d52.Loc == scm.LocImm {
				if d52.Imm.Bool() {
			ps53 := scm.PhiState{General: ps.General}
			ps53.OverlayValues = make([]scm.JITValueDesc, 53)
			ps53.OverlayValues[0] = d0
			ps53.OverlayValues[1] = d1
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
			ps53.OverlayValues[51] = d51
			ps53.OverlayValues[52] = d52
					return bbs[5].RenderPS(ps53)
				}
			ps54 := scm.PhiState{General: ps.General}
			ps54.OverlayValues = make([]scm.JITValueDesc, 53)
			ps54.OverlayValues[0] = d0
			ps54.OverlayValues[1] = d1
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
			ps54.OverlayValues[51] = d51
			ps54.OverlayValues[52] = d52
				return bbs[4].RenderPS(ps54)
			}
			if !ps.General {
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl20 := ctx.ReserveLabel()
			lbl21 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d52.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl20)
			ctx.EmitJmp(lbl21)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl6)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl5)
			ps55 := scm.PhiState{General: true}
			ps55.OverlayValues = make([]scm.JITValueDesc, 53)
			ps55.OverlayValues[0] = d0
			ps55.OverlayValues[1] = d1
			ps55.OverlayValues[9] = d9
			ps55.OverlayValues[10] = d10
			ps55.OverlayValues[11] = d11
			ps55.OverlayValues[12] = d12
			ps55.OverlayValues[13] = d13
			ps55.OverlayValues[14] = d14
			ps55.OverlayValues[15] = d15
			ps55.OverlayValues[16] = d16
			ps55.OverlayValues[17] = d17
			ps55.OverlayValues[18] = d18
			ps55.OverlayValues[19] = d19
			ps55.OverlayValues[20] = d20
			ps55.OverlayValues[21] = d21
			ps55.OverlayValues[22] = d22
			ps55.OverlayValues[23] = d23
			ps55.OverlayValues[24] = d24
			ps55.OverlayValues[25] = d25
			ps55.OverlayValues[26] = d26
			ps55.OverlayValues[27] = d27
			ps55.OverlayValues[28] = d28
			ps55.OverlayValues[29] = d29
			ps55.OverlayValues[30] = d30
			ps55.OverlayValues[31] = d31
			ps55.OverlayValues[32] = d32
			ps55.OverlayValues[33] = d33
			ps55.OverlayValues[34] = d34
			ps55.OverlayValues[35] = d35
			ps55.OverlayValues[36] = d36
			ps55.OverlayValues[37] = d37
			ps55.OverlayValues[38] = d38
			ps55.OverlayValues[39] = d39
			ps55.OverlayValues[40] = d40
			ps55.OverlayValues[41] = d41
			ps55.OverlayValues[42] = d42
			ps55.OverlayValues[43] = d43
			ps55.OverlayValues[44] = d44
			ps55.OverlayValues[45] = d45
			ps55.OverlayValues[46] = d46
			ps55.OverlayValues[47] = d47
			ps55.OverlayValues[48] = d48
			ps55.OverlayValues[49] = d49
			ps55.OverlayValues[50] = d50
			ps55.OverlayValues[51] = d51
			ps55.OverlayValues[52] = d52
			ps56 := scm.PhiState{General: true}
			ps56.OverlayValues = make([]scm.JITValueDesc, 53)
			ps56.OverlayValues[0] = d0
			ps56.OverlayValues[1] = d1
			ps56.OverlayValues[9] = d9
			ps56.OverlayValues[10] = d10
			ps56.OverlayValues[11] = d11
			ps56.OverlayValues[12] = d12
			ps56.OverlayValues[13] = d13
			ps56.OverlayValues[14] = d14
			ps56.OverlayValues[15] = d15
			ps56.OverlayValues[16] = d16
			ps56.OverlayValues[17] = d17
			ps56.OverlayValues[18] = d18
			ps56.OverlayValues[19] = d19
			ps56.OverlayValues[20] = d20
			ps56.OverlayValues[21] = d21
			ps56.OverlayValues[22] = d22
			ps56.OverlayValues[23] = d23
			ps56.OverlayValues[24] = d24
			ps56.OverlayValues[25] = d25
			ps56.OverlayValues[26] = d26
			ps56.OverlayValues[27] = d27
			ps56.OverlayValues[28] = d28
			ps56.OverlayValues[29] = d29
			ps56.OverlayValues[30] = d30
			ps56.OverlayValues[31] = d31
			ps56.OverlayValues[32] = d32
			ps56.OverlayValues[33] = d33
			ps56.OverlayValues[34] = d34
			ps56.OverlayValues[35] = d35
			ps56.OverlayValues[36] = d36
			ps56.OverlayValues[37] = d37
			ps56.OverlayValues[38] = d38
			ps56.OverlayValues[39] = d39
			ps56.OverlayValues[40] = d40
			ps56.OverlayValues[41] = d41
			ps56.OverlayValues[42] = d42
			ps56.OverlayValues[43] = d43
			ps56.OverlayValues[44] = d44
			ps56.OverlayValues[45] = d45
			ps56.OverlayValues[46] = d46
			ps56.OverlayValues[47] = d47
			ps56.OverlayValues[48] = d48
			ps56.OverlayValues[49] = d49
			ps56.OverlayValues[50] = d50
			ps56.OverlayValues[51] = d51
			ps56.OverlayValues[52] = d52
			snap57 := d0
			snap58 := d1
			snap59 := d9
			snap60 := d10
			snap61 := d11
			snap62 := d12
			snap63 := d13
			snap64 := d14
			snap65 := d15
			snap66 := d16
			snap67 := d17
			snap68 := d18
			snap69 := d19
			snap70 := d20
			snap71 := d21
			snap72 := d22
			snap73 := d23
			snap74 := d24
			snap75 := d25
			snap76 := d26
			snap77 := d27
			snap78 := d28
			snap79 := d29
			snap80 := d30
			snap81 := d31
			snap82 := d32
			snap83 := d33
			snap84 := d34
			snap85 := d35
			snap86 := d36
			snap87 := d37
			snap88 := d38
			snap89 := d39
			snap90 := d40
			snap91 := d41
			snap92 := d42
			snap93 := d43
			snap94 := d44
			snap95 := d45
			snap96 := d46
			snap97 := d47
			snap98 := d48
			snap99 := d49
			snap100 := d50
			snap101 := d51
			snap102 := d52
			alloc103 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps56)
			}
			ctx.RestoreAllocState(alloc103)
			d0 = snap57
			d1 = snap58
			d9 = snap59
			d10 = snap60
			d11 = snap61
			d12 = snap62
			d13 = snap63
			d14 = snap64
			d15 = snap65
			d16 = snap66
			d17 = snap67
			d18 = snap68
			d19 = snap69
			d20 = snap70
			d21 = snap71
			d22 = snap72
			d23 = snap73
			d24 = snap74
			d25 = snap75
			d26 = snap76
			d27 = snap77
			d28 = snap78
			d29 = snap79
			d30 = snap80
			d31 = snap81
			d32 = snap82
			d33 = snap83
			d34 = snap84
			d35 = snap85
			d36 = snap86
			d37 = snap87
			d38 = snap88
			d39 = snap89
			d40 = snap90
			d41 = snap91
			d42 = snap92
			d43 = snap93
			d44 = snap94
			d45 = snap95
			d46 = snap96
			d47 = snap97
			d48 = snap98
			d49 = snap99
			d50 = snap100
			d51 = snap101
			d52 = snap102
			if !bbs[5].Rendered {
				return bbs[5].RenderPS(ps55)
			}
			return result
			ctx.FreeDesc(&d51)
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
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d104 = idxInt
			_ = d104
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			lbl22 := ctx.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d104)
			var d106 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d104.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.EmitMovRegReg(r50, d104.Reg)
				ctx.EmitShlRegImm8(r50, 32)
				ctx.EmitShrRegImm8(r50, 32)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d106)
			}
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r51, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d107)
			}
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			var d108 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d107.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.EmitMovRegReg(r52, d107.Reg)
				ctx.EmitShlRegImm8(r52, 56)
				ctx.EmitShrRegImm8(r52, 56)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d108)
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
				ctx.EmitMovRegImm64(scratch, uint64(d106.Imm.Int()))
				ctx.EmitImulInt64(scratch, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.EmitMovRegReg(scratch, d106.Reg)
				if d108.Imm.Int() >= -2147483648 && d108.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d108.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else {
				r53 := ctx.AllocRegExcept(d106.Reg, d108.Reg)
				ctx.EmitMovRegReg(r53, d106.Reg)
				ctx.EmitImulInt64(r53, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d109)
			}
			if d109.Loc == scm.LocReg && d106.Loc == scm.LocReg && d109.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r54, uint64(dataPtr))
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d110)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				ctx.EmitMovRegMem(r54, thisptr.Reg, off)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d110)
			}
			ctx.BindReg(r54, &d110)
			ctx.EnsureDesc(&d109)
			var d111 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(r55, d109.Reg)
				ctx.EmitShrRegImm8(r55, 6)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d111)
			}
			if d111.Loc == scm.LocReg && d109.Loc == scm.LocReg && d111.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d111)
			r56 := ctx.AllocReg()
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d110)
			if d111.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r56, uint64(d111.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r56, d111.Reg)
				ctx.EmitShlRegImm8(r56, 3)
			}
			if d110.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r56, d110.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d112)
			ctx.FreeDesc(&d111)
			ctx.EnsureDesc(&d109)
			var d113 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(r58, d109.Reg)
				ctx.EmitAndRegImm32(r58, 63)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d113)
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
				r59 := ctx.AllocRegExcept(d112.Reg)
				ctx.EmitMovRegReg(r59, d112.Reg)
				ctx.EmitShlRegImm8(r59, uint8(d113.Imm.Int()))
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d114)
			} else {
				{
					shiftSrc := d112.Reg
					r60 := ctx.AllocRegExcept(d112.Reg)
					ctx.EmitMovRegReg(r60, d112.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d113.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d113.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d113.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 25)
				r61 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r61, thisptr.Reg, off)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d115)
			}
			d116 = d115
			ctx.EnsureDesc(&d116)
			if d116.Loc != scm.LocImm && d116.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.ReserveLabel()
			lbl24 := ctx.ReserveLabel()
			lbl25 := ctx.ReserveLabel()
			lbl26 := ctx.ReserveLabel()
			if d116.Loc == scm.LocImm {
				if d116.Imm.Bool() {
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl23)
				} else {
					ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d114)
			if d114.Loc == scm.LocReg {
				ctx.ProtectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.ProtectReg(d114.Reg)
				ctx.ProtectReg(d114.Reg2)
			}
			d117 = d114
			if d117.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d117)
			ctx.EmitStoreToStack(d117, int32(bbs[2].PhiBase)+int32(0))
			if d114.Loc == scm.LocReg {
				ctx.UnprotectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d114.Reg)
				ctx.UnprotectReg(d114.Reg2)
			}
					ctx.EmitJmp(lbl24)
				}
			} else {
				ctx.EmitCmpRegImm32(d116.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl25)
				ctx.EmitJmp(lbl26)
				ctx.MarkLabel(lbl25)
				ctx.EmitJmp(lbl23)
				ctx.MarkLabel(lbl26)
			ctx.EnsureDesc(&d114)
			if d114.Loc == scm.LocReg {
				ctx.ProtectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.ProtectReg(d114.Reg)
				ctx.ProtectReg(d114.Reg2)
			}
			d118 = d114
			if d118.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			ctx.EmitStoreToStack(d118, int32(bbs[2].PhiBase)+int32(0))
			if d114.Loc == scm.LocReg {
				ctx.UnprotectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d114.Reg)
				ctx.UnprotectReg(d114.Reg2)
			}
				ctx.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d115)
			bbpos_2_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl24)
			ctx.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			var d119 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r62 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r62, thisptr.Reg, off)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d119)
			}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d119.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.EmitMovRegReg(r63, d119.Reg)
				ctx.EmitShlRegImm8(r63, 56)
				ctx.EmitShrRegImm8(r63, 56)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d120)
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
				r64 := ctx.AllocRegExcept(d121.Reg)
				ctx.EmitMovRegReg(r64, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
				ctx.EmitSubInt64(scratch, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.EmitMovRegReg(scratch, d121.Reg)
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d120.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r65 := ctx.AllocRegExcept(d121.Reg, d120.Reg)
				ctx.EmitMovRegReg(r65, d121.Reg)
				ctx.EmitSubInt64(r65, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d122)
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
				r66 := ctx.AllocRegExcept(d105.Reg)
				ctx.EmitMovRegReg(r66, d105.Reg)
				ctx.EmitShrRegImm8(r66, uint8(d122.Imm.Int()))
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d123)
			} else {
				{
					shiftSrc := d105.Reg
					r67 := ctx.AllocRegExcept(d105.Reg)
					ctx.EmitMovRegReg(r67, d105.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d122.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d122.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d122.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d123)
			if d123.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r68, d123)
			}
			ctx.EmitJmp(lbl22)
			bbpos_2_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl23)
			ctx.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d109)
			var d124 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(r69, d109.Reg)
				ctx.EmitAndRegImm32(r69, 63)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d124)
			}
			if d124.Loc == scm.LocReg && d109.Loc == scm.LocReg && d124.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			var d125 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r70 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r70, thisptr.Reg, off)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d125)
			}
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d125.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.EmitMovRegReg(r71, d125.Reg)
				ctx.EmitShlRegImm8(r71, 56)
				ctx.EmitShrRegImm8(r71, 56)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d126)
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
				r72 := ctx.AllocRegExcept(d124.Reg)
				ctx.EmitMovRegReg(r72, d124.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d127)
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
				ctx.BindReg(d126.Reg, &d127)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d124.Imm.Int()))
				ctx.EmitAddInt64(scratch, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.EmitMovRegReg(scratch, d124.Reg)
				if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d126.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else {
				r73 := ctx.AllocRegExcept(d124.Reg, d126.Reg)
				ctx.EmitMovRegReg(r73, d124.Reg)
				ctx.EmitAddInt64(r73, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d127)
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
				r74 := ctx.AllocRegExcept(d127.Reg)
				ctx.EmitCmpRegImm32(d127.Reg, 64)
				ctx.EmitSetcc(r74, scm.CcA)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d128)
			}
			ctx.FreeDesc(&d127)
			d129 = d128
			ctx.EnsureDesc(&d129)
			if d129.Loc != scm.LocImm && d129.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.ReserveLabel()
			lbl28 := ctx.ReserveLabel()
			lbl29 := ctx.ReserveLabel()
			if d129.Loc == scm.LocImm {
				if d129.Imm.Bool() {
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl27)
				} else {
					ctx.MarkLabel(lbl29)
			ctx.EnsureDesc(&d114)
			if d114.Loc == scm.LocReg {
				ctx.ProtectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.ProtectReg(d114.Reg)
				ctx.ProtectReg(d114.Reg2)
			}
			d130 = d114
			if d130.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d130)
			ctx.EmitStoreToStack(d130, int32(bbs[2].PhiBase)+int32(0))
			if d114.Loc == scm.LocReg {
				ctx.UnprotectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d114.Reg)
				ctx.UnprotectReg(d114.Reg2)
			}
					ctx.EmitJmp(lbl24)
				}
			} else {
				ctx.EmitCmpRegImm32(d129.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl28)
				ctx.EmitJmp(lbl29)
				ctx.MarkLabel(lbl28)
				ctx.EmitJmp(lbl27)
				ctx.MarkLabel(lbl29)
			ctx.EnsureDesc(&d114)
			if d114.Loc == scm.LocReg {
				ctx.ProtectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.ProtectReg(d114.Reg)
				ctx.ProtectReg(d114.Reg2)
			}
			d131 = d114
			if d131.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d131)
			ctx.EmitStoreToStack(d131, int32(bbs[2].PhiBase)+int32(0))
			if d114.Loc == scm.LocReg {
				ctx.UnprotectReg(d114.Reg)
			} else if d114.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d114.Reg)
				ctx.UnprotectReg(d114.Reg2)
			}
				ctx.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d128)
			bbpos_2_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl27)
			ctx.ResolveFixups()
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			ctx.EnsureDesc(&d109)
			var d132 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(r75, d109.Reg)
				ctx.EmitShrRegImm8(r75, 6)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d132)
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
				ctx.EmitMovRegReg(scratch, d132.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d133)
			}
			if d133.Loc == scm.LocReg && d132.Loc == scm.LocReg && d133.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.EnsureDesc(&d133)
			r76 := ctx.AllocReg()
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d110)
			if d133.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r76, uint64(d133.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r76, d133.Reg)
				ctx.EmitShlRegImm8(r76, 3)
			}
			if d110.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
				ctx.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r76, d110.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d134)
			ctx.FreeDesc(&d133)
			ctx.EnsureDesc(&d109)
			var d135 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d109.Reg)
				ctx.EmitMovRegReg(r78, d109.Reg)
				ctx.EmitAndRegImm32(r78, 63)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d135)
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
				r79 := ctx.AllocRegExcept(d136.Reg)
				ctx.EmitMovRegReg(r79, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d137)
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
				ctx.EmitSubInt64(scratch, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.EmitMovRegReg(scratch, d136.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else {
				r80 := ctx.AllocRegExcept(d136.Reg, d135.Reg)
				ctx.EmitMovRegReg(r80, d136.Reg)
				ctx.EmitSubInt64(r80, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d137)
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
				r81 := ctx.AllocRegExcept(d134.Reg)
				ctx.EmitMovRegReg(r81, d134.Reg)
				ctx.EmitShrRegImm8(r81, uint8(d137.Imm.Int()))
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d138)
			} else {
				{
					shiftSrc := d134.Reg
					r82 := ctx.AllocRegExcept(d134.Reg)
					ctx.EmitMovRegReg(r82, d134.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d137.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d137.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d137.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
				r83 := ctx.AllocRegExcept(d114.Reg)
				ctx.EmitMovRegReg(r83, d114.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d139)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d114.Imm.Int()))
				ctx.EmitOrInt64(scratch, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else if d138.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d114.Reg)
				ctx.EmitMovRegReg(r84, d114.Reg)
				if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r84, int32(d138.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
					ctx.EmitOrInt64(r84, scm.RegR11)
				}
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d139)
			} else {
				r85 := ctx.AllocRegExcept(d114.Reg, d138.Reg)
				ctx.EmitMovRegReg(r85, d114.Reg)
				ctx.EmitOrInt64(r85, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d139)
			}
			if d139.Loc == scm.LocReg && d114.Loc == scm.LocReg && d139.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.EnsureDesc(&d139)
			if d139.Loc == scm.LocReg {
				ctx.ProtectReg(d139.Reg)
			} else if d139.Loc == scm.LocRegPair {
				ctx.ProtectReg(d139.Reg)
				ctx.ProtectReg(d139.Reg2)
			}
			d140 = d139
			if d140.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d140)
			ctx.EmitStoreToStack(d140, int32(bbs[2].PhiBase)+int32(0))
			if d139.Loc == scm.LocReg {
				ctx.UnprotectReg(d139.Reg)
			} else if d139.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d139.Reg)
				ctx.UnprotectReg(d139.Reg2)
			}
			ctx.EmitJmp(lbl24)
			ctx.MarkLabel(lbl22)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d141)
			ctx.BindReg(r68, &d141)
			if r48 { ctx.UnprotectReg(r49) }
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d141)
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d141.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.EmitMovRegReg(r86, d141.Reg)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d142)
			}
			ctx.FreeDesc(&d141)
			var d143 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r87 := ctx.AllocReg()
				ctx.EmitMovRegMem(r87, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d143)
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
				r88 := ctx.AllocRegExcept(d142.Reg)
				ctx.EmitMovRegReg(r88, d142.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d144)
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d143.Reg}
				ctx.BindReg(d143.Reg, &d144)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.EmitAddInt64(scratch, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.EmitMovRegReg(scratch, d142.Reg)
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d143.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d143.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r89 := ctx.AllocRegExcept(d142.Reg, d143.Reg)
				ctx.EmitMovRegReg(r89, d142.Reg)
				ctx.EmitAddInt64(r89, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d144)
			}
			if d144.Loc == scm.LocReg && d142.Loc == scm.LocReg && d144.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d144)
			var d145 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d144.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.EmitMovRegReg(r90, d144.Reg)
				ctx.EmitShlRegImm8(r90, 32)
				ctx.EmitShrRegImm8(r90, 32)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d145)
			}
			ctx.FreeDesc(&d144)
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r91 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r91, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d146)
			}
			d147 = d146
			ctx.EnsureDesc(&d147)
			if d147.Loc != scm.LocImm && d147.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d147.Loc == scm.LocImm {
				if d147.Imm.Bool() {
			ps148 := scm.PhiState{General: ps.General}
			ps148.OverlayValues = make([]scm.JITValueDesc, 148)
			ps148.OverlayValues[0] = d0
			ps148.OverlayValues[1] = d1
			ps148.OverlayValues[9] = d9
			ps148.OverlayValues[10] = d10
			ps148.OverlayValues[11] = d11
			ps148.OverlayValues[12] = d12
			ps148.OverlayValues[13] = d13
			ps148.OverlayValues[14] = d14
			ps148.OverlayValues[15] = d15
			ps148.OverlayValues[16] = d16
			ps148.OverlayValues[17] = d17
			ps148.OverlayValues[18] = d18
			ps148.OverlayValues[19] = d19
			ps148.OverlayValues[20] = d20
			ps148.OverlayValues[21] = d21
			ps148.OverlayValues[22] = d22
			ps148.OverlayValues[23] = d23
			ps148.OverlayValues[24] = d24
			ps148.OverlayValues[25] = d25
			ps148.OverlayValues[26] = d26
			ps148.OverlayValues[27] = d27
			ps148.OverlayValues[28] = d28
			ps148.OverlayValues[29] = d29
			ps148.OverlayValues[30] = d30
			ps148.OverlayValues[31] = d31
			ps148.OverlayValues[32] = d32
			ps148.OverlayValues[33] = d33
			ps148.OverlayValues[34] = d34
			ps148.OverlayValues[35] = d35
			ps148.OverlayValues[36] = d36
			ps148.OverlayValues[37] = d37
			ps148.OverlayValues[38] = d38
			ps148.OverlayValues[39] = d39
			ps148.OverlayValues[40] = d40
			ps148.OverlayValues[41] = d41
			ps148.OverlayValues[42] = d42
			ps148.OverlayValues[43] = d43
			ps148.OverlayValues[44] = d44
			ps148.OverlayValues[45] = d45
			ps148.OverlayValues[46] = d46
			ps148.OverlayValues[47] = d47
			ps148.OverlayValues[48] = d48
			ps148.OverlayValues[49] = d49
			ps148.OverlayValues[50] = d50
			ps148.OverlayValues[51] = d51
			ps148.OverlayValues[52] = d52
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
			ps148.OverlayValues[147] = d147
					return bbs[8].RenderPS(ps148)
				}
			ps149 := scm.PhiState{General: ps.General}
			ps149.OverlayValues = make([]scm.JITValueDesc, 148)
			ps149.OverlayValues[0] = d0
			ps149.OverlayValues[1] = d1
			ps149.OverlayValues[9] = d9
			ps149.OverlayValues[10] = d10
			ps149.OverlayValues[11] = d11
			ps149.OverlayValues[12] = d12
			ps149.OverlayValues[13] = d13
			ps149.OverlayValues[14] = d14
			ps149.OverlayValues[15] = d15
			ps149.OverlayValues[16] = d16
			ps149.OverlayValues[17] = d17
			ps149.OverlayValues[18] = d18
			ps149.OverlayValues[19] = d19
			ps149.OverlayValues[20] = d20
			ps149.OverlayValues[21] = d21
			ps149.OverlayValues[22] = d22
			ps149.OverlayValues[23] = d23
			ps149.OverlayValues[24] = d24
			ps149.OverlayValues[25] = d25
			ps149.OverlayValues[26] = d26
			ps149.OverlayValues[27] = d27
			ps149.OverlayValues[28] = d28
			ps149.OverlayValues[29] = d29
			ps149.OverlayValues[30] = d30
			ps149.OverlayValues[31] = d31
			ps149.OverlayValues[32] = d32
			ps149.OverlayValues[33] = d33
			ps149.OverlayValues[34] = d34
			ps149.OverlayValues[35] = d35
			ps149.OverlayValues[36] = d36
			ps149.OverlayValues[37] = d37
			ps149.OverlayValues[38] = d38
			ps149.OverlayValues[39] = d39
			ps149.OverlayValues[40] = d40
			ps149.OverlayValues[41] = d41
			ps149.OverlayValues[42] = d42
			ps149.OverlayValues[43] = d43
			ps149.OverlayValues[44] = d44
			ps149.OverlayValues[45] = d45
			ps149.OverlayValues[46] = d46
			ps149.OverlayValues[47] = d47
			ps149.OverlayValues[48] = d48
			ps149.OverlayValues[49] = d49
			ps149.OverlayValues[50] = d50
			ps149.OverlayValues[51] = d51
			ps149.OverlayValues[52] = d52
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
			ps149.OverlayValues[147] = d147
				return bbs[7].RenderPS(ps149)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl30 := ctx.ReserveLabel()
			lbl31 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d147.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl30)
			ctx.EmitJmp(lbl31)
			ctx.MarkLabel(lbl30)
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl31)
			ctx.EmitJmp(lbl8)
			ps150 := scm.PhiState{General: true}
			ps150.OverlayValues = make([]scm.JITValueDesc, 148)
			ps150.OverlayValues[0] = d0
			ps150.OverlayValues[1] = d1
			ps150.OverlayValues[9] = d9
			ps150.OverlayValues[10] = d10
			ps150.OverlayValues[11] = d11
			ps150.OverlayValues[12] = d12
			ps150.OverlayValues[13] = d13
			ps150.OverlayValues[14] = d14
			ps150.OverlayValues[15] = d15
			ps150.OverlayValues[16] = d16
			ps150.OverlayValues[17] = d17
			ps150.OverlayValues[18] = d18
			ps150.OverlayValues[19] = d19
			ps150.OverlayValues[20] = d20
			ps150.OverlayValues[21] = d21
			ps150.OverlayValues[22] = d22
			ps150.OverlayValues[23] = d23
			ps150.OverlayValues[24] = d24
			ps150.OverlayValues[25] = d25
			ps150.OverlayValues[26] = d26
			ps150.OverlayValues[27] = d27
			ps150.OverlayValues[28] = d28
			ps150.OverlayValues[29] = d29
			ps150.OverlayValues[30] = d30
			ps150.OverlayValues[31] = d31
			ps150.OverlayValues[32] = d32
			ps150.OverlayValues[33] = d33
			ps150.OverlayValues[34] = d34
			ps150.OverlayValues[35] = d35
			ps150.OverlayValues[36] = d36
			ps150.OverlayValues[37] = d37
			ps150.OverlayValues[38] = d38
			ps150.OverlayValues[39] = d39
			ps150.OverlayValues[40] = d40
			ps150.OverlayValues[41] = d41
			ps150.OverlayValues[42] = d42
			ps150.OverlayValues[43] = d43
			ps150.OverlayValues[44] = d44
			ps150.OverlayValues[45] = d45
			ps150.OverlayValues[46] = d46
			ps150.OverlayValues[47] = d47
			ps150.OverlayValues[48] = d48
			ps150.OverlayValues[49] = d49
			ps150.OverlayValues[50] = d50
			ps150.OverlayValues[51] = d51
			ps150.OverlayValues[52] = d52
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
			ps150.OverlayValues[147] = d147
			ps151 := scm.PhiState{General: true}
			ps151.OverlayValues = make([]scm.JITValueDesc, 148)
			ps151.OverlayValues[0] = d0
			ps151.OverlayValues[1] = d1
			ps151.OverlayValues[9] = d9
			ps151.OverlayValues[10] = d10
			ps151.OverlayValues[11] = d11
			ps151.OverlayValues[12] = d12
			ps151.OverlayValues[13] = d13
			ps151.OverlayValues[14] = d14
			ps151.OverlayValues[15] = d15
			ps151.OverlayValues[16] = d16
			ps151.OverlayValues[17] = d17
			ps151.OverlayValues[18] = d18
			ps151.OverlayValues[19] = d19
			ps151.OverlayValues[20] = d20
			ps151.OverlayValues[21] = d21
			ps151.OverlayValues[22] = d22
			ps151.OverlayValues[23] = d23
			ps151.OverlayValues[24] = d24
			ps151.OverlayValues[25] = d25
			ps151.OverlayValues[26] = d26
			ps151.OverlayValues[27] = d27
			ps151.OverlayValues[28] = d28
			ps151.OverlayValues[29] = d29
			ps151.OverlayValues[30] = d30
			ps151.OverlayValues[31] = d31
			ps151.OverlayValues[32] = d32
			ps151.OverlayValues[33] = d33
			ps151.OverlayValues[34] = d34
			ps151.OverlayValues[35] = d35
			ps151.OverlayValues[36] = d36
			ps151.OverlayValues[37] = d37
			ps151.OverlayValues[38] = d38
			ps151.OverlayValues[39] = d39
			ps151.OverlayValues[40] = d40
			ps151.OverlayValues[41] = d41
			ps151.OverlayValues[42] = d42
			ps151.OverlayValues[43] = d43
			ps151.OverlayValues[44] = d44
			ps151.OverlayValues[45] = d45
			ps151.OverlayValues[46] = d46
			ps151.OverlayValues[47] = d47
			ps151.OverlayValues[48] = d48
			ps151.OverlayValues[49] = d49
			ps151.OverlayValues[50] = d50
			ps151.OverlayValues[51] = d51
			ps151.OverlayValues[52] = d52
			ps151.OverlayValues[104] = d104
			ps151.OverlayValues[105] = d105
			ps151.OverlayValues[106] = d106
			ps151.OverlayValues[107] = d107
			ps151.OverlayValues[108] = d108
			ps151.OverlayValues[109] = d109
			ps151.OverlayValues[110] = d110
			ps151.OverlayValues[111] = d111
			ps151.OverlayValues[112] = d112
			ps151.OverlayValues[113] = d113
			ps151.OverlayValues[114] = d114
			ps151.OverlayValues[115] = d115
			ps151.OverlayValues[116] = d116
			ps151.OverlayValues[117] = d117
			ps151.OverlayValues[118] = d118
			ps151.OverlayValues[119] = d119
			ps151.OverlayValues[120] = d120
			ps151.OverlayValues[121] = d121
			ps151.OverlayValues[122] = d122
			ps151.OverlayValues[123] = d123
			ps151.OverlayValues[124] = d124
			ps151.OverlayValues[125] = d125
			ps151.OverlayValues[126] = d126
			ps151.OverlayValues[127] = d127
			ps151.OverlayValues[128] = d128
			ps151.OverlayValues[129] = d129
			ps151.OverlayValues[130] = d130
			ps151.OverlayValues[131] = d131
			ps151.OverlayValues[132] = d132
			ps151.OverlayValues[133] = d133
			ps151.OverlayValues[134] = d134
			ps151.OverlayValues[135] = d135
			ps151.OverlayValues[136] = d136
			ps151.OverlayValues[137] = d137
			ps151.OverlayValues[138] = d138
			ps151.OverlayValues[139] = d139
			ps151.OverlayValues[140] = d140
			ps151.OverlayValues[141] = d141
			ps151.OverlayValues[142] = d142
			ps151.OverlayValues[143] = d143
			ps151.OverlayValues[144] = d144
			ps151.OverlayValues[145] = d145
			ps151.OverlayValues[146] = d146
			ps151.OverlayValues[147] = d147
			snap152 := d0
			snap153 := d1
			snap154 := d9
			snap155 := d10
			snap156 := d11
			snap157 := d12
			snap158 := d13
			snap159 := d14
			snap160 := d15
			snap161 := d16
			snap162 := d17
			snap163 := d18
			snap164 := d19
			snap165 := d20
			snap166 := d21
			snap167 := d22
			snap168 := d23
			snap169 := d24
			snap170 := d25
			snap171 := d26
			snap172 := d27
			snap173 := d28
			snap174 := d29
			snap175 := d30
			snap176 := d31
			snap177 := d32
			snap178 := d33
			snap179 := d34
			snap180 := d35
			snap181 := d36
			snap182 := d37
			snap183 := d38
			snap184 := d39
			snap185 := d40
			snap186 := d41
			snap187 := d42
			snap188 := d43
			snap189 := d44
			snap190 := d45
			snap191 := d46
			snap192 := d47
			snap193 := d48
			snap194 := d49
			snap195 := d50
			snap196 := d51
			snap197 := d52
			snap198 := d104
			snap199 := d105
			snap200 := d106
			snap201 := d107
			snap202 := d108
			snap203 := d109
			snap204 := d110
			snap205 := d111
			snap206 := d112
			snap207 := d113
			snap208 := d114
			snap209 := d115
			snap210 := d116
			snap211 := d117
			snap212 := d118
			snap213 := d119
			snap214 := d120
			snap215 := d121
			snap216 := d122
			snap217 := d123
			snap218 := d124
			snap219 := d125
			snap220 := d126
			snap221 := d127
			snap222 := d128
			snap223 := d129
			snap224 := d130
			snap225 := d131
			snap226 := d132
			snap227 := d133
			snap228 := d134
			snap229 := d135
			snap230 := d136
			snap231 := d137
			snap232 := d138
			snap233 := d139
			snap234 := d140
			snap235 := d141
			snap236 := d142
			snap237 := d143
			snap238 := d144
			snap239 := d145
			snap240 := d146
			snap241 := d147
			alloc242 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps151)
			}
			ctx.RestoreAllocState(alloc242)
			d0 = snap152
			d1 = snap153
			d9 = snap154
			d10 = snap155
			d11 = snap156
			d12 = snap157
			d13 = snap158
			d14 = snap159
			d15 = snap160
			d16 = snap161
			d17 = snap162
			d18 = snap163
			d19 = snap164
			d20 = snap165
			d21 = snap166
			d22 = snap167
			d23 = snap168
			d24 = snap169
			d25 = snap170
			d26 = snap171
			d27 = snap172
			d28 = snap173
			d29 = snap174
			d30 = snap175
			d31 = snap176
			d32 = snap177
			d33 = snap178
			d34 = snap179
			d35 = snap180
			d36 = snap181
			d37 = snap182
			d38 = snap183
			d39 = snap184
			d40 = snap185
			d41 = snap186
			d42 = snap187
			d43 = snap188
			d44 = snap189
			d45 = snap190
			d46 = snap191
			d47 = snap192
			d48 = snap193
			d49 = snap194
			d50 = snap195
			d51 = snap196
			d52 = snap197
			d104 = snap198
			d105 = snap199
			d106 = snap200
			d107 = snap201
			d108 = snap202
			d109 = snap203
			d110 = snap204
			d111 = snap205
			d112 = snap206
			d113 = snap207
			d114 = snap208
			d115 = snap209
			d116 = snap210
			d117 = snap211
			d118 = snap212
			d119 = snap213
			d120 = snap214
			d121 = snap215
			d122 = snap216
			d123 = snap217
			d124 = snap218
			d125 = snap219
			d126 = snap220
			d127 = snap221
			d128 = snap222
			d129 = snap223
			d130 = snap224
			d131 = snap225
			d132 = snap226
			d133 = snap227
			d134 = snap228
			d135 = snap229
			d136 = snap230
			d137 = snap231
			d138 = snap232
			d139 = snap233
			d140 = snap234
			d141 = snap235
			d142 = snap236
			d143 = snap237
			d144 = snap238
			d145 = snap239
			d146 = snap240
			d147 = snap241
			if !bbs[8].Rendered {
				return bbs[8].RenderPS(ps150)
			}
			return result
			ctx.FreeDesc(&d146)
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			ctx.ReclaimUntrackedRegs()
			d243 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d243)
			ctx.BindReg(r1, &d243)
			ctx.EmitMakeNil(d243)
			ctx.EmitJmp(lbl0)
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
					ctx.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
				d243 = ps.OverlayValues[243]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			d244 = idxInt
			_ = d244
			r92 := idxInt.Loc == scm.LocReg
			r93 := idxInt.Reg
			if r92 { ctx.ProtectReg(r93) }
			d245 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			lbl32 := ctx.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d245 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d244)
			var d246 scm.JITValueDesc
			if d244.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d244.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.EmitMovRegReg(r94, d244.Reg)
				ctx.EmitShlRegImm8(r94, 32)
				ctx.EmitShrRegImm8(r94, 32)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d246)
			}
			var d247 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r95 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r95, thisptr.Reg, off)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d247)
			}
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d247)
			var d248 scm.JITValueDesc
			if d247.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d247.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.EmitMovRegReg(r96, d247.Reg)
				ctx.EmitShlRegImm8(r96, 56)
				ctx.EmitShrRegImm8(r96, 56)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d248)
			}
			ctx.FreeDesc(&d247)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			var d249 scm.JITValueDesc
			if d246.Loc == scm.LocImm && d248.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() * d248.Imm.Int())}
			} else if d246.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d248.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d246.Imm.Int()))
				ctx.EmitImulInt64(scratch, d248.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else if d248.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d246.Reg)
				ctx.EmitMovRegReg(scratch, d246.Reg)
				if d248.Imm.Int() >= -2147483648 && d248.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d248.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d248.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else {
				r97 := ctx.AllocRegExcept(d246.Reg, d248.Reg)
				ctx.EmitMovRegReg(r97, d246.Reg)
				ctx.EmitImulInt64(r97, d248.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d249)
			}
			if d249.Loc == scm.LocReg && d246.Loc == scm.LocReg && d249.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			ctx.FreeDesc(&d248)
			var d250 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r98, uint64(dataPtr))
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d250)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.EmitMovRegMem(r98, thisptr.Reg, off)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d250)
			}
			ctx.BindReg(r98, &d250)
			ctx.EnsureDesc(&d249)
			var d251 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d249.Reg)
				ctx.EmitMovRegReg(r99, d249.Reg)
				ctx.EmitShrRegImm8(r99, 6)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d251)
			}
			if d251.Loc == scm.LocReg && d249.Loc == scm.LocReg && d251.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d251)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			if d251.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r100, uint64(d251.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r100, d251.Reg)
				ctx.EmitShlRegImm8(r100, 3)
			}
			if d250.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
				ctx.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r100, d250.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d252 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d252)
			ctx.FreeDesc(&d251)
			ctx.EnsureDesc(&d249)
			var d253 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d249.Reg)
				ctx.EmitMovRegReg(r102, d249.Reg)
				ctx.EmitAndRegImm32(r102, 63)
				d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d253)
			}
			if d253.Loc == scm.LocReg && d249.Loc == scm.LocReg && d253.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d252)
			ctx.EnsureDesc(&d253)
			var d254 scm.JITValueDesc
			if d252.Loc == scm.LocImm && d253.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d252.Imm.Int()) << uint64(d253.Imm.Int())))}
			} else if d253.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d252.Reg)
				ctx.EmitMovRegReg(r103, d252.Reg)
				ctx.EmitShlRegImm8(r103, uint8(d253.Imm.Int()))
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d254)
			} else {
				{
					shiftSrc := d252.Reg
					r104 := ctx.AllocRegExcept(d252.Reg)
					ctx.EmitMovRegReg(r104, d252.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d253.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d253.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d253.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d254)
				}
			}
			if d254.Loc == scm.LocReg && d252.Loc == scm.LocReg && d254.Reg == d252.Reg {
				ctx.TransferReg(d252.Reg)
				d252.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d252)
			ctx.FreeDesc(&d253)
			var d255 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r105 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r105, thisptr.Reg, off)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d255)
			}
			d256 = d255
			ctx.EnsureDesc(&d256)
			if d256.Loc != scm.LocImm && d256.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl33 := ctx.ReserveLabel()
			lbl34 := ctx.ReserveLabel()
			lbl35 := ctx.ReserveLabel()
			lbl36 := ctx.ReserveLabel()
			if d256.Loc == scm.LocImm {
				if d256.Imm.Bool() {
					ctx.MarkLabel(lbl35)
					ctx.EmitJmp(lbl33)
				} else {
					ctx.MarkLabel(lbl36)
			ctx.EnsureDesc(&d254)
			if d254.Loc == scm.LocReg {
				ctx.ProtectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.ProtectReg(d254.Reg)
				ctx.ProtectReg(d254.Reg2)
			}
			d257 = d254
			if d257.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d257)
			ctx.EmitStoreToStack(d257, int32(bbs[2].PhiBase)+int32(0))
			if d254.Loc == scm.LocReg {
				ctx.UnprotectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d254.Reg)
				ctx.UnprotectReg(d254.Reg2)
			}
					ctx.EmitJmp(lbl34)
				}
			} else {
				ctx.EmitCmpRegImm32(d256.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl35)
				ctx.EmitJmp(lbl36)
				ctx.MarkLabel(lbl35)
				ctx.EmitJmp(lbl33)
				ctx.MarkLabel(lbl36)
			ctx.EnsureDesc(&d254)
			if d254.Loc == scm.LocReg {
				ctx.ProtectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.ProtectReg(d254.Reg)
				ctx.ProtectReg(d254.Reg2)
			}
			d258 = d254
			if d258.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d258)
			ctx.EmitStoreToStack(d258, int32(bbs[2].PhiBase)+int32(0))
			if d254.Loc == scm.LocReg {
				ctx.UnprotectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d254.Reg)
				ctx.UnprotectReg(d254.Reg2)
			}
				ctx.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d255)
			bbpos_3_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl34)
			ctx.ResolveFixups()
			d245 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			var d259 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r106 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r106, thisptr.Reg, off)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d259)
			}
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d259)
			var d260 scm.JITValueDesc
			if d259.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d259.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.EmitMovRegReg(r107, d259.Reg)
				ctx.EmitShlRegImm8(r107, 56)
				ctx.EmitShrRegImm8(r107, 56)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d260)
			}
			ctx.FreeDesc(&d259)
			d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d260)
			var d262 scm.JITValueDesc
			if d261.Loc == scm.LocImm && d260.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d261.Imm.Int() - d260.Imm.Int())}
			} else if d260.Loc == scm.LocImm && d260.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d261.Reg)
				ctx.EmitMovRegReg(r108, d261.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d262)
			} else if d261.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d260.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d261.Imm.Int()))
				ctx.EmitSubInt64(scratch, d260.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else if d260.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.EmitMovRegReg(scratch, d261.Reg)
				if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d260.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d260.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else {
				r109 := ctx.AllocRegExcept(d261.Reg, d260.Reg)
				ctx.EmitMovRegReg(r109, d261.Reg)
				ctx.EmitSubInt64(r109, d260.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d262)
			}
			if d262.Loc == scm.LocReg && d261.Loc == scm.LocReg && d262.Reg == d261.Reg {
				ctx.TransferReg(d261.Reg)
				d261.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d260)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d262)
			var d263 scm.JITValueDesc
			if d245.Loc == scm.LocImm && d262.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d245.Imm.Int()) >> uint64(d262.Imm.Int())))}
			} else if d262.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d245.Reg)
				ctx.EmitMovRegReg(r110, d245.Reg)
				ctx.EmitShrRegImm8(r110, uint8(d262.Imm.Int()))
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d263)
			} else {
				{
					shiftSrc := d245.Reg
					r111 := ctx.AllocRegExcept(d245.Reg)
					ctx.EmitMovRegReg(r111, d245.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d262.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d262.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d262.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d263)
				}
			}
			if d263.Loc == scm.LocReg && d245.Loc == scm.LocReg && d263.Reg == d245.Reg {
				ctx.TransferReg(d245.Reg)
				d245.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d245)
			ctx.FreeDesc(&d262)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d263)
			if d263.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d263)
			}
			ctx.EmitJmp(lbl32)
			bbpos_3_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl33)
			ctx.ResolveFixups()
			d245 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d249)
			var d264 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d249.Reg)
				ctx.EmitMovRegReg(r113, d249.Reg)
				ctx.EmitAndRegImm32(r113, 63)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d264)
			}
			if d264.Loc == scm.LocReg && d249.Loc == scm.LocReg && d264.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			var d265 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r114 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r114, thisptr.Reg, off)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d265)
			}
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d265)
			var d266 scm.JITValueDesc
			if d265.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d265.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.EmitMovRegReg(r115, d265.Reg)
				ctx.EmitShlRegImm8(r115, 56)
				ctx.EmitShrRegImm8(r115, 56)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d266)
			}
			ctx.FreeDesc(&d265)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d266)
			var d267 scm.JITValueDesc
			if d264.Loc == scm.LocImm && d266.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d264.Imm.Int() + d266.Imm.Int())}
			} else if d266.Loc == scm.LocImm && d266.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d264.Reg)
				ctx.EmitMovRegReg(r116, d264.Reg)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d267)
			} else if d264.Loc == scm.LocImm && d264.Imm.Int() == 0 {
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d266.Reg}
				ctx.BindReg(d266.Reg, &d267)
			} else if d264.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d266.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d264.Imm.Int()))
				ctx.EmitAddInt64(scratch, d266.Reg)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d267)
			} else if d266.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d264.Reg)
				ctx.EmitMovRegReg(scratch, d264.Reg)
				if d266.Imm.Int() >= -2147483648 && d266.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d266.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d266.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d267)
			} else {
				r117 := ctx.AllocRegExcept(d264.Reg, d266.Reg)
				ctx.EmitMovRegReg(r117, d264.Reg)
				ctx.EmitAddInt64(r117, d266.Reg)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d267)
			}
			if d267.Loc == scm.LocReg && d264.Loc == scm.LocReg && d267.Reg == d264.Reg {
				ctx.TransferReg(d264.Reg)
				d264.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d264)
			ctx.FreeDesc(&d266)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d267.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d267.Reg)
				ctx.EmitCmpRegImm32(d267.Reg, 64)
				ctx.EmitSetcc(r118, scm.CcA)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d268)
			}
			ctx.FreeDesc(&d267)
			d269 = d268
			ctx.EnsureDesc(&d269)
			if d269.Loc != scm.LocImm && d269.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl37 := ctx.ReserveLabel()
			lbl38 := ctx.ReserveLabel()
			lbl39 := ctx.ReserveLabel()
			if d269.Loc == scm.LocImm {
				if d269.Imm.Bool() {
					ctx.MarkLabel(lbl38)
					ctx.EmitJmp(lbl37)
				} else {
					ctx.MarkLabel(lbl39)
			ctx.EnsureDesc(&d254)
			if d254.Loc == scm.LocReg {
				ctx.ProtectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.ProtectReg(d254.Reg)
				ctx.ProtectReg(d254.Reg2)
			}
			d270 = d254
			if d270.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d270)
			ctx.EmitStoreToStack(d270, int32(bbs[2].PhiBase)+int32(0))
			if d254.Loc == scm.LocReg {
				ctx.UnprotectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d254.Reg)
				ctx.UnprotectReg(d254.Reg2)
			}
					ctx.EmitJmp(lbl34)
				}
			} else {
				ctx.EmitCmpRegImm32(d269.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl38)
				ctx.EmitJmp(lbl39)
				ctx.MarkLabel(lbl38)
				ctx.EmitJmp(lbl37)
				ctx.MarkLabel(lbl39)
			ctx.EnsureDesc(&d254)
			if d254.Loc == scm.LocReg {
				ctx.ProtectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.ProtectReg(d254.Reg)
				ctx.ProtectReg(d254.Reg2)
			}
			d271 = d254
			if d271.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d271)
			ctx.EmitStoreToStack(d271, int32(bbs[2].PhiBase)+int32(0))
			if d254.Loc == scm.LocReg {
				ctx.UnprotectReg(d254.Reg)
			} else if d254.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d254.Reg)
				ctx.UnprotectReg(d254.Reg2)
			}
				ctx.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d268)
			bbpos_3_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl37)
			ctx.ResolveFixups()
			d245 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d249)
			var d272 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d249.Reg)
				ctx.EmitMovRegReg(r119, d249.Reg)
				ctx.EmitShrRegImm8(r119, 6)
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d272)
			}
			if d272.Loc == scm.LocReg && d249.Loc == scm.LocReg && d272.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d272)
			var d273 scm.JITValueDesc
			if d272.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d272.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d272.Reg)
				ctx.EmitMovRegReg(scratch, d272.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d273)
			}
			if d273.Loc == scm.LocReg && d272.Loc == scm.LocReg && d273.Reg == d272.Reg {
				ctx.TransferReg(d272.Reg)
				d272.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d272)
			ctx.EnsureDesc(&d273)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d250)
			if d273.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r120, uint64(d273.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r120, d273.Reg)
				ctx.EmitShlRegImm8(r120, 3)
			}
			if d250.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
				ctx.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r120, d250.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d274 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d274)
			ctx.FreeDesc(&d273)
			ctx.EnsureDesc(&d249)
			var d275 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d249.Reg)
				ctx.EmitMovRegReg(r122, d249.Reg)
				ctx.EmitAndRegImm32(r122, 63)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d275)
			}
			if d275.Loc == scm.LocReg && d249.Loc == scm.LocReg && d275.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			var d277 scm.JITValueDesc
			if d276.Loc == scm.LocImm && d275.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d276.Imm.Int() - d275.Imm.Int())}
			} else if d275.Loc == scm.LocImm && d275.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d276.Reg)
				ctx.EmitMovRegReg(r123, d276.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d277)
			} else if d276.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d275.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d276.Imm.Int()))
				ctx.EmitSubInt64(scratch, d275.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d277)
			} else if d275.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d276.Reg)
				ctx.EmitMovRegReg(scratch, d276.Reg)
				if d275.Imm.Int() >= -2147483648 && d275.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d275.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d275.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d277)
			} else {
				r124 := ctx.AllocRegExcept(d276.Reg, d275.Reg)
				ctx.EmitMovRegReg(r124, d276.Reg)
				ctx.EmitSubInt64(r124, d275.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d277)
			}
			if d277.Loc == scm.LocReg && d276.Loc == scm.LocReg && d277.Reg == d276.Reg {
				ctx.TransferReg(d276.Reg)
				d276.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d275)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d277)
			var d278 scm.JITValueDesc
			if d274.Loc == scm.LocImm && d277.Loc == scm.LocImm {
				d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d274.Imm.Int()) >> uint64(d277.Imm.Int())))}
			} else if d277.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d274.Reg)
				ctx.EmitMovRegReg(r125, d274.Reg)
				ctx.EmitShrRegImm8(r125, uint8(d277.Imm.Int()))
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d278)
			} else {
				{
					shiftSrc := d274.Reg
					r126 := ctx.AllocRegExcept(d274.Reg)
					ctx.EmitMovRegReg(r126, d274.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d277.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d277.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d277.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d278)
				}
			}
			if d278.Loc == scm.LocReg && d274.Loc == scm.LocReg && d278.Reg == d274.Reg {
				ctx.TransferReg(d274.Reg)
				d274.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d274)
			ctx.FreeDesc(&d277)
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d278)
			var d279 scm.JITValueDesc
			if d254.Loc == scm.LocImm && d278.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d254.Imm.Int() | d278.Imm.Int())}
			} else if d254.Loc == scm.LocImm && d254.Imm.Int() == 0 {
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d278.Reg}
				ctx.BindReg(d278.Reg, &d279)
			} else if d278.Loc == scm.LocImm && d278.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d254.Reg)
				ctx.EmitMovRegReg(r127, d254.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d279)
			} else if d254.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d278.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d254.Imm.Int()))
				ctx.EmitOrInt64(scratch, d278.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d279)
			} else if d278.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d254.Reg)
				ctx.EmitMovRegReg(r128, d254.Reg)
				if d278.Imm.Int() >= -2147483648 && d278.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r128, int32(d278.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d278.Imm.Int()))
					ctx.EmitOrInt64(r128, scm.RegR11)
				}
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d279)
			} else {
				r129 := ctx.AllocRegExcept(d254.Reg, d278.Reg)
				ctx.EmitMovRegReg(r129, d254.Reg)
				ctx.EmitOrInt64(r129, d278.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d279)
			}
			if d279.Loc == scm.LocReg && d254.Loc == scm.LocReg && d279.Reg == d254.Reg {
				ctx.TransferReg(d254.Reg)
				d254.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d278)
			ctx.EnsureDesc(&d279)
			if d279.Loc == scm.LocReg {
				ctx.ProtectReg(d279.Reg)
			} else if d279.Loc == scm.LocRegPair {
				ctx.ProtectReg(d279.Reg)
				ctx.ProtectReg(d279.Reg2)
			}
			d280 = d279
			if d280.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d280)
			ctx.EmitStoreToStack(d280, int32(bbs[2].PhiBase)+int32(0))
			if d279.Loc == scm.LocReg {
				ctx.UnprotectReg(d279.Reg)
			} else if d279.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d279.Reg)
				ctx.UnprotectReg(d279.Reg2)
			}
			ctx.EmitJmp(lbl34)
			ctx.MarkLabel(lbl32)
			d281 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d281)
			ctx.BindReg(r112, &d281)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d281)
			var d282 scm.JITValueDesc
			if d281.Loc == scm.LocImm {
				d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d281.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.EmitMovRegReg(r130, d281.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d282)
			}
			ctx.FreeDesc(&d281)
			var d283 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r131 := ctx.AllocReg()
				ctx.EmitMovRegMem(r131, thisptr.Reg, off)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d283)
			}
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d283)
			var d284 scm.JITValueDesc
			if d282.Loc == scm.LocImm && d283.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d282.Imm.Int() + d283.Imm.Int())}
			} else if d283.Loc == scm.LocImm && d283.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d282.Reg)
				ctx.EmitMovRegReg(r132, d282.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d284)
			} else if d282.Loc == scm.LocImm && d282.Imm.Int() == 0 {
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d283.Reg}
				ctx.BindReg(d283.Reg, &d284)
			} else if d282.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d283.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d282.Imm.Int()))
				ctx.EmitAddInt64(scratch, d283.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			} else if d283.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d282.Reg)
				ctx.EmitMovRegReg(scratch, d282.Reg)
				if d283.Imm.Int() >= -2147483648 && d283.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d283.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d283.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			} else {
				r133 := ctx.AllocRegExcept(d282.Reg, d283.Reg)
				ctx.EmitMovRegReg(r133, d282.Reg)
				ctx.EmitAddInt64(r133, d283.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d284)
			}
			if d284.Loc == scm.LocReg && d282.Loc == scm.LocReg && d284.Reg == d282.Reg {
				ctx.TransferReg(d282.Reg)
				d282.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d282)
			ctx.FreeDesc(&d283)
			ctx.EnsureDesc(&d284)
			ctx.EnsureDesc(&d284)
			var d285 scm.JITValueDesc
			if d284.Loc == scm.LocImm {
				d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d284.Imm.Int()))))}
			} else {
				r134 := ctx.AllocReg()
				ctx.EmitMovRegReg(r134, d284.Reg)
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d285)
			}
			ctx.FreeDesc(&d284)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d50)
			var d286 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d50.Imm.Int()))))}
			} else {
				r135 := ctx.AllocReg()
				ctx.EmitMovRegReg(r135, d50.Reg)
				d286 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d286)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d285)
			var d287 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d285.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() + d285.Imm.Int())}
			} else if d285.Loc == scm.LocImm && d285.Imm.Int() == 0 {
				r136 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(r136, d50.Reg)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d287)
			} else if d50.Loc == scm.LocImm && d50.Imm.Int() == 0 {
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d285.Reg}
				ctx.BindReg(d285.Reg, &d287)
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d285.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
				ctx.EmitAddInt64(scratch, d285.Reg)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d287)
			} else if d285.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitMovRegReg(scratch, d50.Reg)
				if d285.Imm.Int() >= -2147483648 && d285.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d285.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d285.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d287)
			} else {
				r137 := ctx.AllocRegExcept(d50.Reg, d285.Reg)
				ctx.EmitMovRegReg(r137, d50.Reg)
				ctx.EmitAddInt64(r137, d285.Reg)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d287)
			}
			if d287.Loc == scm.LocReg && d50.Loc == scm.LocReg && d287.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d285)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d287)
			var d288 scm.JITValueDesc
			if d287.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d287.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.EmitMovRegReg(r138, d287.Reg)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d288)
			}
			ctx.FreeDesc(&d287)
			var d289 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r139 := ctx.AllocReg()
				r140 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r139, fieldAddr)
				ctx.EmitMovRegMem64(r140, fieldAddr+8)
				d289 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r139, Reg2: r140}
				ctx.BindReg(r139, &d289)
				ctx.BindReg(r140, &d289)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r141 := ctx.AllocReg()
				r142 := ctx.AllocReg()
				ctx.EmitMovRegMem(r141, thisptr.Reg, off)
				ctx.EmitMovRegMem(r142, thisptr.Reg, off+8)
				d289 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r141, Reg2: r142}
				ctx.BindReg(r141, &d289)
				ctx.BindReg(r142, &d289)
			}
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d288)
			ctx.EnsureDesc(&d289)
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d288)
			r143 := ctx.AllocReg()
			r144 := ctx.AllocRegExcept(r143)
			ctx.EnsureDesc(&d289)
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d288)
			if d289.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r143, uint64(d289.Imm.Int()))
			} else if d289.Loc == scm.LocRegPair {
				ctx.EmitMovRegReg(r143, d289.Reg)
			} else {
				ctx.EmitMovRegReg(r143, d289.Reg)
			}
			if d286.Loc == scm.LocImm {
				if d286.Imm.Int() != 0 {
					if d286.Imm.Int() >= -2147483648 && d286.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(r143, int32(d286.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(scm.RegR11, uint64(d286.Imm.Int()))
						ctx.EmitAddInt64(r143, scm.RegR11)
					}
				}
			} else {
				ctx.EmitAddInt64(r143, d286.Reg)
			}
			if d288.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r144, uint64(d288.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r144, d288.Reg)
			}
			if d286.Loc == scm.LocImm {
				if d286.Imm.Int() >= -2147483648 && d286.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(r144, int32(d286.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d286.Imm.Int()))
					ctx.EmitSubInt64(r144, scm.RegR11)
				}
			} else {
				ctx.EmitSubInt64(r144, d286.Reg)
			}
			d290 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r143, Reg2: r144}
			ctx.BindReg(r143, &d290)
			ctx.BindReg(r144, &d290)
			ctx.FreeDesc(&d286)
			ctx.FreeDesc(&d288)
			d291 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d291)
			ctx.BindReg(r1, &d291)
			d292 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d290}, 2)
			ctx.EmitMovPairToResult(&d292, &d291)
			ctx.EmitJmp(lbl0)
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			ctx.ReclaimUntrackedRegs()
			var d293 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r145 := ctx.AllocReg()
				ctx.EmitMovRegMem(r145, thisptr.Reg, off)
				d293 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
				ctx.BindReg(r145, &d293)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d293)
			var d294 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d293.Loc == scm.LocImm {
				d294 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d50.Imm.Int()) == uint64(d293.Imm.Int()))}
			} else if d293.Loc == scm.LocImm {
				r146 := ctx.AllocRegExcept(d50.Reg)
				if d293.Imm.Int() >= -2147483648 && d293.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d50.Reg, int32(d293.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d293.Imm.Int()))
					ctx.EmitCmpInt64(d50.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r146, scm.CcE)
				d294 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r146}
				ctx.BindReg(r146, &d294)
			} else if d50.Loc == scm.LocImm {
				r147 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d293.Reg)
				ctx.EmitSetcc(r147, scm.CcE)
				d294 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r147}
				ctx.BindReg(r147, &d294)
			} else {
				r148 := ctx.AllocRegExcept(d50.Reg)
				ctx.EmitCmpInt64(d50.Reg, d293.Reg)
				ctx.EmitSetcc(r148, scm.CcE)
				d294 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r148}
				ctx.BindReg(r148, &d294)
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d293)
			d295 = d294
			ctx.EnsureDesc(&d295)
			if d295.Loc != scm.LocImm && d295.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d295.Loc == scm.LocImm {
				if d295.Imm.Bool() {
			ps296 := scm.PhiState{General: ps.General}
			ps296.OverlayValues = make([]scm.JITValueDesc, 296)
			ps296.OverlayValues[0] = d0
			ps296.OverlayValues[1] = d1
			ps296.OverlayValues[9] = d9
			ps296.OverlayValues[10] = d10
			ps296.OverlayValues[11] = d11
			ps296.OverlayValues[12] = d12
			ps296.OverlayValues[13] = d13
			ps296.OverlayValues[14] = d14
			ps296.OverlayValues[15] = d15
			ps296.OverlayValues[16] = d16
			ps296.OverlayValues[17] = d17
			ps296.OverlayValues[18] = d18
			ps296.OverlayValues[19] = d19
			ps296.OverlayValues[20] = d20
			ps296.OverlayValues[21] = d21
			ps296.OverlayValues[22] = d22
			ps296.OverlayValues[23] = d23
			ps296.OverlayValues[24] = d24
			ps296.OverlayValues[25] = d25
			ps296.OverlayValues[26] = d26
			ps296.OverlayValues[27] = d27
			ps296.OverlayValues[28] = d28
			ps296.OverlayValues[29] = d29
			ps296.OverlayValues[30] = d30
			ps296.OverlayValues[31] = d31
			ps296.OverlayValues[32] = d32
			ps296.OverlayValues[33] = d33
			ps296.OverlayValues[34] = d34
			ps296.OverlayValues[35] = d35
			ps296.OverlayValues[36] = d36
			ps296.OverlayValues[37] = d37
			ps296.OverlayValues[38] = d38
			ps296.OverlayValues[39] = d39
			ps296.OverlayValues[40] = d40
			ps296.OverlayValues[41] = d41
			ps296.OverlayValues[42] = d42
			ps296.OverlayValues[43] = d43
			ps296.OverlayValues[44] = d44
			ps296.OverlayValues[45] = d45
			ps296.OverlayValues[46] = d46
			ps296.OverlayValues[47] = d47
			ps296.OverlayValues[48] = d48
			ps296.OverlayValues[49] = d49
			ps296.OverlayValues[50] = d50
			ps296.OverlayValues[51] = d51
			ps296.OverlayValues[52] = d52
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
			ps296.OverlayValues[147] = d147
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
			ps296.OverlayValues[272] = d272
			ps296.OverlayValues[273] = d273
			ps296.OverlayValues[274] = d274
			ps296.OverlayValues[275] = d275
			ps296.OverlayValues[276] = d276
			ps296.OverlayValues[277] = d277
			ps296.OverlayValues[278] = d278
			ps296.OverlayValues[279] = d279
			ps296.OverlayValues[280] = d280
			ps296.OverlayValues[281] = d281
			ps296.OverlayValues[282] = d282
			ps296.OverlayValues[283] = d283
			ps296.OverlayValues[284] = d284
			ps296.OverlayValues[285] = d285
			ps296.OverlayValues[286] = d286
			ps296.OverlayValues[287] = d287
			ps296.OverlayValues[288] = d288
			ps296.OverlayValues[289] = d289
			ps296.OverlayValues[290] = d290
			ps296.OverlayValues[291] = d291
			ps296.OverlayValues[292] = d292
			ps296.OverlayValues[293] = d293
			ps296.OverlayValues[294] = d294
			ps296.OverlayValues[295] = d295
					return bbs[3].RenderPS(ps296)
				}
			ps297 := scm.PhiState{General: ps.General}
			ps297.OverlayValues = make([]scm.JITValueDesc, 296)
			ps297.OverlayValues[0] = d0
			ps297.OverlayValues[1] = d1
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
			ps297.OverlayValues[22] = d22
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
			ps297.OverlayValues[147] = d147
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
			ps297.OverlayValues[272] = d272
			ps297.OverlayValues[273] = d273
			ps297.OverlayValues[274] = d274
			ps297.OverlayValues[275] = d275
			ps297.OverlayValues[276] = d276
			ps297.OverlayValues[277] = d277
			ps297.OverlayValues[278] = d278
			ps297.OverlayValues[279] = d279
			ps297.OverlayValues[280] = d280
			ps297.OverlayValues[281] = d281
			ps297.OverlayValues[282] = d282
			ps297.OverlayValues[283] = d283
			ps297.OverlayValues[284] = d284
			ps297.OverlayValues[285] = d285
			ps297.OverlayValues[286] = d286
			ps297.OverlayValues[287] = d287
			ps297.OverlayValues[288] = d288
			ps297.OverlayValues[289] = d289
			ps297.OverlayValues[290] = d290
			ps297.OverlayValues[291] = d291
			ps297.OverlayValues[292] = d292
			ps297.OverlayValues[293] = d293
			ps297.OverlayValues[294] = d294
			ps297.OverlayValues[295] = d295
				return bbs[4].RenderPS(ps297)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl40 := ctx.ReserveLabel()
			lbl41 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d295.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl40)
			ctx.EmitJmp(lbl41)
			ctx.MarkLabel(lbl40)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl41)
			ctx.EmitJmp(lbl5)
			ps298 := scm.PhiState{General: true}
			ps298.OverlayValues = make([]scm.JITValueDesc, 296)
			ps298.OverlayValues[0] = d0
			ps298.OverlayValues[1] = d1
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
			ps298.OverlayValues[22] = d22
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
			ps298.OverlayValues[104] = d104
			ps298.OverlayValues[105] = d105
			ps298.OverlayValues[106] = d106
			ps298.OverlayValues[107] = d107
			ps298.OverlayValues[108] = d108
			ps298.OverlayValues[109] = d109
			ps298.OverlayValues[110] = d110
			ps298.OverlayValues[111] = d111
			ps298.OverlayValues[112] = d112
			ps298.OverlayValues[113] = d113
			ps298.OverlayValues[114] = d114
			ps298.OverlayValues[115] = d115
			ps298.OverlayValues[116] = d116
			ps298.OverlayValues[117] = d117
			ps298.OverlayValues[118] = d118
			ps298.OverlayValues[119] = d119
			ps298.OverlayValues[120] = d120
			ps298.OverlayValues[121] = d121
			ps298.OverlayValues[122] = d122
			ps298.OverlayValues[123] = d123
			ps298.OverlayValues[124] = d124
			ps298.OverlayValues[125] = d125
			ps298.OverlayValues[126] = d126
			ps298.OverlayValues[127] = d127
			ps298.OverlayValues[128] = d128
			ps298.OverlayValues[129] = d129
			ps298.OverlayValues[130] = d130
			ps298.OverlayValues[131] = d131
			ps298.OverlayValues[132] = d132
			ps298.OverlayValues[133] = d133
			ps298.OverlayValues[134] = d134
			ps298.OverlayValues[135] = d135
			ps298.OverlayValues[136] = d136
			ps298.OverlayValues[137] = d137
			ps298.OverlayValues[138] = d138
			ps298.OverlayValues[139] = d139
			ps298.OverlayValues[140] = d140
			ps298.OverlayValues[141] = d141
			ps298.OverlayValues[142] = d142
			ps298.OverlayValues[143] = d143
			ps298.OverlayValues[144] = d144
			ps298.OverlayValues[145] = d145
			ps298.OverlayValues[146] = d146
			ps298.OverlayValues[147] = d147
			ps298.OverlayValues[243] = d243
			ps298.OverlayValues[244] = d244
			ps298.OverlayValues[245] = d245
			ps298.OverlayValues[246] = d246
			ps298.OverlayValues[247] = d247
			ps298.OverlayValues[248] = d248
			ps298.OverlayValues[249] = d249
			ps298.OverlayValues[250] = d250
			ps298.OverlayValues[251] = d251
			ps298.OverlayValues[252] = d252
			ps298.OverlayValues[253] = d253
			ps298.OverlayValues[254] = d254
			ps298.OverlayValues[255] = d255
			ps298.OverlayValues[256] = d256
			ps298.OverlayValues[257] = d257
			ps298.OverlayValues[258] = d258
			ps298.OverlayValues[259] = d259
			ps298.OverlayValues[260] = d260
			ps298.OverlayValues[261] = d261
			ps298.OverlayValues[262] = d262
			ps298.OverlayValues[263] = d263
			ps298.OverlayValues[264] = d264
			ps298.OverlayValues[265] = d265
			ps298.OverlayValues[266] = d266
			ps298.OverlayValues[267] = d267
			ps298.OverlayValues[268] = d268
			ps298.OverlayValues[269] = d269
			ps298.OverlayValues[270] = d270
			ps298.OverlayValues[271] = d271
			ps298.OverlayValues[272] = d272
			ps298.OverlayValues[273] = d273
			ps298.OverlayValues[274] = d274
			ps298.OverlayValues[275] = d275
			ps298.OverlayValues[276] = d276
			ps298.OverlayValues[277] = d277
			ps298.OverlayValues[278] = d278
			ps298.OverlayValues[279] = d279
			ps298.OverlayValues[280] = d280
			ps298.OverlayValues[281] = d281
			ps298.OverlayValues[282] = d282
			ps298.OverlayValues[283] = d283
			ps298.OverlayValues[284] = d284
			ps298.OverlayValues[285] = d285
			ps298.OverlayValues[286] = d286
			ps298.OverlayValues[287] = d287
			ps298.OverlayValues[288] = d288
			ps298.OverlayValues[289] = d289
			ps298.OverlayValues[290] = d290
			ps298.OverlayValues[291] = d291
			ps298.OverlayValues[292] = d292
			ps298.OverlayValues[293] = d293
			ps298.OverlayValues[294] = d294
			ps298.OverlayValues[295] = d295
			ps299 := scm.PhiState{General: true}
			ps299.OverlayValues = make([]scm.JITValueDesc, 296)
			ps299.OverlayValues[0] = d0
			ps299.OverlayValues[1] = d1
			ps299.OverlayValues[9] = d9
			ps299.OverlayValues[10] = d10
			ps299.OverlayValues[11] = d11
			ps299.OverlayValues[12] = d12
			ps299.OverlayValues[13] = d13
			ps299.OverlayValues[14] = d14
			ps299.OverlayValues[15] = d15
			ps299.OverlayValues[16] = d16
			ps299.OverlayValues[17] = d17
			ps299.OverlayValues[18] = d18
			ps299.OverlayValues[19] = d19
			ps299.OverlayValues[20] = d20
			ps299.OverlayValues[21] = d21
			ps299.OverlayValues[22] = d22
			ps299.OverlayValues[23] = d23
			ps299.OverlayValues[24] = d24
			ps299.OverlayValues[25] = d25
			ps299.OverlayValues[26] = d26
			ps299.OverlayValues[27] = d27
			ps299.OverlayValues[28] = d28
			ps299.OverlayValues[29] = d29
			ps299.OverlayValues[30] = d30
			ps299.OverlayValues[31] = d31
			ps299.OverlayValues[32] = d32
			ps299.OverlayValues[33] = d33
			ps299.OverlayValues[34] = d34
			ps299.OverlayValues[35] = d35
			ps299.OverlayValues[36] = d36
			ps299.OverlayValues[37] = d37
			ps299.OverlayValues[38] = d38
			ps299.OverlayValues[39] = d39
			ps299.OverlayValues[40] = d40
			ps299.OverlayValues[41] = d41
			ps299.OverlayValues[42] = d42
			ps299.OverlayValues[43] = d43
			ps299.OverlayValues[44] = d44
			ps299.OverlayValues[45] = d45
			ps299.OverlayValues[46] = d46
			ps299.OverlayValues[47] = d47
			ps299.OverlayValues[48] = d48
			ps299.OverlayValues[49] = d49
			ps299.OverlayValues[50] = d50
			ps299.OverlayValues[51] = d51
			ps299.OverlayValues[52] = d52
			ps299.OverlayValues[104] = d104
			ps299.OverlayValues[105] = d105
			ps299.OverlayValues[106] = d106
			ps299.OverlayValues[107] = d107
			ps299.OverlayValues[108] = d108
			ps299.OverlayValues[109] = d109
			ps299.OverlayValues[110] = d110
			ps299.OverlayValues[111] = d111
			ps299.OverlayValues[112] = d112
			ps299.OverlayValues[113] = d113
			ps299.OverlayValues[114] = d114
			ps299.OverlayValues[115] = d115
			ps299.OverlayValues[116] = d116
			ps299.OverlayValues[117] = d117
			ps299.OverlayValues[118] = d118
			ps299.OverlayValues[119] = d119
			ps299.OverlayValues[120] = d120
			ps299.OverlayValues[121] = d121
			ps299.OverlayValues[122] = d122
			ps299.OverlayValues[123] = d123
			ps299.OverlayValues[124] = d124
			ps299.OverlayValues[125] = d125
			ps299.OverlayValues[126] = d126
			ps299.OverlayValues[127] = d127
			ps299.OverlayValues[128] = d128
			ps299.OverlayValues[129] = d129
			ps299.OverlayValues[130] = d130
			ps299.OverlayValues[131] = d131
			ps299.OverlayValues[132] = d132
			ps299.OverlayValues[133] = d133
			ps299.OverlayValues[134] = d134
			ps299.OverlayValues[135] = d135
			ps299.OverlayValues[136] = d136
			ps299.OverlayValues[137] = d137
			ps299.OverlayValues[138] = d138
			ps299.OverlayValues[139] = d139
			ps299.OverlayValues[140] = d140
			ps299.OverlayValues[141] = d141
			ps299.OverlayValues[142] = d142
			ps299.OverlayValues[143] = d143
			ps299.OverlayValues[144] = d144
			ps299.OverlayValues[145] = d145
			ps299.OverlayValues[146] = d146
			ps299.OverlayValues[147] = d147
			ps299.OverlayValues[243] = d243
			ps299.OverlayValues[244] = d244
			ps299.OverlayValues[245] = d245
			ps299.OverlayValues[246] = d246
			ps299.OverlayValues[247] = d247
			ps299.OverlayValues[248] = d248
			ps299.OverlayValues[249] = d249
			ps299.OverlayValues[250] = d250
			ps299.OverlayValues[251] = d251
			ps299.OverlayValues[252] = d252
			ps299.OverlayValues[253] = d253
			ps299.OverlayValues[254] = d254
			ps299.OverlayValues[255] = d255
			ps299.OverlayValues[256] = d256
			ps299.OverlayValues[257] = d257
			ps299.OverlayValues[258] = d258
			ps299.OverlayValues[259] = d259
			ps299.OverlayValues[260] = d260
			ps299.OverlayValues[261] = d261
			ps299.OverlayValues[262] = d262
			ps299.OverlayValues[263] = d263
			ps299.OverlayValues[264] = d264
			ps299.OverlayValues[265] = d265
			ps299.OverlayValues[266] = d266
			ps299.OverlayValues[267] = d267
			ps299.OverlayValues[268] = d268
			ps299.OverlayValues[269] = d269
			ps299.OverlayValues[270] = d270
			ps299.OverlayValues[271] = d271
			ps299.OverlayValues[272] = d272
			ps299.OverlayValues[273] = d273
			ps299.OverlayValues[274] = d274
			ps299.OverlayValues[275] = d275
			ps299.OverlayValues[276] = d276
			ps299.OverlayValues[277] = d277
			ps299.OverlayValues[278] = d278
			ps299.OverlayValues[279] = d279
			ps299.OverlayValues[280] = d280
			ps299.OverlayValues[281] = d281
			ps299.OverlayValues[282] = d282
			ps299.OverlayValues[283] = d283
			ps299.OverlayValues[284] = d284
			ps299.OverlayValues[285] = d285
			ps299.OverlayValues[286] = d286
			ps299.OverlayValues[287] = d287
			ps299.OverlayValues[288] = d288
			ps299.OverlayValues[289] = d289
			ps299.OverlayValues[290] = d290
			ps299.OverlayValues[291] = d291
			ps299.OverlayValues[292] = d292
			ps299.OverlayValues[293] = d293
			ps299.OverlayValues[294] = d294
			ps299.OverlayValues[295] = d295
			snap300 := d0
			snap301 := d1
			snap302 := d9
			snap303 := d10
			snap304 := d11
			snap305 := d12
			snap306 := d13
			snap307 := d14
			snap308 := d15
			snap309 := d16
			snap310 := d17
			snap311 := d18
			snap312 := d19
			snap313 := d20
			snap314 := d21
			snap315 := d22
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
			snap346 := d104
			snap347 := d105
			snap348 := d106
			snap349 := d107
			snap350 := d108
			snap351 := d109
			snap352 := d110
			snap353 := d111
			snap354 := d112
			snap355 := d113
			snap356 := d114
			snap357 := d115
			snap358 := d116
			snap359 := d117
			snap360 := d118
			snap361 := d119
			snap362 := d120
			snap363 := d121
			snap364 := d122
			snap365 := d123
			snap366 := d124
			snap367 := d125
			snap368 := d126
			snap369 := d127
			snap370 := d128
			snap371 := d129
			snap372 := d130
			snap373 := d131
			snap374 := d132
			snap375 := d133
			snap376 := d134
			snap377 := d135
			snap378 := d136
			snap379 := d137
			snap380 := d138
			snap381 := d139
			snap382 := d140
			snap383 := d141
			snap384 := d142
			snap385 := d143
			snap386 := d144
			snap387 := d145
			snap388 := d146
			snap389 := d147
			snap390 := d243
			snap391 := d244
			snap392 := d245
			snap393 := d246
			snap394 := d247
			snap395 := d248
			snap396 := d249
			snap397 := d250
			snap398 := d251
			snap399 := d252
			snap400 := d253
			snap401 := d254
			snap402 := d255
			snap403 := d256
			snap404 := d257
			snap405 := d258
			snap406 := d259
			snap407 := d260
			snap408 := d261
			snap409 := d262
			snap410 := d263
			snap411 := d264
			snap412 := d265
			snap413 := d266
			snap414 := d267
			snap415 := d268
			snap416 := d269
			snap417 := d270
			snap418 := d271
			snap419 := d272
			snap420 := d273
			snap421 := d274
			snap422 := d275
			snap423 := d276
			snap424 := d277
			snap425 := d278
			snap426 := d279
			snap427 := d280
			snap428 := d281
			snap429 := d282
			snap430 := d283
			snap431 := d284
			snap432 := d285
			snap433 := d286
			snap434 := d287
			snap435 := d288
			snap436 := d289
			snap437 := d290
			snap438 := d291
			snap439 := d292
			snap440 := d293
			snap441 := d294
			snap442 := d295
			alloc443 := ctx.SnapshotAllocState()
			if !bbs[4].Rendered {
				bbs[4].RenderPS(ps299)
			}
			ctx.RestoreAllocState(alloc443)
			d0 = snap300
			d1 = snap301
			d9 = snap302
			d10 = snap303
			d11 = snap304
			d12 = snap305
			d13 = snap306
			d14 = snap307
			d15 = snap308
			d16 = snap309
			d17 = snap310
			d18 = snap311
			d19 = snap312
			d20 = snap313
			d21 = snap314
			d22 = snap315
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
			d104 = snap346
			d105 = snap347
			d106 = snap348
			d107 = snap349
			d108 = snap350
			d109 = snap351
			d110 = snap352
			d111 = snap353
			d112 = snap354
			d113 = snap355
			d114 = snap356
			d115 = snap357
			d116 = snap358
			d117 = snap359
			d118 = snap360
			d119 = snap361
			d120 = snap362
			d121 = snap363
			d122 = snap364
			d123 = snap365
			d124 = snap366
			d125 = snap367
			d126 = snap368
			d127 = snap369
			d128 = snap370
			d129 = snap371
			d130 = snap372
			d131 = snap373
			d132 = snap374
			d133 = snap375
			d134 = snap376
			d135 = snap377
			d136 = snap378
			d137 = snap379
			d138 = snap380
			d139 = snap381
			d140 = snap382
			d141 = snap383
			d142 = snap384
			d143 = snap385
			d144 = snap386
			d145 = snap387
			d146 = snap388
			d147 = snap389
			d243 = snap390
			d244 = snap391
			d245 = snap392
			d246 = snap393
			d247 = snap394
			d248 = snap395
			d249 = snap396
			d250 = snap397
			d251 = snap398
			d252 = snap399
			d253 = snap400
			d254 = snap401
			d255 = snap402
			d256 = snap403
			d257 = snap404
			d258 = snap405
			d259 = snap406
			d260 = snap407
			d261 = snap408
			d262 = snap409
			d263 = snap410
			d264 = snap411
			d265 = snap412
			d266 = snap413
			d267 = snap414
			d268 = snap415
			d269 = snap416
			d270 = snap417
			d271 = snap418
			d272 = snap419
			d273 = snap420
			d274 = snap421
			d275 = snap422
			d276 = snap423
			d277 = snap424
			d278 = snap425
			d279 = snap426
			d280 = snap427
			d281 = snap428
			d282 = snap429
			d283 = snap430
			d284 = snap431
			d285 = snap432
			d286 = snap433
			d287 = snap434
			d288 = snap435
			d289 = snap436
			d290 = snap437
			d291 = snap438
			d292 = snap439
			d293 = snap440
			d294 = snap441
			d295 = snap442
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps298)
			}
			return result
			ctx.FreeDesc(&d294)
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			ctx.ReclaimUntrackedRegs()
			d444 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d444)
			ctx.BindReg(r1, &d444)
			ctx.EmitMakeNil(d444)
			ctx.EmitJmp(lbl0)
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
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			if len(ps.OverlayValues) > 444 && ps.OverlayValues[444].Loc != scm.LocNone {
				d444 = ps.OverlayValues[444]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d145)
			d445 = d145
			_ = d445
			r149 := d145.Loc == scm.LocReg
			r150 := d145.Reg
			if r149 { ctx.ProtectReg(r150) }
			d446 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			lbl42 := ctx.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d446 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d445)
			ctx.EnsureDesc(&d445)
			var d447 scm.JITValueDesc
			if d445.Loc == scm.LocImm {
				d447 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d445.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.EmitMovRegReg(r151, d445.Reg)
				ctx.EmitShlRegImm8(r151, 32)
				ctx.EmitShrRegImm8(r151, 32)
				d447 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d447)
			}
			var d448 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d448 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r152 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r152, thisptr.Reg, off)
				d448 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
				ctx.BindReg(r152, &d448)
			}
			ctx.EnsureDesc(&d448)
			ctx.EnsureDesc(&d448)
			var d449 scm.JITValueDesc
			if d448.Loc == scm.LocImm {
				d449 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d448.Imm.Int()))))}
			} else {
				r153 := ctx.AllocReg()
				ctx.EmitMovRegReg(r153, d448.Reg)
				ctx.EmitShlRegImm8(r153, 56)
				ctx.EmitShrRegImm8(r153, 56)
				d449 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d449)
			}
			ctx.FreeDesc(&d448)
			ctx.EnsureDesc(&d447)
			ctx.EnsureDesc(&d449)
			ctx.EnsureDesc(&d447)
			ctx.EnsureDesc(&d449)
			ctx.EnsureDesc(&d447)
			ctx.EnsureDesc(&d449)
			var d450 scm.JITValueDesc
			if d447.Loc == scm.LocImm && d449.Loc == scm.LocImm {
				d450 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d447.Imm.Int() * d449.Imm.Int())}
			} else if d447.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d449.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d447.Imm.Int()))
				ctx.EmitImulInt64(scratch, d449.Reg)
				d450 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d450)
			} else if d449.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d447.Reg)
				ctx.EmitMovRegReg(scratch, d447.Reg)
				if d449.Imm.Int() >= -2147483648 && d449.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d449.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d449.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d450 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d450)
			} else {
				r154 := ctx.AllocRegExcept(d447.Reg, d449.Reg)
				ctx.EmitMovRegReg(r154, d447.Reg)
				ctx.EmitImulInt64(r154, d449.Reg)
				d450 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d450)
			}
			if d450.Loc == scm.LocReg && d447.Loc == scm.LocReg && d450.Reg == d447.Reg {
				ctx.TransferReg(d447.Reg)
				d447.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d447)
			ctx.FreeDesc(&d449)
			var d451 scm.JITValueDesc
			r155 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r155, uint64(dataPtr))
				d451 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155, StackOff: int32(sliceLen)}
				ctx.BindReg(r155, &d451)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.EmitMovRegMem(r155, thisptr.Reg, off)
				d451 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
				ctx.BindReg(r155, &d451)
			}
			ctx.BindReg(r155, &d451)
			ctx.EnsureDesc(&d450)
			var d452 scm.JITValueDesc
			if d450.Loc == scm.LocImm {
				d452 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() / 64)}
			} else {
				r156 := ctx.AllocRegExcept(d450.Reg)
				ctx.EmitMovRegReg(r156, d450.Reg)
				ctx.EmitShrRegImm8(r156, 6)
				d452 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d452)
			}
			if d452.Loc == scm.LocReg && d450.Loc == scm.LocReg && d452.Reg == d450.Reg {
				ctx.TransferReg(d450.Reg)
				d450.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d452)
			r157 := ctx.AllocReg()
			ctx.EnsureDesc(&d452)
			ctx.EnsureDesc(&d451)
			if d452.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r157, uint64(d452.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r157, d452.Reg)
				ctx.EmitShlRegImm8(r157, 3)
			}
			if d451.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d451.Imm.Int()))
				ctx.EmitAddInt64(r157, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r157, d451.Reg)
			}
			r158 := ctx.AllocRegExcept(r157)
			ctx.EmitMovRegMem(r158, r157, 0)
			ctx.FreeReg(r157)
			d453 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
			ctx.BindReg(r158, &d453)
			ctx.FreeDesc(&d452)
			ctx.EnsureDesc(&d450)
			var d454 scm.JITValueDesc
			if d450.Loc == scm.LocImm {
				d454 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() % 64)}
			} else {
				r159 := ctx.AllocRegExcept(d450.Reg)
				ctx.EmitMovRegReg(r159, d450.Reg)
				ctx.EmitAndRegImm32(r159, 63)
				d454 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d454)
			}
			if d454.Loc == scm.LocReg && d450.Loc == scm.LocReg && d454.Reg == d450.Reg {
				ctx.TransferReg(d450.Reg)
				d450.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d453)
			ctx.EnsureDesc(&d454)
			var d455 scm.JITValueDesc
			if d453.Loc == scm.LocImm && d454.Loc == scm.LocImm {
				d455 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d453.Imm.Int()) << uint64(d454.Imm.Int())))}
			} else if d454.Loc == scm.LocImm {
				r160 := ctx.AllocRegExcept(d453.Reg)
				ctx.EmitMovRegReg(r160, d453.Reg)
				ctx.EmitShlRegImm8(r160, uint8(d454.Imm.Int()))
				d455 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d455)
			} else {
				{
					shiftSrc := d453.Reg
					r161 := ctx.AllocRegExcept(d453.Reg)
					ctx.EmitMovRegReg(r161, d453.Reg)
					shiftSrc = r161
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d454.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d454.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d454.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d455 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d455)
				}
			}
			if d455.Loc == scm.LocReg && d453.Loc == scm.LocReg && d455.Reg == d453.Reg {
				ctx.TransferReg(d453.Reg)
				d453.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d453)
			ctx.FreeDesc(&d454)
			var d456 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d456 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r162 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r162, thisptr.Reg, off)
				d456 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r162}
				ctx.BindReg(r162, &d456)
			}
			d457 = d456
			ctx.EnsureDesc(&d457)
			if d457.Loc != scm.LocImm && d457.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.ReserveLabel()
			lbl44 := ctx.ReserveLabel()
			lbl45 := ctx.ReserveLabel()
			lbl46 := ctx.ReserveLabel()
			if d457.Loc == scm.LocImm {
				if d457.Imm.Bool() {
					ctx.MarkLabel(lbl45)
					ctx.EmitJmp(lbl43)
				} else {
					ctx.MarkLabel(lbl46)
			ctx.EnsureDesc(&d455)
			if d455.Loc == scm.LocReg {
				ctx.ProtectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.ProtectReg(d455.Reg)
				ctx.ProtectReg(d455.Reg2)
			}
			d458 = d455
			if d458.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d458)
			ctx.EmitStoreToStack(d458, int32(bbs[2].PhiBase)+int32(0))
			if d455.Loc == scm.LocReg {
				ctx.UnprotectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d455.Reg)
				ctx.UnprotectReg(d455.Reg2)
			}
					ctx.EmitJmp(lbl44)
				}
			} else {
				ctx.EmitCmpRegImm32(d457.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl45)
				ctx.EmitJmp(lbl46)
				ctx.MarkLabel(lbl45)
				ctx.EmitJmp(lbl43)
				ctx.MarkLabel(lbl46)
			ctx.EnsureDesc(&d455)
			if d455.Loc == scm.LocReg {
				ctx.ProtectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.ProtectReg(d455.Reg)
				ctx.ProtectReg(d455.Reg2)
			}
			d459 = d455
			if d459.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d459)
			ctx.EmitStoreToStack(d459, int32(bbs[2].PhiBase)+int32(0))
			if d455.Loc == scm.LocReg {
				ctx.UnprotectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d455.Reg)
				ctx.UnprotectReg(d455.Reg2)
			}
				ctx.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d456)
			bbpos_4_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl44)
			ctx.ResolveFixups()
			d446 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			var d460 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d460 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r163 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r163, thisptr.Reg, off)
				d460 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
				ctx.BindReg(r163, &d460)
			}
			ctx.EnsureDesc(&d460)
			ctx.EnsureDesc(&d460)
			var d461 scm.JITValueDesc
			if d460.Loc == scm.LocImm {
				d461 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d460.Imm.Int()))))}
			} else {
				r164 := ctx.AllocReg()
				ctx.EmitMovRegReg(r164, d460.Reg)
				ctx.EmitShlRegImm8(r164, 56)
				ctx.EmitShrRegImm8(r164, 56)
				d461 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d461)
			}
			ctx.FreeDesc(&d460)
			d462 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d462)
			ctx.EnsureDesc(&d461)
			ctx.EnsureDesc(&d462)
			ctx.EnsureDesc(&d461)
			var d463 scm.JITValueDesc
			if d462.Loc == scm.LocImm && d461.Loc == scm.LocImm {
				d463 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d462.Imm.Int() - d461.Imm.Int())}
			} else if d461.Loc == scm.LocImm && d461.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d462.Reg)
				ctx.EmitMovRegReg(r165, d462.Reg)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d463)
			} else if d462.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d461.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d462.Imm.Int()))
				ctx.EmitSubInt64(scratch, d461.Reg)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d463)
			} else if d461.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d462.Reg)
				ctx.EmitMovRegReg(scratch, d462.Reg)
				if d461.Imm.Int() >= -2147483648 && d461.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d461.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d461.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d463)
			} else {
				r166 := ctx.AllocRegExcept(d462.Reg, d461.Reg)
				ctx.EmitMovRegReg(r166, d462.Reg)
				ctx.EmitSubInt64(r166, d461.Reg)
				d463 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d463)
			}
			if d463.Loc == scm.LocReg && d462.Loc == scm.LocReg && d463.Reg == d462.Reg {
				ctx.TransferReg(d462.Reg)
				d462.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d461)
			ctx.EnsureDesc(&d446)
			ctx.EnsureDesc(&d463)
			var d464 scm.JITValueDesc
			if d446.Loc == scm.LocImm && d463.Loc == scm.LocImm {
				d464 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d446.Imm.Int()) >> uint64(d463.Imm.Int())))}
			} else if d463.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d446.Reg)
				ctx.EmitMovRegReg(r167, d446.Reg)
				ctx.EmitShrRegImm8(r167, uint8(d463.Imm.Int()))
				d464 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d464)
			} else {
				{
					shiftSrc := d446.Reg
					r168 := ctx.AllocRegExcept(d446.Reg)
					ctx.EmitMovRegReg(r168, d446.Reg)
					shiftSrc = r168
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d463.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d463.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d463.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d464 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d464)
				}
			}
			if d464.Loc == scm.LocReg && d446.Loc == scm.LocReg && d464.Reg == d446.Reg {
				ctx.TransferReg(d446.Reg)
				d446.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d446)
			ctx.FreeDesc(&d463)
			r169 := ctx.AllocReg()
			ctx.EnsureDesc(&d464)
			ctx.EnsureDesc(&d464)
			if d464.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r169, d464)
			}
			ctx.EmitJmp(lbl42)
			bbpos_4_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl43)
			ctx.ResolveFixups()
			d446 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d450)
			var d465 scm.JITValueDesc
			if d450.Loc == scm.LocImm {
				d465 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() % 64)}
			} else {
				r170 := ctx.AllocRegExcept(d450.Reg)
				ctx.EmitMovRegReg(r170, d450.Reg)
				ctx.EmitAndRegImm32(r170, 63)
				d465 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d465)
			}
			if d465.Loc == scm.LocReg && d450.Loc == scm.LocReg && d465.Reg == d450.Reg {
				ctx.TransferReg(d450.Reg)
				d450.Loc = scm.LocNone
			}
			var d466 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d466 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r171 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r171, thisptr.Reg, off)
				d466 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
				ctx.BindReg(r171, &d466)
			}
			ctx.EnsureDesc(&d466)
			ctx.EnsureDesc(&d466)
			var d467 scm.JITValueDesc
			if d466.Loc == scm.LocImm {
				d467 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d466.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.EmitMovRegReg(r172, d466.Reg)
				ctx.EmitShlRegImm8(r172, 56)
				ctx.EmitShrRegImm8(r172, 56)
				d467 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d467)
			}
			ctx.FreeDesc(&d466)
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d467)
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d467)
			ctx.EnsureDesc(&d465)
			ctx.EnsureDesc(&d467)
			var d468 scm.JITValueDesc
			if d465.Loc == scm.LocImm && d467.Loc == scm.LocImm {
				d468 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d465.Imm.Int() + d467.Imm.Int())}
			} else if d467.Loc == scm.LocImm && d467.Imm.Int() == 0 {
				r173 := ctx.AllocRegExcept(d465.Reg)
				ctx.EmitMovRegReg(r173, d465.Reg)
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d468)
			} else if d465.Loc == scm.LocImm && d465.Imm.Int() == 0 {
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d467.Reg}
				ctx.BindReg(d467.Reg, &d468)
			} else if d465.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d467.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d465.Imm.Int()))
				ctx.EmitAddInt64(scratch, d467.Reg)
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d468)
			} else if d467.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d465.Reg)
				ctx.EmitMovRegReg(scratch, d465.Reg)
				if d467.Imm.Int() >= -2147483648 && d467.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d467.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d467.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d468)
			} else {
				r174 := ctx.AllocRegExcept(d465.Reg, d467.Reg)
				ctx.EmitMovRegReg(r174, d465.Reg)
				ctx.EmitAddInt64(r174, d467.Reg)
				d468 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d468)
			}
			if d468.Loc == scm.LocReg && d465.Loc == scm.LocReg && d468.Reg == d465.Reg {
				ctx.TransferReg(d465.Reg)
				d465.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d465)
			ctx.FreeDesc(&d467)
			ctx.EnsureDesc(&d468)
			var d469 scm.JITValueDesc
			if d468.Loc == scm.LocImm {
				d469 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d468.Imm.Int()) > uint64(64))}
			} else {
				r175 := ctx.AllocRegExcept(d468.Reg)
				ctx.EmitCmpRegImm32(d468.Reg, 64)
				ctx.EmitSetcc(r175, scm.CcA)
				d469 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r175}
				ctx.BindReg(r175, &d469)
			}
			ctx.FreeDesc(&d468)
			d470 = d469
			ctx.EnsureDesc(&d470)
			if d470.Loc != scm.LocImm && d470.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.ReserveLabel()
			lbl48 := ctx.ReserveLabel()
			lbl49 := ctx.ReserveLabel()
			if d470.Loc == scm.LocImm {
				if d470.Imm.Bool() {
					ctx.MarkLabel(lbl48)
					ctx.EmitJmp(lbl47)
				} else {
					ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d455)
			if d455.Loc == scm.LocReg {
				ctx.ProtectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.ProtectReg(d455.Reg)
				ctx.ProtectReg(d455.Reg2)
			}
			d471 = d455
			if d471.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d471)
			ctx.EmitStoreToStack(d471, int32(bbs[2].PhiBase)+int32(0))
			if d455.Loc == scm.LocReg {
				ctx.UnprotectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d455.Reg)
				ctx.UnprotectReg(d455.Reg2)
			}
					ctx.EmitJmp(lbl44)
				}
			} else {
				ctx.EmitCmpRegImm32(d470.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl48)
				ctx.EmitJmp(lbl49)
				ctx.MarkLabel(lbl48)
				ctx.EmitJmp(lbl47)
				ctx.MarkLabel(lbl49)
			ctx.EnsureDesc(&d455)
			if d455.Loc == scm.LocReg {
				ctx.ProtectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.ProtectReg(d455.Reg)
				ctx.ProtectReg(d455.Reg2)
			}
			d472 = d455
			if d472.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d472)
			ctx.EmitStoreToStack(d472, int32(bbs[2].PhiBase)+int32(0))
			if d455.Loc == scm.LocReg {
				ctx.UnprotectReg(d455.Reg)
			} else if d455.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d455.Reg)
				ctx.UnprotectReg(d455.Reg2)
			}
				ctx.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d469)
			bbpos_4_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl47)
			ctx.ResolveFixups()
			d446 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(48)}
			ctx.EnsureDesc(&d450)
			var d473 scm.JITValueDesc
			if d450.Loc == scm.LocImm {
				d473 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() / 64)}
			} else {
				r176 := ctx.AllocRegExcept(d450.Reg)
				ctx.EmitMovRegReg(r176, d450.Reg)
				ctx.EmitShrRegImm8(r176, 6)
				d473 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d473)
			}
			if d473.Loc == scm.LocReg && d450.Loc == scm.LocReg && d473.Reg == d450.Reg {
				ctx.TransferReg(d450.Reg)
				d450.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d473)
			ctx.EnsureDesc(&d473)
			var d474 scm.JITValueDesc
			if d473.Loc == scm.LocImm {
				d474 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d473.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d473.Reg)
				ctx.EmitMovRegReg(scratch, d473.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d474 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d474)
			}
			if d474.Loc == scm.LocReg && d473.Loc == scm.LocReg && d474.Reg == d473.Reg {
				ctx.TransferReg(d473.Reg)
				d473.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d473)
			ctx.EnsureDesc(&d474)
			r177 := ctx.AllocReg()
			ctx.EnsureDesc(&d474)
			ctx.EnsureDesc(&d451)
			if d474.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r177, uint64(d474.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r177, d474.Reg)
				ctx.EmitShlRegImm8(r177, 3)
			}
			if d451.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d451.Imm.Int()))
				ctx.EmitAddInt64(r177, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r177, d451.Reg)
			}
			r178 := ctx.AllocRegExcept(r177)
			ctx.EmitMovRegMem(r178, r177, 0)
			ctx.FreeReg(r177)
			d475 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r178}
			ctx.BindReg(r178, &d475)
			ctx.FreeDesc(&d474)
			ctx.EnsureDesc(&d450)
			var d476 scm.JITValueDesc
			if d450.Loc == scm.LocImm {
				d476 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d450.Imm.Int() % 64)}
			} else {
				r179 := ctx.AllocRegExcept(d450.Reg)
				ctx.EmitMovRegReg(r179, d450.Reg)
				ctx.EmitAndRegImm32(r179, 63)
				d476 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d476)
			}
			if d476.Loc == scm.LocReg && d450.Loc == scm.LocReg && d476.Reg == d450.Reg {
				ctx.TransferReg(d450.Reg)
				d450.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d450)
			d477 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d476)
			ctx.EnsureDesc(&d477)
			ctx.EnsureDesc(&d476)
			ctx.EnsureDesc(&d477)
			ctx.EnsureDesc(&d476)
			var d478 scm.JITValueDesc
			if d477.Loc == scm.LocImm && d476.Loc == scm.LocImm {
				d478 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d477.Imm.Int() - d476.Imm.Int())}
			} else if d476.Loc == scm.LocImm && d476.Imm.Int() == 0 {
				r180 := ctx.AllocRegExcept(d477.Reg)
				ctx.EmitMovRegReg(r180, d477.Reg)
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d478)
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
				r181 := ctx.AllocRegExcept(d477.Reg, d476.Reg)
				ctx.EmitMovRegReg(r181, d477.Reg)
				ctx.EmitSubInt64(r181, d476.Reg)
				d478 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d478)
			}
			if d478.Loc == scm.LocReg && d477.Loc == scm.LocReg && d478.Reg == d477.Reg {
				ctx.TransferReg(d477.Reg)
				d477.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d476)
			ctx.EnsureDesc(&d475)
			ctx.EnsureDesc(&d478)
			var d479 scm.JITValueDesc
			if d475.Loc == scm.LocImm && d478.Loc == scm.LocImm {
				d479 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d475.Imm.Int()) >> uint64(d478.Imm.Int())))}
			} else if d478.Loc == scm.LocImm {
				r182 := ctx.AllocRegExcept(d475.Reg)
				ctx.EmitMovRegReg(r182, d475.Reg)
				ctx.EmitShrRegImm8(r182, uint8(d478.Imm.Int()))
				d479 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d479)
			} else {
				{
					shiftSrc := d475.Reg
					r183 := ctx.AllocRegExcept(d475.Reg)
					ctx.EmitMovRegReg(r183, d475.Reg)
					shiftSrc = r183
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
			if d479.Loc == scm.LocReg && d475.Loc == scm.LocReg && d479.Reg == d475.Reg {
				ctx.TransferReg(d475.Reg)
				d475.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d475)
			ctx.FreeDesc(&d478)
			ctx.EnsureDesc(&d455)
			ctx.EnsureDesc(&d479)
			var d480 scm.JITValueDesc
			if d455.Loc == scm.LocImm && d479.Loc == scm.LocImm {
				d480 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d455.Imm.Int() | d479.Imm.Int())}
			} else if d455.Loc == scm.LocImm && d455.Imm.Int() == 0 {
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d479.Reg}
				ctx.BindReg(d479.Reg, &d480)
			} else if d479.Loc == scm.LocImm && d479.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d455.Reg)
				ctx.EmitMovRegReg(r184, d455.Reg)
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d480)
			} else if d455.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d479.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d455.Imm.Int()))
				ctx.EmitOrInt64(scratch, d479.Reg)
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d480)
			} else if d479.Loc == scm.LocImm {
				r185 := ctx.AllocRegExcept(d455.Reg)
				ctx.EmitMovRegReg(r185, d455.Reg)
				if d479.Imm.Int() >= -2147483648 && d479.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r185, int32(d479.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d479.Imm.Int()))
					ctx.EmitOrInt64(r185, scm.RegR11)
				}
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d480)
			} else {
				r186 := ctx.AllocRegExcept(d455.Reg, d479.Reg)
				ctx.EmitMovRegReg(r186, d455.Reg)
				ctx.EmitOrInt64(r186, d479.Reg)
				d480 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d480)
			}
			if d480.Loc == scm.LocReg && d455.Loc == scm.LocReg && d480.Reg == d455.Reg {
				ctx.TransferReg(d455.Reg)
				d455.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d479)
			ctx.EnsureDesc(&d480)
			if d480.Loc == scm.LocReg {
				ctx.ProtectReg(d480.Reg)
			} else if d480.Loc == scm.LocRegPair {
				ctx.ProtectReg(d480.Reg)
				ctx.ProtectReg(d480.Reg2)
			}
			d481 = d480
			if d481.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d481)
			ctx.EmitStoreToStack(d481, int32(bbs[2].PhiBase)+int32(0))
			if d480.Loc == scm.LocReg {
				ctx.UnprotectReg(d480.Reg)
			} else if d480.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d480.Reg)
				ctx.UnprotectReg(d480.Reg2)
			}
			ctx.EmitJmp(lbl44)
			ctx.MarkLabel(lbl42)
			d482 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d482)
			ctx.BindReg(r169, &d482)
			if r149 { ctx.UnprotectReg(r150) }
			ctx.EnsureDesc(&d482)
			ctx.EnsureDesc(&d482)
			var d483 scm.JITValueDesc
			if d482.Loc == scm.LocImm {
				d483 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d482.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.EmitMovRegReg(r187, d482.Reg)
				d483 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d483)
			}
			ctx.FreeDesc(&d482)
			var d484 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d484 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r188 := ctx.AllocReg()
				ctx.EmitMovRegMem(r188, thisptr.Reg, off)
				d484 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d484)
			}
			ctx.EnsureDesc(&d483)
			ctx.EnsureDesc(&d484)
			ctx.EnsureDesc(&d483)
			ctx.EnsureDesc(&d484)
			ctx.EnsureDesc(&d483)
			ctx.EnsureDesc(&d484)
			var d485 scm.JITValueDesc
			if d483.Loc == scm.LocImm && d484.Loc == scm.LocImm {
				d485 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d483.Imm.Int() + d484.Imm.Int())}
			} else if d484.Loc == scm.LocImm && d484.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d483.Reg)
				ctx.EmitMovRegReg(r189, d483.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d485)
			} else if d483.Loc == scm.LocImm && d483.Imm.Int() == 0 {
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d484.Reg}
				ctx.BindReg(d484.Reg, &d485)
			} else if d483.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d484.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d483.Imm.Int()))
				ctx.EmitAddInt64(scratch, d484.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d485)
			} else if d484.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d483.Reg)
				ctx.EmitMovRegReg(scratch, d483.Reg)
				if d484.Imm.Int() >= -2147483648 && d484.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d484.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d484.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d485)
			} else {
				r190 := ctx.AllocRegExcept(d483.Reg, d484.Reg)
				ctx.EmitMovRegReg(r190, d483.Reg)
				ctx.EmitAddInt64(r190, d484.Reg)
				d485 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d485)
			}
			if d485.Loc == scm.LocReg && d483.Loc == scm.LocReg && d485.Reg == d483.Reg {
				ctx.TransferReg(d483.Reg)
				d483.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d483)
			ctx.FreeDesc(&d484)
			ctx.EnsureDesc(&d145)
			d486 = d145
			_ = d486
			r191 := d145.Loc == scm.LocReg
			r192 := d145.Reg
			if r191 { ctx.ProtectReg(r192) }
			d487 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			lbl50 := ctx.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d487 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d486)
			ctx.EnsureDesc(&d486)
			var d488 scm.JITValueDesc
			if d486.Loc == scm.LocImm {
				d488 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d486.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.EmitMovRegReg(r193, d486.Reg)
				ctx.EmitShlRegImm8(r193, 32)
				ctx.EmitShrRegImm8(r193, 32)
				d488 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d488)
			}
			var d489 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d489 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r194 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r194, thisptr.Reg, off)
				d489 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d489)
			}
			ctx.EnsureDesc(&d489)
			ctx.EnsureDesc(&d489)
			var d490 scm.JITValueDesc
			if d489.Loc == scm.LocImm {
				d490 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d489.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.EmitMovRegReg(r195, d489.Reg)
				ctx.EmitShlRegImm8(r195, 56)
				ctx.EmitShrRegImm8(r195, 56)
				d490 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d490)
			}
			ctx.FreeDesc(&d489)
			ctx.EnsureDesc(&d488)
			ctx.EnsureDesc(&d490)
			ctx.EnsureDesc(&d488)
			ctx.EnsureDesc(&d490)
			ctx.EnsureDesc(&d488)
			ctx.EnsureDesc(&d490)
			var d491 scm.JITValueDesc
			if d488.Loc == scm.LocImm && d490.Loc == scm.LocImm {
				d491 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d488.Imm.Int() * d490.Imm.Int())}
			} else if d488.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d490.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d488.Imm.Int()))
				ctx.EmitImulInt64(scratch, d490.Reg)
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d491)
			} else if d490.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d488.Reg)
				ctx.EmitMovRegReg(scratch, d488.Reg)
				if d490.Imm.Int() >= -2147483648 && d490.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d490.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d490.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d491)
			} else {
				r196 := ctx.AllocRegExcept(d488.Reg, d490.Reg)
				ctx.EmitMovRegReg(r196, d488.Reg)
				ctx.EmitImulInt64(r196, d490.Reg)
				d491 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d491)
			}
			if d491.Loc == scm.LocReg && d488.Loc == scm.LocReg && d491.Reg == d488.Reg {
				ctx.TransferReg(d488.Reg)
				d488.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d488)
			ctx.FreeDesc(&d490)
			var d492 scm.JITValueDesc
			r197 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r197, uint64(dataPtr))
				d492 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197, StackOff: int32(sliceLen)}
				ctx.BindReg(r197, &d492)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.EmitMovRegMem(r197, thisptr.Reg, off)
				d492 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d492)
			}
			ctx.BindReg(r197, &d492)
			ctx.EnsureDesc(&d491)
			var d493 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d493 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d491.Imm.Int() / 64)}
			} else {
				r198 := ctx.AllocRegExcept(d491.Reg)
				ctx.EmitMovRegReg(r198, d491.Reg)
				ctx.EmitShrRegImm8(r198, 6)
				d493 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d493)
			}
			if d493.Loc == scm.LocReg && d491.Loc == scm.LocReg && d493.Reg == d491.Reg {
				ctx.TransferReg(d491.Reg)
				d491.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d493)
			r199 := ctx.AllocReg()
			ctx.EnsureDesc(&d493)
			ctx.EnsureDesc(&d492)
			if d493.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r199, uint64(d493.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r199, d493.Reg)
				ctx.EmitShlRegImm8(r199, 3)
			}
			if d492.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d492.Imm.Int()))
				ctx.EmitAddInt64(r199, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r199, d492.Reg)
			}
			r200 := ctx.AllocRegExcept(r199)
			ctx.EmitMovRegMem(r200, r199, 0)
			ctx.FreeReg(r199)
			d494 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r200}
			ctx.BindReg(r200, &d494)
			ctx.FreeDesc(&d493)
			ctx.EnsureDesc(&d491)
			var d495 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d495 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d491.Imm.Int() % 64)}
			} else {
				r201 := ctx.AllocRegExcept(d491.Reg)
				ctx.EmitMovRegReg(r201, d491.Reg)
				ctx.EmitAndRegImm32(r201, 63)
				d495 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d495)
			}
			if d495.Loc == scm.LocReg && d491.Loc == scm.LocReg && d495.Reg == d491.Reg {
				ctx.TransferReg(d491.Reg)
				d491.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d494)
			ctx.EnsureDesc(&d495)
			var d496 scm.JITValueDesc
			if d494.Loc == scm.LocImm && d495.Loc == scm.LocImm {
				d496 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d494.Imm.Int()) << uint64(d495.Imm.Int())))}
			} else if d495.Loc == scm.LocImm {
				r202 := ctx.AllocRegExcept(d494.Reg)
				ctx.EmitMovRegReg(r202, d494.Reg)
				ctx.EmitShlRegImm8(r202, uint8(d495.Imm.Int()))
				d496 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d496)
			} else {
				{
					shiftSrc := d494.Reg
					r203 := ctx.AllocRegExcept(d494.Reg)
					ctx.EmitMovRegReg(r203, d494.Reg)
					shiftSrc = r203
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d495.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d495.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d495.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d496 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d496)
				}
			}
			if d496.Loc == scm.LocReg && d494.Loc == scm.LocReg && d496.Reg == d494.Reg {
				ctx.TransferReg(d494.Reg)
				d494.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d494)
			ctx.FreeDesc(&d495)
			var d497 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d497 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r204 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r204, thisptr.Reg, off)
				d497 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d497)
			}
			d498 = d497
			ctx.EnsureDesc(&d498)
			if d498.Loc != scm.LocImm && d498.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl51 := ctx.ReserveLabel()
			lbl52 := ctx.ReserveLabel()
			lbl53 := ctx.ReserveLabel()
			lbl54 := ctx.ReserveLabel()
			if d498.Loc == scm.LocImm {
				if d498.Imm.Bool() {
					ctx.MarkLabel(lbl53)
					ctx.EmitJmp(lbl51)
				} else {
					ctx.MarkLabel(lbl54)
			ctx.EnsureDesc(&d496)
			if d496.Loc == scm.LocReg {
				ctx.ProtectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.ProtectReg(d496.Reg)
				ctx.ProtectReg(d496.Reg2)
			}
			d499 = d496
			if d499.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d499)
			ctx.EmitStoreToStack(d499, int32(bbs[2].PhiBase)+int32(0))
			if d496.Loc == scm.LocReg {
				ctx.UnprotectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d496.Reg)
				ctx.UnprotectReg(d496.Reg2)
			}
					ctx.EmitJmp(lbl52)
				}
			} else {
				ctx.EmitCmpRegImm32(d498.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl53)
				ctx.EmitJmp(lbl54)
				ctx.MarkLabel(lbl53)
				ctx.EmitJmp(lbl51)
				ctx.MarkLabel(lbl54)
			ctx.EnsureDesc(&d496)
			if d496.Loc == scm.LocReg {
				ctx.ProtectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.ProtectReg(d496.Reg)
				ctx.ProtectReg(d496.Reg2)
			}
			d500 = d496
			if d500.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d500)
			ctx.EmitStoreToStack(d500, int32(bbs[2].PhiBase)+int32(0))
			if d496.Loc == scm.LocReg {
				ctx.UnprotectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d496.Reg)
				ctx.UnprotectReg(d496.Reg2)
			}
				ctx.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d497)
			bbpos_5_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl52)
			ctx.ResolveFixups()
			d487 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			var d501 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d501 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r205 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r205, thisptr.Reg, off)
				d501 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
				ctx.BindReg(r205, &d501)
			}
			ctx.EnsureDesc(&d501)
			ctx.EnsureDesc(&d501)
			var d502 scm.JITValueDesc
			if d501.Loc == scm.LocImm {
				d502 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d501.Imm.Int()))))}
			} else {
				r206 := ctx.AllocReg()
				ctx.EmitMovRegReg(r206, d501.Reg)
				ctx.EmitShlRegImm8(r206, 56)
				ctx.EmitShrRegImm8(r206, 56)
				d502 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d502)
			}
			ctx.FreeDesc(&d501)
			d503 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d502)
			ctx.EnsureDesc(&d503)
			ctx.EnsureDesc(&d502)
			ctx.EnsureDesc(&d503)
			ctx.EnsureDesc(&d502)
			var d504 scm.JITValueDesc
			if d503.Loc == scm.LocImm && d502.Loc == scm.LocImm {
				d504 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d503.Imm.Int() - d502.Imm.Int())}
			} else if d502.Loc == scm.LocImm && d502.Imm.Int() == 0 {
				r207 := ctx.AllocRegExcept(d503.Reg)
				ctx.EmitMovRegReg(r207, d503.Reg)
				d504 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d504)
			} else if d503.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d502.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d503.Imm.Int()))
				ctx.EmitSubInt64(scratch, d502.Reg)
				d504 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d504)
			} else if d502.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d503.Reg)
				ctx.EmitMovRegReg(scratch, d503.Reg)
				if d502.Imm.Int() >= -2147483648 && d502.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d502.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d502.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d504 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d504)
			} else {
				r208 := ctx.AllocRegExcept(d503.Reg, d502.Reg)
				ctx.EmitMovRegReg(r208, d503.Reg)
				ctx.EmitSubInt64(r208, d502.Reg)
				d504 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d504)
			}
			if d504.Loc == scm.LocReg && d503.Loc == scm.LocReg && d504.Reg == d503.Reg {
				ctx.TransferReg(d503.Reg)
				d503.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d502)
			ctx.EnsureDesc(&d487)
			ctx.EnsureDesc(&d504)
			var d505 scm.JITValueDesc
			if d487.Loc == scm.LocImm && d504.Loc == scm.LocImm {
				d505 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d487.Imm.Int()) >> uint64(d504.Imm.Int())))}
			} else if d504.Loc == scm.LocImm {
				r209 := ctx.AllocRegExcept(d487.Reg)
				ctx.EmitMovRegReg(r209, d487.Reg)
				ctx.EmitShrRegImm8(r209, uint8(d504.Imm.Int()))
				d505 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d505)
			} else {
				{
					shiftSrc := d487.Reg
					r210 := ctx.AllocRegExcept(d487.Reg)
					ctx.EmitMovRegReg(r210, d487.Reg)
					shiftSrc = r210
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d504.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d504.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d504.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d505 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d505)
				}
			}
			if d505.Loc == scm.LocReg && d487.Loc == scm.LocReg && d505.Reg == d487.Reg {
				ctx.TransferReg(d487.Reg)
				d487.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d487)
			ctx.FreeDesc(&d504)
			r211 := ctx.AllocReg()
			ctx.EnsureDesc(&d505)
			ctx.EnsureDesc(&d505)
			if d505.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r211, d505)
			}
			ctx.EmitJmp(lbl50)
			bbpos_5_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl51)
			ctx.ResolveFixups()
			d487 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d491)
			var d506 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d506 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d491.Imm.Int() % 64)}
			} else {
				r212 := ctx.AllocRegExcept(d491.Reg)
				ctx.EmitMovRegReg(r212, d491.Reg)
				ctx.EmitAndRegImm32(r212, 63)
				d506 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d506)
			}
			if d506.Loc == scm.LocReg && d491.Loc == scm.LocReg && d506.Reg == d491.Reg {
				ctx.TransferReg(d491.Reg)
				d491.Loc = scm.LocNone
			}
			var d507 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d507 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r213 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r213, thisptr.Reg, off)
				d507 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
				ctx.BindReg(r213, &d507)
			}
			ctx.EnsureDesc(&d507)
			ctx.EnsureDesc(&d507)
			var d508 scm.JITValueDesc
			if d507.Loc == scm.LocImm {
				d508 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d507.Imm.Int()))))}
			} else {
				r214 := ctx.AllocReg()
				ctx.EmitMovRegReg(r214, d507.Reg)
				ctx.EmitShlRegImm8(r214, 56)
				ctx.EmitShrRegImm8(r214, 56)
				d508 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d508)
			}
			ctx.FreeDesc(&d507)
			ctx.EnsureDesc(&d506)
			ctx.EnsureDesc(&d508)
			ctx.EnsureDesc(&d506)
			ctx.EnsureDesc(&d508)
			ctx.EnsureDesc(&d506)
			ctx.EnsureDesc(&d508)
			var d509 scm.JITValueDesc
			if d506.Loc == scm.LocImm && d508.Loc == scm.LocImm {
				d509 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d506.Imm.Int() + d508.Imm.Int())}
			} else if d508.Loc == scm.LocImm && d508.Imm.Int() == 0 {
				r215 := ctx.AllocRegExcept(d506.Reg)
				ctx.EmitMovRegReg(r215, d506.Reg)
				d509 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d509)
			} else if d506.Loc == scm.LocImm && d506.Imm.Int() == 0 {
				d509 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d508.Reg}
				ctx.BindReg(d508.Reg, &d509)
			} else if d506.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d508.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d506.Imm.Int()))
				ctx.EmitAddInt64(scratch, d508.Reg)
				d509 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d509)
			} else if d508.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d506.Reg)
				ctx.EmitMovRegReg(scratch, d506.Reg)
				if d508.Imm.Int() >= -2147483648 && d508.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d508.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d508.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d509 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d509)
			} else {
				r216 := ctx.AllocRegExcept(d506.Reg, d508.Reg)
				ctx.EmitMovRegReg(r216, d506.Reg)
				ctx.EmitAddInt64(r216, d508.Reg)
				d509 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d509)
			}
			if d509.Loc == scm.LocReg && d506.Loc == scm.LocReg && d509.Reg == d506.Reg {
				ctx.TransferReg(d506.Reg)
				d506.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d506)
			ctx.FreeDesc(&d508)
			ctx.EnsureDesc(&d509)
			var d510 scm.JITValueDesc
			if d509.Loc == scm.LocImm {
				d510 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d509.Imm.Int()) > uint64(64))}
			} else {
				r217 := ctx.AllocRegExcept(d509.Reg)
				ctx.EmitCmpRegImm32(d509.Reg, 64)
				ctx.EmitSetcc(r217, scm.CcA)
				d510 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r217}
				ctx.BindReg(r217, &d510)
			}
			ctx.FreeDesc(&d509)
			d511 = d510
			ctx.EnsureDesc(&d511)
			if d511.Loc != scm.LocImm && d511.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.ReserveLabel()
			lbl56 := ctx.ReserveLabel()
			lbl57 := ctx.ReserveLabel()
			if d511.Loc == scm.LocImm {
				if d511.Imm.Bool() {
					ctx.MarkLabel(lbl56)
					ctx.EmitJmp(lbl55)
				} else {
					ctx.MarkLabel(lbl57)
			ctx.EnsureDesc(&d496)
			if d496.Loc == scm.LocReg {
				ctx.ProtectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.ProtectReg(d496.Reg)
				ctx.ProtectReg(d496.Reg2)
			}
			d512 = d496
			if d512.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d512)
			ctx.EmitStoreToStack(d512, int32(bbs[2].PhiBase)+int32(0))
			if d496.Loc == scm.LocReg {
				ctx.UnprotectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d496.Reg)
				ctx.UnprotectReg(d496.Reg2)
			}
					ctx.EmitJmp(lbl52)
				}
			} else {
				ctx.EmitCmpRegImm32(d511.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl56)
				ctx.EmitJmp(lbl57)
				ctx.MarkLabel(lbl56)
				ctx.EmitJmp(lbl55)
				ctx.MarkLabel(lbl57)
			ctx.EnsureDesc(&d496)
			if d496.Loc == scm.LocReg {
				ctx.ProtectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.ProtectReg(d496.Reg)
				ctx.ProtectReg(d496.Reg2)
			}
			d513 = d496
			if d513.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d513)
			ctx.EmitStoreToStack(d513, int32(bbs[2].PhiBase)+int32(0))
			if d496.Loc == scm.LocReg {
				ctx.UnprotectReg(d496.Reg)
			} else if d496.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d496.Reg)
				ctx.UnprotectReg(d496.Reg2)
			}
				ctx.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d510)
			bbpos_5_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl55)
			ctx.ResolveFixups()
			d487 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(64)}
			ctx.EnsureDesc(&d491)
			var d514 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d514 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d491.Imm.Int() / 64)}
			} else {
				r218 := ctx.AllocRegExcept(d491.Reg)
				ctx.EmitMovRegReg(r218, d491.Reg)
				ctx.EmitShrRegImm8(r218, 6)
				d514 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d514)
			}
			if d514.Loc == scm.LocReg && d491.Loc == scm.LocReg && d514.Reg == d491.Reg {
				ctx.TransferReg(d491.Reg)
				d491.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d514)
			ctx.EnsureDesc(&d514)
			var d515 scm.JITValueDesc
			if d514.Loc == scm.LocImm {
				d515 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d514.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d514.Reg)
				ctx.EmitMovRegReg(scratch, d514.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d515 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d515)
			}
			if d515.Loc == scm.LocReg && d514.Loc == scm.LocReg && d515.Reg == d514.Reg {
				ctx.TransferReg(d514.Reg)
				d514.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d514)
			ctx.EnsureDesc(&d515)
			r219 := ctx.AllocReg()
			ctx.EnsureDesc(&d515)
			ctx.EnsureDesc(&d492)
			if d515.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r219, uint64(d515.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r219, d515.Reg)
				ctx.EmitShlRegImm8(r219, 3)
			}
			if d492.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d492.Imm.Int()))
				ctx.EmitAddInt64(r219, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r219, d492.Reg)
			}
			r220 := ctx.AllocRegExcept(r219)
			ctx.EmitMovRegMem(r220, r219, 0)
			ctx.FreeReg(r219)
			d516 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r220}
			ctx.BindReg(r220, &d516)
			ctx.FreeDesc(&d515)
			ctx.EnsureDesc(&d491)
			var d517 scm.JITValueDesc
			if d491.Loc == scm.LocImm {
				d517 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d491.Imm.Int() % 64)}
			} else {
				r221 := ctx.AllocRegExcept(d491.Reg)
				ctx.EmitMovRegReg(r221, d491.Reg)
				ctx.EmitAndRegImm32(r221, 63)
				d517 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d517)
			}
			if d517.Loc == scm.LocReg && d491.Loc == scm.LocReg && d517.Reg == d491.Reg {
				ctx.TransferReg(d491.Reg)
				d491.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d491)
			d518 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d517)
			ctx.EnsureDesc(&d518)
			ctx.EnsureDesc(&d517)
			ctx.EnsureDesc(&d518)
			ctx.EnsureDesc(&d517)
			var d519 scm.JITValueDesc
			if d518.Loc == scm.LocImm && d517.Loc == scm.LocImm {
				d519 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d518.Imm.Int() - d517.Imm.Int())}
			} else if d517.Loc == scm.LocImm && d517.Imm.Int() == 0 {
				r222 := ctx.AllocRegExcept(d518.Reg)
				ctx.EmitMovRegReg(r222, d518.Reg)
				d519 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d519)
			} else if d518.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d517.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d518.Imm.Int()))
				ctx.EmitSubInt64(scratch, d517.Reg)
				d519 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d519)
			} else if d517.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d518.Reg)
				ctx.EmitMovRegReg(scratch, d518.Reg)
				if d517.Imm.Int() >= -2147483648 && d517.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d517.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d517.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d519 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d519)
			} else {
				r223 := ctx.AllocRegExcept(d518.Reg, d517.Reg)
				ctx.EmitMovRegReg(r223, d518.Reg)
				ctx.EmitSubInt64(r223, d517.Reg)
				d519 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d519)
			}
			if d519.Loc == scm.LocReg && d518.Loc == scm.LocReg && d519.Reg == d518.Reg {
				ctx.TransferReg(d518.Reg)
				d518.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d517)
			ctx.EnsureDesc(&d516)
			ctx.EnsureDesc(&d519)
			var d520 scm.JITValueDesc
			if d516.Loc == scm.LocImm && d519.Loc == scm.LocImm {
				d520 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d516.Imm.Int()) >> uint64(d519.Imm.Int())))}
			} else if d519.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d516.Reg)
				ctx.EmitMovRegReg(r224, d516.Reg)
				ctx.EmitShrRegImm8(r224, uint8(d519.Imm.Int()))
				d520 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d520)
			} else {
				{
					shiftSrc := d516.Reg
					r225 := ctx.AllocRegExcept(d516.Reg)
					ctx.EmitMovRegReg(r225, d516.Reg)
					shiftSrc = r225
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d519.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d519.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d519.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d520 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d520)
				}
			}
			if d520.Loc == scm.LocReg && d516.Loc == scm.LocReg && d520.Reg == d516.Reg {
				ctx.TransferReg(d516.Reg)
				d516.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d516)
			ctx.FreeDesc(&d519)
			ctx.EnsureDesc(&d496)
			ctx.EnsureDesc(&d520)
			var d521 scm.JITValueDesc
			if d496.Loc == scm.LocImm && d520.Loc == scm.LocImm {
				d521 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d496.Imm.Int() | d520.Imm.Int())}
			} else if d496.Loc == scm.LocImm && d496.Imm.Int() == 0 {
				d521 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d520.Reg}
				ctx.BindReg(d520.Reg, &d521)
			} else if d520.Loc == scm.LocImm && d520.Imm.Int() == 0 {
				r226 := ctx.AllocRegExcept(d496.Reg)
				ctx.EmitMovRegReg(r226, d496.Reg)
				d521 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d521)
			} else if d496.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d520.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d496.Imm.Int()))
				ctx.EmitOrInt64(scratch, d520.Reg)
				d521 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d521)
			} else if d520.Loc == scm.LocImm {
				r227 := ctx.AllocRegExcept(d496.Reg)
				ctx.EmitMovRegReg(r227, d496.Reg)
				if d520.Imm.Int() >= -2147483648 && d520.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r227, int32(d520.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d520.Imm.Int()))
					ctx.EmitOrInt64(r227, scm.RegR11)
				}
				d521 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d521)
			} else {
				r228 := ctx.AllocRegExcept(d496.Reg, d520.Reg)
				ctx.EmitMovRegReg(r228, d496.Reg)
				ctx.EmitOrInt64(r228, d520.Reg)
				d521 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d521)
			}
			if d521.Loc == scm.LocReg && d496.Loc == scm.LocReg && d521.Reg == d496.Reg {
				ctx.TransferReg(d496.Reg)
				d496.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d520)
			ctx.EnsureDesc(&d521)
			if d521.Loc == scm.LocReg {
				ctx.ProtectReg(d521.Reg)
			} else if d521.Loc == scm.LocRegPair {
				ctx.ProtectReg(d521.Reg)
				ctx.ProtectReg(d521.Reg2)
			}
			d522 = d521
			if d522.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d522)
			ctx.EmitStoreToStack(d522, int32(bbs[2].PhiBase)+int32(0))
			if d521.Loc == scm.LocReg {
				ctx.UnprotectReg(d521.Reg)
			} else if d521.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d521.Reg)
				ctx.UnprotectReg(d521.Reg2)
			}
			ctx.EmitJmp(lbl52)
			ctx.MarkLabel(lbl50)
			d523 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
			ctx.BindReg(r211, &d523)
			ctx.BindReg(r211, &d523)
			if r191 { ctx.UnprotectReg(r192) }
			ctx.EnsureDesc(&d523)
			ctx.EnsureDesc(&d523)
			var d524 scm.JITValueDesc
			if d523.Loc == scm.LocImm {
				d524 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d523.Imm.Int()))))}
			} else {
				r229 := ctx.AllocReg()
				ctx.EmitMovRegReg(r229, d523.Reg)
				d524 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d524)
			}
			ctx.FreeDesc(&d523)
			var d525 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d525 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r230 := ctx.AllocReg()
				ctx.EmitMovRegMem(r230, thisptr.Reg, off)
				d525 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r230}
				ctx.BindReg(r230, &d525)
			}
			ctx.EnsureDesc(&d524)
			ctx.EnsureDesc(&d525)
			ctx.EnsureDesc(&d524)
			ctx.EnsureDesc(&d525)
			ctx.EnsureDesc(&d524)
			ctx.EnsureDesc(&d525)
			var d526 scm.JITValueDesc
			if d524.Loc == scm.LocImm && d525.Loc == scm.LocImm {
				d526 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d524.Imm.Int() + d525.Imm.Int())}
			} else if d525.Loc == scm.LocImm && d525.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d524.Reg)
				ctx.EmitMovRegReg(r231, d524.Reg)
				d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d526)
			} else if d524.Loc == scm.LocImm && d524.Imm.Int() == 0 {
				d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d525.Reg}
				ctx.BindReg(d525.Reg, &d526)
			} else if d524.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d525.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d524.Imm.Int()))
				ctx.EmitAddInt64(scratch, d525.Reg)
				d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d526)
			} else if d525.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d524.Reg)
				ctx.EmitMovRegReg(scratch, d524.Reg)
				if d525.Imm.Int() >= -2147483648 && d525.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d525.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d525.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d526)
			} else {
				r232 := ctx.AllocRegExcept(d524.Reg, d525.Reg)
				ctx.EmitMovRegReg(r232, d524.Reg)
				ctx.EmitAddInt64(r232, d525.Reg)
				d526 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d526)
			}
			if d526.Loc == scm.LocReg && d524.Loc == scm.LocReg && d526.Reg == d524.Reg {
				ctx.TransferReg(d524.Reg)
				d524.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d524)
			ctx.FreeDesc(&d525)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d526)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d526)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d526)
			var d528 scm.JITValueDesc
			if d485.Loc == scm.LocImm && d526.Loc == scm.LocImm {
				d528 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d485.Imm.Int() + d526.Imm.Int())}
			} else if d526.Loc == scm.LocImm && d526.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d485.Reg)
				ctx.EmitMovRegReg(r233, d485.Reg)
				d528 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d528)
			} else if d485.Loc == scm.LocImm && d485.Imm.Int() == 0 {
				d528 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d526.Reg}
				ctx.BindReg(d526.Reg, &d528)
			} else if d485.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d526.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d485.Imm.Int()))
				ctx.EmitAddInt64(scratch, d526.Reg)
				d528 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d528)
			} else if d526.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d485.Reg)
				ctx.EmitMovRegReg(scratch, d485.Reg)
				if d526.Imm.Int() >= -2147483648 && d526.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d526.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d526.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d528 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d528)
			} else {
				r234 := ctx.AllocRegExcept(d485.Reg, d526.Reg)
				ctx.EmitMovRegReg(r234, d485.Reg)
				ctx.EmitAddInt64(r234, d526.Reg)
				d528 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d528)
			}
			if d528.Loc == scm.LocReg && d485.Loc == scm.LocReg && d528.Reg == d485.Reg {
				ctx.TransferReg(d485.Reg)
				d485.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d526)
			ctx.EnsureDesc(&d528)
			ctx.EnsureDesc(&d528)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d528)
			ctx.EnsureDesc(&d289)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d528)
			r235 := ctx.AllocReg()
			r236 := ctx.AllocRegExcept(r235)
			ctx.EnsureDesc(&d289)
			ctx.EnsureDesc(&d485)
			ctx.EnsureDesc(&d528)
			if d289.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r235, uint64(d289.Imm.Int()))
			} else if d289.Loc == scm.LocRegPair {
				ctx.EmitMovRegReg(r235, d289.Reg)
			} else {
				ctx.EmitMovRegReg(r235, d289.Reg)
			}
			if d485.Loc == scm.LocImm {
				if d485.Imm.Int() != 0 {
					if d485.Imm.Int() >= -2147483648 && d485.Imm.Int() <= 2147483647 {
						ctx.EmitAddRegImm32(r235, int32(d485.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(scm.RegR11, uint64(d485.Imm.Int()))
						ctx.EmitAddInt64(r235, scm.RegR11)
					}
				}
			} else {
				ctx.EmitAddInt64(r235, d485.Reg)
			}
			if d528.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r236, uint64(d528.Imm.Int()))
			} else {
				ctx.EmitMovRegReg(r236, d528.Reg)
			}
			if d485.Loc == scm.LocImm {
				if d485.Imm.Int() >= -2147483648 && d485.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(r236, int32(d485.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d485.Imm.Int()))
					ctx.EmitSubInt64(r236, scm.RegR11)
				}
			} else {
				ctx.EmitSubInt64(r236, d485.Reg)
			}
			d530 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r235, Reg2: r236}
			ctx.BindReg(r235, &d530)
			ctx.BindReg(r236, &d530)
			ctx.FreeDesc(&d485)
			ctx.FreeDesc(&d528)
			d531 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d531)
			ctx.BindReg(r1, &d531)
			d532 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d530}, 2)
			ctx.EmitMovPairToResult(&d532, &d531)
			ctx.EmitJmp(lbl0)
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
					ctx.EmitJmp(lbl9)
					return result
				}
				bbs[8].Rendered = true
				bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_8 = bbs[8].Address
				ctx.MarkLabel(lbl9)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
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
			if len(ps.OverlayValues) > 495 && ps.OverlayValues[495].Loc != scm.LocNone {
				d495 = ps.OverlayValues[495]
			}
			if len(ps.OverlayValues) > 496 && ps.OverlayValues[496].Loc != scm.LocNone {
				d496 = ps.OverlayValues[496]
			}
			if len(ps.OverlayValues) > 497 && ps.OverlayValues[497].Loc != scm.LocNone {
				d497 = ps.OverlayValues[497]
			}
			if len(ps.OverlayValues) > 498 && ps.OverlayValues[498].Loc != scm.LocNone {
				d498 = ps.OverlayValues[498]
			}
			if len(ps.OverlayValues) > 499 && ps.OverlayValues[499].Loc != scm.LocNone {
				d499 = ps.OverlayValues[499]
			}
			if len(ps.OverlayValues) > 500 && ps.OverlayValues[500].Loc != scm.LocNone {
				d500 = ps.OverlayValues[500]
			}
			if len(ps.OverlayValues) > 501 && ps.OverlayValues[501].Loc != scm.LocNone {
				d501 = ps.OverlayValues[501]
			}
			if len(ps.OverlayValues) > 502 && ps.OverlayValues[502].Loc != scm.LocNone {
				d502 = ps.OverlayValues[502]
			}
			if len(ps.OverlayValues) > 503 && ps.OverlayValues[503].Loc != scm.LocNone {
				d503 = ps.OverlayValues[503]
			}
			if len(ps.OverlayValues) > 504 && ps.OverlayValues[504].Loc != scm.LocNone {
				d504 = ps.OverlayValues[504]
			}
			if len(ps.OverlayValues) > 505 && ps.OverlayValues[505].Loc != scm.LocNone {
				d505 = ps.OverlayValues[505]
			}
			if len(ps.OverlayValues) > 506 && ps.OverlayValues[506].Loc != scm.LocNone {
				d506 = ps.OverlayValues[506]
			}
			if len(ps.OverlayValues) > 507 && ps.OverlayValues[507].Loc != scm.LocNone {
				d507 = ps.OverlayValues[507]
			}
			if len(ps.OverlayValues) > 508 && ps.OverlayValues[508].Loc != scm.LocNone {
				d508 = ps.OverlayValues[508]
			}
			if len(ps.OverlayValues) > 509 && ps.OverlayValues[509].Loc != scm.LocNone {
				d509 = ps.OverlayValues[509]
			}
			if len(ps.OverlayValues) > 510 && ps.OverlayValues[510].Loc != scm.LocNone {
				d510 = ps.OverlayValues[510]
			}
			if len(ps.OverlayValues) > 511 && ps.OverlayValues[511].Loc != scm.LocNone {
				d511 = ps.OverlayValues[511]
			}
			if len(ps.OverlayValues) > 512 && ps.OverlayValues[512].Loc != scm.LocNone {
				d512 = ps.OverlayValues[512]
			}
			if len(ps.OverlayValues) > 513 && ps.OverlayValues[513].Loc != scm.LocNone {
				d513 = ps.OverlayValues[513]
			}
			if len(ps.OverlayValues) > 514 && ps.OverlayValues[514].Loc != scm.LocNone {
				d514 = ps.OverlayValues[514]
			}
			if len(ps.OverlayValues) > 515 && ps.OverlayValues[515].Loc != scm.LocNone {
				d515 = ps.OverlayValues[515]
			}
			if len(ps.OverlayValues) > 516 && ps.OverlayValues[516].Loc != scm.LocNone {
				d516 = ps.OverlayValues[516]
			}
			if len(ps.OverlayValues) > 517 && ps.OverlayValues[517].Loc != scm.LocNone {
				d517 = ps.OverlayValues[517]
			}
			if len(ps.OverlayValues) > 518 && ps.OverlayValues[518].Loc != scm.LocNone {
				d518 = ps.OverlayValues[518]
			}
			if len(ps.OverlayValues) > 519 && ps.OverlayValues[519].Loc != scm.LocNone {
				d519 = ps.OverlayValues[519]
			}
			if len(ps.OverlayValues) > 520 && ps.OverlayValues[520].Loc != scm.LocNone {
				d520 = ps.OverlayValues[520]
			}
			if len(ps.OverlayValues) > 521 && ps.OverlayValues[521].Loc != scm.LocNone {
				d521 = ps.OverlayValues[521]
			}
			if len(ps.OverlayValues) > 522 && ps.OverlayValues[522].Loc != scm.LocNone {
				d522 = ps.OverlayValues[522]
			}
			if len(ps.OverlayValues) > 523 && ps.OverlayValues[523].Loc != scm.LocNone {
				d523 = ps.OverlayValues[523]
			}
			if len(ps.OverlayValues) > 524 && ps.OverlayValues[524].Loc != scm.LocNone {
				d524 = ps.OverlayValues[524]
			}
			if len(ps.OverlayValues) > 525 && ps.OverlayValues[525].Loc != scm.LocNone {
				d525 = ps.OverlayValues[525]
			}
			if len(ps.OverlayValues) > 526 && ps.OverlayValues[526].Loc != scm.LocNone {
				d526 = ps.OverlayValues[526]
			}
			if len(ps.OverlayValues) > 527 && ps.OverlayValues[527].Loc != scm.LocNone {
				d527 = ps.OverlayValues[527]
			}
			if len(ps.OverlayValues) > 528 && ps.OverlayValues[528].Loc != scm.LocNone {
				d528 = ps.OverlayValues[528]
			}
			if len(ps.OverlayValues) > 529 && ps.OverlayValues[529].Loc != scm.LocNone {
				d529 = ps.OverlayValues[529]
			}
			if len(ps.OverlayValues) > 530 && ps.OverlayValues[530].Loc != scm.LocNone {
				d530 = ps.OverlayValues[530]
			}
			if len(ps.OverlayValues) > 531 && ps.OverlayValues[531].Loc != scm.LocNone {
				d531 = ps.OverlayValues[531]
			}
			if len(ps.OverlayValues) > 532 && ps.OverlayValues[532].Loc != scm.LocNone {
				d532 = ps.OverlayValues[532]
			}
			ctx.ReclaimUntrackedRegs()
			var d533 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d533 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r237 := ctx.AllocReg()
				ctx.EmitMovRegMem(r237, thisptr.Reg, off)
				d533 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d533)
			}
			ctx.EnsureDesc(&d533)
			ctx.EnsureDesc(&d533)
			var d534 scm.JITValueDesc
			if d533.Loc == scm.LocImm {
				d534 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d533.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.EmitMovRegReg(r238, d533.Reg)
				ctx.EmitShlRegImm8(r238, 32)
				ctx.EmitShrRegImm8(r238, 32)
				d534 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d534)
			}
			ctx.FreeDesc(&d533)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d534)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d534)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d534)
			var d535 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d534.Loc == scm.LocImm {
				d535 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d145.Imm.Int()) == uint64(d534.Imm.Int()))}
			} else if d534.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d145.Reg)
				if d534.Imm.Int() >= -2147483648 && d534.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d145.Reg, int32(d534.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d534.Imm.Int()))
					ctx.EmitCmpInt64(d145.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r239, scm.CcE)
				d535 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d535)
			} else if d145.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d534.Reg)
				ctx.EmitSetcc(r240, scm.CcE)
				d535 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d535)
			} else {
				r241 := ctx.AllocRegExcept(d145.Reg)
				ctx.EmitCmpInt64(d145.Reg, d534.Reg)
				ctx.EmitSetcc(r241, scm.CcE)
				d535 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d535)
			}
			ctx.FreeDesc(&d145)
			ctx.FreeDesc(&d534)
			d536 = d535
			ctx.EnsureDesc(&d536)
			if d536.Loc != scm.LocImm && d536.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d536.Loc == scm.LocImm {
				if d536.Imm.Bool() {
			ps537 := scm.PhiState{General: ps.General}
			ps537.OverlayValues = make([]scm.JITValueDesc, 537)
			ps537.OverlayValues[0] = d0
			ps537.OverlayValues[1] = d1
			ps537.OverlayValues[9] = d9
			ps537.OverlayValues[10] = d10
			ps537.OverlayValues[11] = d11
			ps537.OverlayValues[12] = d12
			ps537.OverlayValues[13] = d13
			ps537.OverlayValues[14] = d14
			ps537.OverlayValues[15] = d15
			ps537.OverlayValues[16] = d16
			ps537.OverlayValues[17] = d17
			ps537.OverlayValues[18] = d18
			ps537.OverlayValues[19] = d19
			ps537.OverlayValues[20] = d20
			ps537.OverlayValues[21] = d21
			ps537.OverlayValues[22] = d22
			ps537.OverlayValues[23] = d23
			ps537.OverlayValues[24] = d24
			ps537.OverlayValues[25] = d25
			ps537.OverlayValues[26] = d26
			ps537.OverlayValues[27] = d27
			ps537.OverlayValues[28] = d28
			ps537.OverlayValues[29] = d29
			ps537.OverlayValues[30] = d30
			ps537.OverlayValues[31] = d31
			ps537.OverlayValues[32] = d32
			ps537.OverlayValues[33] = d33
			ps537.OverlayValues[34] = d34
			ps537.OverlayValues[35] = d35
			ps537.OverlayValues[36] = d36
			ps537.OverlayValues[37] = d37
			ps537.OverlayValues[38] = d38
			ps537.OverlayValues[39] = d39
			ps537.OverlayValues[40] = d40
			ps537.OverlayValues[41] = d41
			ps537.OverlayValues[42] = d42
			ps537.OverlayValues[43] = d43
			ps537.OverlayValues[44] = d44
			ps537.OverlayValues[45] = d45
			ps537.OverlayValues[46] = d46
			ps537.OverlayValues[47] = d47
			ps537.OverlayValues[48] = d48
			ps537.OverlayValues[49] = d49
			ps537.OverlayValues[50] = d50
			ps537.OverlayValues[51] = d51
			ps537.OverlayValues[52] = d52
			ps537.OverlayValues[104] = d104
			ps537.OverlayValues[105] = d105
			ps537.OverlayValues[106] = d106
			ps537.OverlayValues[107] = d107
			ps537.OverlayValues[108] = d108
			ps537.OverlayValues[109] = d109
			ps537.OverlayValues[110] = d110
			ps537.OverlayValues[111] = d111
			ps537.OverlayValues[112] = d112
			ps537.OverlayValues[113] = d113
			ps537.OverlayValues[114] = d114
			ps537.OverlayValues[115] = d115
			ps537.OverlayValues[116] = d116
			ps537.OverlayValues[117] = d117
			ps537.OverlayValues[118] = d118
			ps537.OverlayValues[119] = d119
			ps537.OverlayValues[120] = d120
			ps537.OverlayValues[121] = d121
			ps537.OverlayValues[122] = d122
			ps537.OverlayValues[123] = d123
			ps537.OverlayValues[124] = d124
			ps537.OverlayValues[125] = d125
			ps537.OverlayValues[126] = d126
			ps537.OverlayValues[127] = d127
			ps537.OverlayValues[128] = d128
			ps537.OverlayValues[129] = d129
			ps537.OverlayValues[130] = d130
			ps537.OverlayValues[131] = d131
			ps537.OverlayValues[132] = d132
			ps537.OverlayValues[133] = d133
			ps537.OverlayValues[134] = d134
			ps537.OverlayValues[135] = d135
			ps537.OverlayValues[136] = d136
			ps537.OverlayValues[137] = d137
			ps537.OverlayValues[138] = d138
			ps537.OverlayValues[139] = d139
			ps537.OverlayValues[140] = d140
			ps537.OverlayValues[141] = d141
			ps537.OverlayValues[142] = d142
			ps537.OverlayValues[143] = d143
			ps537.OverlayValues[144] = d144
			ps537.OverlayValues[145] = d145
			ps537.OverlayValues[146] = d146
			ps537.OverlayValues[147] = d147
			ps537.OverlayValues[243] = d243
			ps537.OverlayValues[244] = d244
			ps537.OverlayValues[245] = d245
			ps537.OverlayValues[246] = d246
			ps537.OverlayValues[247] = d247
			ps537.OverlayValues[248] = d248
			ps537.OverlayValues[249] = d249
			ps537.OverlayValues[250] = d250
			ps537.OverlayValues[251] = d251
			ps537.OverlayValues[252] = d252
			ps537.OverlayValues[253] = d253
			ps537.OverlayValues[254] = d254
			ps537.OverlayValues[255] = d255
			ps537.OverlayValues[256] = d256
			ps537.OverlayValues[257] = d257
			ps537.OverlayValues[258] = d258
			ps537.OverlayValues[259] = d259
			ps537.OverlayValues[260] = d260
			ps537.OverlayValues[261] = d261
			ps537.OverlayValues[262] = d262
			ps537.OverlayValues[263] = d263
			ps537.OverlayValues[264] = d264
			ps537.OverlayValues[265] = d265
			ps537.OverlayValues[266] = d266
			ps537.OverlayValues[267] = d267
			ps537.OverlayValues[268] = d268
			ps537.OverlayValues[269] = d269
			ps537.OverlayValues[270] = d270
			ps537.OverlayValues[271] = d271
			ps537.OverlayValues[272] = d272
			ps537.OverlayValues[273] = d273
			ps537.OverlayValues[274] = d274
			ps537.OverlayValues[275] = d275
			ps537.OverlayValues[276] = d276
			ps537.OverlayValues[277] = d277
			ps537.OverlayValues[278] = d278
			ps537.OverlayValues[279] = d279
			ps537.OverlayValues[280] = d280
			ps537.OverlayValues[281] = d281
			ps537.OverlayValues[282] = d282
			ps537.OverlayValues[283] = d283
			ps537.OverlayValues[284] = d284
			ps537.OverlayValues[285] = d285
			ps537.OverlayValues[286] = d286
			ps537.OverlayValues[287] = d287
			ps537.OverlayValues[288] = d288
			ps537.OverlayValues[289] = d289
			ps537.OverlayValues[290] = d290
			ps537.OverlayValues[291] = d291
			ps537.OverlayValues[292] = d292
			ps537.OverlayValues[293] = d293
			ps537.OverlayValues[294] = d294
			ps537.OverlayValues[295] = d295
			ps537.OverlayValues[444] = d444
			ps537.OverlayValues[445] = d445
			ps537.OverlayValues[446] = d446
			ps537.OverlayValues[447] = d447
			ps537.OverlayValues[448] = d448
			ps537.OverlayValues[449] = d449
			ps537.OverlayValues[450] = d450
			ps537.OverlayValues[451] = d451
			ps537.OverlayValues[452] = d452
			ps537.OverlayValues[453] = d453
			ps537.OverlayValues[454] = d454
			ps537.OverlayValues[455] = d455
			ps537.OverlayValues[456] = d456
			ps537.OverlayValues[457] = d457
			ps537.OverlayValues[458] = d458
			ps537.OverlayValues[459] = d459
			ps537.OverlayValues[460] = d460
			ps537.OverlayValues[461] = d461
			ps537.OverlayValues[462] = d462
			ps537.OverlayValues[463] = d463
			ps537.OverlayValues[464] = d464
			ps537.OverlayValues[465] = d465
			ps537.OverlayValues[466] = d466
			ps537.OverlayValues[467] = d467
			ps537.OverlayValues[468] = d468
			ps537.OverlayValues[469] = d469
			ps537.OverlayValues[470] = d470
			ps537.OverlayValues[471] = d471
			ps537.OverlayValues[472] = d472
			ps537.OverlayValues[473] = d473
			ps537.OverlayValues[474] = d474
			ps537.OverlayValues[475] = d475
			ps537.OverlayValues[476] = d476
			ps537.OverlayValues[477] = d477
			ps537.OverlayValues[478] = d478
			ps537.OverlayValues[479] = d479
			ps537.OverlayValues[480] = d480
			ps537.OverlayValues[481] = d481
			ps537.OverlayValues[482] = d482
			ps537.OverlayValues[483] = d483
			ps537.OverlayValues[484] = d484
			ps537.OverlayValues[485] = d485
			ps537.OverlayValues[486] = d486
			ps537.OverlayValues[487] = d487
			ps537.OverlayValues[488] = d488
			ps537.OverlayValues[489] = d489
			ps537.OverlayValues[490] = d490
			ps537.OverlayValues[491] = d491
			ps537.OverlayValues[492] = d492
			ps537.OverlayValues[493] = d493
			ps537.OverlayValues[494] = d494
			ps537.OverlayValues[495] = d495
			ps537.OverlayValues[496] = d496
			ps537.OverlayValues[497] = d497
			ps537.OverlayValues[498] = d498
			ps537.OverlayValues[499] = d499
			ps537.OverlayValues[500] = d500
			ps537.OverlayValues[501] = d501
			ps537.OverlayValues[502] = d502
			ps537.OverlayValues[503] = d503
			ps537.OverlayValues[504] = d504
			ps537.OverlayValues[505] = d505
			ps537.OverlayValues[506] = d506
			ps537.OverlayValues[507] = d507
			ps537.OverlayValues[508] = d508
			ps537.OverlayValues[509] = d509
			ps537.OverlayValues[510] = d510
			ps537.OverlayValues[511] = d511
			ps537.OverlayValues[512] = d512
			ps537.OverlayValues[513] = d513
			ps537.OverlayValues[514] = d514
			ps537.OverlayValues[515] = d515
			ps537.OverlayValues[516] = d516
			ps537.OverlayValues[517] = d517
			ps537.OverlayValues[518] = d518
			ps537.OverlayValues[519] = d519
			ps537.OverlayValues[520] = d520
			ps537.OverlayValues[521] = d521
			ps537.OverlayValues[522] = d522
			ps537.OverlayValues[523] = d523
			ps537.OverlayValues[524] = d524
			ps537.OverlayValues[525] = d525
			ps537.OverlayValues[526] = d526
			ps537.OverlayValues[527] = d527
			ps537.OverlayValues[528] = d528
			ps537.OverlayValues[529] = d529
			ps537.OverlayValues[530] = d530
			ps537.OverlayValues[531] = d531
			ps537.OverlayValues[532] = d532
			ps537.OverlayValues[533] = d533
			ps537.OverlayValues[534] = d534
			ps537.OverlayValues[535] = d535
			ps537.OverlayValues[536] = d536
					return bbs[6].RenderPS(ps537)
				}
			ps538 := scm.PhiState{General: ps.General}
			ps538.OverlayValues = make([]scm.JITValueDesc, 537)
			ps538.OverlayValues[0] = d0
			ps538.OverlayValues[1] = d1
			ps538.OverlayValues[9] = d9
			ps538.OverlayValues[10] = d10
			ps538.OverlayValues[11] = d11
			ps538.OverlayValues[12] = d12
			ps538.OverlayValues[13] = d13
			ps538.OverlayValues[14] = d14
			ps538.OverlayValues[15] = d15
			ps538.OverlayValues[16] = d16
			ps538.OverlayValues[17] = d17
			ps538.OverlayValues[18] = d18
			ps538.OverlayValues[19] = d19
			ps538.OverlayValues[20] = d20
			ps538.OverlayValues[21] = d21
			ps538.OverlayValues[22] = d22
			ps538.OverlayValues[23] = d23
			ps538.OverlayValues[24] = d24
			ps538.OverlayValues[25] = d25
			ps538.OverlayValues[26] = d26
			ps538.OverlayValues[27] = d27
			ps538.OverlayValues[28] = d28
			ps538.OverlayValues[29] = d29
			ps538.OverlayValues[30] = d30
			ps538.OverlayValues[31] = d31
			ps538.OverlayValues[32] = d32
			ps538.OverlayValues[33] = d33
			ps538.OverlayValues[34] = d34
			ps538.OverlayValues[35] = d35
			ps538.OverlayValues[36] = d36
			ps538.OverlayValues[37] = d37
			ps538.OverlayValues[38] = d38
			ps538.OverlayValues[39] = d39
			ps538.OverlayValues[40] = d40
			ps538.OverlayValues[41] = d41
			ps538.OverlayValues[42] = d42
			ps538.OverlayValues[43] = d43
			ps538.OverlayValues[44] = d44
			ps538.OverlayValues[45] = d45
			ps538.OverlayValues[46] = d46
			ps538.OverlayValues[47] = d47
			ps538.OverlayValues[48] = d48
			ps538.OverlayValues[49] = d49
			ps538.OverlayValues[50] = d50
			ps538.OverlayValues[51] = d51
			ps538.OverlayValues[52] = d52
			ps538.OverlayValues[104] = d104
			ps538.OverlayValues[105] = d105
			ps538.OverlayValues[106] = d106
			ps538.OverlayValues[107] = d107
			ps538.OverlayValues[108] = d108
			ps538.OverlayValues[109] = d109
			ps538.OverlayValues[110] = d110
			ps538.OverlayValues[111] = d111
			ps538.OverlayValues[112] = d112
			ps538.OverlayValues[113] = d113
			ps538.OverlayValues[114] = d114
			ps538.OverlayValues[115] = d115
			ps538.OverlayValues[116] = d116
			ps538.OverlayValues[117] = d117
			ps538.OverlayValues[118] = d118
			ps538.OverlayValues[119] = d119
			ps538.OverlayValues[120] = d120
			ps538.OverlayValues[121] = d121
			ps538.OverlayValues[122] = d122
			ps538.OverlayValues[123] = d123
			ps538.OverlayValues[124] = d124
			ps538.OverlayValues[125] = d125
			ps538.OverlayValues[126] = d126
			ps538.OverlayValues[127] = d127
			ps538.OverlayValues[128] = d128
			ps538.OverlayValues[129] = d129
			ps538.OverlayValues[130] = d130
			ps538.OverlayValues[131] = d131
			ps538.OverlayValues[132] = d132
			ps538.OverlayValues[133] = d133
			ps538.OverlayValues[134] = d134
			ps538.OverlayValues[135] = d135
			ps538.OverlayValues[136] = d136
			ps538.OverlayValues[137] = d137
			ps538.OverlayValues[138] = d138
			ps538.OverlayValues[139] = d139
			ps538.OverlayValues[140] = d140
			ps538.OverlayValues[141] = d141
			ps538.OverlayValues[142] = d142
			ps538.OverlayValues[143] = d143
			ps538.OverlayValues[144] = d144
			ps538.OverlayValues[145] = d145
			ps538.OverlayValues[146] = d146
			ps538.OverlayValues[147] = d147
			ps538.OverlayValues[243] = d243
			ps538.OverlayValues[244] = d244
			ps538.OverlayValues[245] = d245
			ps538.OverlayValues[246] = d246
			ps538.OverlayValues[247] = d247
			ps538.OverlayValues[248] = d248
			ps538.OverlayValues[249] = d249
			ps538.OverlayValues[250] = d250
			ps538.OverlayValues[251] = d251
			ps538.OverlayValues[252] = d252
			ps538.OverlayValues[253] = d253
			ps538.OverlayValues[254] = d254
			ps538.OverlayValues[255] = d255
			ps538.OverlayValues[256] = d256
			ps538.OverlayValues[257] = d257
			ps538.OverlayValues[258] = d258
			ps538.OverlayValues[259] = d259
			ps538.OverlayValues[260] = d260
			ps538.OverlayValues[261] = d261
			ps538.OverlayValues[262] = d262
			ps538.OverlayValues[263] = d263
			ps538.OverlayValues[264] = d264
			ps538.OverlayValues[265] = d265
			ps538.OverlayValues[266] = d266
			ps538.OverlayValues[267] = d267
			ps538.OverlayValues[268] = d268
			ps538.OverlayValues[269] = d269
			ps538.OverlayValues[270] = d270
			ps538.OverlayValues[271] = d271
			ps538.OverlayValues[272] = d272
			ps538.OverlayValues[273] = d273
			ps538.OverlayValues[274] = d274
			ps538.OverlayValues[275] = d275
			ps538.OverlayValues[276] = d276
			ps538.OverlayValues[277] = d277
			ps538.OverlayValues[278] = d278
			ps538.OverlayValues[279] = d279
			ps538.OverlayValues[280] = d280
			ps538.OverlayValues[281] = d281
			ps538.OverlayValues[282] = d282
			ps538.OverlayValues[283] = d283
			ps538.OverlayValues[284] = d284
			ps538.OverlayValues[285] = d285
			ps538.OverlayValues[286] = d286
			ps538.OverlayValues[287] = d287
			ps538.OverlayValues[288] = d288
			ps538.OverlayValues[289] = d289
			ps538.OverlayValues[290] = d290
			ps538.OverlayValues[291] = d291
			ps538.OverlayValues[292] = d292
			ps538.OverlayValues[293] = d293
			ps538.OverlayValues[294] = d294
			ps538.OverlayValues[295] = d295
			ps538.OverlayValues[444] = d444
			ps538.OverlayValues[445] = d445
			ps538.OverlayValues[446] = d446
			ps538.OverlayValues[447] = d447
			ps538.OverlayValues[448] = d448
			ps538.OverlayValues[449] = d449
			ps538.OverlayValues[450] = d450
			ps538.OverlayValues[451] = d451
			ps538.OverlayValues[452] = d452
			ps538.OverlayValues[453] = d453
			ps538.OverlayValues[454] = d454
			ps538.OverlayValues[455] = d455
			ps538.OverlayValues[456] = d456
			ps538.OverlayValues[457] = d457
			ps538.OverlayValues[458] = d458
			ps538.OverlayValues[459] = d459
			ps538.OverlayValues[460] = d460
			ps538.OverlayValues[461] = d461
			ps538.OverlayValues[462] = d462
			ps538.OverlayValues[463] = d463
			ps538.OverlayValues[464] = d464
			ps538.OverlayValues[465] = d465
			ps538.OverlayValues[466] = d466
			ps538.OverlayValues[467] = d467
			ps538.OverlayValues[468] = d468
			ps538.OverlayValues[469] = d469
			ps538.OverlayValues[470] = d470
			ps538.OverlayValues[471] = d471
			ps538.OverlayValues[472] = d472
			ps538.OverlayValues[473] = d473
			ps538.OverlayValues[474] = d474
			ps538.OverlayValues[475] = d475
			ps538.OverlayValues[476] = d476
			ps538.OverlayValues[477] = d477
			ps538.OverlayValues[478] = d478
			ps538.OverlayValues[479] = d479
			ps538.OverlayValues[480] = d480
			ps538.OverlayValues[481] = d481
			ps538.OverlayValues[482] = d482
			ps538.OverlayValues[483] = d483
			ps538.OverlayValues[484] = d484
			ps538.OverlayValues[485] = d485
			ps538.OverlayValues[486] = d486
			ps538.OverlayValues[487] = d487
			ps538.OverlayValues[488] = d488
			ps538.OverlayValues[489] = d489
			ps538.OverlayValues[490] = d490
			ps538.OverlayValues[491] = d491
			ps538.OverlayValues[492] = d492
			ps538.OverlayValues[493] = d493
			ps538.OverlayValues[494] = d494
			ps538.OverlayValues[495] = d495
			ps538.OverlayValues[496] = d496
			ps538.OverlayValues[497] = d497
			ps538.OverlayValues[498] = d498
			ps538.OverlayValues[499] = d499
			ps538.OverlayValues[500] = d500
			ps538.OverlayValues[501] = d501
			ps538.OverlayValues[502] = d502
			ps538.OverlayValues[503] = d503
			ps538.OverlayValues[504] = d504
			ps538.OverlayValues[505] = d505
			ps538.OverlayValues[506] = d506
			ps538.OverlayValues[507] = d507
			ps538.OverlayValues[508] = d508
			ps538.OverlayValues[509] = d509
			ps538.OverlayValues[510] = d510
			ps538.OverlayValues[511] = d511
			ps538.OverlayValues[512] = d512
			ps538.OverlayValues[513] = d513
			ps538.OverlayValues[514] = d514
			ps538.OverlayValues[515] = d515
			ps538.OverlayValues[516] = d516
			ps538.OverlayValues[517] = d517
			ps538.OverlayValues[518] = d518
			ps538.OverlayValues[519] = d519
			ps538.OverlayValues[520] = d520
			ps538.OverlayValues[521] = d521
			ps538.OverlayValues[522] = d522
			ps538.OverlayValues[523] = d523
			ps538.OverlayValues[524] = d524
			ps538.OverlayValues[525] = d525
			ps538.OverlayValues[526] = d526
			ps538.OverlayValues[527] = d527
			ps538.OverlayValues[528] = d528
			ps538.OverlayValues[529] = d529
			ps538.OverlayValues[530] = d530
			ps538.OverlayValues[531] = d531
			ps538.OverlayValues[532] = d532
			ps538.OverlayValues[533] = d533
			ps538.OverlayValues[534] = d534
			ps538.OverlayValues[535] = d535
			ps538.OverlayValues[536] = d536
				return bbs[7].RenderPS(ps538)
			}
			if !ps.General {
				ps.General = true
				return bbs[8].RenderPS(ps)
			}
			lbl58 := ctx.ReserveLabel()
			lbl59 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d536.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl58)
			ctx.EmitJmp(lbl59)
			ctx.MarkLabel(lbl58)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl59)
			ctx.EmitJmp(lbl8)
			ps539 := scm.PhiState{General: true}
			ps539.OverlayValues = make([]scm.JITValueDesc, 537)
			ps539.OverlayValues[0] = d0
			ps539.OverlayValues[1] = d1
			ps539.OverlayValues[9] = d9
			ps539.OverlayValues[10] = d10
			ps539.OverlayValues[11] = d11
			ps539.OverlayValues[12] = d12
			ps539.OverlayValues[13] = d13
			ps539.OverlayValues[14] = d14
			ps539.OverlayValues[15] = d15
			ps539.OverlayValues[16] = d16
			ps539.OverlayValues[17] = d17
			ps539.OverlayValues[18] = d18
			ps539.OverlayValues[19] = d19
			ps539.OverlayValues[20] = d20
			ps539.OverlayValues[21] = d21
			ps539.OverlayValues[22] = d22
			ps539.OverlayValues[23] = d23
			ps539.OverlayValues[24] = d24
			ps539.OverlayValues[25] = d25
			ps539.OverlayValues[26] = d26
			ps539.OverlayValues[27] = d27
			ps539.OverlayValues[28] = d28
			ps539.OverlayValues[29] = d29
			ps539.OverlayValues[30] = d30
			ps539.OverlayValues[31] = d31
			ps539.OverlayValues[32] = d32
			ps539.OverlayValues[33] = d33
			ps539.OverlayValues[34] = d34
			ps539.OverlayValues[35] = d35
			ps539.OverlayValues[36] = d36
			ps539.OverlayValues[37] = d37
			ps539.OverlayValues[38] = d38
			ps539.OverlayValues[39] = d39
			ps539.OverlayValues[40] = d40
			ps539.OverlayValues[41] = d41
			ps539.OverlayValues[42] = d42
			ps539.OverlayValues[43] = d43
			ps539.OverlayValues[44] = d44
			ps539.OverlayValues[45] = d45
			ps539.OverlayValues[46] = d46
			ps539.OverlayValues[47] = d47
			ps539.OverlayValues[48] = d48
			ps539.OverlayValues[49] = d49
			ps539.OverlayValues[50] = d50
			ps539.OverlayValues[51] = d51
			ps539.OverlayValues[52] = d52
			ps539.OverlayValues[104] = d104
			ps539.OverlayValues[105] = d105
			ps539.OverlayValues[106] = d106
			ps539.OverlayValues[107] = d107
			ps539.OverlayValues[108] = d108
			ps539.OverlayValues[109] = d109
			ps539.OverlayValues[110] = d110
			ps539.OverlayValues[111] = d111
			ps539.OverlayValues[112] = d112
			ps539.OverlayValues[113] = d113
			ps539.OverlayValues[114] = d114
			ps539.OverlayValues[115] = d115
			ps539.OverlayValues[116] = d116
			ps539.OverlayValues[117] = d117
			ps539.OverlayValues[118] = d118
			ps539.OverlayValues[119] = d119
			ps539.OverlayValues[120] = d120
			ps539.OverlayValues[121] = d121
			ps539.OverlayValues[122] = d122
			ps539.OverlayValues[123] = d123
			ps539.OverlayValues[124] = d124
			ps539.OverlayValues[125] = d125
			ps539.OverlayValues[126] = d126
			ps539.OverlayValues[127] = d127
			ps539.OverlayValues[128] = d128
			ps539.OverlayValues[129] = d129
			ps539.OverlayValues[130] = d130
			ps539.OverlayValues[131] = d131
			ps539.OverlayValues[132] = d132
			ps539.OverlayValues[133] = d133
			ps539.OverlayValues[134] = d134
			ps539.OverlayValues[135] = d135
			ps539.OverlayValues[136] = d136
			ps539.OverlayValues[137] = d137
			ps539.OverlayValues[138] = d138
			ps539.OverlayValues[139] = d139
			ps539.OverlayValues[140] = d140
			ps539.OverlayValues[141] = d141
			ps539.OverlayValues[142] = d142
			ps539.OverlayValues[143] = d143
			ps539.OverlayValues[144] = d144
			ps539.OverlayValues[145] = d145
			ps539.OverlayValues[146] = d146
			ps539.OverlayValues[147] = d147
			ps539.OverlayValues[243] = d243
			ps539.OverlayValues[244] = d244
			ps539.OverlayValues[245] = d245
			ps539.OverlayValues[246] = d246
			ps539.OverlayValues[247] = d247
			ps539.OverlayValues[248] = d248
			ps539.OverlayValues[249] = d249
			ps539.OverlayValues[250] = d250
			ps539.OverlayValues[251] = d251
			ps539.OverlayValues[252] = d252
			ps539.OverlayValues[253] = d253
			ps539.OverlayValues[254] = d254
			ps539.OverlayValues[255] = d255
			ps539.OverlayValues[256] = d256
			ps539.OverlayValues[257] = d257
			ps539.OverlayValues[258] = d258
			ps539.OverlayValues[259] = d259
			ps539.OverlayValues[260] = d260
			ps539.OverlayValues[261] = d261
			ps539.OverlayValues[262] = d262
			ps539.OverlayValues[263] = d263
			ps539.OverlayValues[264] = d264
			ps539.OverlayValues[265] = d265
			ps539.OverlayValues[266] = d266
			ps539.OverlayValues[267] = d267
			ps539.OverlayValues[268] = d268
			ps539.OverlayValues[269] = d269
			ps539.OverlayValues[270] = d270
			ps539.OverlayValues[271] = d271
			ps539.OverlayValues[272] = d272
			ps539.OverlayValues[273] = d273
			ps539.OverlayValues[274] = d274
			ps539.OverlayValues[275] = d275
			ps539.OverlayValues[276] = d276
			ps539.OverlayValues[277] = d277
			ps539.OverlayValues[278] = d278
			ps539.OverlayValues[279] = d279
			ps539.OverlayValues[280] = d280
			ps539.OverlayValues[281] = d281
			ps539.OverlayValues[282] = d282
			ps539.OverlayValues[283] = d283
			ps539.OverlayValues[284] = d284
			ps539.OverlayValues[285] = d285
			ps539.OverlayValues[286] = d286
			ps539.OverlayValues[287] = d287
			ps539.OverlayValues[288] = d288
			ps539.OverlayValues[289] = d289
			ps539.OverlayValues[290] = d290
			ps539.OverlayValues[291] = d291
			ps539.OverlayValues[292] = d292
			ps539.OverlayValues[293] = d293
			ps539.OverlayValues[294] = d294
			ps539.OverlayValues[295] = d295
			ps539.OverlayValues[444] = d444
			ps539.OverlayValues[445] = d445
			ps539.OverlayValues[446] = d446
			ps539.OverlayValues[447] = d447
			ps539.OverlayValues[448] = d448
			ps539.OverlayValues[449] = d449
			ps539.OverlayValues[450] = d450
			ps539.OverlayValues[451] = d451
			ps539.OverlayValues[452] = d452
			ps539.OverlayValues[453] = d453
			ps539.OverlayValues[454] = d454
			ps539.OverlayValues[455] = d455
			ps539.OverlayValues[456] = d456
			ps539.OverlayValues[457] = d457
			ps539.OverlayValues[458] = d458
			ps539.OverlayValues[459] = d459
			ps539.OverlayValues[460] = d460
			ps539.OverlayValues[461] = d461
			ps539.OverlayValues[462] = d462
			ps539.OverlayValues[463] = d463
			ps539.OverlayValues[464] = d464
			ps539.OverlayValues[465] = d465
			ps539.OverlayValues[466] = d466
			ps539.OverlayValues[467] = d467
			ps539.OverlayValues[468] = d468
			ps539.OverlayValues[469] = d469
			ps539.OverlayValues[470] = d470
			ps539.OverlayValues[471] = d471
			ps539.OverlayValues[472] = d472
			ps539.OverlayValues[473] = d473
			ps539.OverlayValues[474] = d474
			ps539.OverlayValues[475] = d475
			ps539.OverlayValues[476] = d476
			ps539.OverlayValues[477] = d477
			ps539.OverlayValues[478] = d478
			ps539.OverlayValues[479] = d479
			ps539.OverlayValues[480] = d480
			ps539.OverlayValues[481] = d481
			ps539.OverlayValues[482] = d482
			ps539.OverlayValues[483] = d483
			ps539.OverlayValues[484] = d484
			ps539.OverlayValues[485] = d485
			ps539.OverlayValues[486] = d486
			ps539.OverlayValues[487] = d487
			ps539.OverlayValues[488] = d488
			ps539.OverlayValues[489] = d489
			ps539.OverlayValues[490] = d490
			ps539.OverlayValues[491] = d491
			ps539.OverlayValues[492] = d492
			ps539.OverlayValues[493] = d493
			ps539.OverlayValues[494] = d494
			ps539.OverlayValues[495] = d495
			ps539.OverlayValues[496] = d496
			ps539.OverlayValues[497] = d497
			ps539.OverlayValues[498] = d498
			ps539.OverlayValues[499] = d499
			ps539.OverlayValues[500] = d500
			ps539.OverlayValues[501] = d501
			ps539.OverlayValues[502] = d502
			ps539.OverlayValues[503] = d503
			ps539.OverlayValues[504] = d504
			ps539.OverlayValues[505] = d505
			ps539.OverlayValues[506] = d506
			ps539.OverlayValues[507] = d507
			ps539.OverlayValues[508] = d508
			ps539.OverlayValues[509] = d509
			ps539.OverlayValues[510] = d510
			ps539.OverlayValues[511] = d511
			ps539.OverlayValues[512] = d512
			ps539.OverlayValues[513] = d513
			ps539.OverlayValues[514] = d514
			ps539.OverlayValues[515] = d515
			ps539.OverlayValues[516] = d516
			ps539.OverlayValues[517] = d517
			ps539.OverlayValues[518] = d518
			ps539.OverlayValues[519] = d519
			ps539.OverlayValues[520] = d520
			ps539.OverlayValues[521] = d521
			ps539.OverlayValues[522] = d522
			ps539.OverlayValues[523] = d523
			ps539.OverlayValues[524] = d524
			ps539.OverlayValues[525] = d525
			ps539.OverlayValues[526] = d526
			ps539.OverlayValues[527] = d527
			ps539.OverlayValues[528] = d528
			ps539.OverlayValues[529] = d529
			ps539.OverlayValues[530] = d530
			ps539.OverlayValues[531] = d531
			ps539.OverlayValues[532] = d532
			ps539.OverlayValues[533] = d533
			ps539.OverlayValues[534] = d534
			ps539.OverlayValues[535] = d535
			ps539.OverlayValues[536] = d536
			ps540 := scm.PhiState{General: true}
			ps540.OverlayValues = make([]scm.JITValueDesc, 537)
			ps540.OverlayValues[0] = d0
			ps540.OverlayValues[1] = d1
			ps540.OverlayValues[9] = d9
			ps540.OverlayValues[10] = d10
			ps540.OverlayValues[11] = d11
			ps540.OverlayValues[12] = d12
			ps540.OverlayValues[13] = d13
			ps540.OverlayValues[14] = d14
			ps540.OverlayValues[15] = d15
			ps540.OverlayValues[16] = d16
			ps540.OverlayValues[17] = d17
			ps540.OverlayValues[18] = d18
			ps540.OverlayValues[19] = d19
			ps540.OverlayValues[20] = d20
			ps540.OverlayValues[21] = d21
			ps540.OverlayValues[22] = d22
			ps540.OverlayValues[23] = d23
			ps540.OverlayValues[24] = d24
			ps540.OverlayValues[25] = d25
			ps540.OverlayValues[26] = d26
			ps540.OverlayValues[27] = d27
			ps540.OverlayValues[28] = d28
			ps540.OverlayValues[29] = d29
			ps540.OverlayValues[30] = d30
			ps540.OverlayValues[31] = d31
			ps540.OverlayValues[32] = d32
			ps540.OverlayValues[33] = d33
			ps540.OverlayValues[34] = d34
			ps540.OverlayValues[35] = d35
			ps540.OverlayValues[36] = d36
			ps540.OverlayValues[37] = d37
			ps540.OverlayValues[38] = d38
			ps540.OverlayValues[39] = d39
			ps540.OverlayValues[40] = d40
			ps540.OverlayValues[41] = d41
			ps540.OverlayValues[42] = d42
			ps540.OverlayValues[43] = d43
			ps540.OverlayValues[44] = d44
			ps540.OverlayValues[45] = d45
			ps540.OverlayValues[46] = d46
			ps540.OverlayValues[47] = d47
			ps540.OverlayValues[48] = d48
			ps540.OverlayValues[49] = d49
			ps540.OverlayValues[50] = d50
			ps540.OverlayValues[51] = d51
			ps540.OverlayValues[52] = d52
			ps540.OverlayValues[104] = d104
			ps540.OverlayValues[105] = d105
			ps540.OverlayValues[106] = d106
			ps540.OverlayValues[107] = d107
			ps540.OverlayValues[108] = d108
			ps540.OverlayValues[109] = d109
			ps540.OverlayValues[110] = d110
			ps540.OverlayValues[111] = d111
			ps540.OverlayValues[112] = d112
			ps540.OverlayValues[113] = d113
			ps540.OverlayValues[114] = d114
			ps540.OverlayValues[115] = d115
			ps540.OverlayValues[116] = d116
			ps540.OverlayValues[117] = d117
			ps540.OverlayValues[118] = d118
			ps540.OverlayValues[119] = d119
			ps540.OverlayValues[120] = d120
			ps540.OverlayValues[121] = d121
			ps540.OverlayValues[122] = d122
			ps540.OverlayValues[123] = d123
			ps540.OverlayValues[124] = d124
			ps540.OverlayValues[125] = d125
			ps540.OverlayValues[126] = d126
			ps540.OverlayValues[127] = d127
			ps540.OverlayValues[128] = d128
			ps540.OverlayValues[129] = d129
			ps540.OverlayValues[130] = d130
			ps540.OverlayValues[131] = d131
			ps540.OverlayValues[132] = d132
			ps540.OverlayValues[133] = d133
			ps540.OverlayValues[134] = d134
			ps540.OverlayValues[135] = d135
			ps540.OverlayValues[136] = d136
			ps540.OverlayValues[137] = d137
			ps540.OverlayValues[138] = d138
			ps540.OverlayValues[139] = d139
			ps540.OverlayValues[140] = d140
			ps540.OverlayValues[141] = d141
			ps540.OverlayValues[142] = d142
			ps540.OverlayValues[143] = d143
			ps540.OverlayValues[144] = d144
			ps540.OverlayValues[145] = d145
			ps540.OverlayValues[146] = d146
			ps540.OverlayValues[147] = d147
			ps540.OverlayValues[243] = d243
			ps540.OverlayValues[244] = d244
			ps540.OverlayValues[245] = d245
			ps540.OverlayValues[246] = d246
			ps540.OverlayValues[247] = d247
			ps540.OverlayValues[248] = d248
			ps540.OverlayValues[249] = d249
			ps540.OverlayValues[250] = d250
			ps540.OverlayValues[251] = d251
			ps540.OverlayValues[252] = d252
			ps540.OverlayValues[253] = d253
			ps540.OverlayValues[254] = d254
			ps540.OverlayValues[255] = d255
			ps540.OverlayValues[256] = d256
			ps540.OverlayValues[257] = d257
			ps540.OverlayValues[258] = d258
			ps540.OverlayValues[259] = d259
			ps540.OverlayValues[260] = d260
			ps540.OverlayValues[261] = d261
			ps540.OverlayValues[262] = d262
			ps540.OverlayValues[263] = d263
			ps540.OverlayValues[264] = d264
			ps540.OverlayValues[265] = d265
			ps540.OverlayValues[266] = d266
			ps540.OverlayValues[267] = d267
			ps540.OverlayValues[268] = d268
			ps540.OverlayValues[269] = d269
			ps540.OverlayValues[270] = d270
			ps540.OverlayValues[271] = d271
			ps540.OverlayValues[272] = d272
			ps540.OverlayValues[273] = d273
			ps540.OverlayValues[274] = d274
			ps540.OverlayValues[275] = d275
			ps540.OverlayValues[276] = d276
			ps540.OverlayValues[277] = d277
			ps540.OverlayValues[278] = d278
			ps540.OverlayValues[279] = d279
			ps540.OverlayValues[280] = d280
			ps540.OverlayValues[281] = d281
			ps540.OverlayValues[282] = d282
			ps540.OverlayValues[283] = d283
			ps540.OverlayValues[284] = d284
			ps540.OverlayValues[285] = d285
			ps540.OverlayValues[286] = d286
			ps540.OverlayValues[287] = d287
			ps540.OverlayValues[288] = d288
			ps540.OverlayValues[289] = d289
			ps540.OverlayValues[290] = d290
			ps540.OverlayValues[291] = d291
			ps540.OverlayValues[292] = d292
			ps540.OverlayValues[293] = d293
			ps540.OverlayValues[294] = d294
			ps540.OverlayValues[295] = d295
			ps540.OverlayValues[444] = d444
			ps540.OverlayValues[445] = d445
			ps540.OverlayValues[446] = d446
			ps540.OverlayValues[447] = d447
			ps540.OverlayValues[448] = d448
			ps540.OverlayValues[449] = d449
			ps540.OverlayValues[450] = d450
			ps540.OverlayValues[451] = d451
			ps540.OverlayValues[452] = d452
			ps540.OverlayValues[453] = d453
			ps540.OverlayValues[454] = d454
			ps540.OverlayValues[455] = d455
			ps540.OverlayValues[456] = d456
			ps540.OverlayValues[457] = d457
			ps540.OverlayValues[458] = d458
			ps540.OverlayValues[459] = d459
			ps540.OverlayValues[460] = d460
			ps540.OverlayValues[461] = d461
			ps540.OverlayValues[462] = d462
			ps540.OverlayValues[463] = d463
			ps540.OverlayValues[464] = d464
			ps540.OverlayValues[465] = d465
			ps540.OverlayValues[466] = d466
			ps540.OverlayValues[467] = d467
			ps540.OverlayValues[468] = d468
			ps540.OverlayValues[469] = d469
			ps540.OverlayValues[470] = d470
			ps540.OverlayValues[471] = d471
			ps540.OverlayValues[472] = d472
			ps540.OverlayValues[473] = d473
			ps540.OverlayValues[474] = d474
			ps540.OverlayValues[475] = d475
			ps540.OverlayValues[476] = d476
			ps540.OverlayValues[477] = d477
			ps540.OverlayValues[478] = d478
			ps540.OverlayValues[479] = d479
			ps540.OverlayValues[480] = d480
			ps540.OverlayValues[481] = d481
			ps540.OverlayValues[482] = d482
			ps540.OverlayValues[483] = d483
			ps540.OverlayValues[484] = d484
			ps540.OverlayValues[485] = d485
			ps540.OverlayValues[486] = d486
			ps540.OverlayValues[487] = d487
			ps540.OverlayValues[488] = d488
			ps540.OverlayValues[489] = d489
			ps540.OverlayValues[490] = d490
			ps540.OverlayValues[491] = d491
			ps540.OverlayValues[492] = d492
			ps540.OverlayValues[493] = d493
			ps540.OverlayValues[494] = d494
			ps540.OverlayValues[495] = d495
			ps540.OverlayValues[496] = d496
			ps540.OverlayValues[497] = d497
			ps540.OverlayValues[498] = d498
			ps540.OverlayValues[499] = d499
			ps540.OverlayValues[500] = d500
			ps540.OverlayValues[501] = d501
			ps540.OverlayValues[502] = d502
			ps540.OverlayValues[503] = d503
			ps540.OverlayValues[504] = d504
			ps540.OverlayValues[505] = d505
			ps540.OverlayValues[506] = d506
			ps540.OverlayValues[507] = d507
			ps540.OverlayValues[508] = d508
			ps540.OverlayValues[509] = d509
			ps540.OverlayValues[510] = d510
			ps540.OverlayValues[511] = d511
			ps540.OverlayValues[512] = d512
			ps540.OverlayValues[513] = d513
			ps540.OverlayValues[514] = d514
			ps540.OverlayValues[515] = d515
			ps540.OverlayValues[516] = d516
			ps540.OverlayValues[517] = d517
			ps540.OverlayValues[518] = d518
			ps540.OverlayValues[519] = d519
			ps540.OverlayValues[520] = d520
			ps540.OverlayValues[521] = d521
			ps540.OverlayValues[522] = d522
			ps540.OverlayValues[523] = d523
			ps540.OverlayValues[524] = d524
			ps540.OverlayValues[525] = d525
			ps540.OverlayValues[526] = d526
			ps540.OverlayValues[527] = d527
			ps540.OverlayValues[528] = d528
			ps540.OverlayValues[529] = d529
			ps540.OverlayValues[530] = d530
			ps540.OverlayValues[531] = d531
			ps540.OverlayValues[532] = d532
			ps540.OverlayValues[533] = d533
			ps540.OverlayValues[534] = d534
			ps540.OverlayValues[535] = d535
			ps540.OverlayValues[536] = d536
			snap541 := d0
			snap542 := d1
			snap543 := d9
			snap544 := d10
			snap545 := d11
			snap546 := d12
			snap547 := d13
			snap548 := d14
			snap549 := d15
			snap550 := d16
			snap551 := d17
			snap552 := d18
			snap553 := d19
			snap554 := d20
			snap555 := d21
			snap556 := d22
			snap557 := d23
			snap558 := d24
			snap559 := d25
			snap560 := d26
			snap561 := d27
			snap562 := d28
			snap563 := d29
			snap564 := d30
			snap565 := d31
			snap566 := d32
			snap567 := d33
			snap568 := d34
			snap569 := d35
			snap570 := d36
			snap571 := d37
			snap572 := d38
			snap573 := d39
			snap574 := d40
			snap575 := d41
			snap576 := d42
			snap577 := d43
			snap578 := d44
			snap579 := d45
			snap580 := d46
			snap581 := d47
			snap582 := d48
			snap583 := d49
			snap584 := d50
			snap585 := d51
			snap586 := d52
			snap587 := d104
			snap588 := d105
			snap589 := d106
			snap590 := d107
			snap591 := d108
			snap592 := d109
			snap593 := d110
			snap594 := d111
			snap595 := d112
			snap596 := d113
			snap597 := d114
			snap598 := d115
			snap599 := d116
			snap600 := d117
			snap601 := d118
			snap602 := d119
			snap603 := d120
			snap604 := d121
			snap605 := d122
			snap606 := d123
			snap607 := d124
			snap608 := d125
			snap609 := d126
			snap610 := d127
			snap611 := d128
			snap612 := d129
			snap613 := d130
			snap614 := d131
			snap615 := d132
			snap616 := d133
			snap617 := d134
			snap618 := d135
			snap619 := d136
			snap620 := d137
			snap621 := d138
			snap622 := d139
			snap623 := d140
			snap624 := d141
			snap625 := d142
			snap626 := d143
			snap627 := d144
			snap628 := d145
			snap629 := d146
			snap630 := d147
			snap631 := d243
			snap632 := d244
			snap633 := d245
			snap634 := d246
			snap635 := d247
			snap636 := d248
			snap637 := d249
			snap638 := d250
			snap639 := d251
			snap640 := d252
			snap641 := d253
			snap642 := d254
			snap643 := d255
			snap644 := d256
			snap645 := d257
			snap646 := d258
			snap647 := d259
			snap648 := d260
			snap649 := d261
			snap650 := d262
			snap651 := d263
			snap652 := d264
			snap653 := d265
			snap654 := d266
			snap655 := d267
			snap656 := d268
			snap657 := d269
			snap658 := d270
			snap659 := d271
			snap660 := d272
			snap661 := d273
			snap662 := d274
			snap663 := d275
			snap664 := d276
			snap665 := d277
			snap666 := d278
			snap667 := d279
			snap668 := d280
			snap669 := d281
			snap670 := d282
			snap671 := d283
			snap672 := d284
			snap673 := d285
			snap674 := d286
			snap675 := d287
			snap676 := d288
			snap677 := d289
			snap678 := d290
			snap679 := d291
			snap680 := d292
			snap681 := d293
			snap682 := d294
			snap683 := d295
			snap684 := d444
			snap685 := d445
			snap686 := d446
			snap687 := d447
			snap688 := d448
			snap689 := d449
			snap690 := d450
			snap691 := d451
			snap692 := d452
			snap693 := d453
			snap694 := d454
			snap695 := d455
			snap696 := d456
			snap697 := d457
			snap698 := d458
			snap699 := d459
			snap700 := d460
			snap701 := d461
			snap702 := d462
			snap703 := d463
			snap704 := d464
			snap705 := d465
			snap706 := d466
			snap707 := d467
			snap708 := d468
			snap709 := d469
			snap710 := d470
			snap711 := d471
			snap712 := d472
			snap713 := d473
			snap714 := d474
			snap715 := d475
			snap716 := d476
			snap717 := d477
			snap718 := d478
			snap719 := d479
			snap720 := d480
			snap721 := d481
			snap722 := d482
			snap723 := d483
			snap724 := d484
			snap725 := d485
			snap726 := d486
			snap727 := d487
			snap728 := d488
			snap729 := d489
			snap730 := d490
			snap731 := d491
			snap732 := d492
			snap733 := d493
			snap734 := d494
			snap735 := d495
			snap736 := d496
			snap737 := d497
			snap738 := d498
			snap739 := d499
			snap740 := d500
			snap741 := d501
			snap742 := d502
			snap743 := d503
			snap744 := d504
			snap745 := d505
			snap746 := d506
			snap747 := d507
			snap748 := d508
			snap749 := d509
			snap750 := d510
			snap751 := d511
			snap752 := d512
			snap753 := d513
			snap754 := d514
			snap755 := d515
			snap756 := d516
			snap757 := d517
			snap758 := d518
			snap759 := d519
			snap760 := d520
			snap761 := d521
			snap762 := d522
			snap763 := d523
			snap764 := d524
			snap765 := d525
			snap766 := d526
			snap767 := d527
			snap768 := d528
			snap769 := d529
			snap770 := d530
			snap771 := d531
			snap772 := d532
			snap773 := d533
			snap774 := d534
			snap775 := d535
			snap776 := d536
			alloc777 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps540)
			}
			ctx.RestoreAllocState(alloc777)
			d0 = snap541
			d1 = snap542
			d9 = snap543
			d10 = snap544
			d11 = snap545
			d12 = snap546
			d13 = snap547
			d14 = snap548
			d15 = snap549
			d16 = snap550
			d17 = snap551
			d18 = snap552
			d19 = snap553
			d20 = snap554
			d21 = snap555
			d22 = snap556
			d23 = snap557
			d24 = snap558
			d25 = snap559
			d26 = snap560
			d27 = snap561
			d28 = snap562
			d29 = snap563
			d30 = snap564
			d31 = snap565
			d32 = snap566
			d33 = snap567
			d34 = snap568
			d35 = snap569
			d36 = snap570
			d37 = snap571
			d38 = snap572
			d39 = snap573
			d40 = snap574
			d41 = snap575
			d42 = snap576
			d43 = snap577
			d44 = snap578
			d45 = snap579
			d46 = snap580
			d47 = snap581
			d48 = snap582
			d49 = snap583
			d50 = snap584
			d51 = snap585
			d52 = snap586
			d104 = snap587
			d105 = snap588
			d106 = snap589
			d107 = snap590
			d108 = snap591
			d109 = snap592
			d110 = snap593
			d111 = snap594
			d112 = snap595
			d113 = snap596
			d114 = snap597
			d115 = snap598
			d116 = snap599
			d117 = snap600
			d118 = snap601
			d119 = snap602
			d120 = snap603
			d121 = snap604
			d122 = snap605
			d123 = snap606
			d124 = snap607
			d125 = snap608
			d126 = snap609
			d127 = snap610
			d128 = snap611
			d129 = snap612
			d130 = snap613
			d131 = snap614
			d132 = snap615
			d133 = snap616
			d134 = snap617
			d135 = snap618
			d136 = snap619
			d137 = snap620
			d138 = snap621
			d139 = snap622
			d140 = snap623
			d141 = snap624
			d142 = snap625
			d143 = snap626
			d144 = snap627
			d145 = snap628
			d146 = snap629
			d147 = snap630
			d243 = snap631
			d244 = snap632
			d245 = snap633
			d246 = snap634
			d247 = snap635
			d248 = snap636
			d249 = snap637
			d250 = snap638
			d251 = snap639
			d252 = snap640
			d253 = snap641
			d254 = snap642
			d255 = snap643
			d256 = snap644
			d257 = snap645
			d258 = snap646
			d259 = snap647
			d260 = snap648
			d261 = snap649
			d262 = snap650
			d263 = snap651
			d264 = snap652
			d265 = snap653
			d266 = snap654
			d267 = snap655
			d268 = snap656
			d269 = snap657
			d270 = snap658
			d271 = snap659
			d272 = snap660
			d273 = snap661
			d274 = snap662
			d275 = snap663
			d276 = snap664
			d277 = snap665
			d278 = snap666
			d279 = snap667
			d280 = snap668
			d281 = snap669
			d282 = snap670
			d283 = snap671
			d284 = snap672
			d285 = snap673
			d286 = snap674
			d287 = snap675
			d288 = snap676
			d289 = snap677
			d290 = snap678
			d291 = snap679
			d292 = snap680
			d293 = snap681
			d294 = snap682
			d295 = snap683
			d444 = snap684
			d445 = snap685
			d446 = snap686
			d447 = snap687
			d448 = snap688
			d449 = snap689
			d450 = snap690
			d451 = snap691
			d452 = snap692
			d453 = snap693
			d454 = snap694
			d455 = snap695
			d456 = snap696
			d457 = snap697
			d458 = snap698
			d459 = snap699
			d460 = snap700
			d461 = snap701
			d462 = snap702
			d463 = snap703
			d464 = snap704
			d465 = snap705
			d466 = snap706
			d467 = snap707
			d468 = snap708
			d469 = snap709
			d470 = snap710
			d471 = snap711
			d472 = snap712
			d473 = snap713
			d474 = snap714
			d475 = snap715
			d476 = snap716
			d477 = snap717
			d478 = snap718
			d479 = snap719
			d480 = snap720
			d481 = snap721
			d482 = snap722
			d483 = snap723
			d484 = snap724
			d485 = snap725
			d486 = snap726
			d487 = snap727
			d488 = snap728
			d489 = snap729
			d490 = snap730
			d491 = snap731
			d492 = snap732
			d493 = snap733
			d494 = snap734
			d495 = snap735
			d496 = snap736
			d497 = snap737
			d498 = snap738
			d499 = snap739
			d500 = snap740
			d501 = snap741
			d502 = snap742
			d503 = snap743
			d504 = snap744
			d505 = snap745
			d506 = snap746
			d507 = snap747
			d508 = snap748
			d509 = snap749
			d510 = snap750
			d511 = snap751
			d512 = snap752
			d513 = snap753
			d514 = snap754
			d515 = snap755
			d516 = snap756
			d517 = snap757
			d518 = snap758
			d519 = snap759
			d520 = snap760
			d521 = snap761
			d522 = snap762
			d523 = snap763
			d524 = snap764
			d525 = snap765
			d526 = snap766
			d527 = snap767
			d528 = snap768
			d529 = snap769
			d530 = snap770
			d531 = snap771
			d532 = snap772
			d533 = snap773
			d534 = snap774
			d535 = snap775
			d536 = snap776
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps539)
			}
			return result
			ctx.FreeDesc(&d535)
			return result
			}
			ps778 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps778)
			ctx.MarkLabel(lbl0)
			d779 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d779)
			ctx.BindReg(r1, &d779)
			ctx.EmitMovPairToResult(&d779, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r5, int32(80))
			ctx.EmitAddRSP32(int32(80))
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
