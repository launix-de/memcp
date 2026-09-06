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
	var d80 scm.JITValueDesc
	_ = d80
	var d81 scm.JITValueDesc
	_ = d81
	var d82 scm.JITValueDesc
	_ = d82
	var d83 scm.JITValueDesc
	_ = d83
	var d174 scm.JITValueDesc
	_ = d174
	var d175 scm.JITValueDesc
	_ = d175
	var d176 scm.JITValueDesc
	_ = d176
	var d177 scm.JITValueDesc
	_ = d177
	var d178 scm.JITValueDesc
	_ = d178
	var d179 scm.JITValueDesc
	_ = d179
	var d180 scm.JITValueDesc
	_ = d180
	var d181 scm.JITValueDesc
	_ = d181
	var d182 scm.JITValueDesc
	_ = d182
	var d183 scm.JITValueDesc
	_ = d183
	var d184 scm.JITValueDesc
	_ = d184
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
		lbl9 := ctx.ReserveLabel()
		lbl10 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d4.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl9)
		ctx.EmitJmp(lbl10)
		snap7 := d0
		snap8 := d1
		snap9 := d2
		snap10 := d3
		snap11 := d4
		alloc12 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl2)
		ctx.RestoreAllocState(alloc12)
		d0 = snap7
		d1 = snap8
		d2 = snap9
		d3 = snap10
		d4 = snap11
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl3)
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
		lbl11 := ctx.ReserveLabel()
		lbl12 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d25.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl11)
		ctx.EmitJmp(lbl12)
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
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl5)
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
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl4)
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
		ctx.StabilizeDescForControlFlow(&d53)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl13 := ctx.ReserveLabel()
		_ = lbl13
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d53)
		var d54 scm.JITValueDesc
		if d53.Loc == scm.LocImm {
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d53.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d53.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d54)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d55 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r3, thisptr.Reg, off)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d55)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d55)
		ctx.EnsureDesc(&d55)
		var d56 scm.JITValueDesc
		if d55.Loc == scm.LocImm {
			d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d55.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d55.Reg)
			ctx.EmitShlRegImm8(r4, 56)
			ctx.EmitShrRegImm8(r4, 56)
			d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d56)
		}
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d56)
		ctx.EnsureDescsTogether(&d54, &d56)
		var d57 scm.JITValueDesc
		if d54.Loc == scm.LocImm && d56.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() * d56.Imm.Int())}
		} else if d54.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
			ctx.EmitImulInt64(scratch, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d56.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d54.Reg)
			ctx.EmitMovRegReg(scratch, d54.Reg)
			if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d56.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else {
			r5 := ctx.AllocRegExcept(d54.Reg, d56.Reg)
			ctx.EmitMovRegReg(r5, d54.Reg)
			ctx.EmitImulInt64(r5, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d57)
		}
		if d57.Loc == scm.LocReg && d54.Loc == scm.LocReg && d57.Reg == d54.Reg {
			ctx.TransferReg(d54.Reg)
			d54.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d54)
		ctx.FreeDesc(&d56)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d57)
		var d58 scm.JITValueDesc
		if d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() / 64)}
		} else {
			r6 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r6, d57.Reg)
			ctx.EmitShrRegImm8(r6, 6)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d58)
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
			r7 := ctx.AllocRegExcept(d57.Reg)
			ctx.EmitMovRegReg(r7, d57.Reg)
			ctx.EmitAndRegImm32(r7, 63)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d59)
		}
		if d59.Loc == scm.LocReg && d57.Loc == scm.LocReg && d59.Reg == d57.Reg {
			ctx.TransferReg(d57.Reg)
			d57.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d60 scm.JITValueDesc
		r8 := ctx.AllocReg()
		r9 := ctx.AllocRegExcept(r8)
		r10 := ctx.AllocRegExcept(r8, r9)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			ctx.EmitMovRegImm64(r8, uint64(dataPtr))
			ctx.EmitMovRegImm64(r9, uint64(sliceLen))
			ctx.EmitMovRegImm64(r10, uint64(sliceCap))
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
			ctx.EmitMovRegMem(r8, thisptr.Reg, off)
			ctx.EmitMovRegMem(r9, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r10, thisptr.Reg, off+16)
		}
		d60 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
		ctx.BindReg(r8, &d60)
		ctx.BindReg(r9, &d60)
		ctx.BindReg(r10, &d60)
		ctx.BindReg(r8, &d60)
		ctx.BindReg(r9, &d60)
		ctx.BindReg(r10, &d60)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		ctx.ReclaimUntrackedRegs()
		d62 = ctx.EmitSliceElementAddress(&d60, &d58, 8)
		ctx.EnsureDesc(&d62)
		ctx.EmitMovRegMem(d62.Reg, d62.Reg, 0)
		d61 = d62
		d61.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d61)
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d61, &d59)
		var d63 scm.JITValueDesc
		if d61.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) << uint64(d59.Imm.Int())))}
		} else if d59.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitMovRegReg(r11, d61.Reg)
			ctx.EmitShlRegImm8(r11, uint8(d59.Imm.Int()))
			d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d63)
		} else {
			{
				shiftSrc := d61.Reg
				r12 := ctx.AllocRegExcept(d61.Reg, d59.Reg)
				ctx.EmitMovRegReg(r12, d61.Reg)
				shiftSrc = r12
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d59.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d59.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d63)
			}
		}
		if d63.Loc == scm.LocReg && d61.Loc == scm.LocReg && d63.Reg == d61.Reg {
			ctx.TransferReg(d61.Reg)
			d61.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d61)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		ctx.EnsureDesc(&d58)
		var d64 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d58.Reg)
			ctx.EmitMovRegReg(scratch, d58.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d64)
		}
		if d64.Loc == scm.LocReg && d58.Loc == scm.LocReg && d64.Reg == d58.Reg {
			ctx.TransferReg(d58.Reg)
			d58.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d58)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d64)
		ctx.ReclaimUntrackedRegs()
		d66 = ctx.EmitSliceElementAddress(&d60, &d64, 8)
		ctx.EnsureDesc(&d66)
		ctx.EmitMovRegMem(d66.Reg, d66.Reg, 0)
		d65 = d66
		d65.Type = scm.TagInt
		ctx.FreeDesc(&d64)
		ctx.ReclaimUntrackedRegs()
		d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d59)
		ctx.EnsureDescsTogether(&d67, &d59)
		var d68 scm.JITValueDesc
		if d67.Loc == scm.LocImm && d59.Loc == scm.LocImm {
			d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() - d59.Imm.Int())}
		} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
			r13 := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegReg(r13, d67.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d68)
		} else if d67.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
			ctx.EmitSubInt64(scratch, d59.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d68)
		} else if d59.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d67.Reg)
			ctx.EmitMovRegReg(scratch, d67.Reg)
			if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d59.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d68)
		} else {
			r14 := ctx.AllocRegExcept(d67.Reg, d59.Reg)
			ctx.EmitMovRegReg(r14, d67.Reg)
			ctx.EmitSubInt64(r14, d59.Reg)
			d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d68)
		}
		if d68.Loc == scm.LocReg && d67.Loc == scm.LocReg && d68.Reg == d67.Reg {
			ctx.TransferReg(d67.Reg)
			d67.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d59)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d65)
		ctx.EnsureDesc(&d68)
		ctx.EnsureDescsTogether(&d65, &d68)
		var d69 scm.JITValueDesc
		if d65.Loc == scm.LocImm && d68.Loc == scm.LocImm {
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) >> uint64(d68.Imm.Int())))}
		} else if d68.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d65.Reg)
			ctx.EmitMovRegReg(r15, d65.Reg)
			ctx.EmitShrRegImm8(r15, uint8(d68.Imm.Int()))
			d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d69)
		} else {
			{
				shiftSrc := d65.Reg
				r16 := ctx.AllocRegExcept(d65.Reg, d68.Reg)
				ctx.EmitMovRegReg(r16, d65.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d68.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d68.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d68.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d69)
			}
		}
		if d69.Loc == scm.LocReg && d65.Loc == scm.LocReg && d69.Reg == d65.Reg {
			ctx.TransferReg(d65.Reg)
			d65.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d65)
		ctx.FreeDesc(&d68)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d69)
		var d70 scm.JITValueDesc
		if d63.Loc == scm.LocImm && d69.Loc == scm.LocImm {
			d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() | d69.Imm.Int())}
		} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d69.Reg}
			ctx.BindReg(d69.Reg, &d70)
		} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
			r17 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(r17, d63.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d70)
		} else if d63.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d69.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
			ctx.EmitOrInt64(scratch, d69.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d70)
		} else if d69.Loc == scm.LocImm {
			r18 := ctx.AllocRegExcept(d63.Reg)
			ctx.EmitMovRegReg(r18, d63.Reg)
			if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r18, int32(d69.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
				ctx.EmitOrInt64(r18, scm.RegR11)
			}
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d70)
		} else {
			r19 := ctx.AllocRegExcept(d63.Reg, d69.Reg)
			ctx.EmitMovRegReg(r19, d63.Reg)
			ctx.EmitOrInt64(r19, d69.Reg)
			d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d70)
		}
		if d70.Loc == scm.LocReg && d63.Loc == scm.LocReg && d70.Reg == d63.Reg {
			ctx.TransferReg(d63.Reg)
			d63.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d63)
		ctx.FreeDesc(&d69)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d71 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r20 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r20, thisptr.Reg, off)
			d71 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d71)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d71)
		ctx.EnsureDesc(&d71)
		var d72 scm.JITValueDesc
		if d71.Loc == scm.LocImm {
			d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d71.Imm.Int()))))}
		} else {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegReg(r21, d71.Reg)
			ctx.EmitShlRegImm8(r21, 56)
			ctx.EmitShrRegImm8(r21, 56)
			d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d72)
		}
		ctx.FreeDesc(&d71)
		ctx.ReclaimUntrackedRegs()
		d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d72)
		ctx.EnsureDescsTogether(&d73, &d72)
		var d74 scm.JITValueDesc
		if d73.Loc == scm.LocImm && d72.Loc == scm.LocImm {
			d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d73.Imm.Int() - d72.Imm.Int())}
		} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
			r22 := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegReg(r22, d73.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d74)
		} else if d73.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d72.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d73.Imm.Int()))
			ctx.EmitSubInt64(scratch, d72.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d74)
		} else if d72.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d73.Reg)
			ctx.EmitMovRegReg(scratch, d73.Reg)
			if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d72.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d74)
		} else {
			r23 := ctx.AllocRegExcept(d73.Reg, d72.Reg)
			ctx.EmitMovRegReg(r23, d73.Reg)
			ctx.EmitSubInt64(r23, d72.Reg)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d74)
		}
		if d74.Loc == scm.LocReg && d73.Loc == scm.LocReg && d74.Reg == d73.Reg {
			ctx.TransferReg(d73.Reg)
			d73.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d72)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d70)
		ctx.EnsureDesc(&d74)
		ctx.EnsureDescsTogether(&d70, &d74)
		var d75 scm.JITValueDesc
		if d70.Loc == scm.LocImm && d74.Loc == scm.LocImm {
			d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d70.Imm.Int()) >> uint64(d74.Imm.Int())))}
		} else if d74.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d70.Reg)
			ctx.EmitMovRegReg(r24, d70.Reg)
			ctx.EmitShrRegImm8(r24, uint8(d74.Imm.Int()))
			d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d75)
		} else {
			{
				shiftSrc := d70.Reg
				r25 := ctx.AllocRegExcept(d70.Reg, d74.Reg)
				ctx.EmitMovRegReg(r25, d70.Reg)
				shiftSrc = r25
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d74.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d74.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d74.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d75)
			}
		}
		if d75.Loc == scm.LocReg && d70.Loc == scm.LocReg && d75.Reg == d70.Reg {
			ctx.TransferReg(d70.Reg)
			d70.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d70)
		ctx.FreeDesc(&d74)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d75)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d75)
		ctx.EnsureDesc(&d75)
		var d76 scm.JITValueDesc
		if d75.Loc == scm.LocImm {
			d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d75.Imm.Int()))))}
		} else {
			r26 := ctx.AllocReg()
			ctx.EmitMovRegReg(r26, d75.Reg)
			d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d76)
		}
		ctx.FreeDesc(&d75)
		var d77 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56)
			r27 := ctx.AllocReg()
			ctx.EmitMovRegMem(r27, thisptr.Reg, off)
			d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
			ctx.BindReg(r27, &d77)
		}
		ctx.EnsureDesc(&d76)
		ctx.EnsureDesc(&d77)
		ctx.EnsureDescsTogether(&d76, &d77)
		var d78 scm.JITValueDesc
		if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
			d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() + d77.Imm.Int())}
		} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d76.Reg)
			ctx.EmitMovRegReg(r28, d76.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d78)
		} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
			ctx.BindReg(d77.Reg, &d78)
		} else if d76.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d77.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d76.Imm.Int()))
			ctx.EmitAddInt64(scratch, d77.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d78)
		} else if d77.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d76.Reg)
			ctx.EmitMovRegReg(scratch, d76.Reg)
			if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d77.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d78)
		} else {
			r29 := ctx.AllocRegExcept(d76.Reg, d77.Reg)
			ctx.EmitMovRegReg(r29, d76.Reg)
			ctx.EmitAddInt64(r29, d77.Reg)
			d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d78)
		}
		if d78.Loc == scm.LocReg && d76.Loc == scm.LocReg && d78.Reg == d76.Reg {
			ctx.TransferReg(d76.Reg)
			d76.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d78)
		ctx.FreeDesc(&d76)
		ctx.FreeDesc(&d77)
		var d79 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r30 := ctx.AllocReg()
			r31 := ctx.AllocRegExcept(r30)
			r32 := ctx.AllocRegExcept(r30, r31)
			ctx.EmitMovRegMem64(r30, fieldAddr)
			ctx.EmitMovRegMem64(r31, fieldAddr+8)
			ctx.EmitMovRegMem64(r32, fieldAddr+16)
			d79 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
			ctx.BindReg(r30, &d79)
			ctx.BindReg(r31, &d79)
			ctx.BindReg(r32, &d79)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r33 := ctx.AllocReg()
			r34 := ctx.AllocRegExcept(r33)
			r35 := ctx.AllocRegExcept(r33, r34)
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			ctx.EmitMovRegMem(r34, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r35, thisptr.Reg, off+16)
			d79 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
			ctx.BindReg(r33, &d79)
			ctx.BindReg(r34, &d79)
			ctx.BindReg(r35, &d79)
		}
		var d80 scm.JITValueDesc
		if d79.SliceSizeKnown {
			d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d79.KnownSliceLen))}
		} else if d79.Loc == scm.LocImm {
			d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d79.StackOff))}
		} else if d79.Loc == scm.LocStackTriple {
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d79.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d79)
			if d79.Loc == scm.LocRegPair || d79.Loc == scm.LocRegTriple {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg2, ID: 0}
			} else if d79.Loc == scm.LocReg {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d80)
		ctx.EnsureDesc(&d80)
		ctx.EnsureDesc(&d78)
		ctx.EnsureDesc(&d80)
		ctx.EnsureDescsTogether(&d78, &d80)
		var d82 scm.JITValueDesc
		if d78.Loc == scm.LocImm && d80.Loc == scm.LocImm {
			d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d78.Imm.Int() >= d80.Imm.Int())}
		} else if d80.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d78.Reg)
			if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d78.Reg, int32(d80.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
				ctx.EmitCmpInt64(d78.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r36, scm.CondSignedGreaterOrEqual)
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			ctx.BindReg(r36, &d82)
		} else if d78.Loc == scm.LocImm {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d80.Reg)
			ctx.EmitSetcc(r37, scm.CondSignedGreaterOrEqual)
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
			ctx.BindReg(r37, &d82)
		} else {
			r38 := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitCmpInt64(d78.Reg, d80.Reg)
			ctx.EmitSetcc(r38, scm.CondSignedGreaterOrEqual)
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
			ctx.BindReg(r38, &d82)
		}
		ctx.FreeDesc(&d80)
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
				ps84.OverlayValues[1] = d1
				ps84.OverlayValues[2] = d2
				ps84.OverlayValues[3] = d3
				ps84.OverlayValues[4] = d4
				ps84.OverlayValues[21] = d21
				ps84.OverlayValues[22] = d22
				ps84.OverlayValues[23] = d23
				ps84.OverlayValues[24] = d24
				ps84.OverlayValues[25] = d25
				ps84.OverlayValues[52] = d52
				ps84.OverlayValues[53] = d53
				ps84.OverlayValues[54] = d54
				ps84.OverlayValues[55] = d55
				ps84.OverlayValues[56] = d56
				ps84.OverlayValues[57] = d57
				ps84.OverlayValues[58] = d58
				ps84.OverlayValues[59] = d59
				ps84.OverlayValues[60] = d60
				ps84.OverlayValues[61] = d61
				ps84.OverlayValues[62] = d62
				ps84.OverlayValues[63] = d63
				ps84.OverlayValues[64] = d64
				ps84.OverlayValues[65] = d65
				ps84.OverlayValues[66] = d66
				ps84.OverlayValues[67] = d67
				ps84.OverlayValues[68] = d68
				ps84.OverlayValues[69] = d69
				ps84.OverlayValues[70] = d70
				ps84.OverlayValues[71] = d71
				ps84.OverlayValues[72] = d72
				ps84.OverlayValues[73] = d73
				ps84.OverlayValues[74] = d74
				ps84.OverlayValues[75] = d75
				ps84.OverlayValues[76] = d76
				ps84.OverlayValues[77] = d77
				ps84.OverlayValues[78] = d78
				ps84.OverlayValues[79] = d79
				ps84.OverlayValues[80] = d80
				ps84.OverlayValues[81] = d81
				ps84.OverlayValues[82] = d82
				ps84.OverlayValues[83] = d83
				return bbs[5].RenderPS(ps84)
			}
			if ps.General {
			}
			ps85 := scm.PhiState{General: ps.General}
			ps85.OverlayValues = make([]scm.JITValueDesc, 84)
			ps85.OverlayValues[0] = d0
			ps85.OverlayValues[1] = d1
			ps85.OverlayValues[2] = d2
			ps85.OverlayValues[3] = d3
			ps85.OverlayValues[4] = d4
			ps85.OverlayValues[21] = d21
			ps85.OverlayValues[22] = d22
			ps85.OverlayValues[23] = d23
			ps85.OverlayValues[24] = d24
			ps85.OverlayValues[25] = d25
			ps85.OverlayValues[52] = d52
			ps85.OverlayValues[53] = d53
			ps85.OverlayValues[54] = d54
			ps85.OverlayValues[55] = d55
			ps85.OverlayValues[56] = d56
			ps85.OverlayValues[57] = d57
			ps85.OverlayValues[58] = d58
			ps85.OverlayValues[59] = d59
			ps85.OverlayValues[60] = d60
			ps85.OverlayValues[61] = d61
			ps85.OverlayValues[62] = d62
			ps85.OverlayValues[63] = d63
			ps85.OverlayValues[64] = d64
			ps85.OverlayValues[65] = d65
			ps85.OverlayValues[66] = d66
			ps85.OverlayValues[67] = d67
			ps85.OverlayValues[68] = d68
			ps85.OverlayValues[69] = d69
			ps85.OverlayValues[70] = d70
			ps85.OverlayValues[71] = d71
			ps85.OverlayValues[72] = d72
			ps85.OverlayValues[73] = d73
			ps85.OverlayValues[74] = d74
			ps85.OverlayValues[75] = d75
			ps85.OverlayValues[76] = d76
			ps85.OverlayValues[77] = d77
			ps85.OverlayValues[78] = d78
			ps85.OverlayValues[79] = d79
			ps85.OverlayValues[80] = d80
			ps85.OverlayValues[81] = d81
			ps85.OverlayValues[82] = d82
			ps85.OverlayValues[83] = d83
			return bbs[7].RenderPS(ps85)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d83.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		snap86 := d0
		snap87 := d1
		snap88 := d2
		snap89 := d3
		snap90 := d4
		snap91 := d21
		snap92 := d22
		snap93 := d23
		snap94 := d24
		snap95 := d25
		snap96 := d52
		snap97 := d53
		snap98 := d54
		snap99 := d55
		snap100 := d56
		snap101 := d57
		snap102 := d58
		snap103 := d59
		snap104 := d60
		snap105 := d61
		snap106 := d62
		snap107 := d63
		snap108 := d64
		snap109 := d65
		snap110 := d66
		snap111 := d67
		snap112 := d68
		snap113 := d69
		snap114 := d70
		snap115 := d71
		snap116 := d72
		snap117 := d73
		snap118 := d74
		snap119 := d75
		snap120 := d76
		snap121 := d77
		snap122 := d78
		snap123 := d79
		snap124 := d80
		snap125 := d81
		snap126 := d82
		snap127 := d83
		alloc128 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc128)
		d0 = snap86
		d1 = snap87
		d2 = snap88
		d3 = snap89
		d4 = snap90
		d21 = snap91
		d22 = snap92
		d23 = snap93
		d24 = snap94
		d25 = snap95
		d52 = snap96
		d53 = snap97
		d54 = snap98
		d55 = snap99
		d56 = snap100
		d57 = snap101
		d58 = snap102
		d59 = snap103
		d60 = snap104
		d61 = snap105
		d62 = snap106
		d63 = snap107
		d64 = snap108
		d65 = snap109
		d66 = snap110
		d67 = snap111
		d68 = snap112
		d69 = snap113
		d70 = snap114
		d71 = snap115
		d72 = snap116
		d73 = snap117
		d74 = snap118
		d75 = snap119
		d76 = snap120
		d77 = snap121
		d78 = snap122
		d79 = snap123
		d80 = snap124
		d81 = snap125
		d82 = snap126
		d83 = snap127
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl8)
		ctx.RestoreAllocState(alloc128)
		d0 = snap86
		d1 = snap87
		d2 = snap88
		d3 = snap89
		d4 = snap90
		d21 = snap91
		d22 = snap92
		d23 = snap93
		d24 = snap94
		d25 = snap95
		d52 = snap96
		d53 = snap97
		d54 = snap98
		d55 = snap99
		d56 = snap100
		d57 = snap101
		d58 = snap102
		d59 = snap103
		d60 = snap104
		d61 = snap105
		d62 = snap106
		d63 = snap107
		d64 = snap108
		d65 = snap109
		d66 = snap110
		d67 = snap111
		d68 = snap112
		d69 = snap113
		d70 = snap114
		d71 = snap115
		d72 = snap116
		d73 = snap117
		d74 = snap118
		d75 = snap119
		d76 = snap120
		d77 = snap121
		d78 = snap122
		d79 = snap123
		d80 = snap124
		d81 = snap125
		d82 = snap126
		d83 = snap127
		ps129 := scm.PhiState{General: true}
		ps129.OverlayValues = make([]scm.JITValueDesc, 84)
		ps129.OverlayValues[0] = d0
		ps129.OverlayValues[1] = d1
		ps129.OverlayValues[2] = d2
		ps129.OverlayValues[3] = d3
		ps129.OverlayValues[4] = d4
		ps129.OverlayValues[21] = d21
		ps129.OverlayValues[22] = d22
		ps129.OverlayValues[23] = d23
		ps129.OverlayValues[24] = d24
		ps129.OverlayValues[25] = d25
		ps129.OverlayValues[52] = d52
		ps129.OverlayValues[53] = d53
		ps129.OverlayValues[54] = d54
		ps129.OverlayValues[55] = d55
		ps129.OverlayValues[56] = d56
		ps129.OverlayValues[57] = d57
		ps129.OverlayValues[58] = d58
		ps129.OverlayValues[59] = d59
		ps129.OverlayValues[60] = d60
		ps129.OverlayValues[61] = d61
		ps129.OverlayValues[62] = d62
		ps129.OverlayValues[63] = d63
		ps129.OverlayValues[64] = d64
		ps129.OverlayValues[65] = d65
		ps129.OverlayValues[66] = d66
		ps129.OverlayValues[67] = d67
		ps129.OverlayValues[68] = d68
		ps129.OverlayValues[69] = d69
		ps129.OverlayValues[70] = d70
		ps129.OverlayValues[71] = d71
		ps129.OverlayValues[72] = d72
		ps129.OverlayValues[73] = d73
		ps129.OverlayValues[74] = d74
		ps129.OverlayValues[75] = d75
		ps129.OverlayValues[76] = d76
		ps129.OverlayValues[77] = d77
		ps129.OverlayValues[78] = d78
		ps129.OverlayValues[79] = d79
		ps129.OverlayValues[80] = d80
		ps129.OverlayValues[81] = d81
		ps129.OverlayValues[82] = d82
		ps129.OverlayValues[83] = d83
		ps130 := scm.PhiState{General: true}
		ps130.OverlayValues = make([]scm.JITValueDesc, 84)
		ps130.OverlayValues[0] = d0
		ps130.OverlayValues[1] = d1
		ps130.OverlayValues[2] = d2
		ps130.OverlayValues[3] = d3
		ps130.OverlayValues[4] = d4
		ps130.OverlayValues[21] = d21
		ps130.OverlayValues[22] = d22
		ps130.OverlayValues[23] = d23
		ps130.OverlayValues[24] = d24
		ps130.OverlayValues[25] = d25
		ps130.OverlayValues[52] = d52
		ps130.OverlayValues[53] = d53
		ps130.OverlayValues[54] = d54
		ps130.OverlayValues[55] = d55
		ps130.OverlayValues[56] = d56
		ps130.OverlayValues[57] = d57
		ps130.OverlayValues[58] = d58
		ps130.OverlayValues[59] = d59
		ps130.OverlayValues[60] = d60
		ps130.OverlayValues[61] = d61
		ps130.OverlayValues[62] = d62
		ps130.OverlayValues[63] = d63
		ps130.OverlayValues[64] = d64
		ps130.OverlayValues[65] = d65
		ps130.OverlayValues[66] = d66
		ps130.OverlayValues[67] = d67
		ps130.OverlayValues[68] = d68
		ps130.OverlayValues[69] = d69
		ps130.OverlayValues[70] = d70
		ps130.OverlayValues[71] = d71
		ps130.OverlayValues[72] = d72
		ps130.OverlayValues[73] = d73
		ps130.OverlayValues[74] = d74
		ps130.OverlayValues[75] = d75
		ps130.OverlayValues[76] = d76
		ps130.OverlayValues[77] = d77
		ps130.OverlayValues[78] = d78
		ps130.OverlayValues[79] = d79
		ps130.OverlayValues[80] = d80
		ps130.OverlayValues[81] = d81
		ps130.OverlayValues[82] = d82
		ps130.OverlayValues[83] = d83
		snap131 := d0
		snap132 := d1
		snap133 := d2
		snap134 := d3
		snap135 := d4
		snap136 := d21
		snap137 := d22
		snap138 := d23
		snap139 := d24
		snap140 := d25
		snap141 := d52
		snap142 := d53
		snap143 := d54
		snap144 := d55
		snap145 := d56
		snap146 := d57
		snap147 := d58
		snap148 := d59
		snap149 := d60
		snap150 := d61
		snap151 := d62
		snap152 := d63
		snap153 := d64
		snap154 := d65
		snap155 := d66
		snap156 := d67
		snap157 := d68
		snap158 := d69
		snap159 := d70
		snap160 := d71
		snap161 := d72
		snap162 := d73
		snap163 := d74
		snap164 := d75
		snap165 := d76
		snap166 := d77
		snap167 := d78
		snap168 := d79
		snap169 := d80
		snap170 := d81
		snap171 := d82
		snap172 := d83
		alloc173 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps130)
		}
		ctx.RestoreAllocState(alloc173)
		d0 = snap131
		d1 = snap132
		d2 = snap133
		d3 = snap134
		d4 = snap135
		d21 = snap136
		d22 = snap137
		d23 = snap138
		d24 = snap139
		d25 = snap140
		d52 = snap141
		d53 = snap142
		d54 = snap143
		d55 = snap144
		d56 = snap145
		d57 = snap146
		d58 = snap147
		d59 = snap148
		d60 = snap149
		d61 = snap150
		d62 = snap151
		d63 = snap152
		d64 = snap153
		d65 = snap154
		d66 = snap155
		d67 = snap156
		d68 = snap157
		d69 = snap158
		d70 = snap159
		d71 = snap160
		d72 = snap161
		d73 = snap162
		d74 = snap163
		d75 = snap164
		d76 = snap165
		d77 = snap166
		d78 = snap167
		d79 = snap168
		d80 = snap169
		d81 = snap170
		d82 = snap171
		d83 = snap172
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps129)
		}
		return result
		ctx.FreeDesc(&d82)
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
		d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("prefix index out of range")}
		ctx.EnsureDesc(&d174)
		ctx.EnsureDesc(&d174)
		if d174.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d174.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d174)
			} else if d174.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d174)
			} else if d174.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d174)
			} else if d174.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d174.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d174 = tmpPair
		} else if d174.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d174.Type, Reg: ctx.AllocRegExcept(d174.Reg), Reg2: ctx.AllocRegExcept(d174.Reg)}
			switch d174.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d174)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d174)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d174)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d174)
			d174 = tmpPair
		}
		if d174.Loc != scm.LocRegPair && d174.Loc != scm.LocStackPair && d174.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d174})
		ctx.FreeDesc(&d174)
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
		if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
			d174 = ps.OverlayValues[174]
		}
		ctx.ReclaimUntrackedRegs()
		var d175 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r39 := ctx.AllocReg()
			r40 := ctx.AllocRegExcept(r39)
			r41 := ctx.AllocRegExcept(r39, r40)
			ctx.EmitMovRegMem64(r39, fieldAddr)
			ctx.EmitMovRegMem64(r40, fieldAddr+8)
			ctx.EmitMovRegMem64(r41, fieldAddr+16)
			d175 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r39, Reg2: r40, Reg3: r41}
			ctx.BindReg(r39, &d175)
			ctx.BindReg(r40, &d175)
			ctx.BindReg(r41, &d175)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r42 := ctx.AllocReg()
			r43 := ctx.AllocRegExcept(r42)
			r44 := ctx.AllocRegExcept(r42, r43)
			ctx.EmitMovRegMem(r42, thisptr.Reg, off)
			ctx.EmitMovRegMem(r43, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off+16)
			d175 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r42, Reg2: r43, Reg3: r44}
			ctx.BindReg(r42, &d175)
			ctx.BindReg(r43, &d175)
			ctx.BindReg(r44, &d175)
		}
		ctx.EnsureDesc(&d78)
		d177 = ctx.EmitSliceElementAddress(&d175, &d78, 16)
		ctx.EnsureDesc(&d177)
		r45 := ctx.AllocRegExcept(d177.Reg)
		ctx.EmitMovRegMem(r45, d177.Reg, 8)
		ctx.EmitMovRegMem(d177.Reg, d177.Reg, 0)
		d176 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d177.Reg, Reg2: r45}
		ctx.BindReg(d177.Reg, &d176)
		ctx.BindReg(r45, &d176)
		d179 = d1
		ctx.SyncDesc(&d179)
		if d179.Loc == scm.LocMem {
			tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d179.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d179.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d179 = tmpScalar
		}
		d179 = scm.JITPrepareScmerGoArg(ctx, d179)
		if d179.Loc != scm.LocRegPair && d179.Loc != scm.LocStackPair && d179.Loc != scm.LocInputPair {
			panic("jit: scm.Scmer.String receiver not materialized as pair")
		}
		d178 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d179}, 2)
		ctx.EnsureDesc(&d176)
		ctx.EnsureDesc(&d178)
		d180 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d176, d178}, 2)
		ctx.FreeDesc(&d176)
		ctx.EnsureDesc(&d180)
		d181 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d181)
		ctx.BindReg(r1, &d181)
		d182 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d180}, 2)
		ctx.EmitMovPairToResult(&d182, &d181)
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
		if len(ps.OverlayValues) > 174 && ps.OverlayValues[174].Loc != scm.LocNone {
			d174 = ps.OverlayValues[174]
		}
		if len(ps.OverlayValues) > 175 && ps.OverlayValues[175].Loc != scm.LocNone {
			d175 = ps.OverlayValues[175]
		}
		if len(ps.OverlayValues) > 176 && ps.OverlayValues[176].Loc != scm.LocNone {
			d176 = ps.OverlayValues[176]
		}
		if len(ps.OverlayValues) > 177 && ps.OverlayValues[177].Loc != scm.LocNone {
			d177 = ps.OverlayValues[177]
		}
		if len(ps.OverlayValues) > 178 && ps.OverlayValues[178].Loc != scm.LocNone {
			d178 = ps.OverlayValues[178]
		}
		if len(ps.OverlayValues) > 179 && ps.OverlayValues[179].Loc != scm.LocNone {
			d179 = ps.OverlayValues[179]
		}
		if len(ps.OverlayValues) > 180 && ps.OverlayValues[180].Loc != scm.LocNone {
			d180 = ps.OverlayValues[180]
		}
		if len(ps.OverlayValues) > 181 && ps.OverlayValues[181].Loc != scm.LocNone {
			d181 = ps.OverlayValues[181]
		}
		if len(ps.OverlayValues) > 182 && ps.OverlayValues[182].Loc != scm.LocNone {
			d182 = ps.OverlayValues[182]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d78)
		var d183 scm.JITValueDesc
		if d78.Loc == scm.LocImm {
			d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d78.Imm.Int() < 0)}
		} else {
			r46 := ctx.AllocRegExcept(d78.Reg)
			ctx.EmitCmpRegImm32(d78.Reg, 0)
			ctx.EmitSetcc(r46, scm.CondSignedLess)
			d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			ctx.BindReg(r46, &d183)
		}
		d184 = d183
		ctx.EnsureDesc(&d184)
		if d184.Loc != scm.LocImm && d184.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d184.Loc == scm.LocImm {
			if d184.Imm.Bool() {
				if ps.General {
				}
				ps185 := scm.PhiState{General: ps.General}
				ps185.OverlayValues = make([]scm.JITValueDesc, 185)
				ps185.OverlayValues[0] = d0
				ps185.OverlayValues[1] = d1
				ps185.OverlayValues[2] = d2
				ps185.OverlayValues[3] = d3
				ps185.OverlayValues[4] = d4
				ps185.OverlayValues[21] = d21
				ps185.OverlayValues[22] = d22
				ps185.OverlayValues[23] = d23
				ps185.OverlayValues[24] = d24
				ps185.OverlayValues[25] = d25
				ps185.OverlayValues[52] = d52
				ps185.OverlayValues[53] = d53
				ps185.OverlayValues[54] = d54
				ps185.OverlayValues[55] = d55
				ps185.OverlayValues[56] = d56
				ps185.OverlayValues[57] = d57
				ps185.OverlayValues[58] = d58
				ps185.OverlayValues[59] = d59
				ps185.OverlayValues[60] = d60
				ps185.OverlayValues[61] = d61
				ps185.OverlayValues[62] = d62
				ps185.OverlayValues[63] = d63
				ps185.OverlayValues[64] = d64
				ps185.OverlayValues[65] = d65
				ps185.OverlayValues[66] = d66
				ps185.OverlayValues[67] = d67
				ps185.OverlayValues[68] = d68
				ps185.OverlayValues[69] = d69
				ps185.OverlayValues[70] = d70
				ps185.OverlayValues[71] = d71
				ps185.OverlayValues[72] = d72
				ps185.OverlayValues[73] = d73
				ps185.OverlayValues[74] = d74
				ps185.OverlayValues[75] = d75
				ps185.OverlayValues[76] = d76
				ps185.OverlayValues[77] = d77
				ps185.OverlayValues[78] = d78
				ps185.OverlayValues[79] = d79
				ps185.OverlayValues[80] = d80
				ps185.OverlayValues[81] = d81
				ps185.OverlayValues[82] = d82
				ps185.OverlayValues[83] = d83
				ps185.OverlayValues[174] = d174
				ps185.OverlayValues[175] = d175
				ps185.OverlayValues[176] = d176
				ps185.OverlayValues[177] = d177
				ps185.OverlayValues[178] = d178
				ps185.OverlayValues[179] = d179
				ps185.OverlayValues[180] = d180
				ps185.OverlayValues[181] = d181
				ps185.OverlayValues[182] = d182
				ps185.OverlayValues[183] = d183
				ps185.OverlayValues[184] = d184
				return bbs[5].RenderPS(ps185)
			}
			if ps.General {
			}
			ps186 := scm.PhiState{General: ps.General}
			ps186.OverlayValues = make([]scm.JITValueDesc, 185)
			ps186.OverlayValues[0] = d0
			ps186.OverlayValues[1] = d1
			ps186.OverlayValues[2] = d2
			ps186.OverlayValues[3] = d3
			ps186.OverlayValues[4] = d4
			ps186.OverlayValues[21] = d21
			ps186.OverlayValues[22] = d22
			ps186.OverlayValues[23] = d23
			ps186.OverlayValues[24] = d24
			ps186.OverlayValues[25] = d25
			ps186.OverlayValues[52] = d52
			ps186.OverlayValues[53] = d53
			ps186.OverlayValues[54] = d54
			ps186.OverlayValues[55] = d55
			ps186.OverlayValues[56] = d56
			ps186.OverlayValues[57] = d57
			ps186.OverlayValues[58] = d58
			ps186.OverlayValues[59] = d59
			ps186.OverlayValues[60] = d60
			ps186.OverlayValues[61] = d61
			ps186.OverlayValues[62] = d62
			ps186.OverlayValues[63] = d63
			ps186.OverlayValues[64] = d64
			ps186.OverlayValues[65] = d65
			ps186.OverlayValues[66] = d66
			ps186.OverlayValues[67] = d67
			ps186.OverlayValues[68] = d68
			ps186.OverlayValues[69] = d69
			ps186.OverlayValues[70] = d70
			ps186.OverlayValues[71] = d71
			ps186.OverlayValues[72] = d72
			ps186.OverlayValues[73] = d73
			ps186.OverlayValues[74] = d74
			ps186.OverlayValues[75] = d75
			ps186.OverlayValues[76] = d76
			ps186.OverlayValues[77] = d77
			ps186.OverlayValues[78] = d78
			ps186.OverlayValues[79] = d79
			ps186.OverlayValues[80] = d80
			ps186.OverlayValues[81] = d81
			ps186.OverlayValues[82] = d82
			ps186.OverlayValues[83] = d83
			ps186.OverlayValues[174] = d174
			ps186.OverlayValues[175] = d175
			ps186.OverlayValues[176] = d176
			ps186.OverlayValues[177] = d177
			ps186.OverlayValues[178] = d178
			ps186.OverlayValues[179] = d179
			ps186.OverlayValues[180] = d180
			ps186.OverlayValues[181] = d181
			ps186.OverlayValues[182] = d182
			ps186.OverlayValues[183] = d183
			ps186.OverlayValues[184] = d184
			return bbs[6].RenderPS(ps186)
		}
		if !ps.General {
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d184.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		snap187 := d0
		snap188 := d1
		snap189 := d2
		snap190 := d3
		snap191 := d4
		snap192 := d21
		snap193 := d22
		snap194 := d23
		snap195 := d24
		snap196 := d25
		snap197 := d52
		snap198 := d53
		snap199 := d54
		snap200 := d55
		snap201 := d56
		snap202 := d57
		snap203 := d58
		snap204 := d59
		snap205 := d60
		snap206 := d61
		snap207 := d62
		snap208 := d63
		snap209 := d64
		snap210 := d65
		snap211 := d66
		snap212 := d67
		snap213 := d68
		snap214 := d69
		snap215 := d70
		snap216 := d71
		snap217 := d72
		snap218 := d73
		snap219 := d74
		snap220 := d75
		snap221 := d76
		snap222 := d77
		snap223 := d78
		snap224 := d79
		snap225 := d80
		snap226 := d81
		snap227 := d82
		snap228 := d83
		snap229 := d174
		snap230 := d175
		snap231 := d176
		snap232 := d177
		snap233 := d178
		snap234 := d179
		snap235 := d180
		snap236 := d181
		snap237 := d182
		snap238 := d183
		snap239 := d184
		alloc240 := ctx.SnapshotAllocState()
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl6)
		ctx.RestoreAllocState(alloc240)
		d0 = snap187
		d1 = snap188
		d2 = snap189
		d3 = snap190
		d4 = snap191
		d21 = snap192
		d22 = snap193
		d23 = snap194
		d24 = snap195
		d25 = snap196
		d52 = snap197
		d53 = snap198
		d54 = snap199
		d55 = snap200
		d56 = snap201
		d57 = snap202
		d58 = snap203
		d59 = snap204
		d60 = snap205
		d61 = snap206
		d62 = snap207
		d63 = snap208
		d64 = snap209
		d65 = snap210
		d66 = snap211
		d67 = snap212
		d68 = snap213
		d69 = snap214
		d70 = snap215
		d71 = snap216
		d72 = snap217
		d73 = snap218
		d74 = snap219
		d75 = snap220
		d76 = snap221
		d77 = snap222
		d78 = snap223
		d79 = snap224
		d80 = snap225
		d81 = snap226
		d82 = snap227
		d83 = snap228
		d174 = snap229
		d175 = snap230
		d176 = snap231
		d177 = snap232
		d178 = snap233
		d179 = snap234
		d180 = snap235
		d181 = snap236
		d182 = snap237
		d183 = snap238
		d184 = snap239
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl7)
		ctx.RestoreAllocState(alloc240)
		d0 = snap187
		d1 = snap188
		d2 = snap189
		d3 = snap190
		d4 = snap191
		d21 = snap192
		d22 = snap193
		d23 = snap194
		d24 = snap195
		d25 = snap196
		d52 = snap197
		d53 = snap198
		d54 = snap199
		d55 = snap200
		d56 = snap201
		d57 = snap202
		d58 = snap203
		d59 = snap204
		d60 = snap205
		d61 = snap206
		d62 = snap207
		d63 = snap208
		d64 = snap209
		d65 = snap210
		d66 = snap211
		d67 = snap212
		d68 = snap213
		d69 = snap214
		d70 = snap215
		d71 = snap216
		d72 = snap217
		d73 = snap218
		d74 = snap219
		d75 = snap220
		d76 = snap221
		d77 = snap222
		d78 = snap223
		d79 = snap224
		d80 = snap225
		d81 = snap226
		d82 = snap227
		d83 = snap228
		d174 = snap229
		d175 = snap230
		d176 = snap231
		d177 = snap232
		d178 = snap233
		d179 = snap234
		d180 = snap235
		d181 = snap236
		d182 = snap237
		d183 = snap238
		d184 = snap239
		ps241 := scm.PhiState{General: true}
		ps241.OverlayValues = make([]scm.JITValueDesc, 185)
		ps241.OverlayValues[0] = d0
		ps241.OverlayValues[1] = d1
		ps241.OverlayValues[2] = d2
		ps241.OverlayValues[3] = d3
		ps241.OverlayValues[4] = d4
		ps241.OverlayValues[21] = d21
		ps241.OverlayValues[22] = d22
		ps241.OverlayValues[23] = d23
		ps241.OverlayValues[24] = d24
		ps241.OverlayValues[25] = d25
		ps241.OverlayValues[52] = d52
		ps241.OverlayValues[53] = d53
		ps241.OverlayValues[54] = d54
		ps241.OverlayValues[55] = d55
		ps241.OverlayValues[56] = d56
		ps241.OverlayValues[57] = d57
		ps241.OverlayValues[58] = d58
		ps241.OverlayValues[59] = d59
		ps241.OverlayValues[60] = d60
		ps241.OverlayValues[61] = d61
		ps241.OverlayValues[62] = d62
		ps241.OverlayValues[63] = d63
		ps241.OverlayValues[64] = d64
		ps241.OverlayValues[65] = d65
		ps241.OverlayValues[66] = d66
		ps241.OverlayValues[67] = d67
		ps241.OverlayValues[68] = d68
		ps241.OverlayValues[69] = d69
		ps241.OverlayValues[70] = d70
		ps241.OverlayValues[71] = d71
		ps241.OverlayValues[72] = d72
		ps241.OverlayValues[73] = d73
		ps241.OverlayValues[74] = d74
		ps241.OverlayValues[75] = d75
		ps241.OverlayValues[76] = d76
		ps241.OverlayValues[77] = d77
		ps241.OverlayValues[78] = d78
		ps241.OverlayValues[79] = d79
		ps241.OverlayValues[80] = d80
		ps241.OverlayValues[81] = d81
		ps241.OverlayValues[82] = d82
		ps241.OverlayValues[83] = d83
		ps241.OverlayValues[174] = d174
		ps241.OverlayValues[175] = d175
		ps241.OverlayValues[176] = d176
		ps241.OverlayValues[177] = d177
		ps241.OverlayValues[178] = d178
		ps241.OverlayValues[179] = d179
		ps241.OverlayValues[180] = d180
		ps241.OverlayValues[181] = d181
		ps241.OverlayValues[182] = d182
		ps241.OverlayValues[183] = d183
		ps241.OverlayValues[184] = d184
		ps242 := scm.PhiState{General: true}
		ps242.OverlayValues = make([]scm.JITValueDesc, 185)
		ps242.OverlayValues[0] = d0
		ps242.OverlayValues[1] = d1
		ps242.OverlayValues[2] = d2
		ps242.OverlayValues[3] = d3
		ps242.OverlayValues[4] = d4
		ps242.OverlayValues[21] = d21
		ps242.OverlayValues[22] = d22
		ps242.OverlayValues[23] = d23
		ps242.OverlayValues[24] = d24
		ps242.OverlayValues[25] = d25
		ps242.OverlayValues[52] = d52
		ps242.OverlayValues[53] = d53
		ps242.OverlayValues[54] = d54
		ps242.OverlayValues[55] = d55
		ps242.OverlayValues[56] = d56
		ps242.OverlayValues[57] = d57
		ps242.OverlayValues[58] = d58
		ps242.OverlayValues[59] = d59
		ps242.OverlayValues[60] = d60
		ps242.OverlayValues[61] = d61
		ps242.OverlayValues[62] = d62
		ps242.OverlayValues[63] = d63
		ps242.OverlayValues[64] = d64
		ps242.OverlayValues[65] = d65
		ps242.OverlayValues[66] = d66
		ps242.OverlayValues[67] = d67
		ps242.OverlayValues[68] = d68
		ps242.OverlayValues[69] = d69
		ps242.OverlayValues[70] = d70
		ps242.OverlayValues[71] = d71
		ps242.OverlayValues[72] = d72
		ps242.OverlayValues[73] = d73
		ps242.OverlayValues[74] = d74
		ps242.OverlayValues[75] = d75
		ps242.OverlayValues[76] = d76
		ps242.OverlayValues[77] = d77
		ps242.OverlayValues[78] = d78
		ps242.OverlayValues[79] = d79
		ps242.OverlayValues[80] = d80
		ps242.OverlayValues[81] = d81
		ps242.OverlayValues[82] = d82
		ps242.OverlayValues[83] = d83
		ps242.OverlayValues[174] = d174
		ps242.OverlayValues[175] = d175
		ps242.OverlayValues[176] = d176
		ps242.OverlayValues[177] = d177
		ps242.OverlayValues[178] = d178
		ps242.OverlayValues[179] = d179
		ps242.OverlayValues[180] = d180
		ps242.OverlayValues[181] = d181
		ps242.OverlayValues[182] = d182
		ps242.OverlayValues[183] = d183
		ps242.OverlayValues[184] = d184
		snap243 := d0
		snap244 := d1
		snap245 := d2
		snap246 := d3
		snap247 := d4
		snap248 := d21
		snap249 := d22
		snap250 := d23
		snap251 := d24
		snap252 := d25
		snap253 := d52
		snap254 := d53
		snap255 := d54
		snap256 := d55
		snap257 := d56
		snap258 := d57
		snap259 := d58
		snap260 := d59
		snap261 := d60
		snap262 := d61
		snap263 := d62
		snap264 := d63
		snap265 := d64
		snap266 := d65
		snap267 := d66
		snap268 := d67
		snap269 := d68
		snap270 := d69
		snap271 := d70
		snap272 := d71
		snap273 := d72
		snap274 := d73
		snap275 := d74
		snap276 := d75
		snap277 := d76
		snap278 := d77
		snap279 := d78
		snap280 := d79
		snap281 := d80
		snap282 := d81
		snap283 := d82
		snap284 := d83
		snap285 := d174
		snap286 := d175
		snap287 := d176
		snap288 := d177
		snap289 := d178
		snap290 := d179
		snap291 := d180
		snap292 := d181
		snap293 := d182
		snap294 := d183
		snap295 := d184
		alloc296 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps242)
		}
		ctx.RestoreAllocState(alloc296)
		d0 = snap243
		d1 = snap244
		d2 = snap245
		d3 = snap246
		d4 = snap247
		d21 = snap248
		d22 = snap249
		d23 = snap250
		d24 = snap251
		d25 = snap252
		d52 = snap253
		d53 = snap254
		d54 = snap255
		d55 = snap256
		d56 = snap257
		d57 = snap258
		d58 = snap259
		d59 = snap260
		d60 = snap261
		d61 = snap262
		d62 = snap263
		d63 = snap264
		d64 = snap265
		d65 = snap266
		d66 = snap267
		d67 = snap268
		d68 = snap269
		d69 = snap270
		d70 = snap271
		d71 = snap272
		d72 = snap273
		d73 = snap274
		d74 = snap275
		d75 = snap276
		d76 = snap277
		d77 = snap278
		d78 = snap279
		d79 = snap280
		d80 = snap281
		d81 = snap282
		d82 = snap283
		d83 = snap284
		d174 = snap285
		d175 = snap286
		d176 = snap287
		d177 = snap288
		d178 = snap289
		d179 = snap290
		d180 = snap291
		d181 = snap292
		d182 = snap293
		d183 = snap294
		d184 = snap295
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps241)
		}
		return result
		ctx.FreeDesc(&d183)
		return result
	}
	ps297 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps297)
	ctx.MarkLabel(lbl0)
	d298 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d298)
	ctx.BindReg(r1, &d298)
	ctx.EmitMovPairToResult(&d298, &result)
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
