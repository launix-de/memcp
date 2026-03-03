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
	// data
	recordId,
	start,
	stride StorageInt
	count    uint   // number of values
	seqCount uint32 // number of sequences

	// analysis (lastValue also used as atomic pivot cache for concurrent GetValue)
	lastValue      atomic.Int64
	lastStride     int64
	lastValueNil   bool
	lastValueFirst bool
}

func (s *StorageSeq) ComputeSize() uint {
	return s.recordId.ComputeSize() + s.start.ComputeSize() + s.stride.ComputeSize() + 8*8
}

func (s *StorageSeq) String() string {
	return fmt.Sprintf("seq[%dx %s/%s]", s.seqCount, s.start.String(), s.stride.String())
}

func (s *StorageSeq) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(11)) // 11 = StorageSeq
	io.WriteString(f, "1234567")                    // dummy
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.seqCount))
	s.recordId.Serialize(f)
	s.start.Serialize(f)
	s.stride.Serialize(f)
}

func (s *StorageSeq) Deserialize(f io.Reader) uint {
	var dummy [7]byte
	f.Read(dummy[:])
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

func (s *StorageSeq) GetCachedReader() ColumnReader { return s }

func (s *StorageSeq) GetValue(i uint32) scm.Scmer {
	// bisect to the correct index where to find (lowest idx to find our sequence)
	pivot := uint32(s.lastValue.Load()) // atomic pivot cache for concurrent access
	if s.seqCount == 0 {
		return scm.NewNil()
	}
	if pivot >= s.seqCount {
		pivot = s.seqCount - 1
	}
	min := uint32(0)
	max := s.seqCount - 1
	for {
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			if pivot == 0 {
				min, max = 0, 0
				break
			}
			max = pivot - 1
			pivot--
		} else {
			min = pivot
			pivot++
			if pivot >= s.seqCount {
				pivot = s.seqCount - 1
			}
		}
		if min == max {
			break // we found the sequence for i
		}

		// also read the next neighbour (we are in the cache line anyway and we achieve O(1) in case the same sequence is read again!)
		recid = int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			if pivot == 0 {
				min, max = 0, 0
				break
			}
			max = pivot - 1
		} else {
			min = pivot
		}
		if min == max {
			break // we found the sequence for i
		}
		pivot = (min + max) / 2
	}

	// remember match for next time
	s.lastValue.Store(int64(min))

	var value, stride int64
	value = int64(s.start.GetValueUInt(min)) + s.start.offset
	if s.start.hasNull && value == int64(s.start.null) {
		return scm.NewNil()
	}
	stride = int64(s.stride.GetValueUInt(min)) + s.stride.offset
	recid := int64(s.recordId.GetValueUInt(min)) + s.recordId.offset
	return scm.NewFloat(float64(value + int64(int64(i)-recid)*stride))

}
func (s *StorageSeq) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
		if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&idxInt)
		}
		if idxInt.Loc != scm.LocReg {
			panic("jit: idxInt not in register")
		}
		ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
		ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
		ctx.BindReg(idxInt.Reg, &idxInt)
	}
	idxPinned := idxInt.Loc == scm.LocReg
	idxPinnedReg := idxInt.Reg
	if idxPinned {
		ctx.ProtectReg(idxPinnedReg)
	}
	r0 := ctx.W.EmitSubRSP32Fixup()
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
	}
	lbl0 := ctx.W.ReserveLabel()
	r1 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
		ctx.W.EmitMovRegMem64(r1, fieldAddr)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
		ctx.W.EmitMovRegMem(r1, thisptr.Reg, off)
	}
	d0 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
	ctx.BindReg(r1, &d0)
	if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d0)
	}
	if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d0)
	}
	var d1 scm.JITValueDesc
	if d0.Loc == scm.LocImm {
		d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d0.Imm.Int()))))}
	} else {
		r2 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r2, d0.Reg)
		ctx.W.EmitShlRegImm8(r2, 32)
		ctx.W.EmitShrRegImm8(r2, 32)
		d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
		ctx.BindReg(r2, &d1)
	}
	ctx.FreeDesc(&d0)
	var d2 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
		r3 := ctx.AllocReg()
		ctx.W.EmitMovRegMem32(r3, fieldAddr)
		d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
		ctx.BindReg(r3, &d2)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
		r4 := ctx.AllocReg()
		ctx.W.EmitMovRegMemL(r4, thisptr.Reg, off)
		d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
		ctx.BindReg(r4, &d2)
	}
	if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d2)
	}
	var d3 scm.JITValueDesc
	if d2.Loc == scm.LocImm {
		d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
	} else {
		scratch := ctx.AllocRegExcept(d2.Reg)
		ctx.W.EmitMovRegReg(scratch, d2.Reg)
		ctx.W.EmitSubRegImm32(scratch, int32(1))
		d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d3)
	}
	if d3.Loc == scm.LocImm {
		d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: d3.Type, Imm: scm.NewInt(int64(uint64(d3.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d3.Reg, 32)
		ctx.W.EmitShrRegImm8(d3.Reg, 32)
	}
	if d3.Loc == scm.LocReg && d2.Loc == scm.LocReg && d3.Reg == d2.Reg {
		ctx.TransferReg(d2.Reg)
		d2.Loc = scm.LocNone
	}
	lbl1 := ctx.W.ReserveLabel()
	d4 := d1
	if d4.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d4)
	}
	d5 := d4
	if d5.Loc == scm.LocImm {
		d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: d5.Type, Imm: scm.NewInt(int64(uint64(d5.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d5.Reg, 32)
		ctx.W.EmitShrRegImm8(d5.Reg, 32)
	}
	ctx.EmitStoreToStack(d5, 0)
	ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 8)
	d6 := d3
	if d6.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d6)
	}
	d7 := d6
	if d7.Loc == scm.LocImm {
		d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: d7.Type, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d7.Reg, 32)
		ctx.W.EmitShrRegImm8(d7.Reg, 32)
	}
	ctx.EmitStoreToStack(d7, 16)
	ctx.W.MarkLabel(lbl1)
	d8 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
	d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
	d10 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	r5 := d8.Loc == scm.LocReg
	r6 := d8.Reg
	if r5 {
		ctx.ProtectReg(r6)
	}
	lbl2 := ctx.W.ReserveLabel()
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	var d11 scm.JITValueDesc
	if d8.Loc == scm.LocImm {
		d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d8.Imm.Int()))))}
	} else {
		r7 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r7, d8.Reg)
		ctx.W.EmitShlRegImm8(r7, 32)
		ctx.W.EmitShrRegImm8(r7, 32)
		d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
		ctx.BindReg(r7, &d11)
	}
	var d12 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r8 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r8, thisptr.Reg, off)
		d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
		ctx.BindReg(r8, &d12)
	}
	if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d12)
	}
	if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d12)
	}
	var d13 scm.JITValueDesc
	if d12.Loc == scm.LocImm {
		d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
	} else {
		r9 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r9, d12.Reg)
		ctx.W.EmitShlRegImm8(r9, 56)
		ctx.W.EmitShrRegImm8(r9, 56)
		d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
		ctx.BindReg(r9, &d13)
	}
	ctx.FreeDesc(&d12)
	if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d11)
	}
	if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d13)
	}
	var d14 scm.JITValueDesc
	if d11.Loc == scm.LocImm && d13.Loc == scm.LocImm {
		d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() * d13.Imm.Int())}
	} else if d11.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d13.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d13.Reg)
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d14)
	} else if d13.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d11.Reg)
		ctx.W.EmitMovRegReg(scratch, d11.Reg)
		if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d13.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d14)
	} else {
		r10 := ctx.AllocRegExcept(d11.Reg)
		ctx.W.EmitMovRegReg(r10, d11.Reg)
		ctx.W.EmitImulInt64(r10, d13.Reg)
		d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
		ctx.BindReg(r10, &d14)
	}
	if d14.Loc == scm.LocReg && d11.Loc == scm.LocReg && d14.Reg == d11.Reg {
		ctx.TransferReg(d11.Reg)
		d11.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d11)
	ctx.FreeDesc(&d13)
	var d15 scm.JITValueDesc
	r11 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
		dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
		sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
		ctx.W.EmitMovRegImm64(r11, uint64(dataPtr))
		d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11, StackOff: int32(sliceLen)}
		ctx.BindReg(r11, &d15)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
		ctx.W.EmitMovRegMem(r11, thisptr.Reg, off)
		d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
		ctx.BindReg(r11, &d15)
	}
	ctx.BindReg(r11, &d15)
	if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d14)
	}
	var d16 scm.JITValueDesc
	if d14.Loc == scm.LocImm {
		d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
	} else {
		r12 := ctx.AllocRegExcept(d14.Reg)
		ctx.W.EmitMovRegReg(r12, d14.Reg)
		ctx.W.EmitShrRegImm8(r12, 6)
		d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
		ctx.BindReg(r12, &d16)
	}
	if d16.Loc == scm.LocReg && d14.Loc == scm.LocReg && d16.Reg == d14.Reg {
		ctx.TransferReg(d14.Reg)
		d14.Loc = scm.LocNone
	}
	if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d16)
	}
	r13 := ctx.AllocReg()
	if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d16)
	}
	if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d15)
	}
	if d16.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r13, uint64(d16.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r13, d16.Reg)
		ctx.W.EmitShlRegImm8(r13, 3)
	}
	if d15.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
		ctx.W.EmitAddInt64(r13, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r13, d15.Reg)
	}
	r14 := ctx.AllocRegExcept(r13)
	ctx.W.EmitMovRegMem(r14, r13, 0)
	ctx.FreeReg(r13)
	d17 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
	ctx.BindReg(r14, &d17)
	ctx.FreeDesc(&d16)
	if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d14)
	}
	var d18 scm.JITValueDesc
	if d14.Loc == scm.LocImm {
		d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
	} else {
		r15 := ctx.AllocRegExcept(d14.Reg)
		ctx.W.EmitMovRegReg(r15, d14.Reg)
		ctx.W.EmitAndRegImm32(r15, 63)
		d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
		ctx.BindReg(r15, &d18)
	}
	if d18.Loc == scm.LocReg && d14.Loc == scm.LocReg && d18.Reg == d14.Reg {
		ctx.TransferReg(d14.Reg)
		d14.Loc = scm.LocNone
	}
	if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d17)
	}
	if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d18)
	}
	var d19 scm.JITValueDesc
	if d17.Loc == scm.LocImm && d18.Loc == scm.LocImm {
		d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) << uint64(d18.Imm.Int())))}
	} else if d18.Loc == scm.LocImm {
		r16 := ctx.AllocRegExcept(d17.Reg)
		ctx.W.EmitMovRegReg(r16, d17.Reg)
		ctx.W.EmitShlRegImm8(r16, uint8(d18.Imm.Int()))
		d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
		ctx.BindReg(r16, &d19)
	} else {
		{
			shiftSrc := d17.Reg
			r17 := ctx.AllocRegExcept(d17.Reg)
			ctx.W.EmitMovRegReg(r17, d17.Reg)
			shiftSrc = r17
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d18.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d18.Reg)
			}
			ctx.W.EmitShlRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d19)
		}
	}
	if d19.Loc == scm.LocReg && d17.Loc == scm.LocReg && d19.Reg == d17.Reg {
		ctx.TransferReg(d17.Reg)
		d17.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d17)
	ctx.FreeDesc(&d18)
	var d20 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
		r18 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
		d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
		ctx.BindReg(r18, &d20)
	}
	lbl3 := ctx.W.ReserveLabel()
	lbl4 := ctx.W.ReserveLabel()
	lbl5 := ctx.W.ReserveLabel()
	if d20.Loc == scm.LocImm {
		if d20.Imm.Bool() {
			ctx.W.EmitJmp(lbl3)
		} else {
			d21 := d19
			if d21.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d21)
			}
			ctx.EmitStoreToStack(d21, 72)
			ctx.W.EmitJmp(lbl4)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d20.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl5)
		d22 := d19
		if d22.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d22)
		}
		ctx.EmitStoreToStack(d22, 72)
		ctx.W.EmitJmp(lbl4)
		ctx.W.MarkLabel(lbl5)
		ctx.W.EmitJmp(lbl3)
	}
	ctx.FreeDesc(&d20)
	ctx.W.MarkLabel(lbl4)
	d23 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
	var d24 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r19 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
		d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
		ctx.BindReg(r19, &d24)
	}
	if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d24)
	}
	if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d24)
	}
	var d25 scm.JITValueDesc
	if d24.Loc == scm.LocImm {
		d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
	} else {
		r20 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r20, d24.Reg)
		ctx.W.EmitShlRegImm8(r20, 56)
		ctx.W.EmitShrRegImm8(r20, 56)
		d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
		ctx.BindReg(r20, &d25)
	}
	ctx.FreeDesc(&d24)
	d26 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d25)
	}
	var d27 scm.JITValueDesc
	if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
		d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
	} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
		r21 := ctx.AllocRegExcept(d26.Reg)
		ctx.W.EmitMovRegReg(r21, d26.Reg)
		d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
		ctx.BindReg(r21, &d27)
	} else if d26.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d25.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d25.Reg)
		d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d27)
	} else if d25.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d26.Reg)
		ctx.W.EmitMovRegReg(scratch, d26.Reg)
		if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d27)
	} else {
		r22 := ctx.AllocRegExcept(d26.Reg)
		ctx.W.EmitMovRegReg(r22, d26.Reg)
		ctx.W.EmitSubInt64(r22, d25.Reg)
		d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
		ctx.BindReg(r22, &d27)
	}
	if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
		ctx.TransferReg(d26.Reg)
		d26.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d25)
	if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d23)
	}
	if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d27)
	}
	var d28 scm.JITValueDesc
	if d23.Loc == scm.LocImm && d27.Loc == scm.LocImm {
		d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d27.Imm.Int())))}
	} else if d27.Loc == scm.LocImm {
		r23 := ctx.AllocRegExcept(d23.Reg)
		ctx.W.EmitMovRegReg(r23, d23.Reg)
		ctx.W.EmitShrRegImm8(r23, uint8(d27.Imm.Int()))
		d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
		ctx.BindReg(r23, &d28)
	} else {
		{
			shiftSrc := d23.Reg
			r24 := ctx.AllocRegExcept(d23.Reg)
			ctx.W.EmitMovRegReg(r24, d23.Reg)
			shiftSrc = r24
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d27.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d27.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d27.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d28)
		}
	}
	if d28.Loc == scm.LocReg && d23.Loc == scm.LocReg && d28.Reg == d23.Reg {
		ctx.TransferReg(d23.Reg)
		d23.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d23)
	ctx.FreeDesc(&d27)
	r25 := ctx.AllocReg()
	if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d28)
	}
	ctx.EmitMovToReg(r25, d28)
	ctx.W.EmitJmp(lbl2)
	ctx.W.MarkLabel(lbl3)
	if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d14)
	}
	var d29 scm.JITValueDesc
	if d14.Loc == scm.LocImm {
		d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
	} else {
		r26 := ctx.AllocRegExcept(d14.Reg)
		ctx.W.EmitMovRegReg(r26, d14.Reg)
		ctx.W.EmitAndRegImm32(r26, 63)
		d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
		ctx.BindReg(r26, &d29)
	}
	if d29.Loc == scm.LocReg && d14.Loc == scm.LocReg && d29.Reg == d14.Reg {
		ctx.TransferReg(d14.Reg)
		d14.Loc = scm.LocNone
	}
	var d30 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r27 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r27, thisptr.Reg, off)
		d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
		ctx.BindReg(r27, &d30)
	}
	if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d30)
	}
	if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d30)
	}
	var d31 scm.JITValueDesc
	if d30.Loc == scm.LocImm {
		d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d30.Imm.Int()))))}
	} else {
		r28 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r28, d30.Reg)
		ctx.W.EmitShlRegImm8(r28, 56)
		ctx.W.EmitShrRegImm8(r28, 56)
		d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
		ctx.BindReg(r28, &d31)
	}
	ctx.FreeDesc(&d30)
	if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d29)
	}
	if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d31)
	}
	var d32 scm.JITValueDesc
	if d29.Loc == scm.LocImm && d31.Loc == scm.LocImm {
		d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + d31.Imm.Int())}
	} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
		r29 := ctx.AllocRegExcept(d29.Reg)
		ctx.W.EmitMovRegReg(r29, d29.Reg)
		d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
		ctx.BindReg(r29, &d32)
	} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
		d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
		ctx.BindReg(d31.Reg, &d32)
	} else if d29.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d31.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d31.Reg)
		d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d32)
	} else if d31.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d29.Reg)
		ctx.W.EmitMovRegReg(scratch, d29.Reg)
		if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d31.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d32)
	} else {
		r30 := ctx.AllocRegExcept(d29.Reg)
		ctx.W.EmitMovRegReg(r30, d29.Reg)
		ctx.W.EmitAddInt64(r30, d31.Reg)
		d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
		ctx.BindReg(r30, &d32)
	}
	if d32.Loc == scm.LocReg && d29.Loc == scm.LocReg && d32.Reg == d29.Reg {
		ctx.TransferReg(d29.Reg)
		d29.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d29)
	ctx.FreeDesc(&d31)
	if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d32)
	}
	var d33 scm.JITValueDesc
	if d32.Loc == scm.LocImm {
		d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d32.Imm.Int()) > uint64(64))}
	} else {
		r31 := ctx.AllocRegExcept(d32.Reg)
		ctx.W.EmitCmpRegImm32(d32.Reg, 64)
		ctx.W.EmitSetcc(r31, scm.CcA)
		d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
		ctx.BindReg(r31, &d33)
	}
	ctx.FreeDesc(&d32)
	lbl6 := ctx.W.ReserveLabel()
	lbl7 := ctx.W.ReserveLabel()
	if d33.Loc == scm.LocImm {
		if d33.Imm.Bool() {
			ctx.W.EmitJmp(lbl6)
		} else {
			d34 := d19
			if d34.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d34)
			}
			ctx.EmitStoreToStack(d34, 72)
			ctx.W.EmitJmp(lbl4)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d33.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl7)
		d35 := d19
		if d35.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d35)
		}
		ctx.EmitStoreToStack(d35, 72)
		ctx.W.EmitJmp(lbl4)
		ctx.W.MarkLabel(lbl7)
		ctx.W.EmitJmp(lbl6)
	}
	ctx.FreeDesc(&d33)
	ctx.W.MarkLabel(lbl6)
	if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d14)
	}
	var d36 scm.JITValueDesc
	if d14.Loc == scm.LocImm {
		d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
	} else {
		r32 := ctx.AllocRegExcept(d14.Reg)
		ctx.W.EmitMovRegReg(r32, d14.Reg)
		ctx.W.EmitShrRegImm8(r32, 6)
		d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
		ctx.BindReg(r32, &d36)
	}
	if d36.Loc == scm.LocReg && d14.Loc == scm.LocReg && d36.Reg == d14.Reg {
		ctx.TransferReg(d14.Reg)
		d14.Loc = scm.LocNone
	}
	if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d36)
	}
	var d37 scm.JITValueDesc
	if d36.Loc == scm.LocImm {
		d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d36.Reg)
		ctx.W.EmitMovRegReg(scratch, d36.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d37)
	}
	if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
		ctx.TransferReg(d36.Reg)
		d36.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d36)
	if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d37)
	}
	r33 := ctx.AllocReg()
	if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d37)
	}
	if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d15)
	}
	if d37.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r33, uint64(d37.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r33, d37.Reg)
		ctx.W.EmitShlRegImm8(r33, 3)
	}
	if d15.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
		ctx.W.EmitAddInt64(r33, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r33, d15.Reg)
	}
	r34 := ctx.AllocRegExcept(r33)
	ctx.W.EmitMovRegMem(r34, r33, 0)
	ctx.FreeReg(r33)
	d38 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
	ctx.BindReg(r34, &d38)
	ctx.FreeDesc(&d37)
	if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d14)
	}
	var d39 scm.JITValueDesc
	if d14.Loc == scm.LocImm {
		d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
	} else {
		r35 := ctx.AllocRegExcept(d14.Reg)
		ctx.W.EmitMovRegReg(r35, d14.Reg)
		ctx.W.EmitAndRegImm32(r35, 63)
		d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
		ctx.BindReg(r35, &d39)
	}
	if d39.Loc == scm.LocReg && d14.Loc == scm.LocReg && d39.Reg == d14.Reg {
		ctx.TransferReg(d14.Reg)
		d14.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d14)
	d40 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d39)
	}
	var d41 scm.JITValueDesc
	if d40.Loc == scm.LocImm && d39.Loc == scm.LocImm {
		d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() - d39.Imm.Int())}
	} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
		r36 := ctx.AllocRegExcept(d40.Reg)
		ctx.W.EmitMovRegReg(r36, d40.Reg)
		d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
		ctx.BindReg(r36, &d41)
	} else if d40.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d39.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d39.Reg)
		d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d41)
	} else if d39.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d40.Reg)
		ctx.W.EmitMovRegReg(scratch, d40.Reg)
		if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d39.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d41)
	} else {
		r37 := ctx.AllocRegExcept(d40.Reg)
		ctx.W.EmitMovRegReg(r37, d40.Reg)
		ctx.W.EmitSubInt64(r37, d39.Reg)
		d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
		ctx.BindReg(r37, &d41)
	}
	if d41.Loc == scm.LocReg && d40.Loc == scm.LocReg && d41.Reg == d40.Reg {
		ctx.TransferReg(d40.Reg)
		d40.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d39)
	if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d38)
	}
	if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d41)
	}
	var d42 scm.JITValueDesc
	if d38.Loc == scm.LocImm && d41.Loc == scm.LocImm {
		d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) >> uint64(d41.Imm.Int())))}
	} else if d41.Loc == scm.LocImm {
		r38 := ctx.AllocRegExcept(d38.Reg)
		ctx.W.EmitMovRegReg(r38, d38.Reg)
		ctx.W.EmitShrRegImm8(r38, uint8(d41.Imm.Int()))
		d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
		ctx.BindReg(r38, &d42)
	} else {
		{
			shiftSrc := d38.Reg
			r39 := ctx.AllocRegExcept(d38.Reg)
			ctx.W.EmitMovRegReg(r39, d38.Reg)
			shiftSrc = r39
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d41.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d41.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d41.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d42)
		}
	}
	if d42.Loc == scm.LocReg && d38.Loc == scm.LocReg && d42.Reg == d38.Reg {
		ctx.TransferReg(d38.Reg)
		d38.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d38)
	ctx.FreeDesc(&d41)
	if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d19)
	}
	if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d42)
	}
	var d43 scm.JITValueDesc
	if d19.Loc == scm.LocImm && d42.Loc == scm.LocImm {
		d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() | d42.Imm.Int())}
	} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
		d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
		ctx.BindReg(d42.Reg, &d43)
	} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
		r40 := ctx.AllocRegExcept(d19.Reg)
		ctx.W.EmitMovRegReg(r40, d19.Reg)
		d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
		ctx.BindReg(r40, &d43)
	} else if d19.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d42.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
		ctx.W.EmitOrInt64(scratch, d42.Reg)
		d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d43)
	} else if d42.Loc == scm.LocImm {
		r41 := ctx.AllocRegExcept(d19.Reg)
		ctx.W.EmitMovRegReg(r41, d19.Reg)
		if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
			ctx.W.EmitOrRegImm32(r41, int32(d42.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
			ctx.W.EmitOrInt64(r41, scm.RegR11)
		}
		d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
		ctx.BindReg(r41, &d43)
	} else {
		r42 := ctx.AllocRegExcept(d19.Reg)
		ctx.W.EmitMovRegReg(r42, d19.Reg)
		ctx.W.EmitOrInt64(r42, d42.Reg)
		d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
		ctx.BindReg(r42, &d43)
	}
	if d43.Loc == scm.LocReg && d19.Loc == scm.LocReg && d43.Reg == d19.Reg {
		ctx.TransferReg(d19.Reg)
		d19.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d42)
	d44 := d43
	if d44.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d44)
	}
	ctx.EmitStoreToStack(d44, 72)
	ctx.W.EmitJmp(lbl4)
	ctx.W.MarkLabel(lbl2)
	ctx.W.ResolveFixups()
	d45 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
	ctx.BindReg(r25, &d45)
	ctx.BindReg(r25, &d45)
	if r5 {
		ctx.UnprotectReg(r6)
	}
	if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d45)
	}
	if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d45)
	}
	var d46 scm.JITValueDesc
	if d45.Loc == scm.LocImm {
		d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d45.Imm.Int()))))}
	} else {
		r43 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r43, d45.Reg)
		d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
		ctx.BindReg(r43, &d46)
	}
	ctx.FreeDesc(&d45)
	var d47 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
		val := *(*int64)(unsafe.Pointer(fieldAddr))
		d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
		r44 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r44, thisptr.Reg, off)
		d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
		ctx.BindReg(r44, &d47)
	}
	if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d46)
	}
	if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d47)
	}
	var d48 scm.JITValueDesc
	if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
		d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() + d47.Imm.Int())}
	} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
		r45 := ctx.AllocRegExcept(d46.Reg)
		ctx.W.EmitMovRegReg(r45, d46.Reg)
		d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
		ctx.BindReg(r45, &d48)
	} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
		d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d47.Reg}
		ctx.BindReg(d47.Reg, &d48)
	} else if d46.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d47.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d46.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d47.Reg)
		d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d48)
	} else if d47.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d46.Reg)
		ctx.W.EmitMovRegReg(scratch, d46.Reg)
		if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d47.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d48)
	} else {
		r46 := ctx.AllocRegExcept(d46.Reg)
		ctx.W.EmitMovRegReg(r46, d46.Reg)
		ctx.W.EmitAddInt64(r46, d47.Reg)
		d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
		ctx.BindReg(r46, &d48)
	}
	if d48.Loc == scm.LocReg && d46.Loc == scm.LocReg && d48.Reg == d46.Reg {
		ctx.TransferReg(d46.Reg)
		d46.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d46)
	ctx.FreeDesc(&d47)
	if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d48)
	}
	if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d48)
	}
	var d49 scm.JITValueDesc
	if d48.Loc == scm.LocImm {
		d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d48.Imm.Int()))))}
	} else {
		r47 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r47, d48.Reg)
		ctx.W.EmitShlRegImm8(r47, 32)
		ctx.W.EmitShrRegImm8(r47, 32)
		d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
		ctx.BindReg(r47, &d49)
	}
	ctx.FreeDesc(&d48)
	if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&idxInt)
	}
	if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d49)
	}
	var d50 scm.JITValueDesc
	if idxInt.Loc == scm.LocImm && d49.Loc == scm.LocImm {
		d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d49.Imm.Int()))}
	} else if d49.Loc == scm.LocImm {
		r48 := ctx.AllocRegExcept(idxInt.Reg)
		if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
			ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d49.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
			ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
		}
		ctx.W.EmitSetcc(r48, scm.CcB)
		d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
		ctx.BindReg(r48, &d50)
	} else if idxInt.Loc == scm.LocImm {
		r49 := ctx.AllocReg()
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
		ctx.W.EmitCmpInt64(scm.RegR11, d49.Reg)
		ctx.W.EmitSetcc(r49, scm.CcB)
		d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
		ctx.BindReg(r49, &d50)
	} else {
		r50 := ctx.AllocRegExcept(idxInt.Reg)
		ctx.W.EmitCmpInt64(idxInt.Reg, d49.Reg)
		ctx.W.EmitSetcc(r50, scm.CcB)
		d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
		ctx.BindReg(r50, &d50)
	}
	ctx.FreeDesc(&d49)
	lbl8 := ctx.W.ReserveLabel()
	lbl9 := ctx.W.ReserveLabel()
	lbl10 := ctx.W.ReserveLabel()
	if d50.Loc == scm.LocImm {
		if d50.Imm.Bool() {
			ctx.W.EmitJmp(lbl8)
		} else {
			ctx.W.EmitJmp(lbl9)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d50.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl10)
		ctx.W.EmitJmp(lbl9)
		ctx.W.MarkLabel(lbl10)
		ctx.W.EmitJmp(lbl8)
	}
	ctx.FreeDesc(&d50)
	ctx.W.MarkLabel(lbl9)
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	var d51 scm.JITValueDesc
	if d8.Loc == scm.LocImm {
		d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d8.Reg)
		ctx.W.EmitMovRegReg(scratch, d8.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d51)
	}
	if d51.Loc == scm.LocImm {
		d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: d51.Type, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d51.Reg, 32)
		ctx.W.EmitShrRegImm8(d51.Reg, 32)
	}
	if d51.Loc == scm.LocReg && d8.Loc == scm.LocReg && d51.Reg == d8.Reg {
		ctx.TransferReg(d8.Reg)
		d8.Loc = scm.LocNone
	}
	lbl11 := ctx.W.ReserveLabel()
	d52 := d51
	if d52.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d52)
	}
	d53 := d52
	if d53.Loc == scm.LocImm {
		d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: d53.Type, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d53.Reg, 32)
		ctx.W.EmitShrRegImm8(d53.Reg, 32)
	}
	ctx.EmitStoreToStack(d53, 32)
	d54 := d8
	if d54.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d54)
	}
	d55 := d54
	if d55.Loc == scm.LocImm {
		d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: d55.Type, Imm: scm.NewInt(int64(uint64(d55.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d55.Reg, 32)
		ctx.W.EmitShrRegImm8(d55.Reg, 32)
	}
	ctx.EmitStoreToStack(d55, 40)
	d56 := d10
	if d56.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d56.Loc == scm.LocStack || d56.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d56)
	}
	d57 := d56
	if d57.Loc == scm.LocImm {
		d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: d57.Type, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d57.Reg, 32)
		ctx.W.EmitShrRegImm8(d57.Reg, 32)
	}
	ctx.EmitStoreToStack(d57, 48)
	ctx.W.MarkLabel(lbl11)
	d58 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
	d59 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
	d60 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
	if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d59)
	}
	if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d60)
	}
	var d61 scm.JITValueDesc
	if d59.Loc == scm.LocImm && d60.Loc == scm.LocImm {
		d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d59.Imm.Int()) == uint64(d60.Imm.Int()))}
	} else if d60.Loc == scm.LocImm {
		r51 := ctx.AllocRegExcept(d59.Reg)
		if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
			ctx.W.EmitCmpRegImm32(d59.Reg, int32(d60.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
			ctx.W.EmitCmpInt64(d59.Reg, scm.RegR11)
		}
		ctx.W.EmitSetcc(r51, scm.CcE)
		d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
		ctx.BindReg(r51, &d61)
	} else if d59.Loc == scm.LocImm {
		r52 := ctx.AllocReg()
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
		ctx.W.EmitCmpInt64(scm.RegR11, d60.Reg)
		ctx.W.EmitSetcc(r52, scm.CcE)
		d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
		ctx.BindReg(r52, &d61)
	} else {
		r53 := ctx.AllocRegExcept(d59.Reg)
		ctx.W.EmitCmpInt64(d59.Reg, d60.Reg)
		ctx.W.EmitSetcc(r53, scm.CcE)
		d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
		ctx.BindReg(r53, &d61)
	}
	lbl12 := ctx.W.ReserveLabel()
	lbl13 := ctx.W.ReserveLabel()
	lbl14 := ctx.W.ReserveLabel()
	if d61.Loc == scm.LocImm {
		if d61.Imm.Bool() {
			d62 := d59
			if d62.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d62)
			}
			d63 := d62
			if d63.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: d63.Type, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d63.Reg, 32)
				ctx.W.EmitShrRegImm8(d63.Reg, 32)
			}
			ctx.EmitStoreToStack(d63, 24)
			ctx.W.EmitJmp(lbl12)
		} else {
			ctx.W.EmitJmp(lbl13)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d61.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl14)
		ctx.W.EmitJmp(lbl13)
		ctx.W.MarkLabel(lbl14)
		d64 := d59
		if d64.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d64)
		}
		d65 := d64
		if d65.Loc == scm.LocImm {
			d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: d65.Type, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.W.EmitShlRegImm8(d65.Reg, 32)
			ctx.W.EmitShrRegImm8(d65.Reg, 32)
		}
		ctx.EmitStoreToStack(d65, 24)
		ctx.W.EmitJmp(lbl12)
	}
	ctx.FreeDesc(&d61)
	ctx.W.MarkLabel(lbl8)
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	var d66 scm.JITValueDesc
	if d8.Loc == scm.LocImm {
		d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() - 1)}
	} else {
		scratch := ctx.AllocRegExcept(d8.Reg)
		ctx.W.EmitMovRegReg(scratch, d8.Reg)
		ctx.W.EmitSubRegImm32(scratch, int32(1))
		d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d66)
	}
	if d66.Loc == scm.LocImm {
		d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: d66.Type, Imm: scm.NewInt(int64(uint64(d66.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d66.Reg, 32)
		ctx.W.EmitShrRegImm8(d66.Reg, 32)
	}
	if d66.Loc == scm.LocReg && d8.Loc == scm.LocReg && d66.Reg == d8.Reg {
		ctx.TransferReg(d8.Reg)
		d8.Loc = scm.LocNone
	}
	if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d8)
	}
	var d67 scm.JITValueDesc
	if d8.Loc == scm.LocImm {
		d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() - 1)}
	} else {
		scratch := ctx.AllocRegExcept(d8.Reg)
		ctx.W.EmitMovRegReg(scratch, d8.Reg)
		ctx.W.EmitSubRegImm32(scratch, int32(1))
		d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d67)
	}
	if d67.Loc == scm.LocImm {
		d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: d67.Type, Imm: scm.NewInt(int64(uint64(d67.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d67.Reg, 32)
		ctx.W.EmitShrRegImm8(d67.Reg, 32)
	}
	if d67.Loc == scm.LocReg && d8.Loc == scm.LocReg && d67.Reg == d8.Reg {
		ctx.TransferReg(d8.Reg)
		d8.Loc = scm.LocNone
	}
	d68 := d67
	if d68.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d68.Loc == scm.LocStack || d68.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d68)
	}
	d69 := d68
	if d69.Loc == scm.LocImm {
		d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: d69.Type, Imm: scm.NewInt(int64(uint64(d69.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d69.Reg, 32)
		ctx.W.EmitShrRegImm8(d69.Reg, 32)
	}
	ctx.EmitStoreToStack(d69, 32)
	d70 := d9
	if d70.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d70.Loc == scm.LocStack || d70.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d70)
	}
	d71 := d70
	if d71.Loc == scm.LocImm {
		d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: d71.Type, Imm: scm.NewInt(int64(uint64(d71.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d71.Reg, 32)
		ctx.W.EmitShrRegImm8(d71.Reg, 32)
	}
	ctx.EmitStoreToStack(d71, 40)
	d72 := d66
	if d72.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d72)
	}
	d73 := d72
	if d73.Loc == scm.LocImm {
		d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: d73.Type, Imm: scm.NewInt(int64(uint64(d73.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d73.Reg, 32)
		ctx.W.EmitShrRegImm8(d73.Reg, 32)
	}
	ctx.EmitStoreToStack(d73, 48)
	ctx.W.EmitJmp(lbl11)
	ctx.W.MarkLabel(lbl13)
	if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d58)
	}
	r54 := d58.Loc == scm.LocReg
	r55 := d58.Reg
	if r54 {
		ctx.ProtectReg(r55)
	}
	lbl15 := ctx.W.ReserveLabel()
	if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d58)
	}
	if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d58)
	}
	var d74 scm.JITValueDesc
	if d58.Loc == scm.LocImm {
		d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d58.Imm.Int()))))}
	} else {
		r56 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r56, d58.Reg)
		ctx.W.EmitShlRegImm8(r56, 32)
		ctx.W.EmitShrRegImm8(r56, 32)
		d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
		ctx.BindReg(r56, &d74)
	}
	var d75 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r57 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r57, thisptr.Reg, off)
		d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
		ctx.BindReg(r57, &d75)
	}
	if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d75)
	}
	if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d75)
	}
	var d76 scm.JITValueDesc
	if d75.Loc == scm.LocImm {
		d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d75.Imm.Int()))))}
	} else {
		r58 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r58, d75.Reg)
		ctx.W.EmitShlRegImm8(r58, 56)
		ctx.W.EmitShrRegImm8(r58, 56)
		d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
		ctx.BindReg(r58, &d76)
	}
	ctx.FreeDesc(&d75)
	if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d74)
	}
	if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d76)
	}
	var d77 scm.JITValueDesc
	if d74.Loc == scm.LocImm && d76.Loc == scm.LocImm {
		d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() * d76.Imm.Int())}
	} else if d74.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d76.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d76.Reg)
		d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d77)
	} else if d76.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d74.Reg)
		ctx.W.EmitMovRegReg(scratch, d74.Reg)
		if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d76.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d77)
	} else {
		r59 := ctx.AllocRegExcept(d74.Reg)
		ctx.W.EmitMovRegReg(r59, d74.Reg)
		ctx.W.EmitImulInt64(r59, d76.Reg)
		d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
		ctx.BindReg(r59, &d77)
	}
	if d77.Loc == scm.LocReg && d74.Loc == scm.LocReg && d77.Reg == d74.Reg {
		ctx.TransferReg(d74.Reg)
		d74.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d74)
	ctx.FreeDesc(&d76)
	var d78 scm.JITValueDesc
	r60 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
		dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
		sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
		ctx.W.EmitMovRegImm64(r60, uint64(dataPtr))
		d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60, StackOff: int32(sliceLen)}
		ctx.BindReg(r60, &d78)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
		ctx.W.EmitMovRegMem(r60, thisptr.Reg, off)
		d78 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
		ctx.BindReg(r60, &d78)
	}
	ctx.BindReg(r60, &d78)
	if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d77)
	}
	var d79 scm.JITValueDesc
	if d77.Loc == scm.LocImm {
		d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() / 64)}
	} else {
		r61 := ctx.AllocRegExcept(d77.Reg)
		ctx.W.EmitMovRegReg(r61, d77.Reg)
		ctx.W.EmitShrRegImm8(r61, 6)
		d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
		ctx.BindReg(r61, &d79)
	}
	if d79.Loc == scm.LocReg && d77.Loc == scm.LocReg && d79.Reg == d77.Reg {
		ctx.TransferReg(d77.Reg)
		d77.Loc = scm.LocNone
	}
	if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d79)
	}
	r62 := ctx.AllocReg()
	if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d79)
	}
	if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d78)
	}
	if d79.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r62, uint64(d79.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r62, d79.Reg)
		ctx.W.EmitShlRegImm8(r62, 3)
	}
	if d78.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
		ctx.W.EmitAddInt64(r62, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r62, d78.Reg)
	}
	r63 := ctx.AllocRegExcept(r62)
	ctx.W.EmitMovRegMem(r63, r62, 0)
	ctx.FreeReg(r62)
	d80 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
	ctx.BindReg(r63, &d80)
	ctx.FreeDesc(&d79)
	if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d77)
	}
	var d81 scm.JITValueDesc
	if d77.Loc == scm.LocImm {
		d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() % 64)}
	} else {
		r64 := ctx.AllocRegExcept(d77.Reg)
		ctx.W.EmitMovRegReg(r64, d77.Reg)
		ctx.W.EmitAndRegImm32(r64, 63)
		d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
		ctx.BindReg(r64, &d81)
	}
	if d81.Loc == scm.LocReg && d77.Loc == scm.LocReg && d81.Reg == d77.Reg {
		ctx.TransferReg(d77.Reg)
		d77.Loc = scm.LocNone
	}
	if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d80)
	}
	if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d81)
	}
	var d82 scm.JITValueDesc
	if d80.Loc == scm.LocImm && d81.Loc == scm.LocImm {
		d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d80.Imm.Int()) << uint64(d81.Imm.Int())))}
	} else if d81.Loc == scm.LocImm {
		r65 := ctx.AllocRegExcept(d80.Reg)
		ctx.W.EmitMovRegReg(r65, d80.Reg)
		ctx.W.EmitShlRegImm8(r65, uint8(d81.Imm.Int()))
		d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
		ctx.BindReg(r65, &d82)
	} else {
		{
			shiftSrc := d80.Reg
			r66 := ctx.AllocRegExcept(d80.Reg)
			ctx.W.EmitMovRegReg(r66, d80.Reg)
			shiftSrc = r66
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d81.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d81.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d81.Reg)
			}
			ctx.W.EmitShlRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d82)
		}
	}
	if d82.Loc == scm.LocReg && d80.Loc == scm.LocReg && d82.Reg == d80.Reg {
		ctx.TransferReg(d80.Reg)
		d80.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d80)
	ctx.FreeDesc(&d81)
	var d83 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
		r67 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r67, thisptr.Reg, off)
		d83 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r67}
		ctx.BindReg(r67, &d83)
	}
	lbl16 := ctx.W.ReserveLabel()
	lbl17 := ctx.W.ReserveLabel()
	lbl18 := ctx.W.ReserveLabel()
	if d83.Loc == scm.LocImm {
		if d83.Imm.Bool() {
			ctx.W.EmitJmp(lbl16)
		} else {
			d84 := d82
			if d84.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d84)
			}
			ctx.EmitStoreToStack(d84, 80)
			ctx.W.EmitJmp(lbl17)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d83.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl18)
		d85 := d82
		if d85.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d85)
		}
		ctx.EmitStoreToStack(d85, 80)
		ctx.W.EmitJmp(lbl17)
		ctx.W.MarkLabel(lbl18)
		ctx.W.EmitJmp(lbl16)
	}
	ctx.FreeDesc(&d83)
	ctx.W.MarkLabel(lbl17)
	d86 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
	var d87 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r68 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
		d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
		ctx.BindReg(r68, &d87)
	}
	if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d87)
	}
	if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d87)
	}
	var d88 scm.JITValueDesc
	if d87.Loc == scm.LocImm {
		d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d87.Imm.Int()))))}
	} else {
		r69 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r69, d87.Reg)
		ctx.W.EmitShlRegImm8(r69, 56)
		ctx.W.EmitShrRegImm8(r69, 56)
		d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
		ctx.BindReg(r69, &d88)
	}
	ctx.FreeDesc(&d87)
	d89 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d88)
	}
	var d90 scm.JITValueDesc
	if d89.Loc == scm.LocImm && d88.Loc == scm.LocImm {
		d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() - d88.Imm.Int())}
	} else if d88.Loc == scm.LocImm && d88.Imm.Int() == 0 {
		r70 := ctx.AllocRegExcept(d89.Reg)
		ctx.W.EmitMovRegReg(r70, d89.Reg)
		d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
		ctx.BindReg(r70, &d90)
	} else if d89.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d88.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d88.Reg)
		d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d90)
	} else if d88.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d89.Reg)
		ctx.W.EmitMovRegReg(scratch, d89.Reg)
		if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d88.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d90)
	} else {
		r71 := ctx.AllocRegExcept(d89.Reg)
		ctx.W.EmitMovRegReg(r71, d89.Reg)
		ctx.W.EmitSubInt64(r71, d88.Reg)
		d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
		ctx.BindReg(r71, &d90)
	}
	if d90.Loc == scm.LocReg && d89.Loc == scm.LocReg && d90.Reg == d89.Reg {
		ctx.TransferReg(d89.Reg)
		d89.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d88)
	if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d86)
	}
	if d90.Loc == scm.LocStack || d90.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d90)
	}
	var d91 scm.JITValueDesc
	if d86.Loc == scm.LocImm && d90.Loc == scm.LocImm {
		d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d86.Imm.Int()) >> uint64(d90.Imm.Int())))}
	} else if d90.Loc == scm.LocImm {
		r72 := ctx.AllocRegExcept(d86.Reg)
		ctx.W.EmitMovRegReg(r72, d86.Reg)
		ctx.W.EmitShrRegImm8(r72, uint8(d90.Imm.Int()))
		d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
		ctx.BindReg(r72, &d91)
	} else {
		{
			shiftSrc := d86.Reg
			r73 := ctx.AllocRegExcept(d86.Reg)
			ctx.W.EmitMovRegReg(r73, d86.Reg)
			shiftSrc = r73
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d90.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d90.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d90.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d91)
		}
	}
	if d91.Loc == scm.LocReg && d86.Loc == scm.LocReg && d91.Reg == d86.Reg {
		ctx.TransferReg(d86.Reg)
		d86.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d86)
	ctx.FreeDesc(&d90)
	r74 := ctx.AllocReg()
	if d91.Loc == scm.LocStack || d91.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d91)
	}
	ctx.EmitMovToReg(r74, d91)
	ctx.W.EmitJmp(lbl15)
	ctx.W.MarkLabel(lbl16)
	if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d77)
	}
	var d92 scm.JITValueDesc
	if d77.Loc == scm.LocImm {
		d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() % 64)}
	} else {
		r75 := ctx.AllocRegExcept(d77.Reg)
		ctx.W.EmitMovRegReg(r75, d77.Reg)
		ctx.W.EmitAndRegImm32(r75, 63)
		d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
		ctx.BindReg(r75, &d92)
	}
	if d92.Loc == scm.LocReg && d77.Loc == scm.LocReg && d92.Reg == d77.Reg {
		ctx.TransferReg(d77.Reg)
		d77.Loc = scm.LocNone
	}
	var d93 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r76 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r76, thisptr.Reg, off)
		d93 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
		ctx.BindReg(r76, &d93)
	}
	if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d93)
	}
	if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d93)
	}
	var d94 scm.JITValueDesc
	if d93.Loc == scm.LocImm {
		d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d93.Imm.Int()))))}
	} else {
		r77 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r77, d93.Reg)
		ctx.W.EmitShlRegImm8(r77, 56)
		ctx.W.EmitShrRegImm8(r77, 56)
		d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
		ctx.BindReg(r77, &d94)
	}
	ctx.FreeDesc(&d93)
	if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d92)
	}
	if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d94)
	}
	var d95 scm.JITValueDesc
	if d92.Loc == scm.LocImm && d94.Loc == scm.LocImm {
		d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() + d94.Imm.Int())}
	} else if d94.Loc == scm.LocImm && d94.Imm.Int() == 0 {
		r78 := ctx.AllocRegExcept(d92.Reg)
		ctx.W.EmitMovRegReg(r78, d92.Reg)
		d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
		ctx.BindReg(r78, &d95)
	} else if d92.Loc == scm.LocImm && d92.Imm.Int() == 0 {
		d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d94.Reg}
		ctx.BindReg(d94.Reg, &d95)
	} else if d92.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d94.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d92.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d94.Reg)
		d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d95)
	} else if d94.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d92.Reg)
		ctx.W.EmitMovRegReg(scratch, d92.Reg)
		if d94.Imm.Int() >= -2147483648 && d94.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d94.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d95)
	} else {
		r79 := ctx.AllocRegExcept(d92.Reg)
		ctx.W.EmitMovRegReg(r79, d92.Reg)
		ctx.W.EmitAddInt64(r79, d94.Reg)
		d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
		ctx.BindReg(r79, &d95)
	}
	if d95.Loc == scm.LocReg && d92.Loc == scm.LocReg && d95.Reg == d92.Reg {
		ctx.TransferReg(d92.Reg)
		d92.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d92)
	ctx.FreeDesc(&d94)
	if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d95)
	}
	var d96 scm.JITValueDesc
	if d95.Loc == scm.LocImm {
		d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d95.Imm.Int()) > uint64(64))}
	} else {
		r80 := ctx.AllocRegExcept(d95.Reg)
		ctx.W.EmitCmpRegImm32(d95.Reg, 64)
		ctx.W.EmitSetcc(r80, scm.CcA)
		d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r80}
		ctx.BindReg(r80, &d96)
	}
	ctx.FreeDesc(&d95)
	lbl19 := ctx.W.ReserveLabel()
	lbl20 := ctx.W.ReserveLabel()
	if d96.Loc == scm.LocImm {
		if d96.Imm.Bool() {
			ctx.W.EmitJmp(lbl19)
		} else {
			d97 := d82
			if d97.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d97)
			}
			ctx.EmitStoreToStack(d97, 80)
			ctx.W.EmitJmp(lbl17)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d96.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl20)
		d98 := d82
		if d98.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d98)
		}
		ctx.EmitStoreToStack(d98, 80)
		ctx.W.EmitJmp(lbl17)
		ctx.W.MarkLabel(lbl20)
		ctx.W.EmitJmp(lbl19)
	}
	ctx.FreeDesc(&d96)
	ctx.W.MarkLabel(lbl19)
	if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d77)
	}
	var d99 scm.JITValueDesc
	if d77.Loc == scm.LocImm {
		d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() / 64)}
	} else {
		r81 := ctx.AllocRegExcept(d77.Reg)
		ctx.W.EmitMovRegReg(r81, d77.Reg)
		ctx.W.EmitShrRegImm8(r81, 6)
		d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
		ctx.BindReg(r81, &d99)
	}
	if d99.Loc == scm.LocReg && d77.Loc == scm.LocReg && d99.Reg == d77.Reg {
		ctx.TransferReg(d77.Reg)
		d77.Loc = scm.LocNone
	}
	if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d99)
	}
	var d100 scm.JITValueDesc
	if d99.Loc == scm.LocImm {
		d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d99.Reg)
		ctx.W.EmitMovRegReg(scratch, d99.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d100)
	}
	if d100.Loc == scm.LocReg && d99.Loc == scm.LocReg && d100.Reg == d99.Reg {
		ctx.TransferReg(d99.Reg)
		d99.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d99)
	if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d100)
	}
	r82 := ctx.AllocReg()
	if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d100)
	}
	if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d78)
	}
	if d100.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r82, uint64(d100.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r82, d100.Reg)
		ctx.W.EmitShlRegImm8(r82, 3)
	}
	if d78.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
		ctx.W.EmitAddInt64(r82, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r82, d78.Reg)
	}
	r83 := ctx.AllocRegExcept(r82)
	ctx.W.EmitMovRegMem(r83, r82, 0)
	ctx.FreeReg(r82)
	d101 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r83}
	ctx.BindReg(r83, &d101)
	ctx.FreeDesc(&d100)
	if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d77)
	}
	var d102 scm.JITValueDesc
	if d77.Loc == scm.LocImm {
		d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() % 64)}
	} else {
		r84 := ctx.AllocRegExcept(d77.Reg)
		ctx.W.EmitMovRegReg(r84, d77.Reg)
		ctx.W.EmitAndRegImm32(r84, 63)
		d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
		ctx.BindReg(r84, &d102)
	}
	if d102.Loc == scm.LocReg && d77.Loc == scm.LocReg && d102.Reg == d77.Reg {
		ctx.TransferReg(d77.Reg)
		d77.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d77)
	d103 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d102)
	}
	var d104 scm.JITValueDesc
	if d103.Loc == scm.LocImm && d102.Loc == scm.LocImm {
		d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() - d102.Imm.Int())}
	} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
		r85 := ctx.AllocRegExcept(d103.Reg)
		ctx.W.EmitMovRegReg(r85, d103.Reg)
		d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
		ctx.BindReg(r85, &d104)
	} else if d103.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d102.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d102.Reg)
		d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d104)
	} else if d102.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d103.Reg)
		ctx.W.EmitMovRegReg(scratch, d103.Reg)
		if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d102.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d104)
	} else {
		r86 := ctx.AllocRegExcept(d103.Reg)
		ctx.W.EmitMovRegReg(r86, d103.Reg)
		ctx.W.EmitSubInt64(r86, d102.Reg)
		d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
		ctx.BindReg(r86, &d104)
	}
	if d104.Loc == scm.LocReg && d103.Loc == scm.LocReg && d104.Reg == d103.Reg {
		ctx.TransferReg(d103.Reg)
		d103.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d102)
	if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d101)
	}
	if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d104)
	}
	var d105 scm.JITValueDesc
	if d101.Loc == scm.LocImm && d104.Loc == scm.LocImm {
		d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d101.Imm.Int()) >> uint64(d104.Imm.Int())))}
	} else if d104.Loc == scm.LocImm {
		r87 := ctx.AllocRegExcept(d101.Reg)
		ctx.W.EmitMovRegReg(r87, d101.Reg)
		ctx.W.EmitShrRegImm8(r87, uint8(d104.Imm.Int()))
		d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
		ctx.BindReg(r87, &d105)
	} else {
		{
			shiftSrc := d101.Reg
			r88 := ctx.AllocRegExcept(d101.Reg)
			ctx.W.EmitMovRegReg(r88, d101.Reg)
			shiftSrc = r88
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d104.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d104.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d104.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d105)
		}
	}
	if d105.Loc == scm.LocReg && d101.Loc == scm.LocReg && d105.Reg == d101.Reg {
		ctx.TransferReg(d101.Reg)
		d101.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d101)
	ctx.FreeDesc(&d104)
	if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d82)
	}
	if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d105)
	}
	var d106 scm.JITValueDesc
	if d82.Loc == scm.LocImm && d105.Loc == scm.LocImm {
		d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d82.Imm.Int() | d105.Imm.Int())}
	} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
		d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d105.Reg}
		ctx.BindReg(d105.Reg, &d106)
	} else if d105.Loc == scm.LocImm && d105.Imm.Int() == 0 {
		r89 := ctx.AllocRegExcept(d82.Reg)
		ctx.W.EmitMovRegReg(r89, d82.Reg)
		d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
		ctx.BindReg(r89, &d106)
	} else if d82.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d105.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d82.Imm.Int()))
		ctx.W.EmitOrInt64(scratch, d105.Reg)
		d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d106)
	} else if d105.Loc == scm.LocImm {
		r90 := ctx.AllocRegExcept(d82.Reg)
		ctx.W.EmitMovRegReg(r90, d82.Reg)
		if d105.Imm.Int() >= -2147483648 && d105.Imm.Int() <= 2147483647 {
			ctx.W.EmitOrRegImm32(r90, int32(d105.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d105.Imm.Int()))
			ctx.W.EmitOrInt64(r90, scm.RegR11)
		}
		d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
		ctx.BindReg(r90, &d106)
	} else {
		r91 := ctx.AllocRegExcept(d82.Reg)
		ctx.W.EmitMovRegReg(r91, d82.Reg)
		ctx.W.EmitOrInt64(r91, d105.Reg)
		d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
		ctx.BindReg(r91, &d106)
	}
	if d106.Loc == scm.LocReg && d82.Loc == scm.LocReg && d106.Reg == d82.Reg {
		ctx.TransferReg(d82.Reg)
		d82.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d105)
	d107 := d106
	if d107.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d107)
	}
	ctx.EmitStoreToStack(d107, 80)
	ctx.W.EmitJmp(lbl17)
	ctx.W.MarkLabel(lbl15)
	ctx.W.ResolveFixups()
	d108 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
	ctx.BindReg(r74, &d108)
	ctx.BindReg(r74, &d108)
	if r54 {
		ctx.UnprotectReg(r55)
	}
	if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d108)
	}
	if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d108)
	}
	var d109 scm.JITValueDesc
	if d108.Loc == scm.LocImm {
		d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d108.Imm.Int()))))}
	} else {
		r92 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r92, d108.Reg)
		d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
		ctx.BindReg(r92, &d109)
	}
	ctx.FreeDesc(&d108)
	var d110 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
		val := *(*int64)(unsafe.Pointer(fieldAddr))
		d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
		r93 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r93, thisptr.Reg, off)
		d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
		ctx.BindReg(r93, &d110)
	}
	if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d109)
	}
	if d110.Loc == scm.LocStack || d110.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d110)
	}
	var d111 scm.JITValueDesc
	if d109.Loc == scm.LocImm && d110.Loc == scm.LocImm {
		d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() + d110.Imm.Int())}
	} else if d110.Loc == scm.LocImm && d110.Imm.Int() == 0 {
		r94 := ctx.AllocRegExcept(d109.Reg)
		ctx.W.EmitMovRegReg(r94, d109.Reg)
		d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
		ctx.BindReg(r94, &d111)
	} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
		d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
		ctx.BindReg(d110.Reg, &d111)
	} else if d109.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d110.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d109.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d110.Reg)
		d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d111)
	} else if d110.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d109.Reg)
		ctx.W.EmitMovRegReg(scratch, d109.Reg)
		if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d110.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d111)
	} else {
		r95 := ctx.AllocRegExcept(d109.Reg)
		ctx.W.EmitMovRegReg(r95, d109.Reg)
		ctx.W.EmitAddInt64(r95, d110.Reg)
		d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
		ctx.BindReg(r95, &d111)
	}
	if d111.Loc == scm.LocReg && d109.Loc == scm.LocReg && d111.Reg == d109.Reg {
		ctx.TransferReg(d109.Reg)
		d109.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d109)
	ctx.FreeDesc(&d110)
	if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d111)
	}
	if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d111)
	}
	var d112 scm.JITValueDesc
	if d111.Loc == scm.LocImm {
		d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d111.Imm.Int()))))}
	} else {
		r96 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r96, d111.Reg)
		ctx.W.EmitShlRegImm8(r96, 32)
		ctx.W.EmitShrRegImm8(r96, 32)
		d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
		ctx.BindReg(r96, &d112)
	}
	ctx.FreeDesc(&d111)
	if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&idxInt)
	}
	if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d112)
	}
	var d113 scm.JITValueDesc
	if idxInt.Loc == scm.LocImm && d112.Loc == scm.LocImm {
		d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d112.Imm.Int()))}
	} else if d112.Loc == scm.LocImm {
		r97 := ctx.AllocRegExcept(idxInt.Reg)
		if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
			ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d112.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
			ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
		}
		ctx.W.EmitSetcc(r97, scm.CcB)
		d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r97}
		ctx.BindReg(r97, &d113)
	} else if idxInt.Loc == scm.LocImm {
		r98 := ctx.AllocReg()
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
		ctx.W.EmitCmpInt64(scm.RegR11, d112.Reg)
		ctx.W.EmitSetcc(r98, scm.CcB)
		d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r98}
		ctx.BindReg(r98, &d113)
	} else {
		r99 := ctx.AllocRegExcept(idxInt.Reg)
		ctx.W.EmitCmpInt64(idxInt.Reg, d112.Reg)
		ctx.W.EmitSetcc(r99, scm.CcB)
		d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r99}
		ctx.BindReg(r99, &d113)
	}
	ctx.FreeDesc(&d112)
	lbl21 := ctx.W.ReserveLabel()
	lbl22 := ctx.W.ReserveLabel()
	lbl23 := ctx.W.ReserveLabel()
	if d113.Loc == scm.LocImm {
		if d113.Imm.Bool() {
			ctx.W.EmitJmp(lbl21)
		} else {
			ctx.W.EmitJmp(lbl22)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d113.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl23)
		ctx.W.EmitJmp(lbl22)
		ctx.W.MarkLabel(lbl23)
		ctx.W.EmitJmp(lbl21)
	}
	ctx.FreeDesc(&d113)
	ctx.W.MarkLabel(lbl12)
	d114 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	var d115 scm.JITValueDesc
	if d114.Loc == scm.LocImm {
		d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d114.Imm.Int()))))}
	} else {
		r100 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r100, d114.Reg)
		ctx.W.EmitShlRegImm8(r100, 32)
		ctx.W.EmitShrRegImm8(r100, 32)
		d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
		ctx.BindReg(r100, &d115)
	}
	if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d115)
	}
	if thisptr.Loc == scm.LocImm {
		baseReg := ctx.AllocReg()
		if d115.Loc == scm.LocReg {
			ctx.FreeReg(baseReg)
			baseReg = ctx.AllocRegExcept(d115.Reg)
		}
		ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int())+unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
		if d115.Loc == scm.LocImm {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d115.Imm.Int()))
			ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
		} else {
			ctx.W.EmitStoreRegMem(d115.Reg, baseReg, 0)
		}
		ctx.FreeReg(baseReg)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
		if d115.Loc == scm.LocImm {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d115.Imm.Int()))
			ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
		} else {
			ctx.W.EmitStoreRegMem(d115.Reg, thisptr.Reg, off)
		}
	}
	ctx.FreeDesc(&d115)
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	r101 := d114.Loc == scm.LocReg
	r102 := d114.Reg
	if r101 {
		ctx.ProtectReg(r102)
	}
	lbl24 := ctx.W.ReserveLabel()
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	var d116 scm.JITValueDesc
	if d114.Loc == scm.LocImm {
		d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d114.Imm.Int()))))}
	} else {
		r103 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r103, d114.Reg)
		ctx.W.EmitShlRegImm8(r103, 32)
		ctx.W.EmitShrRegImm8(r103, 32)
		d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
		ctx.BindReg(r103, &d116)
	}
	var d117 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
		r104 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r104, thisptr.Reg, off)
		d117 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
		ctx.BindReg(r104, &d117)
	}
	if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d117)
	}
	if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d117)
	}
	var d118 scm.JITValueDesc
	if d117.Loc == scm.LocImm {
		d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d117.Imm.Int()))))}
	} else {
		r105 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r105, d117.Reg)
		ctx.W.EmitShlRegImm8(r105, 56)
		ctx.W.EmitShrRegImm8(r105, 56)
		d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
		ctx.BindReg(r105, &d118)
	}
	ctx.FreeDesc(&d117)
	if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d116)
	}
	if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d118)
	}
	var d119 scm.JITValueDesc
	if d116.Loc == scm.LocImm && d118.Loc == scm.LocImm {
		d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() * d118.Imm.Int())}
	} else if d116.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d118.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d118.Reg)
		d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d119)
	} else if d118.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d116.Reg)
		ctx.W.EmitMovRegReg(scratch, d116.Reg)
		if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d118.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d119)
	} else {
		r106 := ctx.AllocRegExcept(d116.Reg)
		ctx.W.EmitMovRegReg(r106, d116.Reg)
		ctx.W.EmitImulInt64(r106, d118.Reg)
		d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
		ctx.BindReg(r106, &d119)
	}
	if d119.Loc == scm.LocReg && d116.Loc == scm.LocReg && d119.Reg == d116.Reg {
		ctx.TransferReg(d116.Reg)
		d116.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d116)
	ctx.FreeDesc(&d118)
	var d120 scm.JITValueDesc
	r107 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
		dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
		sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
		ctx.W.EmitMovRegImm64(r107, uint64(dataPtr))
		d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107, StackOff: int32(sliceLen)}
		ctx.BindReg(r107, &d120)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
		ctx.W.EmitMovRegMem(r107, thisptr.Reg, off)
		d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
		ctx.BindReg(r107, &d120)
	}
	ctx.BindReg(r107, &d120)
	if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d119)
	}
	var d121 scm.JITValueDesc
	if d119.Loc == scm.LocImm {
		d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() / 64)}
	} else {
		r108 := ctx.AllocRegExcept(d119.Reg)
		ctx.W.EmitMovRegReg(r108, d119.Reg)
		ctx.W.EmitShrRegImm8(r108, 6)
		d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
		ctx.BindReg(r108, &d121)
	}
	if d121.Loc == scm.LocReg && d119.Loc == scm.LocReg && d121.Reg == d119.Reg {
		ctx.TransferReg(d119.Reg)
		d119.Loc = scm.LocNone
	}
	if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d121)
	}
	r109 := ctx.AllocReg()
	if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d121)
	}
	if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d120)
	}
	if d121.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r109, uint64(d121.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r109, d121.Reg)
		ctx.W.EmitShlRegImm8(r109, 3)
	}
	if d120.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
		ctx.W.EmitAddInt64(r109, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r109, d120.Reg)
	}
	r110 := ctx.AllocRegExcept(r109)
	ctx.W.EmitMovRegMem(r110, r109, 0)
	ctx.FreeReg(r109)
	d122 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
	ctx.BindReg(r110, &d122)
	ctx.FreeDesc(&d121)
	if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d119)
	}
	var d123 scm.JITValueDesc
	if d119.Loc == scm.LocImm {
		d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() % 64)}
	} else {
		r111 := ctx.AllocRegExcept(d119.Reg)
		ctx.W.EmitMovRegReg(r111, d119.Reg)
		ctx.W.EmitAndRegImm32(r111, 63)
		d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
		ctx.BindReg(r111, &d123)
	}
	if d123.Loc == scm.LocReg && d119.Loc == scm.LocReg && d123.Reg == d119.Reg {
		ctx.TransferReg(d119.Reg)
		d119.Loc = scm.LocNone
	}
	if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d122)
	}
	if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d123)
	}
	var d124 scm.JITValueDesc
	if d122.Loc == scm.LocImm && d123.Loc == scm.LocImm {
		d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d122.Imm.Int()) << uint64(d123.Imm.Int())))}
	} else if d123.Loc == scm.LocImm {
		r112 := ctx.AllocRegExcept(d122.Reg)
		ctx.W.EmitMovRegReg(r112, d122.Reg)
		ctx.W.EmitShlRegImm8(r112, uint8(d123.Imm.Int()))
		d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
		ctx.BindReg(r112, &d124)
	} else {
		{
			shiftSrc := d122.Reg
			r113 := ctx.AllocRegExcept(d122.Reg)
			ctx.W.EmitMovRegReg(r113, d122.Reg)
			shiftSrc = r113
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d123.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d123.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d123.Reg)
			}
			ctx.W.EmitShlRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d124)
		}
	}
	if d124.Loc == scm.LocReg && d122.Loc == scm.LocReg && d124.Reg == d122.Reg {
		ctx.TransferReg(d122.Reg)
		d122.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d122)
	ctx.FreeDesc(&d123)
	var d125 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
		r114 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
		d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
		ctx.BindReg(r114, &d125)
	}
	lbl25 := ctx.W.ReserveLabel()
	lbl26 := ctx.W.ReserveLabel()
	lbl27 := ctx.W.ReserveLabel()
	if d125.Loc == scm.LocImm {
		if d125.Imm.Bool() {
			ctx.W.EmitJmp(lbl25)
		} else {
			d126 := d124
			if d126.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d126)
			}
			ctx.EmitStoreToStack(d126, 88)
			ctx.W.EmitJmp(lbl26)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d125.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl27)
		d127 := d124
		if d127.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d127)
		}
		ctx.EmitStoreToStack(d127, 88)
		ctx.W.EmitJmp(lbl26)
		ctx.W.MarkLabel(lbl27)
		ctx.W.EmitJmp(lbl25)
	}
	ctx.FreeDesc(&d125)
	ctx.W.MarkLabel(lbl26)
	d128 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
	var d129 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
		r115 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r115, thisptr.Reg, off)
		d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
		ctx.BindReg(r115, &d129)
	}
	if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d129)
	}
	if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d129)
	}
	var d130 scm.JITValueDesc
	if d129.Loc == scm.LocImm {
		d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d129.Imm.Int()))))}
	} else {
		r116 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r116, d129.Reg)
		ctx.W.EmitShlRegImm8(r116, 56)
		ctx.W.EmitShrRegImm8(r116, 56)
		d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
		ctx.BindReg(r116, &d130)
	}
	ctx.FreeDesc(&d129)
	d131 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d130.Loc == scm.LocStack || d130.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d130)
	}
	var d132 scm.JITValueDesc
	if d131.Loc == scm.LocImm && d130.Loc == scm.LocImm {
		d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d131.Imm.Int() - d130.Imm.Int())}
	} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
		r117 := ctx.AllocRegExcept(d131.Reg)
		ctx.W.EmitMovRegReg(r117, d131.Reg)
		d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
		ctx.BindReg(r117, &d132)
	} else if d131.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d130.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d130.Reg)
		d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d132)
	} else if d130.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d131.Reg)
		ctx.W.EmitMovRegReg(scratch, d131.Reg)
		if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d130.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d132)
	} else {
		r118 := ctx.AllocRegExcept(d131.Reg)
		ctx.W.EmitMovRegReg(r118, d131.Reg)
		ctx.W.EmitSubInt64(r118, d130.Reg)
		d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
		ctx.BindReg(r118, &d132)
	}
	if d132.Loc == scm.LocReg && d131.Loc == scm.LocReg && d132.Reg == d131.Reg {
		ctx.TransferReg(d131.Reg)
		d131.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d130)
	if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d128)
	}
	if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d132)
	}
	var d133 scm.JITValueDesc
	if d128.Loc == scm.LocImm && d132.Loc == scm.LocImm {
		d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d128.Imm.Int()) >> uint64(d132.Imm.Int())))}
	} else if d132.Loc == scm.LocImm {
		r119 := ctx.AllocRegExcept(d128.Reg)
		ctx.W.EmitMovRegReg(r119, d128.Reg)
		ctx.W.EmitShrRegImm8(r119, uint8(d132.Imm.Int()))
		d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
		ctx.BindReg(r119, &d133)
	} else {
		{
			shiftSrc := d128.Reg
			r120 := ctx.AllocRegExcept(d128.Reg)
			ctx.W.EmitMovRegReg(r120, d128.Reg)
			shiftSrc = r120
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d132.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d132.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d132.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d133)
		}
	}
	if d133.Loc == scm.LocReg && d128.Loc == scm.LocReg && d133.Reg == d128.Reg {
		ctx.TransferReg(d128.Reg)
		d128.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d128)
	ctx.FreeDesc(&d132)
	r121 := ctx.AllocReg()
	if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d133)
	}
	ctx.EmitMovToReg(r121, d133)
	ctx.W.EmitJmp(lbl24)
	ctx.W.MarkLabel(lbl25)
	if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d119)
	}
	var d134 scm.JITValueDesc
	if d119.Loc == scm.LocImm {
		d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() % 64)}
	} else {
		r122 := ctx.AllocRegExcept(d119.Reg)
		ctx.W.EmitMovRegReg(r122, d119.Reg)
		ctx.W.EmitAndRegImm32(r122, 63)
		d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
		ctx.BindReg(r122, &d134)
	}
	if d134.Loc == scm.LocReg && d119.Loc == scm.LocReg && d134.Reg == d119.Reg {
		ctx.TransferReg(d119.Reg)
		d119.Loc = scm.LocNone
	}
	var d135 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
		r123 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r123, thisptr.Reg, off)
		d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
		ctx.BindReg(r123, &d135)
	}
	if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d135)
	}
	if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d135)
	}
	var d136 scm.JITValueDesc
	if d135.Loc == scm.LocImm {
		d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d135.Imm.Int()))))}
	} else {
		r124 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r124, d135.Reg)
		ctx.W.EmitShlRegImm8(r124, 56)
		ctx.W.EmitShrRegImm8(r124, 56)
		d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
		ctx.BindReg(r124, &d136)
	}
	ctx.FreeDesc(&d135)
	if d134.Loc == scm.LocStack || d134.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d134)
	}
	if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d136)
	}
	var d137 scm.JITValueDesc
	if d134.Loc == scm.LocImm && d136.Loc == scm.LocImm {
		d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() + d136.Imm.Int())}
	} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
		r125 := ctx.AllocRegExcept(d134.Reg)
		ctx.W.EmitMovRegReg(r125, d134.Reg)
		d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
		ctx.BindReg(r125, &d137)
	} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
		d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
		ctx.BindReg(d136.Reg, &d137)
	} else if d134.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d136.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d136.Reg)
		d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d137)
	} else if d136.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d134.Reg)
		ctx.W.EmitMovRegReg(scratch, d134.Reg)
		if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d136.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d137)
	} else {
		r126 := ctx.AllocRegExcept(d134.Reg)
		ctx.W.EmitMovRegReg(r126, d134.Reg)
		ctx.W.EmitAddInt64(r126, d136.Reg)
		d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
		ctx.BindReg(r126, &d137)
	}
	if d137.Loc == scm.LocReg && d134.Loc == scm.LocReg && d137.Reg == d134.Reg {
		ctx.TransferReg(d134.Reg)
		d134.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d134)
	ctx.FreeDesc(&d136)
	if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d137)
	}
	var d138 scm.JITValueDesc
	if d137.Loc == scm.LocImm {
		d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d137.Imm.Int()) > uint64(64))}
	} else {
		r127 := ctx.AllocRegExcept(d137.Reg)
		ctx.W.EmitCmpRegImm32(d137.Reg, 64)
		ctx.W.EmitSetcc(r127, scm.CcA)
		d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r127}
		ctx.BindReg(r127, &d138)
	}
	ctx.FreeDesc(&d137)
	lbl28 := ctx.W.ReserveLabel()
	lbl29 := ctx.W.ReserveLabel()
	if d138.Loc == scm.LocImm {
		if d138.Imm.Bool() {
			ctx.W.EmitJmp(lbl28)
		} else {
			d139 := d124
			if d139.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d139.Loc == scm.LocStack || d139.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d139)
			}
			ctx.EmitStoreToStack(d139, 88)
			ctx.W.EmitJmp(lbl26)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d138.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl29)
		d140 := d124
		if d140.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d140)
		}
		ctx.EmitStoreToStack(d140, 88)
		ctx.W.EmitJmp(lbl26)
		ctx.W.MarkLabel(lbl29)
		ctx.W.EmitJmp(lbl28)
	}
	ctx.FreeDesc(&d138)
	ctx.W.MarkLabel(lbl28)
	if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d119)
	}
	var d141 scm.JITValueDesc
	if d119.Loc == scm.LocImm {
		d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() / 64)}
	} else {
		r128 := ctx.AllocRegExcept(d119.Reg)
		ctx.W.EmitMovRegReg(r128, d119.Reg)
		ctx.W.EmitShrRegImm8(r128, 6)
		d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
		ctx.BindReg(r128, &d141)
	}
	if d141.Loc == scm.LocReg && d119.Loc == scm.LocReg && d141.Reg == d119.Reg {
		ctx.TransferReg(d119.Reg)
		d119.Loc = scm.LocNone
	}
	if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d141)
	}
	var d142 scm.JITValueDesc
	if d141.Loc == scm.LocImm {
		d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d141.Reg)
		ctx.W.EmitMovRegReg(scratch, d141.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d142)
	}
	if d142.Loc == scm.LocReg && d141.Loc == scm.LocReg && d142.Reg == d141.Reg {
		ctx.TransferReg(d141.Reg)
		d141.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d141)
	if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d142)
	}
	r129 := ctx.AllocReg()
	if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d142)
	}
	if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d120)
	}
	if d142.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r129, uint64(d142.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r129, d142.Reg)
		ctx.W.EmitShlRegImm8(r129, 3)
	}
	if d120.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
		ctx.W.EmitAddInt64(r129, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r129, d120.Reg)
	}
	r130 := ctx.AllocRegExcept(r129)
	ctx.W.EmitMovRegMem(r130, r129, 0)
	ctx.FreeReg(r129)
	d143 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r130}
	ctx.BindReg(r130, &d143)
	ctx.FreeDesc(&d142)
	if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d119)
	}
	var d144 scm.JITValueDesc
	if d119.Loc == scm.LocImm {
		d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() % 64)}
	} else {
		r131 := ctx.AllocRegExcept(d119.Reg)
		ctx.W.EmitMovRegReg(r131, d119.Reg)
		ctx.W.EmitAndRegImm32(r131, 63)
		d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
		ctx.BindReg(r131, &d144)
	}
	if d144.Loc == scm.LocReg && d119.Loc == scm.LocReg && d144.Reg == d119.Reg {
		ctx.TransferReg(d119.Reg)
		d119.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d119)
	d145 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d144)
	}
	var d146 scm.JITValueDesc
	if d145.Loc == scm.LocImm && d144.Loc == scm.LocImm {
		d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() - d144.Imm.Int())}
	} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
		r132 := ctx.AllocRegExcept(d145.Reg)
		ctx.W.EmitMovRegReg(r132, d145.Reg)
		d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
		ctx.BindReg(r132, &d146)
	} else if d145.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d144.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d144.Reg)
		d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d146)
	} else if d144.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d145.Reg)
		ctx.W.EmitMovRegReg(scratch, d145.Reg)
		if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d144.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d146)
	} else {
		r133 := ctx.AllocRegExcept(d145.Reg)
		ctx.W.EmitMovRegReg(r133, d145.Reg)
		ctx.W.EmitSubInt64(r133, d144.Reg)
		d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
		ctx.BindReg(r133, &d146)
	}
	if d146.Loc == scm.LocReg && d145.Loc == scm.LocReg && d146.Reg == d145.Reg {
		ctx.TransferReg(d145.Reg)
		d145.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d144)
	if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d143)
	}
	if d146.Loc == scm.LocStack || d146.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d146)
	}
	var d147 scm.JITValueDesc
	if d143.Loc == scm.LocImm && d146.Loc == scm.LocImm {
		d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d143.Imm.Int()) >> uint64(d146.Imm.Int())))}
	} else if d146.Loc == scm.LocImm {
		r134 := ctx.AllocRegExcept(d143.Reg)
		ctx.W.EmitMovRegReg(r134, d143.Reg)
		ctx.W.EmitShrRegImm8(r134, uint8(d146.Imm.Int()))
		d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
		ctx.BindReg(r134, &d147)
	} else {
		{
			shiftSrc := d143.Reg
			r135 := ctx.AllocRegExcept(d143.Reg)
			ctx.W.EmitMovRegReg(r135, d143.Reg)
			shiftSrc = r135
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d146.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d146.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d146.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d147)
		}
	}
	if d147.Loc == scm.LocReg && d143.Loc == scm.LocReg && d147.Reg == d143.Reg {
		ctx.TransferReg(d143.Reg)
		d143.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d143)
	ctx.FreeDesc(&d146)
	if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d124)
	}
	if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d147)
	}
	var d148 scm.JITValueDesc
	if d124.Loc == scm.LocImm && d147.Loc == scm.LocImm {
		d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() | d147.Imm.Int())}
	} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
		d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
		ctx.BindReg(d147.Reg, &d148)
	} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
		r136 := ctx.AllocRegExcept(d124.Reg)
		ctx.W.EmitMovRegReg(r136, d124.Reg)
		d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
		ctx.BindReg(r136, &d148)
	} else if d124.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d147.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d124.Imm.Int()))
		ctx.W.EmitOrInt64(scratch, d147.Reg)
		d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d148)
	} else if d147.Loc == scm.LocImm {
		r137 := ctx.AllocRegExcept(d124.Reg)
		ctx.W.EmitMovRegReg(r137, d124.Reg)
		if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
			ctx.W.EmitOrRegImm32(r137, int32(d147.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
			ctx.W.EmitOrInt64(r137, scm.RegR11)
		}
		d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
		ctx.BindReg(r137, &d148)
	} else {
		r138 := ctx.AllocRegExcept(d124.Reg)
		ctx.W.EmitMovRegReg(r138, d124.Reg)
		ctx.W.EmitOrInt64(r138, d147.Reg)
		d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
		ctx.BindReg(r138, &d148)
	}
	if d148.Loc == scm.LocReg && d124.Loc == scm.LocReg && d148.Reg == d124.Reg {
		ctx.TransferReg(d124.Reg)
		d124.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d147)
	d149 := d148
	if d149.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d149)
	}
	ctx.EmitStoreToStack(d149, 88)
	ctx.W.EmitJmp(lbl26)
	ctx.W.MarkLabel(lbl24)
	ctx.W.ResolveFixups()
	d150 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
	ctx.BindReg(r121, &d150)
	ctx.BindReg(r121, &d150)
	if r101 {
		ctx.UnprotectReg(r102)
	}
	if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d150)
	}
	if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d150)
	}
	var d151 scm.JITValueDesc
	if d150.Loc == scm.LocImm {
		d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d150.Imm.Int()))))}
	} else {
		r139 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r139, d150.Reg)
		d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
		ctx.BindReg(r139, &d151)
	}
	ctx.FreeDesc(&d150)
	var d152 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
		val := *(*int64)(unsafe.Pointer(fieldAddr))
		d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
		r140 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
		d152 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
		ctx.BindReg(r140, &d152)
	}
	if d151.Loc == scm.LocStack || d151.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d151)
	}
	if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d152)
	}
	var d153 scm.JITValueDesc
	if d151.Loc == scm.LocImm && d152.Loc == scm.LocImm {
		d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() + d152.Imm.Int())}
	} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
		r141 := ctx.AllocRegExcept(d151.Reg)
		ctx.W.EmitMovRegReg(r141, d151.Reg)
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
		ctx.BindReg(r141, &d153)
	} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d152.Reg}
		ctx.BindReg(d152.Reg, &d153)
	} else if d151.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d152.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d151.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d152.Reg)
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d153)
	} else if d152.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d151.Reg)
		ctx.W.EmitMovRegReg(scratch, d151.Reg)
		if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d152.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d153)
	} else {
		r142 := ctx.AllocRegExcept(d151.Reg)
		ctx.W.EmitMovRegReg(r142, d151.Reg)
		ctx.W.EmitAddInt64(r142, d152.Reg)
		d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
		ctx.BindReg(r142, &d153)
	}
	if d153.Loc == scm.LocReg && d151.Loc == scm.LocReg && d153.Reg == d151.Reg {
		ctx.TransferReg(d151.Reg)
		d151.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d151)
	ctx.FreeDesc(&d152)
	var d154 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
		r143 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r143, thisptr.Reg, off)
		d154 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
		ctx.BindReg(r143, &d154)
	}
	lbl30 := ctx.W.ReserveLabel()
	lbl31 := ctx.W.ReserveLabel()
	lbl32 := ctx.W.ReserveLabel()
	if d154.Loc == scm.LocImm {
		if d154.Imm.Bool() {
			ctx.W.EmitJmp(lbl30)
		} else {
			ctx.W.EmitJmp(lbl31)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d154.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl32)
		ctx.W.EmitJmp(lbl31)
		ctx.W.MarkLabel(lbl32)
		ctx.W.EmitJmp(lbl30)
	}
	ctx.FreeDesc(&d154)
	ctx.W.MarkLabel(lbl22)
	lbl33 := ctx.W.ReserveLabel()
	d155 := d58
	if d155.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d155)
	}
	d156 := d155
	if d156.Loc == scm.LocImm {
		d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: d156.Type, Imm: scm.NewInt(int64(uint64(d156.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d156.Reg, 32)
		ctx.W.EmitShrRegImm8(d156.Reg, 32)
	}
	ctx.EmitStoreToStack(d156, 56)
	d157 := d60
	if d157.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d157)
	}
	d158 := d157
	if d158.Loc == scm.LocImm {
		d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: d158.Type, Imm: scm.NewInt(int64(uint64(d158.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d158.Reg, 32)
		ctx.W.EmitShrRegImm8(d158.Reg, 32)
	}
	ctx.EmitStoreToStack(d158, 64)
	ctx.W.MarkLabel(lbl33)
	d159 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
	d160 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
	if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d159)
	}
	if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d160)
	}
	var d161 scm.JITValueDesc
	if d159.Loc == scm.LocImm && d160.Loc == scm.LocImm {
		d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d159.Imm.Int()) == uint64(d160.Imm.Int()))}
	} else if d160.Loc == scm.LocImm {
		r144 := ctx.AllocRegExcept(d159.Reg)
		if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
			ctx.W.EmitCmpRegImm32(d159.Reg, int32(d160.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
			ctx.W.EmitCmpInt64(d159.Reg, scm.RegR11)
		}
		ctx.W.EmitSetcc(r144, scm.CcE)
		d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r144}
		ctx.BindReg(r144, &d161)
	} else if d159.Loc == scm.LocImm {
		r145 := ctx.AllocReg()
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
		ctx.W.EmitCmpInt64(scm.RegR11, d160.Reg)
		ctx.W.EmitSetcc(r145, scm.CcE)
		d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r145}
		ctx.BindReg(r145, &d161)
	} else {
		r146 := ctx.AllocRegExcept(d159.Reg)
		ctx.W.EmitCmpInt64(d159.Reg, d160.Reg)
		ctx.W.EmitSetcc(r146, scm.CcE)
		d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r146}
		ctx.BindReg(r146, &d161)
	}
	lbl34 := ctx.W.ReserveLabel()
	lbl35 := ctx.W.ReserveLabel()
	if d161.Loc == scm.LocImm {
		if d161.Imm.Bool() {
			d162 := d159
			if d162.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d162.Loc == scm.LocStack || d162.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d162)
			}
			d163 := d162
			if d163.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: d163.Type, Imm: scm.NewInt(int64(uint64(d163.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d163.Reg, 32)
				ctx.W.EmitShrRegImm8(d163.Reg, 32)
			}
			ctx.EmitStoreToStack(d163, 24)
			ctx.W.EmitJmp(lbl12)
		} else {
			ctx.W.EmitJmp(lbl34)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d161.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl35)
		ctx.W.EmitJmp(lbl34)
		ctx.W.MarkLabel(lbl35)
		d164 := d159
		if d164.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d164)
		}
		d165 := d164
		if d165.Loc == scm.LocImm {
			d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: d165.Type, Imm: scm.NewInt(int64(uint64(d165.Imm.Int()) & 0xffffffff))}
		} else {
			ctx.W.EmitShlRegImm8(d165.Reg, 32)
			ctx.W.EmitShrRegImm8(d165.Reg, 32)
		}
		ctx.EmitStoreToStack(d165, 24)
		ctx.W.EmitJmp(lbl12)
	}
	ctx.FreeDesc(&d161)
	ctx.W.MarkLabel(lbl21)
	if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d58)
	}
	var d166 scm.JITValueDesc
	if d58.Loc == scm.LocImm {
		d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() - 1)}
	} else {
		scratch := ctx.AllocRegExcept(d58.Reg)
		ctx.W.EmitMovRegReg(scratch, d58.Reg)
		ctx.W.EmitSubRegImm32(scratch, int32(1))
		d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d166)
	}
	if d166.Loc == scm.LocImm {
		d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: d166.Type, Imm: scm.NewInt(int64(uint64(d166.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d166.Reg, 32)
		ctx.W.EmitShrRegImm8(d166.Reg, 32)
	}
	if d166.Loc == scm.LocReg && d58.Loc == scm.LocReg && d166.Reg == d58.Reg {
		ctx.TransferReg(d58.Reg)
		d58.Loc = scm.LocNone
	}
	d167 := d59
	if d167.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d167)
	}
	d168 := d167
	if d168.Loc == scm.LocImm {
		d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: d168.Type, Imm: scm.NewInt(int64(uint64(d168.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d168.Reg, 32)
		ctx.W.EmitShrRegImm8(d168.Reg, 32)
	}
	ctx.EmitStoreToStack(d168, 56)
	d169 := d166
	if d169.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d169)
	}
	d170 := d169
	if d170.Loc == scm.LocImm {
		d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: d170.Type, Imm: scm.NewInt(int64(uint64(d170.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d170.Reg, 32)
		ctx.W.EmitShrRegImm8(d170.Reg, 32)
	}
	ctx.EmitStoreToStack(d170, 64)
	ctx.W.EmitJmp(lbl33)
	ctx.W.MarkLabel(lbl31)
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	r147 := d114.Loc == scm.LocReg
	r148 := d114.Reg
	if r147 {
		ctx.ProtectReg(r148)
	}
	lbl36 := ctx.W.ReserveLabel()
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	var d171 scm.JITValueDesc
	if d114.Loc == scm.LocImm {
		d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d114.Imm.Int()))))}
	} else {
		r149 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r149, d114.Reg)
		ctx.W.EmitShlRegImm8(r149, 32)
		ctx.W.EmitShrRegImm8(r149, 32)
		d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
		ctx.BindReg(r149, &d171)
	}
	var d172 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
		r150 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
		d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
		ctx.BindReg(r150, &d172)
	}
	if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d172)
	}
	if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d172)
	}
	var d173 scm.JITValueDesc
	if d172.Loc == scm.LocImm {
		d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d172.Imm.Int()))))}
	} else {
		r151 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r151, d172.Reg)
		ctx.W.EmitShlRegImm8(r151, 56)
		ctx.W.EmitShrRegImm8(r151, 56)
		d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
		ctx.BindReg(r151, &d173)
	}
	ctx.FreeDesc(&d172)
	if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d171)
	}
	if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d173)
	}
	var d174 scm.JITValueDesc
	if d171.Loc == scm.LocImm && d173.Loc == scm.LocImm {
		d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() * d173.Imm.Int())}
	} else if d171.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d173.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d173.Reg)
		d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d174)
	} else if d173.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d171.Reg)
		ctx.W.EmitMovRegReg(scratch, d171.Reg)
		if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d173.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d174)
	} else {
		r152 := ctx.AllocRegExcept(d171.Reg)
		ctx.W.EmitMovRegReg(r152, d171.Reg)
		ctx.W.EmitImulInt64(r152, d173.Reg)
		d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
		ctx.BindReg(r152, &d174)
	}
	if d174.Loc == scm.LocReg && d171.Loc == scm.LocReg && d174.Reg == d171.Reg {
		ctx.TransferReg(d171.Reg)
		d171.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d171)
	ctx.FreeDesc(&d173)
	var d175 scm.JITValueDesc
	r153 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
		dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
		sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
		ctx.W.EmitMovRegImm64(r153, uint64(dataPtr))
		d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153, StackOff: int32(sliceLen)}
		ctx.BindReg(r153, &d175)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
		ctx.W.EmitMovRegMem(r153, thisptr.Reg, off)
		d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
		ctx.BindReg(r153, &d175)
	}
	ctx.BindReg(r153, &d175)
	if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d174)
	}
	var d176 scm.JITValueDesc
	if d174.Loc == scm.LocImm {
		d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() / 64)}
	} else {
		r154 := ctx.AllocRegExcept(d174.Reg)
		ctx.W.EmitMovRegReg(r154, d174.Reg)
		ctx.W.EmitShrRegImm8(r154, 6)
		d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
		ctx.BindReg(r154, &d176)
	}
	if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
		ctx.TransferReg(d174.Reg)
		d174.Loc = scm.LocNone
	}
	if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d176)
	}
	r155 := ctx.AllocReg()
	if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d176)
	}
	if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d175)
	}
	if d176.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r155, uint64(d176.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r155, d176.Reg)
		ctx.W.EmitShlRegImm8(r155, 3)
	}
	if d175.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
		ctx.W.EmitAddInt64(r155, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r155, d175.Reg)
	}
	r156 := ctx.AllocRegExcept(r155)
	ctx.W.EmitMovRegMem(r156, r155, 0)
	ctx.FreeReg(r155)
	d177 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
	ctx.BindReg(r156, &d177)
	ctx.FreeDesc(&d176)
	if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d174)
	}
	var d178 scm.JITValueDesc
	if d174.Loc == scm.LocImm {
		d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() % 64)}
	} else {
		r157 := ctx.AllocRegExcept(d174.Reg)
		ctx.W.EmitMovRegReg(r157, d174.Reg)
		ctx.W.EmitAndRegImm32(r157, 63)
		d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
		ctx.BindReg(r157, &d178)
	}
	if d178.Loc == scm.LocReg && d174.Loc == scm.LocReg && d178.Reg == d174.Reg {
		ctx.TransferReg(d174.Reg)
		d174.Loc = scm.LocNone
	}
	if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d177)
	}
	if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d178)
	}
	var d179 scm.JITValueDesc
	if d177.Loc == scm.LocImm && d178.Loc == scm.LocImm {
		d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d177.Imm.Int()) << uint64(d178.Imm.Int())))}
	} else if d178.Loc == scm.LocImm {
		r158 := ctx.AllocRegExcept(d177.Reg)
		ctx.W.EmitMovRegReg(r158, d177.Reg)
		ctx.W.EmitShlRegImm8(r158, uint8(d178.Imm.Int()))
		d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
		ctx.BindReg(r158, &d179)
	} else {
		{
			shiftSrc := d177.Reg
			r159 := ctx.AllocRegExcept(d177.Reg)
			ctx.W.EmitMovRegReg(r159, d177.Reg)
			shiftSrc = r159
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d178.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d178.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d178.Reg)
			}
			ctx.W.EmitShlRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d179)
		}
	}
	if d179.Loc == scm.LocReg && d177.Loc == scm.LocReg && d179.Reg == d177.Reg {
		ctx.TransferReg(d177.Reg)
		d177.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d177)
	ctx.FreeDesc(&d178)
	var d180 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
		r160 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r160, thisptr.Reg, off)
		d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
		ctx.BindReg(r160, &d180)
	}
	lbl37 := ctx.W.ReserveLabel()
	lbl38 := ctx.W.ReserveLabel()
	lbl39 := ctx.W.ReserveLabel()
	if d180.Loc == scm.LocImm {
		if d180.Imm.Bool() {
			ctx.W.EmitJmp(lbl37)
		} else {
			d181 := d179
			if d181.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d181)
			}
			ctx.EmitStoreToStack(d181, 96)
			ctx.W.EmitJmp(lbl38)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d180.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl39)
		d182 := d179
		if d182.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d182)
		}
		ctx.EmitStoreToStack(d182, 96)
		ctx.W.EmitJmp(lbl38)
		ctx.W.MarkLabel(lbl39)
		ctx.W.EmitJmp(lbl37)
	}
	ctx.FreeDesc(&d180)
	ctx.W.MarkLabel(lbl38)
	d183 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
	var d184 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
		r161 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r161, thisptr.Reg, off)
		d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
		ctx.BindReg(r161, &d184)
	}
	if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d184)
	}
	if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d184)
	}
	var d185 scm.JITValueDesc
	if d184.Loc == scm.LocImm {
		d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d184.Imm.Int()))))}
	} else {
		r162 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r162, d184.Reg)
		ctx.W.EmitShlRegImm8(r162, 56)
		ctx.W.EmitShrRegImm8(r162, 56)
		d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
		ctx.BindReg(r162, &d185)
	}
	ctx.FreeDesc(&d184)
	d186 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d185)
	}
	var d187 scm.JITValueDesc
	if d186.Loc == scm.LocImm && d185.Loc == scm.LocImm {
		d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() - d185.Imm.Int())}
	} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
		r163 := ctx.AllocRegExcept(d186.Reg)
		ctx.W.EmitMovRegReg(r163, d186.Reg)
		d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
		ctx.BindReg(r163, &d187)
	} else if d186.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d185.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d186.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d185.Reg)
		d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d187)
	} else if d185.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d186.Reg)
		ctx.W.EmitMovRegReg(scratch, d186.Reg)
		if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d185.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d187)
	} else {
		r164 := ctx.AllocRegExcept(d186.Reg)
		ctx.W.EmitMovRegReg(r164, d186.Reg)
		ctx.W.EmitSubInt64(r164, d185.Reg)
		d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
		ctx.BindReg(r164, &d187)
	}
	if d187.Loc == scm.LocReg && d186.Loc == scm.LocReg && d187.Reg == d186.Reg {
		ctx.TransferReg(d186.Reg)
		d186.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d185)
	if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d183)
	}
	if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d187)
	}
	var d188 scm.JITValueDesc
	if d183.Loc == scm.LocImm && d187.Loc == scm.LocImm {
		d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d183.Imm.Int()) >> uint64(d187.Imm.Int())))}
	} else if d187.Loc == scm.LocImm {
		r165 := ctx.AllocRegExcept(d183.Reg)
		ctx.W.EmitMovRegReg(r165, d183.Reg)
		ctx.W.EmitShrRegImm8(r165, uint8(d187.Imm.Int()))
		d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
		ctx.BindReg(r165, &d188)
	} else {
		{
			shiftSrc := d183.Reg
			r166 := ctx.AllocRegExcept(d183.Reg)
			ctx.W.EmitMovRegReg(r166, d183.Reg)
			shiftSrc = r166
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d187.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d187.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d187.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d188)
		}
	}
	if d188.Loc == scm.LocReg && d183.Loc == scm.LocReg && d188.Reg == d183.Reg {
		ctx.TransferReg(d183.Reg)
		d183.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d183)
	ctx.FreeDesc(&d187)
	r167 := ctx.AllocReg()
	if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d188)
	}
	ctx.EmitMovToReg(r167, d188)
	ctx.W.EmitJmp(lbl36)
	ctx.W.MarkLabel(lbl37)
	if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d174)
	}
	var d189 scm.JITValueDesc
	if d174.Loc == scm.LocImm {
		d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() % 64)}
	} else {
		r168 := ctx.AllocRegExcept(d174.Reg)
		ctx.W.EmitMovRegReg(r168, d174.Reg)
		ctx.W.EmitAndRegImm32(r168, 63)
		d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
		ctx.BindReg(r168, &d189)
	}
	if d189.Loc == scm.LocReg && d174.Loc == scm.LocReg && d189.Reg == d174.Reg {
		ctx.TransferReg(d174.Reg)
		d174.Loc = scm.LocNone
	}
	var d190 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
		r169 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r169, thisptr.Reg, off)
		d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
		ctx.BindReg(r169, &d190)
	}
	if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d190)
	}
	if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d190)
	}
	var d191 scm.JITValueDesc
	if d190.Loc == scm.LocImm {
		d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d190.Imm.Int()))))}
	} else {
		r170 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r170, d190.Reg)
		ctx.W.EmitShlRegImm8(r170, 56)
		ctx.W.EmitShrRegImm8(r170, 56)
		d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
		ctx.BindReg(r170, &d191)
	}
	ctx.FreeDesc(&d190)
	if d189.Loc == scm.LocStack || d189.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d189)
	}
	if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d191)
	}
	var d192 scm.JITValueDesc
	if d189.Loc == scm.LocImm && d191.Loc == scm.LocImm {
		d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d189.Imm.Int() + d191.Imm.Int())}
	} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
		r171 := ctx.AllocRegExcept(d189.Reg)
		ctx.W.EmitMovRegReg(r171, d189.Reg)
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
		ctx.BindReg(r171, &d192)
	} else if d189.Loc == scm.LocImm && d189.Imm.Int() == 0 {
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d191.Reg}
		ctx.BindReg(d191.Reg, &d192)
	} else if d189.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d191.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d189.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d191.Reg)
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d192)
	} else if d191.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d189.Reg)
		ctx.W.EmitMovRegReg(scratch, d189.Reg)
		if d191.Imm.Int() >= -2147483648 && d191.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d191.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d191.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d192)
	} else {
		r172 := ctx.AllocRegExcept(d189.Reg)
		ctx.W.EmitMovRegReg(r172, d189.Reg)
		ctx.W.EmitAddInt64(r172, d191.Reg)
		d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
		ctx.BindReg(r172, &d192)
	}
	if d192.Loc == scm.LocReg && d189.Loc == scm.LocReg && d192.Reg == d189.Reg {
		ctx.TransferReg(d189.Reg)
		d189.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d189)
	ctx.FreeDesc(&d191)
	if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d192)
	}
	var d193 scm.JITValueDesc
	if d192.Loc == scm.LocImm {
		d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d192.Imm.Int()) > uint64(64))}
	} else {
		r173 := ctx.AllocRegExcept(d192.Reg)
		ctx.W.EmitCmpRegImm32(d192.Reg, 64)
		ctx.W.EmitSetcc(r173, scm.CcA)
		d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r173}
		ctx.BindReg(r173, &d193)
	}
	ctx.FreeDesc(&d192)
	lbl40 := ctx.W.ReserveLabel()
	lbl41 := ctx.W.ReserveLabel()
	if d193.Loc == scm.LocImm {
		if d193.Imm.Bool() {
			ctx.W.EmitJmp(lbl40)
		} else {
			d194 := d179
			if d194.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d194)
			}
			ctx.EmitStoreToStack(d194, 96)
			ctx.W.EmitJmp(lbl38)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d193.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl41)
		d195 := d179
		if d195.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d195.Loc == scm.LocStack || d195.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d195)
		}
		ctx.EmitStoreToStack(d195, 96)
		ctx.W.EmitJmp(lbl38)
		ctx.W.MarkLabel(lbl41)
		ctx.W.EmitJmp(lbl40)
	}
	ctx.FreeDesc(&d193)
	ctx.W.MarkLabel(lbl40)
	if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d174)
	}
	var d196 scm.JITValueDesc
	if d174.Loc == scm.LocImm {
		d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() / 64)}
	} else {
		r174 := ctx.AllocRegExcept(d174.Reg)
		ctx.W.EmitMovRegReg(r174, d174.Reg)
		ctx.W.EmitShrRegImm8(r174, 6)
		d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
		ctx.BindReg(r174, &d196)
	}
	if d196.Loc == scm.LocReg && d174.Loc == scm.LocReg && d196.Reg == d174.Reg {
		ctx.TransferReg(d174.Reg)
		d174.Loc = scm.LocNone
	}
	if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d196)
	}
	var d197 scm.JITValueDesc
	if d196.Loc == scm.LocImm {
		d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d196.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d196.Reg)
		ctx.W.EmitMovRegReg(scratch, d196.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d197)
	}
	if d197.Loc == scm.LocReg && d196.Loc == scm.LocReg && d197.Reg == d196.Reg {
		ctx.TransferReg(d196.Reg)
		d196.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d196)
	if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d197)
	}
	r175 := ctx.AllocReg()
	if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d197)
	}
	if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d175)
	}
	if d197.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r175, uint64(d197.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r175, d197.Reg)
		ctx.W.EmitShlRegImm8(r175, 3)
	}
	if d175.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
		ctx.W.EmitAddInt64(r175, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r175, d175.Reg)
	}
	r176 := ctx.AllocRegExcept(r175)
	ctx.W.EmitMovRegMem(r176, r175, 0)
	ctx.FreeReg(r175)
	d198 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r176}
	ctx.BindReg(r176, &d198)
	ctx.FreeDesc(&d197)
	if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d174)
	}
	var d199 scm.JITValueDesc
	if d174.Loc == scm.LocImm {
		d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() % 64)}
	} else {
		r177 := ctx.AllocRegExcept(d174.Reg)
		ctx.W.EmitMovRegReg(r177, d174.Reg)
		ctx.W.EmitAndRegImm32(r177, 63)
		d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
		ctx.BindReg(r177, &d199)
	}
	if d199.Loc == scm.LocReg && d174.Loc == scm.LocReg && d199.Reg == d174.Reg {
		ctx.TransferReg(d174.Reg)
		d174.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d174)
	d200 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d199)
	}
	var d201 scm.JITValueDesc
	if d200.Loc == scm.LocImm && d199.Loc == scm.LocImm {
		d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() - d199.Imm.Int())}
	} else if d199.Loc == scm.LocImm && d199.Imm.Int() == 0 {
		r178 := ctx.AllocRegExcept(d200.Reg)
		ctx.W.EmitMovRegReg(r178, d200.Reg)
		d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
		ctx.BindReg(r178, &d201)
	} else if d200.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d199.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d200.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d199.Reg)
		d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d201)
	} else if d199.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d200.Reg)
		ctx.W.EmitMovRegReg(scratch, d200.Reg)
		if d199.Imm.Int() >= -2147483648 && d199.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d199.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d199.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d201)
	} else {
		r179 := ctx.AllocRegExcept(d200.Reg)
		ctx.W.EmitMovRegReg(r179, d200.Reg)
		ctx.W.EmitSubInt64(r179, d199.Reg)
		d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
		ctx.BindReg(r179, &d201)
	}
	if d201.Loc == scm.LocReg && d200.Loc == scm.LocReg && d201.Reg == d200.Reg {
		ctx.TransferReg(d200.Reg)
		d200.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d199)
	if d198.Loc == scm.LocStack || d198.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d198)
	}
	if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d201)
	}
	var d202 scm.JITValueDesc
	if d198.Loc == scm.LocImm && d201.Loc == scm.LocImm {
		d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d198.Imm.Int()) >> uint64(d201.Imm.Int())))}
	} else if d201.Loc == scm.LocImm {
		r180 := ctx.AllocRegExcept(d198.Reg)
		ctx.W.EmitMovRegReg(r180, d198.Reg)
		ctx.W.EmitShrRegImm8(r180, uint8(d201.Imm.Int()))
		d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
		ctx.BindReg(r180, &d202)
	} else {
		{
			shiftSrc := d198.Reg
			r181 := ctx.AllocRegExcept(d198.Reg)
			ctx.W.EmitMovRegReg(r181, d198.Reg)
			shiftSrc = r181
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d201.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d201.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d201.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d202)
		}
	}
	if d202.Loc == scm.LocReg && d198.Loc == scm.LocReg && d202.Reg == d198.Reg {
		ctx.TransferReg(d198.Reg)
		d198.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d198)
	ctx.FreeDesc(&d201)
	if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d179)
	}
	if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d202)
	}
	var d203 scm.JITValueDesc
	if d179.Loc == scm.LocImm && d202.Loc == scm.LocImm {
		d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() | d202.Imm.Int())}
	} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
		d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d202.Reg}
		ctx.BindReg(d202.Reg, &d203)
	} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
		r182 := ctx.AllocRegExcept(d179.Reg)
		ctx.W.EmitMovRegReg(r182, d179.Reg)
		d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
		ctx.BindReg(r182, &d203)
	} else if d179.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d202.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d179.Imm.Int()))
		ctx.W.EmitOrInt64(scratch, d202.Reg)
		d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d203)
	} else if d202.Loc == scm.LocImm {
		r183 := ctx.AllocRegExcept(d179.Reg)
		ctx.W.EmitMovRegReg(r183, d179.Reg)
		if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
			ctx.W.EmitOrRegImm32(r183, int32(d202.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
			ctx.W.EmitOrInt64(r183, scm.RegR11)
		}
		d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
		ctx.BindReg(r183, &d203)
	} else {
		r184 := ctx.AllocRegExcept(d179.Reg)
		ctx.W.EmitMovRegReg(r184, d179.Reg)
		ctx.W.EmitOrInt64(r184, d202.Reg)
		d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
		ctx.BindReg(r184, &d203)
	}
	if d203.Loc == scm.LocReg && d179.Loc == scm.LocReg && d203.Reg == d179.Reg {
		ctx.TransferReg(d179.Reg)
		d179.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d202)
	d204 := d203
	if d204.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d204)
	}
	ctx.EmitStoreToStack(d204, 96)
	ctx.W.EmitJmp(lbl38)
	ctx.W.MarkLabel(lbl36)
	ctx.W.ResolveFixups()
	d205 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
	ctx.BindReg(r167, &d205)
	ctx.BindReg(r167, &d205)
	if r147 {
		ctx.UnprotectReg(r148)
	}
	if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d205)
	}
	if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d205)
	}
	var d206 scm.JITValueDesc
	if d205.Loc == scm.LocImm {
		d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d205.Imm.Int()))))}
	} else {
		r185 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r185, d205.Reg)
		d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
		ctx.BindReg(r185, &d206)
	}
	ctx.FreeDesc(&d205)
	var d207 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
		val := *(*int64)(unsafe.Pointer(fieldAddr))
		d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
		r186 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r186, thisptr.Reg, off)
		d207 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r186}
		ctx.BindReg(r186, &d207)
	}
	if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d206)
	}
	if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d207)
	}
	var d208 scm.JITValueDesc
	if d206.Loc == scm.LocImm && d207.Loc == scm.LocImm {
		d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() + d207.Imm.Int())}
	} else if d207.Loc == scm.LocImm && d207.Imm.Int() == 0 {
		r187 := ctx.AllocRegExcept(d206.Reg)
		ctx.W.EmitMovRegReg(r187, d206.Reg)
		d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
		ctx.BindReg(r187, &d208)
	} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
		d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d207.Reg}
		ctx.BindReg(d207.Reg, &d208)
	} else if d206.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d207.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d206.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d207.Reg)
		d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d208)
	} else if d207.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d206.Reg)
		ctx.W.EmitMovRegReg(scratch, d206.Reg)
		if d207.Imm.Int() >= -2147483648 && d207.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d207.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d207.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d208)
	} else {
		r188 := ctx.AllocRegExcept(d206.Reg)
		ctx.W.EmitMovRegReg(r188, d206.Reg)
		ctx.W.EmitAddInt64(r188, d207.Reg)
		d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
		ctx.BindReg(r188, &d208)
	}
	if d208.Loc == scm.LocReg && d206.Loc == scm.LocReg && d208.Reg == d206.Reg {
		ctx.TransferReg(d206.Reg)
		d206.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d206)
	ctx.FreeDesc(&d207)
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	lbl42 := ctx.W.ReserveLabel()
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d114)
	}
	var d209 scm.JITValueDesc
	if d114.Loc == scm.LocImm {
		d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d114.Imm.Int()))))}
	} else {
		r189 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r189, d114.Reg)
		ctx.W.EmitShlRegImm8(r189, 32)
		ctx.W.EmitShrRegImm8(r189, 32)
		d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
		ctx.BindReg(r189, &d209)
	}
	ctx.FreeDesc(&d114)
	var d210 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r190 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r190, thisptr.Reg, off)
		d210 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190}
		ctx.BindReg(r190, &d210)
	}
	if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d210)
	}
	if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d210)
	}
	var d211 scm.JITValueDesc
	if d210.Loc == scm.LocImm {
		d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d210.Imm.Int()))))}
	} else {
		r191 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r191, d210.Reg)
		ctx.W.EmitShlRegImm8(r191, 56)
		ctx.W.EmitShrRegImm8(r191, 56)
		d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
		ctx.BindReg(r191, &d211)
	}
	ctx.FreeDesc(&d210)
	if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d209)
	}
	if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d211)
	}
	var d212 scm.JITValueDesc
	if d209.Loc == scm.LocImm && d211.Loc == scm.LocImm {
		d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d209.Imm.Int() * d211.Imm.Int())}
	} else if d209.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d211.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d209.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d211.Reg)
		d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d212)
	} else if d211.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d209.Reg)
		ctx.W.EmitMovRegReg(scratch, d209.Reg)
		if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d211.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d212)
	} else {
		r192 := ctx.AllocRegExcept(d209.Reg)
		ctx.W.EmitMovRegReg(r192, d209.Reg)
		ctx.W.EmitImulInt64(r192, d211.Reg)
		d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
		ctx.BindReg(r192, &d212)
	}
	if d212.Loc == scm.LocReg && d209.Loc == scm.LocReg && d212.Reg == d209.Reg {
		ctx.TransferReg(d209.Reg)
		d209.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d209)
	ctx.FreeDesc(&d211)
	var d213 scm.JITValueDesc
	r193 := ctx.AllocReg()
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
		dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
		sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
		ctx.W.EmitMovRegImm64(r193, uint64(dataPtr))
		d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193, StackOff: int32(sliceLen)}
		ctx.BindReg(r193, &d213)
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
		ctx.W.EmitMovRegMem(r193, thisptr.Reg, off)
		d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
		ctx.BindReg(r193, &d213)
	}
	ctx.BindReg(r193, &d213)
	if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d212)
	}
	var d214 scm.JITValueDesc
	if d212.Loc == scm.LocImm {
		d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() / 64)}
	} else {
		r194 := ctx.AllocRegExcept(d212.Reg)
		ctx.W.EmitMovRegReg(r194, d212.Reg)
		ctx.W.EmitShrRegImm8(r194, 6)
		d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
		ctx.BindReg(r194, &d214)
	}
	if d214.Loc == scm.LocReg && d212.Loc == scm.LocReg && d214.Reg == d212.Reg {
		ctx.TransferReg(d212.Reg)
		d212.Loc = scm.LocNone
	}
	if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d214)
	}
	r195 := ctx.AllocReg()
	if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d214)
	}
	if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d213)
	}
	if d214.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r195, uint64(d214.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r195, d214.Reg)
		ctx.W.EmitShlRegImm8(r195, 3)
	}
	if d213.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
		ctx.W.EmitAddInt64(r195, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r195, d213.Reg)
	}
	r196 := ctx.AllocRegExcept(r195)
	ctx.W.EmitMovRegMem(r196, r195, 0)
	ctx.FreeReg(r195)
	d215 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r196}
	ctx.BindReg(r196, &d215)
	ctx.FreeDesc(&d214)
	if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d212)
	}
	var d216 scm.JITValueDesc
	if d212.Loc == scm.LocImm {
		d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
	} else {
		r197 := ctx.AllocRegExcept(d212.Reg)
		ctx.W.EmitMovRegReg(r197, d212.Reg)
		ctx.W.EmitAndRegImm32(r197, 63)
		d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
		ctx.BindReg(r197, &d216)
	}
	if d216.Loc == scm.LocReg && d212.Loc == scm.LocReg && d216.Reg == d212.Reg {
		ctx.TransferReg(d212.Reg)
		d212.Loc = scm.LocNone
	}
	if d215.Loc == scm.LocStack || d215.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d215)
	}
	if d216.Loc == scm.LocStack || d216.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d216)
	}
	var d217 scm.JITValueDesc
	if d215.Loc == scm.LocImm && d216.Loc == scm.LocImm {
		d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d215.Imm.Int()) << uint64(d216.Imm.Int())))}
	} else if d216.Loc == scm.LocImm {
		r198 := ctx.AllocRegExcept(d215.Reg)
		ctx.W.EmitMovRegReg(r198, d215.Reg)
		ctx.W.EmitShlRegImm8(r198, uint8(d216.Imm.Int()))
		d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
		ctx.BindReg(r198, &d217)
	} else {
		{
			shiftSrc := d215.Reg
			r199 := ctx.AllocRegExcept(d215.Reg)
			ctx.W.EmitMovRegReg(r199, d215.Reg)
			shiftSrc = r199
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d216.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d216.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d216.Reg)
			}
			ctx.W.EmitShlRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d217)
		}
	}
	if d217.Loc == scm.LocReg && d215.Loc == scm.LocReg && d217.Reg == d215.Reg {
		ctx.TransferReg(d215.Reg)
		d215.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d215)
	ctx.FreeDesc(&d216)
	var d218 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
		val := *(*bool)(unsafe.Pointer(fieldAddr))
		d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
		r200 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r200, thisptr.Reg, off)
		d218 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r200}
		ctx.BindReg(r200, &d218)
	}
	lbl43 := ctx.W.ReserveLabel()
	lbl44 := ctx.W.ReserveLabel()
	lbl45 := ctx.W.ReserveLabel()
	if d218.Loc == scm.LocImm {
		if d218.Imm.Bool() {
			ctx.W.EmitJmp(lbl43)
		} else {
			d219 := d217
			if d219.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d219.Loc == scm.LocStack || d219.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d219)
			}
			ctx.EmitStoreToStack(d219, 104)
			ctx.W.EmitJmp(lbl44)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d218.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl45)
		d220 := d217
		if d220.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d220.Loc == scm.LocStack || d220.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d220)
		}
		ctx.EmitStoreToStack(d220, 104)
		ctx.W.EmitJmp(lbl44)
		ctx.W.MarkLabel(lbl45)
		ctx.W.EmitJmp(lbl43)
	}
	ctx.FreeDesc(&d218)
	ctx.W.MarkLabel(lbl44)
	d221 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
	var d222 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r201 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r201, thisptr.Reg, off)
		d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
		ctx.BindReg(r201, &d222)
	}
	if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d222)
	}
	if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d222)
	}
	var d223 scm.JITValueDesc
	if d222.Loc == scm.LocImm {
		d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d222.Imm.Int()))))}
	} else {
		r202 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r202, d222.Reg)
		ctx.W.EmitShlRegImm8(r202, 56)
		ctx.W.EmitShrRegImm8(r202, 56)
		d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
		ctx.BindReg(r202, &d223)
	}
	ctx.FreeDesc(&d222)
	d224 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d223)
	}
	var d225 scm.JITValueDesc
	if d224.Loc == scm.LocImm && d223.Loc == scm.LocImm {
		d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d224.Imm.Int() - d223.Imm.Int())}
	} else if d223.Loc == scm.LocImm && d223.Imm.Int() == 0 {
		r203 := ctx.AllocRegExcept(d224.Reg)
		ctx.W.EmitMovRegReg(r203, d224.Reg)
		d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
		ctx.BindReg(r203, &d225)
	} else if d224.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d223.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d224.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d223.Reg)
		d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d225)
	} else if d223.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d224.Reg)
		ctx.W.EmitMovRegReg(scratch, d224.Reg)
		if d223.Imm.Int() >= -2147483648 && d223.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d223.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d223.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d225)
	} else {
		r204 := ctx.AllocRegExcept(d224.Reg)
		ctx.W.EmitMovRegReg(r204, d224.Reg)
		ctx.W.EmitSubInt64(r204, d223.Reg)
		d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
		ctx.BindReg(r204, &d225)
	}
	if d225.Loc == scm.LocReg && d224.Loc == scm.LocReg && d225.Reg == d224.Reg {
		ctx.TransferReg(d224.Reg)
		d224.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d223)
	if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d221)
	}
	if d225.Loc == scm.LocStack || d225.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d225)
	}
	var d226 scm.JITValueDesc
	if d221.Loc == scm.LocImm && d225.Loc == scm.LocImm {
		d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d221.Imm.Int()) >> uint64(d225.Imm.Int())))}
	} else if d225.Loc == scm.LocImm {
		r205 := ctx.AllocRegExcept(d221.Reg)
		ctx.W.EmitMovRegReg(r205, d221.Reg)
		ctx.W.EmitShrRegImm8(r205, uint8(d225.Imm.Int()))
		d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
		ctx.BindReg(r205, &d226)
	} else {
		{
			shiftSrc := d221.Reg
			r206 := ctx.AllocRegExcept(d221.Reg)
			ctx.W.EmitMovRegReg(r206, d221.Reg)
			shiftSrc = r206
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d225.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d225.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d225.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d226)
		}
	}
	if d226.Loc == scm.LocReg && d221.Loc == scm.LocReg && d226.Reg == d221.Reg {
		ctx.TransferReg(d221.Reg)
		d221.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d221)
	ctx.FreeDesc(&d225)
	r207 := ctx.AllocReg()
	if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d226)
	}
	ctx.EmitMovToReg(r207, d226)
	ctx.W.EmitJmp(lbl42)
	ctx.W.MarkLabel(lbl43)
	if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d212)
	}
	var d227 scm.JITValueDesc
	if d212.Loc == scm.LocImm {
		d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
	} else {
		r208 := ctx.AllocRegExcept(d212.Reg)
		ctx.W.EmitMovRegReg(r208, d212.Reg)
		ctx.W.EmitAndRegImm32(r208, 63)
		d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
		ctx.BindReg(r208, &d227)
	}
	if d227.Loc == scm.LocReg && d212.Loc == scm.LocReg && d227.Reg == d212.Reg {
		ctx.TransferReg(d212.Reg)
		d212.Loc = scm.LocNone
	}
	var d228 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
		val := *(*uint8)(unsafe.Pointer(fieldAddr))
		d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
		r209 := ctx.AllocReg()
		ctx.W.EmitMovRegMemB(r209, thisptr.Reg, off)
		d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
		ctx.BindReg(r209, &d228)
	}
	if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d228)
	}
	if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d228)
	}
	var d229 scm.JITValueDesc
	if d228.Loc == scm.LocImm {
		d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d228.Imm.Int()))))}
	} else {
		r210 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r210, d228.Reg)
		ctx.W.EmitShlRegImm8(r210, 56)
		ctx.W.EmitShrRegImm8(r210, 56)
		d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
		ctx.BindReg(r210, &d229)
	}
	ctx.FreeDesc(&d228)
	if d227.Loc == scm.LocStack || d227.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d227)
	}
	if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d229)
	}
	var d230 scm.JITValueDesc
	if d227.Loc == scm.LocImm && d229.Loc == scm.LocImm {
		d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() + d229.Imm.Int())}
	} else if d229.Loc == scm.LocImm && d229.Imm.Int() == 0 {
		r211 := ctx.AllocRegExcept(d227.Reg)
		ctx.W.EmitMovRegReg(r211, d227.Reg)
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
		ctx.BindReg(r211, &d230)
	} else if d227.Loc == scm.LocImm && d227.Imm.Int() == 0 {
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d229.Reg}
		ctx.BindReg(d229.Reg, &d230)
	} else if d227.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d229.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d227.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d229.Reg)
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d230)
	} else if d229.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d227.Reg)
		ctx.W.EmitMovRegReg(scratch, d227.Reg)
		if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d229.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d230)
	} else {
		r212 := ctx.AllocRegExcept(d227.Reg)
		ctx.W.EmitMovRegReg(r212, d227.Reg)
		ctx.W.EmitAddInt64(r212, d229.Reg)
		d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
		ctx.BindReg(r212, &d230)
	}
	if d230.Loc == scm.LocReg && d227.Loc == scm.LocReg && d230.Reg == d227.Reg {
		ctx.TransferReg(d227.Reg)
		d227.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d227)
	ctx.FreeDesc(&d229)
	if d230.Loc == scm.LocStack || d230.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d230)
	}
	var d231 scm.JITValueDesc
	if d230.Loc == scm.LocImm {
		d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d230.Imm.Int()) > uint64(64))}
	} else {
		r213 := ctx.AllocRegExcept(d230.Reg)
		ctx.W.EmitCmpRegImm32(d230.Reg, 64)
		ctx.W.EmitSetcc(r213, scm.CcA)
		d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r213}
		ctx.BindReg(r213, &d231)
	}
	ctx.FreeDesc(&d230)
	lbl46 := ctx.W.ReserveLabel()
	lbl47 := ctx.W.ReserveLabel()
	if d231.Loc == scm.LocImm {
		if d231.Imm.Bool() {
			ctx.W.EmitJmp(lbl46)
		} else {
			d232 := d217
			if d232.Loc == scm.LocNone {
				panic("jit: phi source has no location")
			}
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair {
				ctx.EnsureDesc(&d232)
			}
			ctx.EmitStoreToStack(d232, 104)
			ctx.W.EmitJmp(lbl44)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d231.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl47)
		d233 := d217
		if d233.Loc == scm.LocNone {
			panic("jit: phi source has no location")
		}
		if d233.Loc == scm.LocStack || d233.Loc == scm.LocStackPair {
			ctx.EnsureDesc(&d233)
		}
		ctx.EmitStoreToStack(d233, 104)
		ctx.W.EmitJmp(lbl44)
		ctx.W.MarkLabel(lbl47)
		ctx.W.EmitJmp(lbl46)
	}
	ctx.FreeDesc(&d231)
	ctx.W.MarkLabel(lbl46)
	if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d212)
	}
	var d234 scm.JITValueDesc
	if d212.Loc == scm.LocImm {
		d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() / 64)}
	} else {
		r214 := ctx.AllocRegExcept(d212.Reg)
		ctx.W.EmitMovRegReg(r214, d212.Reg)
		ctx.W.EmitShrRegImm8(r214, 6)
		d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
		ctx.BindReg(r214, &d234)
	}
	if d234.Loc == scm.LocReg && d212.Loc == scm.LocReg && d234.Reg == d212.Reg {
		ctx.TransferReg(d212.Reg)
		d212.Loc = scm.LocNone
	}
	if d234.Loc == scm.LocStack || d234.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d234)
	}
	var d235 scm.JITValueDesc
	if d234.Loc == scm.LocImm {
		d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d234.Imm.Int() + 1)}
	} else {
		scratch := ctx.AllocRegExcept(d234.Reg)
		ctx.W.EmitMovRegReg(scratch, d234.Reg)
		ctx.W.EmitAddRegImm32(scratch, int32(1))
		d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d235)
	}
	if d235.Loc == scm.LocReg && d234.Loc == scm.LocReg && d235.Reg == d234.Reg {
		ctx.TransferReg(d234.Reg)
		d234.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d234)
	if d235.Loc == scm.LocStack || d235.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d235)
	}
	r215 := ctx.AllocReg()
	if d235.Loc == scm.LocStack || d235.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d235)
	}
	if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d213)
	}
	if d235.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(r215, uint64(d235.Imm.Int())*8)
	} else {
		ctx.W.EmitMovRegReg(r215, d235.Reg)
		ctx.W.EmitShlRegImm8(r215, 3)
	}
	if d213.Loc == scm.LocImm {
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
		ctx.W.EmitAddInt64(r215, scm.RegR11)
	} else {
		ctx.W.EmitAddInt64(r215, d213.Reg)
	}
	r216 := ctx.AllocRegExcept(r215)
	ctx.W.EmitMovRegMem(r216, r215, 0)
	ctx.FreeReg(r215)
	d236 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
	ctx.BindReg(r216, &d236)
	ctx.FreeDesc(&d235)
	if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d212)
	}
	var d237 scm.JITValueDesc
	if d212.Loc == scm.LocImm {
		d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() % 64)}
	} else {
		r217 := ctx.AllocRegExcept(d212.Reg)
		ctx.W.EmitMovRegReg(r217, d212.Reg)
		ctx.W.EmitAndRegImm32(r217, 63)
		d237 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
		ctx.BindReg(r217, &d237)
	}
	if d237.Loc == scm.LocReg && d212.Loc == scm.LocReg && d237.Reg == d212.Reg {
		ctx.TransferReg(d212.Reg)
		d212.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d212)
	d238 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
	if d237.Loc == scm.LocStack || d237.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d237)
	}
	var d239 scm.JITValueDesc
	if d238.Loc == scm.LocImm && d237.Loc == scm.LocImm {
		d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d238.Imm.Int() - d237.Imm.Int())}
	} else if d237.Loc == scm.LocImm && d237.Imm.Int() == 0 {
		r218 := ctx.AllocRegExcept(d238.Reg)
		ctx.W.EmitMovRegReg(r218, d238.Reg)
		d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
		ctx.BindReg(r218, &d239)
	} else if d238.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d237.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d238.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d237.Reg)
		d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d239)
	} else if d237.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d238.Reg)
		ctx.W.EmitMovRegReg(scratch, d238.Reg)
		if d237.Imm.Int() >= -2147483648 && d237.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d237.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d237.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d239)
	} else {
		r219 := ctx.AllocRegExcept(d238.Reg)
		ctx.W.EmitMovRegReg(r219, d238.Reg)
		ctx.W.EmitSubInt64(r219, d237.Reg)
		d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
		ctx.BindReg(r219, &d239)
	}
	if d239.Loc == scm.LocReg && d238.Loc == scm.LocReg && d239.Reg == d238.Reg {
		ctx.TransferReg(d238.Reg)
		d238.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d237)
	if d236.Loc == scm.LocStack || d236.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d236)
	}
	if d239.Loc == scm.LocStack || d239.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d239)
	}
	var d240 scm.JITValueDesc
	if d236.Loc == scm.LocImm && d239.Loc == scm.LocImm {
		d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d236.Imm.Int()) >> uint64(d239.Imm.Int())))}
	} else if d239.Loc == scm.LocImm {
		r220 := ctx.AllocRegExcept(d236.Reg)
		ctx.W.EmitMovRegReg(r220, d236.Reg)
		ctx.W.EmitShrRegImm8(r220, uint8(d239.Imm.Int()))
		d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
		ctx.BindReg(r220, &d240)
	} else {
		{
			shiftSrc := d236.Reg
			r221 := ctx.AllocRegExcept(d236.Reg)
			ctx.W.EmitMovRegReg(r221, d236.Reg)
			shiftSrc = r221
			rcxUsed := ctx.FreeRegs&(1<<uint(scm.RegRCX)) == 0 && d239.Reg != scm.RegRCX
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
			}
			if d239.Reg != scm.RegRCX {
				ctx.W.EmitMovRegReg(scm.RegRCX, d239.Reg)
			}
			ctx.W.EmitShrRegCl(shiftSrc)
			if rcxUsed {
				ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
			}
			d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
			ctx.BindReg(shiftSrc, &d240)
		}
	}
	if d240.Loc == scm.LocReg && d236.Loc == scm.LocReg && d240.Reg == d236.Reg {
		ctx.TransferReg(d236.Reg)
		d236.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d236)
	ctx.FreeDesc(&d239)
	if d217.Loc == scm.LocStack || d217.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d217)
	}
	if d240.Loc == scm.LocStack || d240.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d240)
	}
	var d241 scm.JITValueDesc
	if d217.Loc == scm.LocImm && d240.Loc == scm.LocImm {
		d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d217.Imm.Int() | d240.Imm.Int())}
	} else if d217.Loc == scm.LocImm && d217.Imm.Int() == 0 {
		d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d240.Reg}
		ctx.BindReg(d240.Reg, &d241)
	} else if d240.Loc == scm.LocImm && d240.Imm.Int() == 0 {
		r222 := ctx.AllocRegExcept(d217.Reg)
		ctx.W.EmitMovRegReg(r222, d217.Reg)
		d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
		ctx.BindReg(r222, &d241)
	} else if d217.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d240.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d217.Imm.Int()))
		ctx.W.EmitOrInt64(scratch, d240.Reg)
		d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d241)
	} else if d240.Loc == scm.LocImm {
		r223 := ctx.AllocRegExcept(d217.Reg)
		ctx.W.EmitMovRegReg(r223, d217.Reg)
		if d240.Imm.Int() >= -2147483648 && d240.Imm.Int() <= 2147483647 {
			ctx.W.EmitOrRegImm32(r223, int32(d240.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d240.Imm.Int()))
			ctx.W.EmitOrInt64(r223, scm.RegR11)
		}
		d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
		ctx.BindReg(r223, &d241)
	} else {
		r224 := ctx.AllocRegExcept(d217.Reg)
		ctx.W.EmitMovRegReg(r224, d217.Reg)
		ctx.W.EmitOrInt64(r224, d240.Reg)
		d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
		ctx.BindReg(r224, &d241)
	}
	if d241.Loc == scm.LocReg && d217.Loc == scm.LocReg && d241.Reg == d217.Reg {
		ctx.TransferReg(d217.Reg)
		d217.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d240)
	d242 := d241
	if d242.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d242.Loc == scm.LocStack || d242.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d242)
	}
	ctx.EmitStoreToStack(d242, 104)
	ctx.W.EmitJmp(lbl44)
	ctx.W.MarkLabel(lbl42)
	ctx.W.ResolveFixups()
	d243 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
	ctx.BindReg(r207, &d243)
	ctx.BindReg(r207, &d243)
	ctx.FreeDesc(&d114)
	if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d243)
	}
	if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d243)
	}
	var d244 scm.JITValueDesc
	if d243.Loc == scm.LocImm {
		d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d243.Imm.Int()))))}
	} else {
		r225 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r225, d243.Reg)
		d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
		ctx.BindReg(r225, &d244)
	}
	ctx.FreeDesc(&d243)
	var d245 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
		val := *(*int64)(unsafe.Pointer(fieldAddr))
		d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
		r226 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r226, thisptr.Reg, off)
		d245 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r226}
		ctx.BindReg(r226, &d245)
	}
	if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d244)
	}
	if d245.Loc == scm.LocStack || d245.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d245)
	}
	var d246 scm.JITValueDesc
	if d244.Loc == scm.LocImm && d245.Loc == scm.LocImm {
		d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d244.Imm.Int() + d245.Imm.Int())}
	} else if d245.Loc == scm.LocImm && d245.Imm.Int() == 0 {
		r227 := ctx.AllocRegExcept(d244.Reg)
		ctx.W.EmitMovRegReg(r227, d244.Reg)
		d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
		ctx.BindReg(r227, &d246)
	} else if d244.Loc == scm.LocImm && d244.Imm.Int() == 0 {
		d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d245.Reg}
		ctx.BindReg(d245.Reg, &d246)
	} else if d244.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d245.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d244.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d245.Reg)
		d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d246)
	} else if d245.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d244.Reg)
		ctx.W.EmitMovRegReg(scratch, d244.Reg)
		if d245.Imm.Int() >= -2147483648 && d245.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d245.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d245.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d246)
	} else {
		r228 := ctx.AllocRegExcept(d244.Reg)
		ctx.W.EmitMovRegReg(r228, d244.Reg)
		ctx.W.EmitAddInt64(r228, d245.Reg)
		d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
		ctx.BindReg(r228, &d246)
	}
	if d246.Loc == scm.LocReg && d244.Loc == scm.LocReg && d246.Reg == d244.Reg {
		ctx.TransferReg(d244.Reg)
		d244.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d244)
	ctx.FreeDesc(&d245)
	if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&idxInt)
	}
	if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&idxInt)
	}
	var d247 scm.JITValueDesc
	if idxInt.Loc == scm.LocImm {
		d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
	} else {
		r229 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r229, idxInt.Reg)
		ctx.W.EmitShlRegImm8(r229, 32)
		ctx.W.EmitShrRegImm8(r229, 32)
		d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
		ctx.BindReg(r229, &d247)
	}
	ctx.FreeDesc(&idxInt)
	if d247.Loc == scm.LocStack || d247.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d247)
	}
	if d246.Loc == scm.LocStack || d246.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d246)
	}
	var d248 scm.JITValueDesc
	if d247.Loc == scm.LocImm && d246.Loc == scm.LocImm {
		d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d247.Imm.Int() - d246.Imm.Int())}
	} else if d246.Loc == scm.LocImm && d246.Imm.Int() == 0 {
		r230 := ctx.AllocRegExcept(d247.Reg)
		ctx.W.EmitMovRegReg(r230, d247.Reg)
		d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
		ctx.BindReg(r230, &d248)
	} else if d247.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d246.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d247.Imm.Int()))
		ctx.W.EmitSubInt64(scratch, d246.Reg)
		d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d248)
	} else if d246.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d247.Reg)
		ctx.W.EmitMovRegReg(scratch, d247.Reg)
		if d246.Imm.Int() >= -2147483648 && d246.Imm.Int() <= 2147483647 {
			ctx.W.EmitSubRegImm32(scratch, int32(d246.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d246.Imm.Int()))
			ctx.W.EmitSubInt64(scratch, scm.RegR11)
		}
		d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d248)
	} else {
		r231 := ctx.AllocRegExcept(d247.Reg)
		ctx.W.EmitMovRegReg(r231, d247.Reg)
		ctx.W.EmitSubInt64(r231, d246.Reg)
		d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
		ctx.BindReg(r231, &d248)
	}
	if d248.Loc == scm.LocReg && d247.Loc == scm.LocReg && d248.Reg == d247.Reg {
		ctx.TransferReg(d247.Reg)
		d247.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d247)
	ctx.FreeDesc(&d246)
	if d248.Loc == scm.LocStack || d248.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d248)
	}
	if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d208)
	}
	var d249 scm.JITValueDesc
	if d248.Loc == scm.LocImm && d208.Loc == scm.LocImm {
		d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d248.Imm.Int() * d208.Imm.Int())}
	} else if d248.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d208.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d248.Imm.Int()))
		ctx.W.EmitImulInt64(scratch, d208.Reg)
		d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d249)
	} else if d208.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d248.Reg)
		ctx.W.EmitMovRegReg(scratch, d248.Reg)
		if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
			ctx.W.EmitImulRegImm32(scratch, int32(d208.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
			ctx.W.EmitImulInt64(scratch, scm.RegR11)
		}
		d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d249)
	} else {
		r232 := ctx.AllocRegExcept(d248.Reg)
		ctx.W.EmitMovRegReg(r232, d248.Reg)
		ctx.W.EmitImulInt64(r232, d208.Reg)
		d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
		ctx.BindReg(r232, &d249)
	}
	if d249.Loc == scm.LocReg && d248.Loc == scm.LocReg && d249.Reg == d248.Reg {
		ctx.TransferReg(d248.Reg)
		d248.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d248)
	ctx.FreeDesc(&d208)
	if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d153)
	}
	if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d249)
	}
	var d250 scm.JITValueDesc
	if d153.Loc == scm.LocImm && d249.Loc == scm.LocImm {
		d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() + d249.Imm.Int())}
	} else if d249.Loc == scm.LocImm && d249.Imm.Int() == 0 {
		r233 := ctx.AllocRegExcept(d153.Reg)
		ctx.W.EmitMovRegReg(r233, d153.Reg)
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
		ctx.BindReg(r233, &d250)
	} else if d153.Loc == scm.LocImm && d153.Imm.Int() == 0 {
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d249.Reg}
		ctx.BindReg(d249.Reg, &d250)
	} else if d153.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d249.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d249.Reg)
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d250)
	} else if d249.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d153.Reg)
		ctx.W.EmitMovRegReg(scratch, d153.Reg)
		if d249.Imm.Int() >= -2147483648 && d249.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d249.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d249.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d250)
	} else {
		r234 := ctx.AllocRegExcept(d153.Reg)
		ctx.W.EmitMovRegReg(r234, d153.Reg)
		ctx.W.EmitAddInt64(r234, d249.Reg)
		d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
		ctx.BindReg(r234, &d250)
	}
	if d250.Loc == scm.LocReg && d153.Loc == scm.LocReg && d250.Reg == d153.Reg {
		ctx.TransferReg(d153.Reg)
		d153.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d249)
	if d250.Loc == scm.LocStack || d250.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d250)
	}
	if d250.Loc == scm.LocStack || d250.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d250)
	}
	var d251 scm.JITValueDesc
	if d250.Loc == scm.LocImm {
		d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d250.Imm.Int()))}
	} else {
		ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d250.Reg)
		d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d250.Reg}
		ctx.BindReg(d250.Reg, &d251)
	}
	ctx.FreeDesc(&d250)
	if d251.Loc == scm.LocStack || d251.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d251)
	}
	ctx.W.EmitMakeFloat(result, d251)
	if d251.Loc == scm.LocReg {
		ctx.FreeReg(d251.Reg)
	}
	result.Type = scm.TagFloat
	ctx.W.EmitJmp(lbl0)
	ctx.W.MarkLabel(lbl30)
	var d252 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
		val := *(*uint64)(unsafe.Pointer(fieldAddr))
		d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
	} else {
		off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
		r235 := ctx.AllocReg()
		ctx.W.EmitMovRegMem(r235, thisptr.Reg, off)
		d252 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r235}
		ctx.BindReg(r235, &d252)
	}
	if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d252)
	}
	if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d252)
	}
	var d253 scm.JITValueDesc
	if d252.Loc == scm.LocImm {
		d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d252.Imm.Int()))))}
	} else {
		r236 := ctx.AllocReg()
		ctx.W.EmitMovRegReg(r236, d252.Reg)
		d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
		ctx.BindReg(r236, &d253)
	}
	ctx.FreeDesc(&d252)
	if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d153)
	}
	if d253.Loc == scm.LocStack || d253.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d253)
	}
	var d254 scm.JITValueDesc
	if d153.Loc == scm.LocImm && d253.Loc == scm.LocImm {
		d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d153.Imm.Int() == d253.Imm.Int())}
	} else if d253.Loc == scm.LocImm {
		r237 := ctx.AllocRegExcept(d153.Reg)
		if d253.Imm.Int() >= -2147483648 && d253.Imm.Int() <= 2147483647 {
			ctx.W.EmitCmpRegImm32(d153.Reg, int32(d253.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d253.Imm.Int()))
			ctx.W.EmitCmpInt64(d153.Reg, scm.RegR11)
		}
		ctx.W.EmitSetcc(r237, scm.CcE)
		d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r237}
		ctx.BindReg(r237, &d254)
	} else if d153.Loc == scm.LocImm {
		r238 := ctx.AllocReg()
		ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d153.Imm.Int()))
		ctx.W.EmitCmpInt64(scm.RegR11, d253.Reg)
		ctx.W.EmitSetcc(r238, scm.CcE)
		d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r238}
		ctx.BindReg(r238, &d254)
	} else {
		r239 := ctx.AllocRegExcept(d153.Reg)
		ctx.W.EmitCmpInt64(d153.Reg, d253.Reg)
		ctx.W.EmitSetcc(r239, scm.CcE)
		d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
		ctx.BindReg(r239, &d254)
	}
	ctx.FreeDesc(&d153)
	ctx.FreeDesc(&d253)
	lbl48 := ctx.W.ReserveLabel()
	lbl49 := ctx.W.ReserveLabel()
	if d254.Loc == scm.LocImm {
		if d254.Imm.Bool() {
			ctx.W.EmitJmp(lbl48)
		} else {
			ctx.W.EmitJmp(lbl31)
		}
	} else {
		ctx.W.EmitCmpRegImm32(d254.Reg, 0)
		ctx.W.EmitJcc(scm.CcNE, lbl49)
		ctx.W.EmitJmp(lbl31)
		ctx.W.MarkLabel(lbl49)
		ctx.W.EmitJmp(lbl48)
	}
	ctx.FreeDesc(&d254)
	ctx.W.MarkLabel(lbl34)
	if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d159)
	}
	if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d160)
	}
	var d255 scm.JITValueDesc
	if d159.Loc == scm.LocImm && d160.Loc == scm.LocImm {
		d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() + d160.Imm.Int())}
	} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
		r240 := ctx.AllocRegExcept(d159.Reg)
		ctx.W.EmitMovRegReg(r240, d159.Reg)
		d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
		ctx.BindReg(r240, &d255)
	} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
		d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
		ctx.BindReg(d160.Reg, &d255)
	} else if d159.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d160.Reg)
		ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
		ctx.W.EmitAddInt64(scratch, d160.Reg)
		d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d255)
	} else if d160.Loc == scm.LocImm {
		scratch := ctx.AllocRegExcept(d159.Reg)
		ctx.W.EmitMovRegReg(scratch, d159.Reg)
		if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
			ctx.W.EmitAddRegImm32(scratch, int32(d160.Imm.Int()))
		} else {
			ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
			ctx.W.EmitAddInt64(scratch, scm.RegR11)
		}
		d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
		ctx.BindReg(scratch, &d255)
	} else {
		r241 := ctx.AllocRegExcept(d159.Reg)
		ctx.W.EmitMovRegReg(r241, d159.Reg)
		ctx.W.EmitAddInt64(r241, d160.Reg)
		d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
		ctx.BindReg(r241, &d255)
	}
	if d255.Loc == scm.LocImm {
		d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: d255.Type, Imm: scm.NewInt(int64(uint64(d255.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d255.Reg, 32)
		ctx.W.EmitShrRegImm8(d255.Reg, 32)
	}
	if d255.Loc == scm.LocReg && d159.Loc == scm.LocReg && d255.Reg == d159.Reg {
		ctx.TransferReg(d159.Reg)
		d159.Loc = scm.LocNone
	}
	if d255.Loc == scm.LocStack || d255.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d255)
	}
	var d256 scm.JITValueDesc
	if d255.Loc == scm.LocImm {
		d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d255.Imm.Int() / 2)}
	} else {
		r242 := ctx.AllocRegExcept(d255.Reg)
		ctx.W.EmitMovRegReg(r242, d255.Reg)
		ctx.W.EmitShrRegImm8(r242, 1)
		d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
		ctx.BindReg(r242, &d256)
	}
	if d256.Loc == scm.LocImm {
		d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: d256.Type, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d256.Reg, 32)
		ctx.W.EmitShrRegImm8(d256.Reg, 32)
	}
	if d256.Loc == scm.LocReg && d255.Loc == scm.LocReg && d256.Reg == d255.Reg {
		ctx.TransferReg(d255.Reg)
		d255.Loc = scm.LocNone
	}
	ctx.FreeDesc(&d255)
	d257 := d256
	if d257.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d257)
	}
	d258 := d257
	if d258.Loc == scm.LocImm {
		d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: d258.Type, Imm: scm.NewInt(int64(uint64(d258.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d258.Reg, 32)
		ctx.W.EmitShrRegImm8(d258.Reg, 32)
	}
	ctx.EmitStoreToStack(d258, 0)
	d259 := d159
	if d259.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d259.Loc == scm.LocStack || d259.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d259)
	}
	d260 := d259
	if d260.Loc == scm.LocImm {
		d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: d260.Type, Imm: scm.NewInt(int64(uint64(d260.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d260.Reg, 32)
		ctx.W.EmitShrRegImm8(d260.Reg, 32)
	}
	ctx.EmitStoreToStack(d260, 8)
	d261 := d160
	if d261.Loc == scm.LocNone {
		panic("jit: phi source has no location")
	}
	if d261.Loc == scm.LocStack || d261.Loc == scm.LocStackPair {
		ctx.EnsureDesc(&d261)
	}
	d262 := d261
	if d262.Loc == scm.LocImm {
		d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: d262.Type, Imm: scm.NewInt(int64(uint64(d262.Imm.Int()) & 0xffffffff))}
	} else {
		ctx.W.EmitShlRegImm8(d262.Reg, 32)
		ctx.W.EmitShrRegImm8(d262.Reg, 32)
	}
	ctx.EmitStoreToStack(d262, 16)
	ctx.W.EmitJmp(lbl1)
	ctx.W.MarkLabel(lbl48)
	ctx.W.EmitMakeNil(result)
	result.Type = scm.TagNil
	ctx.W.EmitJmp(lbl0)
	ctx.W.MarkLabel(lbl0)
	ctx.W.ResolveFixups()
	if idxPinned {
		ctx.UnprotectReg(idxPinnedReg)
	}
	ctx.W.PatchInt32(r0, int32(112))
	ctx.W.EmitAddRSP32(int32(112))
	return result
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
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.scan(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if i != 0 && v == s.lastValue.Load()+s.lastStride {
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
		if s.lastValueFirst {
			// learn stride from second value
			s.lastValueFirst = false
			s.lastStride = v - s.lastValue.Load()
			s.lastValue.Store(v)
			s.stride.build(s.seqCount-1, scm.NewInt(s.lastStride))
		} else if i != 0 && v == s.lastValue.Load()+s.lastStride {
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

	s.lastValue.Store(int64(s.seqCount / 2)) // initialize pivot cache

	/* debug output of the sequence:
	for i := uint(0); i < s.seqCount; i++ {
		fmt.Println(s.recordId.GetValue(i),":",s.start.GetValue(i),":",s.stride.GetValue(i))
	}*/
}
func (s *StorageSeq) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}
