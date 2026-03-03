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
				if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r0 := ctx.W.EmitSubRSP32Fixup()
			lbl1 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d0 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r1 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r1, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r1, 32)
				ctx.W.EmitShrRegImm8(r1, 32)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
				ctx.BindReg(r1, &d0)
			}
			ctx.FreeDesc(&idxInt)
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r2, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d1)
			}
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1.Imm.Int()))))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r3, d1.Reg)
				ctx.W.EmitShlRegImm8(r3, 56)
				ctx.W.EmitShrRegImm8(r3, 56)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
				ctx.BindReg(r3, &d2)
			}
			ctx.FreeDesc(&d1)
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() * d2.Imm.Int())}
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d3)
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d3)
			} else {
				r4 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r4, d0.Reg)
				ctx.W.EmitImulInt64(r4, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d3)
			}
			if d3.Loc == scm.LocReg && d0.Loc == scm.LocReg && d3.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d2)
			var d4 scm.JITValueDesc
			r5 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r5, uint64(dataPtr))
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5, StackOff: int32(sliceLen)}
				ctx.BindReg(r5, &d4)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
				ctx.W.EmitMovRegMem(r5, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d4)
			}
			ctx.BindReg(r5, &d4)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r6 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r6, d3.Reg)
				ctx.W.EmitShrRegImm8(r6, 6)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d5)
			}
			if d5.Loc == scm.LocReg && d3.Loc == scm.LocReg && d5.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			r7 := ctx.AllocReg()
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r7, uint64(d5.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r7, d5.Reg)
				ctx.W.EmitShlRegImm8(r7, 3)
			}
			if d4.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(r7, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r7, d4.Reg)
			}
			r8 := ctx.AllocRegExcept(r7)
			ctx.W.EmitMovRegMem(r8, r7, 0)
			ctx.FreeReg(r7)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			ctx.BindReg(r8, &d6)
			ctx.FreeDesc(&d5)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d7 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r9 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				ctx.W.EmitAndRegImm32(r9, 63)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d7)
			}
			if d7.Loc == scm.LocReg && d3.Loc == scm.LocReg && d7.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) << uint64(d7.Imm.Int())))}
			} else if d7.Loc == scm.LocImm {
				r10 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r10, d6.Reg)
				ctx.W.EmitShlRegImm8(r10, uint8(d7.Imm.Int()))
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d8)
			} else {
				{
					shiftSrc := d6.Reg
					r11 := ctx.AllocRegExcept(d6.Reg)
					ctx.W.EmitMovRegReg(r11, d6.Reg)
					shiftSrc = r11
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d7.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d7.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d7.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d8)
				}
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			ctx.FreeDesc(&d7)
			var d9 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 25)
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r12, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d9)
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d9.Loc == scm.LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
			d10 := d8
			if d10.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			ctx.EmitStoreToStack(d10, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
			d11 := d8
			if d11.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			ctx.EmitStoreToStack(d11, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl3)
			d12 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d13 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r13, thisptr.Reg, off)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
				ctx.BindReg(r13, &d13)
			}
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			var d14 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d13.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d13.Reg)
				ctx.W.EmitShlRegImm8(r14, 56)
				ctx.W.EmitShrRegImm8(r14, 56)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d14)
			}
			ctx.FreeDesc(&d13)
			d15 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			var d16 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() - d14.Imm.Int())}
			} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
				r15 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r15, d15.Reg)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d16)
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d15.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d14.Reg)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d16)
			} else if d14.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d14.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d16)
			} else {
				r16 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r16, d15.Reg)
				ctx.W.EmitSubInt64(r16, d14.Reg)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d16)
			}
			if d16.Loc == scm.LocReg && d15.Loc == scm.LocReg && d16.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 scm.JITValueDesc
			if d12.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) >> uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r17, d12.Reg)
				ctx.W.EmitShrRegImm8(r17, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d17)
			} else {
				{
					shiftSrc := d12.Reg
					r18 := ctx.AllocRegExcept(d12.Reg)
					ctx.W.EmitMovRegReg(r18, d12.Reg)
					shiftSrc = r18
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d16.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d16.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d16.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d17)
				}
			}
			if d17.Loc == scm.LocReg && d12.Loc == scm.LocReg && d17.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			ctx.FreeDesc(&d16)
			r19 := ctx.AllocReg()
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			ctx.EmitMovToReg(r19, d17)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl2)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d18 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r20 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r20, d3.Reg)
				ctx.W.EmitAndRegImm32(r20, 63)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			if d18.Loc == scm.LocReg && d3.Loc == scm.LocReg && d18.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			var d19 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r21, thisptr.Reg, off)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
				ctx.BindReg(r21, &d19)
			}
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d19.Imm.Int()))))}
			} else {
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r22, d19.Reg)
				ctx.W.EmitShlRegImm8(r22, 56)
				ctx.W.EmitShrRegImm8(r22, 56)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d20)
			}
			ctx.FreeDesc(&d19)
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() + d20.Imm.Int())}
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				r23 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r23, d18.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d21)
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
				ctx.BindReg(d20.Reg, &d21)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r24 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r24, d18.Reg)
				ctx.W.EmitAddInt64(r24, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d21)
			}
			if d21.Loc == scm.LocReg && d18.Loc == scm.LocReg && d21.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.FreeDesc(&d20)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d21.Imm.Int()) > uint64(64))}
			} else {
				r25 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitCmpRegImm32(d21.Reg, 64)
				ctx.W.EmitSetcc(r25, scm.CcA)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
				ctx.BindReg(r25, &d22)
			}
			ctx.FreeDesc(&d21)
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d22.Loc == scm.LocImm {
				if d22.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
			d23 := d8
			if d23.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			ctx.EmitStoreToStack(d23, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d22.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
			d24 := d8
			if d24.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			ctx.EmitStoreToStack(d24, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d22)
			ctx.W.MarkLabel(lbl5)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d25 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r26 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r26, d3.Reg)
				ctx.W.EmitShrRegImm8(r26, 6)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d25)
			}
			if d25.Loc == scm.LocReg && d3.Loc == scm.LocReg && d25.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(scratch, d25.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			}
			if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			r27 := ctx.AllocReg()
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			if d26.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r27, uint64(d26.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r27, d26.Reg)
				ctx.W.EmitShlRegImm8(r27, 3)
			}
			if d4.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(r27, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r27, d4.Reg)
			}
			r28 := ctx.AllocRegExcept(r27)
			ctx.W.EmitMovRegMem(r28, r27, 0)
			ctx.FreeReg(r27)
			d27 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d27)
			ctx.FreeDesc(&d26)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d28 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r29 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r29, d3.Reg)
				ctx.W.EmitAndRegImm32(r29, 63)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d28)
			}
			if d28.Loc == scm.LocReg && d3.Loc == scm.LocReg && d28.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			d29 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			var d30 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() - d28.Imm.Int())}
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r30, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d30)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d28.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(scratch, d29.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d28.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else {
				r31 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r31, d29.Reg)
				ctx.W.EmitSubInt64(r31, d28.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d30)
			}
			if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d27.Imm.Int()) >> uint64(d30.Imm.Int())))}
			} else if d30.Loc == scm.LocImm {
				r32 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r32, d27.Reg)
				ctx.W.EmitShrRegImm8(r32, uint8(d30.Imm.Int()))
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d31)
			} else {
				{
					shiftSrc := d27.Reg
					r33 := ctx.AllocRegExcept(d27.Reg)
					ctx.W.EmitMovRegReg(r33, d27.Reg)
					shiftSrc = r33
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d30.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d30.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d30.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d31)
				}
			}
			if d31.Loc == scm.LocReg && d27.Loc == scm.LocReg && d31.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.FreeDesc(&d30)
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			var d32 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() | d31.Imm.Int())}
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
				ctx.BindReg(d31.Reg, &d32)
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r34, d8.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d32)
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else if d31.Loc == scm.LocImm {
				r35 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r35, d8.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r35, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitOrInt64(r35, scm.RegR11)
				}
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d32)
			} else {
				r36 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r36, d8.Reg)
				ctx.W.EmitOrInt64(r36, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d32)
			}
			if d32.Loc == scm.LocReg && d8.Loc == scm.LocReg && d32.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			d33 := d32
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d33.Loc == scm.LocStack || d33.Loc == scm.LocStackPair { ctx.EnsureDesc(&d33) }
			ctx.EmitStoreToStack(d33, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d34 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			ctx.BindReg(r19, &d34)
			ctx.BindReg(r19, &d34)
			ctx.FreeDesc(&idxInt)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
				r37 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r37, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
				ctx.BindReg(r37, &d35)
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl8)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d36 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d34.Imm.Int()))))}
			} else {
				r38 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r38, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d36)
			}
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r39 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r39, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
				ctx.BindReg(r39, &d37)
			}
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r40, d36.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d38)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else {
				r41 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r41, d36.Reg)
				ctx.W.EmitAddInt64(r41, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d38)
			}
			if d38.Loc == scm.LocReg && d36.Loc == scm.LocReg && d38.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d37)
			var d39 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r42, fieldAddr)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
				ctx.BindReg(r42, &d39)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r43, thisptr.Reg, off)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d39)
			}
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d39.Imm.Int() > 0)}
			} else {
				r44 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpRegImm32(d39.Reg, 0)
				ctx.W.EmitSetcc(r44, scm.CcG)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r44}
				ctx.BindReg(r44, &d40)
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d40.Loc == scm.LocImm {
				if d40.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d40.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d40)
			ctx.W.MarkLabel(lbl7)
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d41)
			}
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			var d42 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d34.Imm.Int()) == uint64(d41.Imm.Int()))}
			} else if d41.Loc == scm.LocImm {
				r46 := ctx.AllocRegExcept(d34.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d34.Reg, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitCmpInt64(d34.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r46, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
				ctx.BindReg(r46, &d42)
			} else if d34.Loc == scm.LocImm {
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d41.Reg)
				ctx.W.EmitSetcc(r47, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r47}
				ctx.BindReg(r47, &d42)
			} else {
				r48 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitCmpInt64(d34.Reg, d41.Reg)
				ctx.W.EmitSetcc(r48, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
				ctx.BindReg(r48, &d42)
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d41)
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d42.Loc == scm.LocImm {
				if d42.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d42.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl11)
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d43 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d38.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d38.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d38.Reg}
				ctx.BindReg(d38.Reg, &d43)
			}
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d44 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d39.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d39.Reg)
				ctx.W.EmitShlRegImm8(r49, 56)
				ctx.W.EmitSarRegImm8(r49, 56)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d44)
			}
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() + 15)}
			} else {
				scratch := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(scratch, d44.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(15))
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d45)
			}
			if d45.Loc == scm.LocReg && d44.Loc == scm.LocReg && d45.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			r50 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r50, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r51 := ctx.AllocReg()
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r51, uint64(d45.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r51, d45.Reg)
				ctx.W.EmitShlRegImm8(r51, 3)
			}
			ctx.W.EmitAddInt64(r50, r51)
			ctx.FreeReg(r51)
			r52 := ctx.AllocRegExcept(r50)
			ctx.W.EmitMovRegMem(r52, r50, 0)
			ctx.FreeReg(r50)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d46)
			ctx.FreeDesc(&d45)
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			var d47 scm.JITValueDesc
			if d43.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() * d46.Imm.Int())}
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(scratch, d43.Reg)
				if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d46.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else {
				r53 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(r53, d43.Reg)
				ctx.W.EmitImulInt64(r53, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d47)
			}
			if d47.Loc == scm.LocReg && d43.Loc == scm.LocReg && d47.Reg == d43.Reg {
				ctx.TransferReg(d43.Reg)
				d43.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d43)
			ctx.FreeDesc(&d46)
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			ctx.W.EmitMakeFloat(result, d47)
			if d47.Loc == scm.LocReg { ctx.FreeReg(d47.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			r54 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r54, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r55 := ctx.AllocReg()
			if d39.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r55, uint64(d39.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r55, d39.Reg)
				ctx.W.EmitShlRegImm8(r55, 3)
			}
			ctx.W.EmitAddInt64(r54, r55)
			ctx.FreeReg(r55)
			r56 := ctx.AllocRegExcept(r54)
			ctx.W.EmitMovRegMem(r56, r54, 0)
			ctx.FreeReg(r54)
			d48 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.BindReg(r56, &d48)
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d49 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() * d48.Imm.Int())}
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else {
				r57 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r57, d38.Reg)
				ctx.W.EmitImulInt64(r57, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d49)
			}
			if d49.Loc == scm.LocReg && d38.Loc == scm.LocReg && d49.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d48)
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			ctx.W.EmitMakeInt(result, d49)
			if d49.Loc == scm.LocReg { ctx.FreeReg(d49.Reg) }
			result.Type = scm.TagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r0, int32(8))
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
