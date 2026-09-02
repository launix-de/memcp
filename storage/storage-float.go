/*
Copyright (C) 2023-2026  Carl-Philip Hänsch

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
import "unsafe"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// main type for storage: can store any value, is inefficient but does type analysis how to optimize
type StorageFloat struct {
	values []float64
}

func (s *StorageFloat) ComputeSize() uint {
	return 16 + 8*uint(len(s.values)) + 24 /* a slice */
}

func (s *StorageFloat) String() string {
	return "float64"
}

// storageFloatVersion is the current binary format version for StorageFloat.
// Increment this constant and add a new deserializeFloatV* helper whenever the
// layout after the magic byte changes.  Never delete old helpers.
const storageFloatVersion = 0

// StorageFloat binary layout (magic byte 12 consumed by shard loader):
//
//	[version uint8]      ← first byte read by Deserialize
//	[pad 6 bytes]        ← alignment padding to reach 8-byte boundary before count
//	[count uint64]
//	[values: count × 8 bytes float64, NaN = NULL]
//
// Version history:
//
//	0 (current): layout as above; the version byte was previously the first byte
//	             of a 7-byte ASCII dummy "1234567" (byte value '1'=49).
//	             Legacy detection: if version byte == '1' (49), treat as v0 legacy.

func (s *StorageFloat) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.TrackPointer(unsafe.Pointer(s))
	thisptrPinned := thisptr.Loc == scm.LocReg
	thisptrPinnedReg := thisptr.Reg
	if thisptrPinned {
		ctx.ProtectReg(thisptrPinnedReg)
		defer ctx.UnprotectReg(thisptrPinnedReg)
	}
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
	var bbs [3]scm.BBDescriptor
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
		var d0 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageFloat)(nil).values)
			r2 := ctx.AllocReg()
			r3 := ctx.AllocRegExcept(r2)
			r4 := ctx.AllocRegExcept(r2, r3)
			ctx.EmitMovRegMem64(r2, fieldAddr)
			ctx.EmitMovRegMem64(r3, fieldAddr+8)
			ctx.EmitMovRegMem64(r4, fieldAddr+16)
			d0 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r2, Reg2: r3, Reg3: r4}
			ctx.BindReg(r2, &d0)
			ctx.BindReg(r3, &d0)
			ctx.BindReg(r4, &d0)
		} else {
			off := int32(unsafe.Offsetof((*StorageFloat)(nil).values))
			r5 := ctx.AllocReg()
			r6 := ctx.AllocRegExcept(r5)
			r7 := ctx.AllocRegExcept(r5, r6)
			ctx.EmitMovRegMem(r5, thisptr.Reg, off)
			ctx.EmitMovRegMem(r6, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+16)
			d0 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r5, Reg2: r6, Reg3: r7}
			ctx.BindReg(r5, &d0)
			ctx.BindReg(r6, &d0)
			ctx.BindReg(r7, &d0)
		}
		ctx.EnsureDesc(&idxInt)
		d2 = ctx.EmitSliceElementAddress(&d0, &idxInt, 8)
		ctx.EnsureDesc(&d2)
		ctx.EmitMovRegMem(d2.Reg, d2.Reg, 0)
		d1 = d2
		ctx.EnsureDesc(&d1)
		d3 = d1
		_ = d3
		ctx.StabilizeDescForControlFlow(&d3)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl4 := ctx.ReserveLabel()
		_ = lbl4
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl4)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d3)
		ctx.EnsureDesc(&d3)
		ctx.EnsureDescsTogether(&d3, &d3)
		var d4 scm.JITValueDesc
		if d3.Loc == scm.LocImm {
			d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d3.Imm.Float() != d3.Imm.Float())}
		} else if d3.Loc == scm.LocImm {
			r8 := ctx.AllocRegExcept(d3.Reg)
			_, yBits := d3.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, yBits)
			ctx.EmitCmpFloat64Setcc(r8, d3.Reg, scm.RegR11, scm.CondNotEqual)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
			ctx.BindReg(r8, &d4)
		} else if d3.Loc == scm.LocImm {
			r9 := ctx.AllocRegExcept(d3.Reg)
			_, xBits := d3.Imm.RawWords()
			ctx.EmitMovRegImm64(scm.RegR11, xBits)
			ctx.EmitCmpFloat64Setcc(r9, scm.RegR11, d3.Reg, scm.CondNotEqual)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r9}
			ctx.BindReg(r9, &d4)
		} else {
			r10 := ctx.AllocRegExcept(d3.Reg, d3.Reg)
			ctx.EmitCmpFloat64Setcc(r10, d3.Reg, d3.Reg, scm.CondNotEqual)
			d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
			ctx.BindReg(r10, &d4)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d4)
		ctx.FreeDesc(&d1)
		d5 = d4
		ctx.EnsureDesc(&d5)
		if d5.Loc != scm.LocImm && d5.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d5.Loc == scm.LocImm {
			if d5.Imm.Bool() {
				if ps.General {
				}
				ps6 := scm.PhiState{General: ps.General}
				ps6.OverlayValues = make([]scm.JITValueDesc, 6)
				ps6.OverlayValues[0] = d0
				ps6.OverlayValues[1] = d1
				ps6.OverlayValues[2] = d2
				ps6.OverlayValues[3] = d3
				ps6.OverlayValues[4] = d4
				ps6.OverlayValues[5] = d5
				return bbs[1].RenderPS(ps6)
			}
			if ps.General {
			}
			ps7 := scm.PhiState{General: ps.General}
			ps7.OverlayValues = make([]scm.JITValueDesc, 6)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			ps7.OverlayValues[4] = d4
			ps7.OverlayValues[5] = d5
			return bbs[2].RenderPS(ps7)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		lbl5 := ctx.ReserveLabel()
		lbl6 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d5.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl5)
		ctx.EmitJmp(lbl6)
		ctx.MarkLabel(lbl5)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl6)
		ctx.EmitJmp(lbl3)
		ps8 := scm.PhiState{General: true}
		ps8.OverlayValues = make([]scm.JITValueDesc, 6)
		ps8.OverlayValues[0] = d0
		ps8.OverlayValues[1] = d1
		ps8.OverlayValues[2] = d2
		ps8.OverlayValues[3] = d3
		ps8.OverlayValues[4] = d4
		ps8.OverlayValues[5] = d5
		ps9 := scm.PhiState{General: true}
		ps9.OverlayValues = make([]scm.JITValueDesc, 6)
		ps9.OverlayValues[0] = d0
		ps9.OverlayValues[1] = d1
		ps9.OverlayValues[2] = d2
		ps9.OverlayValues[3] = d3
		ps9.OverlayValues[4] = d4
		ps9.OverlayValues[5] = d5
		snap10 := d0
		snap11 := d1
		snap12 := d2
		snap13 := d3
		snap14 := d4
		snap15 := d5
		alloc16 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps9)
		}
		ctx.RestoreAllocState(alloc16)
		d0 = snap10
		d1 = snap11
		d2 = snap12
		d3 = snap13
		d4 = snap14
		d5 = snap15
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps8)
		}
		return result
		ctx.FreeDesc(&d4)
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
		ctx.ReclaimUntrackedRegs()
		d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d18 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d18)
		ctx.BindReg(r1, &d18)
		ctx.EnsureDesc(&d17)
		if d17.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d17, &d18)
		} else {
			switch d17.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d18, d17)
			case scm.TagInt:
				ctx.EmitMakeInt(d18, d17)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d18, d17)
			case scm.TagNil:
				ctx.EmitMakeNil(d18)
			default:
				ctx.EmitMovPairToResult(&d17, &d18)
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
		if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
			d17 = ps.OverlayValues[17]
		}
		if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
			d18 = ps.OverlayValues[18]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		d20 = ctx.EmitSliceElementAddress(&d0, &idxInt, 8)
		ctx.EnsureDesc(&d20)
		ctx.EmitMovRegMem(d20.Reg, d20.Reg, 0)
		d19 = d20
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d19)
		d21 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d21)
		ctx.BindReg(r1, &d21)
		ctx.EnsureDesc(&d19)
		ctx.EmitMakeFloat(d21, d19)
		if d19.Loc == scm.LocReg {
			ctx.FreeReg(d19.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps22 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps22)
	ctx.MarkLabel(lbl0)
	d23 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d23)
	ctx.BindReg(r1, &d23)
	ctx.EmitMovPairToResult(&d23, &result)
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

