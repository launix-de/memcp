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
	var d34 scm.JITValueDesc
	_ = d34
	var d35 scm.JITValueDesc
	_ = d35
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
	var d132 scm.JITValueDesc
	_ = d132
	var d133 scm.JITValueDesc
	_ = d133
	var d134 scm.JITValueDesc
	_ = d134
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
				ctx.SyncDesc(&d12)
				if d12.Loc == scm.LocReg {
					ctx.ProtectReg(d12.Reg)
				} else if d12.Loc == scm.LocRegPair {
					ctx.ProtectReg(d12.Reg)
					ctx.ProtectReg(d12.Reg2)
				}
				d18 = d12
				if d18.Loc == scm.LocNone {
					panic("jit: phi source has no location")
				}
				ctx.EnsureDesc(&d18)
				ctx.EmitStoreToStack(d18, int32(phiBase1)+int32(0))
				if d12.Loc == scm.LocReg {
					ctx.UnprotectReg(d12.Reg)
				} else if d12.Loc == scm.LocRegPair {
					ctx.UnprotectReg(d12.Reg)
					ctx.UnprotectReg(d12.Reg2)
				}
				ctx.EmitJmp(lbl9)
			}
		} else {
			ctx.EmitCmpRegImm32(d17.Reg, 0)
			ctx.EmitJump(scm.CondNotEqual, lbl10)
			ctx.EmitJmp(lbl11)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl8)
			ctx.MarkLabel(lbl11)
			ctx.SyncDesc(&d12)
			if d12.Loc == scm.LocReg {
				ctx.ProtectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.ProtectReg(d12.Reg)
				ctx.ProtectReg(d12.Reg2)
			}
			d19 = d12
			if d19.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			ctx.EnsureDesc(&d19)
			ctx.EmitStoreToStack(d19, int32(phiBase1)+int32(0))
			if d12.Loc == scm.LocReg {
				ctx.UnprotectReg(d12.Reg)
			} else if d12.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d12.Reg)
				ctx.UnprotectReg(d12.Reg2)
			}
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
		var d20 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d4.Reg)
			ctx.EmitShlRegImm8(r22, 56)
			ctx.EmitShrRegImm8(r22, 56)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d20)
		}
		ctx.ReclaimUntrackedRegs()
		d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d20)
		ctx.EnsureDesc(&d21)
		ctx.ProtectReg(d21.Reg)
		ctx.EnsureDesc(&d20)
		ctx.UnprotectReg(d21.Reg)
		var d22 scm.JITValueDesc
		if d21.Loc == scm.LocImm && d20.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() - d20.Imm.Int())}
		} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
			r23 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(r23, d21.Reg)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d22)
		} else if d21.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d20.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
			ctx.EmitSubInt64(scratch, d20.Reg)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d22)
		} else if d20.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitMovRegReg(scratch, d21.Reg)
			if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d20.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d20.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d22)
		} else {
			r24 := ctx.AllocRegExcept(d21.Reg, d20.Reg)
			ctx.EmitMovRegReg(r24, d21.Reg)
			ctx.EmitSubInt64(r24, d20.Reg)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d22)
		}
		if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
			ctx.TransferReg(d21.Reg)
			d21.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d22)
		var d23 scm.JITValueDesc
		if d2.Loc == scm.LocImm && d22.Loc == scm.LocImm {
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) >> uint64(d22.Imm.Int())))}
		} else if d22.Loc == scm.LocImm {
			r25 := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegReg(r25, d2.Reg)
			ctx.EmitShrRegImm8(r25, uint8(d22.Imm.Int()))
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d23)
		} else {
			{
				shiftSrc := d2.Reg
				r26 := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(r26, d2.Reg)
				shiftSrc = r26
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d22.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d22.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d22.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d23)
			}
		}
		if d23.Loc == scm.LocReg && d2.Loc == scm.LocReg && d23.Reg == d2.Reg {
			ctx.TransferReg(d2.Reg)
			d2.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d2)
		ctx.FreeDesc(&d22)
		ctx.ReclaimUntrackedRegs()
		r27 := ctx.AllocReg()
		ctx.EnsureDesc(&d23)
		ctx.EnsureDesc(&d23)
		if d23.Loc == scm.LocRegPair {
			panic("jit: scalar inline return has scm.LocRegPair")
		} else {
			ctx.EmitMovToReg(r27, d23)
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
		var d24 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
		} else {
			r28 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r28, d6.Reg)
			ctx.EmitShrRegImm8(r28, 6)
			d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d24)
		}
		if d24.Loc == scm.LocReg && d6.Loc == scm.LocReg && d24.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d24)
		ctx.EnsureDesc(&d24)
		var d25 scm.JITValueDesc
		if d24.Loc == scm.LocImm {
			d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d24.Reg)
			ctx.EmitMovRegReg(scratch, d24.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d25)
		}
		if d25.Loc == scm.LocReg && d24.Loc == scm.LocReg && d25.Reg == d24.Reg {
			ctx.TransferReg(d24.Reg)
			d24.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d24)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		d27 = ctx.EmitSliceElementAddress(&d7, &d25, 8)
		ctx.EnsureDesc(&d27)
		ctx.EmitMovRegMem(d27.Reg, d27.Reg, 0)
		d26 = d27
		ctx.FreeDesc(&d25)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d6)
		var d28 scm.JITValueDesc
		if d6.Loc == scm.LocImm {
			d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
		} else {
			r29 := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegReg(r29, d6.Reg)
			ctx.EmitAndRegImm32(r29, 63)
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d28)
		}
		if d28.Loc == scm.LocReg && d6.Loc == scm.LocReg && d28.Reg == d6.Reg {
			ctx.TransferReg(d6.Reg)
			d6.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d29)
		ctx.ProtectReg(d29.Reg)
		ctx.EnsureDesc(&d28)
		ctx.UnprotectReg(d29.Reg)
		var d30 scm.JITValueDesc
		if d29.Loc == scm.LocImm && d28.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() - d28.Imm.Int())}
		} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
			r30 := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(r30, d29.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d30)
		} else if d29.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
			ctx.EmitSubInt64(scratch, d28.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else if d28.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d29.Reg)
			ctx.EmitMovRegReg(scratch, d29.Reg)
			if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d28.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d30)
		} else {
			r31 := ctx.AllocRegExcept(d29.Reg, d28.Reg)
			ctx.EmitMovRegReg(r31, d29.Reg)
			ctx.EmitSubInt64(r31, d28.Reg)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d30)
		}
		if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
			ctx.TransferReg(d29.Reg)
			d29.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d26)
		ctx.EnsureDesc(&d30)
		var d31 scm.JITValueDesc
		if d26.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d26.Imm.Int()) >> uint64(d30.Imm.Int())))}
		} else if d30.Loc == scm.LocImm {
			r32 := ctx.AllocRegExcept(d26.Reg)
			ctx.EmitMovRegReg(r32, d26.Reg)
			ctx.EmitShrRegImm8(r32, uint8(d30.Imm.Int()))
			d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			ctx.BindReg(r32, &d31)
		} else {
			{
				shiftSrc := d26.Reg
				r33 := ctx.AllocRegExcept(d26.Reg)
				ctx.EmitMovRegReg(r33, d26.Reg)
				shiftSrc = r33
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d30.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d30.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d30.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d31)
			}
		}
		if d31.Loc == scm.LocReg && d26.Loc == scm.LocReg && d31.Reg == d26.Reg {
			ctx.TransferReg(d26.Reg)
			d26.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d26)
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d31)
		var d32 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d31.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() | d31.Imm.Int())}
		} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			ctx.BindReg(d31.Reg, &d32)
		} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r34, d12.Reg)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d32)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d31.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitOrInt64(scratch, d31.Reg)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d32)
		} else if d31.Loc == scm.LocImm {
			r35 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r35, d12.Reg)
			if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r35, int32(d31.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
				ctx.EmitOrInt64(r35, scm.RegR11)
			}
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d32)
		} else {
			r36 := ctx.AllocRegExcept(d12.Reg, d31.Reg)
			ctx.EmitMovRegReg(r36, d12.Reg)
			ctx.EmitOrInt64(r36, d31.Reg)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d32)
		}
		if d32.Loc == scm.LocReg && d12.Loc == scm.LocReg && d32.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.EnsureDesc(&d32)
		ctx.EmitStoreToStack(d32, int32(phiBase1)+int32(0))
		ctx.StabilizeDescForControlFlow(&d32)
		ctx.FreeDesc(&d31)
		ctx.ReclaimUntrackedRegs()
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl7)
		d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
		ctx.BindReg(r27, &d33)
		ctx.BindReg(r27, &d33)
		ctx.StabilizeDescForControlFlow(&d33)
		ctx.FreeDesc(&idxInt)
		var d34 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r37, fieldAddr)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d34)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r38 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r38, thisptr.Reg, off)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d34)
		}
		d35 = d34
		ctx.EnsureDesc(&d35)
		if d35.Loc != scm.LocImm && d35.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d35.Loc == scm.LocImm {
			if d35.Imm.Bool() {
				if ps.General {
				}
				ps36 := scm.PhiState{General: ps.General}
				ps36.OverlayValues = make([]scm.JITValueDesc, 36)
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
				ps36.OverlayValues[35] = d35
				return bbs[3].RenderPS(ps36)
			}
			if ps.General {
			}
			ps37 := scm.PhiState{General: ps.General}
			ps37.OverlayValues = make([]scm.JITValueDesc, 36)
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
			ps37.OverlayValues[35] = d35
			return bbs[2].RenderPS(ps37)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d35.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl3)
		ps38 := scm.PhiState{General: true}
		ps38.OverlayValues = make([]scm.JITValueDesc, 36)
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
		ps38.OverlayValues[35] = d35
		ps39 := scm.PhiState{General: true}
		ps39.OverlayValues = make([]scm.JITValueDesc, 36)
		ps39.OverlayValues[0] = d0
		ps39.OverlayValues[2] = d2
		ps39.OverlayValues[3] = d3
		ps39.OverlayValues[4] = d4
		ps39.OverlayValues[5] = d5
		ps39.OverlayValues[6] = d6
		ps39.OverlayValues[7] = d7
		ps39.OverlayValues[8] = d8
		ps39.OverlayValues[9] = d9
		ps39.OverlayValues[10] = d10
		ps39.OverlayValues[11] = d11
		ps39.OverlayValues[12] = d12
		ps39.OverlayValues[13] = d13
		ps39.OverlayValues[14] = d14
		ps39.OverlayValues[15] = d15
		ps39.OverlayValues[16] = d16
		ps39.OverlayValues[17] = d17
		ps39.OverlayValues[18] = d18
		ps39.OverlayValues[19] = d19
		ps39.OverlayValues[20] = d20
		ps39.OverlayValues[21] = d21
		ps39.OverlayValues[22] = d22
		ps39.OverlayValues[23] = d23
		ps39.OverlayValues[24] = d24
		ps39.OverlayValues[25] = d25
		ps39.OverlayValues[26] = d26
		ps39.OverlayValues[27] = d27
		ps39.OverlayValues[28] = d28
		ps39.OverlayValues[29] = d29
		ps39.OverlayValues[30] = d30
		ps39.OverlayValues[31] = d31
		ps39.OverlayValues[32] = d32
		ps39.OverlayValues[33] = d33
		ps39.OverlayValues[34] = d34
		ps39.OverlayValues[35] = d35
		snap40 := d0
		snap41 := d2
		snap42 := d3
		snap43 := d4
		snap44 := d5
		snap45 := d6
		snap46 := d7
		snap47 := d8
		snap48 := d9
		snap49 := d10
		snap50 := d11
		snap51 := d12
		snap52 := d13
		snap53 := d14
		snap54 := d15
		snap55 := d16
		snap56 := d17
		snap57 := d18
		snap58 := d19
		snap59 := d20
		snap60 := d21
		snap61 := d22
		snap62 := d23
		snap63 := d24
		snap64 := d25
		snap65 := d26
		snap66 := d27
		snap67 := d28
		snap68 := d29
		snap69 := d30
		snap70 := d31
		snap71 := d32
		snap72 := d33
		snap73 := d34
		snap74 := d35
		alloc75 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps39)
		}
		ctx.RestoreAllocState(alloc75)
		d0 = snap40
		d2 = snap41
		d3 = snap42
		d4 = snap43
		d5 = snap44
		d6 = snap45
		d7 = snap46
		d8 = snap47
		d9 = snap48
		d10 = snap49
		d11 = snap50
		d12 = snap51
		d13 = snap52
		d14 = snap53
		d15 = snap54
		d16 = snap55
		d17 = snap56
		d18 = snap57
		d19 = snap58
		d20 = snap59
		d21 = snap60
		d22 = snap61
		d23 = snap62
		d24 = snap63
		d25 = snap64
		d26 = snap65
		d27 = snap66
		d28 = snap67
		d29 = snap68
		d30 = snap69
		d31 = snap70
		d32 = snap71
		d33 = snap72
		d34 = snap73
		d35 = snap74
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps38)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		ctx.ReclaimUntrackedRegs()
		d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d77 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d77)
		ctx.BindReg(r1, &d77)
		ctx.EnsureDesc(&d76)
		if d76.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d76, &d77)
		} else {
			switch d76.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d77, d76)
			case scm.TagInt:
				ctx.EmitMakeInt(d77, d76)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d77, d76)
			case scm.TagNil:
				ctx.EmitMakeNil(d77)
			default:
				ctx.EmitMovPairToResult(&d76, &d77)
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
			d76 = ps.OverlayValues[76]
		}
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d33)
		var d78 scm.JITValueDesc
		if d33.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d33.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d33.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d78)
		}
		var d79 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
			r40 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r40, fieldAddr)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d79)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
			r41 := ctx.AllocReg()
			ctx.EmitMovRegMem(r41, thisptr.Reg, off)
			d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d79)
		}
		ctx.EnsureDesc(&d78)
		ctx.EnsureDesc(&d79)
		ctx.EnsureDesc(&d78)
		ctx.ProtectReg(d78.Reg)
		ctx.EnsureDesc(&d79)
		ctx.UnprotectReg(d78.Reg)
		var d80 scm.JITValueDesc
		if d78.Loc == scm.LocImm && d79.Loc == scm.LocImm {
			d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() + d79.Imm.Int())}
		} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
			r42 := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitMovRegReg(r42, d78.Reg)
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			ctx.BindReg(r42, &d80)
		} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg}
			ctx.BindReg(d79.Reg, &d80)
		} else if d78.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d79.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
			ctx.EmitAddInt64(scratch, d79.Reg)
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d80)
		} else if d79.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitMovRegReg(scratch, d78.Reg)
			if d79.Imm.Int() >= -2147483648 && d79.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d79.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d80)
		} else {
			r43 := ctx.AllocRegExcept(d78.Reg, d79.Reg)
			ctx.EmitMovRegReg(r43, d78.Reg)
			ctx.EmitAddInt64(r43, d79.Reg)
			d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			ctx.BindReg(r43, &d80)
		}
		if d80.Loc == scm.LocReg && d78.Loc == scm.LocReg && d80.Reg == d78.Reg {
			ctx.TransferReg(d78.Reg)
			d78.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d80)
		ctx.FreeDesc(&d78)
		var d81 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			r44 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r44, fieldAddr)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
			ctx.BindReg(r44, &d81)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r45 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r45, thisptr.Reg, off)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			ctx.BindReg(r45, &d81)
		}
		ctx.EnsureDesc(&d81)
		var d82 scm.JITValueDesc
		if d81.Loc == scm.LocImm {
			d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d81.Imm.Int() > 0)}
		} else {
			r46 := ctx.AllocRegExcept(d81.Reg)
			ctx.EmitCmpRegImm32(d81.Reg, 0)
			ctx.EmitSetcc(r46, scm.CondSignedGreater)
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			ctx.BindReg(r46, &d82)
		}
		d83 = d82
		ctx.EnsureDesc(&d83)
		if d83.Loc != scm.LocImm && d83.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d83.Loc == scm.LocImm {
			if d83.Imm.Bool() {
				if ps.General {
				}
				ps84 := scm.PhiState{General: ps.General}
				ps84.OverlayValues = make([]scm.JITValueDesc, 84)
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
				ps84.OverlayValues[35] = d35
				ps84.OverlayValues[76] = d76
				ps84.OverlayValues[77] = d77
				ps84.OverlayValues[78] = d78
				ps84.OverlayValues[79] = d79
				ps84.OverlayValues[80] = d80
				ps84.OverlayValues[81] = d81
				ps84.OverlayValues[82] = d82
				ps84.OverlayValues[83] = d83
				return bbs[4].RenderPS(ps84)
			}
			if ps.General {
			}
			ps85 := scm.PhiState{General: ps.General}
			ps85.OverlayValues = make([]scm.JITValueDesc, 84)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[5] = d5
			ps85.OverlayValues[6] = d6
			ps85.OverlayValues[7] = d7
			ps85.OverlayValues[8] = d8
			ps85.OverlayValues[9] = d9
			ps85.OverlayValues[10] = d10
			ps85.OverlayValues[11] = d11
			ps85.OverlayValues[12] = d12
			ps85.OverlayValues[13] = d13
			ps85.OverlayValues[14] = d14
			ps85.OverlayValues[15] = d15
			ps85.OverlayValues[16] = d16
			ps85.OverlayValues[17] = d17
			ps85.OverlayValues[18] = d18
			ps85.OverlayValues[19] = d19
			ps85.OverlayValues[20] = d20
			ps85.OverlayValues[21] = d21
			ps85.OverlayValues[22] = d22
			ps85.OverlayValues[23] = d23
			ps85.OverlayValues[24] = d24
			ps85.OverlayValues[25] = d25
			ps85.OverlayValues[26] = d26
			ps85.OverlayValues[27] = d27
			ps85.OverlayValues[28] = d28
			ps85.OverlayValues[29] = d29
			ps85.OverlayValues[30] = d30
			ps85.OverlayValues[31] = d31
			ps85.OverlayValues[32] = d32
			ps85.OverlayValues[33] = d33
			ps85.OverlayValues[34] = d34
			ps85.OverlayValues[35] = d35
			ps85.OverlayValues[76] = d76
			ps85.OverlayValues[77] = d77
			ps85.OverlayValues[78] = d78
			ps85.OverlayValues[79] = d79
			ps85.OverlayValues[80] = d80
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			return bbs[5].RenderPS(ps85)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d83.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl6)
		ps86 := scm.PhiState{General: true}
		ps86.OverlayValues = make([]scm.JITValueDesc, 84)
		ps86.OverlayValues[0] = d0
		ps86.OverlayValues[2] = d2
		ps86.OverlayValues[3] = d3
		ps86.OverlayValues[4] = d4
		ps86.OverlayValues[5] = d5
		ps86.OverlayValues[6] = d6
		ps86.OverlayValues[7] = d7
		ps86.OverlayValues[8] = d8
		ps86.OverlayValues[9] = d9
		ps86.OverlayValues[10] = d10
		ps86.OverlayValues[11] = d11
		ps86.OverlayValues[12] = d12
		ps86.OverlayValues[13] = d13
		ps86.OverlayValues[14] = d14
		ps86.OverlayValues[15] = d15
		ps86.OverlayValues[16] = d16
		ps86.OverlayValues[17] = d17
		ps86.OverlayValues[18] = d18
		ps86.OverlayValues[19] = d19
		ps86.OverlayValues[20] = d20
		ps86.OverlayValues[21] = d21
		ps86.OverlayValues[22] = d22
		ps86.OverlayValues[23] = d23
		ps86.OverlayValues[24] = d24
		ps86.OverlayValues[25] = d25
		ps86.OverlayValues[26] = d26
		ps86.OverlayValues[27] = d27
		ps86.OverlayValues[28] = d28
		ps86.OverlayValues[29] = d29
		ps86.OverlayValues[30] = d30
		ps86.OverlayValues[31] = d31
		ps86.OverlayValues[32] = d32
		ps86.OverlayValues[33] = d33
		ps86.OverlayValues[34] = d34
		ps86.OverlayValues[35] = d35
		ps86.OverlayValues[76] = d76
		ps86.OverlayValues[77] = d77
		ps86.OverlayValues[78] = d78
		ps86.OverlayValues[79] = d79
		ps86.OverlayValues[80] = d80
		ps86.OverlayValues[81] = d81
		ps86.OverlayValues[82] = d82
		ps86.OverlayValues[83] = d83
		ps87 := scm.PhiState{General: true}
		ps87.OverlayValues = make([]scm.JITValueDesc, 84)
		ps87.OverlayValues[0] = d0
		ps87.OverlayValues[2] = d2
		ps87.OverlayValues[3] = d3
		ps87.OverlayValues[4] = d4
		ps87.OverlayValues[5] = d5
		ps87.OverlayValues[6] = d6
		ps87.OverlayValues[7] = d7
		ps87.OverlayValues[8] = d8
		ps87.OverlayValues[9] = d9
		ps87.OverlayValues[10] = d10
		ps87.OverlayValues[11] = d11
		ps87.OverlayValues[12] = d12
		ps87.OverlayValues[13] = d13
		ps87.OverlayValues[14] = d14
		ps87.OverlayValues[15] = d15
		ps87.OverlayValues[16] = d16
		ps87.OverlayValues[17] = d17
		ps87.OverlayValues[18] = d18
		ps87.OverlayValues[19] = d19
		ps87.OverlayValues[20] = d20
		ps87.OverlayValues[21] = d21
		ps87.OverlayValues[22] = d22
		ps87.OverlayValues[23] = d23
		ps87.OverlayValues[24] = d24
		ps87.OverlayValues[25] = d25
		ps87.OverlayValues[26] = d26
		ps87.OverlayValues[27] = d27
		ps87.OverlayValues[28] = d28
		ps87.OverlayValues[29] = d29
		ps87.OverlayValues[30] = d30
		ps87.OverlayValues[31] = d31
		ps87.OverlayValues[32] = d32
		ps87.OverlayValues[33] = d33
		ps87.OverlayValues[34] = d34
		ps87.OverlayValues[35] = d35
		ps87.OverlayValues[76] = d76
		ps87.OverlayValues[77] = d77
		ps87.OverlayValues[78] = d78
		ps87.OverlayValues[79] = d79
		ps87.OverlayValues[80] = d80
		ps87.OverlayValues[81] = d81
		ps87.OverlayValues[82] = d82
		ps87.OverlayValues[83] = d83
		snap88 := d0
		snap89 := d2
		snap90 := d3
		snap91 := d4
		snap92 := d5
		snap93 := d6
		snap94 := d7
		snap95 := d8
		snap96 := d9
		snap97 := d10
		snap98 := d11
		snap99 := d12
		snap100 := d13
		snap101 := d14
		snap102 := d15
		snap103 := d16
		snap104 := d17
		snap105 := d18
		snap106 := d19
		snap107 := d20
		snap108 := d21
		snap109 := d22
		snap110 := d23
		snap111 := d24
		snap112 := d25
		snap113 := d26
		snap114 := d27
		snap115 := d28
		snap116 := d29
		snap117 := d30
		snap118 := d31
		snap119 := d32
		snap120 := d33
		snap121 := d34
		snap122 := d35
		snap123 := d76
		snap124 := d77
		snap125 := d78
		snap126 := d79
		snap127 := d80
		snap128 := d81
		snap129 := d82
		snap130 := d83
		alloc131 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps87)
		}
		ctx.RestoreAllocState(alloc131)
		d0 = snap88
		d2 = snap89
		d3 = snap90
		d4 = snap91
		d5 = snap92
		d6 = snap93
		d7 = snap94
		d8 = snap95
		d9 = snap96
		d10 = snap97
		d11 = snap98
		d12 = snap99
		d13 = snap100
		d14 = snap101
		d15 = snap102
		d16 = snap103
		d17 = snap104
		d18 = snap105
		d19 = snap106
		d20 = snap107
		d21 = snap108
		d22 = snap109
		d23 = snap110
		d24 = snap111
		d25 = snap112
		d26 = snap113
		d27 = snap114
		d28 = snap115
		d29 = snap116
		d30 = snap117
		d31 = snap118
		d32 = snap119
		d33 = snap120
		d34 = snap121
		d35 = snap122
		d76 = snap123
		d77 = snap124
		d78 = snap125
		d79 = snap126
		d80 = snap127
		d81 = snap128
		d82 = snap129
		d83 = snap130
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps86)
		}
		return result
		ctx.FreeDesc(&d82)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
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
		ctx.ReclaimUntrackedRegs()
		var d132 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
			r47 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r47, fieldAddr)
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			ctx.BindReg(r47, &d132)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
			r48 := ctx.AllocReg()
			ctx.EmitMovRegMem(r48, thisptr.Reg, off)
			d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
			ctx.BindReg(r48, &d132)
		}
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d132)
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d132)
		var d133 scm.JITValueDesc
		if d33.Loc == scm.LocImm && d132.Loc == scm.LocImm {
			d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d33.Imm.Int()) == uint64(d132.Imm.Int()))}
		} else if d132.Loc == scm.LocImm {
			r49 := ctx.AllocRegExcept(d33.Reg)
			if d132.Imm.Int() >= -2147483648 && d132.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d33.Reg, int32(d132.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d132.Imm.Int()))
				ctx.EmitCmpInt64(d33.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r49, scm.CondEqual)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
			ctx.BindReg(r49, &d133)
		} else if d33.Loc == scm.LocImm {
			r50 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d132.Reg)
			ctx.EmitSetcc(r50, scm.CondEqual)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
			ctx.BindReg(r50, &d133)
		} else {
			r51 := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitCmpInt64(d33.Reg, d132.Reg)
			ctx.EmitSetcc(r51, scm.CondEqual)
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
			ctx.BindReg(r51, &d133)
		}
		ctx.FreeDesc(&d33)
		d134 = d133
		ctx.EnsureDesc(&d134)
		if d134.Loc != scm.LocImm && d134.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d134.Loc == scm.LocImm {
			if d134.Imm.Bool() {
				if ps.General {
				}
				ps135 := scm.PhiState{General: ps.General}
				ps135.OverlayValues = make([]scm.JITValueDesc, 135)
				ps135.OverlayValues[0] = d0
				ps135.OverlayValues[2] = d2
				ps135.OverlayValues[3] = d3
				ps135.OverlayValues[4] = d4
				ps135.OverlayValues[5] = d5
				ps135.OverlayValues[6] = d6
				ps135.OverlayValues[7] = d7
				ps135.OverlayValues[8] = d8
				ps135.OverlayValues[9] = d9
				ps135.OverlayValues[10] = d10
				ps135.OverlayValues[11] = d11
				ps135.OverlayValues[12] = d12
				ps135.OverlayValues[13] = d13
				ps135.OverlayValues[14] = d14
				ps135.OverlayValues[15] = d15
				ps135.OverlayValues[16] = d16
				ps135.OverlayValues[17] = d17
				ps135.OverlayValues[18] = d18
				ps135.OverlayValues[19] = d19
				ps135.OverlayValues[20] = d20
				ps135.OverlayValues[21] = d21
				ps135.OverlayValues[22] = d22
				ps135.OverlayValues[23] = d23
				ps135.OverlayValues[24] = d24
				ps135.OverlayValues[25] = d25
				ps135.OverlayValues[26] = d26
				ps135.OverlayValues[27] = d27
				ps135.OverlayValues[28] = d28
				ps135.OverlayValues[29] = d29
				ps135.OverlayValues[30] = d30
				ps135.OverlayValues[31] = d31
				ps135.OverlayValues[32] = d32
				ps135.OverlayValues[33] = d33
				ps135.OverlayValues[34] = d34
				ps135.OverlayValues[35] = d35
				ps135.OverlayValues[76] = d76
				ps135.OverlayValues[77] = d77
				ps135.OverlayValues[78] = d78
				ps135.OverlayValues[79] = d79
				ps135.OverlayValues[80] = d80
				ps135.OverlayValues[81] = d81
				ps135.OverlayValues[82] = d82
				ps135.OverlayValues[83] = d83
				ps135.OverlayValues[132] = d132
				ps135.OverlayValues[133] = d133
				ps135.OverlayValues[134] = d134
				return bbs[1].RenderPS(ps135)
			}
			if ps.General {
			}
			ps136 := scm.PhiState{General: ps.General}
			ps136.OverlayValues = make([]scm.JITValueDesc, 135)
			ps136.OverlayValues[0] = d0
			ps136.OverlayValues[2] = d2
			ps136.OverlayValues[3] = d3
			ps136.OverlayValues[4] = d4
			ps136.OverlayValues[5] = d5
			ps136.OverlayValues[6] = d6
			ps136.OverlayValues[7] = d7
			ps136.OverlayValues[8] = d8
			ps136.OverlayValues[9] = d9
			ps136.OverlayValues[10] = d10
			ps136.OverlayValues[11] = d11
			ps136.OverlayValues[12] = d12
			ps136.OverlayValues[13] = d13
			ps136.OverlayValues[14] = d14
			ps136.OverlayValues[15] = d15
			ps136.OverlayValues[16] = d16
			ps136.OverlayValues[17] = d17
			ps136.OverlayValues[18] = d18
			ps136.OverlayValues[19] = d19
			ps136.OverlayValues[20] = d20
			ps136.OverlayValues[21] = d21
			ps136.OverlayValues[22] = d22
			ps136.OverlayValues[23] = d23
			ps136.OverlayValues[24] = d24
			ps136.OverlayValues[25] = d25
			ps136.OverlayValues[26] = d26
			ps136.OverlayValues[27] = d27
			ps136.OverlayValues[28] = d28
			ps136.OverlayValues[29] = d29
			ps136.OverlayValues[30] = d30
			ps136.OverlayValues[31] = d31
			ps136.OverlayValues[32] = d32
			ps136.OverlayValues[33] = d33
			ps136.OverlayValues[34] = d34
			ps136.OverlayValues[35] = d35
			ps136.OverlayValues[76] = d76
			ps136.OverlayValues[77] = d77
			ps136.OverlayValues[78] = d78
			ps136.OverlayValues[79] = d79
			ps136.OverlayValues[80] = d80
			ps136.OverlayValues[81] = d81
			ps136.OverlayValues[82] = d82
			ps136.OverlayValues[83] = d83
			ps136.OverlayValues[132] = d132
			ps136.OverlayValues[133] = d133
			ps136.OverlayValues[134] = d134
			return bbs[2].RenderPS(ps136)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d134.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl3)
		ps137 := scm.PhiState{General: true}
		ps137.OverlayValues = make([]scm.JITValueDesc, 135)
		ps137.OverlayValues[0] = d0
		ps137.OverlayValues[2] = d2
		ps137.OverlayValues[3] = d3
		ps137.OverlayValues[4] = d4
		ps137.OverlayValues[5] = d5
		ps137.OverlayValues[6] = d6
		ps137.OverlayValues[7] = d7
		ps137.OverlayValues[8] = d8
		ps137.OverlayValues[9] = d9
		ps137.OverlayValues[10] = d10
		ps137.OverlayValues[11] = d11
		ps137.OverlayValues[12] = d12
		ps137.OverlayValues[13] = d13
		ps137.OverlayValues[14] = d14
		ps137.OverlayValues[15] = d15
		ps137.OverlayValues[16] = d16
		ps137.OverlayValues[17] = d17
		ps137.OverlayValues[18] = d18
		ps137.OverlayValues[19] = d19
		ps137.OverlayValues[20] = d20
		ps137.OverlayValues[21] = d21
		ps137.OverlayValues[22] = d22
		ps137.OverlayValues[23] = d23
		ps137.OverlayValues[24] = d24
		ps137.OverlayValues[25] = d25
		ps137.OverlayValues[26] = d26
		ps137.OverlayValues[27] = d27
		ps137.OverlayValues[28] = d28
		ps137.OverlayValues[29] = d29
		ps137.OverlayValues[30] = d30
		ps137.OverlayValues[31] = d31
		ps137.OverlayValues[32] = d32
		ps137.OverlayValues[33] = d33
		ps137.OverlayValues[34] = d34
		ps137.OverlayValues[35] = d35
		ps137.OverlayValues[76] = d76
		ps137.OverlayValues[77] = d77
		ps137.OverlayValues[78] = d78
		ps137.OverlayValues[79] = d79
		ps137.OverlayValues[80] = d80
		ps137.OverlayValues[81] = d81
		ps137.OverlayValues[82] = d82
		ps137.OverlayValues[83] = d83
		ps137.OverlayValues[132] = d132
		ps137.OverlayValues[133] = d133
		ps137.OverlayValues[134] = d134
		ps138 := scm.PhiState{General: true}
		ps138.OverlayValues = make([]scm.JITValueDesc, 135)
		ps138.OverlayValues[0] = d0
		ps138.OverlayValues[2] = d2
		ps138.OverlayValues[3] = d3
		ps138.OverlayValues[4] = d4
		ps138.OverlayValues[5] = d5
		ps138.OverlayValues[6] = d6
		ps138.OverlayValues[7] = d7
		ps138.OverlayValues[8] = d8
		ps138.OverlayValues[9] = d9
		ps138.OverlayValues[10] = d10
		ps138.OverlayValues[11] = d11
		ps138.OverlayValues[12] = d12
		ps138.OverlayValues[13] = d13
		ps138.OverlayValues[14] = d14
		ps138.OverlayValues[15] = d15
		ps138.OverlayValues[16] = d16
		ps138.OverlayValues[17] = d17
		ps138.OverlayValues[18] = d18
		ps138.OverlayValues[19] = d19
		ps138.OverlayValues[20] = d20
		ps138.OverlayValues[21] = d21
		ps138.OverlayValues[22] = d22
		ps138.OverlayValues[23] = d23
		ps138.OverlayValues[24] = d24
		ps138.OverlayValues[25] = d25
		ps138.OverlayValues[26] = d26
		ps138.OverlayValues[27] = d27
		ps138.OverlayValues[28] = d28
		ps138.OverlayValues[29] = d29
		ps138.OverlayValues[30] = d30
		ps138.OverlayValues[31] = d31
		ps138.OverlayValues[32] = d32
		ps138.OverlayValues[33] = d33
		ps138.OverlayValues[34] = d34
		ps138.OverlayValues[35] = d35
		ps138.OverlayValues[76] = d76
		ps138.OverlayValues[77] = d77
		ps138.OverlayValues[78] = d78
		ps138.OverlayValues[79] = d79
		ps138.OverlayValues[80] = d80
		ps138.OverlayValues[81] = d81
		ps138.OverlayValues[82] = d82
		ps138.OverlayValues[83] = d83
		ps138.OverlayValues[132] = d132
		ps138.OverlayValues[133] = d133
		ps138.OverlayValues[134] = d134
		snap139 := d0
		snap140 := d2
		snap141 := d3
		snap142 := d4
		snap143 := d5
		snap144 := d6
		snap145 := d7
		snap146 := d8
		snap147 := d9
		snap148 := d10
		snap149 := d11
		snap150 := d12
		snap151 := d13
		snap152 := d14
		snap153 := d15
		snap154 := d16
		snap155 := d17
		snap156 := d18
		snap157 := d19
		snap158 := d20
		snap159 := d21
		snap160 := d22
		snap161 := d23
		snap162 := d24
		snap163 := d25
		snap164 := d26
		snap165 := d27
		snap166 := d28
		snap167 := d29
		snap168 := d30
		snap169 := d31
		snap170 := d32
		snap171 := d33
		snap172 := d34
		snap173 := d35
		snap174 := d76
		snap175 := d77
		snap176 := d78
		snap177 := d79
		snap178 := d80
		snap179 := d81
		snap180 := d82
		snap181 := d83
		snap182 := d132
		snap183 := d133
		snap184 := d134
		alloc185 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps138)
		}
		ctx.RestoreAllocState(alloc185)
		d0 = snap139
		d2 = snap140
		d3 = snap141
		d4 = snap142
		d5 = snap143
		d6 = snap144
		d7 = snap145
		d8 = snap146
		d9 = snap147
		d10 = snap148
		d11 = snap149
		d12 = snap150
		d13 = snap151
		d14 = snap152
		d15 = snap153
		d16 = snap154
		d17 = snap155
		d18 = snap156
		d19 = snap157
		d20 = snap158
		d21 = snap159
		d22 = snap160
		d23 = snap161
		d24 = snap162
		d25 = snap163
		d26 = snap164
		d27 = snap165
		d28 = snap166
		d29 = snap167
		d30 = snap168
		d31 = snap169
		d32 = snap170
		d33 = snap171
		d34 = snap172
		d35 = snap173
		d76 = snap174
		d77 = snap175
		d78 = snap176
		d79 = snap177
		d80 = snap178
		d81 = snap179
		d82 = snap180
		d83 = snap181
		d132 = snap182
		d133 = snap183
		d134 = snap184
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps137)
		}
		return result
		ctx.FreeDesc(&d133)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d81)
		r52 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r52, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r53 := ctx.AllocReg()
		if d81.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r53, uint64(d81.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r53, d81.Reg)
			ctx.EmitShlRegImm8(r53, 3)
		}
		ctx.EmitAddInt64(r52, r53)
		ctx.FreeReg(r53)
		r54 := ctx.AllocRegExcept(r52)
		ctx.EmitMovRegMem(r54, r52, 0)
		ctx.FreeReg(r52)
		d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
		ctx.BindReg(r54, &d186)
		ctx.EnsureDesc(&d80)
		ctx.EnsureDesc(&d186)
		ctx.EnsureDesc(&d80)
		ctx.ProtectReg(d80.Reg)
		ctx.EnsureDesc(&d186)
		ctx.UnprotectReg(d80.Reg)
		var d187 scm.JITValueDesc
		if d80.Loc == scm.LocImm && d186.Loc == scm.LocImm {
			d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d80.Imm.Int() * d186.Imm.Int())}
		} else if d80.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d186.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d80.Imm.Int()))
			ctx.EmitImulInt64(scratch, d186.Reg)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d187)
		} else if d186.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d80.Reg)
			ctx.EmitMovRegReg(scratch, d80.Reg)
			if d186.Imm.Int() >= -2147483648 && d186.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d186.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d187)
		} else {
			r55 := ctx.AllocRegExcept(d80.Reg, d186.Reg)
			ctx.EmitMovRegReg(r55, d80.Reg)
			ctx.EmitImulInt64(r55, d186.Reg)
			d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
			ctx.BindReg(r55, &d187)
		}
		if d187.Loc == scm.LocReg && d80.Loc == scm.LocReg && d187.Reg == d80.Reg {
			ctx.TransferReg(d80.Reg)
			d80.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d186)
		ctx.EnsureDesc(&d187)
		d188 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d188)
		ctx.BindReg(r1, &d188)
		ctx.EnsureDesc(&d187)
		ctx.EmitMakeInt(d188, d187)
		if d187.Loc == scm.LocReg {
			ctx.FreeReg(d187.Reg)
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
		if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
			d34 = ps.OverlayValues[34]
		}
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
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
		if len(ps.OverlayValues) > 132 && ps.OverlayValues[132].Loc != scm.LocNone {
			d132 = ps.OverlayValues[132]
		}
		if len(ps.OverlayValues) > 133 && ps.OverlayValues[133].Loc != scm.LocNone {
			d133 = ps.OverlayValues[133]
		}
		if len(ps.OverlayValues) > 134 && ps.OverlayValues[134].Loc != scm.LocNone {
			d134 = ps.OverlayValues[134]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d80)
		ctx.EnsureDesc(&d80)
		var d189 scm.JITValueDesc
		if d80.Loc == scm.LocImm {
			d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d80.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d80.Reg)
			d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d80.Reg}
			ctx.BindReg(d80.Reg, &d189)
		}
		ctx.FreeDesc(&d80)
		ctx.EnsureDesc(&d81)
		ctx.EnsureDesc(&d81)
		var d190 scm.JITValueDesc
		if d81.Loc == scm.LocImm {
			d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d81.Imm.Int()))))}
		} else {
			r56 := ctx.AllocReg()
			ctx.EmitMovRegReg(r56, d81.Reg)
			ctx.EmitShlRegImm8(r56, 56)
			ctx.EmitSarRegImm8(r56, 56)
			d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			ctx.BindReg(r56, &d190)
		}
		ctx.EnsureDesc(&d190)
		ctx.EnsureDesc(&d190)
		var d191 scm.JITValueDesc
		if d190.Loc == scm.LocImm {
			d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d190.Reg)
			ctx.EmitMovRegReg(scratch, d190.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d191)
		}
		if d191.Loc == scm.LocReg && d190.Loc == scm.LocReg && d191.Reg == d190.Reg {
			ctx.TransferReg(d190.Reg)
			d190.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d190)
		ctx.EnsureDesc(&d191)
		r57 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r57, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r58 := ctx.AllocReg()
		if d191.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r58, uint64(d191.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r58, d191.Reg)
			ctx.EmitShlRegImm8(r58, 3)
		}
		ctx.EmitAddInt64(r57, r58)
		ctx.FreeReg(r58)
		r59 := ctx.AllocRegExcept(r57)
		ctx.EmitMovRegMem(r59, r57, 0)
		ctx.FreeReg(r57)
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
		ctx.BindReg(r59, &d192)
		ctx.FreeDesc(&d191)
		ctx.EnsureDesc(&d189)
		ctx.EnsureDesc(&d192)
		ctx.EnsureDesc(&d189)
		ctx.EnsureDesc(&d192)
		var d193 scm.JITValueDesc
		if d189.Loc == scm.LocImm && d192.Loc == scm.LocImm {
			d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d189.Imm.Float() * d192.Imm.Float())}
		} else if d189.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d192.Reg)
			_, xBits := d189.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d192.Reg)
			d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d193)
		} else if d192.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d189.Reg)
			ctx.EmitMovRegReg(scratch, d189.Reg)
			_, yBits := d192.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d193)
		} else {
			r60 := ctx.AllocRegExcept(d189.Reg, d192.Reg)
			ctx.EmitMovRegReg(r60, d189.Reg)
			ctx.EmitMulFloat64(r60, d192.Reg)
			d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r60}
			ctx.BindReg(r60, &d193)
		}
		if d193.Loc == scm.LocReg && d189.Loc == scm.LocReg && d193.Reg == d189.Reg {
			ctx.TransferReg(d189.Reg)
			d189.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d189)
		ctx.FreeDesc(&d192)
		ctx.EnsureDesc(&d193)
		d194 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d194)
		ctx.BindReg(r1, &d194)
		ctx.EnsureDesc(&d193)
		ctx.EmitMakeFloat(d194, d193)
		if d193.Loc == scm.LocReg {
			ctx.FreeReg(d193.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps195 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps195)
	ctx.MarkLabel(lbl0)
	d196 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d196)
	ctx.BindReg(r1, &d196)
	ctx.EmitMovPairToResult(&d196, &result)
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
