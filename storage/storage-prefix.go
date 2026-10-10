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
	size := uint(unsafe.Sizeof(*s)-unsafe.Sizeof(s.prefixes)-unsafe.Sizeof(s.values)) + s.prefixes.ComputeSize() + s.values.ComputeSize()
	size += uint(cap(s.prefixdictionary)) * uint(unsafe.Sizeof(string("")))
	for _, prefix := range s.prefixdictionary {
		size += uint(len(prefix))
	}
	return size
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
//
//jitgen:control-flow-stable recid count target/3 stride
func (s *StoragePrefix) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	s.values.GetValueRange(recid, count, target, stride)
	idxbuf := make([]scm.Scmer, count)
	s.prefixes.GetValueRange(recid, count, idxbuf, 1)
	s.applyPrefixInPlace(target, idxbuf, count, stride)
}

//jitgen:control-flow-stable recids/3 target/3 stride
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
	var d27 scm.JITValueDesc
	_ = d27
	var d28 scm.JITValueDesc
	_ = d28
	var d29 scm.JITValueDesc
	_ = d29
	var d30 scm.JITValueDesc
	_ = d30
	var d31 scm.JITValueDesc
	_ = d31
	var d32 scm.JITValueDesc
	_ = d32
	var d33 scm.JITValueDesc
	_ = d33
	var d34 scm.JITValueDesc
	_ = d34
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
	var d94 scm.JITValueDesc
	_ = d94
	var d95 scm.JITValueDesc
	_ = d95
	var d96 scm.JITValueDesc
	_ = d96
	var d97 scm.JITValueDesc
	_ = d97
	var d98 scm.JITValueDesc
	_ = d98
	var d99 scm.JITValueDesc
	_ = d99
	var d100 scm.JITValueDesc
	_ = d100
	var d101 scm.JITValueDesc
	_ = d101
	var d102 scm.JITValueDesc
	_ = d102
	var d103 scm.JITValueDesc
	_ = d103
	var d104 scm.JITValueDesc
	_ = d104
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
		ctx.TrackPointer(unsafe.Pointer((*StorageString)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + uintptr(unsafe.Offsetof((*StoragePrefix)(nil).values))))))
		d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uintptr(unsafe.Pointer((*StorageString)(unsafe.Pointer(uintptr(unsafe.Pointer(s)) + uintptr(unsafe.Offsetof((*StoragePrefix)(nil).values)))))))), RelocatablePointer: true}
		if d0.Loc == scm.LocRegPair || d0.Loc == scm.LocStackPair || d0.Loc == scm.LocRegTriple || d0.Loc == scm.LocStackTriple {
			panic("jit: generic call arg expects 1-word value")
		}
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
				return bbs[1].Render()
			}
			return bbs[2].Render()
		}
		ctx.EmitCmpRegImm32(d4.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl2)
		if bbs[2].Rendered {
			ctx.EmitJmp(lbl3)
		}
		ctx.FlushRegisterMoves()
		if !bbs[2].Rendered {
			snap5 := d0
			snap6 := d1
			snap7 := d2
			snap8 := d3
			snap9 := d4
			alloc10 := ctx.SnapshotAllocState()
			bbs[2].Render()
			ctx.RestoreAllocState(alloc10)
			d0 = snap5
			d1 = snap6
			d2 = snap7
			d3 = snap8
			d4 = snap9
		}
		if !bbs[1].Rendered {
			return bbs[1].Render()
		}
		return result
		ctx.FreeDesc(&d2)
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
		d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagNil, Imm: scm.NewNil()}
		d12 = result
		ctx.EnsureDesc(&d11)
		if d11.Loc == scm.LocRegPair {
			ctx.EmitMovPairToResult(&d11, &d12)
		} else {
			switch d11.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(d12, d11)
			case scm.TagInt:
				ctx.EmitMakeInt(d12, d11)
			case scm.TagFloat:
				ctx.EmitMakeFloat(d12, d11)
			case scm.TagNil:
				ctx.EmitMakeNil(d12)
			default:
				ctx.EmitMovPairToResult(&d11, &d12)
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
		d14 = d1
		d14.ID = 0
		d13 = ctx.EmitIsStringBorrowed(&d14, scm.JITValueDesc{Loc: scm.LocAny})
		d15 = d13
		ctx.EnsureDesc(&d15)
		if d15.Loc != scm.LocImm && d15.Loc != scm.LocReg {
			panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
		}
		if d15.Loc == scm.LocImm {
			if d15.Imm.Bool() {
				return bbs[4].Render()
			}
			return bbs[3].Render()
		}
		ctx.EmitCmpRegImm32(d15.Reg, 0)
		ctx.EmitJump(scm.CondNotEqual, lbl5)
		if bbs[3].Rendered {
			ctx.EmitJmp(lbl4)
		}
		ctx.FlushRegisterMoves()
		if !bbs[3].Rendered {
			snap16 := d0
			snap17 := d1
			snap18 := d2
			snap19 := d3
			snap20 := d4
			snap21 := d11
			snap22 := d12
			snap23 := d13
			snap24 := d14
			snap25 := d15
			alloc26 := ctx.SnapshotAllocState()
			bbs[3].Render()
			ctx.RestoreAllocState(alloc26)
			d0 = snap16
			d1 = snap17
			d2 = snap18
			d3 = snap19
			d4 = snap20
			d11 = snap21
			d12 = snap22
			d13 = snap23
			d14 = snap24
			d15 = snap25
		}
		if !bbs[4].Rendered {
			return bbs[4].Render()
		}
		return result
		ctx.FreeDesc(&d13)
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
		d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("invalid value in prefix storage")}
		ctx.EnsureDesc(&d27)
		ctx.EnsureDesc(&d27)
		if d27.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d27.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d27)
			} else if d27.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d27)
			} else if d27.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d27)
			} else if d27.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d27.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d27 = tmpPair
		} else if d27.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d27.Type, Reg: ctx.AllocRegExcept(d27.Reg), Reg2: ctx.AllocRegExcept(d27.Reg)}
			switch d27.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d27)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d27)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d27)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d27)
			d27 = tmpPair
		}
		if d27.Loc != scm.LocRegPair && d27.Loc != scm.LocStackPair && d27.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d27})
		ctx.FreeDesc(&d27)
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
		ctx.EnsureDesc(&idxInt)
		d28 = idxInt
		_ = d28
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
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48
			val := *(*uint8)(unsafe.Pointer(fieldAddr))
			d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 48)
			r0 := ctx.AllocReg()
			ctx.EmitMovRegMemB(r0, thisptr.Reg, off)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			ctx.BindReg(r0, &d29)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d29)
		ctx.EnsureDesc(&d29)
		if d29.Loc == scm.LocImm {
			d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d29.Imm.Int()))))}
		} else {
			r1 := ctx.AllocReg()
			ctx.EmitMovRegReg(r1, d29.Reg)
			ctx.EmitShlRegImm8(r1, 56)
			ctx.EmitShrRegImm8(r1, 56)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
			ctx.BindReg(r1, &d30)
		}
		ctx.FreeDesc(&d29)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d28)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d28)
		ctx.EnsureDesc(&d30)
		ctx.EnsureDescsTogether(&d28, &d30)
		if d28.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() * d30.Imm.Int())}
		} else if d28.Loc == scm.LocImm {
			ctx.EnsureDesc(&d30)
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegReg(scratch, d30.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d28.Imm.Int())
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d32)
		} else if d30.Loc == scm.LocImm {
			ctx.EnsureDesc(&d28)
			scratch := ctx.AllocRegExcept(d28.Reg)
			ctx.EmitMovRegReg(scratch, d28.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntMul, 64, scratch, d30.Imm.Int())
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d32)
		} else {
			ctx.EnsureDesc(&d28)
			ctx.SyncDesc(&d30)
			r2 := ctx.AllocRegExcept(d28.Reg, d30.Reg)
			ctx.EmitMovRegReg(r2, d28.Reg)
			ctx.EmitIntBinary(scm.JITIntMul, 64, r2, &d30)
			d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			ctx.BindReg(r2, &d32)
		}
		if d32.Loc == scm.LocReg && d28.Loc == scm.LocReg && d32.Reg == d28.Reg {
			ctx.TransferReg(d28.Reg)
			d28.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		if d32.Loc == scm.LocImm {
			d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() / 64)}
		} else {
			r3 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r3, d32.Reg)
			ctx.EmitShrRegImm8(r3, 6)
			d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d33)
		}
		if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d32)
		if d32.Loc == scm.LocImm {
			d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() % 64)}
		} else {
			r4 := ctx.AllocRegExcept(d32.Reg)
			ctx.EmitMovRegReg(r4, d32.Reg)
			ctx.EmitAndRegImm32(r4, 63)
			d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
			ctx.BindReg(r4, &d34)
		}
		if d34.Loc == scm.LocReg && d32.Loc == scm.LocReg && d34.Reg == d32.Reg {
			ctx.TransferReg(d32.Reg)
			d32.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d32)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
			dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
			sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
			sliceCap := *(*int)(unsafe.Pointer(fieldAddr + 16))
			d35 = scm.JITValueDesc{Loc: scm.LocMem, Type: scm.TagSlice, MemPtr: dataPtr, KnownSliceLen: int32(sliceLen), KnownSliceCap: int32(sliceCap), SliceSizeKnown: true, GoArray: true, RelocatablePointer: true, Rooted: true}
		} else {
			r5 := ctx.AllocReg()
			r6 := ctx.AllocRegExcept(r5)
			r7 := ctx.AllocRegExcept(r5, r6)
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
			ctx.EmitMovRegMem(r5, thisptr.Reg, off)
			ctx.EmitMovRegMem(r6, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r7, thisptr.Reg, off+16)
			d35 = scm.JITValueDesc{Loc: scm.LocRegTriple, Type: scm.TagSlice, Reg: r5, Reg2: r6, Reg3: r7}
			ctx.BindReg(r5, &d35)
			ctx.BindReg(r6, &d35)
			ctx.BindReg(r7, &d35)
			ctx.BindReg(r5, &d35)
			ctx.BindReg(r6, &d35)
			ctx.BindReg(r7, &d35)
		}
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		d36 = ctx.EmitLoadScalarSliceElement(&d35, &d33, 8, scm.TagInt)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d36)
		ctx.EnsureDesc(&d34)
		ctx.EnsureDescsTogether(&d36, &d34)
		if d36.Loc == scm.LocImm && d34.Loc == scm.LocImm {
			d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) << uint64(d34.Imm.Int())))}
		} else if d34.Loc == scm.LocImm {
			r8 := ctx.AllocRegExcept(d36.Reg)
			ctx.EmitMovRegReg(r8, d36.Reg)
			ctx.EmitShlRegImm8(r8, uint8(d34.Imm.Int()))
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			ctx.BindReg(r8, &d37)
		} else {
			shiftSrc := d36.Reg
			r9 := ctx.AllocRegExcept(d36.Reg, d34.Reg)
			ctx.EmitMovRegReg(r9, d36.Reg)
			shiftSrc = r9
			ctx.EmitShiftLeft(shiftSrc, d34.Reg, true)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d37)
		}
		if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
			ctx.TransferReg(d36.Reg)
			d36.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d36)
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d33)
		ctx.EnsureDesc(&d33)
		if d33.Loc == scm.LocImm {
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() + 1)}
		} else {
			scratch := ctx.AllocRegExcept(d33.Reg)
			ctx.EmitMovRegReg(scratch, d33.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, 1)
			d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d38)
		}
		if d38.Loc == scm.LocReg && d33.Loc == scm.LocReg && d38.Reg == d33.Reg {
			ctx.TransferReg(d33.Reg)
			d33.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d33)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		d39 = ctx.EmitLoadScalarSliceElement(&d35, &d38, 8, scm.TagInt)
		ctx.FreeDesc(&d38)
		ctx.ReclaimUntrackedRegs()
		d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d34)
		ctx.EnsureDescsTogether(&d40, &d34)
		if d40.Loc == scm.LocImm && d34.Loc == scm.LocImm {
			d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() - d34.Imm.Int())}
		} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
			ctx.EnsureDesc(&d40)
			r10 := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(r10, d40.Reg)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			ctx.BindReg(r10, &d41)
		} else if d40.Loc == scm.LocImm {
			ctx.EnsureDesc(&d34)
			scratch := ctx.AllocRegExcept(d34.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d34)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else if d34.Loc == scm.LocImm {
			ctx.EnsureDesc(&d40)
			scratch := ctx.AllocRegExcept(d40.Reg)
			ctx.EmitMovRegReg(scratch, d40.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d34.Imm.Int())
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d41)
		} else {
			ctx.EnsureDesc(&d40)
			ctx.SyncDesc(&d34)
			r11 := ctx.AllocRegExcept(d40.Reg, d34.Reg)
			ctx.EmitMovRegReg(r11, d40.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r11, &d34)
			d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			ctx.BindReg(r11, &d41)
		}
		if d41.Loc == scm.LocReg && d40.Loc == scm.LocReg && d41.Reg == d40.Reg {
			ctx.TransferReg(d40.Reg)
			d40.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d34)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d39)
		ctx.EnsureDesc(&d41)
		ctx.EnsureDescsTogether(&d39, &d41)
		if d39.Loc == scm.LocImm && d41.Loc == scm.LocImm {
			d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d39.Imm.Int()) >> uint64(d41.Imm.Int())))}
		} else if d41.Loc == scm.LocImm {
			r12 := ctx.AllocRegExcept(d39.Reg)
			ctx.EmitMovRegReg(r12, d39.Reg)
			ctx.EmitShrRegImm8(r12, uint8(d41.Imm.Int()))
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
			ctx.BindReg(r12, &d42)
		} else {
			shiftSrc := d39.Reg
			r13 := ctx.AllocRegExcept(d39.Reg, d41.Reg)
			ctx.EmitMovRegReg(r13, d39.Reg)
			shiftSrc = r13
			ctx.EmitShiftRight(shiftSrc, d41.Reg, false)
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d42)
		}
		if d42.Loc == scm.LocReg && d39.Loc == scm.LocReg && d42.Reg == d39.Reg {
			ctx.TransferReg(d39.Reg)
			d39.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d39)
		ctx.FreeDesc(&d41)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d37)
		ctx.EnsureDesc(&d42)
		if d37.Loc == scm.LocImm && d42.Loc == scm.LocImm {
			d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() | d42.Imm.Int())}
		} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
			ctx.BindReg(d42.Reg, &d43)
		} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
			r14 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(r14, d37.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			ctx.BindReg(r14, &d43)
		} else if d37.Loc == scm.LocImm {
			scratch := ctx.AllocRegExcept(d42.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
			ctx.EmitOrInt64(scratch, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d43)
		} else if d42.Loc == scm.LocImm {
			r15 := ctx.AllocRegExcept(d37.Reg)
			ctx.EmitMovRegReg(r15, d37.Reg)
			if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
				ctx.EmitOrRegImm32(r15, int32(d42.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d42.Imm.Int()))
				ctx.EmitOrInt64(r15, ctx.ScratchReg)
			}
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			ctx.BindReg(r15, &d43)
		} else {
			r16 := ctx.AllocRegExcept(d37.Reg, d42.Reg)
			ctx.EmitMovRegReg(r16, d37.Reg)
			ctx.EmitOrInt64(r16, d42.Reg)
			d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			ctx.BindReg(r16, &d43)
		}
		if d43.Loc == scm.LocReg && d37.Loc == scm.LocReg && d43.Reg == d37.Reg {
			ctx.TransferReg(d37.Reg)
			d37.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d37)
		ctx.FreeDesc(&d42)
		ctx.ReclaimUntrackedRegs()
		d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
		ctx.EnsureDesc(&d30)
		ctx.EnsureDescsTogether(&d44, &d30)
		if d44.Loc == scm.LocImm && d30.Loc == scm.LocImm {
			d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() - d30.Imm.Int())}
		} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
			ctx.EnsureDesc(&d44)
			r17 := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(r17, d44.Reg)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			ctx.BindReg(r17, &d45)
		} else if d44.Loc == scm.LocImm {
			ctx.EnsureDesc(&d30)
			scratch := ctx.AllocRegExcept(d30.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
			ctx.EmitIntBinary(scm.JITIntSub, 64, scratch, &d30)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		} else if d30.Loc == scm.LocImm {
			ctx.EnsureDesc(&d44)
			scratch := ctx.AllocRegExcept(d44.Reg)
			ctx.EmitMovRegReg(scratch, d44.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntSub, 64, scratch, d30.Imm.Int())
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d45)
		} else {
			ctx.EnsureDesc(&d44)
			ctx.SyncDesc(&d30)
			r18 := ctx.AllocRegExcept(d44.Reg, d30.Reg)
			ctx.EmitMovRegReg(r18, d44.Reg)
			ctx.EmitIntBinary(scm.JITIntSub, 64, r18, &d30)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			ctx.BindReg(r18, &d45)
		}
		if d45.Loc == scm.LocReg && d44.Loc == scm.LocReg && d45.Reg == d44.Reg {
			ctx.TransferReg(d44.Reg)
			d44.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d30)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d43)
		ctx.EnsureDesc(&d45)
		ctx.EnsureDescsTogether(&d43, &d45)
		if d43.Loc == scm.LocImm && d45.Loc == scm.LocImm {
			d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d43.Imm.Int()) >> uint64(d45.Imm.Int())))}
		} else if d45.Loc == scm.LocImm {
			r19 := ctx.AllocRegExcept(d43.Reg)
			ctx.EmitMovRegReg(r19, d43.Reg)
			ctx.EmitShrRegImm8(r19, uint8(d45.Imm.Int()))
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			ctx.BindReg(r19, &d46)
		} else {
			shiftSrc := d43.Reg
			r20 := ctx.AllocRegExcept(d43.Reg, d45.Reg)
			ctx.EmitMovRegReg(r20, d43.Reg)
			shiftSrc = r20
			ctx.EmitShiftRight(shiftSrc, d45.Reg, false)
			d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d46)
		}
		if d46.Loc == scm.LocReg && d43.Loc == scm.LocReg && d46.Reg == d43.Reg {
			ctx.TransferReg(d43.Reg)
			d43.Loc = scm.LocNone
		}
		ctx.FreeDesc(&d43)
		ctx.FreeDesc(&d45)
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d46)
		ctx.FreeDesc(&idxInt)
		ctx.EnsureDesc(&d46)
		ctx.EnsureDesc(&d46)
		if d46.Loc == scm.LocImm {
			d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d46.Imm.Int()))))}
		} else {
			r21 := ctx.AllocReg()
			ctx.EmitMovRegReg(r21, d46.Reg)
			d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			ctx.BindReg(r21, &d47)
		}
		ctx.FreeDesc(&d46)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56
			val := *(*int64)(unsafe.Pointer(fieldAddr))
			d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 56)
			r22 := ctx.AllocReg()
			ctx.EmitMovRegMem(r22, thisptr.Reg, off)
			d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
			ctx.BindReg(r22, &d48)
		}
		ctx.EnsureDesc(&d47)
		ctx.EnsureDesc(&d48)
		ctx.EnsureDescsTogether(&d47, &d48)
		if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
			d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() + d48.Imm.Int())}
		} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
			ctx.EnsureDesc(&d47)
			r23 := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(r23, d47.Reg)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			ctx.BindReg(r23, &d49)
		} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
			ctx.EnsureDesc(&d48)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d48.Reg}
			ctx.BindReg(d48.Reg, &d49)
		} else if d47.Loc == scm.LocImm {
			ctx.EnsureDesc(&d48)
			scratch := ctx.AllocRegExcept(d48.Reg)
			ctx.EmitMovRegReg(scratch, d48.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d47.Imm.Int())
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else if d48.Loc == scm.LocImm {
			ctx.EnsureDesc(&d47)
			scratch := ctx.AllocRegExcept(d47.Reg)
			ctx.EmitMovRegReg(scratch, d47.Reg)
			ctx.EmitIntBinaryImm(scm.JITIntAdd, 64, scratch, d48.Imm.Int())
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			ctx.BindReg(scratch, &d49)
		} else {
			ctx.EnsureDesc(&d47)
			ctx.SyncDesc(&d48)
			r24 := ctx.AllocRegExcept(d47.Reg, d48.Reg)
			ctx.EmitMovRegReg(r24, d47.Reg)
			ctx.EmitIntBinary(scm.JITIntAdd, 64, r24, &d48)
			d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			ctx.BindReg(r24, &d49)
		}
		if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
			ctx.TransferReg(d47.Reg)
			d47.Loc = scm.LocNone
		}
		ctx.StabilizeDescForControlFlow(&d49)
		ctx.FreeDesc(&d47)
		ctx.FreeDesc(&d48)
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r25 := ctx.AllocReg()
			r26 := ctx.AllocRegExcept(r25)
			r27 := ctx.AllocRegExcept(r25, r26)
			ctx.EmitMovRegMem64(r25, fieldAddr)
			ctx.EmitMovRegMem64(r26, fieldAddr+8)
			ctx.EmitMovRegMem64(r27, fieldAddr+16)
			d50 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r25, Reg2: r26, Reg3: r27}
			ctx.BindReg(r25, &d50)
			ctx.BindReg(r26, &d50)
			ctx.BindReg(r27, &d50)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r28 := ctx.AllocReg()
			r29 := ctx.AllocRegExcept(r28)
			r30 := ctx.AllocRegExcept(r28, r29)
			ctx.EmitMovRegMem(r28, thisptr.Reg, off)
			ctx.EmitMovRegMem(r29, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r30, thisptr.Reg, off+16)
			d50 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r28, Reg2: r29, Reg3: r30}
			ctx.BindReg(r28, &d50)
			ctx.BindReg(r29, &d50)
			ctx.BindReg(r30, &d50)
		}
		if d50.SliceSizeKnown {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d50.KnownSliceLen))}
		} else if d50.Loc == scm.LocImm {
			d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d50.StackOff))}
		} else if d50.Loc == scm.LocStackTriple {
			d51 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: d50.StackOff + 8, NoHeapPointer: true}
		} else {
			ctx.EnsureDesc(&d50)
			if d50.Loc == scm.LocRegPair || d50.Loc == scm.LocRegTriple {
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d50.Reg2, ID: 0}
			} else if d50.Loc == scm.LocReg {
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d50.Reg, ID: 0}
			} else {
				panic("len on unsupported descriptor location")
			}
		}
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d51)
		ctx.EnsureDesc(&d49)
		ctx.EnsureDesc(&d51)
		ctx.EnsureDescsTogether(&d49, &d51)
		if d49.Loc == scm.LocImm && d51.Loc == scm.LocImm {
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d49.Imm.Int() >= d51.Imm.Int())}
		} else if d51.Loc == scm.LocImm {
			r31 := ctx.AllocRegExcept(d49.Reg)
			if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
				ctx.EmitCmpRegImm32(d49.Reg, int32(d51.Imm.Int()))
			} else {
				ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d51.Imm.Int()))
				ctx.EmitCmpInt64(d49.Reg, ctx.ScratchReg)
			}
			d53 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r31, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r31, &d53)
		} else if d49.Loc == scm.LocImm {
			r32 := ctx.AllocReg()
			ctx.EmitMovRegImm64(ctx.ScratchReg, uint64(d49.Imm.Int()))
			ctx.EmitCmpInt64(ctx.ScratchReg, d51.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r32, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r32, &d53)
		} else {
			r33 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitCmpInt64(d49.Reg, d51.Reg)
			d53 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r33, Condition: scm.CondSignedGreaterOrEqual}
			ctx.BindReg(r33, &d53)
		}
		ctx.FreeDesc(&d51)
		d54 = d53
		ctx.EnsureDesc(&d54)
		if d54.Loc != scm.LocImm && d54.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d54.Loc == scm.LocImm {
			if d54.Imm.Bool() {
				return bbs[5].Render()
			}
			return bbs[7].Render()
		}
		ctx.EmitJump(d54.Condition, lbl6)
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
		}
		ctx.FreeDesc(&d53)
		ctx.FlushRegisterMoves()
		if !bbs[7].Rendered {
			snap55 := d0
			snap56 := d1
			snap57 := d2
			snap58 := d3
			snap59 := d4
			snap60 := d11
			snap61 := d12
			snap62 := d13
			snap63 := d14
			snap64 := d15
			snap65 := d27
			snap66 := d28
			snap67 := d29
			snap68 := d30
			snap69 := d31
			snap70 := d32
			snap71 := d33
			snap72 := d34
			snap73 := d35
			snap74 := d36
			snap75 := d37
			snap76 := d38
			snap77 := d39
			snap78 := d40
			snap79 := d41
			snap80 := d42
			snap81 := d43
			snap82 := d44
			snap83 := d45
			snap84 := d46
			snap85 := d47
			snap86 := d48
			snap87 := d49
			snap88 := d50
			snap89 := d51
			snap90 := d52
			snap91 := d53
			snap92 := d54
			alloc93 := ctx.SnapshotAllocState()
			bbs[7].Render()
			ctx.RestoreAllocState(alloc93)
			d0 = snap55
			d1 = snap56
			d2 = snap57
			d3 = snap58
			d4 = snap59
			d11 = snap60
			d12 = snap61
			d13 = snap62
			d14 = snap63
			d15 = snap64
			d27 = snap65
			d28 = snap66
			d29 = snap67
			d30 = snap68
			d31 = snap69
			d32 = snap70
			d33 = snap71
			d34 = snap72
			d35 = snap73
			d36 = snap74
			d37 = snap75
			d38 = snap76
			d39 = snap77
			d40 = snap78
			d41 = snap79
			d42 = snap80
			d43 = snap81
			d44 = snap82
			d45 = snap83
			d46 = snap84
			d47 = snap85
			d48 = snap86
			d49 = snap87
			d50 = snap88
			d51 = snap89
			d52 = snap90
			d53 = snap91
			d54 = snap92
		}
		if !bbs[5].Rendered {
			return bbs[5].Render()
		}
		return result
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
		d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewString("prefix index out of range")}
		ctx.EnsureDesc(&d94)
		ctx.EnsureDesc(&d94)
		if d94.Loc == scm.LocImm {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			if d94.Imm.GetTag() == scm.TagBool {
				ctx.EmitMakeBool(tmpPair, d94)
			} else if d94.Imm.GetTag() == scm.TagInt {
				ctx.EmitMakeInt(tmpPair, d94)
			} else if d94.Imm.GetTag() == scm.TagFloat {
				ctx.EmitMakeFloat(tmpPair, d94)
			} else if d94.Imm.GetTag() == scm.TagNil {
				ctx.EmitMakeNil(tmpPair)
			} else {
				ptrWord, auxWord := d94.Imm.RawWords()
				ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
				ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
			}
			d94 = tmpPair
		} else if d94.Loc == scm.LocReg {
			tmpPair := scm.JITValueDesc{Loc: scm.LocRegPair, Type: d94.Type, Reg: ctx.AllocRegExcept(d94.Reg), Reg2: ctx.AllocRegExcept(d94.Reg)}
			switch d94.Type {
			case scm.TagBool:
				ctx.EmitMakeBool(tmpPair, d94)
			case scm.TagInt:
				ctx.EmitMakeInt(tmpPair, d94)
			case scm.TagFloat:
				ctx.EmitMakeFloat(tmpPair, d94)
			default:
				panic("jit: panic arg scalar type unknown for scm.Scmer pair")
			}
			ctx.FreeDesc(&d94)
			d94 = tmpPair
		}
		if d94.Loc != scm.LocRegPair && d94.Loc != scm.LocStackPair && d94.Loc != scm.LocInputPair {
			panic("jit: panic arg expects scm.Scmer pair")
		}
		ctx.EmitGoCallVoid(scm.GoFuncAddr(scm.JITPanic), []scm.JITValueDesc{d94})
		ctx.FreeDesc(&d94)
		return result
	}
	bbs[6].Render = func() scm.JITValueDesc {
		if bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
			return result
		}
		bbs[6].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_6 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl7)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		if thisptr.Loc == scm.LocImm {
			fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
			r34 := ctx.AllocReg()
			r35 := ctx.AllocRegExcept(r34)
			r36 := ctx.AllocRegExcept(r34, r35)
			ctx.EmitMovRegMem64(r34, fieldAddr)
			ctx.EmitMovRegMem64(r35, fieldAddr+8)
			ctx.EmitMovRegMem64(r36, fieldAddr+16)
			d95 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r34, Reg2: r35, Reg3: r36}
			ctx.BindReg(r34, &d95)
			ctx.BindReg(r35, &d95)
			ctx.BindReg(r36, &d95)
		} else {
			off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
			r37 := ctx.AllocReg()
			r38 := ctx.AllocRegExcept(r37)
			r39 := ctx.AllocRegExcept(r37, r38)
			ctx.EmitMovRegMem(r37, thisptr.Reg, off)
			ctx.EmitMovRegMem(r38, thisptr.Reg, off+8)
			ctx.EmitMovRegMem(r39, thisptr.Reg, off+16)
			d95 = scm.JITValueDesc{Loc: scm.LocRegTriple, Reg: r37, Reg2: r38, Reg3: r39}
			ctx.BindReg(r37, &d95)
			ctx.BindReg(r38, &d95)
			ctx.BindReg(r39, &d95)
		}
		ctx.EnsureDesc(&d49)
		d97 = ctx.EmitSliceElementAddress(&d95, &d49, 16)
		ctx.EnsureDesc(&d97)
		r40 := ctx.AllocRegExcept(d97.Reg)
		ctx.EmitMovRegMem(r40, d97.Reg, 8)
		ctx.EmitMovRegMem(d97.Reg, d97.Reg, 0)
		d96 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: d97.Reg, Reg2: r40}
		ctx.BindReg(d97.Reg, &d96)
		ctx.BindReg(r40, &d96)
		d99 = d1
		ctx.SyncDesc(&d99)
		if d99.Loc == scm.LocMem {
			tmpScalar := scm.JITValueDesc{Loc: scm.LocReg, Type: d99.Type, Reg: ctx.AllocReg()}
			scratch := ctx.AllocRegExcept(tmpScalar.Reg)
			ctx.EmitMovRegImm64(scratch, uint64(d99.MemPtr))
			ctx.EmitMovRegMem(tmpScalar.Reg, scratch, 0)
			ctx.FreeReg(scratch)
			ctx.BindReg(tmpScalar.Reg, &tmpScalar)
			d99 = tmpScalar
		}
		d99 = scm.JITPrepareScmerGoArg(ctx, d99)
		if d99.Loc != scm.LocRegPair && d99.Loc != scm.LocStackPair && d99.Loc != scm.LocInputPair {
			panic("jit: scm.Scmer.String receiver not materialized as pair")
		}
		d98 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d99}, 2)
		ctx.EnsureDesc(&d96)
		ctx.EnsureDesc(&d98)
		d100 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d96, d98}, 2)
		ctx.FreeDesc(&d96)
		ctx.EnsureDesc(&d100)
		d101 = result
		d102 = ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d100}, 2)
		ctx.EmitMovPairToResult(&d102, &d101)
		ctx.EmitJmp(lbl0)
		return result
	}
	bbs[7].Render = func() scm.JITValueDesc {
		if bbs[7].Rendered {
			ctx.EmitJmp(lbl8)
			return result
		}
		bbs[7].Rendered = true
		ctx.FlushRegisterMoves()
		bbpos_0_7 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
		ctx.MarkLabel(lbl8)
		ctx.ResolveFixups()
		ctx.ReclaimUntrackedRegs()
		ctx.EnsureDesc(&d49)
		if d49.Loc == scm.LocImm {
			d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d49.Imm.Int() < 0)}
		} else {
			r41 := ctx.AllocRegExcept(d49.Reg)
			ctx.EmitCmpRegImm32(d49.Reg, 0)
			d103 = scm.JITValueDesc{Loc: scm.LocFlags, Type: scm.TagBool, Reg: r41, Condition: scm.CondSignedLess}
			ctx.BindReg(r41, &d103)
		}
		d104 = d103
		ctx.EnsureDesc(&d104)
		if d104.Loc != scm.LocImm && d104.Loc != scm.LocFlags {
			panic("jit: fused If condition is neither scm.LocImm nor scm.LocFlags")
		}
		if d104.Loc == scm.LocImm {
			if d104.Imm.Bool() {
				return bbs[5].Render()
			}
			return bbs[6].Render()
		}
		ctx.EmitJump(d104.Condition, lbl6)
		if bbs[6].Rendered {
			ctx.EmitJmp(lbl7)
		}
		ctx.FreeDesc(&d103)
		ctx.FlushRegisterMoves()
		if !bbs[6].Rendered {
			snap105 := d0
			snap106 := d1
			snap107 := d2
			snap108 := d3
			snap109 := d4
			snap110 := d11
			snap111 := d12
			snap112 := d13
			snap113 := d14
			snap114 := d15
			snap115 := d27
			snap116 := d28
			snap117 := d29
			snap118 := d30
			snap119 := d31
			snap120 := d32
			snap121 := d33
			snap122 := d34
			snap123 := d35
			snap124 := d36
			snap125 := d37
			snap126 := d38
			snap127 := d39
			snap128 := d40
			snap129 := d41
			snap130 := d42
			snap131 := d43
			snap132 := d44
			snap133 := d45
			snap134 := d46
			snap135 := d47
			snap136 := d48
			snap137 := d49
			snap138 := d50
			snap139 := d51
			snap140 := d52
			snap141 := d53
			snap142 := d54
			snap143 := d94
			snap144 := d95
			snap145 := d96
			snap146 := d97
			snap147 := d98
			snap148 := d99
			snap149 := d100
			snap150 := d101
			snap151 := d102
			snap152 := d103
			snap153 := d104
			alloc154 := ctx.SnapshotAllocState()
			bbs[6].Render()
			ctx.RestoreAllocState(alloc154)
			d0 = snap105
			d1 = snap106
			d2 = snap107
			d3 = snap108
			d4 = snap109
			d11 = snap110
			d12 = snap111
			d13 = snap112
			d14 = snap113
			d15 = snap114
			d27 = snap115
			d28 = snap116
			d29 = snap117
			d30 = snap118
			d31 = snap119
			d32 = snap120
			d33 = snap121
			d34 = snap122
			d35 = snap123
			d36 = snap124
			d37 = snap125
			d38 = snap126
			d39 = snap127
			d40 = snap128
			d41 = snap129
			d42 = snap130
			d43 = snap131
			d44 = snap132
			d45 = snap133
			d46 = snap134
			d47 = snap135
			d48 = snap136
			d49 = snap137
			d50 = snap138
			d51 = snap139
			d52 = snap140
			d53 = snap141
			d54 = snap142
			d94 = snap143
			d95 = snap144
			d96 = snap145
			d97 = snap146
			d98 = snap147
			d99 = snap148
			d100 = snap149
			d101 = snap150
			d102 = snap151
			d103 = snap152
			d104 = snap153
		}
		if !bbs[5].Rendered {
			return bbs[5].Render()
		}
		return result
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
