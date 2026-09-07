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
	storageJITFunctions
	values []float64 `jit:"immutable-after-finish"`
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

func (s *StorageFloat) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	var d0 scm.JITValueDesc
	_ = d0
	var d1 scm.JITValueDesc
	_ = d1
	var d2 scm.JITValueDesc
	_ = d2
	var d4 scm.JITValueDesc
	_ = d4
	var d19 scm.JITValueDesc
	_ = d19
	var d20 scm.JITValueDesc
	_ = d20
	var d21 scm.JITValueDesc
	_ = d21
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
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d0 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			r2 := ctx.AllocRegExcept(r0, r1)
			off := int32(unsafe.Offsetof((*StorageFloat)(nil).values))
			ctx.EmitMovRegMem(r0, thisptr.Reg, off)
			ctx.EmitMovRegMem(r1, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r2, thisptr.Reg, off+16)
			d0 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r0, Reg2: r1, Reg3: r2}
			ctx.BindReg(r0, &d0)
			ctx.BindReg(r1, &d0)
			ctx.BindReg(r2, &d0)
			ctx.BindReg(r0, &d0)
			ctx.BindReg(r1, &d0)
			ctx.BindReg(r2, &d0)
		}
		ctx.EnsureDesc(&idxInt)
		d1 = ctx.EmitLoadScalarSliceElement(&d0, &idxInt, 8, scm.TagFloat)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d1)
		var d2 scm.JITValueDesc
		if d1.Loc == scm.LocImm {
			d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d1.Imm.Float() != d1.Imm.Float())}
		} else {
			ctx.EnsureDesc(&d1)
			nanSource3 := d1.Reg
			if d1.Loc == scm.LocRegPair {
				nanSource3 = d1.Reg2
			}
			r3 := ctx.AllocRegExcept(nanSource3)
			ctx.EmitCmpFloat64(nanSource3, nanSource3)
			d2 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r3, Condition: scm.CondParity}
			ctx.BindReg(r3, &d2)
		}
		d4 = d2
		ctx.EnsureDesc(&d4)
		if d4.Loc != scm.LocImm && d4.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d4.Loc == scm.LocImm {
			if d4.Imm.Bool() {
				if ps.General {
				}
				ps5 := scm.PhiState{General: ps.General}
				ps5.OverlayValues = make([]scm.JITValueDesc, 5)
				ps5.OverlayValues[0] = d0
				ps5.OverlayValues[1] = d1
				ps5.OverlayValues[2] = d2
				ps5.OverlayValues[4] = d4
				return bbs[1].RenderPS(ps5)
			}
			if ps.General {
			}
			ps6 := scm.PhiState{General: ps.General}
			ps6.OverlayValues = make([]scm.JITValueDesc, 5)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[4] = d4
			return bbs[2].RenderPS(ps6)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitJump(d4.Condition, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FreeDesc(&d2)
		snap7 := d0
		snap8 := d1
		snap9 := d2
		snap10 := d4
		alloc11 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc11)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		d4 = snap10
		ctx.RestoreAllocState(alloc11)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		d4 = snap10
		ps12 := scm.PhiState{General: true}
		ps12.OverlayValues = make([]scm.JITValueDesc, 5)
		ps12.OverlayValues[0] = d0
		ps12.OverlayValues[1] = d1
		ps12.OverlayValues[2] = d2
		ps12.OverlayValues[4] = d4
		ps13 := scm.PhiState{General: true}
		ps13.OverlayValues = make([]scm.JITValueDesc, 5)
		ps13.OverlayValues[0] = d0
		ps13.OverlayValues[1] = d1
		ps13.OverlayValues[2] = d2
		ps13.OverlayValues[4] = d4
		snap14 := d0
		snap15 := d1
		snap16 := d2
		snap17 := d4
		alloc18 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps13)
		}
		ctx.RestoreAllocState(alloc18)
		d0 = snap14
		d1 = snap15
		d2 = snap16
		d4 = snap17
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps12)
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
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		ctx.ReclaimUntrackedRegs()
		d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d20 = result
		ctx.EnsureDesc(&d19)
		if d19.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d19, &d20)
		} else {
			switch d19.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d20, d19)
			case scm.TagInt:
				ctx.EmitMakeInt(d20, d19)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d20, d19)
			case scm.TagNil:
				ctx.EmitMakeNil(d20)
			default:
				ctx.EmitMovPairToResult(&d19, &d20)
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
		if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
			d4 = ps.OverlayValues[4]
		}
		if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
			d19 = ps.OverlayValues[19]
		}
		if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
			d20 = ps.OverlayValues[20]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d1)
		d21 = result
		ctx.EnsureDesc(&d1)
		ctx.EmitMakeFloat(d21, d1)
		if d1.Loc == scm.LocReg {
			ctx.FreeReg(d1.Reg)
		}
		ctx.EmitJmp(lbl0)
		return result
	}
	ps22 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps22)
	ctx.MarkLabel(lbl0)
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

func (s *StorageFloat) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

func (s *StorageFloat) GetValue(i uint32) scm.Scmer {
	// NULL is encoded as NaN in SQL
	v := s.values[i]
	if math.IsNaN(v) {
		return scm.NewNil()
	}
	return scm.NewFloat(v)
}

//jitgen:control-flow-stable recid count target/1 stride
func (s *StorageFloat) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		v := s.values[recid+k]
		if math.IsNaN(v) {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewFloat(v)
		}
		idx += stride
	}
}

//jitgen:control-flow-stable recids/2 target/1 stride
func (s *StorageFloat) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for k := 0; k < len(recids); k++ {
		v := s.values[recids[k]]
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
	s.storageJITFunctions.finish(s)
}

func (s *StorageFloat) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageFloat) DistinctCount() uint { return uint(len(s.values)) }
