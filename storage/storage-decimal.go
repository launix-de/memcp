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
	storageJITFunctions
	inner    StorageInt `jit:"immutable-after-finish"` // embedded, NOT pointer
	scaleExp int8       `jit:"immutable-after-finish"` // real_value = stored_int * 10^scaleExp
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

func (s *StorageDecimal) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

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
	s.storageJITFunctions.finish(s)
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

func (s *StorageDecimal) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	var d24 scm.JITValueDesc
	_ = d24
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
	var d61 scm.JITValueDesc
	_ = d61
	var d62 scm.JITValueDesc
	_ = d62
	var d101 scm.JITValueDesc
	_ = d101
	var d102 scm.JITValueDesc
	_ = d102
	var d103 scm.JITValueDesc
	_ = d103
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
	var d150 scm.JITValueDesc
	_ = d150
	var d151 scm.JITValueDesc
	_ = d151
	var d152 scm.JITValueDesc
	_ = d152
	var d153 scm.JITValueDesc
	_ = d153
	var d154 scm.JITValueDesc
	_ = d154
	var d155 scm.JITValueDesc
	_ = d155
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptr := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer(s)))), NoHeapPointer: true}
	standaloneFrame := ctx.BeginStandaloneFrame()
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
		defer ctx.UnprotectReg(idxPinnedReg)
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
		ctx.StabilizeDescForControlFlow(&d0)
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
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48)
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r3, thisptr.Reg, off)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d2)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d2)
		ctx.EnsureDesc(&d2)
		var d3 scm.JITValueDesc
		if d2.Loc == scm.LocImm {
			d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d2.Reg)
			ctx.EmitShlRegImm8(r4, 56)
			ctx.EmitShrRegImm8(r4, 56)
			d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d3)
		}
		ctx.FreeDesc(&d2)
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
			r5 := ctx.AllocRegExcept(d1.Reg, d3.Reg)
			ctx.EmitMovRegReg(r5, d1.Reg)
			ctx.EmitImulInt64(r5, d3.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d4)
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
			r6 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r6, d4.Reg)
			ctx.EmitShrRegImm8(r6, 6)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d5)
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
			r7 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r7, d4.Reg)
			ctx.EmitAndRegImm32(r7, 63)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d6)
		}
		if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d4)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d7 scm.JITValueDesc
		r8 := ctx.AllocReg()
		r9 := ctx.AllocRegExcept(r8)
		r10 := ctx.AllocRegExcept(r8, r9)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r8, uint64(dataPtr))
			ctx.EmitMovRegImm64(r9, uint64(sliceLen))
			ctx.EmitMovRegImm64(r10, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off+16)
		}
		d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
		ctx.BindReg(r8, &d7)
		ctx.BindReg(r9, &d7)
		ctx.BindReg(r10, &d7)
		ctx.BindReg(r8, &d7)
		ctx.BindReg(r9, &d7)
		ctx.BindReg(r10, &d7)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		d9 = ctx.EmitSliceElementAddress(&d7, &d5, 8)
		ctx.EnsureDesc(&d9)
		ctx.EmitMovRegMem(d9.Reg, d9.Reg, 0)
		d8 = d9
		d8.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d8, &d6)
		var d10 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d6.Imm.Int())))}
		} else if d6.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r11, d8.Reg)
			ctx.EmitShlRegImm8(r11, uint8(d6.Imm.Int()))
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d10)
		} else {
			{
				shiftSrc := d8.Reg
				r12 := ctx.AllocRegExcept(d8.Reg, d6.Reg)
				ctx.EmitMovRegReg(r12, d8.Reg)
				shiftSrc = r12
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d6.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d6.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d6.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
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
		d12.Type = scm.TagInt
		ctx.FreeDesc(&d11)
		ctx.ReclaimUntrackedRegs()
		d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d14, &d6)
		var d15 scm.JITValueDesc
		if d14.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d6.Imm.Int())}
		} else if d6.Loc == scm.LocImm && d6.Imm.Int() == 0 {
			r13 := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegReg(r13, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d15)
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
			r14 := ctx.AllocRegExcept(d14.Reg, d6.Reg)
			ctx.EmitMovRegReg(r14, d14.Reg)
			ctx.EmitSubInt64(r14, d6.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d15)
		}
		if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
			ctx.TransferReg(d14.Reg)
			d14.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d12)
		ctx.EnsureDesc(&d15)
		ctx.EnsureDescsTogether(&d12, &d15)
		var d16 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d15.Loc == scm.LocImm {
			d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) >> uint64(d15.Imm.Int())))}
		} else if d15.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r15, d12.Reg)
			ctx.EmitShrRegImm8(r15, uint8(d15.Imm.Int()))
			d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d16)
		} else {
			{
				shiftSrc := d12.Reg
				r16 := ctx.AllocRegExcept(d12.Reg, d15.Reg)
				ctx.EmitMovRegReg(r16, d12.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d15.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d15.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d15.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
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
			r17 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r17, d10.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d17)
		} else if d10.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
			ctx.EmitOrInt64(scratch, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d16.Loc == scm.LocImm {
			r18 := ctx.AllocRegExcept(d10.Reg)
			ctx.EmitMovRegReg(r18, d10.Reg)
			if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r18, int32(d16.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
				ctx.EmitOrInt64(r18, scm.RegR11)
			}
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d17)
		} else {
			r19 := ctx.AllocRegExcept(d10.Reg, d16.Reg)
			ctx.EmitMovRegReg(r19, d10.Reg)
			ctx.EmitOrInt64(r19, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d17)
		}
		if d17.Loc == scm.LocReg && d10.Loc == scm.LocReg && d17.Reg == d10.Reg {
			ctx.TransferReg(d10.Reg)
			d10.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d10)
		ctx.FreeDesc(&d16)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d18 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48)
			r20 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r20, thisptr.Reg, off)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d18)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d18)
		var d19 scm.JITValueDesc
		if d18.Loc == scm.LocImm {
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
		} else {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegReg(r21, d18.Reg)
			ctx.EmitShlRegImm8(r21, 56)
			ctx.EmitShrRegImm8(r21, 56)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d19)
		}
		ctx.FreeDesc(&d18)
		ctx.ReclaimUntrackedRegs()
		d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d19)
		ctx.EnsureDescsTogether(&d20, &d19)
		var d21 scm.JITValueDesc
		if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
			d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
		} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
			r22 := ctx.AllocRegExcept(d20.Reg)
			ctx.EmitMovRegReg(r22, d20.Reg)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d21)
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
			r23 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
			ctx.EmitMovRegReg(r23, d20.Reg)
			ctx.EmitSubInt64(r23, d19.Reg)
			d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d21)
		}
		if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
			ctx.TransferReg(d20.Reg)
			d20.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d19)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d21)
		ctx.EnsureDescsTogether(&d17, &d21)
		var d22 scm.JITValueDesc
		if d17.Loc == scm.LocImm && d21.Loc == scm.LocImm {
			d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d21.Imm.Int())))}
		} else if d21.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(r24, d17.Reg)
			ctx.EmitShrRegImm8(r24, uint8(d21.Imm.Int()))
			d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d22)
		} else {
			{
				shiftSrc := d17.Reg
				r25 := ctx.AllocRegExcept(d17.Reg, d21.Reg)
				ctx.EmitMovRegReg(r25, d17.Reg)
				shiftSrc = r25
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d21.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d21.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d22)
			}
		}
		if d22.Loc == scm.LocReg && d17.Loc == scm.LocReg && d22.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.FreeDesc(&d21)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.StabilizeDescForControlFlow(&d22)
		ctx.FreeDesc(&idxInt)
		var d23 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80)
			r26 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
			d23 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d23)
		}
		d24 = d23
		ctx.EnsureDesc(&d24)
		if d24.Loc != scm.LocImm && d24.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d24.Loc == scm.LocImm {
			if d24.Imm.Bool() {
				if ps.General {
				}
				ps25 := scm.PhiState{General: ps.General}
				ps25.OverlayValues = make([]scm.JITValueDesc, 25)
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
				ps25.OverlayValues[24] = d24
				return bbs[3].RenderPS(ps25)
			}
			if ps.General {
			}
			ps26 := scm.PhiState{General: ps.General}
			ps26.OverlayValues = make([]scm.JITValueDesc, 25)
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
			ps26.OverlayValues[24] = d24
			return bbs[2].RenderPS(ps26)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl8 := ctx.ReserveLabel()
		lbl9 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d24.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl8)
		ctx.EmitJmp(lbl9)
		ctx.MarkLabel(lbl8)
		ctx.EmitJmp(lbl4)
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl3)
		ps27 := scm.PhiState{General: true}
		ps27.OverlayValues = make([]scm.JITValueDesc, 25)
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
		ps27.OverlayValues[24] = d24
		ps28 := scm.PhiState{General: true}
		ps28.OverlayValues = make([]scm.JITValueDesc, 25)
		ps28.OverlayValues[0] = d0
		ps28.OverlayValues[1] = d1
		ps28.OverlayValues[2] = d2
		ps28.OverlayValues[3] = d3
		ps28.OverlayValues[4] = d4
		ps28.OverlayValues[5] = d5
		ps28.OverlayValues[6] = d6
		ps28.OverlayValues[7] = d7
		ps28.OverlayValues[8] = d8
		ps28.OverlayValues[9] = d9
		ps28.OverlayValues[10] = d10
		ps28.OverlayValues[11] = d11
		ps28.OverlayValues[12] = d12
		ps28.OverlayValues[13] = d13
		ps28.OverlayValues[14] = d14
		ps28.OverlayValues[15] = d15
		ps28.OverlayValues[16] = d16
		ps28.OverlayValues[17] = d17
		ps28.OverlayValues[18] = d18
		ps28.OverlayValues[19] = d19
		ps28.OverlayValues[20] = d20
		ps28.OverlayValues[21] = d21
		ps28.OverlayValues[22] = d22
		ps28.OverlayValues[23] = d23
		ps28.OverlayValues[24] = d24
		snap29 := d0
		snap30 := d1
		snap31 := d2
		snap32 := d3
		snap33 := d4
		snap34 := d5
		snap35 := d6
		snap36 := d7
		snap37 := d8
		snap38 := d9
		snap39 := d10
		snap40 := d11
		snap41 := d12
		snap42 := d13
		snap43 := d14
		snap44 := d15
		snap45 := d16
		snap46 := d17
		snap47 := d18
		snap48 := d19
		snap49 := d20
		snap50 := d21
		snap51 := d22
		snap52 := d23
		snap53 := d24
		alloc54 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps28)
		}
		ctx.RestoreAllocState(alloc54)
		d0 = snap29
		d1 = snap30
		d2 = snap31
		d3 = snap32
		d4 = snap33
		d5 = snap34
		d6 = snap35
		d7 = snap36
		d8 = snap37
		d9 = snap38
		d10 = snap39
		d11 = snap40
		d12 = snap41
		d13 = snap42
		d14 = snap43
		d15 = snap44
		d16 = snap45
		d17 = snap46
		d18 = snap47
		d19 = snap48
		d20 = snap49
		d21 = snap50
		d22 = snap51
		d23 = snap52
		d24 = snap53
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps27)
		}
		return result
		ctx.FreeDesc(&d23)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		ctx.ReclaimUntrackedRegs()
		d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d56 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d56)
		ctx.BindReg(r1, &d56)
		ctx.EnsureDesc(&d55)
		if d55.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d55, &d56)
		} else {
			switch d55.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d56, d55)
			case scm.TagInt:
				ctx.EmitMakeInt(d56, d55)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d56, d55)
			case scm.TagNil:
				ctx.EmitMakeNil(d56)
			default:
				ctx.EmitMovPairToResult(&d55, &d56)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
		}
		if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
			d55 = ps.OverlayValues[55]
		}
		if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
			d56 = ps.OverlayValues[56]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d57 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d22.Imm.Int()))))}
		} else {
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d22.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d57)
		}
		var d58 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMem(r28, thisptr.Reg, off)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d58)
		}
		ctx.EnsureDesc(&d57)
		ctx.EnsureDesc(&d58)
		ctx.EnsureDescsTogether(&d57, &d58)
		var d59 scm.JITValueDesc
		if d57.Loc == scm.LocImm && d58.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + d58.Imm.Int())}
		} else if d58.Loc == scm.LocImm && d58.Imm.Int() == 0 {
			r29 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r29, d57.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d59)
		} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d58.Reg}
			ctx.BindReg(d58.Reg, &d59)
		} else if d57.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d57.Imm.Int()))
			ctx.EmitAddInt64(scratch, d58.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d59)
		} else if d58.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(scratch, d57.Reg)
			if d58.Imm.Int() >= -2147483648 && d58.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d58.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d59)
		} else {
			r30 := ctx.AllocRegExcept(d57.Reg, d58.Reg)
			ctx.EmitMovRegReg(r30, d57.Reg)
			ctx.EmitAddInt64(r30, d58.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d59)
		}
		if d59.Loc == scm.LocReg && d57.Loc == scm.LocReg && d59.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d59)
		ctx.FreeDesc(&d57)
		ctx.FreeDesc(&d58)
		var d60 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r31 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r31, thisptr.Reg, off)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d60)
		}
		ctx.EnsureDesc(&d60)
		var d61 scm.JITValueDesc
		if d60.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d60.Imm.Int() > 0)}
		} else {
			r32 := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitCmpRegImm32(d60.Reg, 0)
			ctx.EmitSetcc(r32, scm.CondSignedGreater)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
			ctx.BindReg(r32, &d61)
		}
		ctx.FreeDesc(&d60)
		d62 = d61
		ctx.EnsureDesc(&d62)
		if d62.Loc != scm.LocImm && d62.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d62.Loc == scm.LocImm {
			if d62.Imm.Bool() {
				if ps.General {
				}
				ps63 := scm.PhiState{General: ps.General}
				ps63.OverlayValues = make([]scm.JITValueDesc, 63)
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
				ps63.OverlayValues[24] = d24
				ps63.OverlayValues[55] = d55
				ps63.OverlayValues[56] = d56
				ps63.OverlayValues[57] = d57
				ps63.OverlayValues[58] = d58
				ps63.OverlayValues[59] = d59
				ps63.OverlayValues[60] = d60
				ps63.OverlayValues[61] = d61
				ps63.OverlayValues[62] = d62
				return bbs[4].RenderPS(ps63)
			}
			if ps.General {
			}
			ps64 := scm.PhiState{General: ps.General}
			ps64.OverlayValues = make([]scm.JITValueDesc, 63)
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
			ps64.OverlayValues[24] = d24
			ps64.OverlayValues[55] = d55
			ps64.OverlayValues[56] = d56
			ps64.OverlayValues[57] = d57
			ps64.OverlayValues[58] = d58
			ps64.OverlayValues[59] = d59
			ps64.OverlayValues[60] = d60
			ps64.OverlayValues[61] = d61
			ps64.OverlayValues[62] = d62
			return bbs[5].RenderPS(ps64)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl10 := ctx.ReserveLabel()
		lbl11 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d62.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl10)
		ctx.EmitJmp(lbl11)
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl6)
		ps65 := scm.PhiState{General: true}
		ps65.OverlayValues = make([]scm.JITValueDesc, 63)
		ps65.OverlayValues[0] = d0
		ps65.OverlayValues[1] = d1
		ps65.OverlayValues[2] = d2
		ps65.OverlayValues[3] = d3
		ps65.OverlayValues[4] = d4
		ps65.OverlayValues[5] = d5
		ps65.OverlayValues[6] = d6
		ps65.OverlayValues[7] = d7
		ps65.OverlayValues[8] = d8
		ps65.OverlayValues[9] = d9
		ps65.OverlayValues[10] = d10
		ps65.OverlayValues[11] = d11
		ps65.OverlayValues[12] = d12
		ps65.OverlayValues[13] = d13
		ps65.OverlayValues[14] = d14
		ps65.OverlayValues[15] = d15
		ps65.OverlayValues[16] = d16
		ps65.OverlayValues[17] = d17
		ps65.OverlayValues[18] = d18
		ps65.OverlayValues[19] = d19
		ps65.OverlayValues[20] = d20
		ps65.OverlayValues[21] = d21
		ps65.OverlayValues[22] = d22
		ps65.OverlayValues[23] = d23
		ps65.OverlayValues[24] = d24
		ps65.OverlayValues[55] = d55
		ps65.OverlayValues[56] = d56
		ps65.OverlayValues[57] = d57
		ps65.OverlayValues[58] = d58
		ps65.OverlayValues[59] = d59
		ps65.OverlayValues[60] = d60
		ps65.OverlayValues[61] = d61
		ps65.OverlayValues[62] = d62
		ps66 := scm.PhiState{General: true}
		ps66.OverlayValues = make([]scm.JITValueDesc, 63)
		ps66.OverlayValues[0] = d0
		ps66.OverlayValues[1] = d1
		ps66.OverlayValues[2] = d2
		ps66.OverlayValues[3] = d3
		ps66.OverlayValues[4] = d4
		ps66.OverlayValues[5] = d5
		ps66.OverlayValues[6] = d6
		ps66.OverlayValues[7] = d7
		ps66.OverlayValues[8] = d8
		ps66.OverlayValues[9] = d9
		ps66.OverlayValues[10] = d10
		ps66.OverlayValues[11] = d11
		ps66.OverlayValues[12] = d12
		ps66.OverlayValues[13] = d13
		ps66.OverlayValues[14] = d14
		ps66.OverlayValues[15] = d15
		ps66.OverlayValues[16] = d16
		ps66.OverlayValues[17] = d17
		ps66.OverlayValues[18] = d18
		ps66.OverlayValues[19] = d19
		ps66.OverlayValues[20] = d20
		ps66.OverlayValues[21] = d21
		ps66.OverlayValues[22] = d22
		ps66.OverlayValues[23] = d23
		ps66.OverlayValues[24] = d24
		ps66.OverlayValues[55] = d55
		ps66.OverlayValues[56] = d56
		ps66.OverlayValues[57] = d57
		ps66.OverlayValues[58] = d58
		ps66.OverlayValues[59] = d59
		ps66.OverlayValues[60] = d60
		ps66.OverlayValues[61] = d61
		ps66.OverlayValues[62] = d62
		snap67 := d0
		snap68 := d1
		snap69 := d2
		snap70 := d3
		snap71 := d4
		snap72 := d5
		snap73 := d6
		snap74 := d7
		snap75 := d8
		snap76 := d9
		snap77 := d10
		snap78 := d11
		snap79 := d12
		snap80 := d13
		snap81 := d14
		snap82 := d15
		snap83 := d16
		snap84 := d17
		snap85 := d18
		snap86 := d19
		snap87 := d20
		snap88 := d21
		snap89 := d22
		snap90 := d23
		snap91 := d24
		snap92 := d55
		snap93 := d56
		snap94 := d57
		snap95 := d58
		snap96 := d59
		snap97 := d60
		snap98 := d61
		snap99 := d62
		alloc100 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps66)
		}
		ctx.RestoreAllocState(alloc100)
		d0 = snap67
		d1 = snap68
		d2 = snap69
		d3 = snap70
		d4 = snap71
		d5 = snap72
		d6 = snap73
		d7 = snap74
		d8 = snap75
		d9 = snap76
		d10 = snap77
		d11 = snap78
		d12 = snap79
		d13 = snap80
		d14 = snap81
		d15 = snap82
		d16 = snap83
		d17 = snap84
		d18 = snap85
		d19 = snap86
		d20 = snap87
		d21 = snap88
		d22 = snap89
		d23 = snap90
		d24 = snap91
		d55 = snap92
		d56 = snap93
		d57 = snap94
		d58 = snap95
		d59 = snap96
		d60 = snap97
		d61 = snap98
		d62 = snap99
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps65)
		}
		return result
		ctx.FreeDesc(&d61)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		ctx.ReclaimUntrackedRegs()
		var d101 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88)
			r33 := ctx.AllocReg()
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d101)
		}
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d101)
		ctx.EnsureDescsTogether(&d22, &d101)
		var d102 scm.JITValueDesc
		if d22.Loc == scm.LocImm && d101.Loc == scm.LocImm {
			d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d22.Imm.Int()) == uint64(d101.Imm.Int()))}
		} else if d101.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d22.Reg)
			if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d22.Reg, int32(d101.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
				ctx.EmitCmpInt64(d22.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r34, scm.CondEqual)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r34}
			ctx.BindReg(r34, &d102)
		} else if d22.Loc == scm.LocImm {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d101.Reg)
			ctx.EmitSetcc(r35, scm.CondEqual)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
			ctx.BindReg(r35, &d102)
		} else {
			r36 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitCmpInt64(d22.Reg, d101.Reg)
			ctx.EmitSetcc(r36, scm.CondEqual)
			d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			ctx.BindReg(r36, &d102)
		}
		ctx.FreeDesc(&d101)
		d103 = d102
		ctx.EnsureDesc(&d103)
		if d103.Loc != scm.LocImm && d103.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d103.Loc == scm.LocImm {
			if d103.Imm.Bool() {
				if ps.General {
				}
				ps104 := scm.PhiState{General: ps.General}
				ps104.OverlayValues = make([]scm.JITValueDesc, 104)
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
				ps104.OverlayValues[24] = d24
				ps104.OverlayValues[55] = d55
				ps104.OverlayValues[56] = d56
				ps104.OverlayValues[57] = d57
				ps104.OverlayValues[58] = d58
				ps104.OverlayValues[59] = d59
				ps104.OverlayValues[60] = d60
				ps104.OverlayValues[61] = d61
				ps104.OverlayValues[62] = d62
				ps104.OverlayValues[101] = d101
				ps104.OverlayValues[102] = d102
				ps104.OverlayValues[103] = d103
				return bbs[1].RenderPS(ps104)
			}
			if ps.General {
			}
			ps105 := scm.PhiState{General: ps.General}
			ps105.OverlayValues = make([]scm.JITValueDesc, 104)
			ps105.OverlayValues[0] = d0
			ps105.OverlayValues[1] = d1
			ps105.OverlayValues[2] = d2
			ps105.OverlayValues[3] = d3
			ps105.OverlayValues[4] = d4
			ps105.OverlayValues[5] = d5
			ps105.OverlayValues[6] = d6
			ps105.OverlayValues[7] = d7
			ps105.OverlayValues[8] = d8
			ps105.OverlayValues[9] = d9
			ps105.OverlayValues[10] = d10
			ps105.OverlayValues[11] = d11
			ps105.OverlayValues[12] = d12
			ps105.OverlayValues[13] = d13
			ps105.OverlayValues[14] = d14
			ps105.OverlayValues[15] = d15
			ps105.OverlayValues[16] = d16
			ps105.OverlayValues[17] = d17
			ps105.OverlayValues[18] = d18
			ps105.OverlayValues[19] = d19
			ps105.OverlayValues[20] = d20
			ps105.OverlayValues[21] = d21
			ps105.OverlayValues[22] = d22
			ps105.OverlayValues[23] = d23
			ps105.OverlayValues[24] = d24
			ps105.OverlayValues[55] = d55
			ps105.OverlayValues[56] = d56
			ps105.OverlayValues[57] = d57
			ps105.OverlayValues[58] = d58
			ps105.OverlayValues[59] = d59
			ps105.OverlayValues[60] = d60
			ps105.OverlayValues[61] = d61
			ps105.OverlayValues[62] = d62
			ps105.OverlayValues[101] = d101
			ps105.OverlayValues[102] = d102
			ps105.OverlayValues[103] = d103
			return bbs[2].RenderPS(ps105)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d103.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl3)
		ps106 := scm.PhiState{General: true}
		ps106.OverlayValues = make([]scm.JITValueDesc, 104)
		ps106.OverlayValues[0] = d0
		ps106.OverlayValues[1] = d1
		ps106.OverlayValues[2] = d2
		ps106.OverlayValues[3] = d3
		ps106.OverlayValues[4] = d4
		ps106.OverlayValues[5] = d5
		ps106.OverlayValues[6] = d6
		ps106.OverlayValues[7] = d7
		ps106.OverlayValues[8] = d8
		ps106.OverlayValues[9] = d9
		ps106.OverlayValues[10] = d10
		ps106.OverlayValues[11] = d11
		ps106.OverlayValues[12] = d12
		ps106.OverlayValues[13] = d13
		ps106.OverlayValues[14] = d14
		ps106.OverlayValues[15] = d15
		ps106.OverlayValues[16] = d16
		ps106.OverlayValues[17] = d17
		ps106.OverlayValues[18] = d18
		ps106.OverlayValues[19] = d19
		ps106.OverlayValues[20] = d20
		ps106.OverlayValues[21] = d21
		ps106.OverlayValues[22] = d22
		ps106.OverlayValues[23] = d23
		ps106.OverlayValues[24] = d24
		ps106.OverlayValues[55] = d55
		ps106.OverlayValues[56] = d56
		ps106.OverlayValues[57] = d57
		ps106.OverlayValues[58] = d58
		ps106.OverlayValues[59] = d59
		ps106.OverlayValues[60] = d60
		ps106.OverlayValues[61] = d61
		ps106.OverlayValues[62] = d62
		ps106.OverlayValues[101] = d101
		ps106.OverlayValues[102] = d102
		ps106.OverlayValues[103] = d103
		ps107 := scm.PhiState{General: true}
		ps107.OverlayValues = make([]scm.JITValueDesc, 104)
		ps107.OverlayValues[0] = d0
		ps107.OverlayValues[1] = d1
		ps107.OverlayValues[2] = d2
		ps107.OverlayValues[3] = d3
		ps107.OverlayValues[4] = d4
		ps107.OverlayValues[5] = d5
		ps107.OverlayValues[6] = d6
		ps107.OverlayValues[7] = d7
		ps107.OverlayValues[8] = d8
		ps107.OverlayValues[9] = d9
		ps107.OverlayValues[10] = d10
		ps107.OverlayValues[11] = d11
		ps107.OverlayValues[12] = d12
		ps107.OverlayValues[13] = d13
		ps107.OverlayValues[14] = d14
		ps107.OverlayValues[15] = d15
		ps107.OverlayValues[16] = d16
		ps107.OverlayValues[17] = d17
		ps107.OverlayValues[18] = d18
		ps107.OverlayValues[19] = d19
		ps107.OverlayValues[20] = d20
		ps107.OverlayValues[21] = d21
		ps107.OverlayValues[22] = d22
		ps107.OverlayValues[23] = d23
		ps107.OverlayValues[24] = d24
		ps107.OverlayValues[55] = d55
		ps107.OverlayValues[56] = d56
		ps107.OverlayValues[57] = d57
		ps107.OverlayValues[58] = d58
		ps107.OverlayValues[59] = d59
		ps107.OverlayValues[60] = d60
		ps107.OverlayValues[61] = d61
		ps107.OverlayValues[62] = d62
		ps107.OverlayValues[101] = d101
		ps107.OverlayValues[102] = d102
		ps107.OverlayValues[103] = d103
		snap108 := d0
		snap109 := d1
		snap110 := d2
		snap111 := d3
		snap112 := d4
		snap113 := d5
		snap114 := d6
		snap115 := d7
		snap116 := d8
		snap117 := d9
		snap118 := d10
		snap119 := d11
		snap120 := d12
		snap121 := d13
		snap122 := d14
		snap123 := d15
		snap124 := d16
		snap125 := d17
		snap126 := d18
		snap127 := d19
		snap128 := d20
		snap129 := d21
		snap130 := d22
		snap131 := d23
		snap132 := d24
		snap133 := d55
		snap134 := d56
		snap135 := d57
		snap136 := d58
		snap137 := d59
		snap138 := d60
		snap139 := d61
		snap140 := d62
		snap141 := d101
		snap142 := d102
		snap143 := d103
		alloc144 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps107)
		}
		ctx.RestoreAllocState(alloc144)
		d0 = snap108
		d1 = snap109
		d2 = snap110
		d3 = snap111
		d4 = snap112
		d5 = snap113
		d6 = snap114
		d7 = snap115
		d8 = snap116
		d9 = snap117
		d10 = snap118
		d11 = snap119
		d12 = snap120
		d13 = snap121
		d14 = snap122
		d15 = snap123
		d16 = snap124
		d17 = snap125
		d18 = snap126
		d19 = snap127
		d20 = snap128
		d21 = snap129
		d22 = snap130
		d23 = snap131
		d24 = snap132
		d55 = snap133
		d56 = snap134
		d57 = snap135
		d58 = snap136
		d59 = snap137
		d60 = snap138
		d61 = snap139
		d62 = snap140
		d101 = snap141
		d102 = snap142
		d103 = snap143
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps106)
		}
		return result
		ctx.FreeDesc(&d102)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
			d101 = ps.OverlayValues[101]
		}
		if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
			d102 = ps.OverlayValues[102]
		}
		if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
			d103 = ps.OverlayValues[103]
		}
		ctx.ReclaimUntrackedRegs()
		var d145 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r37, thisptr.Reg, off)
			d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d145)
		}
		ctx.EnsureDesc(&d145)
		r38 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r38, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r39 := ctx.AllocReg()
		if d145.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r39, uint64(d145.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r39, d145.Reg)
			ctx.EmitShlRegImm8(r39, 3)
		}
		ctx.EmitAddInt64(r38, r39)
		ctx.FreeReg(r39)
		r40 := ctx.AllocRegExcept(r38)
		ctx.EmitMovRegMem(r40, r38, 0)
		ctx.FreeReg(r38)
		d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
		ctx.BindReg(r40, &d146)
		ctx.FreeDesc(&d145)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDesc(&d146)
		ctx.EnsureDescsTogether(&d59, &d146)
		var d147 scm.JITValueDesc
		if d59.Loc == scm.LocImm && d146.Loc == scm.LocImm {
			d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() * d146.Imm.Int())}
		} else if d59.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d146.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
			ctx.EmitImulInt64(scratch, d146.Reg)
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d147)
		} else if d146.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(scratch, d59.Reg)
			if d146.Imm.Int() >= -2147483648 && d146.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d146.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d146.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d147)
		} else {
			r41 := ctx.AllocRegExcept(d59.Reg, d146.Reg)
			ctx.EmitMovRegReg(r41, d59.Reg)
			ctx.EmitImulInt64(r41, d146.Reg)
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d147)
		}
		if d147.Loc == scm.LocReg && d59.Loc == scm.LocReg && d147.Reg == d59.Reg {
			ctx.TransferReg(d59.Reg)
			d59.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d146)
		ctx.EnsureDesc(&d147)
		d148 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d148)
		ctx.BindReg(r1, &d148)
		ctx.EnsureDesc(&d147)
		ctx.EmitMakeInt(d148, d147)
		if d147.Loc == scm.LocReg {
			ctx.FreeReg(d147.Reg)
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
		if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
			d24 = ps.OverlayValues[24]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != scm.LocNone {
			d101 = ps.OverlayValues[101]
		}
		if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != scm.LocNone {
			d102 = ps.OverlayValues[102]
		}
		if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != scm.LocNone {
			d103 = ps.OverlayValues[103]
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
		if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
			d148 = ps.OverlayValues[148]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d59)
		ctx.EnsureDesc(&d59)
		var d149 scm.JITValueDesc
		if d59.Loc == scm.LocImm {
			d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d59.Imm.Int()))}
		} else {
			r42 := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(r42, d59.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r42)
			d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r42}
			ctx.BindReg(r42, &d149)
		}
		var d150 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r43 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r43, thisptr.Reg, off)
			d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
			ctx.BindReg(r43, &d150)
		}
		ctx.EnsureDesc(&d150)
		ctx.EnsureDesc(&d150)
		var d151 scm.JITValueDesc
		if d150.Loc == scm.LocImm {
			d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d150.Imm.Int()))))}
		} else {
			r44 := ctx.AllocReg()
			ctx.EmitMovRegReg(r44, d150.Reg)
			ctx.EmitShlRegImm8(r44, 56)
			ctx.EmitSarRegImm8(r44, 56)
			d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d151)
		}
		ctx.FreeDesc(&d150)
		ctx.EnsureDesc(&d151)
		ctx.EnsureDesc(&d151)
		var d152 scm.JITValueDesc
		if d151.Loc == scm.LocImm {
			d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d151.Reg)
			ctx.EmitMovRegReg(scratch, d151.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d152)
		}
		if d152.Loc == scm.LocReg && d151.Loc == scm.LocReg && d152.Reg == d151.Reg {
			ctx.TransferReg(d151.Reg)
			d151.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d151)
		ctx.EnsureDesc(&d152)
		r45 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r45, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r46 := ctx.AllocReg()
		if d152.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r46, uint64(d152.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r46, d152.Reg)
			ctx.EmitShlRegImm8(r46, 3)
		}
		ctx.EmitAddInt64(r45, r46)
		ctx.FreeReg(r46)
		r47 := ctx.AllocRegExcept(r45)
		ctx.EmitMovRegMem(r47, r45, 0)
		ctx.FreeReg(r45)
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
		ctx.BindReg(r47, &d153)
		ctx.FreeDesc(&d152)
		ctx.EnsureDesc(&d149)
		ctx.EnsureDesc(&d153)
		ctx.EnsureDescsTogether(&d149, &d153)
		var d154 scm.JITValueDesc
		if d149.Loc == scm.LocImm && d153.Loc == scm.LocImm {
			d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d149.Imm.Float() * d153.Imm.Float())}
		} else if d149.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d153.Reg)
			_, xBits := d149.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d153.Reg)
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d154)
		} else if d153.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d149.Reg)
			ctx.EmitMovRegReg(scratch, d149.Reg)
			_, yBits := d153.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d154)
		} else {
			r48 := ctx.AllocRegExcept(d149.Reg, d153.Reg)
			ctx.EmitMovRegReg(r48, d149.Reg)
			ctx.EmitMulFloat64(r48, d153.Reg)
			d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r48}
			ctx.BindReg(r48, &d154)
		}
		if d154.Loc == scm.LocReg && d149.Loc == scm.LocReg && d154.Reg == d149.Reg {
			ctx.TransferReg(d149.Reg)
			d149.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d149)
		ctx.FreeDesc(&d153)
		ctx.EnsureDesc(&d154)
		d155 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d155)
		ctx.BindReg(r1, &d155)
		ctx.EnsureDesc(&d154)
		ctx.EmitMakeFloat(d155, d154)
		if d154.Loc == scm.LocReg {
			ctx.FreeReg(d154.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps156 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps156)
	ctx.MarkLabel(lbl0)
	d157 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d157)
	ctx.BindReg(r1, &d157)
	ctx.EmitMovPairToResult(&d157, &result)
	ctx.FreeReg(r0)
	ctx.FreeReg(r1)
	ctx.ResolveFixups()
	if resultRegsProtected {
		ctx.UnprotectReg(result.Reg2)
		ctx.UnprotectReg(result.Reg)
	}
	ctx.EndStandaloneFrame(standaloneFrame)
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
