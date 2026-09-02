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
	var d98 scm.JITValueDesc
	_ = d98
	var d99 scm.JITValueDesc
	_ = d99
	var d100 scm.JITValueDesc
	_ = d100
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
	var d148 scm.JITValueDesc
	_ = d148
	var d149 scm.JITValueDesc
	_ = d149
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	thisptrPinned := thisptr.Loc == scm.LocReg
	thisptrPinnedReg := thisptr.Reg
	if thisptrPinned {
		ctx.ProtectReg(thisptrPinnedReg)
	}
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
	resultRegsProtected := result.Loc == scm.LocRegPair
	if resultRegsProtected {
		ctx.ProtectReg(result.Reg)
		ctx.ProtectReg(result.Reg2)
	}
	r0 := ctx.AllocReg()
	r1 := ctx.AllocRegExcept(r0)
	lbl0 := ctx.ReserveLabel()
	bbpos_0_0 := int32(-1)
	_ = bbpos_0_0
	lbl1 := ctx.ReserveLabel()
	_ = lbl1
	bbpos_0_1 := int32(-1)
	_ = bbpos_0_1
	lbl2 := ctx.ReserveLabel()
	_ = lbl2
	bbpos_0_2 := int32(-1)
	_ = bbpos_0_2
	lbl3 := ctx.ReserveLabel()
	_ = lbl3
	bbpos_0_3 := int32(-1)
	_ = bbpos_0_3
	lbl4 := ctx.ReserveLabel()
	_ = lbl4
	bbpos_0_4 := int32(-1)
	_ = bbpos_0_4
	lbl5 := ctx.ReserveLabel()
	_ = lbl5
	bbpos_0_5 := int32(-1)
	_ = bbpos_0_5
	lbl6 := ctx.ReserveLabel()
	_ = lbl6
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
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl7 := ctx.ReserveLabel()
		_ = lbl7
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		var d1 scm.JITValueDesc
		if d0.Loc == scm.LocImm {
			d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d0.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d1)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d2 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r3, fieldAddr)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d2)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
			r4 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r4, thisptr.Reg, off)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			ctx.BindReg(r4, &d2)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d2)
		var d3 scm.JITValueDesc
		if d2.Loc == scm.LocImm {
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
		} else {
			r5 := ctx.AllocReg()
			ctx.EmitMovRegReg(r5, d2.Reg)
			ctx.EmitShlRegImm8(r5, 56)
			ctx.EmitShrRegImm8(r5, 56)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d3)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d3)
		ctx.EnsureDescsTogether(&d1, &d3)
		var d4 scm.JITValueDesc
		if d1.Loc == scm.LocImm && d3.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() * d3.Imm.Int())}
		} else if d1.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d3.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
			ctx.EmitImulInt64(scratch, d3.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else if d3.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d1.Reg)
			ctx.EmitMovRegReg(scratch, d1.Reg)
			if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d3.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else {
			r6 := ctx.AllocRegExcept(d1.Reg, d3.Reg)
			ctx.EmitMovRegReg(r6, d1.Reg)
			ctx.EmitImulInt64(r6, d3.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d4)
		}
		if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
			ctx.TransferReg(d1.Reg)
			d1.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d1)
		ctx.FreeDesc(&d3)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		var d5 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
		} else {
			r7 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r7, d4.Reg)
			ctx.EmitShrRegImm8(r7, 6)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d5)
		}
		if d5.Loc == scm.LocReg && d4.Loc == scm.LocReg && d5.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		var d6 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
		} else {
			r8 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r8, d4.Reg)
			ctx.EmitAndRegImm32(r8, 63)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d6)
		}
		if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d4)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d7 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0
			r9 := ctx.AllocReg()
			r10 := ctx.AllocRegExcept(r9)
			r11 := ctx.AllocRegExcept(r9, r10)
			ctx.EmitMovRegMem64(r9, fieldAddr)
			ctx.EmitMovRegMem64(r10, fieldAddr+8)
			ctx.EmitMovRegMem64(r11, fieldAddr+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r9, Reg2: r10, Reg3: r11}
			ctx.BindReg(r9, &d7)
			ctx.BindReg(r10, &d7)
			ctx.BindReg(r11, &d7)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 0)
			r12 := ctx.AllocReg()
			r13 := ctx.AllocRegExcept(r12)
			r14 := ctx.AllocRegExcept(r12, r13)
			ctx.EmitMovRegMem(r12, thisptr.Reg, off)
			ctx.EmitMovRegMem(r13, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r14, thisptr.Reg, off+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
			ctx.BindReg(r12, &d7)
			ctx.BindReg(r13, &d7)
			ctx.BindReg(r14, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		d9 = ctx.EmitSliceElementAddress(&d7, &d5, 8)
		ctx.EnsureDesc(&d9)
		ctx.EmitMovRegMem(d9.Reg, d9.Reg, 0)
		d8 = d9
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d6)
		var d10 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d6.Imm.Int())))}
		} else if d6.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r15, d8.Reg)
			ctx.EmitShlRegImm8(r15, uint8(d6.Imm.Int()))
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d10)
		} else {
			{
				shiftSrc := d8.Reg
				r16 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r16, d8.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d6.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d6.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d6.Reg)
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
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		var d11 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d11)
		}
		if d11.Loc == scm.LocReg && d5.Loc == scm.LocReg && d11.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		d13 = ctx.EmitSliceElementAddress(&d7, &d11, 8)
		ctx.EnsureDesc(&d13)
		ctx.EmitMovRegMem(d13.Reg, d13.Reg, 0)
		d12 = d13
		ctx.FreeDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d14, &d6)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d6.Imm.Int())}
		} else if d6.Loc == scm.LocImm && d6.Imm.Int() == 0 {
			r17 := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(r17, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d15)
		} else if d14.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
			ctx.EmitSubInt64(scratch, d6.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d6.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(scratch, d14.Reg)
			if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d6.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else {
			r18 := ctx.AllocRegExcept(d14.Reg, d6.Reg)
			ctx.EmitMovRegReg(r18, d14.Reg)
			ctx.EmitSubInt64(r18, d6.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d15)
		}
		if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
			ctx.TransferReg(d14.Reg)
			d14.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d15)
		var d16 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d15.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) >> uint64(d15.Imm.Int())))}
		} else if d15.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r19, d12.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d15.Imm.Int()))
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d16)
		} else {
			{
				shiftSrc := d12.Reg
				r20 := ctx.AllocRegExcept(d12.Reg)
				ctx.EmitMovRegReg(r20, d12.Reg)
				shiftSrc = r20
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d15.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d15.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d15.Reg)
				}
				ctx.EmitShrRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d16)
			}
		}
		if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d12)
		ctx.FreeDesc(&d15)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.EnsureDesc(&d16)
		var d17 scm.JITValueDesc
		if d10.Loc == scm.LocImm && d16.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d16.Imm.Int())}
		} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			ctx.BindReg(d16.Reg, &d17)
		} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
			r21 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r21, d10.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d17)
		} else if d10.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
			ctx.EmitOrInt64(scratch, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d16.Loc == scm.LocImm {
			r22 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r22, d10.Reg)
			if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r22, int32(d16.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
				ctx.EmitOrInt64(r22, scm.RegR11)
			}
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d17)
		} else {
			r23 := ctx.AllocRegExcept(d10.Reg, d16.Reg)
			ctx.EmitMovRegReg(r23, d10.Reg)
			ctx.EmitOrInt64(r23, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d17)
		}
		if d17.Loc == scm.LocReg && d10.Loc == scm.LocReg && d17.Reg == d10.Reg {
			ctx.TransferReg(d10.Reg)
			d10.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d10)
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d2)
		var d18 scm.JITValueDesc
		if d2.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d2.Reg)
			ctx.EmitShlRegImm8(r24, 56)
			ctx.EmitShrRegImm8(r24, 56)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d18)
		}
		ctx.ReclaimUntrackedRegs()
		d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDescsTogether(&d19, &d18)
		var d20 scm.JITValueDesc
		if d19.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() - d18.Imm.Int())}
		} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d19.Reg)
			ctx.EmitMovRegReg(r25, d19.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d20)
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
			r26 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
			ctx.EmitMovRegReg(r26, d19.Reg)
			ctx.EmitSubInt64(r26, d18.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d20)
		}
		if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
			ctx.TransferReg(d19.Reg)
			d19.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d20)
		var d21 scm.JITValueDesc
		if d17.Loc == scm.LocImm && d20.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d20.Imm.Int())))}
		} else if d20.Loc == scm.LocImm {
			r27 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r27, d17.Reg)
			ctx.EmitShrRegImm8(r27, uint8(d20.Imm.Int()))
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d21)
		} else {
			{
				shiftSrc := d17.Reg
				r28 := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(r28, d17.Reg)
				shiftSrc = r28
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
		if d21.Loc == scm.LocReg && d17.Loc == scm.LocReg && d21.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.FreeDesc(&d20)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.StabilizeDescForControlFlow(&d21)
		ctx.FreeDesc(&idxInt)
		var d22 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			r29 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r29, fieldAddr)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d22)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r30, thisptr.Reg, off)
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d22)
		}
		d23 = d22
		ctx.EnsureDesc(&d23)
		if d23.Loc != scm.LocImm && d23.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d23.Loc == scm.LocImm {
			if d23.Imm.Bool() {
				if ps.General {
				}
				ps24 := scm.PhiState{General: ps.General}
				ps24.OverlayValues = make([]scm.JITValueDesc, 24)
				ps24.OverlayValues[0] = d0
				ps24.OverlayValues[1] = d1
				ps24.OverlayValues[2] = d2
				ps24.OverlayValues[3] = d3
				ps24.OverlayValues[4] = d4
				ps24.OverlayValues[5] = d5
				ps24.OverlayValues[6] = d6
				ps24.OverlayValues[7] = d7
				ps24.OverlayValues[8] = d8
				ps24.OverlayValues[9] = d9
				ps24.OverlayValues[10] = d10
				ps24.OverlayValues[11] = d11
				ps24.OverlayValues[12] = d12
				ps24.OverlayValues[13] = d13
				ps24.OverlayValues[14] = d14
				ps24.OverlayValues[15] = d15
				ps24.OverlayValues[16] = d16
				ps24.OverlayValues[17] = d17
				ps24.OverlayValues[18] = d18
				ps24.OverlayValues[19] = d19
				ps24.OverlayValues[20] = d20
				ps24.OverlayValues[21] = d21
				ps24.OverlayValues[22] = d22
				ps24.OverlayValues[23] = d23
				return bbs[3].RenderPS(ps24)
			}
			if ps.General {
			}
			ps25 := scm.PhiState{General: ps.General}
			ps25.OverlayValues = make([]scm.JITValueDesc, 24)
			ps25.OverlayValues[0] = d0
			ps25.OverlayValues[1] = d1
			ps25.OverlayValues[2] = d2
			ps25.OverlayValues[3] = d3
			ps25.OverlayValues[4] = d4
			ps25.OverlayValues[5] = d5
			ps25.OverlayValues[6] = d6
			ps25.OverlayValues[7] = d7
			ps25.OverlayValues[8] = d8
			ps25.OverlayValues[9] = d9
			ps25.OverlayValues[10] = d10
			ps25.OverlayValues[11] = d11
			ps25.OverlayValues[12] = d12
			ps25.OverlayValues[13] = d13
			ps25.OverlayValues[14] = d14
			ps25.OverlayValues[15] = d15
			ps25.OverlayValues[16] = d16
			ps25.OverlayValues[17] = d17
			ps25.OverlayValues[18] = d18
			ps25.OverlayValues[19] = d19
			ps25.OverlayValues[20] = d20
			ps25.OverlayValues[21] = d21
			ps25.OverlayValues[22] = d22
			ps25.OverlayValues[23] = d23
			return bbs[2].RenderPS(ps25)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl8 := ctx.ReserveLabel()
		lbl9 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d23.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl8)
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl8)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl3)
		ps26 := scm.PhiState{General: true}
		ps26.OverlayValues = make([]scm.JITValueDesc, 24)
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
		ps26.OverlayValues[15] = d15
		ps26.OverlayValues[16] = d16
		ps26.OverlayValues[17] = d17
		ps26.OverlayValues[18] = d18
		ps26.OverlayValues[19] = d19
		ps26.OverlayValues[20] = d20
		ps26.OverlayValues[21] = d21
		ps26.OverlayValues[22] = d22
		ps26.OverlayValues[23] = d23
		ps27 := scm.PhiState{General: true}
		ps27.OverlayValues = make([]scm.JITValueDesc, 24)
		ps27.OverlayValues[0] = d0
		ps27.OverlayValues[1] = d1
		ps27.OverlayValues[2] = d2
		ps27.OverlayValues[3] = d3
		ps27.OverlayValues[4] = d4
		ps27.OverlayValues[5] = d5
		ps27.OverlayValues[6] = d6
		ps27.OverlayValues[7] = d7
		ps27.OverlayValues[8] = d8
		ps27.OverlayValues[9] = d9
		ps27.OverlayValues[10] = d10
		ps27.OverlayValues[11] = d11
		ps27.OverlayValues[12] = d12
		ps27.OverlayValues[13] = d13
		ps27.OverlayValues[14] = d14
		ps27.OverlayValues[15] = d15
		ps27.OverlayValues[16] = d16
		ps27.OverlayValues[17] = d17
		ps27.OverlayValues[18] = d18
		ps27.OverlayValues[19] = d19
		ps27.OverlayValues[20] = d20
		ps27.OverlayValues[21] = d21
		ps27.OverlayValues[22] = d22
		ps27.OverlayValues[23] = d23
		snap28 := d0
		snap29 := d1
		snap30 := d2
		snap31 := d3
		snap32 := d4
		snap33 := d5
		snap34 := d6
		snap35 := d7
		snap36 := d8
		snap37 := d9
		snap38 := d10
		snap39 := d11
		snap40 := d12
		snap41 := d13
		snap42 := d14
		snap43 := d15
		snap44 := d16
		snap45 := d17
		snap46 := d18
		snap47 := d19
		snap48 := d20
		snap49 := d21
		snap50 := d22
		snap51 := d23
		alloc52 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps27)
		}
		ctx.RestoreAllocState(alloc52)
		d0 = snap28
		d1 = snap29
		d2 = snap30
		d3 = snap31
		d4 = snap32
		d5 = snap33
		d6 = snap34
		d7 = snap35
		d8 = snap36
		d9 = snap37
		d10 = snap38
		d11 = snap39
		d12 = snap40
		d13 = snap41
		d14 = snap42
		d15 = snap43
		d16 = snap44
		d17 = snap45
		d18 = snap46
		d19 = snap47
		d20 = snap48
		d21 = snap49
		d22 = snap50
		d23 = snap51
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps26)
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
		ctx.ReclaimUntrackedRegs()
		d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d54 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d54)
		ctx.BindReg(r1, &d54)
		ctx.EnsureDesc(&d53)
		if d53.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d53, &d54)
		} else {
			switch d53.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d54, d53)
			case scm.TagInt:
				ctx.EmitMakeInt(d54, d53)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d54, d53)
			case scm.TagNil:
				ctx.EmitMakeNil(d54)
			default:
				ctx.EmitMovPairToResult(&d53, &d54)
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
		if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
			d53 = ps.OverlayValues[53]
		}
		if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
			d54 = ps.OverlayValues[54]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d21)
		var d55 scm.JITValueDesc
		if d21.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d21.Imm.Int()))))}
		} else {
			r31 := ctx.AllocReg()
			ctx.EmitMovRegReg(r31, d21.Reg)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			ctx.BindReg(r31, &d55)
		}
		var d56 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32
			r32 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r32, fieldAddr)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d56)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 32)
			r33 := ctx.AllocReg()
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d56)
		}
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d56)
		ctx.EnsureDescsTogether(&d55, &d56)
		var d57 scm.JITValueDesc
		if d55.Loc == scm.LocImm && d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() + d56.Imm.Int())}
		} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
			r34 := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(r34, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			ctx.BindReg(r34, &d57)
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
			ctx.BindReg(d56.Reg, &d57)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
			ctx.EmitAddInt64(scratch, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d56.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegReg(scratch, d55.Reg)
			if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d56.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else {
			r35 := ctx.AllocRegExcept(d55.Reg, d56.Reg)
			ctx.EmitMovRegReg(r35, d55.Reg)
			ctx.EmitAddInt64(r35, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			ctx.BindReg(r35, &d57)
		}
		if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
			ctx.TransferReg(d55.Reg)
			d55.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d57)
		ctx.FreeDesc(&d55)
		var d58 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			r36 := ctx.AllocReg()
			ctx.EmitMovRegMem8(r36, fieldAddr)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			ctx.BindReg(r36, &d58)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r37, thisptr.Reg, off)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d58)
		}
		ctx.EnsureDesc(&d58)
		var d59 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d58.Imm.Int() > 0)}
		} else {
			r38 := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitCmpRegImm32(d58.Reg, 0)
			ctx.EmitSetcc(r38, scm.CondSignedGreater)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
			ctx.BindReg(r38, &d59)
		}
		d60 = d59
		ctx.EnsureDesc(&d60)
		if d60.Loc != scm.LocImm && d60.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d60.Loc == scm.LocImm {
			if d60.Imm.Bool() {
				if ps.General {
				}
				ps61 := scm.PhiState{General: ps.General}
				ps61.OverlayValues = make([]scm.JITValueDesc, 61)
				ps61.OverlayValues[0] = d0
				ps61.OverlayValues[1] = d1
				ps61.OverlayValues[2] = d2
				ps61.OverlayValues[3] = d3
				ps61.OverlayValues[4] = d4
				ps61.OverlayValues[5] = d5
				ps61.OverlayValues[6] = d6
				ps61.OverlayValues[7] = d7
				ps61.OverlayValues[8] = d8
				ps61.OverlayValues[9] = d9
				ps61.OverlayValues[10] = d10
				ps61.OverlayValues[11] = d11
				ps61.OverlayValues[12] = d12
				ps61.OverlayValues[13] = d13
				ps61.OverlayValues[14] = d14
				ps61.OverlayValues[15] = d15
				ps61.OverlayValues[16] = d16
				ps61.OverlayValues[17] = d17
				ps61.OverlayValues[18] = d18
				ps61.OverlayValues[19] = d19
				ps61.OverlayValues[20] = d20
				ps61.OverlayValues[21] = d21
				ps61.OverlayValues[22] = d22
				ps61.OverlayValues[23] = d23
				ps61.OverlayValues[53] = d53
				ps61.OverlayValues[54] = d54
				ps61.OverlayValues[55] = d55
				ps61.OverlayValues[56] = d56
				ps61.OverlayValues[57] = d57
				ps61.OverlayValues[58] = d58
				ps61.OverlayValues[59] = d59
				ps61.OverlayValues[60] = d60
				return bbs[4].RenderPS(ps61)
			}
			if ps.General {
			}
			ps62 := scm.PhiState{General: ps.General}
			ps62.OverlayValues = make([]scm.JITValueDesc, 61)
			ps62.OverlayValues[0] = d0
			ps62.OverlayValues[1] = d1
			ps62.OverlayValues[2] = d2
			ps62.OverlayValues[3] = d3
			ps62.OverlayValues[4] = d4
			ps62.OverlayValues[5] = d5
			ps62.OverlayValues[6] = d6
			ps62.OverlayValues[7] = d7
			ps62.OverlayValues[8] = d8
			ps62.OverlayValues[9] = d9
			ps62.OverlayValues[10] = d10
			ps62.OverlayValues[11] = d11
			ps62.OverlayValues[12] = d12
			ps62.OverlayValues[13] = d13
			ps62.OverlayValues[14] = d14
			ps62.OverlayValues[15] = d15
			ps62.OverlayValues[16] = d16
			ps62.OverlayValues[17] = d17
			ps62.OverlayValues[18] = d18
			ps62.OverlayValues[19] = d19
			ps62.OverlayValues[20] = d20
			ps62.OverlayValues[21] = d21
			ps62.OverlayValues[22] = d22
			ps62.OverlayValues[23] = d23
			ps62.OverlayValues[53] = d53
			ps62.OverlayValues[54] = d54
			ps62.OverlayValues[55] = d55
			ps62.OverlayValues[56] = d56
			ps62.OverlayValues[57] = d57
			ps62.OverlayValues[58] = d58
			ps62.OverlayValues[59] = d59
			ps62.OverlayValues[60] = d60
			return bbs[5].RenderPS(ps62)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl10 := ctx.ReserveLabel()
		lbl11 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d60.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl10)
		ctx.EmitJmp(lbl11)
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl6)
		ps63 := scm.PhiState{General: true}
		ps63.OverlayValues = make([]scm.JITValueDesc, 61)
		ps63.OverlayValues[0] = d0
		ps63.OverlayValues[1] = d1
		ps63.OverlayValues[2] = d2
		ps63.OverlayValues[3] = d3
		ps63.OverlayValues[4] = d4
		ps63.OverlayValues[5] = d5
		ps63.OverlayValues[6] = d6
		ps63.OverlayValues[7] = d7
		ps63.OverlayValues[8] = d8
		ps63.OverlayValues[9] = d9
		ps63.OverlayValues[10] = d10
		ps63.OverlayValues[11] = d11
		ps63.OverlayValues[12] = d12
		ps63.OverlayValues[13] = d13
		ps63.OverlayValues[14] = d14
		ps63.OverlayValues[15] = d15
		ps63.OverlayValues[16] = d16
		ps63.OverlayValues[17] = d17
		ps63.OverlayValues[18] = d18
		ps63.OverlayValues[19] = d19
		ps63.OverlayValues[20] = d20
		ps63.OverlayValues[21] = d21
		ps63.OverlayValues[22] = d22
		ps63.OverlayValues[23] = d23
		ps63.OverlayValues[53] = d53
		ps63.OverlayValues[54] = d54
		ps63.OverlayValues[55] = d55
		ps63.OverlayValues[56] = d56
		ps63.OverlayValues[57] = d57
		ps63.OverlayValues[58] = d58
		ps63.OverlayValues[59] = d59
		ps63.OverlayValues[60] = d60
		ps64 := scm.PhiState{General: true}
		ps64.OverlayValues = make([]scm.JITValueDesc, 61)
		ps64.OverlayValues[0] = d0
		ps64.OverlayValues[1] = d1
		ps64.OverlayValues[2] = d2
		ps64.OverlayValues[3] = d3
		ps64.OverlayValues[4] = d4
		ps64.OverlayValues[5] = d5
		ps64.OverlayValues[6] = d6
		ps64.OverlayValues[7] = d7
		ps64.OverlayValues[8] = d8
		ps64.OverlayValues[9] = d9
		ps64.OverlayValues[10] = d10
		ps64.OverlayValues[11] = d11
		ps64.OverlayValues[12] = d12
		ps64.OverlayValues[13] = d13
		ps64.OverlayValues[14] = d14
		ps64.OverlayValues[15] = d15
		ps64.OverlayValues[16] = d16
		ps64.OverlayValues[17] = d17
		ps64.OverlayValues[18] = d18
		ps64.OverlayValues[19] = d19
		ps64.OverlayValues[20] = d20
		ps64.OverlayValues[21] = d21
		ps64.OverlayValues[22] = d22
		ps64.OverlayValues[23] = d23
		ps64.OverlayValues[53] = d53
		ps64.OverlayValues[54] = d54
		ps64.OverlayValues[55] = d55
		ps64.OverlayValues[56] = d56
		ps64.OverlayValues[57] = d57
		ps64.OverlayValues[58] = d58
		ps64.OverlayValues[59] = d59
		ps64.OverlayValues[60] = d60
		snap65 := d0
		snap66 := d1
		snap67 := d2
		snap68 := d3
		snap69 := d4
		snap70 := d5
		snap71 := d6
		snap72 := d7
		snap73 := d8
		snap74 := d9
		snap75 := d10
		snap76 := d11
		snap77 := d12
		snap78 := d13
		snap79 := d14
		snap80 := d15
		snap81 := d16
		snap82 := d17
		snap83 := d18
		snap84 := d19
		snap85 := d20
		snap86 := d21
		snap87 := d22
		snap88 := d23
		snap89 := d53
		snap90 := d54
		snap91 := d55
		snap92 := d56
		snap93 := d57
		snap94 := d58
		snap95 := d59
		snap96 := d60
		alloc97 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps64)
		}
		ctx.RestoreAllocState(alloc97)
		d0 = snap65
		d1 = snap66
		d2 = snap67
		d3 = snap68
		d4 = snap69
		d5 = snap70
		d6 = snap71
		d7 = snap72
		d8 = snap73
		d9 = snap74
		d10 = snap75
		d11 = snap76
		d12 = snap77
		d13 = snap78
		d14 = snap79
		d15 = snap80
		d16 = snap81
		d17 = snap82
		d18 = snap83
		d19 = snap84
		d20 = snap85
		d21 = snap86
		d22 = snap87
		d23 = snap88
		d53 = snap89
		d54 = snap90
		d55 = snap91
		d56 = snap92
		d57 = snap93
		d58 = snap94
		d59 = snap95
		d60 = snap96
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps63)
		}
		return result
		ctx.FreeDesc(&d59)
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
		ctx.ReclaimUntrackedRegs()
		var d98 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64
			r39 := ctx.AllocReg()
			ctx.EmitMovRegMem64(r39, fieldAddr)
			d98 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d98)
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 64)
			r40 := ctx.AllocReg()
			ctx.EmitMovRegMem(r40, thisptr.Reg, off)
			d98 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d98)
		}
		ctx.EnsureDesc(&d21)
		ctx.EnsureDesc(&d98)
		ctx.EnsureDescsTogether(&d21, &d98)
		var d99 scm.JITValueDesc
		if d21.Loc == scm.LocImm && d98.Loc == scm.LocImm {
			d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d21.Imm.Int()) == uint64(d98.Imm.Int()))}
		} else if d98.Loc == scm.LocImm {
			r41 := ctx.AllocRegExcept(d21.Reg)
			if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d21.Reg, int32(d98.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
				ctx.EmitCmpInt64(d21.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r41, scm.CondEqual)
			d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r41}
			ctx.BindReg(r41, &d99)
		} else if d21.Loc == scm.LocImm {
			r42 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d98.Reg)
			ctx.EmitSetcc(r42, scm.CondEqual)
			d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r42}
			ctx.BindReg(r42, &d99)
		} else {
			r43 := ctx.AllocRegExcept(d21.Reg)
			ctx.EmitCmpInt64(d21.Reg, d98.Reg)
			ctx.EmitSetcc(r43, scm.CondEqual)
			d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r43}
			ctx.BindReg(r43, &d99)
		}
		ctx.FreeDesc(&d21)
		d100 = d99
		ctx.EnsureDesc(&d100)
		if d100.Loc != scm.LocImm && d100.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d100.Loc == scm.LocImm {
			if d100.Imm.Bool() {
				if ps.General {
				}
				ps101 := scm.PhiState{General: ps.General}
				ps101.OverlayValues = make([]scm.JITValueDesc, 101)
				ps101.OverlayValues[0] = d0
				ps101.OverlayValues[1] = d1
				ps101.OverlayValues[2] = d2
				ps101.OverlayValues[3] = d3
				ps101.OverlayValues[4] = d4
				ps101.OverlayValues[5] = d5
				ps101.OverlayValues[6] = d6
				ps101.OverlayValues[7] = d7
				ps101.OverlayValues[8] = d8
				ps101.OverlayValues[9] = d9
				ps101.OverlayValues[10] = d10
				ps101.OverlayValues[11] = d11
				ps101.OverlayValues[12] = d12
				ps101.OverlayValues[13] = d13
				ps101.OverlayValues[14] = d14
				ps101.OverlayValues[15] = d15
				ps101.OverlayValues[16] = d16
				ps101.OverlayValues[17] = d17
				ps101.OverlayValues[18] = d18
				ps101.OverlayValues[19] = d19
				ps101.OverlayValues[20] = d20
				ps101.OverlayValues[21] = d21
				ps101.OverlayValues[22] = d22
				ps101.OverlayValues[23] = d23
				ps101.OverlayValues[53] = d53
				ps101.OverlayValues[54] = d54
				ps101.OverlayValues[55] = d55
				ps101.OverlayValues[56] = d56
				ps101.OverlayValues[57] = d57
				ps101.OverlayValues[58] = d58
				ps101.OverlayValues[59] = d59
				ps101.OverlayValues[60] = d60
				ps101.OverlayValues[98] = d98
				ps101.OverlayValues[99] = d99
				ps101.OverlayValues[100] = d100
				return bbs[1].RenderPS(ps101)
			}
			if ps.General {
			}
			ps102 := scm.PhiState{General: ps.General}
			ps102.OverlayValues = make([]scm.JITValueDesc, 101)
			ps102.OverlayValues[0] = d0
			ps102.OverlayValues[1] = d1
			ps102.OverlayValues[2] = d2
			ps102.OverlayValues[3] = d3
			ps102.OverlayValues[4] = d4
			ps102.OverlayValues[5] = d5
			ps102.OverlayValues[6] = d6
			ps102.OverlayValues[7] = d7
			ps102.OverlayValues[8] = d8
			ps102.OverlayValues[9] = d9
			ps102.OverlayValues[10] = d10
			ps102.OverlayValues[11] = d11
			ps102.OverlayValues[12] = d12
			ps102.OverlayValues[13] = d13
			ps102.OverlayValues[14] = d14
			ps102.OverlayValues[15] = d15
			ps102.OverlayValues[16] = d16
			ps102.OverlayValues[17] = d17
			ps102.OverlayValues[18] = d18
			ps102.OverlayValues[19] = d19
			ps102.OverlayValues[20] = d20
			ps102.OverlayValues[21] = d21
			ps102.OverlayValues[22] = d22
			ps102.OverlayValues[23] = d23
			ps102.OverlayValues[53] = d53
			ps102.OverlayValues[54] = d54
			ps102.OverlayValues[55] = d55
			ps102.OverlayValues[56] = d56
			ps102.OverlayValues[57] = d57
			ps102.OverlayValues[58] = d58
			ps102.OverlayValues[59] = d59
			ps102.OverlayValues[60] = d60
			ps102.OverlayValues[98] = d98
			ps102.OverlayValues[99] = d99
			ps102.OverlayValues[100] = d100
			return bbs[2].RenderPS(ps102)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d100.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl3)
		ps103 := scm.PhiState{General: true}
		ps103.OverlayValues = make([]scm.JITValueDesc, 101)
		ps103.OverlayValues[0] = d0
		ps103.OverlayValues[1] = d1
		ps103.OverlayValues[2] = d2
		ps103.OverlayValues[3] = d3
		ps103.OverlayValues[4] = d4
		ps103.OverlayValues[5] = d5
		ps103.OverlayValues[6] = d6
		ps103.OverlayValues[7] = d7
		ps103.OverlayValues[8] = d8
		ps103.OverlayValues[9] = d9
		ps103.OverlayValues[10] = d10
		ps103.OverlayValues[11] = d11
		ps103.OverlayValues[12] = d12
		ps103.OverlayValues[13] = d13
		ps103.OverlayValues[14] = d14
		ps103.OverlayValues[15] = d15
		ps103.OverlayValues[16] = d16
		ps103.OverlayValues[17] = d17
		ps103.OverlayValues[18] = d18
		ps103.OverlayValues[19] = d19
		ps103.OverlayValues[20] = d20
		ps103.OverlayValues[21] = d21
		ps103.OverlayValues[22] = d22
		ps103.OverlayValues[23] = d23
		ps103.OverlayValues[53] = d53
		ps103.OverlayValues[54] = d54
		ps103.OverlayValues[55] = d55
		ps103.OverlayValues[56] = d56
		ps103.OverlayValues[57] = d57
		ps103.OverlayValues[58] = d58
		ps103.OverlayValues[59] = d59
		ps103.OverlayValues[60] = d60
		ps103.OverlayValues[98] = d98
		ps103.OverlayValues[99] = d99
		ps103.OverlayValues[100] = d100
		ps104 := scm.PhiState{General: true}
		ps104.OverlayValues = make([]scm.JITValueDesc, 101)
		ps104.OverlayValues[0] = d0
		ps104.OverlayValues[1] = d1
		ps104.OverlayValues[2] = d2
		ps104.OverlayValues[3] = d3
		ps104.OverlayValues[4] = d4
		ps104.OverlayValues[5] = d5
		ps104.OverlayValues[6] = d6
		ps104.OverlayValues[7] = d7
		ps104.OverlayValues[8] = d8
		ps104.OverlayValues[9] = d9
		ps104.OverlayValues[10] = d10
		ps104.OverlayValues[11] = d11
		ps104.OverlayValues[12] = d12
		ps104.OverlayValues[13] = d13
		ps104.OverlayValues[14] = d14
		ps104.OverlayValues[15] = d15
		ps104.OverlayValues[16] = d16
		ps104.OverlayValues[17] = d17
		ps104.OverlayValues[18] = d18
		ps104.OverlayValues[19] = d19
		ps104.OverlayValues[20] = d20
		ps104.OverlayValues[21] = d21
		ps104.OverlayValues[22] = d22
		ps104.OverlayValues[23] = d23
		ps104.OverlayValues[53] = d53
		ps104.OverlayValues[54] = d54
		ps104.OverlayValues[55] = d55
		ps104.OverlayValues[56] = d56
		ps104.OverlayValues[57] = d57
		ps104.OverlayValues[58] = d58
		ps104.OverlayValues[59] = d59
		ps104.OverlayValues[60] = d60
		ps104.OverlayValues[98] = d98
		ps104.OverlayValues[99] = d99
		ps104.OverlayValues[100] = d100
		snap105 := d0
		snap106 := d1
		snap107 := d2
		snap108 := d3
		snap109 := d4
		snap110 := d5
		snap111 := d6
		snap112 := d7
		snap113 := d8
		snap114 := d9
		snap115 := d10
		snap116 := d11
		snap117 := d12
		snap118 := d13
		snap119 := d14
		snap120 := d15
		snap121 := d16
		snap122 := d17
		snap123 := d18
		snap124 := d19
		snap125 := d20
		snap126 := d21
		snap127 := d22
		snap128 := d23
		snap129 := d53
		snap130 := d54
		snap131 := d55
		snap132 := d56
		snap133 := d57
		snap134 := d58
		snap135 := d59
		snap136 := d60
		snap137 := d98
		snap138 := d99
		snap139 := d100
		alloc140 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps104)
		}
		ctx.RestoreAllocState(alloc140)
		d0 = snap105
		d1 = snap106
		d2 = snap107
		d3 = snap108
		d4 = snap109
		d5 = snap110
		d6 = snap111
		d7 = snap112
		d8 = snap113
		d9 = snap114
		d10 = snap115
		d11 = snap116
		d12 = snap117
		d13 = snap118
		d14 = snap119
		d15 = snap120
		d16 = snap121
		d17 = snap122
		d18 = snap123
		d19 = snap124
		d20 = snap125
		d21 = snap126
		d22 = snap127
		d23 = snap128
		d53 = snap129
		d54 = snap130
		d55 = snap131
		d56 = snap132
		d57 = snap133
		d58 = snap134
		d59 = snap135
		d60 = snap136
		d98 = snap137
		d99 = snap138
		d100 = snap139
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps103)
		}
		return result
		ctx.FreeDesc(&d99)
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
		if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
			d98 = ps.OverlayValues[98]
		}
		if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
			d99 = ps.OverlayValues[99]
		}
		if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
			d100 = ps.OverlayValues[100]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		r44 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r44, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r45 := ctx.AllocReg()
		if d58.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r45, uint64(d58.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r45, d58.Reg)
			ctx.EmitShlRegImm8(r45, 3)
		}
		ctx.EmitAddInt64(r44, r45)
		ctx.FreeReg(r45)
		r46 := ctx.AllocRegExcept(r44)
		ctx.EmitMovRegMem(r46, r44, 0)
		ctx.FreeReg(r44)
		d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
		ctx.BindReg(r46, &d141)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&d141)
		ctx.EnsureDescsTogether(&d57, &d141)
		var d142 scm.JITValueDesc
		if d57.Loc == scm.LocImm && d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() * d141.Imm.Int())}
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d141.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d57.Imm.Int()))
			ctx.EmitImulInt64(scratch, d141.Reg)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d142)
		} else if d141.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(scratch, d57.Reg)
			if d141.Imm.Int() >= -2147483648 && d141.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d141.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d142)
		} else {
			r47 := ctx.AllocRegExcept(d57.Reg, d141.Reg)
			ctx.EmitMovRegReg(r47, d57.Reg)
			ctx.EmitImulInt64(r47, d141.Reg)
			d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			ctx.BindReg(r47, &d142)
		}
		if d142.Loc == scm.LocReg && d57.Loc == scm.LocReg && d142.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d141)
		ctx.EnsureDesc(&d142)
		d143 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d143)
		ctx.BindReg(r1, &d143)
		ctx.EnsureDesc(&d142)
		ctx.EmitMakeInt(d143, d142)
		if d142.Loc == scm.LocReg {
			ctx.FreeReg(d142.Reg)
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
		if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != scm.LocNone {
			d98 = ps.OverlayValues[98]
		}
		if len(ps.OverlayValues) > 99 && ps.OverlayValues[99].Loc != scm.LocNone {
			d99 = ps.OverlayValues[99]
		}
		if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != scm.LocNone {
			d100 = ps.OverlayValues[100]
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
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&d57)
		var d144 scm.JITValueDesc
		if d57.Loc == scm.LocImm {
			d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d57.Imm.Int()))}
		} else {
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, d57.Reg)
			d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d57.Reg}
			ctx.BindReg(d57.Reg, &d144)
		}
		ctx.FreeDesc(&d57)
		ctx.EnsureDesc(&d58)
		ctx.EnsureDesc(&d58)
		var d145 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d58.Imm.Int()))))}
		} else {
			r48 := ctx.AllocReg()
			ctx.EmitMovRegReg(r48, d58.Reg)
			ctx.EmitShlRegImm8(r48, 56)
			ctx.EmitSarRegImm8(r48, 56)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			ctx.BindReg(r48, &d145)
		}
		ctx.EnsureDesc(&d145)
		ctx.EnsureDesc(&d145)
		var d146 scm.JITValueDesc
		if d145.Loc == scm.LocImm {
			d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d145.Reg)
			ctx.EmitMovRegReg(scratch, d145.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d146)
		}
		if d146.Loc == scm.LocReg && d145.Loc == scm.LocReg && d146.Reg == d145.Reg {
			ctx.TransferReg(d145.Reg)
			d145.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d145)
		ctx.EnsureDesc(&d146)
		r49 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r49, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r50 := ctx.AllocReg()
		if d146.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r50, uint64(d146.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r50, d146.Reg)
			ctx.EmitShlRegImm8(r50, 3)
		}
		ctx.EmitAddInt64(r49, r50)
		ctx.FreeReg(r50)
		r51 := ctx.AllocRegExcept(r49)
		ctx.EmitMovRegMem(r51, r49, 0)
		ctx.FreeReg(r49)
		d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
		ctx.BindReg(r51, &d147)
		ctx.FreeDesc(&d146)
		ctx.EnsureDesc(&d144)
		ctx.EnsureDesc(&d147)
		ctx.EnsureDescsTogether(&d144, &d147)
		var d148 scm.JITValueDesc
		if d144.Loc == scm.LocImm && d147.Loc == scm.LocImm {
			d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d144.Imm.Float() * d147.Imm.Float())}
		} else if d144.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d147.Reg)
			_, xBits := d144.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d147.Reg)
			d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d148)
		} else if d147.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d144.Reg)
			ctx.EmitMovRegReg(scratch, d144.Reg)
			_, yBits := d147.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d148)
		} else {
			r52 := ctx.AllocRegExcept(d144.Reg, d147.Reg)
			ctx.EmitMovRegReg(r52, d144.Reg)
			ctx.EmitMulFloat64(r52, d147.Reg)
			d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r52}
			ctx.BindReg(r52, &d148)
		}
		if d148.Loc == scm.LocReg && d144.Loc == scm.LocReg && d148.Reg == d144.Reg {
			ctx.TransferReg(d144.Reg)
			d144.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d144)
		ctx.FreeDesc(&d147)
		ctx.EnsureDesc(&d148)
		d149 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d149)
		ctx.BindReg(r1, &d149)
		ctx.EnsureDesc(&d148)
		ctx.EmitMakeFloat(d149, d148)
		if d148.Loc == scm.LocReg {
			ctx.FreeReg(d148.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps150 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps150)
	ctx.MarkLabel(lbl0)
	d151 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d151)
	ctx.BindReg(r1, &d151)
	ctx.EmitMovPairToResult(&d151, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	if thisptrPinned {
		ctx.UnprotectReg(thisptrPinnedReg)
	}
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
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
