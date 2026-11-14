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
import "encoding/binary"
import "sync/atomic"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StorageSeq struct {
	storageJITFunctions
	// data
	recordId,
	start,
	stride StorageInt `jit:"immutable-after-finish"`
	count    uint   `jit:"immutable-after-finish"` // number of values
	seqCount uint32 `jit:"immutable-after-finish"` // number of sequences

	// Private construction statistics; readers never update a shared pivot.
	lastValue      atomic.Int64
	lastStride     int64
	lastValueNil   bool
	lastValueFirst bool
}

func (s *StorageSeq) ComputeSize() uint {
	return uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.recordId)-unsafe.Sizeof(s.start)-unsafe.Sizeof(s.stride)) + s.recordId.ComputeSize() + s.start.ComputeSize() + s.stride.ComputeSize()
}

func (s *StorageSeq) String() string {
	return fmt.Sprintf("seq[%dx %s/%s]", s.seqCount, s.start.String(), s.stride.String())
}

// storageSeqVersion is the current binary format version for StorageSeq.
// Increment this constant and add a new deserializeSeqV* helper whenever the
// layout after the magic byte changes.  Never delete old helpers.
const storageSeqVersion = 0

// StorageSeq binary layout (magic byte 11 consumed by shard loader):
//
//	[version uint8]    ← first byte read by Deserialize
//	[pad 7 bytes]      ← alignment padding
//	[count uint64]
//	[seqCount uint64]
//	[recordId StorageInt] (with its own magic byte)
//	[start StorageInt]    (with its own magic byte)
//	[stride StorageInt]   (with its own magic byte)
//
// Version history:
//
//	0 (current): layout as above; the version byte was previously the first byte
//	             of a 7-byte ASCII dummy "1234567" (byte value '1'=49).
//	             Legacy detection: if version byte == '1' (49), treat as v0 legacy.

