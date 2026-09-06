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
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d84 scm.JITValueDesc
	_ = d84
	var d85 scm.JITValueDesc
	_ = d85
	var d86 scm.JITValueDesc
	_ = d86
	var d87 scm.JITValueDesc
	_ = d87
	var d88 scm.JITValueDesc
	_ = d88
	var d161 scm.JITValueDesc
	_ = d161
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d242 scm.JITValueDesc
	_ = d242
	var d243 scm.JITValueDesc
	_ = d243
	var d244 scm.JITValueDesc
	_ = d244
	var d245 scm.JITValueDesc
	_ = d245
	var d246 scm.JITValueDesc
	_ = d246
	var d247 scm.JITValueDesc
	_ = d247
	var d248 scm.JITValueDesc
	_ = d248
	var d249 scm.JITValueDesc
	_ = d249
	var d250 scm.JITValueDesc
	_ = d250
	var d251 scm.JITValueDesc
	_ = d251
	var d252 scm.JITValueDesc
	_ = d252
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
		snap27 := d0
		snap28 := d1
		snap29 := d2
		snap30 := d3
		snap31 := d4
		snap32 := d5
		snap33 := d6
		snap34 := d7
		snap35 := d8
		snap36 := d9
		snap37 := d10
		snap38 := d11
		snap39 := d12
		snap40 := d13
		snap41 := d14
		snap42 := d15
		snap43 := d16
		snap44 := d17
		snap45 := d18
		snap46 := d19
		snap47 := d20
		snap48 := d21
		snap49 := d22
		snap50 := d23
		snap51 := d24
		alloc52 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl8)
		ctx.EmitJmp(lbl4)
		ctx.RestoreAllocState(alloc52)
		d0 = snap27
		d1 = snap28
		d2 = snap29
		d3 = snap30
		d4 = snap31
		d5 = snap32
		d6 = snap33
		d7 = snap34
		d8 = snap35
		d9 = snap36
		d10 = snap37
		d11 = snap38
		d12 = snap39
		d13 = snap40
		d14 = snap41
		d15 = snap42
		d16 = snap43
		d17 = snap44
		d18 = snap45
		d19 = snap46
		d20 = snap47
		d21 = snap48
		d22 = snap49
		d23 = snap50
		d24 = snap51
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc52)
		d0 = snap27
		d1 = snap28
		d2 = snap29
		d3 = snap30
		d4 = snap31
		d5 = snap32
		d6 = snap33
		d7 = snap34
		d8 = snap35
		d9 = snap36
		d10 = snap37
		d11 = snap38
		d12 = snap39
		d13 = snap40
		d14 = snap41
		d15 = snap42
		d16 = snap43
		d17 = snap44
		d18 = snap45
		d19 = snap46
		d20 = snap47
		d21 = snap48
		d22 = snap49
		d23 = snap50
		d24 = snap51
		ps53 := scm.PhiState{General: true}
		ps53.OverlayValues = make([]scm.JITValueDesc, 25)
		ps53.OverlayValues[0] = d0
		ps53.OverlayValues[1] = d1
		ps53.OverlayValues[2] = d2
		ps53.OverlayValues[3] = d3
		ps53.OverlayValues[4] = d4
		ps53.OverlayValues[5] = d5
		ps53.OverlayValues[6] = d6
		ps53.OverlayValues[7] = d7
		ps53.OverlayValues[8] = d8
		ps53.OverlayValues[9] = d9
		ps53.OverlayValues[10] = d10
		ps53.OverlayValues[11] = d11
		ps53.OverlayValues[12] = d12
		ps53.OverlayValues[13] = d13
		ps53.OverlayValues[14] = d14
		ps53.OverlayValues[15] = d15
		ps53.OverlayValues[16] = d16
		ps53.OverlayValues[17] = d17
		ps53.OverlayValues[18] = d18
		ps53.OverlayValues[19] = d19
		ps53.OverlayValues[20] = d20
		ps53.OverlayValues[21] = d21
		ps53.OverlayValues[22] = d22
		ps53.OverlayValues[23] = d23
		ps53.OverlayValues[24] = d24
		ps54 := scm.PhiState{General: true}
		ps54.OverlayValues = make([]scm.JITValueDesc, 25)
		ps54.OverlayValues[0] = d0
		ps54.OverlayValues[1] = d1
		ps54.OverlayValues[2] = d2
		ps54.OverlayValues[3] = d3
		ps54.OverlayValues[4] = d4
		ps54.OverlayValues[5] = d5
		ps54.OverlayValues[6] = d6
		ps54.OverlayValues[7] = d7
		ps54.OverlayValues[8] = d8
		ps54.OverlayValues[9] = d9
		ps54.OverlayValues[10] = d10
		ps54.OverlayValues[11] = d11
		ps54.OverlayValues[12] = d12
		ps54.OverlayValues[13] = d13
		ps54.OverlayValues[14] = d14
		ps54.OverlayValues[15] = d15
		ps54.OverlayValues[16] = d16
		ps54.OverlayValues[17] = d17
		ps54.OverlayValues[18] = d18
		ps54.OverlayValues[19] = d19
		ps54.OverlayValues[20] = d20
		ps54.OverlayValues[21] = d21
		ps54.OverlayValues[22] = d22
		ps54.OverlayValues[23] = d23
		ps54.OverlayValues[24] = d24
		snap55 := d0
		snap56 := d1
		snap57 := d2
		snap58 := d3
		snap59 := d4
		snap60 := d5
		snap61 := d6
		snap62 := d7
		snap63 := d8
		snap64 := d9
		snap65 := d10
		snap66 := d11
		snap67 := d12
		snap68 := d13
		snap69 := d14
		snap70 := d15
		snap71 := d16
		snap72 := d17
		snap73 := d18
		snap74 := d19
		snap75 := d20
		snap76 := d21
		snap77 := d22
		snap78 := d23
		snap79 := d24
		alloc80 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps54)
		}
		ctx.RestoreAllocState(alloc80)
		d0 = snap55
		d1 = snap56
		d2 = snap57
		d3 = snap58
		d4 = snap59
		d5 = snap60
		d6 = snap61
		d7 = snap62
		d8 = snap63
		d9 = snap64
		d10 = snap65
		d11 = snap66
		d12 = snap67
		d13 = snap68
		d14 = snap69
		d15 = snap70
		d16 = snap71
		d17 = snap72
		d18 = snap73
		d19 = snap74
		d20 = snap75
		d21 = snap76
		d22 = snap77
		d23 = snap78
		d24 = snap79
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps53)
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
		d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d82 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d82)
		ctx.BindReg(r1, &d82)
		ctx.EnsureDesc(&d81)
		if d81.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d81, &d82)
		} else {
			switch d81.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d82, d81)
			case scm.TagInt:
				ctx.EmitMakeInt(d82, d81)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d82, d81)
			case scm.TagNil:
				ctx.EmitMakeNil(d82)
			default:
				ctx.EmitMovPairToResult(&d81, &d82)
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
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d22)
		var d83 scm.JITValueDesc
		if d22.Loc == scm.LocImm {
			d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d22.Imm.Int()))))}
		} else {
			r27 := ctx.AllocReg()
			ctx.EmitMovRegReg(r27, d22.Reg)
			d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d83)
		}
		var d84 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMem(r28, thisptr.Reg, off)
			d84 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d84)
		}
		ctx.EnsureDesc(&d83)
		ctx.EnsureDesc(&d84)
		ctx.EnsureDescsTogether(&d83, &d84)
		var d85 scm.JITValueDesc
		if d83.Loc == scm.LocImm && d84.Loc == scm.LocImm {
			d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() + d84.Imm.Int())}
		} else if d84.Loc == scm.LocImm && d84.Imm.Int() == 0 {
			r29 := ctx.AllocRegExcept(d83.Reg)
			ctx.EmitMovRegReg(r29, d83.Reg)
			d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d85)
		} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
			d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d84.Reg}
			ctx.BindReg(d84.Reg, &d85)
		} else if d83.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d84.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
			ctx.EmitAddInt64(scratch, d84.Reg)
			d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d85)
		} else if d84.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d83.Reg)
			ctx.EmitMovRegReg(scratch, d83.Reg)
			if d84.Imm.Int() >= -2147483648 && d84.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d84.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d84.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d85)
		} else {
			r30 := ctx.AllocRegExcept(d83.Reg, d84.Reg)
			ctx.EmitMovRegReg(r30, d83.Reg)
			ctx.EmitAddInt64(r30, d84.Reg)
			d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			ctx.BindReg(r30, &d85)
		}
		if d85.Loc == scm.LocReg && d83.Loc == scm.LocReg && d85.Reg == d83.Reg {
			ctx.TransferReg(d83.Reg)
			d83.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d85)
		ctx.FreeDesc(&d83)
		ctx.FreeDesc(&d84)
		var d86 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r31 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r31, thisptr.Reg, off)
			d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d86)
		}
		ctx.EnsureDesc(&d86)
		var d87 scm.JITValueDesc
		if d86.Loc == scm.LocImm {
			d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d86.Imm.Int() > 0)}
		} else {
			r32 := ctx.AllocRegExcept(d86.Reg)
			ctx.EmitCmpRegImm32(d86.Reg, 0)
			d87 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r32, Condition: scm.CondSignedGreater}
			ctx.BindReg(r32, &d87)
		}
		ctx.FreeDesc(&d86)
		d88 = d87
		ctx.EnsureDesc(&d88)
		if d88.Loc != scm.LocImm && d88.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d88.Loc == scm.LocImm {
			if d88.Imm.Bool() {
				if ps.General {
				}
				ps89 := scm.PhiState{General: ps.General}
				ps89.OverlayValues = make([]scm.JITValueDesc, 89)
				ps89.OverlayValues[0] = d0
				ps89.OverlayValues[1] = d1
				ps89.OverlayValues[2] = d2
				ps89.OverlayValues[3] = d3
				ps89.OverlayValues[4] = d4
				ps89.OverlayValues[5] = d5
				ps89.OverlayValues[6] = d6
				ps89.OverlayValues[7] = d7
				ps89.OverlayValues[8] = d8
				ps89.OverlayValues[9] = d9
				ps89.OverlayValues[10] = d10
				ps89.OverlayValues[11] = d11
				ps89.OverlayValues[12] = d12
				ps89.OverlayValues[13] = d13
				ps89.OverlayValues[14] = d14
				ps89.OverlayValues[15] = d15
				ps89.OverlayValues[16] = d16
				ps89.OverlayValues[17] = d17
				ps89.OverlayValues[18] = d18
				ps89.OverlayValues[19] = d19
				ps89.OverlayValues[20] = d20
				ps89.OverlayValues[21] = d21
				ps89.OverlayValues[22] = d22
				ps89.OverlayValues[23] = d23
				ps89.OverlayValues[24] = d24
				ps89.OverlayValues[81] = d81
				ps89.OverlayValues[82] = d82
				ps89.OverlayValues[83] = d83
				ps89.OverlayValues[84] = d84
				ps89.OverlayValues[85] = d85
				ps89.OverlayValues[86] = d86
				ps89.OverlayValues[87] = d87
				ps89.OverlayValues[88] = d88
				return bbs[4].RenderPS(ps89)
			}
			if ps.General {
			}
			ps90 := scm.PhiState{General: ps.General}
			ps90.OverlayValues = make([]scm.JITValueDesc, 89)
			ps90.OverlayValues[0] = d0
			ps90.OverlayValues[1] = d1
			ps90.OverlayValues[2] = d2
			ps90.OverlayValues[3] = d3
			ps90.OverlayValues[4] = d4
			ps90.OverlayValues[5] = d5
			ps90.OverlayValues[6] = d6
			ps90.OverlayValues[7] = d7
			ps90.OverlayValues[8] = d8
			ps90.OverlayValues[9] = d9
			ps90.OverlayValues[10] = d10
			ps90.OverlayValues[11] = d11
			ps90.OverlayValues[12] = d12
			ps90.OverlayValues[13] = d13
			ps90.OverlayValues[14] = d14
			ps90.OverlayValues[15] = d15
			ps90.OverlayValues[16] = d16
			ps90.OverlayValues[17] = d17
			ps90.OverlayValues[18] = d18
			ps90.OverlayValues[19] = d19
			ps90.OverlayValues[20] = d20
			ps90.OverlayValues[21] = d21
			ps90.OverlayValues[22] = d22
			ps90.OverlayValues[23] = d23
			ps90.OverlayValues[24] = d24
			ps90.OverlayValues[81] = d81
			ps90.OverlayValues[82] = d82
			ps90.OverlayValues[83] = d83
			ps90.OverlayValues[84] = d84
			ps90.OverlayValues[85] = d85
			ps90.OverlayValues[86] = d86
			ps90.OverlayValues[87] = d87
			ps90.OverlayValues[88] = d88
			return bbs[5].RenderPS(ps90)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl10 := ctx.ReserveLabel()
		lbl11 := ctx.ReserveLabel()
		ctx.EmitJump(d88.Condition, lbl10)
		ctx.EmitJmp(lbl11)
		ctx.FreeDesc(&d87)
		snap91 := d0
		snap92 := d1
		snap93 := d2
		snap94 := d3
		snap95 := d4
		snap96 := d5
		snap97 := d6
		snap98 := d7
		snap99 := d8
		snap100 := d9
		snap101 := d10
		snap102 := d11
		snap103 := d12
		snap104 := d13
		snap105 := d14
		snap106 := d15
		snap107 := d16
		snap108 := d17
		snap109 := d18
		snap110 := d19
		snap111 := d20
		snap112 := d21
		snap113 := d22
		snap114 := d23
		snap115 := d24
		snap116 := d81
		snap117 := d82
		snap118 := d83
		snap119 := d84
		snap120 := d85
		snap121 := d86
		snap122 := d87
		snap123 := d88
		alloc124 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl5)
		ctx.RestoreAllocState(alloc124)
		d0 = snap91
		d1 = snap92
		d2 = snap93
		d3 = snap94
		d4 = snap95
		d5 = snap96
		d6 = snap97
		d7 = snap98
		d8 = snap99
		d9 = snap100
		d10 = snap101
		d11 = snap102
		d12 = snap103
		d13 = snap104
		d14 = snap105
		d15 = snap106
		d16 = snap107
		d17 = snap108
		d18 = snap109
		d19 = snap110
		d20 = snap111
		d21 = snap112
		d22 = snap113
		d23 = snap114
		d24 = snap115
		d81 = snap116
		d82 = snap117
		d83 = snap118
		d84 = snap119
		d85 = snap120
		d86 = snap121
		d87 = snap122
		d88 = snap123
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc124)
		d0 = snap91
		d1 = snap92
		d2 = snap93
		d3 = snap94
		d4 = snap95
		d5 = snap96
		d6 = snap97
		d7 = snap98
		d8 = snap99
		d9 = snap100
		d10 = snap101
		d11 = snap102
		d12 = snap103
		d13 = snap104
		d14 = snap105
		d15 = snap106
		d16 = snap107
		d17 = snap108
		d18 = snap109
		d19 = snap110
		d20 = snap111
		d21 = snap112
		d22 = snap113
		d23 = snap114
		d24 = snap115
		d81 = snap116
		d82 = snap117
		d83 = snap118
		d84 = snap119
		d85 = snap120
		d86 = snap121
		d87 = snap122
		d88 = snap123
		ps125 := scm.PhiState{General: true}
		ps125.OverlayValues = make([]scm.JITValueDesc, 89)
		ps125.OverlayValues[0] = d0
		ps125.OverlayValues[1] = d1
		ps125.OverlayValues[2] = d2
		ps125.OverlayValues[3] = d3
		ps125.OverlayValues[4] = d4
		ps125.OverlayValues[5] = d5
		ps125.OverlayValues[6] = d6
		ps125.OverlayValues[7] = d7
		ps125.OverlayValues[8] = d8
		ps125.OverlayValues[9] = d9
		ps125.OverlayValues[10] = d10
		ps125.OverlayValues[11] = d11
		ps125.OverlayValues[12] = d12
		ps125.OverlayValues[13] = d13
		ps125.OverlayValues[14] = d14
		ps125.OverlayValues[15] = d15
		ps125.OverlayValues[16] = d16
		ps125.OverlayValues[17] = d17
		ps125.OverlayValues[18] = d18
		ps125.OverlayValues[19] = d19
		ps125.OverlayValues[20] = d20
		ps125.OverlayValues[21] = d21
		ps125.OverlayValues[22] = d22
		ps125.OverlayValues[23] = d23
		ps125.OverlayValues[24] = d24
		ps125.OverlayValues[81] = d81
		ps125.OverlayValues[82] = d82
		ps125.OverlayValues[83] = d83
		ps125.OverlayValues[84] = d84
		ps125.OverlayValues[85] = d85
		ps125.OverlayValues[86] = d86
		ps125.OverlayValues[87] = d87
		ps125.OverlayValues[88] = d88
		ps126 := scm.PhiState{General: true}
		ps126.OverlayValues = make([]scm.JITValueDesc, 89)
		ps126.OverlayValues[0] = d0
		ps126.OverlayValues[1] = d1
		ps126.OverlayValues[2] = d2
		ps126.OverlayValues[3] = d3
		ps126.OverlayValues[4] = d4
		ps126.OverlayValues[5] = d5
		ps126.OverlayValues[6] = d6
		ps126.OverlayValues[7] = d7
		ps126.OverlayValues[8] = d8
		ps126.OverlayValues[9] = d9
		ps126.OverlayValues[10] = d10
		ps126.OverlayValues[11] = d11
		ps126.OverlayValues[12] = d12
		ps126.OverlayValues[13] = d13
		ps126.OverlayValues[14] = d14
		ps126.OverlayValues[15] = d15
		ps126.OverlayValues[16] = d16
		ps126.OverlayValues[17] = d17
		ps126.OverlayValues[18] = d18
		ps126.OverlayValues[19] = d19
		ps126.OverlayValues[20] = d20
		ps126.OverlayValues[21] = d21
		ps126.OverlayValues[22] = d22
		ps126.OverlayValues[23] = d23
		ps126.OverlayValues[24] = d24
		ps126.OverlayValues[81] = d81
		ps126.OverlayValues[82] = d82
		ps126.OverlayValues[83] = d83
		ps126.OverlayValues[84] = d84
		ps126.OverlayValues[85] = d85
		ps126.OverlayValues[86] = d86
		ps126.OverlayValues[87] = d87
		ps126.OverlayValues[88] = d88
		snap127 := d0
		snap128 := d1
		snap129 := d2
		snap130 := d3
		snap131 := d4
		snap132 := d5
		snap133 := d6
		snap134 := d7
		snap135 := d8
		snap136 := d9
		snap137 := d10
		snap138 := d11
		snap139 := d12
		snap140 := d13
		snap141 := d14
		snap142 := d15
		snap143 := d16
		snap144 := d17
		snap145 := d18
		snap146 := d19
		snap147 := d20
		snap148 := d21
		snap149 := d22
		snap150 := d23
		snap151 := d24
		snap152 := d81
		snap153 := d82
		snap154 := d83
		snap155 := d84
		snap156 := d85
		snap157 := d86
		snap158 := d87
		snap159 := d88
		alloc160 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps126)
		}
		ctx.RestoreAllocState(alloc160)
		d0 = snap127
		d1 = snap128
		d2 = snap129
		d3 = snap130
		d4 = snap131
		d5 = snap132
		d6 = snap133
		d7 = snap134
		d8 = snap135
		d9 = snap136
		d10 = snap137
		d11 = snap138
		d12 = snap139
		d13 = snap140
		d14 = snap141
		d15 = snap142
		d16 = snap143
		d17 = snap144
		d18 = snap145
		d19 = snap146
		d20 = snap147
		d21 = snap148
		d22 = snap149
		d23 = snap150
		d24 = snap151
		d81 = snap152
		d82 = snap153
		d83 = snap154
		d84 = snap155
		d85 = snap156
		d86 = snap157
		d87 = snap158
		d88 = snap159
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps125)
		}
		return result
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
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
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
		ctx.ReclaimUntrackedRegs()
		var d161 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88)
			r33 := ctx.AllocReg()
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			d161 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d161)
		}
		ctx.EnsureDesc(&d22)
		ctx.EnsureDesc(&d161)
		ctx.EnsureDescsTogether(&d22, &d161)
		var d162 scm.JITValueDesc
		if d22.Loc == scm.LocImm && d161.Loc == scm.LocImm {
			d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d22.Imm.Int()) == uint64(d161.Imm.Int()))}
		} else if d161.Loc == scm.LocImm {
			r34 := ctx.AllocRegExcept(d22.Reg)
			if d161.Imm.Int() >= -2147483648 && d161.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d22.Reg, int32(d161.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d161.Imm.Int()))
				ctx.EmitCmpInt64(d22.Reg, scm.RegR11)
			}
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r34, Condition: scm.CondEqual}
			ctx.BindReg(r34, &d162)
		} else if d22.Loc == scm.LocImm {
			r35 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d161.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondEqual}
			ctx.BindReg(r35, &d162)
		} else {
			r36 := ctx.AllocRegExcept(d22.Reg)
			ctx.EmitCmpInt64(d22.Reg, d161.Reg)
			d162 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r36, Condition: scm.CondEqual}
			ctx.BindReg(r36, &d162)
		}
		ctx.FreeDesc(&d161)
		d163 = d162
		ctx.EnsureDesc(&d163)
		if d163.Loc != scm.LocImm && d163.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d163.Loc == scm.LocImm {
			if d163.Imm.Bool() {
				if ps.General {
				}
				ps164 := scm.PhiState{General: ps.General}
				ps164.OverlayValues = make([]scm.JITValueDesc, 164)
				ps164.OverlayValues[0] = d0
				ps164.OverlayValues[1] = d1
				ps164.OverlayValues[2] = d2
				ps164.OverlayValues[3] = d3
				ps164.OverlayValues[4] = d4
				ps164.OverlayValues[5] = d5
				ps164.OverlayValues[6] = d6
				ps164.OverlayValues[7] = d7
				ps164.OverlayValues[8] = d8
				ps164.OverlayValues[9] = d9
				ps164.OverlayValues[10] = d10
				ps164.OverlayValues[11] = d11
				ps164.OverlayValues[12] = d12
				ps164.OverlayValues[13] = d13
				ps164.OverlayValues[14] = d14
				ps164.OverlayValues[15] = d15
				ps164.OverlayValues[16] = d16
				ps164.OverlayValues[17] = d17
				ps164.OverlayValues[18] = d18
				ps164.OverlayValues[19] = d19
				ps164.OverlayValues[20] = d20
				ps164.OverlayValues[21] = d21
				ps164.OverlayValues[22] = d22
				ps164.OverlayValues[23] = d23
				ps164.OverlayValues[24] = d24
				ps164.OverlayValues[81] = d81
				ps164.OverlayValues[82] = d82
				ps164.OverlayValues[83] = d83
				ps164.OverlayValues[84] = d84
				ps164.OverlayValues[85] = d85
				ps164.OverlayValues[86] = d86
				ps164.OverlayValues[87] = d87
				ps164.OverlayValues[88] = d88
				ps164.OverlayValues[161] = d161
				ps164.OverlayValues[162] = d162
				ps164.OverlayValues[163] = d163
				return bbs[1].RenderPS(ps164)
			}
			if ps.General {
			}
			ps165 := scm.PhiState{General: ps.General}
			ps165.OverlayValues = make([]scm.JITValueDesc, 164)
			ps165.OverlayValues[0] = d0
			ps165.OverlayValues[1] = d1
			ps165.OverlayValues[2] = d2
			ps165.OverlayValues[3] = d3
			ps165.OverlayValues[4] = d4
			ps165.OverlayValues[5] = d5
			ps165.OverlayValues[6] = d6
			ps165.OverlayValues[7] = d7
			ps165.OverlayValues[8] = d8
			ps165.OverlayValues[9] = d9
			ps165.OverlayValues[10] = d10
			ps165.OverlayValues[11] = d11
			ps165.OverlayValues[12] = d12
			ps165.OverlayValues[13] = d13
			ps165.OverlayValues[14] = d14
			ps165.OverlayValues[15] = d15
			ps165.OverlayValues[16] = d16
			ps165.OverlayValues[17] = d17
			ps165.OverlayValues[18] = d18
			ps165.OverlayValues[19] = d19
			ps165.OverlayValues[20] = d20
			ps165.OverlayValues[21] = d21
			ps165.OverlayValues[22] = d22
			ps165.OverlayValues[23] = d23
			ps165.OverlayValues[24] = d24
			ps165.OverlayValues[81] = d81
			ps165.OverlayValues[82] = d82
			ps165.OverlayValues[83] = d83
			ps165.OverlayValues[84] = d84
			ps165.OverlayValues[85] = d85
			ps165.OverlayValues[86] = d86
			ps165.OverlayValues[87] = d87
			ps165.OverlayValues[88] = d88
			ps165.OverlayValues[161] = d161
			ps165.OverlayValues[162] = d162
			ps165.OverlayValues[163] = d163
			return bbs[2].RenderPS(ps165)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		lbl12 := ctx.ReserveLabel()
		lbl13 := ctx.ReserveLabel()
		ctx.EmitJump(d163.Condition, lbl12)
		ctx.EmitJmp(lbl13)
		ctx.FreeDesc(&d162)
		snap166 := d0
		snap167 := d1
		snap168 := d2
		snap169 := d3
		snap170 := d4
		snap171 := d5
		snap172 := d6
		snap173 := d7
		snap174 := d8
		snap175 := d9
		snap176 := d10
		snap177 := d11
		snap178 := d12
		snap179 := d13
		snap180 := d14
		snap181 := d15
		snap182 := d16
		snap183 := d17
		snap184 := d18
		snap185 := d19
		snap186 := d20
		snap187 := d21
		snap188 := d22
		snap189 := d23
		snap190 := d24
		snap191 := d81
		snap192 := d82
		snap193 := d83
		snap194 := d84
		snap195 := d85
		snap196 := d86
		snap197 := d87
		snap198 := d88
		snap199 := d161
		snap200 := d162
		snap201 := d163
		alloc202 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl2)
		ctx.RestoreAllocState(alloc202)
		d0 = snap166
		d1 = snap167
		d2 = snap168
		d3 = snap169
		d4 = snap170
		d5 = snap171
		d6 = snap172
		d7 = snap173
		d8 = snap174
		d9 = snap175
		d10 = snap176
		d11 = snap177
		d12 = snap178
		d13 = snap179
		d14 = snap180
		d15 = snap181
		d16 = snap182
		d17 = snap183
		d18 = snap184
		d19 = snap185
		d20 = snap186
		d21 = snap187
		d22 = snap188
		d23 = snap189
		d24 = snap190
		d81 = snap191
		d82 = snap192
		d83 = snap193
		d84 = snap194
		d85 = snap195
		d86 = snap196
		d87 = snap197
		d88 = snap198
		d161 = snap199
		d162 = snap200
		d163 = snap201
		ctx.MarkLabel(lbl13)
		ctx.EmitJmp(lbl3)
		ctx.RestoreAllocState(alloc202)
		d0 = snap166
		d1 = snap167
		d2 = snap168
		d3 = snap169
		d4 = snap170
		d5 = snap171
		d6 = snap172
		d7 = snap173
		d8 = snap174
		d9 = snap175
		d10 = snap176
		d11 = snap177
		d12 = snap178
		d13 = snap179
		d14 = snap180
		d15 = snap181
		d16 = snap182
		d17 = snap183
		d18 = snap184
		d19 = snap185
		d20 = snap186
		d21 = snap187
		d22 = snap188
		d23 = snap189
		d24 = snap190
		d81 = snap191
		d82 = snap192
		d83 = snap193
		d84 = snap194
		d85 = snap195
		d86 = snap196
		d87 = snap197
		d88 = snap198
		d161 = snap199
		d162 = snap200
		d163 = snap201
		ps203 := scm.PhiState{General: true}
		ps203.OverlayValues = make([]scm.JITValueDesc, 164)
		ps203.OverlayValues[0] = d0
		ps203.OverlayValues[1] = d1
		ps203.OverlayValues[2] = d2
		ps203.OverlayValues[3] = d3
		ps203.OverlayValues[4] = d4
		ps203.OverlayValues[5] = d5
		ps203.OverlayValues[6] = d6
		ps203.OverlayValues[7] = d7
		ps203.OverlayValues[8] = d8
		ps203.OverlayValues[9] = d9
		ps203.OverlayValues[10] = d10
		ps203.OverlayValues[11] = d11
		ps203.OverlayValues[12] = d12
		ps203.OverlayValues[13] = d13
		ps203.OverlayValues[14] = d14
		ps203.OverlayValues[15] = d15
		ps203.OverlayValues[16] = d16
		ps203.OverlayValues[17] = d17
		ps203.OverlayValues[18] = d18
		ps203.OverlayValues[19] = d19
		ps203.OverlayValues[20] = d20
		ps203.OverlayValues[21] = d21
		ps203.OverlayValues[22] = d22
		ps203.OverlayValues[23] = d23
		ps203.OverlayValues[24] = d24
		ps203.OverlayValues[81] = d81
		ps203.OverlayValues[82] = d82
		ps203.OverlayValues[83] = d83
		ps203.OverlayValues[84] = d84
		ps203.OverlayValues[85] = d85
		ps203.OverlayValues[86] = d86
		ps203.OverlayValues[87] = d87
		ps203.OverlayValues[88] = d88
		ps203.OverlayValues[161] = d161
		ps203.OverlayValues[162] = d162
		ps203.OverlayValues[163] = d163
		ps204 := scm.PhiState{General: true}
		ps204.OverlayValues = make([]scm.JITValueDesc, 164)
		ps204.OverlayValues[0] = d0
		ps204.OverlayValues[1] = d1
		ps204.OverlayValues[2] = d2
		ps204.OverlayValues[3] = d3
		ps204.OverlayValues[4] = d4
		ps204.OverlayValues[5] = d5
		ps204.OverlayValues[6] = d6
		ps204.OverlayValues[7] = d7
		ps204.OverlayValues[8] = d8
		ps204.OverlayValues[9] = d9
		ps204.OverlayValues[10] = d10
		ps204.OverlayValues[11] = d11
		ps204.OverlayValues[12] = d12
		ps204.OverlayValues[13] = d13
		ps204.OverlayValues[14] = d14
		ps204.OverlayValues[15] = d15
		ps204.OverlayValues[16] = d16
		ps204.OverlayValues[17] = d17
		ps204.OverlayValues[18] = d18
		ps204.OverlayValues[19] = d19
		ps204.OverlayValues[20] = d20
		ps204.OverlayValues[21] = d21
		ps204.OverlayValues[22] = d22
		ps204.OverlayValues[23] = d23
		ps204.OverlayValues[24] = d24
		ps204.OverlayValues[81] = d81
		ps204.OverlayValues[82] = d82
		ps204.OverlayValues[83] = d83
		ps204.OverlayValues[84] = d84
		ps204.OverlayValues[85] = d85
		ps204.OverlayValues[86] = d86
		ps204.OverlayValues[87] = d87
		ps204.OverlayValues[88] = d88
		ps204.OverlayValues[161] = d161
		ps204.OverlayValues[162] = d162
		ps204.OverlayValues[163] = d163
		snap205 := d0
		snap206 := d1
		snap207 := d2
		snap208 := d3
		snap209 := d4
		snap210 := d5
		snap211 := d6
		snap212 := d7
		snap213 := d8
		snap214 := d9
		snap215 := d10
		snap216 := d11
		snap217 := d12
		snap218 := d13
		snap219 := d14
		snap220 := d15
		snap221 := d16
		snap222 := d17
		snap223 := d18
		snap224 := d19
		snap225 := d20
		snap226 := d21
		snap227 := d22
		snap228 := d23
		snap229 := d24
		snap230 := d81
		snap231 := d82
		snap232 := d83
		snap233 := d84
		snap234 := d85
		snap235 := d86
		snap236 := d87
		snap237 := d88
		snap238 := d161
		snap239 := d162
		snap240 := d163
		alloc241 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps204)
		}
		ctx.RestoreAllocState(alloc241)
		d0 = snap205
		d1 = snap206
		d2 = snap207
		d3 = snap208
		d4 = snap209
		d5 = snap210
		d6 = snap211
		d7 = snap212
		d8 = snap213
		d9 = snap214
		d10 = snap215
		d11 = snap216
		d12 = snap217
		d13 = snap218
		d14 = snap219
		d15 = snap220
		d16 = snap221
		d17 = snap222
		d18 = snap223
		d19 = snap224
		d20 = snap225
		d21 = snap226
		d22 = snap227
		d23 = snap228
		d24 = snap229
		d81 = snap230
		d82 = snap231
		d83 = snap232
		d84 = snap233
		d85 = snap234
		d86 = snap235
		d87 = snap236
		d88 = snap237
		d161 = snap238
		d162 = snap239
		d163 = snap240
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps203)
		}
		return result
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
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
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
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		ctx.ReclaimUntrackedRegs()
		var d242 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r37 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r37, thisptr.Reg, off)
			d242 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			ctx.BindReg(r37, &d242)
		}
		ctx.EnsureDesc(&d242)
		r38 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r38, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r39 := ctx.AllocReg()
		if d242.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r39, uint64(d242.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r39, d242.Reg)
			ctx.EmitShlRegImm8(r39, 3)
		}
		ctx.EmitAddInt64(r38, r39)
		ctx.FreeReg(r39)
		r40 := ctx.AllocRegExcept(r38)
		ctx.EmitMovRegMem(r40, r38, 0)
		ctx.FreeReg(r38)
		d243 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
		ctx.BindReg(r40, &d243)
		ctx.FreeDesc(&d242)
		ctx.EnsureDesc(&d85)
		ctx.EnsureDesc(&d243)
		ctx.EnsureDescsTogether(&d85, &d243)
		var d244 scm.JITValueDesc
		if d85.Loc == scm.LocImm && d243.Loc == scm.LocImm {
			d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() * d243.Imm.Int())}
		} else if d85.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d243.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
			ctx.EmitImulInt64(scratch, d243.Reg)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d244)
		} else if d243.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d85.Reg)
			ctx.EmitMovRegReg(scratch, d85.Reg)
			if d243.Imm.Int() >= -2147483648 && d243.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d243.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d243.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d244)
		} else {
			r41 := ctx.AllocRegExcept(d85.Reg, d243.Reg)
			ctx.EmitMovRegReg(r41, d85.Reg)
			ctx.EmitImulInt64(r41, d243.Reg)
			d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d244)
		}
		if d244.Loc == scm.LocReg && d85.Loc == scm.LocReg && d244.Reg == d85.Reg {
			ctx.TransferReg(d85.Reg)
			d85.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d243)
		ctx.EnsureDesc(&d244)
		d245 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d245)
		ctx.BindReg(r1, &d245)
		ctx.EnsureDesc(&d244)
		ctx.EmitMakeInt(d245, d244)
		if d244.Loc == scm.LocReg {
			ctx.FreeReg(d244.Reg)
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
		if len(ps.OverlayValues) > 81 && ps.OverlayValues[81].Loc != scm.LocNone {
			d81 = ps.OverlayValues[81]
		}
		if len(ps.OverlayValues) > 82 && ps.OverlayValues[82].Loc != scm.LocNone {
			d82 = ps.OverlayValues[82]
		}
		if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
			d83 = ps.OverlayValues[83]
		}
		if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
			d84 = ps.OverlayValues[84]
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
		if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != scm.LocNone {
			d161 = ps.OverlayValues[161]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 242 && ps.OverlayValues[242].Loc != scm.LocNone {
			d242 = ps.OverlayValues[242]
		}
		if len(ps.OverlayValues) > 243 && ps.OverlayValues[243].Loc != scm.LocNone {
			d243 = ps.OverlayValues[243]
		}
		if len(ps.OverlayValues) > 244 && ps.OverlayValues[244].Loc != scm.LocNone {
			d244 = ps.OverlayValues[244]
		}
		if len(ps.OverlayValues) > 245 && ps.OverlayValues[245].Loc != scm.LocNone {
			d245 = ps.OverlayValues[245]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d85)
		ctx.EnsureDesc(&d85)
		var d246 scm.JITValueDesc
		if d85.Loc == scm.LocImm {
			d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d85.Imm.Int()))}
		} else {
			r42 := ctx.AllocRegExcept(d85.Reg)
			ctx.EmitMovRegReg(r42, d85.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r42)
			d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r42}
			ctx.BindReg(r42, &d246)
		}
		var d247 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r43 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r43, thisptr.Reg, off)
			d247 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
			ctx.BindReg(r43, &d247)
		}
		ctx.EnsureDesc(&d247)
		ctx.EnsureDesc(&d247)
		var d248 scm.JITValueDesc
		if d247.Loc == scm.LocImm {
			d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d247.Imm.Int()))))}
		} else {
			r44 := ctx.AllocReg()
			ctx.EmitMovRegReg(r44, d247.Reg)
			ctx.EmitShlRegImm8(r44, 56)
			ctx.EmitSarRegImm8(r44, 56)
			d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			ctx.BindReg(r44, &d248)
		}
		ctx.FreeDesc(&d247)
		ctx.EnsureDesc(&d248)
		ctx.EnsureDesc(&d248)
		var d249 scm.JITValueDesc
		if d248.Loc == scm.LocImm {
			d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d248.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d248.Reg)
			ctx.EmitMovRegReg(scratch, d248.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d249)
		}
		if d249.Loc == scm.LocReg && d248.Loc == scm.LocReg && d249.Reg == d248.Reg {
			ctx.TransferReg(d248.Reg)
			d248.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d248)
		ctx.EnsureDesc(&d249)
		r45 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r45, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r46 := ctx.AllocReg()
		if d249.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r46, uint64(d249.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r46, d249.Reg)
			ctx.EmitShlRegImm8(r46, 3)
		}
		ctx.EmitAddInt64(r45, r46)
		ctx.FreeReg(r46)
		r47 := ctx.AllocRegExcept(r45)
		ctx.EmitMovRegMem(r47, r45, 0)
		ctx.FreeReg(r45)
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
		ctx.BindReg(r47, &d250)
		ctx.FreeDesc(&d249)
		ctx.EnsureDesc(&d246)
		ctx.EnsureDesc(&d250)
		ctx.EnsureDescsTogether(&d246, &d250)
		var d251 scm.JITValueDesc
		if d246.Loc == scm.LocImm && d250.Loc == scm.LocImm {
			d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d246.Imm.Float() * d250.Imm.Float())}
		} else if d246.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d250.Reg)
			_, xBits := d246.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d250.Reg)
			d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d251)
		} else if d250.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d246.Reg)
			ctx.EmitMovRegReg(scratch, d246.Reg)
			_, yBits := d250.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d251)
		} else {
			r48 := ctx.AllocRegExcept(d246.Reg, d250.Reg)
			ctx.EmitMovRegReg(r48, d246.Reg)
			ctx.EmitMulFloat64(r48, d250.Reg)
			d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r48}
			ctx.BindReg(r48, &d251)
		}
		if d251.Loc == scm.LocReg && d246.Loc == scm.LocReg && d251.Reg == d246.Reg {
			ctx.TransferReg(d246.Reg)
			d246.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d246)
		ctx.FreeDesc(&d250)
		ctx.EnsureDesc(&d251)
		d252 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d252)
		ctx.BindReg(r1, &d252)
		ctx.EnsureDesc(&d251)
		ctx.EmitMakeFloat(d252, d251)
		if d251.Loc == scm.LocReg {
			ctx.FreeReg(d251.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps253 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps253)
	ctx.MarkLabel(lbl0)
	d254 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d254)
	ctx.BindReg(r1, &d254)
	ctx.EmitMovPairToResult(&d254, &result)
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
