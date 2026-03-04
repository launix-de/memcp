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
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&idxInt)
			d0 := idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 := ctx.W.EmitSubRSP32Fixup()
			lbl3 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.MarkLabel(lbl7)
			d12 := d9
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			ctx.EmitStoreToStack(d12, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl7)
			d13 := d9
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 0)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d10)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d14 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d15 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
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
				ctx.W.EmitMovRegReg(r18, d15.Reg)
				ctx.W.EmitShlRegImm8(r18, 56)
				ctx.W.EmitShrRegImm8(r18, 56)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d16)
			}
			ctx.FreeDesc(&d15)
			d17 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
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
				ctx.W.EmitMovRegReg(r19, d17.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d18)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else {
				r20 := ctx.AllocRegExcept(d17.Reg, d16.Reg)
				ctx.W.EmitMovRegReg(r20, d17.Reg)
				ctx.W.EmitSubInt64(r20, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) >> uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r21, d14.Reg)
				ctx.W.EmitShrRegImm8(r21, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			} else {
				{
					shiftSrc := d14.Reg
					r22 := ctx.AllocRegExcept(d14.Reg)
					ctx.W.EmitMovRegReg(r22, d14.Reg)
					shiftSrc = r22
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d14.Loc == scm.LocReg && d19.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			ctx.FreeDesc(&d18)
			r23 := ctx.AllocReg()
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r23, d19)
			}
			ctx.W.EmitJmp(lbl3)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d20 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r24 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r24, d4.Reg)
				ctx.W.EmitAndRegImm32(r24, 63)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d20)
			}
			if d20.Loc == scm.LocReg && d4.Loc == scm.LocReg && d20.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
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
				ctx.W.EmitMovRegReg(r26, d21.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
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
				ctx.W.EmitMovRegReg(r27, d20.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r28 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.W.EmitMovRegReg(r28, d20.Reg)
				ctx.W.EmitAddInt64(r28, d22.Reg)
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
				ctx.W.EmitCmpRegImm32(d23.Reg, 64)
				ctx.W.EmitSetcc(r29, scm.CcA)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
				ctx.BindReg(r29, &d24)
			}
			ctx.FreeDesc(&d23)
			d25 := d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != scm.LocImm && d25.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d25.Loc == scm.LocImm {
				if d25.Imm.Bool() {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl10)
			d26 := d9
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl10)
			d27 := d9
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, 0)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d24)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl8)
			ctx.W.ResolveFixups()
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d28 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r30, d4.Reg)
				ctx.W.EmitShrRegImm8(r30, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d28)
			}
			if d28.Loc == scm.LocReg && d4.Loc == scm.LocReg && d28.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(scratch, d28.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
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
			ctx.EnsureDesc(&d5)
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r31, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r31, d29.Reg)
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
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d30)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d4)
			var d31 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r33, d4.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			}
			if d31.Loc == scm.LocReg && d4.Loc == scm.LocReg && d31.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d32 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
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
				ctx.W.EmitMovRegReg(r34, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(scratch, d32.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r35 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.W.EmitMovRegReg(r35, d32.Reg)
				ctx.W.EmitSubInt64(r35, d31.Reg)
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
				ctx.W.EmitMovRegReg(r36, d30.Reg)
				ctx.W.EmitShrRegImm8(r36, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r37 := ctx.AllocRegExcept(d30.Reg)
					ctx.W.EmitMovRegReg(r37, d30.Reg)
					shiftSrc = r37
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d33.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d33.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d34.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r38, d9.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d35)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r39, d9.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r39, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitOrInt64(r39, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d35)
			} else {
				r40 := ctx.AllocRegExcept(d9.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				ctx.W.EmitOrInt64(r40, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			}
			if d35.Loc == scm.LocReg && d9.Loc == scm.LocReg && d35.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			d36 := d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl3)
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
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
				ctx.W.EmitMovRegMemB(r41, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
				ctx.BindReg(r41, &d38)
			}
			d39 := d38
			ctx.EnsureDesc(&d39)
			if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocImm {
				if d39.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d39.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d38)
			bbpos_0_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d40 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d37.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d40)
			}
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d41)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() + d41.Imm.Int())}
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r44, d40.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d42)
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else {
				r45 := ctx.AllocRegExcept(d40.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r45, d40.Reg)
				ctx.W.EmitAddInt64(r45, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d42)
			}
			if d42.Loc == scm.LocReg && d40.Loc == scm.LocReg && d42.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r46, fieldAddr)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d43)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d43)
			}
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d43.Imm.Int() > 0)}
			} else {
				r48 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitSetcc(r48, scm.CcG)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			d45 := d44
			ctx.EnsureDesc(&d45)
			if d45.Loc != scm.LocImm && d45.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.MarkLabel(lbl16)
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.MarkLabel(lbl17)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.FreeDesc(&d44)
			bbpos_0_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl15)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d46 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d42.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d42.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d46)
			}
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d43)
			var d47 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d43.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d43.Reg)
				ctx.W.EmitShlRegImm8(r49, 56)
				ctx.W.EmitSarRegImm8(r49, 56)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d47)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() + 15)}
			} else {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(15))
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			}
			if d48.Loc == scm.LocReg && d47.Loc == scm.LocReg && d48.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.EnsureDesc(&d48)
			r50 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r50, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r51 := ctx.AllocReg()
			if d48.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r51, uint64(d48.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r51, d48.Reg)
				ctx.W.EmitShlRegImm8(r51, 3)
			}
			ctx.W.EmitAddInt64(r50, r51)
			ctx.FreeReg(r51)
			r52 := ctx.AllocRegExcept(r50)
			ctx.W.EmitMovRegMem(r52, r50, 0)
			ctx.FreeReg(r50)
			d49 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			ctx.BindReg(r52, &d49)
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d46.Imm.Float() * d49.Imm.Float())}
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				_, xBits := d46.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scratch, xBits)
				ctx.W.EmitMulFloat64(scratch, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(scratch, d46.Reg)
				_, yBits := d49.Imm.RawWords()
				ctx.W.EmitMovRegImm64(scm.RegR11, yBits)
				ctx.W.EmitMulFloat64(scratch, scm.RegR11)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r53 := ctx.AllocRegExcept(d46.Reg, d49.Reg)
				ctx.W.EmitMovRegReg(r53, d46.Reg)
				ctx.W.EmitMulFloat64(r53, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r53}
				ctx.BindReg(r53, &d50)
			}
			if d50.Loc == scm.LocReg && d46.Loc == scm.LocReg && d50.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d50)
			d51 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d51)
			ctx.BindReg(r1, &d51)
			ctx.EnsureDesc(&d50)
			ctx.W.EmitMakeFloat(d51, d50)
			if d50.Loc == scm.LocReg { ctx.FreeReg(d50.Reg) }
			ctx.W.EmitJmp(lbl0)
			bbpos_0_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl11)
			ctx.W.ResolveFixups()
			var d52 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d52)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d52)
			var d53 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) == uint64(d52.Imm.Int()))}
			} else if d52.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d37.Reg)
				if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d37.Reg, int32(d52.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
					ctx.W.EmitCmpInt64(d37.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcE)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d53)
			} else if d37.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d52.Reg)
				ctx.W.EmitSetcc(r56, scm.CcE)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d53)
			} else {
				r57 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitCmpInt64(d37.Reg, d52.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d53)
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d52)
			d54 := d53
			ctx.EnsureDesc(&d54)
			if d54.Loc != scm.LocImm && d54.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d54.Loc == scm.LocImm {
				if d54.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d54.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d53)
			bbpos_0_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl14)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d43)
			r58 := ctx.AllocReg()
			ctx.W.EmitMovRegImm64(r58, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r59 := ctx.AllocReg()
			if d43.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r59, uint64(d43.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r59, d43.Reg)
				ctx.W.EmitShlRegImm8(r59, 3)
			}
			ctx.W.EmitAddInt64(r58, r59)
			ctx.FreeReg(r59)
			r60 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r60, r58, 0)
			ctx.FreeReg(r58)
			d55 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
			ctx.BindReg(r60, &d55)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() * d55.Imm.Int())}
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d42.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d56)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegReg(scratch, d42.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d56)
			} else {
				r61 := ctx.AllocRegExcept(d42.Reg, d55.Reg)
				ctx.W.EmitMovRegReg(r61, d42.Reg)
				ctx.W.EmitImulInt64(r61, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d56)
			}
			if d56.Loc == scm.LocReg && d42.Loc == scm.LocReg && d56.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d42)
			ctx.FreeDesc(&d55)
			ctx.EnsureDesc(&d56)
			d57 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d57)
			ctx.BindReg(r1, &d57)
			ctx.EnsureDesc(&d56)
			ctx.W.EmitMakeInt(d57, d56)
			if d56.Loc == scm.LocReg { ctx.FreeReg(d56.Reg) }
			ctx.W.EmitJmp(lbl0)
			bbpos_0_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d58 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d58)
			ctx.BindReg(r1, &d58)
			ctx.W.EmitMakeNil(d58)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d59 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d59)
			ctx.BindReg(r1, &d59)
			ctx.EmitMovPairToResult(&d59, &result)
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