func (s *StorageSeq) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&idxInt)
		d0 = ctx.EmitGoCallScalar(jitGoABI_0a5b4104daa925ca, []scm.JITValueDesc{thisptr, idxInt}, 1)
		d0.NoHeapPointer = true
		ctx.BindReg(d0.Reg, &d0)
		if thisptr.Loc == scm.LocRegPair || thisptr.Loc == scm.LocStackPair || thisptr.Loc == scm.LocRegTriple || thisptr.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		if d0.Loc == scm.LocRegPair || d0.Loc == scm.LocStackPair || d0.Loc == scm.LocRegTriple || d0.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&thisptr)
		ctx.SyncDesc(&d0)
		callResults1 := scm.JITEmitGoCallResults(ctx, jitGoABI_45f3d4c79979ea6d, []scm.JITValueDesc{thisptr, d0})
		d2 = callResults1[0]
		_ = d2
		d3 = callResults1[1]
		_ = d3
		d4 = callResults1[2]
		_ = d4
		d5 = callResults1[3]
		_ = d5
		ctx.FreeDesc(&d0)
		d6 = d3
		ctx.EnsureDesc(&d6)
		if d6.Loc != scm.LocImm && d6.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d6.Loc == scm.LocImm {
			if d6.Imm.Bool() {
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitCmpRegImm32(d6.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap7 := d0
			snap8 := d2
			snap9 := d3
			snap10 := d4
			snap11 := d5
			snap12 := d6
			alloc13 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc13)
			d0 = snap7
			d2 = snap8
			d3 = snap9
			d4 = snap10
			d5 = snap11
			d6 = snap12
		}
		if !bbs[1].Rendered {
			return bbs[1].Render()
		}
		return result
		ctx.FreeDesc(&d3)
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
		d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d15 = result
		ctx.EnsureDesc(&d14)
		if d14.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d14, &d15)
		} else {
			switch d14.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d15, d14)
			case scm.TagInt:
				ctx.EmitMakeInt(d15, d14)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d15, d14)
			case scm.TagNil:
				ctx.EmitMakeNil(d15)
			default:
				ctx.EmitMovPairToResult(&d14, &d15)
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
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&d2)
		ctx.EnsureDescsTogether(&idxInt, &d2)
		if idxInt.Loc == scm.LocImm && d2.Loc == scm.LocImm {
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d2.Imm.Int())}
		} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
			ctx.EnsureDesc(&idxInt)
			r0 := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(r0, idxInt.Reg)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r0}
			ctx.BindReg(r0, &d17)
		} else if idxInt.Loc == scm.LocImm {
			ctx.EnsureDesc(&d2)
			scratch := ctx.AllocRegExcept(d2.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d2)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else if d2.Loc == scm.LocImm {
			ctx.EnsureDesc(&idxInt)
			scratch := ctx.AllocRegExcept(idxInt.Reg)
			ctx.EmitMovRegReg(scratch, idxInt.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d2.Imm.Int())
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d17)
		} else {
			ctx.EnsureDesc(&idxInt)
			ctx.SyncDesc(&d2)
			r1 := ctx.AllocRegExcept(idxInt.Reg, d2.Reg)
			ctx.EmitMovRegReg(r1, idxInt.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r1, &d2)
			d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
			ctx.BindReg(r1, &d17)
		}
		if d17.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d17.Reg == idxInt.Reg {
			ctx.TransferReg(idxInt.Reg)
			idxInt.Loc = scm.LocNone
		}
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d17)
		ctx.EnsureDesc(&d5)
		ctx.EnsureDescsTogether(&d17, &d5)
		if d17.Loc == scm.LocImm && d5.Loc == scm.LocImm {
			d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() * d5.Imm.Int())}
		} else if d17.Loc == scm.LocImm {
			ctx.EnsureDesc(&d5)
			scratch := ctx.AllocRegExcept(d5.Reg)
			ctx.EmitMovRegReg(scratch, d5.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d17.Imm.Int())
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d18)
		} else if d5.Loc == scm.LocImm {
			ctx.EnsureDesc(&d17)
			scratch := ctx.AllocRegExcept(d17.Reg)
			ctx.EmitMovRegReg(scratch, d17.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d5.Imm.Int())
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d18)
		} else {
			ctx.EnsureDesc(&d17)
			ctx.SyncDesc(&d5)
			r2 := ctx.AllocRegExcept(d17.Reg, d5.Reg)
			ctx.EmitMovRegReg(r2, d17.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r2, &d5)
			d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d18)
		}
		if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
			ctx.TransferReg(d17.Reg)
			d17.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d17)
		ctx.EnsureDesc(&d4)
		resultTarget19 := false
		_ = resultTarget19
		ctx.EnsureDesc(&d18)
		ctx.EnsureDescsTogether(&d4, &d18)
		if d4.Loc == scm.LocImm && d18.Loc == scm.LocImm {
			d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + d18.Imm.Int())}
		} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
			ctx.EnsureDesc(&d4)
			var r3 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d4.Reg {
				r3 = result.Reg2
				resultTarget19 = true
			} else {
				r3 = ctx.AllocRegExcept(d4.Reg)
			}
			ctx.EmitMovRegReg(r3, d4.Reg)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d20)
		} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
			ctx.EnsureDesc(&d18)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			ctx.BindReg(d18.Reg, &d20)
		} else if d4.Loc == scm.LocImm {
			ctx.EnsureDesc(&d18)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d18.Reg {
				scratch = result.Reg2
				resultTarget19 = true
			} else {
				scratch = ctx.AllocRegExcept(d18.Reg)
			}
			ctx.EmitMovRegReg(scratch, d18.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d4.Imm.Int())
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else if d18.Loc == scm.LocImm {
			ctx.EnsureDesc(&d4)
			var scratch scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d4.Reg {
				scratch = result.Reg2
				resultTarget19 = true
			} else {
				scratch = ctx.AllocRegExcept(d4.Reg)
			}
			ctx.EmitMovRegReg(scratch, d4.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d18.Imm.Int())
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d20)
		} else {
			ctx.EnsureDesc(&d4)
			ctx.SyncDesc(&d18)
			var r4 scm.Reg
			if result.Loc == scm.LocRegPair && result.Reg2 != d4.Reg && result.Reg2 != d18.Reg {
				r4 = result.Reg2
				resultTarget19 = true
			} else {
				r4 = ctx.AllocRegExcept(d4.Reg, d18.Reg)
			}
			ctx.EmitMovRegReg(r4, d4.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r4, &d18)
			d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d20)
		}
		if d20.Loc == scm.LocReg && d4.Loc == scm.LocReg && d20.Reg == d4.Reg {
			ctx.TransferReg(d4.Reg)
			d4.Loc = scm.LocNone
		}
		if resultTarget19 && d20.Loc == scm.LocReg {
			ctx.BindReg(result.Reg2, &result)
		}
		ctx.FreeDesc(&d18)
		ctx.EnsureDesc(&d20)
		d21 = result
		ctx.EnsureDesc(&d20)
		ctx.EmitMakeInt(d21, d20)
		if d20.Loc == scm.LocReg {
			ctx.FreeReg(d20.Reg)
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

func (s *StorageSeq) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(11))                // 11 = StorageSeq
	binary.Write(f, binary.LittleEndian, uint8(storageSeqVersion)) // version byte (was '1' in legacy)
	var pad [6]byte
	f.Write(pad[:]) // remaining alignment padding (was "234567")
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.seqCount))
	s.recordId.Serialize(f)
	s.start.Serialize(f)
	s.stride.Serialize(f)
}

