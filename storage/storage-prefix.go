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

import "fmt"
import "strings"
import "unsafe"
import "github.com/launix-de/memcp/scm"

type StoragePrefix struct {
	storageJITFunctions
	// prefix compression
	prefixes         StorageInt
	prefixdictionary []string      // pref
	values           StorageString // only one depth (but can be cascaded!)
}

func (s *StoragePrefix) ComputeSize() uint {
	return s.prefixes.ComputeSize() + 24 + s.values.ComputeSize()
}

func (s *StoragePrefix) String() string {
	return fmt.Sprintf("prefix[%s]-%s", s.prefixdictionary[1], s.values.String())
}

func (s *StoragePrefix) GetCachedReader() ColumnReader { return s.storageJITFunctions.reader(s) }

func (s *StoragePrefix) GetValue(i uint32) scm.Scmer {
	inner := s.values.GetValue(i)
	if inner.IsNil() {
		return scm.NewNil()
	}
	if !inner.IsString() {
		panic("invalid value in prefix storage")
	}
	idx := int64(s.prefixes.GetValueUInt(i)) + s.prefixes.offset
	if idx >= int64(len(s.prefixdictionary)) || idx < 0 {
		panic("prefix index out of range")
	}
	prefix := s.prefixdictionary[idx]
	return scm.NewString(prefix + inner.String())
}

// GetValueRange and GetValueMulti bulk-fetch the suffix strings and the raw
// prefix-dictionary indices via the two wrapped storages' own bulk methods
// (one call each instead of 2*n GetValue calls) and then stitch prefix+suffix
// together in a single post-process pass.
func (s *StoragePrefix) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.values.GetValueRange(recid, count, target, stride)
	idxbuf := make([]scm.Scmer, count)
	s.prefixes.GetValueRange(recid, count, idxbuf, 1)
	s.applyPrefixInPlace(target, idxbuf, count, stride)
}

func (s *StoragePrefix) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.values.GetValueMulti(recids, target, stride)
	idxbuf := make([]scm.Scmer, len(recids))
	s.prefixes.GetValueMulti(recids, idxbuf, 1)
	s.applyPrefixInPlace(target, idxbuf, uint32(len(recids)), stride)
}

// applyPrefixInPlace stitches each row's static dictionary prefix and its
// already-bulk-fetched suffix (target[idx], from s.values' own bulk method)
// together. Instead of one Go string concatenation (= one allocation) per
// row, it sizes a single shared []byte arena for the whole batch, memcpys
// every row's prefix+suffix bytes into it, and wraps each row's slice as a
// zero-copy scm.NewString view — one allocation for the batch instead of one
// per cell.
func (s *StoragePrefix) applyPrefixInPlace(target []scm.Scmer, idxbuf []scm.Scmer, count uint32, stride int) {
	pidxs := make([]int64, count)
	rowLens := make([]int, count)
	total := 0
	idx := 0
	for k := uint32(0); k < count; k++ {
		inner := target[idx]
		if !inner.IsNil() {
			if !inner.IsString() {
				panic("invalid value in prefix storage")
			}
			pidx := idxbuf[k].Int()
			if pidx < 0 || pidx >= int64(len(s.prefixdictionary)) {
				panic("prefix index out of range")
			}
			pidxs[k] = pidx
			rowLens[k] = len(s.prefixdictionary[pidx]) + len(inner.String())
			total += rowLens[k]
		}
		idx += stride
	}

	buf := make([]byte, total)
	offset := 0
	idx = 0
	for k := uint32(0); k < count; k++ {
		inner := target[idx]
		if !inner.IsNil() {
			n := copy(buf[offset:], s.prefixdictionary[pidxs[k]])
			n += copy(buf[offset+n:], inner.String())
			target[idx] = scm.NewString(unsafe.String(&buf[offset], n))
			offset += n
		}
		idx += stride
	}
}

