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
		// Scaled integer formats cannot retain IEEE negative zero. Keep it in
		// floating storage instead of changing the value during compression.
		if math.Signbit(f) {
			return math.MinInt8
		}
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
	return uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.inner)) + s.inner.ComputeSize()
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
//
//jitgen:control-flow-stable recid count target/3 stride
func (s *StorageDecimal) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.inner.GetValueRange(recid, count, target, stride)
	s.rescaleInPlace(target, count, stride)
}

//jitgen:control-flow-stable recids/3 target/3 stride
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
	var d43 scm.JITValueDesc
	_ = d43
	var d44 scm.JITValueDesc
	_ = d44
	var d45 scm.JITValueDesc
	_ = d45
	var d46 scm.JITValueDesc
	_ = d46
	var d47 scm.JITValueDesc
	_ = d47
	var d48 scm.JITValueDesc
	_ = d48
	var d49 scm.JITValueDesc
	_ = d49
	var d50 scm.JITValueDesc
	_ = d50
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d117 scm.JITValueDesc
	_ = d117
	var d118 scm.JITValueDesc
	_ = d118
	var d120 scm.JITValueDesc
	_ = d120
	var d121 scm.JITValueDesc
	_ = d121
	var d122 scm.JITValueDesc
	_ = d122
	var d123 scm.JITValueDesc
	_ = d123
	var d124 scm.JITValueDesc
	_ = d124
	var d125 scm.JITValueDesc
	_ = d125
	var d126 scm.JITValueDesc
	_ = d126
	var d128 scm.JITValueDesc
	_ = d128
	var d129 scm.JITValueDesc
	_ = d129
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
	bbs[0].Render = func() scm.JITValueDesc {
		if bbs[0].Rendered {
			ctx.EmitJmp(lbl1)
			return result
		}
		bbs[0].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl1)
		ctx.ResolveFixups()
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
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48)
			r0 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r0, thisptr.Reg, off)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			ctx.BindReg(r0, &d1)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		if d1.Loc == scm.LocImm {
			d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1.Imm.Int()))))}
		} else {
			r1 := ctx.AllocReg()
			ctx.EmitMovRegReg(r1, d1.Reg)
			ctx.EmitShlRegImm8(r1, 56)
			ctx.EmitShrRegImm8(r1, 56)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
			ctx.BindReg(r1, &d2)
		}
		ctx.FreeDesc(&d1)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d0, &d2)
		if d0.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() * d2.Imm.Int())}
		} else if d0.Loc == scm.LocImm {
			ctx.EnsureDesc(&d2)
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegReg(scratch, d2.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d0.Imm.Int())
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else if d2.Loc == scm.LocImm {
			ctx.EnsureDesc(&d0)
			scratch := ctx.AllocRegExcept(d0.Reg)
			ctx.EmitMovRegReg(scratch, d0.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d2.Imm.Int())
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else {
			ctx.EnsureDesc(&d0)
			ctx.SyncDesc(&d2)
			r2 := ctx.AllocRegExcept(d0.Reg, d2.Reg)
			ctx.EmitMovRegReg(r2, d0.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r2, &d2)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d4)
		}
		if d4.Loc == scm.LocReg && d0.Loc == scm.LocReg && d4.Reg == d0.Reg {
			ctx.TransferReg(d0.Reg)
			d0.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		if d4.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
		} else {
			r3 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r3, d4.Reg)
			ctx.EmitShrRegImm8(r3, 6)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d5)
		}
		if d5.Loc == scm.LocReg && d4.Loc == scm.LocReg && d5.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		if d4.Loc == scm.LocImm {
			d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
		} else {
			r4 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r4, d4.Reg)
			ctx.EmitAndRegImm32(r4, 63)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d6)
		}
		if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d4)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d7 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r5 := ctx.AllocReg()
			r6 := ctx.AllocRegExcept(r5)
			r7 := ctx.AllocRegExcept(r5, r6)
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
			ctx.EmitMovRegMem(r5, thisptr.Reg, off)
			ctx.EmitMovRegMem(r6, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r5, Reg2: r6, Reg3: r7}
			ctx.BindReg(r5, &d7)
			ctx.BindReg(r6, &d7)
			ctx.BindReg(r7, &d7)
			ctx.BindReg(r5, &d7)
			ctx.BindReg(r6, &d7)
			ctx.BindReg(r7, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		d8 = ctx.EmitLoadScalarSliceElement(&d7, &d5, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d8, &d6)
		if d8.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d6.Imm.Int())))}
		} else if d6.Loc == scm.LocImm {
			r8 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r8, d8.Reg)
			ctx.EmitShlRegImm8(r8, uint8(d6.Imm.Int()))
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d9)
		} else {
			shiftSrc := d8.Reg
			r9 := ctx.AllocRegExcept(d8.Reg, d6.Reg)
			ctx.EmitMovRegReg(r9, d8.Reg)
			shiftSrc = r9
			ctx.EmitShiftLeft(shiftSrc, d6.Reg, true)
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d9)
		}
		if d9.Loc == scm.LocReg && d8.Loc == scm.LocReg && d9.Reg == d8.Reg {
			ctx.TransferReg(d8.Reg)
			d8.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d8)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.EnsureDesc(&d5)
		if d5.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d10)
		}
		if d10.Loc == scm.LocReg && d5.Loc == scm.LocReg && d10.Reg == d5.Reg {
			ctx.TransferReg(d5.Reg)
			d5.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d10)
		ctx.ReclaimUntrackedRegs()
		d11 = ctx.EmitLoadScalarSliceElement(&d7, &d10, 8, scm.TagInt)
		ctx.FreeDesc(&d10)
		ctx.ReclaimUntrackedRegs()
		d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d12, &d6)
		if d12.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - d6.Imm.Int())}
		} else if d6.Loc == scm.LocImm && d6.Imm.Int() == 0 {
			ctx.EnsureDesc(&d12)
			r10 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r10, d12.Reg)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d13)
		} else if d12.Loc == scm.LocImm {
			ctx.EnsureDesc(&d6)
			scratch := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d6)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d13)
		} else if d6.Loc == scm.LocImm {
			ctx.EnsureDesc(&d12)
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d6.Imm.Int())
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d13)
		} else {
			ctx.EnsureDesc(&d12)
			ctx.SyncDesc(&d6)
			r11 := ctx.AllocRegExcept(d12.Reg, d6.Reg)
			ctx.EmitMovRegReg(r11, d12.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r11, &d6)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d13)
		}
		if d13.Loc == scm.LocReg && d12.Loc == scm.LocReg && d13.Reg == d12.Reg {
			ctx.TransferReg(d12.Reg)
			d12.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d6)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d11)
		ctx.EnsureDesc(&d13)
		ctx.EnsureDescsTogether(&d11, &d13)
		if d11.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) >> uint64(d13.Imm.Int())))}
		} else if d13.Loc == scm.LocImm {
			r12 := ctx.AllocRegExcept(d11.Reg)
			ctx.EmitMovRegReg(r12, d11.Reg)
			ctx.EmitShrRegImm8(r12, uint8(d13.Imm.Int()))
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d14)
		} else {
			shiftSrc := d11.Reg
			r13 := ctx.AllocRegExcept(d11.Reg, d13.Reg)
			ctx.EmitMovRegReg(r13, d11.Reg)
			shiftSrc = r13
			ctx.EmitShiftRight(shiftSrc, d13.Reg, false)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d14)
		}
		if d14.Loc == scm.LocReg && d11.Loc == scm.LocReg && d14.Reg == d11.Reg {
			ctx.TransferReg(d11.Reg)
			d11.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d11)
		ctx.FreeDesc(&d13)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d9)
		ctx.EnsureDesc(&d14)
		if d9.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d14.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			ctx.BindReg(d14.Reg, &d15)
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r14 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r14, d9.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d15)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
			ctx.EmitOrInt64(scratch, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d14.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r15, d9.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r15, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d14.Imm.Int()))
				ctx.EmitOrInt64(r15, ctx.ScratchReg)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d15)
		} else {
			r16 := ctx.AllocRegExcept(d9.Reg, d14.Reg)
			ctx.EmitMovRegReg(r16, d9.Reg)
			ctx.EmitOrInt64(r16, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d15)
		}
		if d15.Loc == scm.LocReg && d9.Loc == scm.LocReg && d15.Reg == d9.Reg {
			ctx.TransferReg(d9.Reg)
			d9.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d9)
		ctx.FreeDesc(&d14)
		ctx.ReclaimUntrackedRegs()
		d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d16, &d2)
		if d16.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			ctx.EnsureDesc(&d16)
			r17 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r17, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d17)
		} else if d16.Loc == scm.LocImm {
			ctx.EnsureDesc(&d2)
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d2)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d2.Loc == scm.LocImm {
			ctx.EnsureDesc(&d16)
			scratch := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(scratch, d16.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d2.Imm.Int())
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else {
			ctx.EnsureDesc(&d16)
			ctx.SyncDesc(&d2)
			r18 := ctx.AllocRegExcept(d16.Reg, d2.Reg)
			ctx.EmitMovRegReg(r18, d16.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r18, &d2)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d17)
		}
		if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
			ctx.TransferReg(d16.Reg)
			d16.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d2)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d15)
		ctx.EnsureDesc(&d17)
		ctx.EnsureDescsTogether(&d15, &d17)
		if d15.Loc == scm.LocImm && d17.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) >> uint64(d17.Imm.Int())))}
		} else if d17.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegReg(r19, d15.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d17.Imm.Int()))
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d18)
		} else {
			shiftSrc := d15.Reg
			r20 := ctx.AllocRegExcept(d15.Reg, d17.Reg)
			ctx.EmitMovRegReg(r20, d15.Reg)
			shiftSrc = r20
			ctx.EmitShiftRight(shiftSrc, d17.Reg, false)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d18)
		}
		if d18.Loc == scm.LocReg && d15.Loc == scm.LocReg && d18.Reg == d15.Reg {
			ctx.TransferReg(d15.Reg)
			d15.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d15)
		ctx.FreeDesc(&d17)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.StabilizeDescForControlFlow(&d18)
		ctx.FreeDesc(&idxInt)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80)
			r21 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r21, thisptr.Reg, off)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d19)
		}
		d20 = d19
		ctx.EnsureDesc(&d20)
		if d20.Loc != scm.LocImm && d20.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d20.Loc == scm.LocImm {
			if d20.Imm.Bool() {
				return bbs[3].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitCmpRegImm32(d20.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl4)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap21 := d0
			snap22 := d1
			snap23 := d2
			snap24 := d3
			snap25 := d4
			snap26 := d5
			snap27 := d6
			snap28 := d7
			snap29 := d8
			snap30 := d9
			snap31 := d10
			snap32 := d11
			snap33 := d12
			snap34 := d13
			snap35 := d14
			snap36 := d15
			snap37 := d16
			snap38 := d17
			snap39 := d18
			snap40 := d19
			snap41 := d20
			alloc42 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc42)
			d0 = snap21
			d1 = snap22
			d2 = snap23
			d3 = snap24
			d4 = snap25
			d5 = snap26
			d6 = snap27
			d7 = snap28
			d8 = snap29
			d9 = snap30
			d10 = snap31
			d11 = snap32
			d12 = snap33
			d13 = snap34
			d14 = snap35
			d15 = snap36
			d16 = snap37
			d17 = snap38
			d18 = snap39
			d19 = snap40
			d20 = snap41
		}
		if !bbs[3].Rendered {
			return bbs[3].Render()
		}
		return result
		ctx.FreeDesc(&d19)
		return result
	}
	bbs[1].Render = func() scm.JITValueDesc {
		if bbs[1].Rendered {
			ctx.EmitJmp(lbl2)
			return result
		}
		bbs[1].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl2)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d44 = result
		ctx.EnsureDesc(&d43)
		if d43.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d43, &d44)
		} else {
			switch d43.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d44, d43)
			case scm.TagInt:
				ctx.EmitMakeInt(d44, d43)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d44, d43)
			case scm.TagNil:
				ctx.EmitMakeNil(d44)
			default:
				ctx.EmitMovPairToResult(&d43, &d44)
			}
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[2].Render = func() scm.JITValueDesc {
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
			return result
		}
		bbs[2].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl3)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d18)
		if d18.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d18.Imm.Int()))))}
		} else {
			r22 := ctx.AllocReg()
			ctx.EmitMovRegReg(r22, d18.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d45)
		}
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMem(r23, thisptr.Reg, off)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d46)
		}
		ctx.EnsureDesc(&d45)
		ctx.EnsureDesc(&d46)
		ctx.EnsureDescsTogether(&d45, &d46)
		if d45.Loc == scm.LocImm && d46.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() + d46.Imm.Int())}
		} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
			ctx.EnsureDesc(&d45)
			r24 := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(r24, d45.Reg)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d47)
		} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
			ctx.EnsureDesc(&d46)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
			ctx.BindReg(d46.Reg, &d47)
		} else if d45.Loc == scm.LocImm {
			ctx.EnsureDesc(&d46)
			scratch := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(scratch, d46.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d45.Imm.Int())
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d47)
		} else if d46.Loc == scm.LocImm {
			ctx.EnsureDesc(&d45)
			scratch := ctx.AllocRegExcept(d45.Reg)
			ctx.EmitMovRegReg(scratch, d45.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d46.Imm.Int())
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d47)
		} else {
			ctx.EnsureDesc(&d45)
			ctx.SyncDesc(&d46)
			r25 := ctx.AllocRegExcept(d45.Reg, d46.Reg)
			ctx.EmitMovRegReg(r25, d45.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r25, &d46)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d47)
		}
		if d47.Loc == scm.LocReg && d45.Loc == scm.LocReg && d47.Reg == d45.Reg {
			ctx.TransferReg(d45.Reg)
			d45.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d47)
		ctx.FreeDesc(&d45)
		ctx.FreeDesc(&d46)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r26 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d48)
		}
		ctx.EnsureDesc(&d48)
		if d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d48.Imm.Int() > 0)}
		} else {
			r27 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitCmpRegImm32(d48.Reg, 0)
			d49 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r27, Condition: scm.CondSignedGreater}
			ctx.BindReg(r27, &d49)
		}
		ctx.FreeDesc(&d48)
		d50 = d49
		ctx.EnsureDesc(&d50)
		if d50.Loc != scm.LocImm && d50.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d50.Loc == scm.LocImm {
			if d50.Imm.Bool() {
				return bbs[4].Render()
			}
			return bbs[5].Render()
		}
		ctx.EmitJump(d50.Condition, lbl5)
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
		}
		ctx.FreeDesc(&d49)
		ctx.FlushRegisterMoves()
		if !bbs[5].Rendered {
			snap51 := d0
			snap52 := d1
			snap53 := d2
			snap54 := d3
			snap55 := d4
			snap56 := d5
			snap57 := d6
			snap58 := d7
			snap59 := d8
			snap60 := d9
			snap61 := d10
			snap62 := d11
			snap63 := d12
			snap64 := d13
			snap65 := d14
			snap66 := d15
			snap67 := d16
			snap68 := d17
			snap69 := d18
			snap70 := d19
			snap71 := d20
			snap72 := d43
			snap73 := d44
			snap74 := d45
			snap75 := d46
			snap76 := d47
			snap77 := d48
			snap78 := d49
			snap79 := d50
			alloc80 := ctx.SnapshotAllocState()
			bbs[5].Render()
			ctx.RestoreAllocState(alloc80)
			d0 = snap51
			d1 = snap52
			d2 = snap53
			d3 = snap54
			d4 = snap55
			d5 = snap56
			d6 = snap57
			d7 = snap58
			d8 = snap59
			d9 = snap60
			d10 = snap61
			d11 = snap62
			d12 = snap63
			d13 = snap64
			d14 = snap65
			d15 = snap66
			d16 = snap67
			d17 = snap68
			d18 = snap69
			d19 = snap70
			d20 = snap71
			d43 = snap72
			d44 = snap73
			d45 = snap74
			d46 = snap75
			d47 = snap76
			d48 = snap77
			d49 = snap78
			d50 = snap79
		}
		if !bbs[4].Rendered {
			return bbs[4].Render()
		}
		return result
		return result
	}
	bbs[3].Render = func() scm.JITValueDesc {
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
			return result
		}
		bbs[3].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl4)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88)
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMem(r28, thisptr.Reg, off)
			d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d81)
		}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d81)
		ctx.EnsureDescsTogether(&d18, &d81)
		if d18.Loc == scm.LocImm && d81.Loc == scm.LocImm {
			d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d18.Imm.Int()) == uint64(d81.Imm.Int()))}
		} else if d81.Loc == scm.LocImm {
			r29 := ctx.AllocRegExcept(d18.Reg)
			if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d18.Reg, int32(d81.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d81.Imm.Int()))
				ctx.EmitCmpInt64(d18.Reg, ctx.ScratchReg)
			}
			d82 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r29, Condition: scm.CondEqual}
			ctx.BindReg(r29, &d82)
		} else if d18.Loc == scm.LocImm {
			r30 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d18.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d81.Reg)
			d82 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r30, Condition: scm.CondEqual}
			ctx.BindReg(r30, &d82)
		} else {
			r31 := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitCmpInt64(d18.Reg, d81.Reg)
			d82 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r31, Condition: scm.CondEqual}
			ctx.BindReg(r31, &d82)
		}
		ctx.FreeDesc(&d81)
		d83 = d82
		ctx.EnsureDesc(&d83)
		if d83.Loc != scm.LocImm && d83.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d83.Loc == scm.LocImm {
			if d83.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitJump(d83.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d82)
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap84 := d0
			snap85 := d1
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
			snap105 := d43
			snap106 := d44
			snap107 := d45
			snap108 := d46
			snap109 := d47
			snap110 := d48
			snap111 := d49
			snap112 := d50
			snap113 := d81
			snap114 := d82
			snap115 := d83
			alloc116 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc116)
			d0 = snap84
			d1 = snap85
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
			d43 = snap105
			d44 = snap106
			d45 = snap107
			d46 = snap108
			d47 = snap109
			d48 = snap110
			d49 = snap111
			d50 = snap112
			d81 = snap113
			d82 = snap114
			d83 = snap115
		}
		if !bbs[1].Rendered {
			return bbs[1].Render()
		}
		return result
		return result
	}
	bbs[4].Render = func() scm.JITValueDesc {
		if bbs[4].Rendered {
			ctx.EmitJmp(lbl5)
			return result
		}
		bbs[4].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_4 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl5)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r32 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r32, thisptr.Reg, off)
			d117 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d117)
		}
		ctx.EnsureDesc(&d117)
		r33 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r33, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r34 := ctx.AllocReg()
		if d117.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r34, uint64(d117.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r34, d117.Reg)
			ctx.EmitShlRegImm8(r34, 3)
		}
		ctx.EmitAddInt64(r33, r34)
		ctx.FreeReg(r34)
		r35 := ctx.AllocRegExcept(r33)
		ctx.EmitMovRegMem(r35, r33, 0)
		ctx.FreeReg(r33)
		d118 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
		ctx.BindReg(r35, &d118)
		ctx.FreeDesc(&d117)
		ctx.EnsureDesc(&d47)
		resultTarget119 := false
		_ = resultTarget119
		ctx.EnsureDesc(&d118)
		ctx.EnsureDescsTogether(&d47, &d118)
		if d47.Loc == scm.LocImm && d118.Loc == scm.LocImm {
			d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() * d118.Imm.Int())}
		} else if d47.Loc == scm.LocImm {
			ctx.EnsureDesc(&d118)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d118.Reg {
				scratch = result.Reg2
				resultTarget119 = true
			} else {
				scratch = ctx.AllocRegExcept(d118.Reg)
			}
			ctx.EmitMovRegReg(scratch, d118.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d47.Imm.Int())
			d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d120)
		} else if d118.Loc == scm.LocImm {
			ctx.EnsureDesc(&d47)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d47.Reg {
				scratch = result.Reg2
				resultTarget119 = true
			} else {
				scratch = ctx.AllocRegExcept(d47.Reg)
			}
			ctx.EmitMovRegReg(scratch, d47.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d118.Imm.Int())
			d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d120)
		} else {
			ctx.EnsureDesc(&d47)
			ctx.SyncDesc(&d118)
			var r36 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d47.Reg && result.Reg2 != d118.Reg {
				r36 = result.Reg2
				resultTarget119 = true
			} else {
				r36 = ctx.AllocRegExcept(d47.Reg, d118.Reg)
			}
			ctx.EmitMovRegReg(r36, d47.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r36, &d118)
			d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			ctx.BindReg(r36, &d120)
		}
		if d120.Loc == scm.LocReg && d47.Loc == scm.LocReg && d120.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		if resultTarget119 && d120.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d118)
		ctx.EnsureDesc(&d120)
		d121 = result
		ctx.EnsureDesc(&d120)
		ctx.EmitMakeInt(d121, d120)
		if d120.Loc == scm.LocReg {
			ctx.FreeReg(d120.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[5].Render = func() scm.JITValueDesc {
		if bbs[5].Rendered {
			ctx.EmitJmp(lbl6)
			return result
		}
		bbs[5].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_5 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl6)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d47)
		if d47.Loc == scm.LocImm {
			d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d47.Imm.Int()))}
		} else {
			var r37 scm.Reg
			r37 = ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r37, d47.Reg)
			ctx.EmitInt64ToFloatBits(r37)
			d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r37}
			ctx.BindReg(r37, &d122)
		}
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r38 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r38, thisptr.Reg, off)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d123)
		}
		ctx.EnsureDesc(&d123)
		ctx.EnsureDesc(&d123)
		if d123.Loc == scm.LocImm {
			d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d123.Imm.Int()))))}
		} else {
			r39 := ctx.AllocReg()
			ctx.EmitMovRegReg(r39, d123.Reg)
			ctx.EmitShlRegImm8(r39, 56)
			ctx.EmitSarRegImm8(r39, 56)
			d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
			ctx.BindReg(r39, &d124)
		}
		ctx.FreeDesc(&d123)
		ctx.EnsureDesc(&d124)
		ctx.EnsureDesc(&d124)
		if d124.Loc == scm.LocImm {
			d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d124.Reg)
			ctx.EmitMovRegReg(scratch, d124.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 15)
			d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d125)
		}
		if d125.Loc == scm.LocReg && d124.Loc == scm.LocReg && d125.Reg == d124.Reg {
			ctx.TransferReg(d124.Reg)
			d124.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d124)
		ctx.EnsureDesc(&d125)
		r40 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r40, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r41 := ctx.AllocReg()
		if d125.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r41, uint64(d125.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r41, d125.Reg)
			ctx.EmitShlRegImm8(r41, 3)
		}
		ctx.EmitAddInt64(r40, r41)
		ctx.FreeReg(r41)
		r42 := ctx.AllocRegExcept(r40)
		ctx.EmitMovRegMem(r42, r40, 0)
		ctx.FreeReg(r40)
		d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
		ctx.BindReg(r42, &d126)
		ctx.FreeDesc(&d125)
		ctx.EnsureDesc(&d122)
		resultTarget127 := false
		_ = resultTarget127
		ctx.EnsureDesc(&d126)
		ctx.EnsureDescsTogether(&d122, &d126)
		if d122.Loc == scm.LocImm && d126.Loc == scm.LocImm {
			d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d122.Imm.Float() * d126.Imm.Float())}
		} else if d122.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d126.Reg {
				scratch = result.Reg2
				resultTarget127 = true
			} else {
				scratch = ctx.AllocRegExcept(d126.Reg)
			}
			_, xBits := d122.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d126.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d128)
		} else if d126.Loc == scm.LocImm {
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d122.Reg {
				scratch = result.Reg2
				resultTarget127 = true
			} else {
				scratch = ctx.AllocRegExcept(d122.Reg)
			}
			ctx.EmitMovRegReg(scratch, d122.Reg)
			_, yBits := d126.Imm.RawWords()
			ctx.EmitMovRegImm64(ctx.ScratchReg, yBits)
			ctx.EmitMulFloat64(scratch, ctx.ScratchReg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d128)
		} else {
			var r43 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d122.Reg && result.Reg2 != d126.Reg {
				r43 = result.Reg2
				resultTarget127 = true
			} else {
				r43 = ctx.AllocRegExcept(d122.Reg, d126.Reg)
			}
			ctx.EmitMovRegReg(r43, d122.Reg)
			ctx.EmitMulFloat64(r43, d126.Reg)
			d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r43}
			ctx.BindReg(r43, &d128)
		}
		if d128.Loc == scm.LocReg && d122.Loc == scm.LocReg && d128.Reg == d122.Reg {
			ctx.TransferReg(d122.Reg)
			d122.Loc = scm.LocNone
		}
		if resultTarget127 && d128.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d122)
		ctx.FreeDesc(&d126)
		ctx.EnsureDesc(&d128)
		d129 = result
		ctx.EnsureDesc(&d128)
		ctx.EmitMakeFloat(d129, d128)
		if d128.Loc == scm.LocReg {
			ctx.FreeReg(d128.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	_ = bbs[0].Render()
	ctx.MarkLabel(lbl0)
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