func (s *StorageSeq) Deserialize(f io.Reader) uint {
	var version uint8
	binary.Read(f, binary.LittleEndian, &version)
	var pad [6]byte
	readStorageBytes(f, pad[:])
	switch version {
	case 0, '1': // '1'=49: legacy pre-versioning dummy byte; treat as v0
		return s.deserializeSeqV0(f)
	default:
		panic(fmt.Sprintf("StorageSeq: unknown version %d", version))
	}
}

func (s *StorageSeq) deserializeSeqV0(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.count = uint(l)
	var sc uint64
	binary.Read(f, binary.LittleEndian, &sc)
	s.seqCount = uint32(sc)
	s.recordId.DeserializeEx(f, true)
	s.start.DeserializeEx(f, true)
	s.stride.DeserializeEx(f, true)
	return uint(l)
}

func (s *StorageSeq) GetCachedReader(hints ColumnReadHints) ColumnReader {
	return s.storageJITFunctions.reader(s)
}

func (s *StorageSeq) GetValue(i uint32) scm.Scmer {
	segment := s.findSegment(i)
	recid, isNil, value, stride := s.segmentAt(segment)
	if isNil {
		return scm.NewNil()
	}
	return scm.NewInt(value + (int64(i)-recid)*stride)
}

// findSegment searches immutable segment boundaries with call-local state.
// Bulk readers seed their own cursor once and then walk across segments;
// scalar readers use the same search without synchronizing on other readers.
func (s *StorageSeq) findSegment(i uint32) uint32 {
	var min, max uint32 = 0, s.seqCount - 1
	for min < max {
		pivot := (min + max + 1) / 2
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if uint32(recid) <= i {
			min = pivot
		} else {
			max = pivot - 1
		}
	}
	return min
}

// segmentAt reads the (recordId, isNil, start, stride) tuple for segment
// seg. Called once per segment touched, not once per row.
func (s *StorageSeq) segmentAt(seg uint32) (recordId int64, isNil bool, start int64, stride int64) {
	recordId = int64(s.recordId.GetValueUInt(seg)) + s.recordId.offset
	startRaw := s.start.GetValueUInt(seg)
	if s.start.hasNull && startRaw == s.start.null {
		isNil = true
		return
	}
	start = int64(startRaw) + s.start.offset
	stride = int64(s.stride.GetValueUInt(seg)) + s.stride.offset
	return
}

func (s *StorageSeq) segmentEnd(seg uint32) int64 {
	if seg+1 < s.seqCount {
		return int64(s.recordId.GetValueUInt(seg+1)) + s.recordId.offset
	}
	return int64(s.count)
}

// GetValueRange reads count consecutive rows starting at recid. It seeds the
// segment cursor with one local binary search and then walks forward: each
// arithmetic-sequence segment is read as start+delta*stride incrementally
// (a running add, no per-row multiply or search), and a nil segment fills
// its whole span directly.
//
//jitgen:control-flow-stable recid count target/1 stride
func (s *StorageSeq) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	if count == 0 {
		return
	}
	seg := s.findSegment(recid)
	segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
	nextRecordId := s.segmentEnd(seg)
	curVal := segStart + (int64(recid)-segRecordId)*segStride

	idx := 0
	for k := uint32(0); k < count; k++ {
		i := int64(recid) + int64(k)
		if i >= nextRecordId {
			seg++
			segRecordId, isNil, segStart, segStride = s.segmentAt(seg)
			nextRecordId = s.segmentEnd(seg)
			curVal = segStart + (i-segRecordId)*segStride
		}
		if isNil {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(curVal)
			curVal += segStride
		}
		idx += stride
	}
}

