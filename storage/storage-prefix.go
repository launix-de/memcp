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
	var d35 scm.JITValueDesc
	_ = d35
	var d36 scm.JITValueDesc
	_ = d36
	var d37 scm.JITValueDesc
	_ = d37
	var d38 scm.JITValueDesc
	_ = d38
	var d39 scm.JITValueDesc
	_ = d39
	var d40 scm.JITValueDesc
	_ = d40
	var d41 scm.JITValueDesc
	_ = d41
	var d42 scm.JITValueDesc
	_ = d42
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
	var d51 scm.JITValueDesc
	_ = d51
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
	var d114 scm.JITValueDesc
	_ = d114
	var d115 scm.JITValueDesc
	_ = d115
	var d116 scm.JITValueDesc
	_ = d116
	var d117 scm.JITValueDesc
	_ = d117
	var d118 scm.JITValueDesc
	_ = d118
	var d119 scm.JITValueDesc
	_ = d119
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
		ctx.MarkLabel(lbl9)
		ctx.EmitJmp(lbl2)
		ctx.MarkLabel(lbl10)
		ctx.EmitJmp(lbl3)
		ps7 := scm.PhiState{General: true}
		ps7.OverlayValues = make([]scm.JITValueDesc, 5)
		ps7.OverlayValues[0] = d0
		ps7.OverlayValues[1] = d1
		ps7.OverlayValues[2] = d2
		ps7.OverlayValues[3] = d3
		ps7.OverlayValues[4] = d4
		ps8 := scm.PhiState{General: true}
		ps8.OverlayValues = make([]scm.JITValueDesc, 5)
		ps8.OverlayValues[0] = d0
		ps8.OverlayValues[1] = d1
		ps8.OverlayValues[2] = d2
		ps8.OverlayValues[3] = d3
		ps8.OverlayValues[4] = d4
		snap9 := d0
		snap10 := d1
		snap11 := d2
		snap12 := d3
		snap13 := d4
		alloc14 := ctx.SnapshotAllocState()
		if !bbs[2].Rendered {
			bbs[2].RenderPS(ps8)
		}
		ctx.RestoreAllocState(alloc14)
		d0 = snap9
		d1 = snap10
		d2 = snap11
		d3 = snap12
		d4 = snap13
		if !bbs[1].Rendered {
			return bbs[1].RenderPS(ps7)
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
		d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d16 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d16)
		ctx.BindReg(r1, &d16)
		ctx.EnsureDesc(&d15)
		if d15.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d15, &d16)
		} else {
			switch d15.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d16, d15)
			case scm.TagInt:
				ctx.EmitMakeInt(d16, d15)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d16, d15)
			case scm.TagNil:
				ctx.EmitMakeNil(d16)
			default:
				ctx.EmitMovPairToResult(&d15, &d16)
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
		if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
			d15 = ps.OverlayValues[15]
		}
		if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
			d16 = ps.OverlayValues[16]
		}
		ctx.ReclaimUntrackedRegs()
		d18 = d1
		d18.ID = 0
		d17 = ctx.EmitIsStringBorrowed(&d18, scm.JITValueDesc{Loc: scm.LocAny})
		d19 = d17
		ctx.EnsureDesc(&d19)
		if d19.Loc != scm.LocImm && d19.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d19.Loc == scm.LocImm {
			if d19.Imm.Bool() {
				if ps.General {
				}
				ps20 := scm.PhiState{General: ps.General}
				ps20.OverlayValues = make([]scm.JITValueDesc, 20)
				ps20.OverlayValues[0] = d0
				ps20.OverlayValues[1] = d1
				ps20.OverlayValues[2] = d2
				ps20.OverlayValues[3] = d3
				ps20.OverlayValues[4] = d4
				ps20.OverlayValues[15] = d15
				ps20.OverlayValues[16] = d16
				ps20.OverlayValues[17] = d17
				ps20.OverlayValues[18] = d18
				ps20.OverlayValues[19] = d19
				return bbs[4].RenderPS(ps20)
			}
			if ps.General {
			}
			ps21 := scm.PhiState{General: ps.General}
			ps21.OverlayValues = make([]scm.JITValueDesc, 20)
			ps21.OverlayValues[0] = d0
			ps21.OverlayValues[1] = d1
			ps21.OverlayValues[2] = d2
			ps21.OverlayValues[3] = d3
			ps21.OverlayValues[4] = d4
			ps21.OverlayValues[15] = d15
			ps21.OverlayValues[16] = d16
			ps21.OverlayValues[17] = d17
			ps21.OverlayValues[18] = d18
			ps21.OverlayValues[19] = d19
			return bbs[3].RenderPS(ps21)
		}
		if !ps.General {
			ps.General = true
			return bbs[2].RenderPS(ps)
		}
		lbl11 := ctx.ReserveLabel()
		lbl12 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d19.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl11)
		ctx.EmitJmp(lbl12)
		ctx.MarkLabel(lbl11)
		ctx.EmitJmp(lbl5)
		ctx.MarkLabel(lbl12)
		ctx.EmitJmp(lbl4)
		ps22 := scm.PhiState{General: true}
		ps22.OverlayValues = make([]scm.JITValueDesc, 20)
		ps22.OverlayValues[0] = d0
		ps22.OverlayValues[1] = d1
		ps22.OverlayValues[2] = d2
		ps22.OverlayValues[3] = d3
		ps22.OverlayValues[4] = d4
		ps22.OverlayValues[15] = d15
		ps22.OverlayValues[16] = d16
		ps22.OverlayValues[17] = d17
		ps22.OverlayValues[18] = d18
		ps22.OverlayValues[19] = d19
		ps23 := scm.PhiState{General: true}
		ps23.OverlayValues = make([]scm.JITValueDesc, 20)
		ps23.OverlayValues[0] = d0
		ps23.OverlayValues[1] = d1
		ps23.OverlayValues[2] = d2
		ps23.OverlayValues[3] = d3
		ps23.OverlayValues[4] = d4
		ps23.OverlayValues[15] = d15
		ps23.OverlayValues[16] = d16
		ps23.OverlayValues[17] = d17
		ps23.OverlayValues[18] = d18
		ps23.OverlayValues[19] = d19
		snap24 := d0
		snap25 := d1
		snap26 := d2
		snap27 := d3
		snap28 := d4
		snap29 := d15
		snap30 := d16
		snap31 := d17
		snap32 := d18
		snap33 := d19
		alloc34 := ctx.SnapshotAllocState()
		if !bbs[3].Rendered {
			bbs[3].RenderPS(ps23)
		}
		ctx.RestoreAllocState(alloc34)
		d0 = snap24
		d1 = snap25
		d2 = snap26
		d3 = snap27
		d4 = snap28
		d15 = snap29
		d16 = snap30
		d17 = snap31
		d18 = snap32
		d19 = snap33
		if !bbs[4].Rendered {
			return bbs[4].RenderPS(ps22)
		}
		return result
		ctx.FreeDesc(&d17)
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
		ctx.ReclaimUntrackedRegs()
		d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("invalid value in prefix storage")}
		ctx.EnsureDesc(&d35)
		ctx.EnsureDesc(&d35)
		if d35.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d35.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d35)
			} else if d35.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d35)
			} else if d35.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d35)
			} else if d35.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d35.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d35 = tmpPair
		} else if d35.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d35.Type, Reg: ctx.AllocRegExcept(d35.Reg), Reg2: ctx.AllocRegExcept(d35.Reg)}
			switch d35.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d35)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d35)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d35)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d35)
			d35 = tmpPair
		}
		if d35.Loc != scm.LocRegPair && d35.Loc != scm.LocStackPair && d35.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d35})
		ctx.FreeDesc(&d35)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&idxInt)
		d36 = idxInt
		_ = d36
		ctx.StabilizeDescForControlFlow(&d36)
		bbpos_1_0 := int32(-1)
		_ = bbpos_1_0
		lbl13 := ctx.ReserveLabel()
		_ = lbl13
		bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl13)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d36)
		var d37 scm.JITValueDesc
		if d36.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d36.Imm.Int()))))}
		} else {
			r2 := ctx.AllocReg()
			ctx.EmitMovRegReg(r2, d36.Reg)
			ctx.EmitShlRegImm8(r2, 32)
			ctx.EmitShrRegImm8(r2, 32)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d37)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d38 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r3 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r3, thisptr.Reg, off)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			ctx.BindReg(r3, &d38)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.EnsureDesc(&d38)
		var d39 scm.JITValueDesc
		if d38.Loc == scm.LocImm {
			d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d38.Imm.Int()))))}
		} else {
			r4 := ctx.AllocReg()
			ctx.EmitMovRegReg(r4, d38.Reg)
			ctx.EmitShlRegImm8(r4, 56)
			ctx.EmitShrRegImm8(r4, 56)
			d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d39)
		}
		ctx.FreeDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d39)
		ctx.EnsureDescsTogether(&d37, &d39)
		var d40 scm.JITValueDesc
		if d37.Loc == scm.LocImm && d39.Loc == scm.LocImm {
			d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() * d39.Imm.Int())}
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
			ctx.EmitImulInt64(scratch, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d40)
		} else if d39.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(scratch, d37.Reg)
			if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
				ctx.EmitImulRegImm32(scratch, int32(d39.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.EmitImulInt64(scratch, scm.RegR11)
			}
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d40)
		} else {
			r5 := ctx.AllocRegExcept(d37.Reg, d39.Reg)
			ctx.EmitMovRegReg(r5, d37.Reg)
			ctx.EmitImulInt64(r5, d39.Reg)
			d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			ctx.BindReg(r5, &d40)
		}
		if d40.Loc == scm.LocReg && d37.Loc == scm.LocReg && d40.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d37)
		ctx.FreeDesc(&d39)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		var d41 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() / 64)}
		} else {
			r6 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(r6, d40.Reg)
			ctx.EmitShrRegImm8(r6, 6)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
			ctx.BindReg(r6, &d41)
		}
		if d41.Loc == scm.LocReg && d40.Loc == scm.LocReg && d41.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d40)
		var d42 scm.JITValueDesc
		if d40.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() % 64)}
		} else {
			r7 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(r7, d40.Reg)
			ctx.EmitAndRegImm32(r7, 63)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			ctx.BindReg(r7, &d42)
		}
		if d42.Loc == scm.LocReg && d40.Loc == scm.LocReg && d42.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d40)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d43 scm.JITValueDesc
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
		d43 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r8, Reg2: r9, Reg3: r10}
		ctx.BindReg(r8, &d43)
		ctx.BindReg(r9, &d43)
		ctx.BindReg(r10, &d43)
		ctx.BindReg(r8, &d43)
		ctx.BindReg(r9, &d43)
		ctx.BindReg(r10, &d43)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		d45 = ctx.EmitSliceElementAddress(&d43, &d41, 8)
		ctx.EnsureDesc(&d45)
		ctx.EmitMovRegMem(d45.Reg, d45.Reg, 0)
		d44 = d45
		d44.Type = scm.TagInt
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d44)
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d44, &d42)
		var d46 scm.JITValueDesc
		if d44.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d44.Imm.Int()) << uint64(d42.Imm.Int())))}
		} else if d42.Loc == scm.LocImm {
			r11 := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(r11, d44.Reg)
			ctx.EmitShlRegImm8(r11, uint8(d42.Imm.Int()))
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d46)
		} else {
			{
				shiftSrc := d44.Reg
				r12 := ctx.AllocRegExcept(d44.Reg, d42.Reg)
				ctx.EmitMovRegReg(r12, d44.Reg)
				shiftSrc = r12
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d42.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d42.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d42.Reg)
				}
				ctx.EmitShlRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d46)
			}
		}
		if d46.Loc == scm.LocReg && d44.Loc == scm.LocReg && d46.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d44)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d41)
		ctx.EnsureDesc(&d41)
		var d47 scm.JITValueDesc
		if d41.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d41.Reg)
			ctx.EmitMovRegReg(scratch, d41.Reg)
			ctx.EmitAddRegImm32(scratch, int32(1))
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d47)
		}
		if d47.Loc == scm.LocReg && d41.Loc == scm.LocReg && d47.Reg == d41.Reg {
			ctx.TransferReg(d41.Reg)
			d41.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d49 = ctx.EmitSliceElementAddress(&d43, &d47, 8)
		ctx.EnsureDesc(&d49)
		ctx.EmitMovRegMem(d49.Reg, d49.Reg, 0)
		d48 = d49
		d48.Type = scm.TagInt
		ctx.FreeDesc(&d47)
		ctx.ReclaimUntrackedRegs()
		d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d42)
		ctx.EnsureDescsTogether(&d50, &d42)
		var d51 scm.JITValueDesc
		if d50.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() - d42.Imm.Int())}
		} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
			r13 := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(r13, d50.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			ctx.BindReg(r13, &d51)
		} else if d50.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
			ctx.EmitSubInt64(scratch, d42.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d51)
		} else if d42.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d50.Reg)
			ctx.EmitMovRegReg(scratch, d50.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d51)
		} else {
			r14 := ctx.AllocRegExcept(d50.Reg, d42.Reg)
			ctx.EmitMovRegReg(r14, d50.Reg)
			ctx.EmitSubInt64(r14, d42.Reg)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d51)
		}
		if d51.Loc == scm.LocReg && d50.Loc == scm.LocReg && d51.Reg == d50.Reg {
			ctx.TransferReg(d50.Reg)
			d50.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d48)
		ctx.EnsureDesc(&d51)
		ctx.EnsureDescsTogether(&d48, &d51)
		var d52 scm.JITValueDesc
		if d48.Loc == scm.LocImm && d51.Loc == scm.LocImm {
			d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) >> uint64(d51.Imm.Int())))}
		} else if d51.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(r15, d48.Reg)
			ctx.EmitShrRegImm8(r15, uint8(d51.Imm.Int()))
			d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d52)
		} else {
			{
				shiftSrc := d48.Reg
				r16 := ctx.AllocRegExcept(d48.Reg, d51.Reg)
				ctx.EmitMovRegReg(r16, d48.Reg)
				shiftSrc = r16
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d51.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d51.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d51.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d52)
			}
		}
		if d52.Loc == scm.LocReg && d48.Loc == scm.LocReg && d52.Reg == d48.Reg {
			ctx.TransferReg(d48.Reg)
			d48.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d48)
		ctx.FreeDesc(&d51)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d52)
		var d53 scm.JITValueDesc
		if d46.Loc == scm.LocImm && d52.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() | d52.Imm.Int())}
		} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			ctx.BindReg(d52.Reg, &d53)
		} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
			r17 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r17, d46.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d53)
		} else if d46.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d52.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d46.Imm.Int()))
			ctx.EmitOrInt64(scratch, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d53)
		} else if d52.Loc == scm.LocImm {
			r18 := ctx.AllocRegExcept(d46.Reg)
			ctx.EmitMovRegReg(r18, d46.Reg)
			if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r18, int32(d52.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.EmitOrInt64(r18, scm.RegR11)
			}
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d53)
		} else {
			r19 := ctx.AllocRegExcept(d46.Reg, d52.Reg)
			ctx.EmitMovRegReg(r19, d46.Reg)
			ctx.EmitOrInt64(r19, d52.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d53)
		}
		if d53.Loc == scm.LocReg && d46.Loc == scm.LocReg && d53.Reg == d46.Reg {
			ctx.TransferReg(d46.Reg)
			d46.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d46)
		ctx.FreeDesc(&d52)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		var d54 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r20 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r20, thisptr.Reg, off)
			d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d54)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d54)
		ctx.EnsureDesc(&d54)
		var d55 scm.JITValueDesc
		if d54.Loc == scm.LocImm {
			d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d54.Imm.Int()))))}
		} else {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegReg(r21, d54.Reg)
			ctx.EmitShlRegImm8(r21, 56)
			ctx.EmitShrRegImm8(r21, 56)
			d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d55)
		}
		ctx.FreeDesc(&d54)
		ctx.ReclaimUntrackedRegs()
		d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d55)
		ctx.EnsureDescsTogether(&d56, &d55)
		var d57 scm.JITValueDesc
		if d56.Loc == scm.LocImm && d55.Loc == scm.LocImm {
			d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() - d55.Imm.Int())}
		} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
			r22 := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegReg(r22, d56.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			ctx.BindReg(r22, &d57)
		} else if d56.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d55.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
			ctx.EmitSubInt64(scratch, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else if d55.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d56.Reg)
			ctx.EmitMovRegReg(scratch, d56.Reg)
			if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
				ctx.EmitSubRegImm32(scratch, int32(d55.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
				ctx.EmitSubInt64(scratch, scm.RegR11)
			}
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d57)
		} else {
			r23 := ctx.AllocRegExcept(d56.Reg, d55.Reg)
			ctx.EmitMovRegReg(r23, d56.Reg)
			ctx.EmitSubInt64(r23, d55.Reg)
			d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d57)
		}
		if d57.Loc == scm.LocReg && d56.Loc == scm.LocReg && d57.Reg == d56.Reg {
			ctx.TransferReg(d56.Reg)
			d56.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d55)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d53)
		ctx.EnsureDesc(&d57)
		ctx.EnsureDescsTogether(&d53, &d57)
		var d58 scm.JITValueDesc
		if d53.Loc == scm.LocImm && d57.Loc == scm.LocImm {
			d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d57.Imm.Int())))}
		} else if d57.Loc == scm.LocImm {
			r24 := ctx.AllocRegExcept(d53.Reg)
			ctx.EmitMovRegReg(r24, d53.Reg)
			ctx.EmitShrRegImm8(r24, uint8(d57.Imm.Int()))
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d58)
		} else {
			{
				shiftSrc := d53.Reg
				r25 := ctx.AllocRegExcept(d53.Reg, d57.Reg)
				ctx.EmitMovRegReg(r25, d53.Reg)
				shiftSrc = r25
				rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d57.Reg != scm.RegRCX
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
				}
				if d57.Reg != scm.RegRCX {
					ctx.EmitMovRegReg(scm.RegRCX, d57.Reg)
				}
				ctx.EmitShrRegClGo64(shiftSrc)
				if rcxUsed {
					ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
				}
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				ctx.BindReg(shiftSrc, &d58)
			}
		}
		if d58.Loc == scm.LocReg && d53.Loc == scm.LocReg && d58.Reg == d53.Reg {
			ctx.TransferReg(d53.Reg)
			d53.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d53)
		ctx.FreeDesc(&d57)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d58)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d58)
		ctx.EnsureDesc(&d58)
		var d59 scm.JITValueDesc
		if d58.Loc == scm.LocImm {
			d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d58.Imm.Int()))))}
		} else {
			r26 := ctx.AllocReg()
			ctx.EmitMovRegReg(r26, d58.Reg)
			d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			ctx.BindReg(r26, &d59)
		}
		ctx.FreeDesc(&d58)
		var d60 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56)
			r27 := ctx.AllocReg()
			ctx.EmitMovRegMem(r27, thisptr.Reg, off)
			d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
			ctx.BindReg(r27, &d60)
		}
		ctx.EnsureDesc(&d59)
		ctx.EnsureDesc(&d60)
		ctx.EnsureDescsTogether(&d59, &d60)
		var d61 scm.JITValueDesc
		if d59.Loc == scm.LocImm && d60.Loc == scm.LocImm {
			d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + d60.Imm.Int())}
		} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
			r28 := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(r28, d59.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			ctx.BindReg(r28, &d61)
		} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d60.Reg}
			ctx.BindReg(d60.Reg, &d61)
		} else if d59.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d60.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
			ctx.EmitAddInt64(scratch, d60.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d61)
		} else if d60.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d59.Reg)
			ctx.EmitMovRegReg(scratch, d59.Reg)
			if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
				ctx.EmitAddRegImm32(scratch, int32(d60.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
				ctx.EmitAddInt64(scratch, scm.RegR11)
			}
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d61)
		} else {
			r29 := ctx.AllocRegExcept(d59.Reg, d60.Reg)
			ctx.EmitMovRegReg(r29, d59.Reg)
			ctx.EmitAddInt64(r29, d60.Reg)
			d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			ctx.BindReg(r29, &d61)
		}
		if d61.Loc == scm.LocReg && d59.Loc == scm.LocReg && d61.Reg == d59.Reg {
			ctx.TransferReg(d59.Reg)
			d59.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d61)
		ctx.FreeDesc(&d59)
		ctx.FreeDesc(&d60)
		var d62 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r30 := ctx.AllocReg()
			r31 := ctx.AllocRegExcept(r30)
			r32 := ctx.AllocRegExcept(r30, r31)
			ctx.EmitMovRegMem64(r30, fieldAddr)
			ctx.EmitMovRegMem64(r31, fieldAddr+8)
			ctx.EmitMovRegMem64(r32, fieldAddr+16)
			d62 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r30, Reg2: r31, Reg3: r32}
			ctx.BindReg(r30, &d62)
			ctx.BindReg(r31, &d62)
			ctx.BindReg(r32, &d62)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r33 := ctx.AllocReg()
			r34 := ctx.AllocRegExcept(r33)
			r35 := ctx.AllocRegExcept(r33, r34)
			ctx.EmitMovRegMem(r33, thisptr.Reg, off)
			ctx.EmitMovRegMem(r34, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r35, thisptr.Reg, off+16)
			d62 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r33, Reg2: r34, Reg3: r35}
			ctx.BindReg(r33, &d62)
			ctx.BindReg(r34, &d62)
			ctx.BindReg(r35, &d62)
		}
		var d63 scm.JITValueDesc
		if d62.SliceSizeKnown {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d62.KnownSliceLen))}
		} else if d62.Loc == scm.LocImm {
			d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d62.StackOff))}
		} else if d62.Loc == scm.LocStackTriple {
			d63 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d62.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d62)
			if d62.Loc == scm.LocRegPair || d62.Loc == scm.LocRegTriple {
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d62.Reg2, ID: 0}
			} else if d62.Loc == scm.LocReg {
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d62.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d63)
		ctx.EnsureDesc(&d61)
		ctx.EnsureDesc(&d63)
		ctx.EnsureDescsTogether(&d61, &d63)
		var d65 scm.JITValueDesc
		if d61.Loc == scm.LocImm && d63.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d61.Imm.Int() >= d63.Imm.Int())}
		} else if d63.Loc == scm.LocImm {
			r36 := ctx.AllocRegExcept(d61.Reg)
			if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d61.Reg, int32(d63.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.EmitCmpInt64(d61.Reg, scm.RegR11)
			}
			ctx.EmitSetcc(r36, scm.CondSignedGreaterOrEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			ctx.BindReg(r36, &d65)
		} else if d61.Loc == scm.LocImm {
			r37 := ctx.AllocReg()
			ctx.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
			ctx.EmitCmpInt64(scm.RegR11, d63.Reg)
			ctx.EmitSetcc(r37, scm.CondSignedGreaterOrEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
			ctx.BindReg(r37, &d65)
		} else {
			r38 := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitCmpInt64(d61.Reg, d63.Reg)
			ctx.EmitSetcc(r38, scm.CondSignedGreaterOrEqual)
			d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
			ctx.BindReg(r38, &d65)
		}
		ctx.FreeDesc(&d63)
		d66 = d65
		ctx.EnsureDesc(&d66)
		if d66.Loc != scm.LocImm && d66.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d66.Loc == scm.LocImm {
			if d66.Imm.Bool() {
				if ps.General {
				}
				ps67 := scm.PhiState{General: ps.General}
				ps67.OverlayValues = make([]scm.JITValueDesc, 67)
				ps67.OverlayValues[0] = d0
				ps67.OverlayValues[1] = d1
				ps67.OverlayValues[2] = d2
				ps67.OverlayValues[3] = d3
				ps67.OverlayValues[4] = d4
				ps67.OverlayValues[15] = d15
				ps67.OverlayValues[16] = d16
				ps67.OverlayValues[17] = d17
				ps67.OverlayValues[18] = d18
				ps67.OverlayValues[19] = d19
				ps67.OverlayValues[35] = d35
				ps67.OverlayValues[36] = d36
				ps67.OverlayValues[37] = d37
				ps67.OverlayValues[38] = d38
				ps67.OverlayValues[39] = d39
				ps67.OverlayValues[40] = d40
				ps67.OverlayValues[41] = d41
				ps67.OverlayValues[42] = d42
				ps67.OverlayValues[43] = d43
				ps67.OverlayValues[44] = d44
				ps67.OverlayValues[45] = d45
				ps67.OverlayValues[46] = d46
				ps67.OverlayValues[47] = d47
				ps67.OverlayValues[48] = d48
				ps67.OverlayValues[49] = d49
				ps67.OverlayValues[50] = d50
				ps67.OverlayValues[51] = d51
				ps67.OverlayValues[52] = d52
				ps67.OverlayValues[53] = d53
				ps67.OverlayValues[54] = d54
				ps67.OverlayValues[55] = d55
				ps67.OverlayValues[56] = d56
				ps67.OverlayValues[57] = d57
				ps67.OverlayValues[58] = d58
				ps67.OverlayValues[59] = d59
				ps67.OverlayValues[60] = d60
				ps67.OverlayValues[61] = d61
				ps67.OverlayValues[62] = d62
				ps67.OverlayValues[63] = d63
				ps67.OverlayValues[64] = d64
				ps67.OverlayValues[65] = d65
				ps67.OverlayValues[66] = d66
				return bbs[5].RenderPS(ps67)
			}
			if ps.General {
			}
			ps68 := scm.PhiState{General: ps.General}
			ps68.OverlayValues = make([]scm.JITValueDesc, 67)
			ps68.OverlayValues[0] = d0
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[15] = d15
			ps68.OverlayValues[16] = d16
			ps68.OverlayValues[17] = d17
			ps68.OverlayValues[18] = d18
			ps68.OverlayValues[19] = d19
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[37] = d37
			ps68.OverlayValues[38] = d38
			ps68.OverlayValues[39] = d39
			ps68.OverlayValues[40] = d40
			ps68.OverlayValues[41] = d41
			ps68.OverlayValues[42] = d42
			ps68.OverlayValues[43] = d43
			ps68.OverlayValues[44] = d44
			ps68.OverlayValues[45] = d45
			ps68.OverlayValues[46] = d46
			ps68.OverlayValues[47] = d47
			ps68.OverlayValues[48] = d48
			ps68.OverlayValues[49] = d49
			ps68.OverlayValues[50] = d50
			ps68.OverlayValues[51] = d51
			ps68.OverlayValues[52] = d52
			ps68.OverlayValues[53] = d53
			ps68.OverlayValues[54] = d54
			ps68.OverlayValues[55] = d55
			ps68.OverlayValues[56] = d56
			ps68.OverlayValues[57] = d57
			ps68.OverlayValues[58] = d58
			ps68.OverlayValues[59] = d59
			ps68.OverlayValues[60] = d60
			ps68.OverlayValues[61] = d61
			ps68.OverlayValues[62] = d62
			ps68.OverlayValues[63] = d63
			ps68.OverlayValues[64] = d64
			ps68.OverlayValues[65] = d65
			ps68.OverlayValues[66] = d66
			return bbs[7].RenderPS(ps68)
		}
		if !ps.General {
			ps.General = true
			return bbs[4].RenderPS(ps)
		}
		lbl14 := ctx.ReserveLabel()
		lbl15 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d66.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl14)
		ctx.EmitJmp(lbl15)
		ctx.MarkLabel(lbl14)
		ctx.EmitJmp(lbl6)
		ctx.MarkLabel(lbl15)
		ctx.EmitJmp(lbl8)
		ps69 := scm.PhiState{General: true}
		ps69.OverlayValues = make([]scm.JITValueDesc, 67)
		ps69.OverlayValues[0] = d0
		ps69.OverlayValues[1] = d1
		ps69.OverlayValues[2] = d2
		ps69.OverlayValues[3] = d3
		ps69.OverlayValues[4] = d4
		ps69.OverlayValues[15] = d15
		ps69.OverlayValues[16] = d16
		ps69.OverlayValues[17] = d17
		ps69.OverlayValues[18] = d18
		ps69.OverlayValues[19] = d19
		ps69.OverlayValues[35] = d35
		ps69.OverlayValues[36] = d36
		ps69.OverlayValues[37] = d37
		ps69.OverlayValues[38] = d38
		ps69.OverlayValues[39] = d39
		ps69.OverlayValues[40] = d40
		ps69.OverlayValues[41] = d41
		ps69.OverlayValues[42] = d42
		ps69.OverlayValues[43] = d43
		ps69.OverlayValues[44] = d44
		ps69.OverlayValues[45] = d45
		ps69.OverlayValues[46] = d46
		ps69.OverlayValues[47] = d47
		ps69.OverlayValues[48] = d48
		ps69.OverlayValues[49] = d49
		ps69.OverlayValues[50] = d50
		ps69.OverlayValues[51] = d51
		ps69.OverlayValues[52] = d52
		ps69.OverlayValues[53] = d53
		ps69.OverlayValues[54] = d54
		ps69.OverlayValues[55] = d55
		ps69.OverlayValues[56] = d56
		ps69.OverlayValues[57] = d57
		ps69.OverlayValues[58] = d58
		ps69.OverlayValues[59] = d59
		ps69.OverlayValues[60] = d60
		ps69.OverlayValues[61] = d61
		ps69.OverlayValues[62] = d62
		ps69.OverlayValues[63] = d63
		ps69.OverlayValues[64] = d64
		ps69.OverlayValues[65] = d65
		ps69.OverlayValues[66] = d66
		ps70 := scm.PhiState{General: true}
		ps70.OverlayValues = make([]scm.JITValueDesc, 67)
		ps70.OverlayValues[0] = d0
		ps70.OverlayValues[1] = d1
		ps70.OverlayValues[2] = d2
		ps70.OverlayValues[3] = d3
		ps70.OverlayValues[4] = d4
		ps70.OverlayValues[15] = d15
		ps70.OverlayValues[16] = d16
		ps70.OverlayValues[17] = d17
		ps70.OverlayValues[18] = d18
		ps70.OverlayValues[19] = d19
		ps70.OverlayValues[35] = d35
		ps70.OverlayValues[36] = d36
		ps70.OverlayValues[37] = d37
		ps70.OverlayValues[38] = d38
		ps70.OverlayValues[39] = d39
		ps70.OverlayValues[40] = d40
		ps70.OverlayValues[41] = d41
		ps70.OverlayValues[42] = d42
		ps70.OverlayValues[43] = d43
		ps70.OverlayValues[44] = d44
		ps70.OverlayValues[45] = d45
		ps70.OverlayValues[46] = d46
		ps70.OverlayValues[47] = d47
		ps70.OverlayValues[48] = d48
		ps70.OverlayValues[49] = d49
		ps70.OverlayValues[50] = d50
		ps70.OverlayValues[51] = d51
		ps70.OverlayValues[52] = d52
		ps70.OverlayValues[53] = d53
		ps70.OverlayValues[54] = d54
		ps70.OverlayValues[55] = d55
		ps70.OverlayValues[56] = d56
		ps70.OverlayValues[57] = d57
		ps70.OverlayValues[58] = d58
		ps70.OverlayValues[59] = d59
		ps70.OverlayValues[60] = d60
		ps70.OverlayValues[61] = d61
		ps70.OverlayValues[62] = d62
		ps70.OverlayValues[63] = d63
		ps70.OverlayValues[64] = d64
		ps70.OverlayValues[65] = d65
		ps70.OverlayValues[66] = d66
		snap71 := d0
		snap72 := d1
		snap73 := d2
		snap74 := d3
		snap75 := d4
		snap76 := d15
		snap77 := d16
		snap78 := d17
		snap79 := d18
		snap80 := d19
		snap81 := d35
		snap82 := d36
		snap83 := d37
		snap84 := d38
		snap85 := d39
		snap86 := d40
		snap87 := d41
		snap88 := d42
		snap89 := d43
		snap90 := d44
		snap91 := d45
		snap92 := d46
		snap93 := d47
		snap94 := d48
		snap95 := d49
		snap96 := d50
		snap97 := d51
		snap98 := d52
		snap99 := d53
		snap100 := d54
		snap101 := d55
		snap102 := d56
		snap103 := d57
		snap104 := d58
		snap105 := d59
		snap106 := d60
		snap107 := d61
		snap108 := d62
		snap109 := d63
		snap110 := d64
		snap111 := d65
		snap112 := d66
		alloc113 := ctx.SnapshotAllocState()
		if !bbs[7].Rendered {
			bbs[7].RenderPS(ps70)
		}
		ctx.RestoreAllocState(alloc113)
		d0 = snap71
		d1 = snap72
		d2 = snap73
		d3 = snap74
		d4 = snap75
		d15 = snap76
		d16 = snap77
		d17 = snap78
		d18 = snap79
		d19 = snap80
		d35 = snap81
		d36 = snap82
		d37 = snap83
		d38 = snap84
		d39 = snap85
		d40 = snap86
		d41 = snap87
		d42 = snap88
		d43 = snap89
		d44 = snap90
		d45 = snap91
		d46 = snap92
		d47 = snap93
		d48 = snap94
		d49 = snap95
		d50 = snap96
		d51 = snap97
		d52 = snap98
		d53 = snap99
		d54 = snap100
		d55 = snap101
		d56 = snap102
		d57 = snap103
		d58 = snap104
		d59 = snap105
		d60 = snap106
		d61 = snap107
		d62 = snap108
		d63 = snap109
		d64 = snap110
		d65 = snap111
		d66 = snap112
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps69)
		}
		return result
		ctx.FreeDesc(&d65)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
			d45 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
			d46 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
			d49 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		ctx.ReclaimUntrackedRegs()
		d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("prefix index out of range")}
		ctx.EnsureDesc(&d114)
		ctx.EnsureDesc(&d114)
		if d114.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d114.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d114)
			} else if d114.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d114)
			} else if d114.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d114)
			} else if d114.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d114.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d114 = tmpPair
		} else if d114.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d114.Type, Reg: ctx.AllocRegExcept(d114.Reg), Reg2: ctx.AllocRegExcept(d114.Reg)}
			switch d114.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d114)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d114)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d114)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d114)
			d114 = tmpPair
		}
		if d114.Loc != scm.LocRegPair && d114.Loc != scm.LocStackPair && d114.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d114})
		ctx.FreeDesc(&d114)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
			d45 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
			d46 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
			d49 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		ctx.ReclaimUntrackedRegs()
		var d115 scm.JITValueDesc
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r39 := ctx.AllocReg()
			r40 := ctx.AllocRegExcept(r39)
			r41 := ctx.AllocRegExcept(r39, r40)
			ctx.EmitMovRegMem64(r39, fieldAddr)
			ctx.EmitMovRegMem64(r40, fieldAddr+8)
			ctx.EmitMovRegMem64(r41, fieldAddr+16)
			d115 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r39, Reg2: r40, Reg3: r41}
			ctx.BindReg(r39, &d115)
			ctx.BindReg(r40, &d115)
			ctx.BindReg(r41, &d115)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r42 := ctx.AllocReg()
			r43 := ctx.AllocRegExcept(r42)
			r44 := ctx.AllocRegExcept(r42, r43)
			ctx.EmitMovRegMem(r42, thisptr.Reg, off)
			ctx.EmitMovRegMem(r43, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r44, thisptr.Reg, off+16)
			d115 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r42, Reg2: r43, Reg3: r44}
			ctx.BindReg(r42, &d115)
			ctx.BindReg(r43, &d115)
			ctx.BindReg(r44, &d115)
		}
		ctx.EnsureDesc(&d61)
		d117 = ctx.EmitSliceElementAddress(&d115, &d61, 16)
		ctx.EnsureDesc(&d117)
		r45 := ctx.AllocRegExcept(d117.Reg)
		ctx.EmitMovRegMem(r45, d117.Reg, 8)
		ctx.EmitMovRegMem(d117.Reg, d117.Reg, 0)
		d116 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d117.Reg, Reg2: r45}
		ctx.BindReg(d117.Reg, &d116)
		ctx.BindReg(r45, &d116)
		d119 = d1
		ctx.SyncDesc(&d119)
		if d119.Loc == scm.LocMem {
			tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d119.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d119.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d119 = tmpScalar
		}
		d119 = scm.JITPrepareScmerGoArg(ctx, d119)
		if d119.Loc != scm.LocRegPair && d119.Loc != scm.LocStackPair && d119.Loc != scm.LocInputPair {
			panic("jit: scm.Scmer.String receiver not materialized as pair")
		}
		d118 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d119}, 2)
		ctx.EnsureDesc(&d116)
		ctx.EnsureDesc(&d118)
		d120 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d116, d118}, 2)
		ctx.FreeDesc(&d116)
		ctx.EnsureDesc(&d120)
		d121 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d121)
		ctx.BindReg(r1, &d121)
		d122 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d120}, 2)
		ctx.EmitMovPairToResult(&d122, &d121)
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
		if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
			d35 = ps.OverlayValues[35]
		}
		if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
			d36 = ps.OverlayValues[36]
		}
		if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
			d37 = ps.OverlayValues[37]
		}
		if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
			d38 = ps.OverlayValues[38]
		}
		if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
			d39 = ps.OverlayValues[39]
		}
		if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
			d40 = ps.OverlayValues[40]
		}
		if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
			d41 = ps.OverlayValues[41]
		}
		if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
			d42 = ps.OverlayValues[42]
		}
		if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
			d43 = ps.OverlayValues[43]
		}
		if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
			d44 = ps.OverlayValues[44]
		}
		if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
			d45 = ps.OverlayValues[45]
		}
		if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
			d46 = ps.OverlayValues[46]
		}
		if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
			d47 = ps.OverlayValues[47]
		}
		if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
			d48 = ps.OverlayValues[48]
		}
		if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
			d49 = ps.OverlayValues[49]
		}
		if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
			d50 = ps.OverlayValues[50]
		}
		if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
			d51 = ps.OverlayValues[51]
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
		if len(ps.OverlayValues) > 114 && ps.OverlayValues[114].Loc != scm.LocNone {
			d114 = ps.OverlayValues[114]
		}
		if len(ps.OverlayValues) > 115 && ps.OverlayValues[115].Loc != scm.LocNone {
			d115 = ps.OverlayValues[115]
		}
		if len(ps.OverlayValues) > 116 && ps.OverlayValues[116].Loc != scm.LocNone {
			d116 = ps.OverlayValues[116]
		}
		if len(ps.OverlayValues) > 117 && ps.OverlayValues[117].Loc != scm.LocNone {
			d117 = ps.OverlayValues[117]
		}
		if len(ps.OverlayValues) > 118 && ps.OverlayValues[118].Loc != scm.LocNone {
			d118 = ps.OverlayValues[118]
		}
		if len(ps.OverlayValues) > 119 && ps.OverlayValues[119].Loc != scm.LocNone {
			d119 = ps.OverlayValues[119]
		}
		if len(ps.OverlayValues) > 120 && ps.OverlayValues[120].Loc != scm.LocNone {
			d120 = ps.OverlayValues[120]
		}
		if len(ps.OverlayValues) > 121 && ps.OverlayValues[121].Loc != scm.LocNone {
			d121 = ps.OverlayValues[121]
		}
		if len(ps.OverlayValues) > 122 && ps.OverlayValues[122].Loc != scm.LocNone {
			d122 = ps.OverlayValues[122]
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d61)
		var d123 scm.JITValueDesc
		if d61.Loc == scm.LocImm {
			d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d61.Imm.Int() < 0)}
		} else {
			r46 := ctx.AllocRegExcept(d61.Reg)
			ctx.EmitCmpRegImm32(d61.Reg, 0)
			ctx.EmitSetcc(r46, scm.CondSignedLess)
			d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			ctx.BindReg(r46, &d123)
		}
		d124 = d123
		ctx.EnsureDesc(&d124)
		if d124.Loc != scm.LocImm && d124.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d124.Loc == scm.LocImm {
			if d124.Imm.Bool() {
				if ps.General {
				}
				ps125 := scm.PhiState{General: ps.General}
				ps125.OverlayValues = make([]scm.JITValueDesc, 125)
				ps125.OverlayValues[0] = d0
				ps125.OverlayValues[1] = d1
				ps125.OverlayValues[2] = d2
				ps125.OverlayValues[3] = d3
				ps125.OverlayValues[4] = d4
				ps125.OverlayValues[15] = d15
				ps125.OverlayValues[16] = d16
				ps125.OverlayValues[17] = d17
				ps125.OverlayValues[18] = d18
				ps125.OverlayValues[19] = d19
				ps125.OverlayValues[35] = d35
				ps125.OverlayValues[36] = d36
				ps125.OverlayValues[37] = d37
				ps125.OverlayValues[38] = d38
				ps125.OverlayValues[39] = d39
				ps125.OverlayValues[40] = d40
				ps125.OverlayValues[41] = d41
				ps125.OverlayValues[42] = d42
				ps125.OverlayValues[43] = d43
				ps125.OverlayValues[44] = d44
				ps125.OverlayValues[45] = d45
				ps125.OverlayValues[46] = d46
				ps125.OverlayValues[47] = d47
				ps125.OverlayValues[48] = d48
				ps125.OverlayValues[49] = d49
				ps125.OverlayValues[50] = d50
				ps125.OverlayValues[51] = d51
				ps125.OverlayValues[52] = d52
				ps125.OverlayValues[53] = d53
				ps125.OverlayValues[54] = d54
				ps125.OverlayValues[55] = d55
				ps125.OverlayValues[56] = d56
				ps125.OverlayValues[57] = d57
				ps125.OverlayValues[58] = d58
				ps125.OverlayValues[59] = d59
				ps125.OverlayValues[60] = d60
				ps125.OverlayValues[61] = d61
				ps125.OverlayValues[62] = d62
				ps125.OverlayValues[63] = d63
				ps125.OverlayValues[64] = d64
				ps125.OverlayValues[65] = d65
				ps125.OverlayValues[66] = d66
				ps125.OverlayValues[114] = d114
				ps125.OverlayValues[115] = d115
				ps125.OverlayValues[116] = d116
				ps125.OverlayValues[117] = d117
				ps125.OverlayValues[118] = d118
				ps125.OverlayValues[119] = d119
				ps125.OverlayValues[120] = d120
				ps125.OverlayValues[121] = d121
				ps125.OverlayValues[122] = d122
				ps125.OverlayValues[123] = d123
				ps125.OverlayValues[124] = d124
				return bbs[5].RenderPS(ps125)
			}
			if ps.General {
			}
			ps126 := scm.PhiState{General: ps.General}
			ps126.OverlayValues = make([]scm.JITValueDesc, 125)
			ps126.OverlayValues[0] = d0
			ps126.OverlayValues[1] = d1
			ps126.OverlayValues[2] = d2
			ps126.OverlayValues[3] = d3
			ps126.OverlayValues[4] = d4
			ps126.OverlayValues[15] = d15
			ps126.OverlayValues[16] = d16
			ps126.OverlayValues[17] = d17
			ps126.OverlayValues[18] = d18
			ps126.OverlayValues[19] = d19
			ps126.OverlayValues[35] = d35
			ps126.OverlayValues[36] = d36
			ps126.OverlayValues[37] = d37
			ps126.OverlayValues[38] = d38
			ps126.OverlayValues[39] = d39
			ps126.OverlayValues[40] = d40
			ps126.OverlayValues[41] = d41
			ps126.OverlayValues[42] = d42
			ps126.OverlayValues[43] = d43
			ps126.OverlayValues[44] = d44
			ps126.OverlayValues[45] = d45
			ps126.OverlayValues[46] = d46
			ps126.OverlayValues[47] = d47
			ps126.OverlayValues[48] = d48
			ps126.OverlayValues[49] = d49
			ps126.OverlayValues[50] = d50
			ps126.OverlayValues[51] = d51
			ps126.OverlayValues[52] = d52
			ps126.OverlayValues[53] = d53
			ps126.OverlayValues[54] = d54
			ps126.OverlayValues[55] = d55
			ps126.OverlayValues[56] = d56
			ps126.OverlayValues[57] = d57
			ps126.OverlayValues[58] = d58
			ps126.OverlayValues[59] = d59
			ps126.OverlayValues[60] = d60
			ps126.OverlayValues[61] = d61
			ps126.OverlayValues[62] = d62
			ps126.OverlayValues[63] = d63
			ps126.OverlayValues[64] = d64
			ps126.OverlayValues[65] = d65
			ps126.OverlayValues[66] = d66
			ps126.OverlayValues[114] = d114
			ps126.OverlayValues[115] = d115
			ps126.OverlayValues[116] = d116
			ps126.OverlayValues[117] = d117
			ps126.OverlayValues[118] = d118
			ps126.OverlayValues[119] = d119
			ps126.OverlayValues[120] = d120
			ps126.OverlayValues[121] = d121
			ps126.OverlayValues[122] = d122
			ps126.OverlayValues[123] = d123
			ps126.OverlayValues[124] = d124
			return bbs[6].RenderPS(ps126)
		}
		if !ps.General {
			ps.General = true
			return bbs[7].RenderPS(ps)
		}
		lbl16 := ctx.ReserveLabel()
		lbl17 := ctx.ReserveLabel()
		ctx.EmitCmpRegImm32(d124.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl16)
		ctx.EmitJmp(lbl17)
		ctx.MarkLabel(lbl16)
		ctx.EmitJmp(lbl6)
		ctx.MarkLabel(lbl17)
		ctx.EmitJmp(lbl7)
		ps127 := scm.PhiState{General: true}
		ps127.OverlayValues = make([]scm.JITValueDesc, 125)
		ps127.OverlayValues[0] = d0
		ps127.OverlayValues[1] = d1
		ps127.OverlayValues[2] = d2
		ps127.OverlayValues[3] = d3
		ps127.OverlayValues[4] = d4
		ps127.OverlayValues[15] = d15
		ps127.OverlayValues[16] = d16
		ps127.OverlayValues[17] = d17
		ps127.OverlayValues[18] = d18
		ps127.OverlayValues[19] = d19
		ps127.OverlayValues[35] = d35
		ps127.OverlayValues[36] = d36
		ps127.OverlayValues[37] = d37
		ps127.OverlayValues[38] = d38
		ps127.OverlayValues[39] = d39
		ps127.OverlayValues[40] = d40
		ps127.OverlayValues[41] = d41
		ps127.OverlayValues[42] = d42
		ps127.OverlayValues[43] = d43
		ps127.OverlayValues[44] = d44
		ps127.OverlayValues[45] = d45
		ps127.OverlayValues[46] = d46
		ps127.OverlayValues[47] = d47
		ps127.OverlayValues[48] = d48
		ps127.OverlayValues[49] = d49
		ps127.OverlayValues[50] = d50
		ps127.OverlayValues[51] = d51
		ps127.OverlayValues[52] = d52
		ps127.OverlayValues[53] = d53
		ps127.OverlayValues[54] = d54
		ps127.OverlayValues[55] = d55
		ps127.OverlayValues[56] = d56
		ps127.OverlayValues[57] = d57
		ps127.OverlayValues[58] = d58
		ps127.OverlayValues[59] = d59
		ps127.OverlayValues[60] = d60
		ps127.OverlayValues[61] = d61
		ps127.OverlayValues[62] = d62
		ps127.OverlayValues[63] = d63
		ps127.OverlayValues[64] = d64
		ps127.OverlayValues[65] = d65
		ps127.OverlayValues[66] = d66
		ps127.OverlayValues[114] = d114
		ps127.OverlayValues[115] = d115
		ps127.OverlayValues[116] = d116
		ps127.OverlayValues[117] = d117
		ps127.OverlayValues[118] = d118
		ps127.OverlayValues[119] = d119
		ps127.OverlayValues[120] = d120
		ps127.OverlayValues[121] = d121
		ps127.OverlayValues[122] = d122
		ps127.OverlayValues[123] = d123
		ps127.OverlayValues[124] = d124
		ps128 := scm.PhiState{General: true}
		ps128.OverlayValues = make([]scm.JITValueDesc, 125)
		ps128.OverlayValues[0] = d0
		ps128.OverlayValues[1] = d1
		ps128.OverlayValues[2] = d2
		ps128.OverlayValues[3] = d3
		ps128.OverlayValues[4] = d4
		ps128.OverlayValues[15] = d15
		ps128.OverlayValues[16] = d16
		ps128.OverlayValues[17] = d17
		ps128.OverlayValues[18] = d18
		ps128.OverlayValues[19] = d19
		ps128.OverlayValues[35] = d35
		ps128.OverlayValues[36] = d36
		ps128.OverlayValues[37] = d37
		ps128.OverlayValues[38] = d38
		ps128.OverlayValues[39] = d39
		ps128.OverlayValues[40] = d40
		ps128.OverlayValues[41] = d41
		ps128.OverlayValues[42] = d42
		ps128.OverlayValues[43] = d43
		ps128.OverlayValues[44] = d44
		ps128.OverlayValues[45] = d45
		ps128.OverlayValues[46] = d46
		ps128.OverlayValues[47] = d47
		ps128.OverlayValues[48] = d48
		ps128.OverlayValues[49] = d49
		ps128.OverlayValues[50] = d50
		ps128.OverlayValues[51] = d51
		ps128.OverlayValues[52] = d52
		ps128.OverlayValues[53] = d53
		ps128.OverlayValues[54] = d54
		ps128.OverlayValues[55] = d55
		ps128.OverlayValues[56] = d56
		ps128.OverlayValues[57] = d57
		ps128.OverlayValues[58] = d58
		ps128.OverlayValues[59] = d59
		ps128.OverlayValues[60] = d60
		ps128.OverlayValues[61] = d61
		ps128.OverlayValues[62] = d62
		ps128.OverlayValues[63] = d63
		ps128.OverlayValues[64] = d64
		ps128.OverlayValues[65] = d65
		ps128.OverlayValues[66] = d66
		ps128.OverlayValues[114] = d114
		ps128.OverlayValues[115] = d115
		ps128.OverlayValues[116] = d116
		ps128.OverlayValues[117] = d117
		ps128.OverlayValues[118] = d118
		ps128.OverlayValues[119] = d119
		ps128.OverlayValues[120] = d120
		ps128.OverlayValues[121] = d121
		ps128.OverlayValues[122] = d122
		ps128.OverlayValues[123] = d123
		ps128.OverlayValues[124] = d124
		snap129 := d0
		snap130 := d1
		snap131 := d2
		snap132 := d3
		snap133 := d4
		snap134 := d15
		snap135 := d16
		snap136 := d17
		snap137 := d18
		snap138 := d19
		snap139 := d35
		snap140 := d36
		snap141 := d37
		snap142 := d38
		snap143 := d39
		snap144 := d40
		snap145 := d41
		snap146 := d42
		snap147 := d43
		snap148 := d44
		snap149 := d45
		snap150 := d46
		snap151 := d47
		snap152 := d48
		snap153 := d49
		snap154 := d50
		snap155 := d51
		snap156 := d52
		snap157 := d53
		snap158 := d54
		snap159 := d55
		snap160 := d56
		snap161 := d57
		snap162 := d58
		snap163 := d59
		snap164 := d60
		snap165 := d61
		snap166 := d62
		snap167 := d63
		snap168 := d64
		snap169 := d65
		snap170 := d66
		snap171 := d114
		snap172 := d115
		snap173 := d116
		snap174 := d117
		snap175 := d118
		snap176 := d119
		snap177 := d120
		snap178 := d121
		snap179 := d122
		snap180 := d123
		snap181 := d124
		alloc182 := ctx.SnapshotAllocState()
		if !bbs[6].Rendered {
			bbs[6].RenderPS(ps128)
		}
		ctx.RestoreAllocState(alloc182)
		d0 = snap129
		d1 = snap130
		d2 = snap131
		d3 = snap132
		d4 = snap133
		d15 = snap134
		d16 = snap135
		d17 = snap136
		d18 = snap137
		d19 = snap138
		d35 = snap139
		d36 = snap140
		d37 = snap141
		d38 = snap142
		d39 = snap143
		d40 = snap144
		d41 = snap145
		d42 = snap146
		d43 = snap147
		d44 = snap148
		d45 = snap149
		d46 = snap150
		d47 = snap151
		d48 = snap152
		d49 = snap153
		d50 = snap154
		d51 = snap155
		d52 = snap156
		d53 = snap157
		d54 = snap158
		d55 = snap159
		d56 = snap160
		d57 = snap161
		d58 = snap162
		d59 = snap163
		d60 = snap164
		d61 = snap165
		d62 = snap166
		d63 = snap167
		d64 = snap168
		d65 = snap169
		d66 = snap170
		d114 = snap171
		d115 = snap172
		d116 = snap173
		d117 = snap174
		d118 = snap175
		d119 = snap176
		d120 = snap177
		d121 = snap178
		d122 = snap179
		d123 = snap180
		d124 = snap181
		if !bbs[5].Rendered {
			return bbs[5].RenderPS(ps127)
		}
		return result
		ctx.FreeDesc(&d123)
		return result
	}
	ps183 := scm.PhiState{General: false}
	_ = bbs[0].RenderPS(ps183)
	ctx.MarkLabel(lbl0)
	d184 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
	ctx.BindReg(r0, &d184)
	ctx.BindReg(r1, &d184)
	ctx.EmitMovPairToResult(&d184, &result)
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
