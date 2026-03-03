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
			} else {
				idxInt = idx
			}
			if idxInt.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			r0 := ctx.W.EmitSubRSP32Fixup()
			r1 := ctx.AllocReg()
			lbl1 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r2, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r2, 32)
				ctx.W.EmitShrRegImm8(r2, 32)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			}
			ctx.FreeDesc(&idxInt)
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r3, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			}
			var d2 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r4, d1.Reg)
				ctx.W.EmitShlRegImm8(r4, 56)
				ctx.W.EmitShrRegImm8(r4, 56)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			}
			ctx.FreeDesc(&d1)
			var d3 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() * d2.Imm.Int())}
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d2.Loc == scm.LocImm {
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d0.Reg, int32(d2.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitImulInt64(d0.Reg, scm.RegR11)
				}
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d0.Reg}
			} else {
				ctx.W.EmitImulInt64(d0.Reg, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d0.Reg}
			}
			if d3.Loc == scm.LocReg && d0.Loc == scm.LocReg && d3.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d2)
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r5, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			}
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r6 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r6, d3.Reg)
				ctx.W.EmitShrRegImm8(r6, 6)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			}
			if d5.Loc == scm.LocReg && d3.Loc == scm.LocReg && d5.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			r7 := ctx.AllocReg()
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
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r9 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				ctx.W.EmitAndRegImm32(r9, 63)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			}
			if d7.Loc == scm.LocReg && d3.Loc == scm.LocReg && d7.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) << uint64(d7.Imm.Int())))}
			} else if d7.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d6.Reg, uint8(d7.Imm.Int()))
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d6.Reg}
			} else {
				{
					shiftSrc := d6.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r10, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d9.Loc == scm.LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
			ctx.EmitStoreToStack(d8, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl3)
			r11 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r11, 0)
			ctx.ProtectReg(r11)
			d10 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r11}
			ctx.UnprotectReg(r11)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r12, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			}
			var d12 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d11.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d11.Reg)
				ctx.W.EmitShlRegImm8(r13, 56)
				ctx.W.EmitShrRegImm8(r13, 56)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			}
			ctx.FreeDesc(&d11)
			d13 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d14 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() - d12.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d12.Loc == scm.LocImm {
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d13.Reg, int32(d12.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitSubInt64(d13.Reg, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else {
				ctx.W.EmitSubInt64(d13.Reg, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			}
			if d14.Loc == scm.LocReg && d13.Loc == scm.LocReg && d14.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			var d15 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) >> uint64(d14.Imm.Int())))}
			} else if d14.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d10.Reg, uint8(d14.Imm.Int()))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d10.Reg}
			} else {
				{
					shiftSrc := d10.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d14.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d14.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d14.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d15.Loc == scm.LocReg && d10.Loc == scm.LocReg && d15.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d14)
			ctx.EmitMovToReg(r1, d15)
			ctx.W.EmitJmp(lbl1)
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl2)
			var d16 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r14, d3.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			}
			if d16.Loc == scm.LocReg && d3.Loc == scm.LocReg && d16.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			}
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d17.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d17.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			ctx.FreeDesc(&d17)
			var d19 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() + d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d18.Loc == scm.LocImm {
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d16.Reg, int32(d18.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.W.EmitAddInt64(d16.Reg, scm.RegR11)
				}
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			} else {
				ctx.W.EmitAddInt64(d16.Reg, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			}
			if d19.Loc == scm.LocReg && d16.Loc == scm.LocReg && d19.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.FreeDesc(&d18)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d19.Imm.Int()) > uint64(64))}
			} else {
				r17 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d19.Reg, 64)
				ctx.W.EmitSetcc(r17, scm.CcA)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r17}
			}
			ctx.FreeDesc(&d19)
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d20.Loc == scm.LocImm {
				if d20.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d20.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
			ctx.EmitStoreToStack(d8, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl5)
			var d21 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r18, d3.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			}
			if d21.Loc == scm.LocReg && d3.Loc == scm.LocReg && d21.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d21.Reg, int32(1))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
			}
			if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			r19 := ctx.AllocReg()
			if d22.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d22.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d22.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d4.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d4.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d23 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.FreeDesc(&d22)
			var d24 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d3.Reg, 63)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d3.Reg}
			}
			if d24.Loc == scm.LocReg && d3.Loc == scm.LocReg && d24.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			d25 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d24.Loc == scm.LocImm {
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d25.Reg, int32(d24.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitSubInt64(d25.Reg, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else {
				ctx.W.EmitSubInt64(d25.Reg, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			}
			if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			var d27 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d26.Imm.Int())))}
			} else if d26.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d23.Reg, uint8(d26.Imm.Int()))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else {
				{
					shiftSrc := d23.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d26.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d26.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d26.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d27.Loc == scm.LocReg && d23.Loc == scm.LocReg && d27.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d26)
			var d28 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() | d27.Imm.Int())}
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				r21 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r21, d8.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d27.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r22, d8.Reg)
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r22, int32(d27.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
					ctx.W.EmitOrInt64(r22, scratch)
					ctx.FreeReg(scratch)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			} else {
				r23 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r23, d8.Reg)
				ctx.W.EmitOrInt64(r23, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			}
			if d28.Loc == scm.LocReg && d8.Loc == scm.LocReg && d28.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.EmitStoreToStack(d28, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d29 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.FreeDesc(&idxInt)
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d30.Loc == scm.LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d30)
			ctx.W.MarkLabel(lbl8)
			var d31 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d29.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			}
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r26, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			}
			var d33 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d31.Reg, int32(d32.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.W.EmitAddInt64(d31.Reg, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			} else {
				ctx.W.EmitAddInt64(d31.Reg, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			}
			if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.FreeDesc(&d32)
			var d34 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r27, fieldAddr)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r28, thisptr.Reg, off)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			}
			var d35 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d34.Imm.Int() > 0)}
			} else {
				r29 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitCmpRegImm32(d34.Reg, 0)
				ctx.W.EmitSetcc(r29, scm.CcG)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl7)
			var d36 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r30 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r30, thisptr.Reg, off)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			}
			var d37 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d29.Imm.Int()) == uint64(d36.Imm.Int()))}
			} else if d36.Loc == scm.LocImm {
				r31 := ctx.AllocReg()
				if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d29.Reg, int32(d36.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
					ctx.W.EmitCmpInt64(d29.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r31, scm.CcE)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
			} else if d29.Loc == scm.LocImm {
				r32 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d36.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r32, scm.CcE)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
			} else {
				r33 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d29.Reg, d36.Reg)
				ctx.W.EmitSetcc(r33, scm.CcE)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r33}
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d36)
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d37.Loc == scm.LocImm {
				if d37.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d37.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d37)
			ctx.W.MarkLabel(lbl11)
			var d38 scm.JITValueDesc
			if d33.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d33.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d33.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d33.Reg}
			}
			var d39 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d34.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, d34.Reg)
				ctx.W.EmitShlRegImm8(r34, 56)
				ctx.W.EmitSarRegImm8(r34, 56)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			}
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() + 15)}
			} else {
				ctx.W.EmitAddRegImm32(d39.Reg, int32(15))
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			}
			if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			r35 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r35, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r36 := ctx.AllocReg()
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r36, uint64(d40.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r36, d40.Reg)
				ctx.W.EmitShlRegImm8(r36, 3)
			}
			ctx.W.EmitAddInt64(r35, r36)
			ctx.FreeReg(r36)
			r37 := ctx.AllocRegExcept(r35)
			ctx.W.EmitMovRegMem(r37, r35, 0)
			ctx.FreeReg(r35)
			d41 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() * d41.Imm.Int())}
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d41.Loc == scm.LocImm {
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d38.Reg, int32(d41.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.W.EmitImulInt64(d38.Reg, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
			} else {
				ctx.W.EmitImulInt64(d38.Reg, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
			}
			if d42.Loc == scm.LocReg && d38.Loc == scm.LocReg && d42.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d41)
			ctx.W.EmitMakeFloat(result, d42)
			if d42.Loc == scm.LocReg { ctx.FreeReg(d42.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			r38 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r38, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r39 := ctx.AllocReg()
			if d34.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d34.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d34.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			ctx.W.EmitAddInt64(r38, r39)
			ctx.FreeReg(r39)
			r40 := ctx.AllocRegExcept(r38)
			ctx.W.EmitMovRegMem(r40, r38, 0)
			ctx.FreeReg(r38)
			d43 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			var d44 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() * d43.Imm.Int())}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d43.Loc == scm.LocImm {
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d33.Reg, int32(d43.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.W.EmitImulInt64(d33.Reg, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			} else {
				ctx.W.EmitImulInt64(d33.Reg, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			}
			if d44.Loc == scm.LocReg && d33.Loc == scm.LocReg && d44.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d43)
			ctx.W.EmitMakeInt(result, d44)
			if d44.Loc == scm.LocReg { ctx.FreeReg(d44.Reg) }
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