func (s *StoragePrefix) JITEmit(ctx *scm.JITContext, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
	var d52 scm.JITValueDesc
	_ = d52
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
	var d61 scm.JITValueDesc
	_ = d61
	var d62 scm.JITValueDesc
	_ = d62
	var d63 scm.JITValueDesc
	_ = d63
	var d64 scm.JITValueDesc
	_ = d64
	var d65 scm.JITValueDesc
	_ = d65
	var d66 scm.JITValueDesc
	_ = d66
	var d67 scm.JITValueDesc
	_ = d67
	var d68 scm.JITValueDesc
	_ = d68
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
	var d77 scm.JITValueDesc
	_ = d77
	var d78 scm.JITValueDesc
	_ = d78
	var d79 scm.JITValueDesc
	_ = d79
	var d162 scm.JITValueDesc
	_ = d162
	var d163 scm.JITValueDesc
	_ = d163
	var d164 scm.JITValueDesc
	_ = d164
	var d165 scm.JITValueDesc
	_ = d165
	var d166 scm.JITValueDesc
	_ = d166
	var d167 scm.JITValueDesc
	_ = d167
	var d168 scm.JITValueDesc
	_ = d168
	var d169 scm.JITValueDesc
	_ = d169
	var d170 scm.JITValueDesc
	_ = d170
	var d171 scm.JITValueDesc
	_ = d171
	var d172 scm.JITValueDesc
	_ = d172
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
	var bbs [8]scm.BBDescriptor
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
	bbpos_0_6 := int32(-1)
	_ = bbpos_0_6
	lbl7 := ctx.ReserveLabel()
	_ = lbl7
	bbpos_0_7 := int32(-1)
	_ = bbpos_0_7
	lbl8 := ctx.ReserveLabel()
	_ = lbl8
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
		ctx.TrackPointer(unsafe.Pointer((*StorageString)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + uintptr(unsafe.Offsetof((*StoragePrefix)(nil).values))))))
		d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer((*StorageString)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + uintptr(unsafe.Offsetof((*StoragePrefix)(nil).values)))))))), RelocatablePointer: true}
		ctx.EnsureDesc(&d0)
		ctx.EnsureDesc(&d0)
		if d0.Loc == scm.LocRegPair || d0.Loc == scm.LocStackPair || d0.Loc == scm.LocRegTriple || d0.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.EnsureDesc(&idxInt)
		ctx.EnsureDesc(&idxInt)
		if idxInt.Loc == scm.LocRegPair || idxInt.Loc == scm.LocStackPair || idxInt.Loc == scm.LocRegTriple || idxInt.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
		ctx.SyncDesc(&d0)
		ctx.SyncDesc(&idxInt)
		d1 = scm.JITEmitGoCallScmerToFrame(ctx, scm.GoFuncAddr((*StorageString).GetValue), []scm.JITValueDesc{d0, idxInt})
		d1.NoHeapPointer = false
		ctx.StabilizeDescForControlFlow(&d1)
		d3 = d1
		d3.ID = 0
		d2 = ctx.EmitTagEqualsBorrowed(&d3, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
		d4 = d2
		ctx.EnsureDesc(&d4)
		if d4.Loc != scm.LocImm && d4.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
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
				ps5.OverlayValues[3] = d3
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
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			return bbs[2].RenderPS(ps6)
		}
		if !ps.General {
			ps.General = true
			return bbs[0].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d4.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl2)
		snap7 := d0
		snap8 := d1
		snap9 := d2
		snap10 := d3
		snap11 := d4
		alloc12 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc12)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		d3 = snap10
		d4 = snap11
		ctx.RestoreAllocState(alloc12)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		d3 = snap10
		d4 = snap11
		ps13 := scm.PhiState{General: true}
		ps13.OverlayValues = make([]scm.JITValueDesc, 5)
		ps13.OverlayValues[0] = d0
		ps13.OverlayValues[1] = d1
		ps13.OverlayValues[2] = d2
		ps13.OverlayValues[3] = d3
		ps13.OverlayValues[4] = d4
		ps14 := scm.PhiState{General: true}
		ps14.OverlayValues = make([]scm.JITValueDesc, 5)
		ps14.OverlayValues[0] = d0
		ps14.OverlayValues[1] = d1
		ps14.OverlayValues[2] = d2
		ps14.OverlayValues[3] = d3
		ps14.OverlayValues[4] = d4
		snap15 := d0
		snap16 := d1
		snap17 := d2
		snap18 := d3
		snap19 := d4
		alloc20 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps14)
		}
		ctx.RestoreAllocState(alloc20)
		d0 = snap15
		d1 = snap16
		d2 = snap17
		d3 = snap18
		d4 = snap19
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps13)
		}
		return result
		ctx.FreeDesc(&d2)
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
		ctx.ReclaimUntrackedRegs()
		d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d22 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d22)
		ctx.BindReg(r1, &d22)
		ctx.EnsureDesc(&d21)
		if d21.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d21, &d22)
		} else {
			switch d21.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d22, d21)
			case scm.TagInt:
				ctx.EmitMakeInt(d22, d21)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d22, d21)
			case scm.TagNil:
				ctx.EmitMakeNil(d22)
			default:
				ctx.EmitMovPairToResult(&d21, &d22)
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
		if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
			d21 = ps.OverlayValues[21]
		}
		if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
			d22 = ps.OverlayValues[22]
		}
		ctx.ReclaimUntrackedRegs()
		d24 = d1
		d24.ID = 0
		d23 = ctx.EmitIsStringBorrowed(&d24, scm.JITValueDesc{Loc: scm.LocAny})
		d25 = d23
		ctx.EnsureDesc(&d25)
		if d25.Loc != scm.LocImm && d25.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d25.Loc == scm.LocImm {
			if d25.Imm.Bool() {
				if ps.General {
				}
				ps26 := scm.PhiState{General: ps.General}
				ps26.OverlayValues = make([]scm.JITValueDesc, 26)
				ps26.OverlayValues[0] = d0
				ps26.OverlayValues[1] = d1
				ps26.OverlayValues[2] = d2
				ps26.OverlayValues[3] = d3
				ps26.OverlayValues[4] = d4
				ps26.OverlayValues[21] = d21
				ps26.OverlayValues[22] = d22
				ps26.OverlayValues[23] = d23
				ps26.OverlayValues[24] = d24
				ps26.OverlayValues[25] = d25
				return bbs[4].RenderPS(ps26)
			}
			if ps.General {
			}
			ps27 := scm.PhiState{General: ps.General}
			ps27.OverlayValues = make([]scm.JITValueDesc, 26)
			ps27.OverlayValues[0] = d0
			ps27.OverlayValues[1] = d1
			ps27.OverlayValues[2] = d2
			ps27.OverlayValues[3] = d3
			ps27.OverlayValues[4] = d4
			ps27.OverlayValues[21] = d21
			ps27.OverlayValues[22] = d22
			ps27.OverlayValues[23] = d23
			ps27.OverlayValues[24] = d24
			ps27.OverlayValues[25] = d25
			return bbs[3].RenderPS(ps27)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		ctx.EmitCmpRegImm32(d25.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl5)
		snap28 := d0
		snap29 := d1
		snap30 := d2
		snap31 := d3
		snap32 := d4
		snap33 := d21
		snap34 := d22
		snap35 := d23
		snap36 := d24
		snap37 := d25
		alloc38 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc38)
		d0 = snap28
		d1 = snap29
		d2 = snap30
		d3 = snap31
		d4 = snap32
		d21 = snap33
		d22 = snap34
		d23 = snap35
		d24 = snap36
		d25 = snap37
		ctx.RestoreAllocState(alloc38)
		d0 = snap28
		d1 = snap29
		d2 = snap30
		d3 = snap31
		d4 = snap32
		d21 = snap33
		d22 = snap34
		d23 = snap35
		d24 = snap36
		d25 = snap37
		ps39 := scm.PhiState{General: true}
		ps39.OverlayValues = make([]scm.JITValueDesc, 26)
		ps39.OverlayValues[0] = d0
		ps39.OverlayValues[1] = d1
		ps39.OverlayValues[2] = d2
		ps39.OverlayValues[3] = d3
		ps39.OverlayValues[4] = d4
		ps39.OverlayValues[21] = d21
		ps39.OverlayValues[22] = d22
		ps39.OverlayValues[23] = d23
		ps39.OverlayValues[24] = d24
		ps39.OverlayValues[25] = d25
		ps40 := scm.PhiState{General: true}
		ps40.OverlayValues = make([]scm.JITValueDesc, 26)
		ps40.OverlayValues[0] = d0
		ps40.OverlayValues[1] = d1
		ps40.OverlayValues[2] = d2
		ps40.OverlayValues[3] = d3
		ps40.OverlayValues[4] = d4
		ps40.OverlayValues[21] = d21
		ps40.OverlayValues[22] = d22
		ps40.OverlayValues[23] = d23
		ps40.OverlayValues[24] = d24
		ps40.OverlayValues[25] = d25
		snap41 := d0
		snap42 := d1
		snap43 := d2
		snap44 := d3
		snap45 := d4
		snap46 := d21
		snap47 := d22
		snap48 := d23
		snap49 := d24
		snap50 := d25
		alloc51 := ctx.SnapshotAllocState()
		if !bbs[3].Rendered {
			bbs[3].RenderPS(ps40)
		}
		ctx.RestoreAllocState(alloc51)
		d0 = snap41
		d1 = snap42
		d2 = snap43
		d3 = snap44
		d4 = snap45
		d21 = snap46
		d22 = snap47
		d23 = snap48
		d24 = snap49
		d25 = snap50
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps39)
		}
		return result
		ctx.FreeDesc(&d23)
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
		ctx.ReclaimUntrackedRegs()
		d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("invalid value in prefix storage")}
		ctx.EnsureDesc(&d52)
		ctx.EnsureDesc(&d52)
		if d52.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d52.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d52)
			} else if d52.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d52)
			} else if d52.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d52)
			} else if d52.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d52.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d52 = tmpPair
		} else if d52.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d52.Type, Reg: ctx.AllocRegExcept(d52.Reg), Reg2: ctx.AllocRegExcept(d52.Reg)}
			switch d52.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d52)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d52)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d52)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d52)
			d52 = tmpPair
		}
		if d52.Loc != scm.LocRegPair && d52.Loc != scm.LocStackPair && d52.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d52})
		ctx.FreeDesc(&d52)
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		d53 = idxInt
		_ = d53
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl9 := ctx.ReserveLabel()
		_ = lbl9
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl9)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d54 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r2 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r2, thisptr.Reg, off)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			ctx.BindReg(r2, &d54)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d54.Imm.Int()))))}
		} else {
			r3 := ctx.AllocReg()
			ctx.EmitMovRegReg(r3, d54.Reg)
			ctx.EmitShlRegImm8(r3, 56)
			ctx.EmitShrRegImm8(r3, 56)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d55)
		}
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&d53, &d55)
		var d57 scm.JITValueDesc
		if d53.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() * d55.Imm.Int())}
		} else if d53.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
			ctx.EmitImulInt64(scratch, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(scratch, d53.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else {
			r4 := ctx.AllocRegExcept(d53.Reg, d55.Reg)
			ctx.EmitMovRegReg(r4, d53.Reg)
			ctx.EmitImulInt64(r4, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d57)
		}
		if d57.Loc == scm.LocReg && d53.Loc == scm.LocReg && d57.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d57)
		var d58 scm.JITValueDesc
		if d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() / 64)}
		} else {
			r5 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r5, d57.Reg)
			ctx.EmitShrRegImm8(r5, 6)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d58)
		}
		if d58.Loc == scm.LocReg && d57.Loc == scm.LocReg && d58.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d57)
		var d59 scm.JITValueDesc
		if d57.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() % 64)}
		} else {
			r6 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r6, d57.Reg)
			ctx.EmitAndRegImm32(r6, 63)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d59)
		}
		if d59.Loc == scm.LocReg && d57.Loc == scm.LocReg && d59.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d60 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d60 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r7 := ctx.AllocReg()
			r8 := ctx.AllocRegExcept(r7)
			r9 := ctx.AllocRegExcept(r7, r8)
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+16)
			d60 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r7, Reg2: r8, Reg3: r9}
			ctx.BindReg(r7, &d60)
			ctx.BindReg(r8, &d60)
			ctx.BindReg(r9, &d60)
			ctx.BindReg(r7, &d60)
			ctx.BindReg(r8, &d60)
			ctx.BindReg(r9, &d60)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		ctx.ReclaimUntrackedRegs()
		d61 = ctx.EmitLoadScalarSliceElement(&d60, &d58, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d61)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d61, &d59)
		var d62 scm.JITValueDesc
		if d61.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) << uint64(d59.Imm.Int())))}
		} else if d59.Loc == scm.LocImm {
			r10 := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitMovRegReg(r10, d61.Reg)
			ctx.EmitShlRegImm8(r10, uint8(d59.Imm.Int()))
			d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d62)
		} else {
			{
				shiftSrc := d61.Reg
				r11 := ctx.AllocRegExcept(d61.Reg, d59.Reg)
				ctx.EmitMovRegReg(r11, d61.Reg)
				shiftSrc = r11
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d59.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d59.Reg)
				}
				ctx.EmitShlRegCl(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d62)
			}
		}
		if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
			ctx.TransferReg(d61.Reg)
			d61.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d61)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		ctx.EnsureDesc(&d58)
		var d63 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(scratch, d58.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d63)
		}
		if d63.Loc == scm.LocReg && d58.Loc == scm.LocReg && d63.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d58)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d63)
		ctx.ReclaimUntrackedRegs()
		d64 = ctx.EmitLoadScalarSliceElement(&d60, &d63, 8, scm.TagInt)
		ctx.FreeDesc(&d63)
		ctx.ReclaimUntrackedRegs()
		d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d65, &d59)
		var d66 scm.JITValueDesc
		if d65.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d65.Imm.Int() - d59.Imm.Int())}
		} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
			r12 := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegReg(r12, d65.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d66)
		} else if d65.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d65.Imm.Int()))
			ctx.EmitSubInt64(scratch, d59.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d66)
		} else if d59.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegReg(scratch, d65.Reg)
			if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d59.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d66)
		} else {
			r13 := ctx.AllocRegExcept(d65.Reg, d59.Reg)
			ctx.EmitMovRegReg(r13, d65.Reg)
			ctx.EmitSubInt64(r13, d59.Reg)
			d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d66)
		}
		if d66.Loc == scm.LocReg && d65.Loc == scm.LocReg && d66.Reg == d65.Reg {
			ctx.TransferReg(d65.Reg)
			d65.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d64)
		ctx.EnsureDesc(&d66)
		ctx.EnsureDescsTogether(&d64, &d66)
		var d67 scm.JITValueDesc
		if d64.Loc == scm.LocImm && d66.Loc == scm.LocImm {
			d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d64.Imm.Int()) >> uint64(d66.Imm.Int())))}
		} else if d66.Loc == scm.LocImm {
			r14 := ctx.AllocRegExcept(d64.Reg)
			ctx.EmitMovRegReg(r14, d64.Reg)
			ctx.EmitShrRegImm8(r14, uint8(d66.Imm.Int()))
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d67)
		} else {
			{
				shiftSrc := d64.Reg
				r15 := ctx.AllocRegExcept(d64.Reg, d66.Reg)
				ctx.EmitMovRegReg(r15, d64.Reg)
				shiftSrc = r15
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d66.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d66.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d66.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d67)
			}
		}
		if d67.Loc == scm.LocReg && d64.Loc == scm.LocReg && d67.Reg == d64.Reg {
			ctx.TransferReg(d64.Reg)
			d64.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d64)
		ctx.FreeDesc(&d66)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d62)
		ctx.EnsureDesc(&d67)
		var d68 scm.JITValueDesc
		if d62.Loc == scm.LocImm && d67.Loc == scm.LocImm {
			d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() | d67.Imm.Int())}
		} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
			ctx.BindReg(d67.Reg, &d68)
		} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
			r16 := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegReg(r16, d62.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d68)
		} else if d62.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d62.Imm.Int()))
			ctx.EmitOrInt64(scratch, d67.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d68)
		} else if d67.Loc == scm.LocImm {
			r17 := ctx.AllocRegExcept(d62.Reg)
			ctx.EmitMovRegReg(r17, d62.Reg)
			if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r17, int32(d67.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
				ctx.EmitOrInt64(r17, scm.RegR11)
			}
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d68)
		} else {
			r18 := ctx.AllocRegExcept(d62.Reg, d67.Reg)
			ctx.EmitMovRegReg(r18, d62.Reg)
			ctx.EmitOrInt64(r18, d67.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d68)
		}
		if d68.Loc == scm.LocReg && d62.Loc == scm.LocReg && d68.Reg == d62.Reg {
			ctx.TransferReg(d62.Reg)
			d62.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d62)
		ctx.FreeDesc(&d67)
		ctx.ReclaimUntrackedRegs()
		d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&d69, &d55)
		var d70 scm.JITValueDesc
		if d69.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() - d55.Imm.Int())}
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r19 := ctx.AllocRegExcept(d69.Reg)
			ctx.EmitMovRegReg(r19, d69.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d70)
		} else if d69.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
			ctx.EmitSubInt64(scratch, d55.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d70)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d69.Reg)
			ctx.EmitMovRegReg(scratch, d69.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d70)
		} else {
			r20 := ctx.AllocRegExcept(d69.Reg, d55.Reg)
			ctx.EmitMovRegReg(r20, d69.Reg)
			ctx.EmitSubInt64(r20, d55.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
			ctx.BindReg(r20, &d70)
		}
		if d70.Loc == scm.LocReg && d69.Loc == scm.LocReg && d70.Reg == d69.Reg {
			ctx.TransferReg(d69.Reg)
			d69.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d68)
		ctx.EnsureDesc(&d70)
		ctx.EnsureDescsTogether(&d68, &d70)
		var d71 scm.JITValueDesc
		if d68.Loc == scm.LocImm && d70.Loc == scm.LocImm {
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) >> uint64(d70.Imm.Int())))}
		} else if d70.Loc == scm.LocImm {
			r21 := ctx.AllocRegExcept(d68.Reg)
			ctx.EmitMovRegReg(r21, d68.Reg)
			ctx.EmitShrRegImm8(r21, uint8(d70.Imm.Int()))
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d71)
		} else {
			{
				shiftSrc := d68.Reg
				r22 := ctx.AllocRegExcept(d68.Reg, d70.Reg)
				ctx.EmitMovRegReg(r22, d68.Reg)
				shiftSrc = r22
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d70.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d70.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d70.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d71)
			}
		}
		if d71.Loc == scm.LocReg && d68.Loc == scm.LocReg && d71.Reg == d68.Reg {
			ctx.TransferReg(d68.Reg)
			d68.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d68)
		ctx.FreeDesc(&d70)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d71)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d71)
		ctx.EnsureDesc(&d71)
		var d72 scm.JITValueDesc
		if d71.Loc == scm.LocImm {
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d71.Imm.Int()))))}
		} else {
			r23 := ctx.AllocReg()
			ctx.EmitMovRegReg(r23, d71.Reg)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d72)
		}
		ctx.FreeDesc(&d71)
		var d73 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56)
			r24 := ctx.AllocReg()
			ctx.EmitMovRegMem(r24, thisptr.Reg, off)
			d73 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d73)
		}
		ctx.EnsureDesc(&d72)
		ctx.EnsureDesc(&d73)
		ctx.EnsureDescsTogether(&d72, &d73)
		var d74 scm.JITValueDesc
		if d72.Loc == scm.LocImm && d73.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d72.Imm.Int() + d73.Imm.Int())}
		} else if d73.Loc == scm.LocImm && d73.Imm.Int() == 0 {
			r25 := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegReg(r25, d72.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			ctx.BindReg(r25, &d74)
		} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d73.Reg}
			ctx.BindReg(d73.Reg, &d74)
		} else if d72.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
			ctx.EmitAddInt64(scratch, d73.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d74)
		} else if d73.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegReg(scratch, d72.Reg)
			if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d73.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d74)
		} else {
			r26 := ctx.AllocRegExcept(d72.Reg, d73.Reg)
			ctx.EmitMovRegReg(r26, d72.Reg)
			ctx.EmitAddInt64(r26, d73.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d74)
		}
		if d74.Loc == scm.LocReg && d72.Loc == scm.LocReg && d74.Reg == d72.Reg {
			ctx.TransferReg(d72.Reg)
			d72.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d74)
		ctx.FreeDesc(&d72)
		ctx.FreeDesc(&d73)
		var d75 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r27 := ctx.AllocReg()
			r28 := ctx.AllocRegExcept(r27)
			r29 := ctx.AllocRegExcept(r27, r28)
			ctx.EmitMovRegMem64(r27, fieldAddr)
			ctx.EmitMovRegMem64(r28, fieldAddr+8)
			ctx.EmitMovRegMem64(r29, fieldAddr+16)
			d75 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r27, Reg2: r28, Reg3: r29}
			ctx.BindReg(r27, &d75)
			ctx.BindReg(r28, &d75)
			ctx.BindReg(r29, &d75)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r30 := ctx.AllocReg()
			r31 := ctx.AllocRegExcept(r30)
			r32 := ctx.AllocRegExcept(r30, r31)
			ctx.EmitMovRegMem(r30, thisptr.Reg, off)
			ctx.EmitMovRegMem(r31, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r32, thisptr.Reg, off+16)
			d75 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
			ctx.BindReg(r30, &d75)
			ctx.BindReg(r31, &d75)
			ctx.BindReg(r32, &d75)
		}
		var d76 scm.JITValueDesc
		if d75.SliceSizeKnown {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d75.KnownSliceLen))}
		} else if d75.Loc == scm.LocImm {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d75.StackOff))}
		} else if d75.Loc == scm.LocStackTriple {
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d75.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d75)
			if d75.Loc == scm.LocRegPair || d75.Loc == scm.LocRegTriple {
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d75.Reg2, ID: 0}
			} else if d75.Loc == scm.LocReg {
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d75.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d74)
		ctx.EnsureDesc(&d76)
		ctx.EnsureDescsTogether(&d74, &d76)
		var d78 scm.JITValueDesc
		if d74.Loc == scm.LocImm && d76.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d74.Imm.Int() >= d76.Imm.Int())}
		} else if d76.Loc == scm.LocImm {
			r33 := ctx.AllocRegExcept(d74.Reg)
			if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d74.Reg, int32(d76.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
				ctx.EmitCmpInt64(d74.Reg, scm.RegR11)
			}
			d78 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r33, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r33, &d78)
		} else if d74.Loc == scm.LocImm {
			r34 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d76.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r34, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r34, &d78)
		} else {
			r35 := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitCmpInt64(d74.Reg, d76.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r35, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r35, &d78)
		}
		ctx.FreeDesc(&d76)
		d79 = d78
		ctx.EnsureDesc(&d79)
		if d79.Loc != scm.LocImm && d79.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d79.Loc == scm.LocImm {
			if d79.Imm.Bool() {
				if ps.General {
				}
				ps80 := scm.PhiState{General: ps.General}
				ps80.OverlayValues = make([]scm.JITValueDesc, 80)
				ps80.OverlayValues[0] = d0
				ps80.OverlayValues[1] = d1
				ps80.OverlayValues[2] = d2
				ps80.OverlayValues[3] = d3
				ps80.OverlayValues[4] = d4
				ps80.OverlayValues[21] = d21
				ps80.OverlayValues[22] = d22
				ps80.OverlayValues[23] = d23
				ps80.OverlayValues[24] = d24
				ps80.OverlayValues[25] = d25
				ps80.OverlayValues[52] = d52
				ps80.OverlayValues[53] = d53
				ps80.OverlayValues[54] = d54
				ps80.OverlayValues[55] = d55
				ps80.OverlayValues[56] = d56
				ps80.OverlayValues[57] = d57
				ps80.OverlayValues[58] = d58
				ps80.OverlayValues[59] = d59
				ps80.OverlayValues[60] = d60
				ps80.OverlayValues[61] = d61
				ps80.OverlayValues[62] = d62
				ps80.OverlayValues[63] = d63
				ps80.OverlayValues[64] = d64
				ps80.OverlayValues[65] = d65
				ps80.OverlayValues[66] = d66
				ps80.OverlayValues[67] = d67
				ps80.OverlayValues[68] = d68
				ps80.OverlayValues[69] = d69
				ps80.OverlayValues[70] = d70
				ps80.OverlayValues[71] = d71
				ps80.OverlayValues[72] = d72
				ps80.OverlayValues[73] = d73
				ps80.OverlayValues[74] = d74
				ps80.OverlayValues[75] = d75
				ps80.OverlayValues[76] = d76
				ps80.OverlayValues[77] = d77
				ps80.OverlayValues[78] = d78
				ps80.OverlayValues[79] = d79
				return bbs[5].RenderPS(ps80)
			}
			if ps.General {
			}
			ps81 := scm.PhiState{General: ps.General}
			ps81.OverlayValues = make([]scm.JITValueDesc, 80)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[1] = d1
			ps81.OverlayValues[2] = d2
			ps81.OverlayValues[3] = d3
			ps81.OverlayValues[4] = d4
			ps81.OverlayValues[21] = d21
			ps81.OverlayValues[22] = d22
			ps81.OverlayValues[23] = d23
			ps81.OverlayValues[24] = d24
			ps81.OverlayValues[25] = d25
			ps81.OverlayValues[52] = d52
			ps81.OverlayValues[53] = d53
			ps81.OverlayValues[54] = d54
			ps81.OverlayValues[55] = d55
			ps81.OverlayValues[56] = d56
			ps81.OverlayValues[57] = d57
			ps81.OverlayValues[58] = d58
			ps81.OverlayValues[59] = d59
			ps81.OverlayValues[60] = d60
			ps81.OverlayValues[61] = d61
			ps81.OverlayValues[62] = d62
			ps81.OverlayValues[63] = d63
			ps81.OverlayValues[64] = d64
			ps81.OverlayValues[65] = d65
			ps81.OverlayValues[66] = d66
			ps81.OverlayValues[67] = d67
			ps81.OverlayValues[68] = d68
			ps81.OverlayValues[69] = d69
			ps81.OverlayValues[70] = d70
			ps81.OverlayValues[71] = d71
			ps81.OverlayValues[72] = d72
			ps81.OverlayValues[73] = d73
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps81.OverlayValues[79] = d79
			return bbs[7].RenderPS(ps81)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		ctx.EmitJump(d79.Condition, lbl6)
		snap82 := d0
		snap83 := d1
		snap84 := d2
		snap85 := d3
		snap86 := d4
		snap87 := d21
		snap88 := d22
		snap89 := d23
		snap90 := d24
		snap91 := d25
		snap92 := d52
		snap93 := d53
		snap94 := d54
		snap95 := d55
		snap96 := d56
		snap97 := d57
		snap98 := d58
		snap99 := d59
		snap100 := d60
		snap101 := d61
		snap102 := d62
		snap103 := d63
		snap104 := d64
		snap105 := d65
		snap106 := d66
		snap107 := d67
		snap108 := d68
		snap109 := d69
		snap110 := d70
		snap111 := d71
		snap112 := d72
		snap113 := d73
		snap114 := d74
		snap115 := d75
		snap116 := d76
		snap117 := d77
		snap118 := d78
		snap119 := d79
		alloc120 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc120)
		d0 = snap82
		d1 = snap83
		d2 = snap84
		d3 = snap85
		d4 = snap86
		d21 = snap87
		d22 = snap88
		d23 = snap89
		d24 = snap90
		d25 = snap91
		d52 = snap92
		d53 = snap93
		d54 = snap94
		d55 = snap95
		d56 = snap96
		d57 = snap97
		d58 = snap98
		d59 = snap99
		d60 = snap100
		d61 = snap101
		d62 = snap102
		d63 = snap103
		d64 = snap104
		d65 = snap105
		d66 = snap106
		d67 = snap107
		d68 = snap108
		d69 = snap109
		d70 = snap110
		d71 = snap111
		d72 = snap112
		d73 = snap113
		d74 = snap114
		d75 = snap115
		d76 = snap116
		d77 = snap117
		d78 = snap118
		d79 = snap119
		ctx.RestoreAllocState(alloc120)
		d0 = snap82
		d1 = snap83
		d2 = snap84
		d3 = snap85
		d4 = snap86
		d21 = snap87
		d22 = snap88
		d23 = snap89
		d24 = snap90
		d25 = snap91
		d52 = snap92
		d53 = snap93
		d54 = snap94
		d55 = snap95
		d56 = snap96
		d57 = snap97
		d58 = snap98
		d59 = snap99
		d60 = snap100
		d61 = snap101
		d62 = snap102
		d63 = snap103
		d64 = snap104
		d65 = snap105
		d66 = snap106
		d67 = snap107
		d68 = snap108
		d69 = snap109
		d70 = snap110
		d71 = snap111
		d72 = snap112
		d73 = snap113
		d74 = snap114
		d75 = snap115
		d76 = snap116
		d77 = snap117
		d78 = snap118
		d79 = snap119
		ps121 := scm.PhiState{General: true}
		ps121.OverlayValues = make([]scm.JITValueDesc, 80)
		ps121.OverlayValues[0] = d0
		ps121.OverlayValues[1] = d1
		ps121.OverlayValues[2] = d2
		ps121.OverlayValues[3] = d3
		ps121.OverlayValues[4] = d4
		ps121.OverlayValues[21] = d21
		ps121.OverlayValues[22] = d22
		ps121.OverlayValues[23] = d23
		ps121.OverlayValues[24] = d24
		ps121.OverlayValues[25] = d25
		ps121.OverlayValues[52] = d52
		ps121.OverlayValues[53] = d53
		ps121.OverlayValues[54] = d54
		ps121.OverlayValues[55] = d55
		ps121.OverlayValues[56] = d56
		ps121.OverlayValues[57] = d57
		ps121.OverlayValues[58] = d58
		ps121.OverlayValues[59] = d59
		ps121.OverlayValues[60] = d60
		ps121.OverlayValues[61] = d61
		ps121.OverlayValues[62] = d62
		ps121.OverlayValues[63] = d63
		ps121.OverlayValues[64] = d64
		ps121.OverlayValues[65] = d65
		ps121.OverlayValues[66] = d66
		ps121.OverlayValues[67] = d67
		ps121.OverlayValues[68] = d68
		ps121.OverlayValues[69] = d69
		ps121.OverlayValues[70] = d70
		ps121.OverlayValues[71] = d71
		ps121.OverlayValues[72] = d72
		ps121.OverlayValues[73] = d73
		ps121.OverlayValues[74] = d74
		ps121.OverlayValues[75] = d75
		ps121.OverlayValues[76] = d76
		ps121.OverlayValues[77] = d77
		ps121.OverlayValues[78] = d78
		ps121.OverlayValues[79] = d79
		ps122 := scm.PhiState{General: true}
		ps122.OverlayValues = make([]scm.JITValueDesc, 80)
		ps122.OverlayValues[0] = d0
		ps122.OverlayValues[1] = d1
		ps122.OverlayValues[2] = d2
		ps122.OverlayValues[3] = d3
		ps122.OverlayValues[4] = d4
		ps122.OverlayValues[21] = d21
		ps122.OverlayValues[22] = d22
		ps122.OverlayValues[23] = d23
		ps122.OverlayValues[24] = d24
		ps122.OverlayValues[25] = d25
		ps122.OverlayValues[52] = d52
		ps122.OverlayValues[53] = d53
		ps122.OverlayValues[54] = d54
		ps122.OverlayValues[55] = d55
		ps122.OverlayValues[56] = d56
		ps122.OverlayValues[57] = d57
		ps122.OverlayValues[58] = d58
		ps122.OverlayValues[59] = d59
		ps122.OverlayValues[60] = d60
		ps122.OverlayValues[61] = d61
		ps122.OverlayValues[62] = d62
		ps122.OverlayValues[63] = d63
		ps122.OverlayValues[64] = d64
		ps122.OverlayValues[65] = d65
		ps122.OverlayValues[66] = d66
		ps122.OverlayValues[67] = d67
		ps122.OverlayValues[68] = d68
		ps122.OverlayValues[69] = d69
		ps122.OverlayValues[70] = d70
		ps122.OverlayValues[71] = d71
		ps122.OverlayValues[72] = d72
		ps122.OverlayValues[73] = d73
		ps122.OverlayValues[74] = d74
		ps122.OverlayValues[75] = d75
		ps122.OverlayValues[76] = d76
		ps122.OverlayValues[77] = d77
		ps122.OverlayValues[78] = d78
		ps122.OverlayValues[79] = d79
		snap123 := d0
		snap124 := d1
		snap125 := d2
		snap126 := d3
		snap127 := d4
		snap128 := d21
		snap129 := d22
		snap130 := d23
		snap131 := d24
		snap132 := d25
		snap133 := d52
		snap134 := d53
		snap135 := d54
		snap136 := d55
		snap137 := d56
		snap138 := d57
		snap139 := d58
		snap140 := d59
		snap141 := d60
		snap142 := d61
		snap143 := d62
		snap144 := d63
		snap145 := d64
		snap146 := d65
		snap147 := d66
		snap148 := d67
		snap149 := d68
		snap150 := d69
		snap151 := d70
		snap152 := d71
		snap153 := d72
		snap154 := d73
		snap155 := d74
		snap156 := d75
		snap157 := d76
		snap158 := d77
		snap159 := d78
		snap160 := d79
		alloc161 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps122)
		}
		ctx.RestoreAllocState(alloc161)
		d0 = snap123
		d1 = snap124
		d2 = snap125
		d3 = snap126
		d4 = snap127
		d21 = snap128
		d22 = snap129
		d23 = snap130
		d24 = snap131
		d25 = snap132
		d52 = snap133
		d53 = snap134
		d54 = snap135
		d55 = snap136
		d56 = snap137
		d57 = snap138
		d58 = snap139
		d59 = snap140
		d60 = snap141
		d61 = snap142
		d62 = snap143
		d63 = snap144
		d64 = snap145
		d65 = snap146
		d66 = snap147
		d67 = snap148
		d68 = snap149
		d69 = snap150
		d70 = snap151
		d71 = snap152
		d72 = snap153
		d73 = snap154
		d74 = snap155
		d75 = snap156
		d76 = snap157
		d77 = snap158
		d78 = snap159
		d79 = snap160
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps121)
		}
		return result
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
			d63 = ps.OverlayValues[63]
		}
		if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
			d64 = ps.OverlayValues[64]
		}
		if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
			d65 = ps.OverlayValues[65]
		}
		if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
			d66 = ps.OverlayValues[66]
		}
		if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
			d67 = ps.OverlayValues[67]
		}
		if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
			d68 = ps.OverlayValues[68]
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
		d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("prefix index out of range")}
		ctx.EnsureDesc(&d162)
		ctx.EnsureDesc(&d162)
		if d162.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d162.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d162)
			} else if d162.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d162)
			} else if d162.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d162)
			} else if d162.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d162.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d162 = tmpPair
		} else if d162.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d162.Type, Reg: ctx.AllocRegExcept(d162.Reg), Reg2: ctx.AllocRegExcept(d162.Reg)}
			switch d162.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d162)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d162)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d162)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d162)
			d162 = tmpPair
		}
		if d162.Loc != scm.LocRegPair && d162.Loc != scm.LocStackPair && d162.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d162})
		ctx.FreeDesc(&d162)
		return result
	}
	bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[6].VisitCount >= 0 {
				ps.General = true
				return bbs[6].RenderPS(ps)
			}
		}
		bbs[6].VisitCount++
		if ps.General {
			if bbs[6].Rendered {
				ctx.EmitJmp(lbl7)
				return result
			}
			bbs[6].Rendered = true
			bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_6 = bbs[6].Address
			ctx.MarkLabel(lbl7)
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
			d63 = ps.OverlayValues[63]
		}
		if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
			d64 = ps.OverlayValues[64]
		}
		if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
			d65 = ps.OverlayValues[65]
		}
		if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
			d66 = ps.OverlayValues[66]
		}
		if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
			d67 = ps.OverlayValues[67]
		}
		if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
			d68 = ps.OverlayValues[68]
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		ctx.ReclaimUntrackedRegs()
		var d163 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r36 := ctx.AllocReg()
			r37 := ctx.AllocRegExcept(r36)
			r38 := ctx.AllocRegExcept(r36, r37)
			ctx.EmitMovRegMem64(r36, fieldAddr)
			ctx.EmitMovRegMem64(r37, fieldAddr+8)
			ctx.EmitMovRegMem64(r38, fieldAddr+16)
			d163 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r36, Reg2: r37, Reg3: r38}
			ctx.BindReg(r36, &d163)
			ctx.BindReg(r37, &d163)
			ctx.BindReg(r38, &d163)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r39 := ctx.AllocReg()
			r40 := ctx.AllocRegExcept(r39)
			r41 := ctx.AllocRegExcept(r39, r40)
			ctx.EmitMovRegMem(r39, thisptr.Reg, off)
			ctx.EmitMovRegMem(r40, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r41, thisptr.Reg, off+16)
			d163 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r39, Reg2: r40, Reg3: r41}
			ctx.BindReg(r39, &d163)
			ctx.BindReg(r40, &d163)
			ctx.BindReg(r41, &d163)
		}
		ctx.EnsureDesc(&d74)
		d165 = ctx.EmitSliceElementAddress(&d163, &d74, 16)
		ctx.EnsureDesc(&d165)
		r42 := ctx.AllocRegExcept(d165.Reg)
		ctx.EmitMovRegMem(r42, d165.Reg, 8)
		ctx.EmitMovRegMem(d165.Reg, d165.Reg, 0)
		d164 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d165.Reg, Reg2: r42}
		ctx.BindReg(d165.Reg, &d164)
		ctx.BindReg(r42, &d164)
		d167 = d1
		ctx.SyncDesc(&d167)
		if d167.Loc == scm.LocMem {
			tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d167.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d167.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d167 = tmpScalar
		}
		d167 = scm.JITPrepareScmerGoArg(ctx, d167)
		if d167.Loc != scm.LocRegPair && d167.Loc != scm.LocStackPair && d167.Loc != scm.LocInputPair {
			panic("jit: scm.Scmer.String receiver not materialized as pair")
		}
		d166 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d167}, 2)
		ctx.EnsureDesc(&d164)
		ctx.EnsureDesc(&d166)
		d168 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d164, d166}, 2)
		ctx.FreeDesc(&d164)
		ctx.EnsureDesc(&d168)
		d169 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d169)
		ctx.BindReg(r1, &d169)
		d170 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d168}, 2)
		ctx.EmitMovPairToResult(&d170, &d169)
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
		if !ps.General {
			if bbs[7].VisitCount >= 0 {
				ps.General = true
				return bbs[7].RenderPS(ps)
			}
		}
		bbs[7].VisitCount++
		if ps.General {
			if bbs[7].Rendered {
				ctx.EmitJmp(lbl8)
				return result
			}
			bbs[7].Rendered = true
			bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			bbpos_0_7 = bbs[7].Address
			ctx.MarkLabel(lbl8)
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
		if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
			d52 = ps.OverlayValues[52]
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
		if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
			d61 = ps.OverlayValues[61]
		}
		if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
			d62 = ps.OverlayValues[62]
		}
		if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
			d63 = ps.OverlayValues[63]
		}
		if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
			d64 = ps.OverlayValues[64]
		}
		if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
			d65 = ps.OverlayValues[65]
		}
		if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
			d66 = ps.OverlayValues[66]
		}
		if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
			d67 = ps.OverlayValues[67]
		}
		if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
			d68 = ps.OverlayValues[68]
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
		if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
			d77 = ps.OverlayValues[77]
		}
		if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
			d78 = ps.OverlayValues[78]
		}
		if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
			d79 = ps.OverlayValues[79]
		}
		if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != scm.LocNone {
			d162 = ps.OverlayValues[162]
		}
		if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != scm.LocNone {
			d163 = ps.OverlayValues[163]
		}
		if len(ps.OverlayValues) > 164 && ps.OverlayValues[164].Loc != scm.LocNone {
			d164 = ps.OverlayValues[164]
		}
		if len(ps.OverlayValues) > 165 && ps.OverlayValues[165].Loc != scm.LocNone {
			d165 = ps.OverlayValues[165]
		}
		if len(ps.OverlayValues) > 166 && ps.OverlayValues[166].Loc != scm.LocNone {
			d166 = ps.OverlayValues[166]
		}
		if len(ps.OverlayValues) > 167 && ps.OverlayValues[167].Loc != scm.LocNone {
			d167 = ps.OverlayValues[167]
		}
		if len(ps.OverlayValues) > 168 && ps.OverlayValues[168].Loc != scm.LocNone {
			d168 = ps.OverlayValues[168]
		}
		if len(ps.OverlayValues) > 169 && ps.OverlayValues[169].Loc != scm.LocNone {
			d169 = ps.OverlayValues[169]
		}
		if len(ps.OverlayValues) > 170 && ps.OverlayValues[170].Loc != scm.LocNone {
			d170 = ps.OverlayValues[170]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d74)
		var d171 scm.JITValueDesc
		if d74.Loc == scm.LocImm {
			d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d74.Imm.Int() < 0)}
		} else {
			r43 := ctx.AllocRegExcept(d74.Reg)
			ctx.EmitCmpRegImm32(d74.Reg, 0)
			d171 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r43, Condition: scm.CondSignedLess}
			ctx.BindReg(r43, &d171)
		}
		d172 = d171
		ctx.EnsureDesc(&d172)
		if d172.Loc != scm.LocImm && d172.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d172.Loc == scm.LocImm {
			if d172.Imm.Bool() {
				if ps.General {
				}
				ps173 := scm.PhiState{General: ps.General}
				ps173.OverlayValues = make([]scm.JITValueDesc, 173)
				ps173.OverlayValues[0] = d0
				ps173.OverlayValues[1] = d1
				ps173.OverlayValues[2] = d2
				ps173.OverlayValues[3] = d3
				ps173.OverlayValues[4] = d4
				ps173.OverlayValues[21] = d21
				ps173.OverlayValues[22] = d22
				ps173.OverlayValues[23] = d23
				ps173.OverlayValues[24] = d24
				ps173.OverlayValues[25] = d25
				ps173.OverlayValues[52] = d52
				ps173.OverlayValues[53] = d53
				ps173.OverlayValues[54] = d54
				ps173.OverlayValues[55] = d55
				ps173.OverlayValues[56] = d56
				ps173.OverlayValues[57] = d57
				ps173.OverlayValues[58] = d58
				ps173.OverlayValues[59] = d59
				ps173.OverlayValues[60] = d60
				ps173.OverlayValues[61] = d61
				ps173.OverlayValues[62] = d62
				ps173.OverlayValues[63] = d63
				ps173.OverlayValues[64] = d64
				ps173.OverlayValues[65] = d65
				ps173.OverlayValues[66] = d66
				ps173.OverlayValues[67] = d67
				ps173.OverlayValues[68] = d68
				ps173.OverlayValues[69] = d69
				ps173.OverlayValues[70] = d70
				ps173.OverlayValues[71] = d71
				ps173.OverlayValues[72] = d72
				ps173.OverlayValues[73] = d73
				ps173.OverlayValues[74] = d74
				ps173.OverlayValues[75] = d75
				ps173.OverlayValues[76] = d76
				ps173.OverlayValues[77] = d77
				ps173.OverlayValues[78] = d78
				ps173.OverlayValues[79] = d79
				ps173.OverlayValues[162] = d162
				ps173.OverlayValues[163] = d163
				ps173.OverlayValues[164] = d164
				ps173.OverlayValues[165] = d165
				ps173.OverlayValues[166] = d166
				ps173.OverlayValues[167] = d167
				ps173.OverlayValues[168] = d168
				ps173.OverlayValues[169] = d169
				ps173.OverlayValues[170] = d170
				ps173.OverlayValues[171] = d171
				ps173.OverlayValues[172] = d172
				return bbs[5].RenderPS(ps173)
			}
			if ps.General {
			}
			ps174 := scm.PhiState{General: ps.General}
			ps174.OverlayValues = make([]scm.JITValueDesc, 173)
			ps174.OverlayValues[0] = d0
			ps174.OverlayValues[1] = d1
			ps174.OverlayValues[2] = d2
			ps174.OverlayValues[3] = d3
			ps174.OverlayValues[4] = d4
			ps174.OverlayValues[21] = d21
			ps174.OverlayValues[22] = d22
			ps174.OverlayValues[23] = d23
			ps174.OverlayValues[24] = d24
			ps174.OverlayValues[25] = d25
			ps174.OverlayValues[52] = d52
			ps174.OverlayValues[53] = d53
			ps174.OverlayValues[54] = d54
			ps174.OverlayValues[55] = d55
			ps174.OverlayValues[56] = d56
			ps174.OverlayValues[57] = d57
			ps174.OverlayValues[58] = d58
			ps174.OverlayValues[59] = d59
			ps174.OverlayValues[60] = d60
			ps174.OverlayValues[61] = d61
			ps174.OverlayValues[62] = d62
			ps174.OverlayValues[63] = d63
			ps174.OverlayValues[64] = d64
			ps174.OverlayValues[65] = d65
			ps174.OverlayValues[66] = d66
			ps174.OverlayValues[67] = d67
			ps174.OverlayValues[68] = d68
			ps174.OverlayValues[69] = d69
			ps174.OverlayValues[70] = d70
			ps174.OverlayValues[71] = d71
			ps174.OverlayValues[72] = d72
			ps174.OverlayValues[73] = d73
			ps174.OverlayValues[74] = d74
			ps174.OverlayValues[75] = d75
			ps174.OverlayValues[76] = d76
			ps174.OverlayValues[77] = d77
			ps174.OverlayValues[78] = d78
			ps174.OverlayValues[79] = d79
			ps174.OverlayValues[162] = d162
			ps174.OverlayValues[163] = d163
			ps174.OverlayValues[164] = d164
			ps174.OverlayValues[165] = d165
			ps174.OverlayValues[166] = d166
			ps174.OverlayValues[167] = d167
			ps174.OverlayValues[168] = d168
			ps174.OverlayValues[169] = d169
			ps174.OverlayValues[170] = d170
			ps174.OverlayValues[171] = d171
			ps174.OverlayValues[172] = d172
			return bbs[6].RenderPS(ps174)
		}
		if !ps.General {
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		ctx.EmitJump(d172.Condition, lbl6)
		snap175 := d0
		snap176 := d1
		snap177 := d2
		snap178 := d3
		snap179 := d4
		snap180 := d21
		snap181 := d22
		snap182 := d23
		snap183 := d24
		snap184 := d25
		snap185 := d52
		snap186 := d53
		snap187 := d54
		snap188 := d55
		snap189 := d56
		snap190 := d57
		snap191 := d58
		snap192 := d59
		snap193 := d60
		snap194 := d61
		snap195 := d62
		snap196 := d63
		snap197 := d64
		snap198 := d65
		snap199 := d66
		snap200 := d67
		snap201 := d68
		snap202 := d69
		snap203 := d70
		snap204 := d71
		snap205 := d72
		snap206 := d73
		snap207 := d74
		snap208 := d75
		snap209 := d76
		snap210 := d77
		snap211 := d78
		snap212 := d79
		snap213 := d162
		snap214 := d163
		snap215 := d164
		snap216 := d165
		snap217 := d166
		snap218 := d167
		snap219 := d168
		snap220 := d169
		snap221 := d170
		snap222 := d171
		snap223 := d172
		alloc224 := ctx.SnapshotAllocState()
		ctx.RestoreAllocState(alloc224)
		d0 = snap175
		d1 = snap176
		d2 = snap177
		d3 = snap178
		d4 = snap179
		d21 = snap180
		d22 = snap181
		d23 = snap182
		d24 = snap183
		d25 = snap184
		d52 = snap185
		d53 = snap186
		d54 = snap187
		d55 = snap188
		d56 = snap189
		d57 = snap190
		d58 = snap191
		d59 = snap192
		d60 = snap193
		d61 = snap194
		d62 = snap195
		d63 = snap196
		d64 = snap197
		d65 = snap198
		d66 = snap199
		d67 = snap200
		d68 = snap201
		d69 = snap202
		d70 = snap203
		d71 = snap204
		d72 = snap205
		d73 = snap206
		d74 = snap207
		d75 = snap208
		d76 = snap209
		d77 = snap210
		d78 = snap211
		d79 = snap212
		d162 = snap213
		d163 = snap214
		d164 = snap215
		d165 = snap216
		d166 = snap217
		d167 = snap218
		d168 = snap219
		d169 = snap220
		d170 = snap221
		d171 = snap222
		d172 = snap223
		ctx.RestoreAllocState(alloc224)
		d0 = snap175
		d1 = snap176
		d2 = snap177
		d3 = snap178
		d4 = snap179
		d21 = snap180
		d22 = snap181
		d23 = snap182
		d24 = snap183
		d25 = snap184
		d52 = snap185
		d53 = snap186
		d54 = snap187
		d55 = snap188
		d56 = snap189
		d57 = snap190
		d58 = snap191
		d59 = snap192
		d60 = snap193
		d61 = snap194
		d62 = snap195
		d63 = snap196
		d64 = snap197
		d65 = snap198
		d66 = snap199
		d67 = snap200
		d68 = snap201
		d69 = snap202
		d70 = snap203
		d71 = snap204
		d72 = snap205
		d73 = snap206
		d74 = snap207
		d75 = snap208
		d76 = snap209
		d77 = snap210
		d78 = snap211
		d79 = snap212
		d162 = snap213
		d163 = snap214
		d164 = snap215
		d165 = snap216
		d166 = snap217
		d167 = snap218
		d168 = snap219
		d169 = snap220
		d170 = snap221
		d171 = snap222
		d172 = snap223
		ps225 := scm.PhiState{General: true}
		ps225.OverlayValues = make([]scm.JITValueDesc, 173)
		ps225.OverlayValues[0] = d0
		ps225.OverlayValues[1] = d1
		ps225.OverlayValues[2] = d2
		ps225.OverlayValues[3] = d3
		ps225.OverlayValues[4] = d4
		ps225.OverlayValues[21] = d21
		ps225.OverlayValues[22] = d22
		ps225.OverlayValues[23] = d23
		ps225.OverlayValues[24] = d24
		ps225.OverlayValues[25] = d25
		ps225.OverlayValues[52] = d52
		ps225.OverlayValues[53] = d53
		ps225.OverlayValues[54] = d54
		ps225.OverlayValues[55] = d55
		ps225.OverlayValues[56] = d56
		ps225.OverlayValues[57] = d57
		ps225.OverlayValues[58] = d58
		ps225.OverlayValues[59] = d59
		ps225.OverlayValues[60] = d60
		ps225.OverlayValues[61] = d61
		ps225.OverlayValues[62] = d62
		ps225.OverlayValues[63] = d63
		ps225.OverlayValues[64] = d64
		ps225.OverlayValues[65] = d65
		ps225.OverlayValues[66] = d66
		ps225.OverlayValues[67] = d67
		ps225.OverlayValues[68] = d68
		ps225.OverlayValues[69] = d69
		ps225.OverlayValues[70] = d70
		ps225.OverlayValues[71] = d71
		ps225.OverlayValues[72] = d72
		ps225.OverlayValues[73] = d73
		ps225.OverlayValues[74] = d74
		ps225.OverlayValues[75] = d75
		ps225.OverlayValues[76] = d76
		ps225.OverlayValues[77] = d77
		ps225.OverlayValues[78] = d78
		ps225.OverlayValues[79] = d79
		ps225.OverlayValues[162] = d162
		ps225.OverlayValues[163] = d163
		ps225.OverlayValues[164] = d164
		ps225.OverlayValues[165] = d165
		ps225.OverlayValues[166] = d166
		ps225.OverlayValues[167] = d167
		ps225.OverlayValues[168] = d168
		ps225.OverlayValues[169] = d169
		ps225.OverlayValues[170] = d170
		ps225.OverlayValues[171] = d171
		ps225.OverlayValues[172] = d172
		ps226 := scm.PhiState{General: true}
		ps226.OverlayValues = make([]scm.JITValueDesc, 173)
		ps226.OverlayValues[0] = d0
		ps226.OverlayValues[1] = d1
		ps226.OverlayValues[2] = d2
		ps226.OverlayValues[3] = d3
		ps226.OverlayValues[4] = d4
		ps226.OverlayValues[21] = d21
		ps226.OverlayValues[22] = d22
		ps226.OverlayValues[23] = d23
		ps226.OverlayValues[24] = d24
		ps226.OverlayValues[25] = d25
		ps226.OverlayValues[52] = d52
		ps226.OverlayValues[53] = d53
		ps226.OverlayValues[54] = d54
		ps226.OverlayValues[55] = d55
		ps226.OverlayValues[56] = d56
		ps226.OverlayValues[57] = d57
		ps226.OverlayValues[58] = d58
		ps226.OverlayValues[59] = d59
		ps226.OverlayValues[60] = d60
		ps226.OverlayValues[61] = d61
		ps226.OverlayValues[62] = d62
		ps226.OverlayValues[63] = d63
		ps226.OverlayValues[64] = d64
		ps226.OverlayValues[65] = d65
		ps226.OverlayValues[66] = d66
		ps226.OverlayValues[67] = d67
		ps226.OverlayValues[68] = d68
		ps226.OverlayValues[69] = d69
		ps226.OverlayValues[70] = d70
		ps226.OverlayValues[71] = d71
		ps226.OverlayValues[72] = d72
		ps226.OverlayValues[73] = d73
		ps226.OverlayValues[74] = d74
		ps226.OverlayValues[75] = d75
		ps226.OverlayValues[76] = d76
		ps226.OverlayValues[77] = d77
		ps226.OverlayValues[78] = d78
		ps226.OverlayValues[79] = d79
		ps226.OverlayValues[162] = d162
		ps226.OverlayValues[163] = d163
		ps226.OverlayValues[164] = d164
		ps226.OverlayValues[165] = d165
		ps226.OverlayValues[166] = d166
		ps226.OverlayValues[167] = d167
		ps226.OverlayValues[168] = d168
		ps226.OverlayValues[169] = d169
		ps226.OverlayValues[170] = d170
		ps226.OverlayValues[171] = d171
		ps226.OverlayValues[172] = d172
		snap227 := d0
		snap228 := d1
		snap229 := d2
		snap230 := d3
		snap231 := d4
		snap232 := d21
		snap233 := d22
		snap234 := d23
		snap235 := d24
		snap236 := d25
		snap237 := d52
		snap238 := d53
		snap239 := d54
		snap240 := d55
		snap241 := d56
		snap242 := d57
		snap243 := d58
		snap244 := d59
		snap245 := d60
		snap246 := d61
		snap247 := d62
		snap248 := d63
		snap249 := d64
		snap250 := d65
		snap251 := d66
		snap252 := d67
		snap253 := d68
		snap254 := d69
		snap255 := d70
		snap256 := d71
		snap257 := d72
		snap258 := d73
		snap259 := d74
		snap260 := d75
		snap261 := d76
		snap262 := d77
		snap263 := d78
		snap264 := d79
		snap265 := d162
		snap266 := d163
		snap267 := d164
		snap268 := d165
		snap269 := d166
		snap270 := d167
		snap271 := d168
		snap272 := d169
		snap273 := d170
		snap274 := d171
		snap275 := d172
		alloc276 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps226)
		}
		ctx.RestoreAllocState(alloc276)
		d0 = snap227
		d1 = snap228
		d2 = snap229
		d3 = snap230
		d4 = snap231
		d21 = snap232
		d22 = snap233
		d23 = snap234
		d24 = snap235
		d25 = snap236
		d52 = snap237
		d53 = snap238
		d54 = snap239
		d55 = snap240
		d56 = snap241
		d57 = snap242
		d58 = snap243
		d59 = snap244
		d60 = snap245
		d61 = snap246
		d62 = snap247
		d63 = snap248
		d64 = snap249
		d65 = snap250
		d66 = snap251
		d67 = snap252
		d68 = snap253
		d69 = snap254
		d70 = snap255
		d71 = snap256
		d72 = snap257
		d73 = snap258
		d74 = snap259
		d75 = snap260
		d76 = snap261
		d77 = snap262
		d78 = snap263
		d79 = snap264
		d162 = snap265
		d163 = snap266
		d164 = snap267
		d165 = snap268
		d166 = snap269
		d167 = snap270
		d168 = snap271
		d169 = snap272
		d170 = snap273
		d171 = snap274
		d172 = snap275
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps225)
		}
		return result
		return result
	}
	ps277 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps277)
	ctx.MarkLabel(lbl0)
	d278 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d278)
	ctx.BindReg(r1, &d278)
	ctx.EmitMovPairToResult(&d278, &result)
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

func (s *StoragePrefix) prepare() {
	// set up scan
	s.prefixes.prepare()
	s.values.prepare()
}
func (s *StoragePrefix) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		s.values.scan(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.scan(i, scm.NewInt(int64(pfid)))
			s.values.scan(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) init(i uint32) {
	s.prefixes.init(i)
	s.values.init(i)
}
func (s *StoragePrefix) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values.build(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.build(i, scm.NewInt(int64(pfid)))
			s.values.build(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) finish() {
	s.prefixes.finish()
	s.values.finish()
	s.storageJITFunctions.finish(s)
}
func (s *StoragePrefix) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	// TODO: if s.values proposes a StoragePrefix, build it into our cascade??
	return nil
}

func (s *StoragePrefix) DistinctCount() uint { return s.values.DistinctCount() }