func (s *StorageFloat) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(12))                  // 12 = StorageFloat
	binary.Write(f, binary.LittleEndian, uint8(storageFloatVersion)) // version byte (was '1' in legacy)
	var pad [6]byte
	f.Write(pad[:]) // remaining alignment padding (was "234567")
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	// now at offset 16 begin data
	rawdata := unsafe.Slice((*byte)(unsafe.Pointer(&s.values[0])), 8*len(s.values))
	f.Write(rawdata)
	// free allocated memory and mmap
	/* TODO: runtime.SetFinalizer(s, func(s *StorageSCMER) {f.Close()})
	newrawdata = mmap.Map(f, RDWR, 0)
	s.values = unsafe.Slice((*float64)&newrawdata[16], len(s.values))
	*/
}
func (s *StorageFloat) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	var pad [6]byte
	f.Read(pad[:])
	switch version {
	case 0, '1': // '1'=49: legacy pre-versioning dummy byte; treat as v0
		return s.deserializeFloatV0(f)
	default:
		panic(fmt.Sprintf("StorageFloat: unknown version %d", version))
	}
}

func (s *StorageFloat) deserializeFloatV0(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	/* TODO: runtime.SetFinalizer(s, func(s *StorageSCMER) { f.Close() })
	rawdata := mmap.Map(f, RDWR, 0)
	*/
	rawdata := make([]byte, 8*l)
	f.Read(rawdata)
	s.values = unsafe.Slice((*float64)(unsafe.Pointer(&rawdata[0])), l)
	return uint(l)
}

func (s *StorageFloat) GetCachedReader() ColumnReader { return s }

func (s *StorageFloat) GetValue(i uint32) scm.Scmer {
	// NULL is encoded as NaN in SQL
	if math.IsNaN(s.values[i]) {
		return scm.NewNil()
	}
	return scm.NewFloat(s.values[i])
}

func (s *StorageFloat) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	values := s.values[recid : uint32(recid)+count]
	idx := 0
	for _, v := range values {
		if math.IsNaN(v) {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewFloat(v)
		}
		idx += stride
	}
}

func (s *StorageFloat) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	values := s.values
	idx := 0
	for _, recid := range recids {
		v := values[recid]
		if math.IsNaN(v) {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewFloat(v)
		}
		idx += stride
	}
}

func (s *StorageFloat) scan(i uint32, value scm.Scmer) {
}
func (s *StorageFloat) prepare() {
}
func (s *StorageFloat) init(i uint32) {
	// allocate
	s.values = make([]float64, i)
}
func (s *StorageFloat) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values[i] = math.NaN()
	} else {
		s.values[i] = value.Float()
	}
}
func (s *StorageFloat) finish() {
}

func (s *StorageFloat) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageFloat) DistinctCount() uint { return uint(len(s.values)) }