// GetValueMulti gathers arbitrary recids. When the batch is ascending (the
// common case for an index-probe or range-scan batch), it walks the segment
// cursor forward exactly like GetValueRange, recomputing the value with one
// multiply per row (deltas between requested recids aren't necessarily 1)
// but still only touching each crossed segment's recordId/start/stride once.
// A genuinely unordered batch falls back to a fresh local findSegment per
// row — still O(log seqCount) per row like GetValue, but without the shared
// atomic pivot-cache contention.
//
//jitgen:control-flow-stable recids/2 target/1 stride
func (s *StorageSeq) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	n := len(recids)
	if n == 0 {
		return
	}
	ascending := true
	for k := 1; k < n; k++ {
		if recids[k] < recids[k-1] {
			ascending = false
			break
		}
	}

	idx := 0
	if ascending {
		seg := s.findSegment(recids[0])
		segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
		nextRecordId := s.segmentEnd(seg)
		// Keep one explicit induction variable across the nested segment walk;
		// the range form introduces a second copied-element phi with no benefit.
		for k := 0; k < n; k++ {
			recid := recids[k]
			i := int64(recid)
			for i >= nextRecordId {
				seg++
				segRecordId, isNil, segStart, segStride = s.segmentAt(seg)
				nextRecordId = s.segmentEnd(seg)
			}
			if isNil {
				target[idx] = scm.NewNil()
			} else {
				target[idx] = scm.NewInt(segStart + (i-segRecordId)*segStride)
			}
			idx += stride
		}
		return
	}

	for k := 0; k < n; k++ {
		recid := recids[k]
		seg := s.findSegment(recid)
		segRecordId, isNil, segStart, segStride := s.segmentAt(seg)
		if isNil {
			target[idx] = scm.NewNil()
		} else {
			target[idx] = scm.NewInt(segStart + (int64(recid)-segRecordId)*segStride)
		}
		idx += stride
	}
}

func (s *StorageSeq) prepare() {
	// set up scan
	s.recordId.prepare()
	s.start.prepare()
	s.stride.prepare()
}
func (s *StorageSeq) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, scm.NewNil())
			s.stride.scan(s.seqCount-1, scm.NewInt(0))
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, scm.NewNil())
			s.stride.scan(s.seqCount-1, scm.NewInt(0))
		}
	} else {
		// integer
		v := value.Int()
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.scan(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
			// sequence stays the same
			s.lastValue.Store(v)
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue.Store(v)
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.scan(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.scan(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) init(i uint32) {
	s.recordId.init(s.seqCount)
	s.start.init(s.seqCount)
	s.stride.init(s.seqCount)
	s.lastValue.Store(0)
	s.lastStride = 0
	s.lastValueNil = false
	s.lastValueFirst = false
	s.count = uint(i)
	s.seqCount = 0
}
func (s *StorageSeq) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		// nil (stride is 0)
		if i == 0 {
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, scm.NewNil())
			s.stride.build(s.seqCount-1, scm.NewInt(0))
		} else if s.lastValueNil {
			// sequence stays the same
		} else {
			// start nil
			s.lastValueNil = true
			s.seqCount = s.seqCount + 1
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, scm.NewNil())
			s.stride.build(s.seqCount-1, scm.NewInt(0))
		}
	} else {
		// integer
		v := value.Int()
		// A value after NULL must start a fresh numeric segment.
		if !s.lastValueNil && s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.build(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if !s.lastValueNil && i != 0 && v == s.lastValue.Load()+s.lastStride {
			// sequence stays the same
			s.lastValue.Store(v)
		} else {
			// restart with new sequence
			s.seqCount = s.seqCount + 1
			s.lastValue.Store(v)
			s.lastValueFirst = true
			s.lastValueNil = false
			s.recordId.build(s.seqCount-1, scm.NewInt(int64(i)))
			s.start.build(s.seqCount-1, value)
		}
	}
}
func (s *StorageSeq) finish() {
	s.recordId.finish()
	s.start.finish()
	s.stride.finish()
	s.storageJITFunctions.finish(s)

	/* debug output of the sequence:
	for i := uint(0); i < s.seqCount; i++ {
		fmt.Println(s.recordId.GetValue(i),":",s.start.GetValue(i),":",s.stride.GetValue(i))
	}*/
}
func (s *StorageSeq) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}

func (s *StorageSeq) DistinctCount() uint { return uint(s.count) }

// BEGIN jitgen prepared Go ABI entries
var jitGoABI_0a5b4104daa925ca uint64
var jitGoABI_45f3d4c79979ea6d uint64

func init() {
	jitGoABI_0a5b4104daa925ca = scm.GoFuncAddr((*StorageSeq).findSegment)
	jitGoABI_45f3d4c79979ea6d = scm.GoFuncAddr((*StorageSeq).segmentAt)
}
