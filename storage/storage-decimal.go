/*
Copyright (C) 2026  Carl-Philip Hänsch

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
import "math"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"
import "unsafe"

// StorageDecimal stores decimal values as scaled integers using the existing
// StorageInt bit-packing. real_value = stored_int * 10^scaleExp
type StorageDecimal struct {
	inner    StorageInt // embedded, NOT pointer
	scaleExp int8       // real_value = stored_int * 10^scaleExp
}

// pow10f: precomputed float64 powers of ten, index 0 = 10^-15, index 15 = 10^0, ...
// Access: pow10f[exp+15]
var pow10f [34]float64

// pow10i: precomputed int64 powers of ten for exp >= 0
var pow10i = [19]int64{
	1, 10, 100, 1000, 10000, 100000, 1000000, 10000000, 100000000,
	1000000000, 10000000000, 100000000000, 1000000000000,
	10000000000000, 100000000000000, 1000000000000000,
	10000000000000000, 100000000000000000, 1000000000000000000,
}

func init() {
	for i := range pow10f {
		pow10f[i] = math.Pow(10, float64(i-15))
	}
}

// isCloseToInt checks whether v is close enough to an integer value.
// Uses relative epsilon tolerance for float64 imprecision.
func isCloseToInt(v float64) bool {
	return math.Abs(v-math.Round(v)) < 1e-9*math.Max(1.0, math.Abs(v))
}

// trailingZeroPow10 returns how many times v is divisible by 10.
// 100 → 2, 1550 → 1, 7 → 0, 0 → MaxInt8 (infinitely divisible)
func trailingZeroPow10(v int64) int8 {
	if v == 0 {
		return math.MaxInt8
	}
	if v < 0 {
		v = -v
	}
	var exp int8
	for v%10 == 0 {
		v /= 10
		exp++
	}
	return exp
}

// detectFloatScale determines the power-of-ten exponent that describes a float.
// Bidirectional: checks if integer first (→ trailing zeros), else multiplies
// by 10 until integer (→ negative exp), else MinInt8 (not scalable).
//
// 0.0 → MaxInt8, 100.0 → 2, 7.0 → 0, 3.5 → -1, 12.57 → -2, π → MinInt8
func detectFloatScale(f float64) int8 {
	if f == 0 {
		return math.MaxInt8
	}
	v := math.Abs(f)
	// Phase 1: already integer? → positive direction (trailing zeros)
	if isCloseToInt(v) {
		return trailingZeroPow10(int64(math.Round(v)))
	}
	// Phase 2: not integer → negative direction (× 10 until integer)
	scaled := v
	for exp := int8(-1); exp >= -15; exp-- {
		scaled *= 10
		if isCloseToInt(scaled) {
			return exp
		}
	}
	return math.MinInt8
}

func (s *StorageDecimal) ComputeSize() uint {
	return s.inner.ComputeSize() + 2 // 1 byte magic + 1 byte scaleExp
}

func (s *StorageDecimal) String() string {
	return fmt.Sprintf("decimal[1e%d]", s.scaleExp)
}

func (s *StorageDecimal) GetCachedReader() ColumnReader { return s }

func (s *StorageDecimal) GetValue(i uint32) scm.Scmer {
	raw := s.inner.GetValueUInt(i)
	if s.inner.hasNull && raw == s.inner.null {
		return scm.NewNil()
	}
	v := int64(raw) + s.inner.offset
	if s.scaleExp > 0 {
		// multiples of 10^n → result is integer
		return scm.NewInt(v * pow10i[s.scaleExp])
	}
	// scaleExp < 0 → result is float
	return scm.NewFloat(float64(v) * pow10f[int(s.scaleExp)+15])
}
func (s *StorageDecimal) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
			var r4 unsafe.Pointer
			_ = r4
			var d1 scm.JITValueDesc
			_ = d1
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
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
			var d144 scm.JITValueDesc
			_ = d144
			var d145 scm.JITValueDesc
			_ = d145
			var d146 scm.JITValueDesc
			_ = d146
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
			var bbs [6]scm.BBDescriptor
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
			ctx.EnsureDesc(&idxInt)
			d0 = idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 = ctx.EmitSubRSP32Fixup()
			_ = r4
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			lbl7 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d2 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.EmitMovRegReg(r5, d0.Reg)
				ctx.EmitShlRegImm8(r5, 32)
				ctx.EmitShrRegImm8(r5, 32)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d2)
			}
			var d3 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r6 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r6, thisptr.Reg, off)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d3)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegReg(r7, d3.Reg)
				ctx.EmitShlRegImm8(r7, 56)
				ctx.EmitShrRegImm8(r7, 56)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d4)
			}
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			var d5 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() * d4.Imm.Int())}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.EmitImulInt64(scratch, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d4.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else {
				r8 := ctx.AllocRegExcept(d2.Reg, d4.Reg)
				ctx.EmitMovRegReg(r8, d2.Reg)
				ctx.EmitImulInt64(r8, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			if d5.Loc == scm.LocReg && d2.Loc == scm.LocReg && d5.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d4)
			var d6 scm.JITValueDesc
			r9 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r9, uint64(dataPtr))
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9, StackOff: int32(sliceLen)}
				ctx.BindReg(r9, &d6)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
				ctx.EmitMovRegMem(r9, thisptr.Reg, off)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d6)
			}
			ctx.BindReg(r9, &d6)
			ctx.EnsureDesc(&d5)
			var d7 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r10 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r10, d5.Reg)
				ctx.EmitShrRegImm8(r10, 6)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d7)
			}
			if d7.Loc == scm.LocReg && d5.Loc == scm.LocReg && d7.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			r11 := ctx.AllocReg()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r11, uint64(d7.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r11, d7.Reg)
				ctx.EmitShlRegImm8(r11, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r11, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r11, d6.Reg)
			}
			r12 := ctx.AllocRegExcept(r11)
			ctx.EmitMovRegMem(r12, r11, 0)
			ctx.FreeReg(r11)
			d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			ctx.BindReg(r12, &d8)
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d5)
			var d9 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r13 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r13, d5.Reg)
				ctx.EmitAndRegImm32(r13, 63)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d9)
			}
			if d9.Loc == scm.LocReg && d5.Loc == scm.LocReg && d9.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			var d10 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d9.Imm.Int())))}
			} else if d9.Loc == scm.LocImm {
				r14 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r14, d8.Reg)
				ctx.EmitShlRegImm8(r14, uint8(d9.Imm.Int()))
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d10)
			} else {
				{
					shiftSrc := d8.Reg
					r15 := ctx.AllocRegExcept(d8.Reg)
					ctx.EmitMovRegReg(r15, d8.Reg)
					shiftSrc = r15
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d9.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d9.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d9.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d10)
				}
			}
			if d10.Loc == scm.LocReg && d8.Loc == scm.LocReg && d10.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d9)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25)
				r16 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r16, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != scm.LocImm && d12.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl8)
				} else {
					ctx.MarkLabel(lbl11)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d13 = d10
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
					ctx.EmitJmp(lbl9)
				}
			} else {
				ctx.EmitCmpRegImm32(d12.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl10)
				ctx.EmitJmp(lbl11)
				ctx.MarkLabel(lbl10)
				ctx.EmitJmp(lbl8)
				ctx.MarkLabel(lbl11)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d14 = d10
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
				ctx.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d11)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl9)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			var d15 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r17 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r17, thisptr.Reg, off)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d15)
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			var d16 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d15.Imm.Int()))))}
			} else {
				r18 := ctx.AllocReg()
				ctx.EmitMovRegReg(r18, d15.Reg)
				ctx.EmitShlRegImm8(r18, 56)
				ctx.EmitShrRegImm8(r18, 56)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d16)
			}
			ctx.FreeDesc(&d15)
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d16)
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() - d16.Imm.Int())}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				r19 := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(r19, d17.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d18)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.EmitSubInt64(scratch, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(scratch, d17.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else {
				r20 := ctx.AllocRegExcept(d17.Reg, d16.Reg)
				ctx.EmitMovRegReg(r20, d17.Reg)
				ctx.EmitSubInt64(r20, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1.Imm.Int()) >> uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(r21, d1.Reg)
				ctx.EmitShrRegImm8(r21, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			} else {
				{
					shiftSrc := d1.Reg
					r22 := ctx.AllocRegExcept(d1.Reg)
					ctx.EmitMovRegReg(r22, d1.Reg)
					shiftSrc = r22
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d1.Loc == scm.LocReg && d19.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d18)
			r23 := ctx.AllocReg()
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r23, d19)
			}
			ctx.EmitJmp(lbl7)
			bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl8)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d20 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r24 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r24, d5.Reg)
				ctx.EmitAndRegImm32(r24, 63)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d20)
			}
			if d20.Loc == scm.LocReg && d5.Loc == scm.LocReg && d20.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r25 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r25, thisptr.Reg, off)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d21)
			}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.EmitMovRegReg(r26, d21.Reg)
				ctx.EmitShlRegImm8(r26, 56)
				ctx.EmitShrRegImm8(r26, 56)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d22)
			}
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() + d22.Imm.Int())}
			} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(r27, d20.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitAddInt64(scratch, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r28 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.EmitMovRegReg(r28, d20.Reg)
				ctx.EmitAddInt64(r28, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d23)
			}
			if d23.Loc == scm.LocReg && d20.Loc == scm.LocReg && d23.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d23.Imm.Int()) > uint64(64))}
			} else {
				r29 := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitCmpRegImm32(d23.Reg, 64)
				ctx.EmitSetcc(r29, scm.CcA)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
				ctx.BindReg(r29, &d24)
			}
			ctx.FreeDesc(&d23)
			d25 = d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != scm.LocImm && d25.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			if d25.Loc == scm.LocImm {
				if d25.Imm.Bool() {
					ctx.MarkLabel(lbl13)
					ctx.EmitJmp(lbl12)
				} else {
					ctx.MarkLabel(lbl14)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d26 = d10
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
					ctx.EmitJmp(lbl9)
				}
			} else {
				ctx.EmitCmpRegImm32(d25.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl13)
				ctx.EmitJmp(lbl14)
				ctx.MarkLabel(lbl13)
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl14)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d27 = d10
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
				ctx.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d24)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d28 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r30, d5.Reg)
				ctx.EmitShrRegImm8(r30, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d28)
			}
			if d28.Loc == scm.LocReg && d5.Loc == scm.LocReg && d28.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.EmitMovRegReg(scratch, d28.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			}
			if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d29)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d6)
			if d29.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r31, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r31, d29.Reg)
				ctx.EmitShlRegImm8(r31, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r31, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r31, d6.Reg)
			}
			r32 := ctx.AllocRegExcept(r31)
			ctx.EmitMovRegMem(r32, r31, 0)
			ctx.FreeReg(r31)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d30)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d5)
			var d31 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r33, d5.Reg)
				ctx.EmitAndRegImm32(r33, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			}
			if d31.Loc == scm.LocReg && d5.Loc == scm.LocReg && d31.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitMovRegReg(r34, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitMovRegReg(scratch, d32.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r35 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.EmitMovRegReg(r35, d32.Reg)
				ctx.EmitSubInt64(r35, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d33)
			}
			if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(d30.Reg)
				ctx.EmitMovRegReg(r36, d30.Reg)
				ctx.EmitShrRegImm8(r36, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r37 := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegReg(r37, d30.Reg)
					shiftSrc = r37
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d33.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d33.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d34)
				}
			}
			if d34.Loc == scm.LocReg && d30.Loc == scm.LocReg && d34.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d34.Imm.Int())}
			} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r38, d10.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d35)
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
				ctx.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r39, d10.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r39, int32(d34.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.EmitOrInt64(r39, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d35)
			} else {
				r40 := ctx.AllocRegExcept(d10.Reg, d34.Reg)
				ctx.EmitMovRegReg(r40, d10.Reg)
				ctx.EmitOrInt64(r40, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			}
			if d35.Loc == scm.LocReg && d10.Loc == scm.LocReg && d35.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d35)
			if d35.Loc == scm.LocReg {
				ctx.ProtectReg(d35.Reg)
			} else if d35.Loc == scm.LocRegPair {
				ctx.ProtectReg(d35.Reg)
				ctx.ProtectReg(d35.Reg2)
			}
			d36 = d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, int32(bbs[2].PhiBase)+int32(0))
			if d35.Loc == scm.LocReg {
				ctx.UnprotectReg(d35.Reg)
			} else if d35.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d35.Reg)
				ctx.UnprotectReg(d35.Reg2)
			}
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl7)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d37)
			ctx.BindReg(r23, &d37)
			if r2 { ctx.UnprotectReg(r3) }
			ctx.FreeDesc(&idxInt)
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
				r41 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r41, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
				ctx.BindReg(r41, &d38)
			}
			d39 = d38
			ctx.EnsureDesc(&d39)
			if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d39.Loc == scm.LocImm {
				if d39.Imm.Bool() {
			ps40 := scm.PhiState{General: ps.General}
			ps40.OverlayValues = make([]scm.JITValueDesc, 40)
			ps40.OverlayValues[0] = d0
			ps40.OverlayValues[1] = d1
			ps40.OverlayValues[2] = d2
			ps40.OverlayValues[3] = d3
			ps40.OverlayValues[4] = d4
			ps40.OverlayValues[5] = d5
			ps40.OverlayValues[6] = d6
			ps40.OverlayValues[7] = d7
			ps40.OverlayValues[8] = d8
			ps40.OverlayValues[9] = d9
			ps40.OverlayValues[10] = d10
			ps40.OverlayValues[11] = d11
			ps40.OverlayValues[12] = d12
			ps40.OverlayValues[13] = d13
			ps40.OverlayValues[14] = d14
			ps40.OverlayValues[15] = d15
			ps40.OverlayValues[16] = d16
			ps40.OverlayValues[17] = d17
			ps40.OverlayValues[18] = d18
			ps40.OverlayValues[19] = d19
			ps40.OverlayValues[20] = d20
			ps40.OverlayValues[21] = d21
			ps40.OverlayValues[22] = d22
			ps40.OverlayValues[23] = d23
			ps40.OverlayValues[24] = d24
			ps40.OverlayValues[25] = d25
			ps40.OverlayValues[26] = d26
			ps40.OverlayValues[27] = d27
			ps40.OverlayValues[28] = d28
			ps40.OverlayValues[29] = d29
			ps40.OverlayValues[30] = d30
			ps40.OverlayValues[31] = d31
			ps40.OverlayValues[32] = d32
			ps40.OverlayValues[33] = d33
			ps40.OverlayValues[34] = d34
			ps40.OverlayValues[35] = d35
			ps40.OverlayValues[36] = d36
			ps40.OverlayValues[37] = d37
			ps40.OverlayValues[38] = d38
			ps40.OverlayValues[39] = d39
					return bbs[3].RenderPS(ps40)
				}
			ps41 := scm.PhiState{General: ps.General}
			ps41.OverlayValues = make([]scm.JITValueDesc, 40)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[3] = d3
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[7] = d7
			ps41.OverlayValues[8] = d8
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[11] = d11
			ps41.OverlayValues[12] = d12
			ps41.OverlayValues[13] = d13
			ps41.OverlayValues[14] = d14
			ps41.OverlayValues[15] = d15
			ps41.OverlayValues[16] = d16
			ps41.OverlayValues[17] = d17
			ps41.OverlayValues[18] = d18
			ps41.OverlayValues[19] = d19
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[23] = d23
			ps41.OverlayValues[24] = d24
			ps41.OverlayValues[25] = d25
			ps41.OverlayValues[26] = d26
			ps41.OverlayValues[27] = d27
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[29] = d29
			ps41.OverlayValues[30] = d30
			ps41.OverlayValues[31] = d31
			ps41.OverlayValues[32] = d32
			ps41.OverlayValues[33] = d33
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[38] = d38
			ps41.OverlayValues[39] = d39
				return bbs[2].RenderPS(ps41)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d39.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl3)
			ps42 := scm.PhiState{General: true}
			ps42.OverlayValues = make([]scm.JITValueDesc, 40)
			ps42.OverlayValues[0] = d0
			ps42.OverlayValues[1] = d1
			ps42.OverlayValues[2] = d2
			ps42.OverlayValues[3] = d3
			ps42.OverlayValues[4] = d4
			ps42.OverlayValues[5] = d5
			ps42.OverlayValues[6] = d6
			ps42.OverlayValues[7] = d7
			ps42.OverlayValues[8] = d8
			ps42.OverlayValues[9] = d9
			ps42.OverlayValues[10] = d10
			ps42.OverlayValues[11] = d11
			ps42.OverlayValues[12] = d12
			ps42.OverlayValues[13] = d13
			ps42.OverlayValues[14] = d14
			ps42.OverlayValues[15] = d15
			ps42.OverlayValues[16] = d16
			ps42.OverlayValues[17] = d17
			ps42.OverlayValues[18] = d18
			ps42.OverlayValues[19] = d19
			ps42.OverlayValues[20] = d20
			ps42.OverlayValues[21] = d21
			ps42.OverlayValues[22] = d22
			ps42.OverlayValues[23] = d23
			ps42.OverlayValues[24] = d24
			ps42.OverlayValues[25] = d25
			ps42.OverlayValues[26] = d26
			ps42.OverlayValues[27] = d27
			ps42.OverlayValues[28] = d28
			ps42.OverlayValues[29] = d29
			ps42.OverlayValues[30] = d30
			ps42.OverlayValues[31] = d31
			ps42.OverlayValues[32] = d32
			ps42.OverlayValues[33] = d33
			ps42.OverlayValues[34] = d34
			ps42.OverlayValues[35] = d35
			ps42.OverlayValues[36] = d36
			ps42.OverlayValues[37] = d37
			ps42.OverlayValues[38] = d38
			ps42.OverlayValues[39] = d39
			ps43 := scm.PhiState{General: true}
			ps43.OverlayValues = make([]scm.JITValueDesc, 40)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[6] = d6
			ps43.OverlayValues[7] = d7
			ps43.OverlayValues[8] = d8
			ps43.OverlayValues[9] = d9
			ps43.OverlayValues[10] = d10
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[12] = d12
			ps43.OverlayValues[13] = d13
			ps43.OverlayValues[14] = d14
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[16] = d16
			ps43.OverlayValues[17] = d17
			ps43.OverlayValues[18] = d18
			ps43.OverlayValues[19] = d19
			ps43.OverlayValues[20] = d20
			ps43.OverlayValues[21] = d21
			ps43.OverlayValues[22] = d22
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[24] = d24
			ps43.OverlayValues[25] = d25
			ps43.OverlayValues[26] = d26
			ps43.OverlayValues[27] = d27
			ps43.OverlayValues[28] = d28
			ps43.OverlayValues[29] = d29
			ps43.OverlayValues[30] = d30
			ps43.OverlayValues[31] = d31
			ps43.OverlayValues[32] = d32
			ps43.OverlayValues[33] = d33
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[37] = d37
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[39] = d39
			snap44 := d0
			snap45 := d1
			snap46 := d2
			snap47 := d3
			snap48 := d4
			snap49 := d5
			snap50 := d6
			snap51 := d7
			snap52 := d8
			snap53 := d9
			snap54 := d10
			snap55 := d11
			snap56 := d12
			snap57 := d13
			snap58 := d14
			snap59 := d15
			snap60 := d16
			snap61 := d17
			snap62 := d18
			snap63 := d19
			snap64 := d20
			snap65 := d21
			snap66 := d22
			snap67 := d23
			snap68 := d24
			snap69 := d25
			snap70 := d26
			snap71 := d27
			snap72 := d28
			snap73 := d29
			snap74 := d30
			snap75 := d31
			snap76 := d32
			snap77 := d33
			snap78 := d34
			snap79 := d35
			snap80 := d36
			snap81 := d37
			snap82 := d38
			snap83 := d39
			alloc84 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps43)
			}
			ctx.RestoreAllocState(alloc84)
			d0 = snap44
			d1 = snap45
			d2 = snap46
			d3 = snap47
			d4 = snap48
			d5 = snap49
			d6 = snap50
			d7 = snap51
			d8 = snap52
			d9 = snap53
			d10 = snap54
			d11 = snap55
			d12 = snap56
			d13 = snap57
			d14 = snap58
			d15 = snap59
			d16 = snap60
			d17 = snap61
			d18 = snap62
			d19 = snap63
			d20 = snap64
			d21 = snap65
			d22 = snap66
			d23 = snap67
			d24 = snap68
			d25 = snap69
			d26 = snap70
			d27 = snap71
			d28 = snap72
			d29 = snap73
			d30 = snap74
			d31 = snap75
			d32 = snap76
			d33 = snap77
			d34 = snap78
			d35 = snap79
			d36 = snap80
			d37 = snap81
			d38 = snap82
			d39 = snap83
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps42)
			}
			return result
			ctx.FreeDesc(&d38)
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
			ctx.ReclaimUntrackedRegs()
			d85 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d85)
			ctx.BindReg(r1, &d85)
			ctx.EmitMakeNil(d85)
			ctx.EmitJmp(lbl0)
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
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d86 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.EmitMovRegReg(r42, d37.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d86)
			}
			var d87 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r43 := ctx.AllocReg()
				ctx.EmitMovRegMem(r43, thisptr.Reg, off)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d87)
			}
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d87.Imm.Int())}
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegReg(r44, d86.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d88)
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
				ctx.BindReg(d87.Reg, &d88)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.EmitAddInt64(scratch, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegReg(scratch, d86.Reg)
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d87.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else {
				r45 := ctx.AllocRegExcept(d86.Reg, d87.Reg)
				ctx.EmitMovRegReg(r45, d86.Reg)
				ctx.EmitAddInt64(r45, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d88)
			}
			if d88.Loc == scm.LocReg && d86.Loc == scm.LocReg && d88.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d87)
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r46 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r46, fieldAddr)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d89)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r47 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r47, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d89)
			}
			ctx.EnsureDesc(&d89)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d89.Imm.Int() > 0)}
			} else {
				r48 := ctx.AllocRegExcept(d89.Reg)
				ctx.EmitCmpRegImm32(d89.Reg, 0)
				ctx.EmitSetcc(r48, scm.CcG)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
				ctx.BindReg(r48, &d90)
			}
			d91 = d90
			ctx.EnsureDesc(&d91)
			if d91.Loc != scm.LocImm && d91.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d91.Loc == scm.LocImm {
				if d91.Imm.Bool() {
			ps92 := scm.PhiState{General: ps.General}
			ps92.OverlayValues = make([]scm.JITValueDesc, 92)
			ps92.OverlayValues[0] = d0
			ps92.OverlayValues[1] = d1
			ps92.OverlayValues[2] = d2
			ps92.OverlayValues[3] = d3
			ps92.OverlayValues[4] = d4
			ps92.OverlayValues[5] = d5
			ps92.OverlayValues[6] = d6
			ps92.OverlayValues[7] = d7
			ps92.OverlayValues[8] = d8
			ps92.OverlayValues[9] = d9
			ps92.OverlayValues[10] = d10
			ps92.OverlayValues[11] = d11
			ps92.OverlayValues[12] = d12
			ps92.OverlayValues[13] = d13
			ps92.OverlayValues[14] = d14
			ps92.OverlayValues[15] = d15
			ps92.OverlayValues[16] = d16
			ps92.OverlayValues[17] = d17
			ps92.OverlayValues[18] = d18
			ps92.OverlayValues[19] = d19
			ps92.OverlayValues[20] = d20
			ps92.OverlayValues[21] = d21
			ps92.OverlayValues[22] = d22
			ps92.OverlayValues[23] = d23
			ps92.OverlayValues[24] = d24
			ps92.OverlayValues[25] = d25
			ps92.OverlayValues[26] = d26
			ps92.OverlayValues[27] = d27
			ps92.OverlayValues[28] = d28
			ps92.OverlayValues[29] = d29
			ps92.OverlayValues[30] = d30
			ps92.OverlayValues[31] = d31
			ps92.OverlayValues[32] = d32
			ps92.OverlayValues[33] = d33
			ps92.OverlayValues[34] = d34
			ps92.OverlayValues[35] = d35
			ps92.OverlayValues[36] = d36
			ps92.OverlayValues[37] = d37
			ps92.OverlayValues[38] = d38
			ps92.OverlayValues[39] = d39
			ps92.OverlayValues[85] = d85
			ps92.OverlayValues[86] = d86
			ps92.OverlayValues[87] = d87
			ps92.OverlayValues[88] = d88
			ps92.OverlayValues[89] = d89
			ps92.OverlayValues[90] = d90
			ps92.OverlayValues[91] = d91
					return bbs[4].RenderPS(ps92)
				}
			ps93 := scm.PhiState{General: ps.General}
			ps93.OverlayValues = make([]scm.JITValueDesc, 92)
			ps93.OverlayValues[0] = d0
			ps93.OverlayValues[1] = d1
			ps93.OverlayValues[2] = d2
			ps93.OverlayValues[3] = d3
			ps93.OverlayValues[4] = d4
			ps93.OverlayValues[5] = d5
			ps93.OverlayValues[6] = d6
			ps93.OverlayValues[7] = d7
			ps93.OverlayValues[8] = d8
			ps93.OverlayValues[9] = d9
			ps93.OverlayValues[10] = d10
			ps93.OverlayValues[11] = d11
			ps93.OverlayValues[12] = d12
			ps93.OverlayValues[13] = d13
			ps93.OverlayValues[14] = d14
			ps93.OverlayValues[15] = d15
			ps93.OverlayValues[16] = d16
			ps93.OverlayValues[17] = d17
			ps93.OverlayValues[18] = d18
			ps93.OverlayValues[19] = d19
			ps93.OverlayValues[20] = d20
			ps93.OverlayValues[21] = d21
			ps93.OverlayValues[22] = d22
			ps93.OverlayValues[23] = d23
			ps93.OverlayValues[24] = d24
			ps93.OverlayValues[25] = d25
			ps93.OverlayValues[26] = d26
			ps93.OverlayValues[27] = d27
			ps93.OverlayValues[28] = d28
			ps93.OverlayValues[29] = d29
			ps93.OverlayValues[30] = d30
			ps93.OverlayValues[31] = d31
			ps93.OverlayValues[32] = d32
			ps93.OverlayValues[33] = d33
			ps93.OverlayValues[34] = d34
			ps93.OverlayValues[35] = d35
			ps93.OverlayValues[36] = d36
			ps93.OverlayValues[37] = d37
			ps93.OverlayValues[38] = d38
			ps93.OverlayValues[39] = d39
			ps93.OverlayValues[85] = d85
			ps93.OverlayValues[86] = d86
			ps93.OverlayValues[87] = d87
			ps93.OverlayValues[88] = d88
			ps93.OverlayValues[89] = d89
			ps93.OverlayValues[90] = d90
			ps93.OverlayValues[91] = d91
				return bbs[5].RenderPS(ps93)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d91.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl17)
			ctx.EmitJmp(lbl18)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl18)
			ctx.EmitJmp(lbl6)
			ps94 := scm.PhiState{General: true}
			ps94.OverlayValues = make([]scm.JITValueDesc, 92)
			ps94.OverlayValues[0] = d0
			ps94.OverlayValues[1] = d1
			ps94.OverlayValues[2] = d2
			ps94.OverlayValues[3] = d3
			ps94.OverlayValues[4] = d4
			ps94.OverlayValues[5] = d5
			ps94.OverlayValues[6] = d6
			ps94.OverlayValues[7] = d7
			ps94.OverlayValues[8] = d8
			ps94.OverlayValues[9] = d9
			ps94.OverlayValues[10] = d10
			ps94.OverlayValues[11] = d11
			ps94.OverlayValues[12] = d12
			ps94.OverlayValues[13] = d13
			ps94.OverlayValues[14] = d14
			ps94.OverlayValues[15] = d15
			ps94.OverlayValues[16] = d16
			ps94.OverlayValues[17] = d17
			ps94.OverlayValues[18] = d18
			ps94.OverlayValues[19] = d19
			ps94.OverlayValues[20] = d20
			ps94.OverlayValues[21] = d21
			ps94.OverlayValues[22] = d22
			ps94.OverlayValues[23] = d23
			ps94.OverlayValues[24] = d24
			ps94.OverlayValues[25] = d25
			ps94.OverlayValues[26] = d26
			ps94.OverlayValues[27] = d27
			ps94.OverlayValues[28] = d28
			ps94.OverlayValues[29] = d29
			ps94.OverlayValues[30] = d30
			ps94.OverlayValues[31] = d31
			ps94.OverlayValues[32] = d32
			ps94.OverlayValues[33] = d33
			ps94.OverlayValues[34] = d34
			ps94.OverlayValues[35] = d35
			ps94.OverlayValues[36] = d36
			ps94.OverlayValues[37] = d37
			ps94.OverlayValues[38] = d38
			ps94.OverlayValues[39] = d39
			ps94.OverlayValues[85] = d85
			ps94.OverlayValues[86] = d86
			ps94.OverlayValues[87] = d87
			ps94.OverlayValues[88] = d88
			ps94.OverlayValues[89] = d89
			ps94.OverlayValues[90] = d90
			ps94.OverlayValues[91] = d91
			ps95 := scm.PhiState{General: true}
			ps95.OverlayValues = make([]scm.JITValueDesc, 92)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[6] = d6
			ps95.OverlayValues[7] = d7
			ps95.OverlayValues[8] = d8
			ps95.OverlayValues[9] = d9
			ps95.OverlayValues[10] = d10
			ps95.OverlayValues[11] = d11
			ps95.OverlayValues[12] = d12
			ps95.OverlayValues[13] = d13
			ps95.OverlayValues[14] = d14
			ps95.OverlayValues[15] = d15
			ps95.OverlayValues[16] = d16
			ps95.OverlayValues[17] = d17
			ps95.OverlayValues[18] = d18
			ps95.OverlayValues[19] = d19
			ps95.OverlayValues[20] = d20
			ps95.OverlayValues[21] = d21
			ps95.OverlayValues[22] = d22
			ps95.OverlayValues[23] = d23
			ps95.OverlayValues[24] = d24
			ps95.OverlayValues[25] = d25
			ps95.OverlayValues[26] = d26
			ps95.OverlayValues[27] = d27
			ps95.OverlayValues[28] = d28
			ps95.OverlayValues[29] = d29
			ps95.OverlayValues[30] = d30
			ps95.OverlayValues[31] = d31
			ps95.OverlayValues[32] = d32
			ps95.OverlayValues[33] = d33
			ps95.OverlayValues[34] = d34
			ps95.OverlayValues[35] = d35
			ps95.OverlayValues[36] = d36
			ps95.OverlayValues[37] = d37
			ps95.OverlayValues[38] = d38
			ps95.OverlayValues[39] = d39
			ps95.OverlayValues[85] = d85
			ps95.OverlayValues[86] = d86
			ps95.OverlayValues[87] = d87
			ps95.OverlayValues[88] = d88
			ps95.OverlayValues[89] = d89
			ps95.OverlayValues[90] = d90
			ps95.OverlayValues[91] = d91
			snap96 := d0
			snap97 := d1
			snap98 := d2
			snap99 := d3
			snap100 := d4
			snap101 := d5
			snap102 := d6
			snap103 := d7
			snap104 := d8
			snap105 := d9
			snap106 := d10
			snap107 := d11
			snap108 := d12
			snap109 := d13
			snap110 := d14
			snap111 := d15
			snap112 := d16
			snap113 := d17
			snap114 := d18
			snap115 := d19
			snap116 := d20
			snap117 := d21
			snap118 := d22
			snap119 := d23
			snap120 := d24
			snap121 := d25
			snap122 := d26
			snap123 := d27
			snap124 := d28
			snap125 := d29
			snap126 := d30
			snap127 := d31
			snap128 := d32
			snap129 := d33
			snap130 := d34
			snap131 := d35
			snap132 := d36
			snap133 := d37
			snap134 := d38
			snap135 := d39
			snap136 := d85
			snap137 := d86
			snap138 := d87
			snap139 := d88
			snap140 := d89
			snap141 := d90
			snap142 := d91
			alloc143 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps95)
			}
			ctx.RestoreAllocState(alloc143)
			d0 = snap96
			d1 = snap97
			d2 = snap98
			d3 = snap99
			d4 = snap100
			d5 = snap101
			d6 = snap102
			d7 = snap103
			d8 = snap104
			d9 = snap105
			d10 = snap106
			d11 = snap107
			d12 = snap108
			d13 = snap109
			d14 = snap110
			d15 = snap111
			d16 = snap112
			d17 = snap113
			d18 = snap114
			d19 = snap115
			d20 = snap116
			d21 = snap117
			d22 = snap118
			d23 = snap119
			d24 = snap120
			d25 = snap121
			d26 = snap122
			d27 = snap123
			d28 = snap124
			d29 = snap125
			d30 = snap126
			d31 = snap127
			d32 = snap128
			d33 = snap129
			d34 = snap130
			d35 = snap131
			d36 = snap132
			d37 = snap133
			d38 = snap134
			d39 = snap135
			d85 = snap136
			d86 = snap137
			d87 = snap138
			d88 = snap139
			d89 = snap140
			d90 = snap141
			d91 = snap142
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps94)
			}
			return result
			ctx.FreeDesc(&d90)
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
			ctx.ReclaimUntrackedRegs()
			var d144 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r49 := ctx.AllocReg()
				ctx.EmitMovRegMem(r49, thisptr.Reg, off)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d144)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d144)
			var d145 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) == uint64(d144.Imm.Int()))}
			} else if d144.Loc == scm.LocImm {
				r50 := ctx.AllocRegExcept(d37.Reg)
				if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d37.Reg, int32(d144.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
					ctx.EmitCmpInt64(d37.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r50, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
				ctx.BindReg(r50, &d145)
			} else if d37.Loc == scm.LocImm {
				r51 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d144.Reg)
				ctx.EmitSetcc(r51, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
				ctx.BindReg(r51, &d145)
			} else {
				r52 := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitCmpInt64(d37.Reg, d144.Reg)
				ctx.EmitSetcc(r52, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d145)
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d144)
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
			ps147.OverlayValues[15] = d15
			ps147.OverlayValues[16] = d16
			ps147.OverlayValues[17] = d17
			ps147.OverlayValues[18] = d18
			ps147.OverlayValues[19] = d19
			ps147.OverlayValues[20] = d20
			ps147.OverlayValues[21] = d21
			ps147.OverlayValues[22] = d22
			ps147.OverlayValues[23] = d23
			ps147.OverlayValues[24] = d24
			ps147.OverlayValues[25] = d25
			ps147.OverlayValues[26] = d26
			ps147.OverlayValues[27] = d27
			ps147.OverlayValues[28] = d28
			ps147.OverlayValues[29] = d29
			ps147.OverlayValues[30] = d30
			ps147.OverlayValues[31] = d31
			ps147.OverlayValues[32] = d32
			ps147.OverlayValues[33] = d33
			ps147.OverlayValues[34] = d34
			ps147.OverlayValues[35] = d35
			ps147.OverlayValues[36] = d36
			ps147.OverlayValues[37] = d37
			ps147.OverlayValues[38] = d38
			ps147.OverlayValues[39] = d39
			ps147.OverlayValues[85] = d85
			ps147.OverlayValues[86] = d86
			ps147.OverlayValues[87] = d87
			ps147.OverlayValues[88] = d88
			ps147.OverlayValues[89] = d89
			ps147.OverlayValues[90] = d90
			ps147.OverlayValues[91] = d91
			ps147.OverlayValues[144] = d144
			ps147.OverlayValues[145] = d145
			ps147.OverlayValues[146] = d146
					return bbs[1].RenderPS(ps147)
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
			ps148.OverlayValues[85] = d85
			ps148.OverlayValues[86] = d86
			ps148.OverlayValues[87] = d87
			ps148.OverlayValues[88] = d88
			ps148.OverlayValues[89] = d89
			ps148.OverlayValues[90] = d90
			ps148.OverlayValues[91] = d91
			ps148.OverlayValues[144] = d144
			ps148.OverlayValues[145] = d145
			ps148.OverlayValues[146] = d146
				return bbs[2].RenderPS(ps148)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl19 := ctx.ReserveLabel()
			lbl20 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d146.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl19)
			ctx.EmitJmp(lbl20)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl3)
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
			ps149.OverlayValues[85] = d85
			ps149.OverlayValues[86] = d86
			ps149.OverlayValues[87] = d87
			ps149.OverlayValues[88] = d88
			ps149.OverlayValues[89] = d89
			ps149.OverlayValues[90] = d90
			ps149.OverlayValues[91] = d91
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
			ps150.OverlayValues[85] = d85
			ps150.OverlayValues[86] = d86
			ps150.OverlayValues[87] = d87
			ps150.OverlayValues[88] = d88
			ps150.OverlayValues[89] = d89
			ps150.OverlayValues[90] = d90
			ps150.OverlayValues[91] = d91
			ps150.OverlayValues[144] = d144
			ps150.OverlayValues[145] = d145
			ps150.OverlayValues[146] = d146
			snap151 := d0
			snap152 := d1
			snap153 := d2
			snap154 := d3
			snap155 := d4
			snap156 := d5
			snap157 := d6
			snap158 := d7
			snap159 := d8
			snap160 := d9
			snap161 := d10
			snap162 := d11
			snap163 := d12
			snap164 := d13
			snap165 := d14
			snap166 := d15
			snap167 := d16
			snap168 := d17
			snap169 := d18
			snap170 := d19
			snap171 := d20
			snap172 := d21
			snap173 := d22
			snap174 := d23
			snap175 := d24
			snap176 := d25
			snap177 := d26
			snap178 := d27
			snap179 := d28
			snap180 := d29
			snap181 := d30
			snap182 := d31
			snap183 := d32
			snap184 := d33
			snap185 := d34
			snap186 := d35
			snap187 := d36
			snap188 := d37
			snap189 := d38
			snap190 := d39
			snap191 := d85
			snap192 := d86
			snap193 := d87
			snap194 := d88
			snap195 := d89
			snap196 := d90
			snap197 := d91
			snap198 := d144
			snap199 := d145
			snap200 := d146
			alloc201 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps150)
			}
			ctx.RestoreAllocState(alloc201)
			d0 = snap151
			d1 = snap152
			d2 = snap153
			d3 = snap154
			d4 = snap155
			d5 = snap156
			d6 = snap157
			d7 = snap158
			d8 = snap159
			d9 = snap160
			d10 = snap161
			d11 = snap162
			d12 = snap163
			d13 = snap164
			d14 = snap165
			d15 = snap166
			d16 = snap167
			d17 = snap168
			d18 = snap169
			d19 = snap170
			d20 = snap171
			d21 = snap172
			d22 = snap173
			d23 = snap174
			d24 = snap175
			d25 = snap176
			d26 = snap177
			d27 = snap178
			d28 = snap179
			d29 = snap180
			d30 = snap181
			d31 = snap182
			d32 = snap183
			d33 = snap184
			d34 = snap185
			d35 = snap186
			d36 = snap187
			d37 = snap188
			d38 = snap189
			d39 = snap190
			d85 = snap191
			d86 = snap192
			d87 = snap193
			d88 = snap194
			d89 = snap195
			d90 = snap196
			d91 = snap197
			d144 = snap198
			d145 = snap199
			d146 = snap200
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps149)
			}
			return result
			ctx.FreeDesc(&d145)
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
			ctx.EnsureDesc(&d89)
			r53 := ctx.AllocReg()
			ctx.EmitMovRegImm64(r53, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r54 := ctx.AllocReg()
			if d89.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r54, uint64(d89.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r54, d89.Reg)
				ctx.EmitShlRegImm8(r54, 3)
			}
			ctx.EmitAddInt64(r53, r54)
			ctx.FreeReg(r54)
			r55 := ctx.AllocRegExcept(r53)
			ctx.EmitMovRegMem(r55, r53, 0)
			ctx.FreeReg(r53)
			d202 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r55}
			ctx.BindReg(r55, &d202)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d88.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() * d202.Imm.Int())}
			} else if d88.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d88.Imm.Int()))
				ctx.EmitImulInt64(scratch, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d88.Reg)
				ctx.EmitMovRegReg(scratch, d88.Reg)
				if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d202.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r56 := ctx.AllocRegExcept(d88.Reg, d202.Reg)
				ctx.EmitMovRegReg(r56, d88.Reg)
				ctx.EmitImulInt64(r56, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
				ctx.BindReg(r56, &d203)
			}
			if d203.Loc == scm.LocReg && d88.Loc == scm.LocReg && d203.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			d204 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d204)
			ctx.BindReg(r1, &d204)
			ctx.EnsureDesc(&d203)
			ctx.EmitMakeInt(d204, d203)
			if d203.Loc == scm.LocReg { ctx.FreeReg(d203.Reg) }
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
			if len(ps.OverlayValues) > 144 && ps.OverlayValues[144].Loc != scm.LocNone {
				d144 = ps.OverlayValues[144]
			}
			if len(ps.OverlayValues) > 145 && ps.OverlayValues[145].Loc != scm.LocNone {
				d145 = ps.OverlayValues[145]
			}
			if len(ps.OverlayValues) > 146 && ps.OverlayValues[146].Loc != scm.LocNone {
				d146 = ps.OverlayValues[146]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d205 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d88.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(scm.RegX0, d88.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d88.Reg}
				ctx.BindReg(d88.Reg, &d205)
			}
			ctx.FreeDesc(&d88)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d89)
			var d206 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d89.Imm.Int()))))}
			} else {
				r57 := ctx.AllocReg()
				ctx.EmitMovRegReg(r57, d89.Reg)
				ctx.EmitShlRegImm8(r57, 56)
				ctx.EmitSarRegImm8(r57, 56)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d206)
			}
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() + 15)}
			} else {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.EmitMovRegReg(scratch, d206.Reg)
				ctx.EmitAddRegImm32(scratch, int32(15))
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			}
			if d207.Loc == scm.LocReg && d206.Loc == scm.LocReg && d207.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d206)
			ctx.EnsureDesc(&d207)
			r58 := ctx.AllocReg()
			ctx.EmitMovRegImm64(r58, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r59 := ctx.AllocReg()
			if d207.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r59, uint64(d207.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r59, d207.Reg)
				ctx.EmitShlRegImm8(r59, 3)
			}
			ctx.EmitAddInt64(r58, r59)
			ctx.FreeReg(r59)
			r60 := ctx.AllocRegExcept(r58)
			ctx.EmitMovRegMem(r60, r58, 0)
			ctx.FreeReg(r58)
			d208 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
			ctx.BindReg(r60, &d208)
			ctx.FreeDesc(&d207)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d205.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d205.Imm.Float() * d208.Imm.Float())}
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				_, xBits := d205.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitMulFloat64(scratch, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.EmitMovRegReg(scratch, d205.Reg)
				_, yBits := d208.Imm.RawWords()
				ctx.EmitMovRegImm64(scm.RegR11, yBits)
				ctx.EmitMulFloat64(scratch, scm.RegR11)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else {
				r61 := ctx.AllocRegExcept(d205.Reg, d208.Reg)
				ctx.EmitMovRegReg(r61, d205.Reg)
				ctx.EmitMulFloat64(r61, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r61}
				ctx.BindReg(r61, &d209)
			}
			if d209.Loc == scm.LocReg && d205.Loc == scm.LocReg && d209.Reg == d205.Reg {
				ctx.TransferReg(d205.Reg)
				d205.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			d210 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d210)
			ctx.BindReg(r1, &d210)
			ctx.EnsureDesc(&d209)
			ctx.EmitMakeFloat(d210, d209)
			if d209.Loc == scm.LocReg { ctx.FreeReg(d209.Reg) }
			ctx.EmitJmp(lbl0)
			return result
			}
			ps211 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps211)
			ctx.MarkLabel(lbl0)
			d212 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d212)
			ctx.BindReg(r1, &d212)
			ctx.EmitMovPairToResult(&d212, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r4, int32(16))
			ctx.EmitAddRSP32(int32(16))
			return result
}

// scaleValue converts a scm.Scmer to the scaled integer representation
func (s *StorageDecimal) scaleValue(value scm.Scmer) scm.Scmer {
	if value.IsNil() {
		return value
	}
	if s.scaleExp < 0 {
		f := value.Float()
		scaled := math.Round(f * pow10f[int(-s.scaleExp)+15])
		return scm.NewInt(int64(scaled))
	}
	// scaleExp > 0: divide
	v := value.Int()
	return scm.NewInt(v / pow10i[s.scaleExp])
}

func (s *StorageDecimal) prepare() {
	s.inner.prepare()
}

func (s *StorageDecimal) scan(i uint32, value scm.Scmer) {
	s.inner.scan(i, s.scaleValue(value))
}

func (s *StorageDecimal) proposeCompression(i uint32) ColumnStorage {
	return nil // terminal format
}

func (s *StorageDecimal) init(i uint32) {
	s.inner.init(i)
}

func (s *StorageDecimal) build(i uint32, value scm.Scmer) {
	s.inner.build(i, s.scaleValue(value))
}

func (s *StorageDecimal) finish() {
	s.inner.finish()
}

func (s *StorageDecimal) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(13))
	binary.Write(f, binary.LittleEndian, s.scaleExp)
	s.inner.Serialize(f) // writes magic 10 + data
}

func (s *StorageDecimal) Deserialize(f io.Reader) uint {
	binary.Read(f, binary.LittleEndian, &s.scaleExp)
	return s.inner.DeserializeEx(f, true) // reads magic 10 + data
}
