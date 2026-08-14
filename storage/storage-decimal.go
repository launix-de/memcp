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
	return fmt.Sprintf("decimal[1e%d %s]", s.scaleExp, s.inner.String())
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

// StorageDecimal binary layout (magic byte 13 consumed by shard loader):
//
//	[scaleExp int8]      ← power-of-ten exponent: real_value = stored_int * 10^scaleExp
//	[inner StorageInt]   ← with its own magic byte 10
//
// Version history:
//
//	v0 (original, no version byte): layout as above.  The first byte after the
//	magic is scaleExp (int8, typically non-zero), so there is no safe location
//	for an inline version byte.  Format changes require a NEW magic byte in
//	storages[] (storage.go); keep magic 13 as a legacy reader forever.

func (s *StorageDecimal) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
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
			var d127 scm.JITValueDesc
			_ = d127
			var d128 scm.JITValueDesc
			_ = d128
			var d129 scm.JITValueDesc
			_ = d129
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
			phiBase1 := ctx.AllocStack(int32(16))
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase1)+int32(0)}
			lbl7 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d3 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.EmitMovRegReg(r4, d0.Reg)
				ctx.EmitShlRegImm8(r4, 32)
				ctx.EmitShrRegImm8(r4, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d3)
			}
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
				r5 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r5, fieldAddr)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d4)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
				r6 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r6, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d4)
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d5 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegReg(r7, d4.Reg)
				ctx.EmitShlRegImm8(r7, 56)
				ctx.EmitShrRegImm8(r7, 56)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d5)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d3)
			ctx.ProtectReg(d3.Reg)
			ctx.EnsureDesc(&d5)
			ctx.UnprotectReg(d3.Reg)
			var d6 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() * d5.Imm.Int())}
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.EmitImulInt64(scratch, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.EmitMovRegReg(scratch, d3.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d5.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else {
				r8 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
				ctx.EmitMovRegReg(r8, d3.Reg)
				ctx.EmitImulInt64(r8, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d6)
			}
			if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
				r9 := ctx.AllocReg()
				r10 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r9, fieldAddr)
				ctx.EmitMovRegMem64(r10, fieldAddr+8)
				d7 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r9, Reg2: r10}
				ctx.BindReg(r9, &d7)
				ctx.BindReg(r10, &d7)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
				r11 := ctx.AllocReg()
				r12 := ctx.AllocReg()
				ctx.EmitMovRegMem(r11, thisptr.Reg, off)
				ctx.EmitMovRegMem(r12, thisptr.Reg, off+8)
				d7 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r11, Reg2: r12}
				ctx.BindReg(r11, &d7)
				ctx.BindReg(r12, &d7)
			}
			ctx.EnsureDesc(&d6)
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r13, d6.Reg)
				ctx.EmitShrRegImm8(r13, 6)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d8)
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			r14 := ctx.AllocReg()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			if d8.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r14, uint64(d8.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r14, d8.Reg)
				ctx.EmitShlRegImm8(r14, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r14, d7.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.BindReg(r15, &d9)
			ctx.FreeDesc(&d8)
			ctx.EnsureDesc(&d6)
			var d10 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r16, d6.Reg)
				ctx.EmitAndRegImm32(r16, 63)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d10)
			}
			if d10.Loc == scm.LocReg && d6.Loc == scm.LocReg && d10.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) << uint64(d10.Imm.Int())))}
			} else if d10.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d9.Reg)
				ctx.EmitMovRegReg(r17, d9.Reg)
				ctx.EmitShlRegImm8(r17, uint8(d10.Imm.Int()))
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d11)
			} else {
				{
					shiftSrc := d9.Reg
					r18 := ctx.AllocRegExcept(d9.Reg)
					ctx.EmitMovRegReg(r18, d9.Reg)
					shiftSrc = r18
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d10.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d10.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d10.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d11)
				}
			}
			if d11.Loc == scm.LocReg && d9.Loc == scm.LocReg && d11.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d10)
			ctx.EnsureDesc(&d6)
			var d12 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r19 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r19, d6.Reg)
				ctx.EmitAndRegImm32(r19, 63)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d12)
			}
			if d12.Loc == scm.LocReg && d6.Loc == scm.LocReg && d12.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d13 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r20 := ctx.AllocReg()
				ctx.EmitMovRegReg(r20, d4.Reg)
				ctx.EmitShlRegImm8(r20, 56)
				ctx.EmitShrRegImm8(r20, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d13)
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d12)
			ctx.ProtectReg(d12.Reg)
			ctx.EnsureDesc(&d13)
			ctx.UnprotectReg(d12.Reg)
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() + d13.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				r21 := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(r21, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d14)
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
				ctx.BindReg(d13.Reg, &d14)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.EmitAddInt64(scratch, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(scratch, d12.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r22 := ctx.AllocRegExcept(d12.Reg, d13.Reg)
				ctx.EmitMovRegReg(r22, d12.Reg)
				ctx.EmitAddInt64(r22, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d14)
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			ctx.FreeDesc(&d13)
			ctx.EnsureDesc(&d14)
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d14.Imm.Int()) > uint64(64))}
			} else {
				r23 := ctx.AllocRegExcept(d14.Reg)
				ctx.EmitCmpRegImm32(d14.Reg, 64)
				ctx.EmitSetcc(r23, scm.CcA)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r23}
				ctx.BindReg(r23, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 = d15
			ctx.EnsureDesc(&d16)
			if d16.Loc != scm.LocImm && d16.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			if d16.Loc == scm.LocImm {
				if d16.Imm.Bool() {
					ctx.MarkLabel(lbl10)
					ctx.EmitJmp(lbl8)
				} else {
					ctx.MarkLabel(lbl11)
			ctx.EnsureDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d17 = d11
			if d17.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d17)
			ctx.EmitStoreToStack(d17, int32(bbs[2].PhiBase)+int32(0))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
					ctx.EmitJmp(lbl9)
				}
			} else {
				ctx.EmitCmpRegImm32(d16.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl10)
				ctx.EmitJmp(lbl11)
				ctx.MarkLabel(lbl10)
				ctx.EmitJmp(lbl8)
				ctx.MarkLabel(lbl11)
			ctx.EnsureDesc(&d11)
			if d11.Loc == scm.LocReg {
				ctx.ProtectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.ProtectReg(d11.Reg)
				ctx.ProtectReg(d11.Reg2)
			}
			d18 = d11
			if d18.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d18)
			ctx.EmitStoreToStack(d18, int32(bbs[2].PhiBase)+int32(0))
			if d11.Loc == scm.LocReg {
				ctx.UnprotectReg(d11.Reg)
			} else if d11.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d11.Reg)
				ctx.UnprotectReg(d11.Reg2)
			}
				ctx.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d15)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl9)
			ctx.ResolveFixups()
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d19 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r24 := ctx.AllocReg()
				ctx.EmitMovRegReg(r24, d4.Reg)
				ctx.EmitShlRegImm8(r24, 56)
				ctx.EmitShrRegImm8(r24, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d19)
			}
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.ProtectReg(d20.Reg)
			ctx.EnsureDesc(&d19)
			ctx.UnprotectReg(d20.Reg)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(r25, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitSubInt64(scratch, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r26 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
				ctx.EmitMovRegReg(r26, d20.Reg)
				ctx.EmitSubInt64(r26, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d21)
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				r27 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(r27, d2.Reg)
				ctx.EmitShrRegImm8(r27, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d22)
			} else {
				{
					shiftSrc := d2.Reg
					r28 := ctx.AllocRegExcept(d2.Reg)
					ctx.EmitMovRegReg(r28, d2.Reg)
					shiftSrc = r28
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d22)
				}
			}
			if d22.Loc == scm.LocReg && d2.Loc == scm.LocReg && d22.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d21)
			r29 := ctx.AllocReg()
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			if d22.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r29, d22)
			}
			ctx.EmitJmp(lbl7)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl8)
			ctx.ResolveFixups()
			d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d23 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r30, d6.Reg)
				ctx.EmitShrRegImm8(r30, 6)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d23)
			}
			if d23.Loc == scm.LocReg && d6.Loc == scm.LocReg && d23.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.EmitMovRegReg(scratch, d23.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			}
			if d24.Loc == scm.LocReg && d23.Loc == scm.LocReg && d24.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d24)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d7)
			if d24.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r31, uint64(d24.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r31, d24.Reg)
				ctx.EmitShlRegImm8(r31, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.EmitAddInt64(r31, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r31, d7.Reg)
			}
			r32 := ctx.AllocRegExcept(r31)
			ctx.EmitMovRegMem(r32, r31, 0)
			ctx.FreeReg(r31)
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d25)
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d6)
			var d26 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d6.Reg)
				ctx.EmitMovRegReg(r33, d6.Reg)
				ctx.EmitAndRegImm32(r33, 63)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d26)
			}
			if d26.Loc == scm.LocReg && d6.Loc == scm.LocReg && d26.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d27)
			ctx.ProtectReg(d27.Reg)
			ctx.EnsureDesc(&d26)
			ctx.UnprotectReg(d27.Reg)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() - d26.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d27.Reg)
				ctx.EmitMovRegReg(r34, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d28)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.EmitSubInt64(scratch, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.EmitMovRegReg(scratch, d27.Reg)
				if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d26.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r35 := ctx.AllocRegExcept(d27.Reg, d26.Reg)
				ctx.EmitMovRegReg(r35, d27.Reg)
				ctx.EmitSubInt64(r35, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d28)
			}
			if d28.Loc == scm.LocReg && d27.Loc == scm.LocReg && d28.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d25.Imm.Int()) >> uint64(d28.Imm.Int())))}
			} else if d28.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(d25.Reg)
				ctx.EmitMovRegReg(r36, d25.Reg)
				ctx.EmitShrRegImm8(r36, uint8(d28.Imm.Int()))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d29)
			} else {
				{
					shiftSrc := d25.Reg
					r37 := ctx.AllocRegExcept(d25.Reg)
					ctx.EmitMovRegReg(r37, d25.Reg)
					shiftSrc = r37
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d28.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d28.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d29)
				}
			}
			if d29.Loc == scm.LocReg && d25.Loc == scm.LocReg && d29.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() | d29.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r38, d11.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d30)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.EmitOrInt64(scratch, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d29.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d11.Reg)
				ctx.EmitMovRegReg(r39, d11.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r39, int32(d29.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.EmitOrInt64(r39, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d30)
			} else {
				r40 := ctx.AllocRegExcept(d11.Reg, d29.Reg)
				ctx.EmitMovRegReg(r40, d11.Reg)
				ctx.EmitOrInt64(r40, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d30)
			}
			if d30.Loc == scm.LocReg && d11.Loc == scm.LocReg && d30.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d30)
			if d30.Loc == scm.LocReg {
				ctx.ProtectReg(d30.Reg)
			} else if d30.Loc == scm.LocRegPair {
				ctx.ProtectReg(d30.Reg)
				ctx.ProtectReg(d30.Reg2)
			}
			d31 = d30
			if d31.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			ctx.EmitStoreToStack(d31, int32(bbs[2].PhiBase)+int32(0))
			if d30.Loc == scm.LocReg {
				ctx.UnprotectReg(d30.Reg)
			} else if d30.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d30.Reg)
				ctx.UnprotectReg(d30.Reg2)
			}
			ctx.EmitJmp(lbl9)
			ctx.MarkLabel(lbl7)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d32)
			ctx.BindReg(r29, &d32)
			if r2 { ctx.UnprotectReg(r3) }
			ctx.FreeDesc(&idxInt)
			var d33 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
				r41 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r41, fieldAddr)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
				ctx.BindReg(r41, &d33)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
				r42 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r42, thisptr.Reg, off)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
				ctx.BindReg(r42, &d33)
			}
			d34 = d33
			ctx.EnsureDesc(&d34)
			if d34.Loc != scm.LocImm && d34.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d34.Loc == scm.LocImm {
				if d34.Imm.Bool() {
			ps35 := scm.PhiState{General: ps.General}
			ps35.OverlayValues = make([]scm.JITValueDesc, 35)
			ps35.OverlayValues[0] = d0
			ps35.OverlayValues[2] = d2
			ps35.OverlayValues[3] = d3
			ps35.OverlayValues[4] = d4
			ps35.OverlayValues[5] = d5
			ps35.OverlayValues[6] = d6
			ps35.OverlayValues[7] = d7
			ps35.OverlayValues[8] = d8
			ps35.OverlayValues[9] = d9
			ps35.OverlayValues[10] = d10
			ps35.OverlayValues[11] = d11
			ps35.OverlayValues[12] = d12
			ps35.OverlayValues[13] = d13
			ps35.OverlayValues[14] = d14
			ps35.OverlayValues[15] = d15
			ps35.OverlayValues[16] = d16
			ps35.OverlayValues[17] = d17
			ps35.OverlayValues[18] = d18
			ps35.OverlayValues[19] = d19
			ps35.OverlayValues[20] = d20
			ps35.OverlayValues[21] = d21
			ps35.OverlayValues[22] = d22
			ps35.OverlayValues[23] = d23
			ps35.OverlayValues[24] = d24
			ps35.OverlayValues[25] = d25
			ps35.OverlayValues[26] = d26
			ps35.OverlayValues[27] = d27
			ps35.OverlayValues[28] = d28
			ps35.OverlayValues[29] = d29
			ps35.OverlayValues[30] = d30
			ps35.OverlayValues[31] = d31
			ps35.OverlayValues[32] = d32
			ps35.OverlayValues[33] = d33
			ps35.OverlayValues[34] = d34
					return bbs[3].RenderPS(ps35)
				}
			ps36 := scm.PhiState{General: ps.General}
			ps36.OverlayValues = make([]scm.JITValueDesc, 35)
			ps36.OverlayValues[0] = d0
			ps36.OverlayValues[2] = d2
			ps36.OverlayValues[3] = d3
			ps36.OverlayValues[4] = d4
			ps36.OverlayValues[5] = d5
			ps36.OverlayValues[6] = d6
			ps36.OverlayValues[7] = d7
			ps36.OverlayValues[8] = d8
			ps36.OverlayValues[9] = d9
			ps36.OverlayValues[10] = d10
			ps36.OverlayValues[11] = d11
			ps36.OverlayValues[12] = d12
			ps36.OverlayValues[13] = d13
			ps36.OverlayValues[14] = d14
			ps36.OverlayValues[15] = d15
			ps36.OverlayValues[16] = d16
			ps36.OverlayValues[17] = d17
			ps36.OverlayValues[18] = d18
			ps36.OverlayValues[19] = d19
			ps36.OverlayValues[20] = d20
			ps36.OverlayValues[21] = d21
			ps36.OverlayValues[22] = d22
			ps36.OverlayValues[23] = d23
			ps36.OverlayValues[24] = d24
			ps36.OverlayValues[25] = d25
			ps36.OverlayValues[26] = d26
			ps36.OverlayValues[27] = d27
			ps36.OverlayValues[28] = d28
			ps36.OverlayValues[29] = d29
			ps36.OverlayValues[30] = d30
			ps36.OverlayValues[31] = d31
			ps36.OverlayValues[32] = d32
			ps36.OverlayValues[33] = d33
			ps36.OverlayValues[34] = d34
				return bbs[2].RenderPS(ps36)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d34.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl12)
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl12)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl3)
			ps37 := scm.PhiState{General: true}
			ps37.OverlayValues = make([]scm.JITValueDesc, 35)
			ps37.OverlayValues[0] = d0
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
			ps37.OverlayValues[15] = d15
			ps37.OverlayValues[16] = d16
			ps37.OverlayValues[17] = d17
			ps37.OverlayValues[18] = d18
			ps37.OverlayValues[19] = d19
			ps37.OverlayValues[20] = d20
			ps37.OverlayValues[21] = d21
			ps37.OverlayValues[22] = d22
			ps37.OverlayValues[23] = d23
			ps37.OverlayValues[24] = d24
			ps37.OverlayValues[25] = d25
			ps37.OverlayValues[26] = d26
			ps37.OverlayValues[27] = d27
			ps37.OverlayValues[28] = d28
			ps37.OverlayValues[29] = d29
			ps37.OverlayValues[30] = d30
			ps37.OverlayValues[31] = d31
			ps37.OverlayValues[32] = d32
			ps37.OverlayValues[33] = d33
			ps37.OverlayValues[34] = d34
			ps38 := scm.PhiState{General: true}
			ps38.OverlayValues = make([]scm.JITValueDesc, 35)
			ps38.OverlayValues[0] = d0
			ps38.OverlayValues[2] = d2
			ps38.OverlayValues[3] = d3
			ps38.OverlayValues[4] = d4
			ps38.OverlayValues[5] = d5
			ps38.OverlayValues[6] = d6
			ps38.OverlayValues[7] = d7
			ps38.OverlayValues[8] = d8
			ps38.OverlayValues[9] = d9
			ps38.OverlayValues[10] = d10
			ps38.OverlayValues[11] = d11
			ps38.OverlayValues[12] = d12
			ps38.OverlayValues[13] = d13
			ps38.OverlayValues[14] = d14
			ps38.OverlayValues[15] = d15
			ps38.OverlayValues[16] = d16
			ps38.OverlayValues[17] = d17
			ps38.OverlayValues[18] = d18
			ps38.OverlayValues[19] = d19
			ps38.OverlayValues[20] = d20
			ps38.OverlayValues[21] = d21
			ps38.OverlayValues[22] = d22
			ps38.OverlayValues[23] = d23
			ps38.OverlayValues[24] = d24
			ps38.OverlayValues[25] = d25
			ps38.OverlayValues[26] = d26
			ps38.OverlayValues[27] = d27
			ps38.OverlayValues[28] = d28
			ps38.OverlayValues[29] = d29
			ps38.OverlayValues[30] = d30
			ps38.OverlayValues[31] = d31
			ps38.OverlayValues[32] = d32
			ps38.OverlayValues[33] = d33
			ps38.OverlayValues[34] = d34
			snap39 := d0
			snap40 := d2
			snap41 := d3
			snap42 := d4
			snap43 := d5
			snap44 := d6
			snap45 := d7
			snap46 := d8
			snap47 := d9
			snap48 := d10
			snap49 := d11
			snap50 := d12
			snap51 := d13
			snap52 := d14
			snap53 := d15
			snap54 := d16
			snap55 := d17
			snap56 := d18
			snap57 := d19
			snap58 := d20
			snap59 := d21
			snap60 := d22
			snap61 := d23
			snap62 := d24
			snap63 := d25
			snap64 := d26
			snap65 := d27
			snap66 := d28
			snap67 := d29
			snap68 := d30
			snap69 := d31
			snap70 := d32
			snap71 := d33
			snap72 := d34
			alloc73 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps38)
			}
			ctx.RestoreAllocState(alloc73)
			d0 = snap39
			d2 = snap40
			d3 = snap41
			d4 = snap42
			d5 = snap43
			d6 = snap44
			d7 = snap45
			d8 = snap46
			d9 = snap47
			d10 = snap48
			d11 = snap49
			d12 = snap50
			d13 = snap51
			d14 = snap52
			d15 = snap53
			d16 = snap54
			d17 = snap55
			d18 = snap56
			d19 = snap57
			d20 = snap58
			d21 = snap59
			d22 = snap60
			d23 = snap61
			d24 = snap62
			d25 = snap63
			d26 = snap64
			d27 = snap65
			d28 = snap66
			d29 = snap67
			d30 = snap68
			d31 = snap69
			d32 = snap70
			d33 = snap71
			d34 = snap72
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps37)
			}
			return result
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
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
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
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
			ctx.ReclaimUntrackedRegs()
			d74 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d74)
			ctx.BindReg(r1, &d74)
			ctx.EmitMakeNil(d74)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
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
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
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
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d32)
			var d75 scm.JITValueDesc
			if d32.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d32.Imm.Int()))))}
			} else {
				r43 := ctx.AllocReg()
				ctx.EmitMovRegReg(r43, d32.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d75)
			}
			var d76 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
				r44 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r44, fieldAddr)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
				ctx.BindReg(r44, &d76)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
				r45 := ctx.AllocReg()
				ctx.EmitMovRegMem(r45, thisptr.Reg, off)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d76)
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d75)
			ctx.ProtectReg(d75.Reg)
			ctx.EnsureDesc(&d76)
			ctx.UnprotectReg(d75.Reg)
			var d77 scm.JITValueDesc
			if d75.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + d76.Imm.Int())}
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegReg(r46, d75.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d77)
			} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
				ctx.BindReg(d76.Reg, &d77)
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d75.Imm.Int()))
				ctx.EmitAddInt64(scratch, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else if d76.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.EmitMovRegReg(scratch, d75.Reg)
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d76.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else {
				r47 := ctx.AllocRegExcept(d75.Reg, d76.Reg)
				ctx.EmitMovRegReg(r47, d75.Reg)
				ctx.EmitAddInt64(r47, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d77)
			}
			if d77.Loc == scm.LocReg && d75.Loc == scm.LocReg && d77.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			var d78 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
				r48 := ctx.AllocReg()
				ctx.EmitMovRegMem8(r48, fieldAddr)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d78)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
				r49 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r49, thisptr.Reg, off)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d78)
			}
			ctx.EnsureDesc(&d78)
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d78.Imm.Int() > 0)}
			} else {
				r50 := ctx.AllocRegExcept(d78.Reg)
				ctx.EmitCmpRegImm32(d78.Reg, 0)
				ctx.EmitSetcc(r50, scm.CcG)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
				ctx.BindReg(r50, &d79)
			}
			d80 = d79
			ctx.EnsureDesc(&d80)
			if d80.Loc != scm.LocImm && d80.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
			ps81 := scm.PhiState{General: ps.General}
			ps81.OverlayValues = make([]scm.JITValueDesc, 81)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[2] = d2
			ps81.OverlayValues[3] = d3
			ps81.OverlayValues[4] = d4
			ps81.OverlayValues[5] = d5
			ps81.OverlayValues[6] = d6
			ps81.OverlayValues[7] = d7
			ps81.OverlayValues[8] = d8
			ps81.OverlayValues[9] = d9
			ps81.OverlayValues[10] = d10
			ps81.OverlayValues[11] = d11
			ps81.OverlayValues[12] = d12
			ps81.OverlayValues[13] = d13
			ps81.OverlayValues[14] = d14
			ps81.OverlayValues[15] = d15
			ps81.OverlayValues[16] = d16
			ps81.OverlayValues[17] = d17
			ps81.OverlayValues[18] = d18
			ps81.OverlayValues[19] = d19
			ps81.OverlayValues[20] = d20
			ps81.OverlayValues[21] = d21
			ps81.OverlayValues[22] = d22
			ps81.OverlayValues[23] = d23
			ps81.OverlayValues[24] = d24
			ps81.OverlayValues[25] = d25
			ps81.OverlayValues[26] = d26
			ps81.OverlayValues[27] = d27
			ps81.OverlayValues[28] = d28
			ps81.OverlayValues[29] = d29
			ps81.OverlayValues[30] = d30
			ps81.OverlayValues[31] = d31
			ps81.OverlayValues[32] = d32
			ps81.OverlayValues[33] = d33
			ps81.OverlayValues[34] = d34
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps81.OverlayValues[79] = d79
			ps81.OverlayValues[80] = d80
					return bbs[4].RenderPS(ps81)
				}
			ps82 := scm.PhiState{General: ps.General}
			ps82.OverlayValues = make([]scm.JITValueDesc, 81)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[2] = d2
			ps82.OverlayValues[3] = d3
			ps82.OverlayValues[4] = d4
			ps82.OverlayValues[5] = d5
			ps82.OverlayValues[6] = d6
			ps82.OverlayValues[7] = d7
			ps82.OverlayValues[8] = d8
			ps82.OverlayValues[9] = d9
			ps82.OverlayValues[10] = d10
			ps82.OverlayValues[11] = d11
			ps82.OverlayValues[12] = d12
			ps82.OverlayValues[13] = d13
			ps82.OverlayValues[14] = d14
			ps82.OverlayValues[15] = d15
			ps82.OverlayValues[16] = d16
			ps82.OverlayValues[17] = d17
			ps82.OverlayValues[18] = d18
			ps82.OverlayValues[19] = d19
			ps82.OverlayValues[20] = d20
			ps82.OverlayValues[21] = d21
			ps82.OverlayValues[22] = d22
			ps82.OverlayValues[23] = d23
			ps82.OverlayValues[24] = d24
			ps82.OverlayValues[25] = d25
			ps82.OverlayValues[26] = d26
			ps82.OverlayValues[27] = d27
			ps82.OverlayValues[28] = d28
			ps82.OverlayValues[29] = d29
			ps82.OverlayValues[30] = d30
			ps82.OverlayValues[31] = d31
			ps82.OverlayValues[32] = d32
			ps82.OverlayValues[33] = d33
			ps82.OverlayValues[34] = d34
			ps82.OverlayValues[74] = d74
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			ps82.OverlayValues[79] = d79
			ps82.OverlayValues[80] = d80
				return bbs[5].RenderPS(ps82)
			}
			if !ps.General {
				ps.General = true
				return bbs[2].RenderPS(ps)
			}
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d80.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl14)
			ctx.EmitJmp(lbl15)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl6)
			ps83 := scm.PhiState{General: true}
			ps83.OverlayValues = make([]scm.JITValueDesc, 81)
			ps83.OverlayValues[0] = d0
			ps83.OverlayValues[2] = d2
			ps83.OverlayValues[3] = d3
			ps83.OverlayValues[4] = d4
			ps83.OverlayValues[5] = d5
			ps83.OverlayValues[6] = d6
			ps83.OverlayValues[7] = d7
			ps83.OverlayValues[8] = d8
			ps83.OverlayValues[9] = d9
			ps83.OverlayValues[10] = d10
			ps83.OverlayValues[11] = d11
			ps83.OverlayValues[12] = d12
			ps83.OverlayValues[13] = d13
			ps83.OverlayValues[14] = d14
			ps83.OverlayValues[15] = d15
			ps83.OverlayValues[16] = d16
			ps83.OverlayValues[17] = d17
			ps83.OverlayValues[18] = d18
			ps83.OverlayValues[19] = d19
			ps83.OverlayValues[20] = d20
			ps83.OverlayValues[21] = d21
			ps83.OverlayValues[22] = d22
			ps83.OverlayValues[23] = d23
			ps83.OverlayValues[24] = d24
			ps83.OverlayValues[25] = d25
			ps83.OverlayValues[26] = d26
			ps83.OverlayValues[27] = d27
			ps83.OverlayValues[28] = d28
			ps83.OverlayValues[29] = d29
			ps83.OverlayValues[30] = d30
			ps83.OverlayValues[31] = d31
			ps83.OverlayValues[32] = d32
			ps83.OverlayValues[33] = d33
			ps83.OverlayValues[34] = d34
			ps83.OverlayValues[74] = d74
			ps83.OverlayValues[75] = d75
			ps83.OverlayValues[76] = d76
			ps83.OverlayValues[77] = d77
			ps83.OverlayValues[78] = d78
			ps83.OverlayValues[79] = d79
			ps83.OverlayValues[80] = d80
			ps84 := scm.PhiState{General: true}
			ps84.OverlayValues = make([]scm.JITValueDesc, 81)
			ps84.OverlayValues[0] = d0
			ps84.OverlayValues[2] = d2
			ps84.OverlayValues[3] = d3
			ps84.OverlayValues[4] = d4
			ps84.OverlayValues[5] = d5
			ps84.OverlayValues[6] = d6
			ps84.OverlayValues[7] = d7
			ps84.OverlayValues[8] = d8
			ps84.OverlayValues[9] = d9
			ps84.OverlayValues[10] = d10
			ps84.OverlayValues[11] = d11
			ps84.OverlayValues[12] = d12
			ps84.OverlayValues[13] = d13
			ps84.OverlayValues[14] = d14
			ps84.OverlayValues[15] = d15
			ps84.OverlayValues[16] = d16
			ps84.OverlayValues[17] = d17
			ps84.OverlayValues[18] = d18
			ps84.OverlayValues[19] = d19
			ps84.OverlayValues[20] = d20
			ps84.OverlayValues[21] = d21
			ps84.OverlayValues[22] = d22
			ps84.OverlayValues[23] = d23
			ps84.OverlayValues[24] = d24
			ps84.OverlayValues[25] = d25
			ps84.OverlayValues[26] = d26
			ps84.OverlayValues[27] = d27
			ps84.OverlayValues[28] = d28
			ps84.OverlayValues[29] = d29
			ps84.OverlayValues[30] = d30
			ps84.OverlayValues[31] = d31
			ps84.OverlayValues[32] = d32
			ps84.OverlayValues[33] = d33
			ps84.OverlayValues[34] = d34
			ps84.OverlayValues[74] = d74
			ps84.OverlayValues[75] = d75
			ps84.OverlayValues[76] = d76
			ps84.OverlayValues[77] = d77
			ps84.OverlayValues[78] = d78
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			snap85 := d0
			snap86 := d2
			snap87 := d3
			snap88 := d4
			snap89 := d5
			snap90 := d6
			snap91 := d7
			snap92 := d8
			snap93 := d9
			snap94 := d10
			snap95 := d11
			snap96 := d12
			snap97 := d13
			snap98 := d14
			snap99 := d15
			snap100 := d16
			snap101 := d17
			snap102 := d18
			snap103 := d19
			snap104 := d20
			snap105 := d21
			snap106 := d22
			snap107 := d23
			snap108 := d24
			snap109 := d25
			snap110 := d26
			snap111 := d27
			snap112 := d28
			snap113 := d29
			snap114 := d30
			snap115 := d31
			snap116 := d32
			snap117 := d33
			snap118 := d34
			snap119 := d74
			snap120 := d75
			snap121 := d76
			snap122 := d77
			snap123 := d78
			snap124 := d79
			snap125 := d80
			alloc126 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps84)
			}
			ctx.RestoreAllocState(alloc126)
			d0 = snap85
			d2 = snap86
			d3 = snap87
			d4 = snap88
			d5 = snap89
			d6 = snap90
			d7 = snap91
			d8 = snap92
			d9 = snap93
			d10 = snap94
			d11 = snap95
			d12 = snap96
			d13 = snap97
			d14 = snap98
			d15 = snap99
			d16 = snap100
			d17 = snap101
			d18 = snap102
			d19 = snap103
			d20 = snap104
			d21 = snap105
			d22 = snap106
			d23 = snap107
			d24 = snap108
			d25 = snap109
			d26 = snap110
			d27 = snap111
			d28 = snap112
			d29 = snap113
			d30 = snap114
			d31 = snap115
			d32 = snap116
			d33 = snap117
			d34 = snap118
			d74 = snap119
			d75 = snap120
			d76 = snap121
			d77 = snap122
			d78 = snap123
			d79 = snap124
			d80 = snap125
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps83)
			}
			return result
			ctx.FreeDesc(&d79)
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
				bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
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
			ctx.ReclaimUntrackedRegs()
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
				r51 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r51, fieldAddr)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d127)
			} else {
				off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
				r52 := ctx.AllocReg()
				ctx.EmitMovRegMem(r52, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
				ctx.BindReg(r52, &d127)
			}
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d32.Imm.Int()) == uint64(d127.Imm.Int()))}
			} else if d127.Loc == scm.LocImm {
				r53 := ctx.AllocRegExcept(d32.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d32.Reg, int32(d127.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.EmitCmpInt64(d32.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r53, scm.CcE)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d128)
			} else if d32.Loc == scm.LocImm {
				r54 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d127.Reg)
				ctx.EmitSetcc(r54, scm.CcE)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d128)
			} else {
				r55 := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitCmpInt64(d32.Reg, d127.Reg)
				ctx.EmitSetcc(r55, scm.CcE)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d128)
			}
			ctx.FreeDesc(&d32)
			d129 = d128
			ctx.EnsureDesc(&d129)
			if d129.Loc != scm.LocImm && d129.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d129.Loc == scm.LocImm {
				if d129.Imm.Bool() {
			ps130 := scm.PhiState{General: ps.General}
			ps130.OverlayValues = make([]scm.JITValueDesc, 130)
			ps130.OverlayValues[0] = d0
			ps130.OverlayValues[2] = d2
			ps130.OverlayValues[3] = d3
			ps130.OverlayValues[4] = d4
			ps130.OverlayValues[5] = d5
			ps130.OverlayValues[6] = d6
			ps130.OverlayValues[7] = d7
			ps130.OverlayValues[8] = d8
			ps130.OverlayValues[9] = d9
			ps130.OverlayValues[10] = d10
			ps130.OverlayValues[11] = d11
			ps130.OverlayValues[12] = d12
			ps130.OverlayValues[13] = d13
			ps130.OverlayValues[14] = d14
			ps130.OverlayValues[15] = d15
			ps130.OverlayValues[16] = d16
			ps130.OverlayValues[17] = d17
			ps130.OverlayValues[18] = d18
			ps130.OverlayValues[19] = d19
			ps130.OverlayValues[20] = d20
			ps130.OverlayValues[21] = d21
			ps130.OverlayValues[22] = d22
			ps130.OverlayValues[23] = d23
			ps130.OverlayValues[24] = d24
			ps130.OverlayValues[25] = d25
			ps130.OverlayValues[26] = d26
			ps130.OverlayValues[27] = d27
			ps130.OverlayValues[28] = d28
			ps130.OverlayValues[29] = d29
			ps130.OverlayValues[30] = d30
			ps130.OverlayValues[31] = d31
			ps130.OverlayValues[32] = d32
			ps130.OverlayValues[33] = d33
			ps130.OverlayValues[34] = d34
			ps130.OverlayValues[74] = d74
			ps130.OverlayValues[75] = d75
			ps130.OverlayValues[76] = d76
			ps130.OverlayValues[77] = d77
			ps130.OverlayValues[78] = d78
			ps130.OverlayValues[79] = d79
			ps130.OverlayValues[80] = d80
			ps130.OverlayValues[127] = d127
			ps130.OverlayValues[128] = d128
			ps130.OverlayValues[129] = d129
					return bbs[1].RenderPS(ps130)
				}
			ps131 := scm.PhiState{General: ps.General}
			ps131.OverlayValues = make([]scm.JITValueDesc, 130)
			ps131.OverlayValues[0] = d0
			ps131.OverlayValues[2] = d2
			ps131.OverlayValues[3] = d3
			ps131.OverlayValues[4] = d4
			ps131.OverlayValues[5] = d5
			ps131.OverlayValues[6] = d6
			ps131.OverlayValues[7] = d7
			ps131.OverlayValues[8] = d8
			ps131.OverlayValues[9] = d9
			ps131.OverlayValues[10] = d10
			ps131.OverlayValues[11] = d11
			ps131.OverlayValues[12] = d12
			ps131.OverlayValues[13] = d13
			ps131.OverlayValues[14] = d14
			ps131.OverlayValues[15] = d15
			ps131.OverlayValues[16] = d16
			ps131.OverlayValues[17] = d17
			ps131.OverlayValues[18] = d18
			ps131.OverlayValues[19] = d19
			ps131.OverlayValues[20] = d20
			ps131.OverlayValues[21] = d21
			ps131.OverlayValues[22] = d22
			ps131.OverlayValues[23] = d23
			ps131.OverlayValues[24] = d24
			ps131.OverlayValues[25] = d25
			ps131.OverlayValues[26] = d26
			ps131.OverlayValues[27] = d27
			ps131.OverlayValues[28] = d28
			ps131.OverlayValues[29] = d29
			ps131.OverlayValues[30] = d30
			ps131.OverlayValues[31] = d31
			ps131.OverlayValues[32] = d32
			ps131.OverlayValues[33] = d33
			ps131.OverlayValues[34] = d34
			ps131.OverlayValues[74] = d74
			ps131.OverlayValues[75] = d75
			ps131.OverlayValues[76] = d76
			ps131.OverlayValues[77] = d77
			ps131.OverlayValues[78] = d78
			ps131.OverlayValues[79] = d79
			ps131.OverlayValues[80] = d80
			ps131.OverlayValues[127] = d127
			ps131.OverlayValues[128] = d128
			ps131.OverlayValues[129] = d129
				return bbs[2].RenderPS(ps131)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d129.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl16)
			ctx.EmitJmp(lbl17)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl17)
			ctx.EmitJmp(lbl3)
			ps132 := scm.PhiState{General: true}
			ps132.OverlayValues = make([]scm.JITValueDesc, 130)
			ps132.OverlayValues[0] = d0
			ps132.OverlayValues[2] = d2
			ps132.OverlayValues[3] = d3
			ps132.OverlayValues[4] = d4
			ps132.OverlayValues[5] = d5
			ps132.OverlayValues[6] = d6
			ps132.OverlayValues[7] = d7
			ps132.OverlayValues[8] = d8
			ps132.OverlayValues[9] = d9
			ps132.OverlayValues[10] = d10
			ps132.OverlayValues[11] = d11
			ps132.OverlayValues[12] = d12
			ps132.OverlayValues[13] = d13
			ps132.OverlayValues[14] = d14
			ps132.OverlayValues[15] = d15
			ps132.OverlayValues[16] = d16
			ps132.OverlayValues[17] = d17
			ps132.OverlayValues[18] = d18
			ps132.OverlayValues[19] = d19
			ps132.OverlayValues[20] = d20
			ps132.OverlayValues[21] = d21
			ps132.OverlayValues[22] = d22
			ps132.OverlayValues[23] = d23
			ps132.OverlayValues[24] = d24
			ps132.OverlayValues[25] = d25
			ps132.OverlayValues[26] = d26
			ps132.OverlayValues[27] = d27
			ps132.OverlayValues[28] = d28
			ps132.OverlayValues[29] = d29
			ps132.OverlayValues[30] = d30
			ps132.OverlayValues[31] = d31
			ps132.OverlayValues[32] = d32
			ps132.OverlayValues[33] = d33
			ps132.OverlayValues[34] = d34
			ps132.OverlayValues[74] = d74
			ps132.OverlayValues[75] = d75
			ps132.OverlayValues[76] = d76
			ps132.OverlayValues[77] = d77
			ps132.OverlayValues[78] = d78
			ps132.OverlayValues[79] = d79
			ps132.OverlayValues[80] = d80
			ps132.OverlayValues[127] = d127
			ps132.OverlayValues[128] = d128
			ps132.OverlayValues[129] = d129
			ps133 := scm.PhiState{General: true}
			ps133.OverlayValues = make([]scm.JITValueDesc, 130)
			ps133.OverlayValues[0] = d0
			ps133.OverlayValues[2] = d2
			ps133.OverlayValues[3] = d3
			ps133.OverlayValues[4] = d4
			ps133.OverlayValues[5] = d5
			ps133.OverlayValues[6] = d6
			ps133.OverlayValues[7] = d7
			ps133.OverlayValues[8] = d8
			ps133.OverlayValues[9] = d9
			ps133.OverlayValues[10] = d10
			ps133.OverlayValues[11] = d11
			ps133.OverlayValues[12] = d12
			ps133.OverlayValues[13] = d13
			ps133.OverlayValues[14] = d14
			ps133.OverlayValues[15] = d15
			ps133.OverlayValues[16] = d16
			ps133.OverlayValues[17] = d17
			ps133.OverlayValues[18] = d18
			ps133.OverlayValues[19] = d19
			ps133.OverlayValues[20] = d20
			ps133.OverlayValues[21] = d21
			ps133.OverlayValues[22] = d22
			ps133.OverlayValues[23] = d23
			ps133.OverlayValues[24] = d24
			ps133.OverlayValues[25] = d25
			ps133.OverlayValues[26] = d26
			ps133.OverlayValues[27] = d27
			ps133.OverlayValues[28] = d28
			ps133.OverlayValues[29] = d29
			ps133.OverlayValues[30] = d30
			ps133.OverlayValues[31] = d31
			ps133.OverlayValues[32] = d32
			ps133.OverlayValues[33] = d33
			ps133.OverlayValues[34] = d34
			ps133.OverlayValues[74] = d74
			ps133.OverlayValues[75] = d75
			ps133.OverlayValues[76] = d76
			ps133.OverlayValues[77] = d77
			ps133.OverlayValues[78] = d78
			ps133.OverlayValues[79] = d79
			ps133.OverlayValues[80] = d80
			ps133.OverlayValues[127] = d127
			ps133.OverlayValues[128] = d128
			ps133.OverlayValues[129] = d129
			snap134 := d0
			snap135 := d2
			snap136 := d3
			snap137 := d4
			snap138 := d5
			snap139 := d6
			snap140 := d7
			snap141 := d8
			snap142 := d9
			snap143 := d10
			snap144 := d11
			snap145 := d12
			snap146 := d13
			snap147 := d14
			snap148 := d15
			snap149 := d16
			snap150 := d17
			snap151 := d18
			snap152 := d19
			snap153 := d20
			snap154 := d21
			snap155 := d22
			snap156 := d23
			snap157 := d24
			snap158 := d25
			snap159 := d26
			snap160 := d27
			snap161 := d28
			snap162 := d29
			snap163 := d30
			snap164 := d31
			snap165 := d32
			snap166 := d33
			snap167 := d34
			snap168 := d74
			snap169 := d75
			snap170 := d76
			snap171 := d77
			snap172 := d78
			snap173 := d79
			snap174 := d80
			snap175 := d127
			snap176 := d128
			snap177 := d129
			alloc178 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps133)
			}
			ctx.RestoreAllocState(alloc178)
			d0 = snap134
			d2 = snap135
			d3 = snap136
			d4 = snap137
			d5 = snap138
			d6 = snap139
			d7 = snap140
			d8 = snap141
			d9 = snap142
			d10 = snap143
			d11 = snap144
			d12 = snap145
			d13 = snap146
			d14 = snap147
			d15 = snap148
			d16 = snap149
			d17 = snap150
			d18 = snap151
			d19 = snap152
			d20 = snap153
			d21 = snap154
			d22 = snap155
			d23 = snap156
			d24 = snap157
			d25 = snap158
			d26 = snap159
			d27 = snap160
			d28 = snap161
			d29 = snap162
			d30 = snap163
			d31 = snap164
			d32 = snap165
			d33 = snap166
			d34 = snap167
			d74 = snap168
			d75 = snap169
			d76 = snap170
			d77 = snap171
			d78 = snap172
			d79 = snap173
			d80 = snap174
			d127 = snap175
			d128 = snap176
			d129 = snap177
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps132)
			}
			return result
			ctx.FreeDesc(&d128)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
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
				bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d78)
			r56 := ctx.AllocReg()
			ctx.EmitMovRegImm64(r56, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
			r57 := ctx.AllocReg()
			if d78.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r57, uint64(d78.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r57, d78.Reg)
				ctx.EmitShlRegImm8(r57, 3)
			}
			ctx.EmitAddInt64(r56, r57)
			ctx.FreeReg(r57)
			r58 := ctx.AllocRegExcept(r56)
			ctx.EmitMovRegMem(r58, r56, 0)
			ctx.FreeReg(r56)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
			ctx.BindReg(r58, &d179)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d77)
			ctx.ProtectReg(d77.Reg)
			ctx.EnsureDesc(&d179)
			ctx.UnprotectReg(d77.Reg)
			var d180 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() * d179.Imm.Int())}
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d179.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d77.Imm.Int()))
				ctx.EmitImulInt64(scratch, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d180)
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.EmitMovRegReg(scratch, d77.Reg)
				if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d179.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d180)
			} else {
				r59 := ctx.AllocRegExcept(d77.Reg, d179.Reg)
				ctx.EmitMovRegReg(r59, d77.Reg)
				ctx.EmitImulInt64(r59, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d180)
			}
			if d180.Loc == scm.LocReg && d77.Loc == scm.LocReg && d180.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.EnsureDesc(&d180)
			d181 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d181)
			ctx.BindReg(r1, &d181)
			ctx.EnsureDesc(&d180)
			ctx.EmitMakeInt(d181, d180)
			if d180.Loc == scm.LocReg { ctx.FreeReg(d180.Reg) }
			ctx.EmitJmp(lbl0)
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
				bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
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
			if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
				d127 = ps.OverlayValues[127]
			}
			if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
				d128 = ps.OverlayValues[128]
			}
			if len(ps.OverlayValues) > 129 && ps.OverlayValues[129].Loc != scm.LocNone {
				d129 = ps.OverlayValues[129]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d77)
			var d182 scm.JITValueDesc
			if d77.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d77.Imm.Int()))}
			} else {
				ctx.EmitCvtInt64ToFloat64(scm.RegX0, d77.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d77.Reg}
				ctx.BindReg(d77.Reg, &d182)
			}
			ctx.FreeDesc(&d77)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d78)
			var d183 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d78.Imm.Int()))))}
			} else {
				r60 := ctx.AllocReg()
				ctx.EmitMovRegReg(r60, d78.Reg)
				ctx.EmitShlRegImm8(r60, 56)
				ctx.EmitSarRegImm8(r60, 56)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d183)
			}
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() + 15)}
			} else {
				scratch := ctx.AllocRegExcept(d183.Reg)
				ctx.EmitMovRegReg(scratch, d183.Reg)
				ctx.EmitAddRegImm32(scratch, int32(15))
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d184)
			}
			if d184.Loc == scm.LocReg && d183.Loc == scm.LocReg && d184.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d183)
			ctx.EnsureDesc(&d184)
			r61 := ctx.AllocReg()
			ctx.EmitMovRegImm64(r61, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
			r62 := ctx.AllocReg()
			if d184.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r62, uint64(d184.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r62, d184.Reg)
				ctx.EmitShlRegImm8(r62, 3)
			}
			ctx.EmitAddInt64(r61, r62)
			ctx.FreeReg(r62)
			r63 := ctx.AllocRegExcept(r61)
			ctx.EmitMovRegMem(r63, r61, 0)
			ctx.FreeReg(r61)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			ctx.BindReg(r63, &d185)
			ctx.FreeDesc(&d184)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d185)
			var d186 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d182.Imm.Float() * d185.Imm.Float())}
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d185.Reg)
				_, xBits := d182.Imm.RawWords()
				ctx.EmitMovRegImm64(scratch, xBits)
				ctx.EmitMulFloat64(scratch, d185.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d186)
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.EmitMovRegReg(scratch, d182.Reg)
				_, yBits := d185.Imm.RawWords()
				ctx.EmitMovRegImm64(scm.RegR11, yBits)
				ctx.EmitMulFloat64(scratch, scm.RegR11)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
				ctx.BindReg(scratch, &d186)
			} else {
				r64 := ctx.AllocRegExcept(d182.Reg, d185.Reg)
				ctx.EmitMovRegReg(r64, d182.Reg)
				ctx.EmitMulFloat64(r64, d185.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r64}
				ctx.BindReg(r64, &d186)
			}
			if d186.Loc == scm.LocReg && d182.Loc == scm.LocReg && d186.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.FreeDesc(&d185)
			ctx.EnsureDesc(&d186)
			d187 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d187)
			ctx.BindReg(r1, &d187)
			ctx.EnsureDesc(&d186)
			ctx.EmitMakeFloat(d187, d186)
			if d186.Loc == scm.LocReg { ctx.FreeReg(d186.Reg) }
			ctx.EmitJmp(lbl0)
			return result
			}
			ps188 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps188)
			ctx.MarkLabel(lbl0)
			d189 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d189)
			ctx.BindReg(r1, &d189)
			ctx.EmitMovPairToResult(&d189, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			return result
}

func (s *StorageDecimal) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(13))
	binary.Write(f, binary.LittleEndian, s.scaleExp)
	s.inner.Serialize(f) // writes magic 10 + data
}

func (s *StorageDecimal) Deserialize(f io.Reader) uint {
	// No version byte: the first byte is scaleExp (int8).
	// Format changes require a new magic byte.
	binary.Read(f, binary.LittleEndian, &s.scaleExp)
	return s.inner.DeserializeEx(f, true) // reads magic 10 + data
}

func (s *StorageDecimal) DistinctCount() uint { return s.inner.DistinctCount() }
