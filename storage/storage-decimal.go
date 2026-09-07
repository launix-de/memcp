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
	var d69 scm.JITValueDesc
	_ = d69
	var d70 scm.JITValueDesc
	_ = d70
	var d71 scm.JITValueDesc
	_ = d71
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
	var d141 scm.JITValueDesc
	_ = d141
	var d142 scm.JITValueDesc
	_ = d142
	var d143 scm.JITValueDesc
	_ = d143
	var d214 scm.JITValueDesc
	_ = d214
	var d215 scm.JITValueDesc
	_ = d215
	var d216 scm.JITValueDesc
	_ = d216
	var d217 scm.JITValueDesc
	_ = d217
	var d218 scm.JITValueDesc
	_ = d218
	var d219 scm.JITValueDesc
	_ = d219
	var d220 scm.JITValueDesc
	_ = d220
	var d221 scm.JITValueDesc
	_ = d221
	var d222 scm.JITValueDesc
	_ = d222
	var d223 scm.JITValueDesc
	_ = d223
	var d224 scm.JITValueDesc
	_ = d224
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
		ctx.ReclaimUntrackedRegs()
		var d1 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 48)
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r2, thisptr.Reg, off)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d1)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		ctx.EnsureDesc(&d1)
		var d2 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1.Imm.Int()))))}
		} else {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegReg(r3, d1.Reg)
			ctx.EmitShlRegImm8(r3, 56)
			ctx.EmitShrRegImm8(r3, 56)
			d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d2)
		}
		ctx.FreeDesc(&d1)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&d0, &d2)
		var d4 scm.JITValueDesc
		if d0.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() * d2.Imm.Int())}
		} else if d0.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
			ctx.EmitImulInt64(scratch, d2.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else if d2.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d0.Reg)
			ctx.EmitMovRegReg(scratch, d0.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d4)
		} else {
			r4 := ctx.AllocRegExcept(d0.Reg, d2.Reg)
			ctx.EmitMovRegReg(r4, d0.Reg)
			ctx.EmitImulInt64(r4, d2.Reg)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d4)
		}
		if d4.Loc == scm.LocReg && d0.Loc == scm.LocReg && d4.Reg == d0.Reg {
			ctx.TransferReg(d0.Reg)
			d0.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		var d5 scm.JITValueDesc
		if d4.Loc == scm.LocImm {
			d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
		} else {
			r5 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r5, d4.Reg)
			ctx.EmitShrRegImm8(r5, 6)
			d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d5)
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
			r6 := ctx.AllocRegExcept(d4.Reg)
			ctx.EmitMovRegReg(r6, d4.Reg)
			ctx.EmitAndRegImm32(r6, 63)
			d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d6)
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
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d7 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r7 := ctx.AllocReg()
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 24)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+16)
			d7 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r7, Reg2: r8, Reg3: r9}
			ctx.BindReg(r7, &d7)
			ctx.BindReg(r8, &d7)
			ctx.BindReg(r9, &d7)
			ctx.BindReg(r7, &d7)
			ctx.BindReg(r8, &d7)
			ctx.BindReg(r9, &d7)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d5)
		ctx.ReclaimUntrackedRegs()
		d8 = ctx.EmitLoadScalarSliceElement(&d7, &d5, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d8)
		ctx.EnsureDesc(&d6)
		ctx.EnsureDescsTogether(&d8, &d6)
		var d9 scm.JITValueDesc
		if d8.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d6.Imm.Int())))}
		} else if d6.Loc == scm.LocImm {
			r10 := ctx.AllocRegExcept(d8.Reg)
			ctx.EmitMovRegReg(r10, d8.Reg)
			ctx.EmitShlRegImm8(r10, uint8(d6.Imm.Int()))
			d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d9)
		} else {
			{
				shiftSrc := d8.Reg
				r11 := ctx.AllocRegExcept(d8.Reg, d6.Reg)
				ctx.EmitMovRegReg(r11, d8.Reg)
				shiftSrc = r11
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
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d9)
			}
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
		var d10 scm.JITValueDesc
		if d5.Loc == scm.LocImm {
			d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
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
		var d13 scm.JITValueDesc
		if d12.Loc == scm.LocImm && d6.Loc == scm.LocImm {
			d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() - d6.Imm.Int())}
		} else if d6.Loc == scm.LocImm && d6.Imm.Int() == 0 {
			r12 := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(r12, d12.Reg)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d13)
		} else if d12.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d6.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
			ctx.EmitSubInt64(scratch, d6.Reg)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d13)
		} else if d6.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d12.Reg)
			ctx.EmitMovRegReg(scratch, d12.Reg)
			if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d6.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d13)
		} else {
			r13 := ctx.AllocRegExcept(d12.Reg, d6.Reg)
			ctx.EmitMovRegReg(r13, d12.Reg)
			ctx.EmitSubInt64(r13, d6.Reg)
			d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d13)
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
		var d14 scm.JITValueDesc
		if d11.Loc == scm.LocImm && d13.Loc == scm.LocImm {
			d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) >> uint64(d13.Imm.Int())))}
		} else if d13.Loc == scm.LocImm {
			r14 := ctx.AllocRegExcept(d11.Reg)
			ctx.EmitMovRegReg(r14, d11.Reg)
			ctx.EmitShrRegImm8(r14, uint8(d13.Imm.Int()))
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d14)
		} else {
			{
				shiftSrc := d11.Reg
				r15 := ctx.AllocRegExcept(d11.Reg, d13.Reg)
				ctx.EmitMovRegReg(r15, d11.Reg)
				shiftSrc = r15
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d13.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d13.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d13.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d14)
			}
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
		var d15 scm.JITValueDesc
		if d9.Loc == scm.LocImm && d14.Loc == scm.LocImm {
			d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d14.Imm.Int())}
		} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			ctx.BindReg(d14.Reg, &d15)
		} else if d14.Loc == scm.LocImm && d14.Imm.Int() == 0 {
			r16 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r16, d9.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d15)
		} else if d9.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d14.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
			ctx.EmitOrInt64(scratch, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d15)
		} else if d14.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d9.Reg)
			ctx.EmitMovRegReg(r17, d9.Reg)
			if d14.Imm.Int() >= -2147483648 && d14.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r17, int32(d14.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.EmitOrInt64(r17, scm.RegR11)
			}
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d15)
		} else {
			r18 := ctx.AllocRegExcept(d9.Reg, d14.Reg)
			ctx.EmitMovRegReg(r18, d9.Reg)
			ctx.EmitOrInt64(r18, d14.Reg)
			d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d15)
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
		var d17 scm.JITValueDesc
		if d16.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			r19 := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(r19, d16.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d17)
		} else if d16.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
			ctx.EmitSubInt64(scratch, d2.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d2.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d16.Reg)
			ctx.EmitMovRegReg(scratch, d16.Reg)
			if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d2.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else {
			r20 := ctx.AllocRegExcept(d16.Reg, d2.Reg)
			ctx.EmitMovRegReg(r20, d16.Reg)
			ctx.EmitSubInt64(r20, d2.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d17)
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
		var d18 scm.JITValueDesc
		if d15.Loc == scm.LocImm && d17.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) >> uint64(d17.Imm.Int())))}
		} else if d17.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d15.Reg)
			ctx.EmitMovRegReg(r21, d15.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d17.Imm.Int()))
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d18)
		} else {
			{
				shiftSrc := d15.Reg
				r22 := ctx.AllocRegExcept(d15.Reg, d17.Reg)
				ctx.EmitMovRegReg(r22, d15.Reg)
				shiftSrc = r22
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d17.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d17.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d17.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d18)
			}
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
		var d19 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80
			val := *(*bool)(unsafe.Pointer(fieldAddr))
			d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 80)
			r23 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r23, thisptr.Reg, off)
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d19)
		}
		d20 = d19
		ctx.EnsureDesc(&d20)
		if d20.Loc != scm.LocImm && d20.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d20.Loc == scm.LocImm {
			if d20.Imm.Bool() {
				if ps.General {
				}
				ps21 := scm.PhiState{General: ps.General}
				ps21.OverlayValues = make([]scm.JITValueDesc, 21)
				ps21.OverlayValues[0] = d0
				ps21.OverlayValues[1] = d1
				ps21.OverlayValues[2] = d2
				ps21.OverlayValues[3] = d3
				ps21.OverlayValues[4] = d4
				ps21.OverlayValues[5] = d5
				ps21.OverlayValues[6] = d6
				ps21.OverlayValues[7] = d7
				ps21.OverlayValues[8] = d8
				ps21.OverlayValues[9] = d9
				ps21.OverlayValues[10] = d10
				ps21.OverlayValues[11] = d11
				ps21.OverlayValues[12] = d12
				ps21.OverlayValues[13] = d13
				ps21.OverlayValues[14] = d14
				ps21.OverlayValues[15] = d15
				ps21.OverlayValues[16] = d16
				ps21.OverlayValues[17] = d17
				ps21.OverlayValues[18] = d18
				ps21.OverlayValues[19] = d19
				ps21.OverlayValues[20] = d20
				return bbs[3].RenderPS(ps21)
			}
			if ps.General {
			}
			ps22 := scm.PhiState{General: ps.General}
			ps22.OverlayValues = make([]scm.JITValueDesc, 21)
			ps22.OverlayValues[0] = d0
			ps22.OverlayValues[1] = d1
			ps22.OverlayValues[2] = d2
			ps22.OverlayValues[3] = d3
			ps22.OverlayValues[4] = d4
			ps22.OverlayValues[5] = d5
			ps22.OverlayValues[6] = d6
			ps22.OverlayValues[7] = d7
			ps22.OverlayValues[8] = d8
			ps22.OverlayValues[9] = d9
			ps22.OverlayValues[10] = d10
			ps22.OverlayValues[11] = d11
			ps22.OverlayValues[12] = d12
			ps22.OverlayValues[13] = d13
			ps22.OverlayValues[14] = d14
			ps22.OverlayValues[15] = d15
			ps22.OverlayValues[16] = d16
			ps22.OverlayValues[17] = d17
			ps22.OverlayValues[18] = d18
			ps22.OverlayValues[19] = d19
			ps22.OverlayValues[20] = d20
			return bbs[2].RenderPS(ps22)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d20.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl4)
		snap23 := d0
		snap24 := d1
		snap25 := d2
		snap26 := d3
		snap27 := d4
		snap28 := d5
		snap29 := d6
		snap30 := d7
		snap31 := d8
		snap32 := d9
		snap33 := d10
		snap34 := d11
		snap35 := d12
		snap36 := d13
		snap37 := d14
		snap38 := d15
		snap39 := d16
		snap40 := d17
		snap41 := d18
		snap42 := d19
		snap43 := d20
		alloc44 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc44)
		d0 = snap23
		d1 = snap24
		d2 = snap25
		d3 = snap26
		d4 = snap27
		d5 = snap28
		d6 = snap29
		d7 = snap30
		d8 = snap31
		d9 = snap32
		d10 = snap33
		d11 = snap34
		d12 = snap35
		d13 = snap36
		d14 = snap37
		d15 = snap38
		d16 = snap39
		d17 = snap40
		d18 = snap41
		d19 = snap42
		d20 = snap43
		ctx.RestoreAllocState(alloc44)
		d0 = snap23
		d1 = snap24
		d2 = snap25
		d3 = snap26
		d4 = snap27
		d5 = snap28
		d6 = snap29
		d7 = snap30
		d8 = snap31
		d9 = snap32
		d10 = snap33
		d11 = snap34
		d12 = snap35
		d13 = snap36
		d14 = snap37
		d15 = snap38
		d16 = snap39
		d17 = snap40
		d18 = snap41
		d19 = snap42
		d20 = snap43
		ps45 := scm.PhiState{General: true}
		ps45.OverlayValues = make([]scm.JITValueDesc, 21)
		ps45.OverlayValues[0] = d0
		ps45.OverlayValues[1] = d1
		ps45.OverlayValues[2] = d2
		ps45.OverlayValues[3] = d3
		ps45.OverlayValues[4] = d4
		ps45.OverlayValues[5] = d5
		ps45.OverlayValues[6] = d6
		ps45.OverlayValues[7] = d7
		ps45.OverlayValues[8] = d8
		ps45.OverlayValues[9] = d9
		ps45.OverlayValues[10] = d10
		ps45.OverlayValues[11] = d11
		ps45.OverlayValues[12] = d12
		ps45.OverlayValues[13] = d13
		ps45.OverlayValues[14] = d14
		ps45.OverlayValues[15] = d15
		ps45.OverlayValues[16] = d16
		ps45.OverlayValues[17] = d17
		ps45.OverlayValues[18] = d18
		ps45.OverlayValues[19] = d19
		ps45.OverlayValues[20] = d20
		ps46 := scm.PhiState{General: true}
		ps46.OverlayValues = make([]scm.JITValueDesc, 21)
		ps46.OverlayValues[0] = d0
		ps46.OverlayValues[1] = d1
		ps46.OverlayValues[2] = d2
		ps46.OverlayValues[3] = d3
		ps46.OverlayValues[4] = d4
		ps46.OverlayValues[5] = d5
		ps46.OverlayValues[6] = d6
		ps46.OverlayValues[7] = d7
		ps46.OverlayValues[8] = d8
		ps46.OverlayValues[9] = d9
		ps46.OverlayValues[10] = d10
		ps46.OverlayValues[11] = d11
		ps46.OverlayValues[12] = d12
		ps46.OverlayValues[13] = d13
		ps46.OverlayValues[14] = d14
		ps46.OverlayValues[15] = d15
		ps46.OverlayValues[16] = d16
		ps46.OverlayValues[17] = d17
		ps46.OverlayValues[18] = d18
		ps46.OverlayValues[19] = d19
		ps46.OverlayValues[20] = d20
		snap47 := d0
		snap48 := d1
		snap49 := d2
		snap50 := d3
		snap51 := d4
		snap52 := d5
		snap53 := d6
		snap54 := d7
		snap55 := d8
		snap56 := d9
		snap57 := d10
		snap58 := d11
		snap59 := d12
		snap60 := d13
		snap61 := d14
		snap62 := d15
		snap63 := d16
		snap64 := d17
		snap65 := d18
		snap66 := d19
		snap67 := d20
		alloc68 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps46)
		}
		ctx.RestoreAllocState(alloc68)
		d0 = snap47
		d1 = snap48
		d2 = snap49
		d3 = snap50
		d4 = snap51
		d5 = snap52
		d6 = snap53
		d7 = snap54
		d8 = snap55
		d9 = snap56
		d10 = snap57
		d11 = snap58
		d12 = snap59
		d13 = snap60
		d14 = snap61
		d15 = snap62
		d16 = snap63
		d17 = snap64
		d18 = snap65
		d19 = snap66
		d20 = snap67
		if !bbs[3].Rendered {
			return bbs[3].RenderPS(ps45)
		}
		return result
		ctx.FreeDesc(&d19)
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
		ctx.ReclaimUntrackedRegs()
		d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d70 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d70)
		ctx.BindReg(r1, &d70)
		ctx.EnsureDesc(&d69)
		if d69.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d69, &d70)
		} else {
			switch d69.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d70, d69)
			case scm.TagInt:
				ctx.EmitMakeInt(d70, d69)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d70, d69)
			case scm.TagNil:
				ctx.EmitMakeNil(d70)
			default:
				ctx.EmitMovPairToResult(&d69, &d70)
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
		if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
			d69 = ps.OverlayValues[69]
		}
		if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
			d70 = ps.OverlayValues[70]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d18)
		var d71 scm.JITValueDesc
		if d18.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d18.Imm.Int()))))}
		} else {
			r24 := ctx.AllocReg()
			ctx.EmitMovRegReg(r24, d18.Reg)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d71)
		}
		var d72 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 56)
			r25 := ctx.AllocReg()
			ctx.EmitMovRegMem(r25, thisptr.Reg, off)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d72)
		}
		ctx.EnsureDesc(&d71)
		ctx.EnsureDesc(&d72)
		ctx.EnsureDescsTogether(&d71, &d72)
		var d73 scm.JITValueDesc
		if d71.Loc == scm.LocImm && d72.Loc == scm.LocImm {
			d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d71.Imm.Int() + d72.Imm.Int())}
		} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
			r26 := ctx.AllocRegExcept(d71.Reg)
			ctx.EmitMovRegReg(r26, d71.Reg)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d73)
		} else if d71.Loc == scm.LocImm && d71.Imm.Int() == 0 {
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d72.Reg}
			ctx.BindReg(d72.Reg, &d73)
		} else if d71.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d71.Imm.Int()))
			ctx.EmitAddInt64(scratch, d72.Reg)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d73)
		} else if d72.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d71.Reg)
			ctx.EmitMovRegReg(scratch, d71.Reg)
			if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d72.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d73)
		} else {
			r27 := ctx.AllocRegExcept(d71.Reg, d72.Reg)
			ctx.EmitMovRegReg(r27, d71.Reg)
			ctx.EmitAddInt64(r27, d72.Reg)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			ctx.BindReg(r27, &d73)
		}
		if d73.Loc == scm.LocReg && d71.Loc == scm.LocReg && d73.Reg == d71.Reg {
			ctx.TransferReg(d71.Reg)
			d71.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d73)
		ctx.FreeDesc(&d71)
		ctx.FreeDesc(&d72)
		var d74 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r28 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r28, thisptr.Reg, off)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.BindReg(r28, &d74)
		}
		ctx.EnsureDesc(&d74)
		var d75 scm.JITValueDesc
		if d74.Loc == scm.LocImm {
			d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d74.Imm.Int() > 0)}
		} else {
			r29 := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitCmpRegImm32(d74.Reg, 0)
			d75 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r29, Condition: scm.CondSignedGreater}
			ctx.BindReg(r29, &d75)
		}
		ctx.FreeDesc(&d74)
		d76 = d75
		ctx.EnsureDesc(&d76)
		if d76.Loc != scm.LocImm && d76.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d76.Loc == scm.LocImm {
			if d76.Imm.Bool() {
				if ps.General {
				}
				ps77 := scm.PhiState{General: ps.General}
				ps77.OverlayValues = make([]scm.JITValueDesc, 77)
				ps77.OverlayValues[0] = d0
				ps77.OverlayValues[1] = d1
				ps77.OverlayValues[2] = d2
				ps77.OverlayValues[3] = d3
				ps77.OverlayValues[4] = d4
				ps77.OverlayValues[5] = d5
				ps77.OverlayValues[6] = d6
				ps77.OverlayValues[7] = d7
				ps77.OverlayValues[8] = d8
				ps77.OverlayValues[9] = d9
				ps77.OverlayValues[10] = d10
				ps77.OverlayValues[11] = d11
				ps77.OverlayValues[12] = d12
				ps77.OverlayValues[13] = d13
				ps77.OverlayValues[14] = d14
				ps77.OverlayValues[15] = d15
				ps77.OverlayValues[16] = d16
				ps77.OverlayValues[17] = d17
				ps77.OverlayValues[18] = d18
				ps77.OverlayValues[19] = d19
				ps77.OverlayValues[20] = d20
				ps77.OverlayValues[69] = d69
				ps77.OverlayValues[70] = d70
				ps77.OverlayValues[71] = d71
				ps77.OverlayValues[72] = d72
				ps77.OverlayValues[73] = d73
				ps77.OverlayValues[74] = d74
				ps77.OverlayValues[75] = d75
				ps77.OverlayValues[76] = d76
				return bbs[4].RenderPS(ps77)
			}
			if ps.General {
			}
			ps78 := scm.PhiState{General: ps.General}
			ps78.OverlayValues = make([]scm.JITValueDesc, 77)
			ps78.OverlayValues[0] = d0
			ps78.OverlayValues[1] = d1
			ps78.OverlayValues[2] = d2
			ps78.OverlayValues[3] = d3
			ps78.OverlayValues[4] = d4
			ps78.OverlayValues[5] = d5
			ps78.OverlayValues[6] = d6
			ps78.OverlayValues[7] = d7
			ps78.OverlayValues[8] = d8
			ps78.OverlayValues[9] = d9
			ps78.OverlayValues[10] = d10
			ps78.OverlayValues[11] = d11
			ps78.OverlayValues[12] = d12
			ps78.OverlayValues[13] = d13
			ps78.OverlayValues[14] = d14
			ps78.OverlayValues[15] = d15
			ps78.OverlayValues[16] = d16
			ps78.OverlayValues[17] = d17
			ps78.OverlayValues[18] = d18
			ps78.OverlayValues[19] = d19
			ps78.OverlayValues[20] = d20
			ps78.OverlayValues[69] = d69
			ps78.OverlayValues[70] = d70
			ps78.OverlayValues[71] = d71
			ps78.OverlayValues[72] = d72
			ps78.OverlayValues[73] = d73
			ps78.OverlayValues[74] = d74
			ps78.OverlayValues[75] = d75
			ps78.OverlayValues[76] = d76
			return bbs[5].RenderPS(ps78)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitJump(d76.Condition, lbl5)
		ctx.FreeDesc(&d75)
		snap79 := d0
		snap80 := d1
		snap81 := d2
		snap82 := d3
		snap83 := d4
		snap84 := d5
		snap85 := d6
		snap86 := d7
		snap87 := d8
		snap88 := d9
		snap89 := d10
		snap90 := d11
		snap91 := d12
		snap92 := d13
		snap93 := d14
		snap94 := d15
		snap95 := d16
		snap96 := d17
		snap97 := d18
		snap98 := d19
		snap99 := d20
		snap100 := d69
		snap101 := d70
		snap102 := d71
		snap103 := d72
		snap104 := d73
		snap105 := d74
		snap106 := d75
		snap107 := d76
		alloc108 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc108)
		d0 = snap79
		d1 = snap80
		d2 = snap81
		d3 = snap82
		d4 = snap83
		d5 = snap84
		d6 = snap85
		d7 = snap86
		d8 = snap87
		d9 = snap88
		d10 = snap89
		d11 = snap90
		d12 = snap91
		d13 = snap92
		d14 = snap93
		d15 = snap94
		d16 = snap95
		d17 = snap96
		d18 = snap97
		d19 = snap98
		d20 = snap99
		d69 = snap100
		d70 = snap101
		d71 = snap102
		d72 = snap103
		d73 = snap104
		d74 = snap105
		d75 = snap106
		d76 = snap107
		ctx.RestoreAllocState(alloc108)
		d0 = snap79
		d1 = snap80
		d2 = snap81
		d3 = snap82
		d4 = snap83
		d5 = snap84
		d6 = snap85
		d7 = snap86
		d8 = snap87
		d9 = snap88
		d10 = snap89
		d11 = snap90
		d12 = snap91
		d13 = snap92
		d14 = snap93
		d15 = snap94
		d16 = snap95
		d17 = snap96
		d18 = snap97
		d19 = snap98
		d20 = snap99
		d69 = snap100
		d70 = snap101
		d71 = snap102
		d72 = snap103
		d73 = snap104
		d74 = snap105
		d75 = snap106
		d76 = snap107
		ps109 := scm.PhiState{General: true}
		ps109.OverlayValues = make([]scm.JITValueDesc, 77)
		ps109.OverlayValues[0] = d0
		ps109.OverlayValues[1] = d1
		ps109.OverlayValues[2] = d2
		ps109.OverlayValues[3] = d3
		ps109.OverlayValues[4] = d4
		ps109.OverlayValues[5] = d5
		ps109.OverlayValues[6] = d6
		ps109.OverlayValues[7] = d7
		ps109.OverlayValues[8] = d8
		ps109.OverlayValues[9] = d9
		ps109.OverlayValues[10] = d10
		ps109.OverlayValues[11] = d11
		ps109.OverlayValues[12] = d12
		ps109.OverlayValues[13] = d13
		ps109.OverlayValues[14] = d14
		ps109.OverlayValues[15] = d15
		ps109.OverlayValues[16] = d16
		ps109.OverlayValues[17] = d17
		ps109.OverlayValues[18] = d18
		ps109.OverlayValues[19] = d19
		ps109.OverlayValues[20] = d20
		ps109.OverlayValues[69] = d69
		ps109.OverlayValues[70] = d70
		ps109.OverlayValues[71] = d71
		ps109.OverlayValues[72] = d72
		ps109.OverlayValues[73] = d73
		ps109.OverlayValues[74] = d74
		ps109.OverlayValues[75] = d75
		ps109.OverlayValues[76] = d76
		ps110 := scm.PhiState{General: true}
		ps110.OverlayValues = make([]scm.JITValueDesc, 77)
		ps110.OverlayValues[0] = d0
		ps110.OverlayValues[1] = d1
		ps110.OverlayValues[2] = d2
		ps110.OverlayValues[3] = d3
		ps110.OverlayValues[4] = d4
		ps110.OverlayValues[5] = d5
		ps110.OverlayValues[6] = d6
		ps110.OverlayValues[7] = d7
		ps110.OverlayValues[8] = d8
		ps110.OverlayValues[9] = d9
		ps110.OverlayValues[10] = d10
		ps110.OverlayValues[11] = d11
		ps110.OverlayValues[12] = d12
		ps110.OverlayValues[13] = d13
		ps110.OverlayValues[14] = d14
		ps110.OverlayValues[15] = d15
		ps110.OverlayValues[16] = d16
		ps110.OverlayValues[17] = d17
		ps110.OverlayValues[18] = d18
		ps110.OverlayValues[19] = d19
		ps110.OverlayValues[20] = d20
		ps110.OverlayValues[69] = d69
		ps110.OverlayValues[70] = d70
		ps110.OverlayValues[71] = d71
		ps110.OverlayValues[72] = d72
		ps110.OverlayValues[73] = d73
		ps110.OverlayValues[74] = d74
		ps110.OverlayValues[75] = d75
		ps110.OverlayValues[76] = d76
		snap111 := d0
		snap112 := d1
		snap113 := d2
		snap114 := d3
		snap115 := d4
		snap116 := d5
		snap117 := d6
		snap118 := d7
		snap119 := d8
		snap120 := d9
		snap121 := d10
		snap122 := d11
		snap123 := d12
		snap124 := d13
		snap125 := d14
		snap126 := d15
		snap127 := d16
		snap128 := d17
		snap129 := d18
		snap130 := d19
		snap131 := d20
		snap132 := d69
		snap133 := d70
		snap134 := d71
		snap135 := d72
		snap136 := d73
		snap137 := d74
		snap138 := d75
		snap139 := d76
		alloc140 := ctx.SnapshotAllocState()
		if !bbs[5].Rendered {
			bbs[5].RenderPS(ps110)
		}
		ctx.RestoreAllocState(alloc140)
		d0 = snap111
		d1 = snap112
		d2 = snap113
		d3 = snap114
		d4 = snap115
		d5 = snap116
		d6 = snap117
		d7 = snap118
		d8 = snap119
		d9 = snap120
		d10 = snap121
		d11 = snap122
		d12 = snap123
		d13 = snap124
		d14 = snap125
		d15 = snap126
		d16 = snap127
		d17 = snap128
		d18 = snap129
		d19 = snap130
		d20 = snap131
		d69 = snap132
		d70 = snap133
		d71 = snap134
		d72 = snap135
		d73 = snap136
		d74 = snap137
		d75 = snap138
		d76 = snap139
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps109)
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
		if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
			d69 = ps.OverlayValues[69]
		}
		if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
			d70 = ps.OverlayValues[70]
		}
		if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
			d71 = ps.OverlayValues[71]
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
		ctx.ReclaimUntrackedRegs()
		var d141 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88
			val := *(*uint64)(unsafe.Pointer(fieldAddr))
			d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).inner) + 88)
			r30 := ctx.AllocReg()
			ctx.EmitMovRegMem(r30, thisptr.Reg, off)
			d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d141)
		}
		ctx.EnsureDesc(&d18)
		ctx.EnsureDesc(&d141)
		ctx.EnsureDescsTogether(&d18, &d141)
		var d142 scm.JITValueDesc
		if d18.Loc == scm.LocImm && d141.Loc == scm.LocImm {
			d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d18.Imm.Int()) == uint64(d141.Imm.Int()))}
		} else if d141.Loc == scm.LocImm {
			r31 := ctx.AllocRegExcept(d18.Reg)
			if d141.Imm.Int() >= -2147483648 && d141.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d18.Reg, int32(d141.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.EmitCmpInt64(d18.Reg, scm.RegR11)
			}
			d142 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r31, Condition: scm.CondEqual}
			ctx.BindReg(r31, &d142)
		} else if d18.Loc == scm.LocImm {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d141.Reg)
			d142 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r32, Condition: scm.CondEqual}
			ctx.BindReg(r32, &d142)
		} else {
			r33 := ctx.AllocRegExcept(d18.Reg)
			ctx.EmitCmpInt64(d18.Reg, d141.Reg)
			d142 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r33, Condition: scm.CondEqual}
			ctx.BindReg(r33, &d142)
		}
		ctx.FreeDesc(&d141)
		d143 = d142
		ctx.EnsureDesc(&d143)
		if d143.Loc != scm.LocImm && d143.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d143.Loc == scm.LocImm {
			if d143.Imm.Bool() {
				if ps.General {
				}
				ps144 := scm.PhiState{General: ps.General}
				ps144.OverlayValues = make([]scm.JITValueDesc, 144)
				ps144.OverlayValues[0] = d0
				ps144.OverlayValues[1] = d1
				ps144.OverlayValues[2] = d2
				ps144.OverlayValues[3] = d3
				ps144.OverlayValues[4] = d4
				ps144.OverlayValues[5] = d5
				ps144.OverlayValues[6] = d6
				ps144.OverlayValues[7] = d7
				ps144.OverlayValues[8] = d8
				ps144.OverlayValues[9] = d9
				ps144.OverlayValues[10] = d10
				ps144.OverlayValues[11] = d11
				ps144.OverlayValues[12] = d12
				ps144.OverlayValues[13] = d13
				ps144.OverlayValues[14] = d14
				ps144.OverlayValues[15] = d15
				ps144.OverlayValues[16] = d16
				ps144.OverlayValues[17] = d17
				ps144.OverlayValues[18] = d18
				ps144.OverlayValues[19] = d19
				ps144.OverlayValues[20] = d20
				ps144.OverlayValues[69] = d69
				ps144.OverlayValues[70] = d70
				ps144.OverlayValues[71] = d71
				ps144.OverlayValues[72] = d72
				ps144.OverlayValues[73] = d73
				ps144.OverlayValues[74] = d74
				ps144.OverlayValues[75] = d75
				ps144.OverlayValues[76] = d76
				ps144.OverlayValues[141] = d141
				ps144.OverlayValues[142] = d142
				ps144.OverlayValues[143] = d143
				return bbs[1].RenderPS(ps144)
			}
			if ps.General {
			}
			ps145 := scm.PhiState{General: ps.General}
			ps145.OverlayValues = make([]scm.JITValueDesc, 144)
			ps145.OverlayValues[0] = d0
			ps145.OverlayValues[1] = d1
			ps145.OverlayValues[2] = d2
			ps145.OverlayValues[3] = d3
			ps145.OverlayValues[4] = d4
			ps145.OverlayValues[5] = d5
			ps145.OverlayValues[6] = d6
			ps145.OverlayValues[7] = d7
			ps145.OverlayValues[8] = d8
			ps145.OverlayValues[9] = d9
			ps145.OverlayValues[10] = d10
			ps145.OverlayValues[11] = d11
			ps145.OverlayValues[12] = d12
			ps145.OverlayValues[13] = d13
			ps145.OverlayValues[14] = d14
			ps145.OverlayValues[15] = d15
			ps145.OverlayValues[16] = d16
			ps145.OverlayValues[17] = d17
			ps145.OverlayValues[18] = d18
			ps145.OverlayValues[19] = d19
			ps145.OverlayValues[20] = d20
			ps145.OverlayValues[69] = d69
			ps145.OverlayValues[70] = d70
			ps145.OverlayValues[71] = d71
			ps145.OverlayValues[72] = d72
			ps145.OverlayValues[73] = d73
			ps145.OverlayValues[74] = d74
			ps145.OverlayValues[75] = d75
			ps145.OverlayValues[76] = d76
			ps145.OverlayValues[141] = d141
			ps145.OverlayValues[142] = d142
			ps145.OverlayValues[143] = d143
			return bbs[2].RenderPS(ps145)
		}
		if !ps.General {
			ps.General = true
			return bbs[3].RenderPS(ps)
		}
		ctx.EmitJump(d143.Condition, lbl2)
		ctx.FreeDesc(&d142)
		snap146 := d0
		snap147 := d1
		snap148 := d2
		snap149 := d3
		snap150 := d4
		snap151 := d5
		snap152 := d6
		snap153 := d7
		snap154 := d8
		snap155 := d9
		snap156 := d10
		snap157 := d11
		snap158 := d12
		snap159 := d13
		snap160 := d14
		snap161 := d15
		snap162 := d16
		snap163 := d17
		snap164 := d18
		snap165 := d19
		snap166 := d20
		snap167 := d69
		snap168 := d70
		snap169 := d71
		snap170 := d72
		snap171 := d73
		snap172 := d74
		snap173 := d75
		snap174 := d76
		snap175 := d141
		snap176 := d142
		snap177 := d143
		alloc178 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc178)
		d0 = snap146
		d1 = snap147
		d2 = snap148
		d3 = snap149
		d4 = snap150
		d5 = snap151
		d6 = snap152
		d7 = snap153
		d8 = snap154
		d9 = snap155
		d10 = snap156
		d11 = snap157
		d12 = snap158
		d13 = snap159
		d14 = snap160
		d15 = snap161
		d16 = snap162
		d17 = snap163
		d18 = snap164
		d19 = snap165
		d20 = snap166
		d69 = snap167
		d70 = snap168
		d71 = snap169
		d72 = snap170
		d73 = snap171
		d74 = snap172
		d75 = snap173
		d76 = snap174
		d141 = snap175
		d142 = snap176
		d143 = snap177
		ctx.RestoreAllocState(alloc178)
		d0 = snap146
		d1 = snap147
		d2 = snap148
		d3 = snap149
		d4 = snap150
		d5 = snap151
		d6 = snap152
		d7 = snap153
		d8 = snap154
		d9 = snap155
		d10 = snap156
		d11 = snap157
		d12 = snap158
		d13 = snap159
		d14 = snap160
		d15 = snap161
		d16 = snap162
		d17 = snap163
		d18 = snap164
		d19 = snap165
		d20 = snap166
		d69 = snap167
		d70 = snap168
		d71 = snap169
		d72 = snap170
		d73 = snap171
		d74 = snap172
		d75 = snap173
		d76 = snap174
		d141 = snap175
		d142 = snap176
		d143 = snap177
		ps179 := scm.PhiState{General: true}
		ps179.OverlayValues = make([]scm.JITValueDesc, 144)
		ps179.OverlayValues[0] = d0
		ps179.OverlayValues[1] = d1
		ps179.OverlayValues[2] = d2
		ps179.OverlayValues[3] = d3
		ps179.OverlayValues[4] = d4
		ps179.OverlayValues[5] = d5
		ps179.OverlayValues[6] = d6
		ps179.OverlayValues[7] = d7
		ps179.OverlayValues[8] = d8
		ps179.OverlayValues[9] = d9
		ps179.OverlayValues[10] = d10
		ps179.OverlayValues[11] = d11
		ps179.OverlayValues[12] = d12
		ps179.OverlayValues[13] = d13
		ps179.OverlayValues[14] = d14
		ps179.OverlayValues[15] = d15
		ps179.OverlayValues[16] = d16
		ps179.OverlayValues[17] = d17
		ps179.OverlayValues[18] = d18
		ps179.OverlayValues[19] = d19
		ps179.OverlayValues[20] = d20
		ps179.OverlayValues[69] = d69
		ps179.OverlayValues[70] = d70
		ps179.OverlayValues[71] = d71
		ps179.OverlayValues[72] = d72
		ps179.OverlayValues[73] = d73
		ps179.OverlayValues[74] = d74
		ps179.OverlayValues[75] = d75
		ps179.OverlayValues[76] = d76
		ps179.OverlayValues[141] = d141
		ps179.OverlayValues[142] = d142
		ps179.OverlayValues[143] = d143
		ps180 := scm.PhiState{General: true}
		ps180.OverlayValues = make([]scm.JITValueDesc, 144)
		ps180.OverlayValues[0] = d0
		ps180.OverlayValues[1] = d1
		ps180.OverlayValues[2] = d2
		ps180.OverlayValues[3] = d3
		ps180.OverlayValues[4] = d4
		ps180.OverlayValues[5] = d5
		ps180.OverlayValues[6] = d6
		ps180.OverlayValues[7] = d7
		ps180.OverlayValues[8] = d8
		ps180.OverlayValues[9] = d9
		ps180.OverlayValues[10] = d10
		ps180.OverlayValues[11] = d11
		ps180.OverlayValues[12] = d12
		ps180.OverlayValues[13] = d13
		ps180.OverlayValues[14] = d14
		ps180.OverlayValues[15] = d15
		ps180.OverlayValues[16] = d16
		ps180.OverlayValues[17] = d17
		ps180.OverlayValues[18] = d18
		ps180.OverlayValues[19] = d19
		ps180.OverlayValues[20] = d20
		ps180.OverlayValues[69] = d69
		ps180.OverlayValues[70] = d70
		ps180.OverlayValues[71] = d71
		ps180.OverlayValues[72] = d72
		ps180.OverlayValues[73] = d73
		ps180.OverlayValues[74] = d74
		ps180.OverlayValues[75] = d75
		ps180.OverlayValues[76] = d76
		ps180.OverlayValues[141] = d141
		ps180.OverlayValues[142] = d142
		ps180.OverlayValues[143] = d143
		snap181 := d0
		snap182 := d1
		snap183 := d2
		snap184 := d3
		snap185 := d4
		snap186 := d5
		snap187 := d6
		snap188 := d7
		snap189 := d8
		snap190 := d9
		snap191 := d10
		snap192 := d11
		snap193 := d12
		snap194 := d13
		snap195 := d14
		snap196 := d15
		snap197 := d16
		snap198 := d17
		snap199 := d18
		snap200 := d19
		snap201 := d20
		snap202 := d69
		snap203 := d70
		snap204 := d71
		snap205 := d72
		snap206 := d73
		snap207 := d74
		snap208 := d75
		snap209 := d76
		snap210 := d141
		snap211 := d142
		snap212 := d143
		alloc213 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps180)
		}
		ctx.RestoreAllocState(alloc213)
		d0 = snap181
		d1 = snap182
		d2 = snap183
		d3 = snap184
		d4 = snap185
		d5 = snap186
		d6 = snap187
		d7 = snap188
		d8 = snap189
		d9 = snap190
		d10 = snap191
		d11 = snap192
		d12 = snap193
		d13 = snap194
		d14 = snap195
		d15 = snap196
		d16 = snap197
		d17 = snap198
		d18 = snap199
		d19 = snap200
		d20 = snap201
		d69 = snap202
		d70 = snap203
		d71 = snap204
		d72 = snap205
		d73 = snap206
		d74 = snap207
		d75 = snap208
		d76 = snap209
		d141 = snap210
		d142 = snap211
		d143 = snap212
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps179)
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
		if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
			d69 = ps.OverlayValues[69]
		}
		if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
			d70 = ps.OverlayValues[70]
		}
		if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
			d71 = ps.OverlayValues[71]
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
		var d214 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r34 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r34, thisptr.Reg, off)
			d214 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d214)
		}
		ctx.EnsureDesc(&d214)
		r35 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r35, uint64(uintptr(unsafe.Pointer(&pow10i[0]))))
		r36 := ctx.AllocReg()
		if d214.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r36, uint64(d214.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r36, d214.Reg)
			ctx.EmitShlRegImm8(r36, 3)
		}
		ctx.EmitAddInt64(r35, r36)
		ctx.FreeReg(r36)
		r37 := ctx.AllocRegExcept(r35)
		ctx.EmitMovRegMem(r37, r35, 0)
		ctx.FreeReg(r35)
		d215 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
		ctx.BindReg(r37, &d215)
		ctx.FreeDesc(&d214)
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d215)
		ctx.EnsureDescsTogether(&d73, &d215)
		var d216 scm.JITValueDesc
		if d73.Loc == scm.LocImm && d215.Loc == scm.LocImm {
			d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d73.Imm.Int() * d215.Imm.Int())}
		} else if d73.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d215.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d73.Imm.Int()))
			ctx.EmitImulInt64(scratch, d215.Reg)
			d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d216)
		} else if d215.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegReg(scratch, d73.Reg)
			if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d215.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d216)
		} else {
			r38 := ctx.AllocRegExcept(d73.Reg, d215.Reg)
			ctx.EmitMovRegReg(r38, d73.Reg)
			ctx.EmitImulInt64(r38, d215.Reg)
			d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			ctx.BindReg(r38, &d216)
		}
		if d216.Loc == scm.LocReg && d73.Loc == scm.LocReg && d216.Reg == d73.Reg {
			ctx.TransferReg(d73.Reg)
			d73.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d215)
		ctx.EnsureDesc(&d216)
		d217 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d217)
		ctx.BindReg(r1, &d217)
		ctx.EnsureDesc(&d216)
		ctx.EmitMakeInt(d217, d216)
		if d216.Loc == scm.LocReg {
			ctx.FreeReg(d216.Reg)
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
		if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
			d69 = ps.OverlayValues[69]
		}
		if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
			d70 = ps.OverlayValues[70]
		}
		if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
			d71 = ps.OverlayValues[71]
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
		if len(ps.OverlayValues) > 141 && ps.OverlayValues[141].Loc != scm.LocNone {
			d141 = ps.OverlayValues[141]
		}
		if len(ps.OverlayValues) > 142 && ps.OverlayValues[142].Loc != scm.LocNone {
			d142 = ps.OverlayValues[142]
		}
		if len(ps.OverlayValues) > 143 && ps.OverlayValues[143].Loc != scm.LocNone {
			d143 = ps.OverlayValues[143]
		}
		if len(ps.OverlayValues) > 214 && ps.OverlayValues[214].Loc != scm.LocNone {
			d214 = ps.OverlayValues[214]
		}
		if len(ps.OverlayValues) > 215 && ps.OverlayValues[215].Loc != scm.LocNone {
			d215 = ps.OverlayValues[215]
		}
		if len(ps.OverlayValues) > 216 && ps.OverlayValues[216].Loc != scm.LocNone {
			d216 = ps.OverlayValues[216]
		}
		if len(ps.OverlayValues) > 217 && ps.OverlayValues[217].Loc != scm.LocNone {
			d217 = ps.OverlayValues[217]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d73)
		ctx.EnsureDesc(&d73)
		var d218 scm.JITValueDesc
		if d73.Loc == scm.LocImm {
			d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d73.Imm.Int()))}
		} else {
			r39 := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegReg(r39, d73.Reg)
			ctx.EmitCvtInt64ToFloat64(scm.RegX0, r39)
			d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r39}
			ctx.BindReg(r39, &d218)
		}
		var d219 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageDecimal)(nil).scaleExp)
			val := *(*int8)(unsafe.Pointer(fieldAddr))
			d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StorageDecimal)(nil).scaleExp))
			r40 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r40, thisptr.Reg, off)
			d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d219)
		}
		ctx.EnsureDesc(&d219)
		ctx.EnsureDesc(&d219)
		var d220 scm.JITValueDesc
		if d219.Loc == scm.LocImm {
			d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(int8(d219.Imm.Int()))))}
		} else {
			r41 := ctx.AllocReg()
			ctx.EmitMovRegReg(r41, d219.Reg)
			ctx.EmitShlRegImm8(r41, 56)
			ctx.EmitSarRegImm8(r41, 56)
			d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
			ctx.BindReg(r41, &d220)
		}
		ctx.FreeDesc(&d219)
		ctx.EnsureDesc(&d220)
		ctx.EnsureDesc(&d220)
		var d221 scm.JITValueDesc
		if d220.Loc == scm.LocImm {
			d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d220.Imm.Int() + 15)}
		} else {
			scratch := ctx.AllocRegExcept(d220.Reg)
			ctx.EmitMovRegReg(scratch, d220.Reg)
			ctx.EmitAddRegImm32(scratch, int32(15))
			d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d221)
		}
		if d221.Loc == scm.LocReg && d220.Loc == scm.LocReg && d221.Reg == d220.Reg {
			ctx.TransferReg(d220.Reg)
			d220.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d220)
		ctx.EnsureDesc(&d221)
		r42 := ctx.AllocReg()
		ctx.EmitMovRegImm64(r42, uint64(uintptr(unsafe.Pointer(&pow10f[0]))))
		r43 := ctx.AllocReg()
		if d221.Loc == scm.LocImm {
			ctx.EmitMovRegImm64(r43, uint64(d221.Imm.Int())*8)
		} else {
			ctx.EmitMovRegReg(r43, d221.Reg)
			ctx.EmitShlRegImm8(r43, 3)
		}
		ctx.EmitAddInt64(r42, r43)
		ctx.FreeReg(r43)
		r44 := ctx.AllocRegExcept(r42)
		ctx.EmitMovRegMem(r44, r42, 0)
		ctx.FreeReg(r42)
		d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
		ctx.BindReg(r44, &d222)
		ctx.FreeDesc(&d221)
		ctx.EnsureDesc(&d218)
		ctx.EnsureDesc(&d222)
		ctx.EnsureDescsTogether(&d218, &d222)
		var d223 scm.JITValueDesc
		if d218.Loc == scm.LocImm && d222.Loc == scm.LocImm {
			d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(d218.Imm.Float() * d222.Imm.Float())}
		} else if d218.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d222.Reg)
			_, xBits := d218.Imm.RawWords()
			ctx.EmitMovRegImm64(scratch, xBits)
			ctx.EmitMulFloat64(scratch, d222.Reg)
			d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d223)
		} else if d222.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d218.Reg)
			ctx.EmitMovRegReg(scratch, d218.Reg)
			_, yBits := d222.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitMulFloat64(scratch, scm.RegR11)
			d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: scratch}
			ctx.BindReg(scratch, &d223)
		} else {
			r45 := ctx.AllocRegExcept(d218.Reg, d222.Reg)
			ctx.EmitMovRegReg(r45, d218.Reg)
			ctx.EmitMulFloat64(r45, d222.Reg)
			d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: r45}
			ctx.BindReg(r45, &d223)
		}
		if d223.Loc == scm.LocReg && d218.Loc == scm.LocReg && d223.Reg == d218.Reg {
			ctx.TransferReg(d218.Reg)
			d218.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d218)
		ctx.FreeDesc(&d222)
		ctx.EnsureDesc(&d223)
		d224 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d224)
		ctx.BindReg(r1, &d224)
		ctx.EnsureDesc(&d223)
		ctx.EmitMakeFloat(d224, d223)
		if d223.Loc == scm.LocReg {
			ctx.FreeReg(d223.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps225 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps225)
	ctx.MarkLabel(lbl0)
	d226 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d226)
	ctx.BindReg(r1, &d226)
	ctx.EmitMovPairToResult(&d226, &result)
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
