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

// GetValueRange and GetValueMulti delegate the raw (offset-applied,
// null-checked) integer decode to the wrapped StorageInt's own bulk method
// — which is where the bit-unpacking cursor optimization lives — and then
// rescale each non-nil result in place, avoiding a second per-element
// GetValue dispatch.
func (s *StorageDecimal) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.inner.GetValueRange(recid, count, target, stride)
	s.rescaleInPlace(target, count, stride)
}

func (s *StorageDecimal) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.inner.GetValueMulti(recids, target, stride)
	s.rescaleInPlace(target, uint32(len(recids)), stride)
}

func (s *StorageDecimal) rescaleInPlace(target []scm.Scmer, count uint32, stride int) {
	idx := 0
	for k := uint32(0); k < count; k++ {
		v := target[idx]
		if !v.IsNil() {
			raw := v.Int()
			if s.scaleExp > 0 {
				target[idx] = scm.NewInt(raw * pow10i[s.scaleExp])
			} else {
				target[idx] = scm.NewFloat(float64(raw) * pow10f[int(s.scaleExp)+15])
			}
		}
		idx += stride
	}
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
	var phiBase1 int32
	_ = phiBase1
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
	var d126 scm.JITValueDesc
	_ = d126
	var d127 scm.JITValueDesc
	_ = d127
	var d128 scm.JITValueDesc
	_ = d128
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
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
	}
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
		ctx.StabilizeDescForControlFlow(&d0)
		phiBase1 = ctx.AllocStack(int32(16))
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(phiBase1) + int32(0)}
		_ = d2
		lbl7 := ctx.ReserveLabel()
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		bbpos_1_1 := int32(-1)
		_ = bbpos_1_1
		bbpos_1_2 := int32(-1)
		_ = bbpos_1_2
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		var d3 scm.JITValueDesc
		if d0.Loc == scm.LocImm {
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d0.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d3)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d4 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r3, fieldAddr)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d4)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r4, thisptr.Reg, off)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d4)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d5 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
		} else {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegReg(r5, d4.Reg)
			ctx.EmitShlRegImm8(r5, 56)
			ctx.EmitShrRegImm8(r5, 56)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d5)
		}
		ctx.ReclaimUntrackedRegs()
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
			r6 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
			ctx.EmitMovRegReg(r6, d3.Reg)
			ctx.EmitImulInt64(r6, d5.Reg)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d6)
		}
		if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
			ctx.TransferReg(d3.Reg)
			d3.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d6)
		ctx.FreeDesc(&d3)
		ctx.FreeDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d7 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
			r7 := ctx.AllocReg()
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			ctx.EmitMovRegMem64(r7, fieldAddr)
			ctx.EmitMovRegMem64(r8, fieldAddr+8)
			ctx.EmitMovRegMem64(r9, fieldAddr+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r7, Reg2: r8, Reg3: r9}
			ctx.BindReg(r7, &d7)
			ctx.BindReg(r8, &d7)
			ctx.BindReg(r9, &d7)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
			r10 := ctx.AllocReg()
			r11 := ctx.AllocRegExcept(r10)
			r12 := ctx.AllocRegExcept(r10, r11)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off)
			ctx.EmitMovRegMem(r11, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r10, Reg2: r11, Reg3: r12}
			ctx.BindReg(r10, &d7)
			ctx.BindReg(r11, &d7)
			ctx.BindReg(r12, &d7)
		}
		ctx.ReclaimUntrackedRegs()
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.ReclaimUntrackedRegs()
		d10 = ctx.EmitSliceElementAddress(&d7, &d8, 8)
		ctx.EnsureDesc(&d10)
		ctx.EmitMovRegMem(d10.Reg, d10.Reg, 0)
		d9 = d10
		ctx.FreeDesc(&d8)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		var d11 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
		} else {
			r14 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r14, d6.Reg)
			ctx.EmitAndRegImm32(r14, 63)
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d11)
		}
		if d11.Loc == scm.LocReg && d6.Loc == scm.LocReg && d11.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d11)
		var d12 scm.JITValueDesc
		if d9.Loc == scm.LocImm && d11.Loc == scm.LocImm {
			d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) << uint64(d11.Imm.Int())))}
		} else if d11.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r15, d9.Reg)
			ctx.EmitShlRegImm8(r15, uint8(d11.Imm.Int()))
			d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d12)
		} else {
			{
				shiftSrc := d9.Reg
				r16 := ctx.AllocRegExcept(d9.Reg)
				ctx.EmitMovRegReg(r16, d9.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d11.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d11.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d11.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d12)
			}
		}
		if d12.Loc == scm.LocReg && d9.Loc == scm.LocReg && d12.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d12)
		ctx.EmitStoreToStack(d12, int32(phiBase1)+int32(0))
		ctx.StabilizeDescForControlFlow(&d12)
		ctx.FreeDesc(&d9)
		ctx.FreeDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		var d13 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
		} else {
			r17 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r17, d6.Reg)
			ctx.EmitAndRegImm32(r17, 63)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d13)
		}
		if d13.Loc == scm.LocReg && d6.Loc == scm.LocReg && d13.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d14 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
		} else {
			r18 := ctx.AllocReg()
			ctx.EmitMovRegReg(r18, d4.Reg)
			ctx.EmitShlRegImm8(r18, 56)
			ctx.EmitShrRegImm8(r18, 56)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d14)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d13)
		ctx.EnsureDesc(&d14)
		ctx.EnsureDesc(&d13)
		ctx.ProtectReg(d13.Reg)
		ctx.EnsureDesc(&d14)
		ctx.UnprotectReg(d13.Reg)
		var d15 scm.JITValueDesc
		if d13.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() + d14.Imm.Int())}
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r19 := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(r19, d13.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d15)
		} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			ctx.BindReg(d14.Reg, &d15)
		} else if d13.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
			ctx.EmitAddInt64(scratch, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d13.Reg)
			ctx.EmitMovRegReg(scratch, d13.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else {
			r20 := ctx.AllocRegExcept(d13.Reg, d14.Reg)
			ctx.EmitMovRegReg(r20, d13.Reg)
			ctx.EmitAddInt64(r20, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d15)
		}
		if d15.Loc == scm.LocReg && d13.Loc == scm.LocReg && d15.Reg == d13.Reg {
			ctx.TransferReg(d13.Reg)
			d13.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d13)
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d15)
		var d16 scm.JITValueDesc
		if d15.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d15.Imm.Int()) > uint64(0x40))}
		} else {
			r21 := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitCmpRegImm32(d15.Reg, 64)
			ctx.EmitSetcc(r21, scm.CondUnsignedAbove)
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r21}
			ctx.BindReg(r21, &d16)
		}
		ctx.FreeDesc(&d15)
		ctx.ReclaimUntrackedRegs()
		d17 = d16
		ctx.EnsureDesc(&d17)
		if d17.Loc != scm.LocImm && d17.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		lbl8 := ctx.ReserveLabel()
		lbl9 := ctx.ReserveLabel()
		lbl10 := ctx.ReserveLabel()
		lbl11 := ctx.ReserveLabel()
		if d17.Loc == scm.LocImm {
			if d17.Imm.Bool() {
				ctx.MarkLabel(lbl10)
				ctx.EmitJmp(lbl8)
			} else {
				ctx.MarkLabel(lbl11)
				ctx.EmitJmp(lbl9)
			}
		} else {
			ctx.EmitCmpRegImm32(d17.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl10)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl11)
			ctx.EmitJmp(lbl9)
		}
		ctx.FreeDesc(&d16)
		bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		ctx.EnsureDesc(&d4)
		var d18 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d4.Reg)
			ctx.EmitShlRegImm8(r22, 56)
			ctx.EmitShrRegImm8(r22, 56)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d18)
		}
		ctx.ReclaimUntrackedRegs()
		d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d19)
		ctx.ProtectReg(d19.Reg)
		ctx.EnsureDesc(&d18)
		ctx.UnprotectReg(d19.Reg)
		var d20 scm.JITValueDesc
		if d19.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() - d18.Imm.Int())}
		} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
			r23 := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(r23, d19.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d20)
		} else if d19.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
			ctx.EmitSubInt64(scratch, d18.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else if d18.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(scratch, d19.Reg)
			if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d18.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else {
			r24 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
			ctx.EmitMovRegReg(r24, d19.Reg)
			ctx.EmitSubInt64(r24, d18.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d20)
		}
		if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
			ctx.TransferReg(d19.Reg)
			d19.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d20)
		var d21 scm.JITValueDesc
		if d2.Loc == scm.LocImm && d20.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) >> uint64(d20.Imm.Int())))}
		} else if d20.Loc == scm.LocImm {
			r25 := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegReg(r25, d2.Reg)
			ctx.EmitShrRegImm8(r25, uint8(d20.Imm.Int()))
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d21)
		} else {
			{
				shiftSrc := d2.Reg
				r26 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(r26, d2.Reg)
				shiftSrc = r26
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d20.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d20.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d20.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d21)
			}
		}
		if d21.Loc == scm.LocReg && d2.Loc == scm.LocReg && d21.Reg == d2.Reg {
			ctx.TransferReg(d2.Reg)
			d2.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d2)
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		r27 := ctx.AllocReg()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d21)
		if d21.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r27, d21)
		}
		ctx.EmitJmp(lbl7)
		bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl8)
		ctx.ResolveFixups()
		d2 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		var d22 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
		} else {
			r28 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r28, d6.Reg)
			ctx.EmitShrRegImm8(r28, 6)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d22)
		}
		if d22.Loc == scm.LocReg && d6.Loc == scm.LocReg && d22.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d23 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitMovRegReg(scratch, d22.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d23)
		}
		if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
			ctx.TransferReg(d22.Reg)
			d22.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d22)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		d25 = ctx.EmitSliceElementAddress(&d7, &d23, 8)
		ctx.EnsureDesc(&d25)
		ctx.EmitMovRegMem(d25.Reg, d25.Reg, 0)
		d24 = d25
		ctx.FreeDesc(&d23)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		var d26 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
		} else {
			r29 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r29, d6.Reg)
			ctx.EmitAndRegImm32(r29, 63)
			d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d26)
		}
		if d26.Loc == scm.LocReg && d6.Loc == scm.LocReg && d26.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
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
			r30 := ctx.AllocRegExcept(d27.Reg)
			ctx.EmitMovRegReg(r30, d27.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d28)
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
			r31 := ctx.AllocRegExcept(d27.Reg, d26.Reg)
			ctx.EmitMovRegReg(r31, d27.Reg)
			ctx.EmitSubInt64(r31, d26.Reg)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d28)
		}
		if d28.Loc == scm.LocReg && d27.Loc == scm.LocReg && d28.Reg == d27.Reg {
			ctx.TransferReg(d27.Reg)
			d27.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d26)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d28)
		var d29 scm.JITValueDesc
		if d24.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d28.Imm.Int())))}
		} else if d28.Loc == scm.LocImm {
			r32 := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(r32, d24.Reg)
			ctx.EmitShrRegImm8(r32, uint8(d28.Imm.Int()))
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d29)
		} else {
			{
				shiftSrc := d24.Reg
				r33 := ctx.AllocRegExcept(d24.Reg)
				ctx.EmitMovRegReg(r33, d24.Reg)
				shiftSrc = r33
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
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
		if d29.Loc == scm.LocReg && d24.Loc == scm.LocReg && d29.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d29)
		var d30 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d29.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() | d29.Imm.Int())}
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
			ctx.BindReg(d29.Reg, &d30)
		} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r34, d12.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d30)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitOrInt64(scratch, d29.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else if d29.Loc == scm.LocImm {
			r35 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r35, d12.Reg)
			if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r35, int32(d29.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
				ctx.EmitOrInt64(r35, scm.RegR11)
			}
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d30)
		} else {
			r36 := ctx.AllocRegExcept(d12.Reg, d29.Reg)
			ctx.EmitMovRegReg(r36, d12.Reg)
			ctx.EmitOrInt64(r36, d29.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d30)
		}
		if d30.Loc == scm.LocReg && d12.Loc == scm.LocReg && d30.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d30)
		ctx.EmitStoreToStack(d30, int32(phiBase1)+int32(0))
		ctx.StabilizeDescForControlFlow(&d30)
		ctx.FreeDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl7)
		d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
		ctx.BindReg(r27, &d31)
		ctx.BindReg(r27, &d31)
		ctx.StabilizeDescForControlFlow(&d31)
		ctx.FreeDesc(&idxInt)
		var d32 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r37, fieldAddr)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d32)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r38 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r38, thisptr.Reg, off)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d32)
		}
		d33 = d32
		ctx.EnsureDesc(&d33)
		if d33.Loc != scm.LocImm && d33.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d33.Loc == scm.LocImm {
			if d33.Imm.Bool() {
				if ps.General {
				}
				ps34 := scm.PhiState{General: ps.General}
				ps34.OverlayValues = make([]scm.JITValueDesc, 34)
				ps34.OverlayValues[0] = d0
				ps34.OverlayValues[2] = d2
				ps34.OverlayValues[3] = d3
				ps34.OverlayValues[4] = d4
				ps34.OverlayValues[5] = d5
				ps34.OverlayValues[6] = d6
				ps34.OverlayValues[7] = d7
				ps34.OverlayValues[8] = d8
				ps34.OverlayValues[9] = d9
				ps34.OverlayValues[10] = d10
				ps34.OverlayValues[11] = d11
				ps34.OverlayValues[12] = d12
				ps34.OverlayValues[13] = d13
				ps34.OverlayValues[14] = d14
				ps34.OverlayValues[15] = d15
				ps34.OverlayValues[16] = d16
				ps34.OverlayValues[17] = d17
				ps34.OverlayValues[18] = d18
				ps34.OverlayValues[19] = d19
				ps34.OverlayValues[20] = d20
				ps34.OverlayValues[21] = d21
				ps34.OverlayValues[22] = d22
				ps34.OverlayValues[23] = d23
				ps34.OverlayValues[24] = d24
				ps34.OverlayValues[25] = d25
				ps34.OverlayValues[26] = d26
				ps34.OverlayValues[27] = d27
				ps34.OverlayValues[28] = d28
				ps34.OverlayValues[29] = d29
				ps34.OverlayValues[30] = d30
				ps34.OverlayValues[31] = d31
				ps34.OverlayValues[32] = d32
				ps34.OverlayValues[33] = d33
				return bbs[3].RenderPS(ps34)
			}
			if ps.General {
			}
			ps35 := scm.PhiState{General: ps.General}
			ps35.OverlayValues = make([]scm.JITValueDesc, 34)
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
			return bbs[2].RenderPS(ps35)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d33.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl3)
		ps36 := scm.PhiState{General: true}
		ps36.OverlayValues = make([]scm.JITValueDesc, 34)
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
		ps37 := scm.PhiState{General: true}
		ps37.OverlayValues = make([]scm.JITValueDesc, 34)
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
		snap38 := d0
		snap39 := d2
		snap40 := d3
		snap41 := d4
		snap42 := d5
		snap43 := d6
		snap44 := d7
		snap45 := d8
		snap46 := d9
		snap47 := d10
		snap48 := d11
		snap49 := d12
		snap50 := d13
		snap51 := d14
		snap52 := d15
		snap53 := d16
		snap54 := d17
		snap55 := d18
		snap56 := d19
		snap57 := d20
		snap58 := d21
		snap59 := d22
		snap60 := d23
		snap61 := d24
		snap62 := d25
		snap63 := d26
		snap64 := d27
		snap65 := d28
		snap66 := d29
		snap67 := d30
		snap68 := d31
		snap69 := d32
		snap70 := d33
		alloc71 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps37)
		}
		ctx.RestoreAllocState(alloc71)
		d0 = snap38
		d2 = snap39
		d3 = snap40
		d4 = snap41
		d5 = snap42
		d6 = snap43
		d7 = snap44
		d8 = snap45
		d9 = snap46
		d10 = snap47
		d11 = snap48
		d12 = snap49
		d13 = snap50
		d14 = snap51
		d15 = snap52
		d16 = snap53
		d17 = snap54
		d18 = snap55
		d19 = snap56
		d20 = snap57
		d21 = snap58
		d22 = snap59
		d23 = snap60
		d24 = snap61
		d25 = snap62
		d26 = snap63
		d27 = snap64
		d28 = snap65
		d29 = snap66
		d30 = snap67
		d31 = snap68
		d32 = snap69
		d33 = snap70
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps36)
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
		ctx.ReclaimUntrackedRegs()
		d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d73 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d73)
		ctx.BindReg(r1, &d73)
		ctx.EnsureDesc(&d72)
		if d72.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d72, &d73)
		} else {
			switch d72.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d73, d72)
			case scm.TagInt:
				ctx.EmitMakeInt(d73, d72)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d73, d72)
			case scm.TagNil:
				ctx.EmitMakeNil(d73)
			default:
				ctx.EmitMovPairToResult(&d72, &d73)
			}
		}
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
		if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
			d72 = ps.OverlayValues[72]
		}
		if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
			d73 = ps.OverlayValues[73]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d31)
		var d74 scm.JITValueDesc
		if d31.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d31.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d31.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d74)
		}
		var d75 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
			r40 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r40, fieldAddr)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d75)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
			r41 := ctx.AllocReg()
			ctx.EmitMovRegMem(r41, thisptr.Reg, off)
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d75)
		}
		ctx.EnsureDesc(&d74)
		ctx.EnsureDesc(&d75)
		ctx.EnsureDesc(&d74)
		ctx.ProtectReg(d74.Reg)
		ctx.EnsureDesc(&d75)
		ctx.UnprotectReg(d74.Reg)
		var d76 scm.JITValueDesc
		if d74.Loc == scm.LocImm && d75.Loc == scm.LocImm {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() + d75.Imm.Int())}
		} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
			r42 := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitMovRegReg(r42, d74.Reg)
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d76)
		} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d75.Reg}
			ctx.BindReg(d75.Reg, &d76)
		} else if d74.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d75.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
			ctx.EmitAddInt64(scratch, d75.Reg)
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d76)
		} else if d75.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitMovRegReg(scratch, d74.Reg)
			if d75.Imm.Int() >= -2147483648 && d75.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d75.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d76)
		} else {
			r43 := ctx.AllocRegExcept(d74.Reg, d75.Reg)
			ctx.EmitMovRegReg(r43, d74.Reg)
			ctx.EmitAddInt64(r43, d75.Reg)
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d76)
		}
		if d76.Loc == scm.LocReg && d74.Loc == scm.LocReg && d76.Reg == d74.Reg {
			ctx.TransferReg(d74.Reg)
			d74.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d76)
		ctx.FreeDesc(&d74)
		var d77 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			r44 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r44, fieldAddr)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
			ctx.BindReg(r44, &d77)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r45 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r45, thisptr.Reg, off)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			ctx.BindReg(r45, &d77)
		}
		ctx.EnsureDesc(&d77)
		var d78 scm.JITValueDesc
		if d77.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d77.Imm.Int() > 0)}
		} else {
			r46 := ctx.AllocRegExcept(d77.Reg)
			ctx.EmitCmpRegImm32(d77.Reg, 0)
			ctx.EmitSetcc(r46, scm.CondSignedGreater)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			ctx.BindReg(r46, &d78)
		}
		d79 = d78
		ctx.EnsureDesc(&d79)
		if d79.Loc != scm.LocImm && d79.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d79.Loc == scm.LocImm {
			if d79.Imm.Bool() {
				if ps.General {
				}
				ps80 := scm.PhiState{General: ps.General}
				ps80.OverlayValues = make([]scm.JITValueDesc, 80)
				ps80.OverlayValues[0] = d0
				ps80.OverlayValues[2] = d2
				ps80.OverlayValues[3] = d3
				ps80.OverlayValues[4] = d4
				ps80.OverlayValues[5] = d5
				ps80.OverlayValues[6] = d6
				ps80.OverlayValues[7] = d7
				ps80.OverlayValues[8] = d8
				ps80.OverlayValues[9] = d9
				ps80.OverlayValues[10] = d10
				ps80.OverlayValues[11] = d11
				ps80.OverlayValues[12] = d12
				ps80.OverlayValues[13] = d13
				ps80.OverlayValues[14] = d14
				ps80.OverlayValues[15] = d15
				ps80.OverlayValues[16] = d16
				ps80.OverlayValues[17] = d17
				ps80.OverlayValues[18] = d18
				ps80.OverlayValues[19] = d19
				ps80.OverlayValues[20] = d20
				ps80.OverlayValues[21] = d21
				ps80.OverlayValues[22] = d22
				ps80.OverlayValues[23] = d23
				ps80.OverlayValues[24] = d24
				ps80.OverlayValues[25] = d25
				ps80.OverlayValues[26] = d26
				ps80.OverlayValues[27] = d27
				ps80.OverlayValues[28] = d28
				ps80.OverlayValues[29] = d29
				ps80.OverlayValues[30] = d30
				ps80.OverlayValues[31] = d31
				ps80.OverlayValues[32] = d32
				ps80.OverlayValues[33] = d33
				ps80.OverlayValues[72] = d72
				ps80.OverlayValues[73] = d73
				ps80.OverlayValues[74] = d74
				ps80.OverlayValues[75] = d75
				ps80.OverlayValues[76] = d76
				ps80.OverlayValues[77] = d77
				ps80.OverlayValues[78] = d78
				ps80.OverlayValues[79] = d79
				return bbs[4].RenderPS(ps80)
			}
			if ps.General {
			}
			ps81 := scm.PhiState{General: ps.General}
			ps81.OverlayValues = make([]scm.JITValueDesc, 80)
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
			ps81.OverlayValues[72] = d72
			ps81.OverlayValues[73] = d73
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps81.OverlayValues[79] = d79
			return bbs[5].RenderPS(ps81)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d79.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl6)
		ps82 := scm.PhiState{General: true}
		ps82.OverlayValues = make([]scm.JITValueDesc, 80)
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
		ps82.OverlayValues[72] = d72
		ps82.OverlayValues[73] = d73
		ps82.OverlayValues[74] = d74
		ps82.OverlayValues[75] = d75
		ps82.OverlayValues[76] = d76
		ps82.OverlayValues[77] = d77
		ps82.OverlayValues[78] = d78
		ps82.OverlayValues[79] = d79
		ps83 := scm.PhiState{General: true}
		ps83.OverlayValues = make([]scm.JITValueDesc, 80)
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
		ps83.OverlayValues[72] = d72
		ps83.OverlayValues[73] = d73
		ps83.OverlayValues[74] = d74
		ps83.OverlayValues[75] = d75
		ps83.OverlayValues[76] = d76
		ps83.OverlayValues[77] = d77
		ps83.OverlayValues[78] = d78
		ps83.OverlayValues[79] = d79
		snap84 := d0
		snap85 := d2
		snap86 := d3
		snap87 := d4
		snap88 := d5
		snap89 := d6
		snap90 := d7
		snap91 := d8
		snap92 := d9
		snap93 := d10
		snap94 := d11
		snap95 := d12
		snap96 := d13
		snap97 := d14
		snap98 := d15
		snap99 := d16
		snap100 := d17
		snap101 := d18
		snap102 := d19
		snap103 := d20
		snap104 := d21
		snap105 := d22
		snap106 := d23
		snap107 := d24
		snap108 := d25
		snap109 := d26
		snap110 := d27
		snap111 := d28
		snap112 := d29
		snap113 := d30
		snap114 := d31
		snap115 := d32
		snap116 := d33
		snap117 := d72
		snap118 := d73
		snap119 := d74
		snap120 := d75
		snap121 := d76
		snap122 := d77
		snap123 := d78
		snap124 := d79
		alloc125 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps83)
		}
		ctx.RestoreAllocState(alloc125)
		d0 = snap84
		d2 = snap85
		d3 = snap86
		d4 = snap87
		d5 = snap88
		d6 = snap89
		d7 = snap90
		d8 = snap91
		d9 = snap92
		d10 = snap93
		d11 = snap94
		d12 = snap95
		d13 = snap96
		d14 = snap97
		d15 = snap98
		d16 = snap99
		d17 = snap100
		d18 = snap101
		d19 = snap102
		d20 = snap103
		d21 = snap104
		d22 = snap105
		d23 = snap106
		d24 = snap107
		d25 = snap108
		d26 = snap109
		d27 = snap110
		d28 = snap111
		d29 = snap112
		d30 = snap113
		d31 = snap114
		d32 = snap115
		d33 = snap116
		d72 = snap117
		d73 = snap118
		d74 = snap119
		d75 = snap120
		d76 = snap121
		d77 = snap122
		d78 = snap123
		d79 = snap124
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps82)
		}
		return result
		ctx.FreeDesc(&d78)
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
		ctx.ReclaimUntrackedRegs()
		var d126 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r47, fieldAddr)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d126)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
			r48 := ctx.AllocReg()
			ctx.EmitMovRegMem(r48, thisptr.Reg, off)
			d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
			ctx.BindReg(r48, &d126)
		}
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d126)
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d126)
		ctx.EnsureDesc(&d31)
		ctx.EnsureDesc(&d126)
		var d127 scm.JITValueDesc
		if d31.Loc == scm.LocImm && d126.Loc == scm.LocImm {
			d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d31.Imm.Int()) == uint64(d126.Imm.Int()))}
		} else if d126.Loc == scm.LocImm {
			r49 := ctx.AllocRegExcept(d31.Reg)
			if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d31.Reg, int32(d126.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
				ctx.EmitCmpInt64(d31.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r49, scm.CondEqual)
			d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
			ctx.BindReg(r49, &d127)
		} else if d31.Loc == scm.LocImm {
			r50 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d126.Reg)
			ctx.EmitSetcc(r50, scm.CondEqual)
			d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
			ctx.BindReg(r50, &d127)
		} else {
			r51 := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitCmpInt64(d31.Reg, d126.Reg)
			ctx.EmitSetcc(r51, scm.CondEqual)
			d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
			ctx.BindReg(r51, &d127)
		}
		ctx.FreeDesc(&d31)
		d128 = d127
		ctx.EnsureDesc(&d128)
		if d128.Loc != scm.LocImm && d128.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d128.Loc == scm.LocImm {
			if d128.Imm.Bool() {
				if ps.General {
				}
				ps129 := scm.PhiState{General: ps.General}
				ps129.OverlayValues = make([]scm.JITValueDesc, 129)
				ps129.OverlayValues[0] = d0
				ps129.OverlayValues[2] = d2
				ps129.OverlayValues[3] = d3
				ps129.OverlayValues[4] = d4
				ps129.OverlayValues[5] = d5
				ps129.OverlayValues[6] = d6
				ps129.OverlayValues[7] = d7
				ps129.OverlayValues[8] = d8
				ps129.OverlayValues[9] = d9
				ps129.OverlayValues[10] = d10
				ps129.OverlayValues[11] = d11
				ps129.OverlayValues[12] = d12
				ps129.OverlayValues[13] = d13
				ps129.OverlayValues[14] = d14
				ps129.OverlayValues[15] = d15
				ps129.OverlayValues[16] = d16
				ps129.OverlayValues[17] = d17
				ps129.OverlayValues[18] = d18
				ps129.OverlayValues[19] = d19
				ps129.OverlayValues[20] = d20
				ps129.OverlayValues[21] = d21
				ps129.OverlayValues[22] = d22
				ps129.OverlayValues[23] = d23
				ps129.OverlayValues[24] = d24
				ps129.OverlayValues[25] = d25
				ps129.OverlayValues[26] = d26
				ps129.OverlayValues[27] = d27
				ps129.OverlayValues[28] = d28
				ps129.OverlayValues[29] = d29
				ps129.OverlayValues[30] = d30
				ps129.OverlayValues[31] = d31
				ps129.OverlayValues[32] = d32
				ps129.OverlayValues[33] = d33
				ps129.OverlayValues[72] = d72
				ps129.OverlayValues[73] = d73
				ps129.OverlayValues[74] = d74
				ps129.OverlayValues[75] = d75
				ps129.OverlayValues[76] = d76
				ps129.OverlayValues[77] = d77
				ps129.OverlayValues[78] = d78
				ps129.OverlayValues[79] = d79
				ps129.OverlayValues[126] = d126
				ps129.OverlayValues[127] = d127
				ps129.OverlayValues[128] = d128
				return bbs[1].RenderPS(ps129)
			}
			if ps.General {
			}
			ps130 := scm.PhiState{General: ps.General}
			ps130.OverlayValues = make([]scm.JITValueDesc, 129)
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
			ps130.OverlayValues[72] = d72
			ps130.OverlayValues[73] = d73
			ps130.OverlayValues[74] = d74
			ps130.OverlayValues[75] = d75
			ps130.OverlayValues[76] = d76
			ps130.OverlayValues[77] = d77
			ps130.OverlayValues[78] = d78
			ps130.OverlayValues[79] = d79
			ps130.OverlayValues[126] = d126
			ps130.OverlayValues[127] = d127
			ps130.OverlayValues[128] = d128
			return bbs[2].RenderPS(ps130)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d128.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl3)
		ps131 := scm.PhiState{General: true}
		ps131.OverlayValues = make([]scm.JITValueDesc, 129)
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
		ps131.OverlayValues[72] = d72
		ps131.OverlayValues[73] = d73
		ps131.OverlayValues[74] = d74
		ps131.OverlayValues[75] = d75
		ps131.OverlayValues[76] = d76
		ps131.OverlayValues[77] = d77
		ps131.OverlayValues[78] = d78
		ps131.OverlayValues[79] = d79
		ps131.OverlayValues[126] = d126
		ps131.OverlayValues[127] = d127
		ps131.OverlayValues[128] = d128
		ps132 := scm.PhiState{General: true}
		ps132.OverlayValues = make([]scm.JITValueDesc, 129)
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
		ps132.OverlayValues[72] = d72
		ps132.OverlayValues[73] = d73
		ps132.OverlayValues[74] = d74
		ps132.OverlayValues[75] = d75
		ps132.OverlayValues[76] = d76
		ps132.OverlayValues[77] = d77
		ps132.OverlayValues[78] = d78
		ps132.OverlayValues[79] = d79
		ps132.OverlayValues[126] = d126
		ps132.OverlayValues[127] = d127
		ps132.OverlayValues[128] = d128
		snap133 := d0
		snap134 := d2
		snap135 := d3
		snap136 := d4
		snap137 := d5
		snap138 := d6
		snap139 := d7
		snap140 := d8
		snap141 := d9
		snap142 := d10
		snap143 := d11
		snap144 := d12
		snap145 := d13
		snap146 := d14
		snap147 := d15
		snap148 := d16
		snap149 := d17
		snap150 := d18
		snap151 := d19
		snap152 := d20
		snap153 := d21
		snap154 := d22
		snap155 := d23
		snap156 := d24
		snap157 := d25
		snap158 := d26
		snap159 := d27
		snap160 := d28
		snap161 := d29
		snap162 := d30
		snap163 := d31
		snap164 := d32
		snap165 := d33
		snap166 := d72
		snap167 := d73
		snap168 := d74
		snap169 := d75
		snap170 := d76
		snap171 := d77
		snap172 := d78
		snap173 := d79
		snap174 := d126
		snap175 := d127
		snap176 := d128
		alloc177 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps132)
		}
		ctx.RestoreAllocState(alloc177)
		d0 = snap133
		d2 = snap134
		d3 = snap135
		d4 = snap136
		d5 = snap137
		d6 = snap138
		d7 = snap139
		d8 = snap140
		d9 = snap141
		d10 = snap142
		d11 = snap143
		d12 = snap144
		d13 = snap145
		d14 = snap146
		d15 = snap147
		d16 = snap148
		d17 = snap149
		d18 = snap150
		d19 = snap151
		d20 = snap152
		d21 = snap153
		d22 = snap154
		d23 = snap155
		d24 = snap156
		d25 = snap157
		d26 = snap158
		d27 = snap159
		d28 = snap160
		d29 = snap161
		d30 = snap162
		d31 = snap163
		d32 = snap164
		d33 = snap165
		d72 = snap166
		d73 = snap167
		d74 = snap168
		d75 = snap169
		d76 = snap170
		d77 = snap171
		d78 = snap172
		d79 = snap173
		d126 = snap174
		d127 = snap175
		d128 = snap176
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps131)
		}
		return result
		ctx.FreeDesc(&d127)
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
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d77)
		r52 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r52, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r53 := ctx.AllocReg()
		if d77.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r53, uint64(d77.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r53, d77.Reg)
			ctx.EmitShlRegImm8(r53, 3)
		}
		ctx.EmitAddInt64(r52, r53)
		ctx.FreeReg(r53)
		r54 := ctx.AllocRegExcept(r52)
		ctx.EmitMovRegMem(r54, r52, 0)
		ctx.FreeReg(r52)
		d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
		ctx.BindReg(r54, &d178)
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d178)
		ctx.EnsureDesc(&d76)
		ctx.ProtectReg(d76.Reg)
		ctx.EnsureDesc(&d178)
		ctx.UnprotectReg(d76.Reg)
		var d179 scm.JITValueDesc
		if d76.Loc == scm.LocImm && d178.Loc == scm.LocImm {
			d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() * d178.Imm.Int())}
		} else if d76.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d178.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d76.Imm.Int()))
			ctx.EmitImulInt64(scratch, d178.Reg)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d179)
		} else if d178.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d76.Reg)
			ctx.EmitMovRegReg(scratch, d76.Reg)
			if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d178.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d179)
		} else {
			r55 := ctx.AllocRegExcept(d76.Reg, d178.Reg)
			ctx.EmitMovRegReg(r55, d76.Reg)
			ctx.EmitImulInt64(r55, d178.Reg)
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d179)
		}
		if d179.Loc == scm.LocReg && d76.Loc == scm.LocReg && d179.Reg == d76.Reg {
			ctx.TransferReg(d76.Reg)
			d76.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d178)
		ctx.EnsureDesc(&d179)
		d180 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d180)
		ctx.BindReg(r1, &d180)
		ctx.EnsureDesc(&d179)
		ctx.EmitMakeInt(d180, d179)
		if d179.Loc == scm.LocReg {
			ctx.FreeReg(d179.Reg)
		}
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
		if len(ps.OverlayValues) > 126 && ps.OverlayValues[126].Loc != scm.LocNone {
			d126 = ps.OverlayValues[126]
		}
		if len(ps.OverlayValues) > 127 && ps.OverlayValues[127].Loc != scm.LocNone {
			d127 = ps.OverlayValues[127]
		}
		if len(ps.OverlayValues) > 128 && ps.OverlayValues[128].Loc != scm.LocNone {
			d128 = ps.OverlayValues[128]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d76)
		var d181 scm.JITValueDesc
		if d76.Loc == scm.LocImm {
			d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d76.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d76.Reg)
			d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d76.Reg}
			ctx.BindReg(d76.Reg, &d181)
		}
		ctx.FreeDesc(&d76)
		ctx.EnsureDesc(&d77)
		ctx.EnsureDesc(&d77)
		var d182 scm.JITValueDesc
		if d77.Loc == scm.LocImm {
			d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d77.Imm.Int()))))}
		} else {
			r56 := ctx.AllocReg()
			ctx.EmitMovRegReg(r56, d77.Reg)
			ctx.EmitShlRegImm8(r56, 56)
			ctx.EmitSarRegImm8(r56, 56)
			d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d182)
		}
		ctx.EnsureDesc(&d182)
		ctx.EnsureDesc(&d182)
		var d183 scm.JITValueDesc
		if d182.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d182.Reg)
			ctx.EmitMovRegReg(scratch, d182.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d183)
		}
		if d183.Loc == scm.LocReg && d182.Loc == scm.LocReg && d183.Reg == d182.Reg {
			ctx.TransferReg(d182.Reg)
			d182.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d182)
		ctx.EnsureDesc(&d183)
		r57 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r57, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r58 := ctx.AllocReg()
		if d183.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r58, uint64(d183.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r58, d183.Reg)
			ctx.EmitShlRegImm8(r58, 3)
		}
		ctx.EmitAddInt64(r57, r58)
		ctx.FreeReg(r58)
		r59 := ctx.AllocRegExcept(r57)
		ctx.EmitMovRegMem(r59, r57, 0)
		ctx.FreeReg(r57)
		d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
		ctx.BindReg(r59, &d184)
		ctx.FreeDesc(&d183)
		ctx.EnsureDesc(&d181)
		ctx.EnsureDesc(&d184)
		ctx.EnsureDesc(&d181)
		ctx.EnsureDesc(&d184)
		var d185 scm.JITValueDesc
		if d181.Loc == scm.LocImm && d184.Loc == scm.LocImm {
			d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d181.Imm.Float() * d184.Imm.Float())}
		} else if d181.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d184.Reg)
			_, xBits := d181.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d184.Reg)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d185)
		} else if d184.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d181.Reg)
			ctx.EmitMovRegReg(scratch, d181.Reg)
			_, yBits := d184.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d185)
		} else {
			r60 := ctx.AllocRegExcept(d181.Reg, d184.Reg)
			ctx.EmitMovRegReg(r60, d181.Reg)
			ctx.EmitMulFloat64(r60, d184.Reg)
			d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r60}
			ctx.BindReg(r60, &d185)
		}
		if d185.Loc == scm.LocReg && d181.Loc == scm.LocReg && d185.Reg == d181.Reg {
			ctx.TransferReg(d181.Reg)
			d181.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d181)
		ctx.FreeDesc(&d184)
		ctx.EnsureDesc(&d185)
		d186 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d186)
		ctx.BindReg(r1, &d186)
		ctx.EnsureDesc(&d185)
		ctx.EmitMakeFloat(d186, d185)
		if d185.Loc == scm.LocReg {
			ctx.FreeReg(d185.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps187 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps187)
	ctx.MarkLabel(lbl0)
	d188 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d188)
	ctx.BindReg(r1, &d188)
	ctx.EmitMovPairToResult(&d188, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
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
