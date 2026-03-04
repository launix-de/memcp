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
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			ctx.EnsureDesc(&idxInt)
			d0 := idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 := ctx.W.EmitSubRSP32Fixup()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl3)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r5, d0.Reg)
				ctx.W.EmitShlRegImm8(r5, 32)
				ctx.W.EmitShrRegImm8(r5, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d1)
			}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r6, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d2)
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r7, d2.Reg)
				ctx.W.EmitShlRegImm8(r7, 56)
				ctx.W.EmitShrRegImm8(r7, 56)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d3)
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() * d3.Imm.Int())}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d4)
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d3.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d4)
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg, d3.Reg)
				ctx.W.EmitMovRegReg(r8, d1.Reg)
				ctx.W.EmitImulInt64(r8, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d4)
			}
			if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d3)
			var d5 scm.JITValueDesc
			r9 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r9, uint64(dataPtr))
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9, StackOff: int32(sliceLen)}
				ctx.BindReg(r9, &d5)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
				ctx.W.EmitMovRegMem(r9, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d5)
			}
			ctx.BindReg(r9, &d5)
			ctx.EnsureDesc(&d4)
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r10 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitShrRegImm8(r10, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d6)
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d6)
			r11 := ctx.AllocReg()
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r11, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r11, d6.Reg)
				ctx.W.EmitShlRegImm8(r11, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r11, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r11, d5.Reg)
			}
			r12 := ctx.AllocRegExcept(r11)
			ctx.W.EmitMovRegMem(r12, r11, 0)
			ctx.FreeReg(r11)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			ctx.BindReg(r12, &d7)
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d4)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r13 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r13, d4.Reg)
				ctx.W.EmitAndRegImm32(r13, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d8)
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) << uint64(d8.Imm.Int())))}
			} else if d8.Loc == scm.LocImm {
				r14 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r14, d7.Reg)
				ctx.W.EmitShlRegImm8(r14, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d9)
			} else {
				{
					shiftSrc := d7.Reg
					r15 := ctx.AllocRegExcept(d7.Reg)
					ctx.W.EmitMovRegReg(r15, d7.Reg)
					shiftSrc = r15
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d8.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d8.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d8.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d9)
				}
			}
			if d9.Loc == scm.LocReg && d7.Loc == scm.LocReg && d9.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d7)
			ctx.FreeDesc(&d8)
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25)
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r16, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d10)
			}
			d11 := d10
			ctx.EnsureDesc(&d11)
			if d11.Loc != scm.LocImm && d11.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d11.Loc == scm.LocImm {
				if d11.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.W.MarkLabel(lbl6)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl7)
			d12 := d9
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			ctx.EmitStoreToStack(d12, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl5)
			d13 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d14 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d14)
			}
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d14)
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d14.Imm.Int()))))}
			} else {
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r18, d14.Reg)
				ctx.W.EmitShlRegImm8(r18, 56)
				ctx.W.EmitShrRegImm8(r18, 56)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d15)
			var d17 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - d15.Imm.Int())}
			} else if d15.Loc == scm.LocImm && d15.Imm.Int() == 0 {
				r19 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r19, d16.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d17)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d15.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else {
				r20 := ctx.AllocRegExcept(d16.Reg, d15.Reg)
				ctx.W.EmitMovRegReg(r20, d16.Reg)
				ctx.W.EmitSubInt64(r20, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d17)
			}
			if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d17)
			var d18 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) >> uint64(d17.Imm.Int())))}
			} else if d17.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r21, d13.Reg)
				ctx.W.EmitShrRegImm8(r21, uint8(d17.Imm.Int()))
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d18)
			} else {
				{
					shiftSrc := d13.Reg
					r22 := ctx.AllocRegExcept(d13.Reg)
					ctx.W.EmitMovRegReg(r22, d13.Reg)
					shiftSrc = r22
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d17.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d17.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d17.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d18)
				}
			}
			if d18.Loc == scm.LocReg && d13.Loc == scm.LocReg && d18.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d17)
			r23 := ctx.AllocReg()
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			if d18.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r23, d18)
			}
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl4)
			d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d19 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r24 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r24, d4.Reg)
				ctx.W.EmitAndRegImm32(r24, 63)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d19)
			}
			if d19.Loc == scm.LocReg && d4.Loc == scm.LocReg && d19.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d20)
			}
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d20.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d20.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d21)
			}
			ctx.FreeDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() + d21.Imm.Int())}
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r27, d19.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d22)
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
				ctx.BindReg(d21.Reg, &d22)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(scratch, d19.Reg)
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else {
				r28 := ctx.AllocRegExcept(d19.Reg, d21.Reg)
				ctx.W.EmitMovRegReg(r28, d19.Reg)
				ctx.W.EmitAddInt64(r28, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d22)
			}
			if d22.Loc == scm.LocReg && d19.Loc == scm.LocReg && d22.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d22.Imm.Int()) > uint64(64))}
			} else {
				r29 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitCmpRegImm32(d22.Reg, 64)
				ctx.W.EmitSetcc(r29, scm.CcA)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
				ctx.BindReg(r29, &d23)
			}
			ctx.FreeDesc(&d22)
			d24 := d23
			ctx.EnsureDesc(&d24)
			if d24.Loc != scm.LocImm && d24.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d24.Loc == scm.LocImm {
				if d24.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d24.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl10)
			d25 := d9
			if d25.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.FreeDesc(&d23)
			ctx.W.MarkLabel(lbl8)
			d13 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d26 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r30, d4.Reg)
				ctx.W.EmitShrRegImm8(r30, 6)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d26)
			}
			if d26.Loc == scm.LocReg && d4.Loc == scm.LocReg && d26.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d26)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.EnsureDesc(&d27)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d5)
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r31, uint64(d27.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r31, d27.Reg)
				ctx.W.EmitShlRegImm8(r31, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r31, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r31, d5.Reg)
			}
			r32 := ctx.AllocRegExcept(r31)
			ctx.W.EmitMovRegMem(r32, r31, 0)
			ctx.FreeReg(r31)
			d28 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d28)
			ctx.FreeDesc(&d27)
			ctx.EnsureDesc(&d4)
			var d29 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r33, d4.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d29)
			}
			if d29.Loc == scm.LocReg && d4.Loc == scm.LocReg && d29.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d30 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d29)
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() - d29.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r34, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d31)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r35 := ctx.AllocRegExcept(d30.Reg, d29.Reg)
				ctx.W.EmitMovRegReg(r35, d30.Reg)
				ctx.W.EmitSubInt64(r35, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d31)
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) >> uint64(d31.Imm.Int())))}
			} else if d31.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r36, d28.Reg)
				ctx.W.EmitShrRegImm8(r36, uint8(d31.Imm.Int()))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d32)
			} else {
				{
					shiftSrc := d28.Reg
					r37 := ctx.AllocRegExcept(d28.Reg)
					ctx.W.EmitMovRegReg(r37, d28.Reg)
					shiftSrc = r37
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d31.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d31.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d31.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d32)
				}
			}
			if d32.Loc == scm.LocReg && d28.Loc == scm.LocReg && d32.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d32)
			var d33 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d32.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d33)
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r38, d9.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d33)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r39, d9.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r39, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitOrInt64(r39, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d33)
			} else {
				r40 := ctx.AllocRegExcept(d9.Reg, d32.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				ctx.W.EmitOrInt64(r40, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d33)
			}
			if d33.Loc == scm.LocReg && d9.Loc == scm.LocReg && d33.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			d34 := d33
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d34)
			ctx.EmitStoreToStack(d34, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl2)
			d35 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d35)
			ctx.BindReg(r23, &d35)
			if r2 { ctx.UnprotectReg(r3) }
			ctx.FreeDesc(&idxInt)
			var d36 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r41, thisptr.Reg, off)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
				ctx.BindReg(r41, &d36)
			}
			d37 := d36
			ctx.EnsureDesc(&d37)
			if d37.Loc != scm.LocImm && d37.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d37.Loc == scm.LocImm {
				if d37.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d37.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitJmp(lbl12)
			ctx.FreeDesc(&d36)
			ctx.W.MarkLabel(lbl12)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d35)
			var d38 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d35.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d35.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			}
			var d39 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d39)
			}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r44, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d40)
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
				ctx.BindReg(d39.Reg, &d40)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r45 := ctx.AllocRegExcept(d38.Reg, d39.Reg)
				ctx.W.EmitMovRegReg(r45, d38.Reg)
				ctx.W.EmitAddInt64(r45, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d40)
			}
			if d40.Loc == scm.LocReg && d38.Loc == scm.LocReg && d40.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d39)
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r46, fieldAddr)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d41)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d41)
			}
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d41.Imm.Int() > 0)}
			} else {
				r48 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitCmpRegImm32(d41.Reg, 0)
				ctx.W.EmitSetcc(r48, scm.CcG)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
				ctx.BindReg(r48, &d42)
			}
			d43 := d42
			ctx.EnsureDesc(&d43)
			if d43.Loc != scm.LocImm && d43.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.W.MarkLabel(lbl17)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitJmp(lbl16)
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl11)
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r49, thisptr.Reg, off)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d44)
			}
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d35.Imm.Int()) == uint64(d44.Imm.Int()))}
			} else if d44.Loc == scm.LocImm {
				r50 := ctx.AllocRegExcept(d35.Reg)
				if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d35.Reg, int32(d44.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
					ctx.W.EmitCmpInt64(d35.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r50, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
				ctx.BindReg(r50, &d45)
			} else if d35.Loc == scm.LocImm {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d35.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d44.Reg)
				ctx.W.EmitSetcc(r51, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
				ctx.BindReg(r51, &d45)
			} else {
				r52 := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitCmpInt64(d35.Reg, d44.Reg)
				ctx.W.EmitSetcc(r52, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d45)
			}
			ctx.FreeDesc(&d35)
			ctx.FreeDesc(&d44)
			d46 := d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl20)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl21)
			ctx.W.EmitJmp(lbl12)
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl16)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d47 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d40.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d40.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d40.Reg}
				ctx.BindReg(d40.Reg, &d47)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			var d48 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d41.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d41.Reg)
				ctx.W.EmitShlRegImm8(r53, 56)
				ctx.W.EmitSarRegImm8(r53, 56)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d48)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() + 15)}
			} else {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(15))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			}
			if d49.Loc == scm.LocReg && d48.Loc == scm.LocReg && d49.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d49)
			r54 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r54, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r55 := ctx.AllocReg()
			if d49.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r55, uint64(d49.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r55, d49.Reg)
				ctx.W.EmitShlRegImm8(r55, 3)
			}
			ctx.W.EmitAddInt64(r54, r55)
			ctx.FreeReg(r55)
			r56 := ctx.AllocRegExcept(r54)
			ctx.W.EmitMovRegMem(r56, r54, 0)
			ctx.FreeReg(r54)
			d50 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.BindReg(r56, &d50)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d47.Imm.Float() * d50.Imm.Float())}
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d50.Reg)
				_, xBits := d47.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitMulFloat64(scratch, d50.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				_, yBits := d50.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scm.RegR11, yBits)
				ctx.W.EmitMulFloat64(scratch, scm.RegR11)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else {
				r57 := ctx.AllocRegExcept(d47.Reg, d50.Reg)
				ctx.W.EmitMovRegReg(r57, d47.Reg)
				ctx.W.EmitMulFloat64(r57, d50.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r57}
				ctx.BindReg(r57, &d51)
			}
			if d51.Loc == scm.LocReg && d47.Loc == scm.LocReg && d51.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d51)
			d52 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d52)
			ctx.BindReg(r1, &d52)
			ctx.EnsureDesc(&d51)
			ctx.W.EmitMakeFloat(d52, d51)
			if d51.Loc == scm.LocReg { ctx.FreeReg(d51.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			ctx.EnsureDesc(&d41)
			r58 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r58, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r59 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r59, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r59, d41.Reg)
				ctx.W.EmitShlRegImm8(r59, 3)
			}
			ctx.W.EmitAddInt64(r58, r59)
			ctx.FreeReg(r59)
			r60 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r60, r58, 0)
			ctx.FreeReg(r58)
			d53 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
			ctx.BindReg(r60, &d53)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d53)
			var d54 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() * d53.Imm.Int())}
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d53.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d53.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else {
				r61 := ctx.AllocRegExcept(d40.Reg, d53.Reg)
				ctx.W.EmitMovRegReg(r61, d40.Reg)
				ctx.W.EmitImulInt64(r61, d53.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d54)
			}
			if d54.Loc == scm.LocReg && d40.Loc == scm.LocReg && d54.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d54)
			d55 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d55)
			ctx.BindReg(r1, &d55)
			ctx.EnsureDesc(&d54)
			ctx.W.EmitMakeInt(d55, d54)
			if d54.Loc == scm.LocReg { ctx.FreeReg(d54.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			d56 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d56)
			ctx.BindReg(r1, &d56)
			ctx.W.EmitMakeNil(d56)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d57 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d57)
			ctx.BindReg(r1, &d57)
			ctx.EmitMovPairToResult(&d57, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r4, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
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
