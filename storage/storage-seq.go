/*
Copyright (C) 2023  Carl-Philip Hänsch

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
	min := uint32(0)
	max := s.seqCount - 1
	for {
		recid := int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
			max = pivot - 1
			pivot--
		} else {
			min = pivot
			pivot++
		}
		if min == max {
			break // we found the sequence for i
		}

		// also read the next neighbour (we are in the cache line anyway and we achieve O(1) in case the same sequence is read again!)
		recid = int64(s.recordId.GetValueUInt(pivot)) + s.recordId.offset
		if i < uint32(recid) {
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
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
			} else {
				idxInt = idx
			}
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			r1 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0
				ctx.W.EmitMovRegMem64(r1, fieldAddr)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0)
				ctx.W.EmitMovRegMem(r1, thisptr.Reg, off)
			}
			d0 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r1}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMem32(r2, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMemL(r3, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			}
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitSubInt64(d2.Reg, scm.RegR11)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d2.Reg}
			}
			if d3.Loc == scm.LocReg && d2.Loc == scm.LocReg && d3.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.EmitStoreToStack(d0, 0)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 8)
			ctx.EmitStoreToStack(d3, 16)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r4 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r4, 0)
			ctx.ProtectReg(r4)
			d4 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r4}
			r5 := ctx.AllocRegExcept(r4)
			ctx.EmitLoadFromStack(r5, 8)
			ctx.ProtectReg(r5)
			d5 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r5}
			r6 := ctx.AllocRegExcept(r4, r5)
			ctx.EmitLoadFromStack(r6, 16)
			ctx.ProtectReg(r6)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r6}
			ctx.UnprotectReg(r4)
			ctx.UnprotectReg(r5)
			ctx.UnprotectReg(r6)
			if d4.Loc == scm.LocReg { ctx.ProtectReg(d4.Reg) }
			r7 := ctx.AllocReg()
			lbl2 := ctx.W.ReserveLabel()
			var d8 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r8, thisptr.Reg, off)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			}
			var d10 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() * d8.Imm.Int())}
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d8.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d4.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				ctx.W.EmitImulInt64(r9, d8.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			}
			if d10.Loc == scm.LocReg && d4.Loc == scm.LocReg && d10.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			}
			var d12 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r11, d10.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			}
			if d12.Loc == scm.LocReg && d10.Loc == scm.LocReg && d12.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			r12 := ctx.AllocReg()
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d12.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d12.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r12, d11.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d13 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.FreeDesc(&d12)
			var d14 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r14, d10.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			}
			if d14.Loc == scm.LocReg && d10.Loc == scm.LocReg && d14.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			var d15 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) << uint64(d14.Imm.Int())))}
			} else if d14.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d13.Reg, uint8(d14.Imm.Int()))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else {
				{
					shiftSrc := d13.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d14.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d14.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d14.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d15.Loc == scm.LocReg && d13.Loc == scm.LocReg && d15.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d14)
			var d16 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d16.Loc == scm.LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.EmitStoreToStack(d15, 72)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl5)
				ctx.EmitStoreToStack(d15, 72)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl4)
			r16 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r16, 72)
			ctx.ProtectReg(r16)
			d17 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r16}
			ctx.UnprotectReg(r16)
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
			}
			d20 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d18.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d18.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.W.EmitSubInt64(d20.Reg, scm.RegR11)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			} else {
				ctx.W.EmitSubInt64(d20.Reg, d18.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			var d22 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d17.Reg, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else {
				{
					shiftSrc := d17.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d22.Loc == scm.LocReg && d17.Loc == scm.LocReg && d22.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d21)
			ctx.EmitMovToReg(r7, d22)
			ctx.W.EmitJmp(lbl2)
			ctx.FreeDesc(&d22)
			ctx.W.MarkLabel(lbl3)
			var d23 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				r18 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r18, d10.Reg)
				ctx.W.EmitAndRegImm32(r18, 63)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			}
			if d23.Loc == scm.LocReg && d10.Loc == scm.LocReg && d23.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
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
			}
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d24.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(d23.Reg, scm.RegR11)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else {
				ctx.W.EmitAddInt64(d23.Reg, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d24)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d26.Imm.Int() > 64)}
			} else {
				r20 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d26.Reg, 64)
				ctx.W.EmitSetcc(r20, scm.CcG)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r20}
			}
			ctx.FreeDesc(&d26)
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.EmitStoreToStack(d15, 72)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
				ctx.EmitStoreToStack(d15, 72)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d27)
			ctx.W.MarkLabel(lbl6)
			var d28 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() / 64)}
			} else {
				r21 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r21, d10.Reg)
				ctx.W.EmitShrRegImm8(r21, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			}
			if d28.Loc == scm.LocReg && d10.Loc == scm.LocReg && d28.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d28.Reg, scm.RegR11)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
			}
			if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			r22 := ctx.AllocReg()
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r22, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r22, d29.Reg)
				ctx.W.EmitShlRegImm8(r22, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r22, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r22, d11.Reg)
			}
			r23 := ctx.AllocRegExcept(r22)
			ctx.W.EmitMovRegMem(r23, r22, 0)
			ctx.FreeReg(r22)
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.FreeDesc(&d29)
			var d31 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d10.Reg, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d10.Reg}
			}
			if d31.Loc == scm.LocReg && d10.Loc == scm.LocReg && d31.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			d32 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d31.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
				ctx.W.EmitSubInt64(d32.Reg, scm.RegR11)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else {
				ctx.W.EmitSubInt64(d32.Reg, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			}
			if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			var d34 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d30.Reg, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else {
				{
					shiftSrc := d30.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d33.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d33.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d34.Loc == scm.LocReg && d30.Loc == scm.LocReg && d34.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d33)
			var d35 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() | d34.Imm.Int())}
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d15.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitOrInt64(d15.Reg, scratch)
				ctx.FreeReg(scratch)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			} else {
				ctx.W.EmitOrInt64(d15.Reg, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			}
			if d35.Loc == scm.LocReg && d15.Loc == scm.LocReg && d35.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EmitStoreToStack(d35, 72)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			d36 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			if d4.Loc == scm.LocReg { ctx.UnprotectReg(d4.Reg) }
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r24, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			}
			var d39 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d38.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
				ctx.W.EmitAddInt64(d36.Reg, scm.RegR11)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else {
				ctx.W.EmitAddInt64(d36.Reg, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			}
			if d39.Loc == scm.LocReg && d36.Loc == scm.LocReg && d39.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d38)
			var d41 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(idxInt.Imm.Int() < d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm {
				r25 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d39.Imm.Int()))
				ctx.W.EmitSetcc(r25, scm.CcL)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
			} else if idxInt.Loc == scm.LocImm {
				r26 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d39.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r26, scm.CcL)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r26}
			} else {
				r27 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d39.Reg)
				ctx.W.EmitSetcc(r27, scm.CcL)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r27}
			}
			ctx.FreeDesc(&d39)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d41.Loc == scm.LocImm {
				if d41.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d41.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d41)
			ctx.W.MarkLabel(lbl9)
			var d42 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(scratch, d4.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d42.Loc == scm.LocReg && d4.Loc == scm.LocReg && d42.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EmitStoreToStack(d42, 32)
			ctx.EmitStoreToStack(d4, 40)
			ctx.EmitStoreToStack(d6, 48)
			lbl11 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			var d43 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitSubInt64(scratch, scm.RegR11)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d43.Loc == scm.LocReg && d4.Loc == scm.LocReg && d43.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d44 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitSubInt64(scratch, scm.RegR11)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d44.Loc == scm.LocReg && d4.Loc == scm.LocReg && d44.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EmitStoreToStack(d44, 32)
			ctx.EmitStoreToStack(d5, 40)
			ctx.EmitStoreToStack(d43, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			r28 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r28, 32)
			ctx.ProtectReg(r28)
			d45 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r28}
			r29 := ctx.AllocRegExcept(r28)
			ctx.EmitLoadFromStack(r29, 40)
			ctx.ProtectReg(r29)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r29}
			r30 := ctx.AllocRegExcept(r28, r29)
			ctx.EmitLoadFromStack(r30, 48)
			ctx.ProtectReg(r30)
			d47 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r30}
			ctx.UnprotectReg(r28)
			ctx.UnprotectReg(r29)
			ctx.UnprotectReg(r30)
			var d48 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d46.Imm.Int() == d47.Imm.Int())}
			} else if d47.Loc == scm.LocImm {
				r31 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitCmpRegImm32(d46.Reg, int32(d47.Imm.Int()))
				ctx.W.EmitSetcc(r31, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
			} else if d46.Loc == scm.LocImm {
				r32 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d46.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d47.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r32, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
			} else {
				r33 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitCmpInt64(d46.Reg, d47.Reg)
				ctx.W.EmitSetcc(r33, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r33}
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d48.Loc == scm.LocImm {
				if d48.Imm.Bool() {
					ctx.EmitStoreToStack(d46, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d48.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
				ctx.EmitStoreToStack(d46, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d48)
			ctx.W.MarkLabel(lbl13)
			if d45.Loc == scm.LocReg { ctx.ProtectReg(d45.Reg) }
			r34 := ctx.AllocReg()
			lbl15 := ctx.W.ReserveLabel()
			var d50 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r35, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			}
			var d52 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() * d50.Imm.Int())}
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d50.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d50.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d45.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r36 := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(r36, d45.Reg)
				ctx.W.EmitImulInt64(r36, d50.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			}
			if d52.Loc == scm.LocReg && d45.Loc == scm.LocReg && d52.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r37 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r37, d52.Reg)
				ctx.W.EmitShrRegImm8(r37, 6)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			}
			if d53.Loc == scm.LocReg && d52.Loc == scm.LocReg && d53.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			r38 := ctx.AllocReg()
			if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r38, uint64(d53.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r38, d53.Reg)
				ctx.W.EmitShlRegImm8(r38, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r38, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r38, d11.Reg)
			}
			r39 := ctx.AllocRegExcept(r38)
			ctx.W.EmitMovRegMem(r39, r38, 0)
			ctx.FreeReg(r38)
			d54 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.FreeDesc(&d53)
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r40 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r40, d52.Reg)
				ctx.W.EmitAndRegImm32(r40, 63)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			}
			if d55.Loc == scm.LocReg && d52.Loc == scm.LocReg && d55.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			var d56 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d54.Imm.Int()) << uint64(d55.Imm.Int())))}
			} else if d55.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d54.Reg, uint8(d55.Imm.Int()))
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
			} else {
				{
					shiftSrc := d54.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d55.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d55.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d55.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d56.Loc == scm.LocReg && d54.Loc == scm.LocReg && d56.Reg == d54.Reg {
				ctx.TransferReg(d54.Reg)
				d54.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d54)
			ctx.FreeDesc(&d55)
			var d57 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r41, thisptr.Reg, off)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d57.Loc == scm.LocImm {
				if d57.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.EmitStoreToStack(d56, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d57.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.EmitStoreToStack(d56, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d57)
			ctx.W.MarkLabel(lbl17)
			r42 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r42, 80)
			ctx.ProtectReg(r42)
			d58 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r42}
			ctx.UnprotectReg(r42)
			var d59 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r43, thisptr.Reg, off)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
			}
			d61 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() - d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d61.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d59.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d59.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
				ctx.W.EmitSubInt64(d61.Reg, scm.RegR11)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else {
				ctx.W.EmitSubInt64(d61.Reg, d59.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			var d63 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) >> uint64(d62.Imm.Int())))}
			} else if d62.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d58.Reg, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d58.Reg}
			} else {
				{
					shiftSrc := d58.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d62.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d62.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d62.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d63.Loc == scm.LocReg && d58.Loc == scm.LocReg && d63.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.FreeDesc(&d62)
			ctx.EmitMovToReg(r34, d63)
			ctx.W.EmitJmp(lbl15)
			ctx.FreeDesc(&d63)
			ctx.W.MarkLabel(lbl16)
			var d64 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				r44 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r44, d52.Reg)
				ctx.W.EmitAndRegImm32(r44, 63)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			}
			if d64.Loc == scm.LocReg && d52.Loc == scm.LocReg && d64.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			var d65 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r45, thisptr.Reg, off)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			}
			var d67 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() + d65.Imm.Int())}
			} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d64.Reg}
			} else if d64.Loc == scm.LocImm && d64.Imm.Int() == 0 {
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d65.Reg)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d65.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d65.Imm.Int()))
				ctx.W.EmitAddInt64(d64.Reg, scm.RegR11)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d64.Reg}
			} else {
				ctx.W.EmitAddInt64(d64.Reg, d65.Reg)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d64.Reg}
			}
			if d67.Loc == scm.LocReg && d64.Loc == scm.LocReg && d67.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d64)
			ctx.FreeDesc(&d65)
			var d68 scm.JITValueDesc
			if d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d67.Imm.Int() > 64)}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d67.Reg, 64)
				ctx.W.EmitSetcc(r46, scm.CcG)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
			}
			ctx.FreeDesc(&d67)
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d68.Loc == scm.LocImm {
				if d68.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.EmitStoreToStack(d56, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d68.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl20)
				ctx.EmitStoreToStack(d56, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d68)
			ctx.W.MarkLabel(lbl19)
			var d69 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() / 64)}
			} else {
				r47 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r47, d52.Reg)
				ctx.W.EmitShrRegImm8(r47, 6)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			}
			if d69.Loc == scm.LocReg && d52.Loc == scm.LocReg && d69.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d69.Reg, scm.RegR11)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d69.Reg}
			}
			if d70.Loc == scm.LocReg && d69.Loc == scm.LocReg && d70.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d69)
			r48 := ctx.AllocReg()
			if d70.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r48, uint64(d70.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r48, d70.Reg)
				ctx.W.EmitShlRegImm8(r48, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r48, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r48, d11.Reg)
			}
			r49 := ctx.AllocRegExcept(r48)
			ctx.W.EmitMovRegMem(r49, r48, 0)
			ctx.FreeReg(r48)
			d71 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
			ctx.FreeDesc(&d70)
			var d72 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d52.Reg, 63)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			}
			if d72.Loc == scm.LocReg && d52.Loc == scm.LocReg && d72.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			d73 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d74 scm.JITValueDesc
			if d73.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d73.Imm.Int() - d72.Imm.Int())}
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d73.Reg}
			} else if d73.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d73.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d72.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
				ctx.W.EmitSubInt64(d73.Reg, scm.RegR11)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d73.Reg}
			} else {
				ctx.W.EmitSubInt64(d73.Reg, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d73.Reg}
			}
			if d74.Loc == scm.LocReg && d73.Loc == scm.LocReg && d74.Reg == d73.Reg {
				ctx.TransferReg(d73.Reg)
				d73.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			var d75 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d71.Imm.Int()) >> uint64(d74.Imm.Int())))}
			} else if d74.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d71.Reg, uint8(d74.Imm.Int()))
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			} else {
				{
					shiftSrc := d71.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d74.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d74.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d74.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d75.Loc == scm.LocReg && d71.Loc == scm.LocReg && d75.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.FreeDesc(&d74)
			var d76 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() | d75.Imm.Int())}
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d75.Imm.Int()))
				ctx.W.EmitOrInt64(d56.Reg, scratch)
				ctx.FreeReg(scratch)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
			} else {
				ctx.W.EmitOrInt64(d56.Reg, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
			}
			if d76.Loc == scm.LocReg && d56.Loc == scm.LocReg && d76.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EmitStoreToStack(d76, 80)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl15)
			ctx.W.ResolveFixups()
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			if d45.Loc == scm.LocReg { ctx.UnprotectReg(d45.Reg) }
			var d79 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
			}
			var d80 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d77.Imm.Int() + d79.Imm.Int())}
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg}
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d77.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d79.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
				ctx.W.EmitAddInt64(d77.Reg, scm.RegR11)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
			} else {
				ctx.W.EmitAddInt64(d77.Reg, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
			}
			if d80.Loc == scm.LocReg && d77.Loc == scm.LocReg && d80.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d79)
			var d82 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(idxInt.Imm.Int() < d80.Imm.Int())}
			} else if d80.Loc == scm.LocImm {
				r51 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d80.Imm.Int()))
				ctx.W.EmitSetcc(r51, scm.CcL)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
			} else if idxInt.Loc == scm.LocImm {
				r52 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d80.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r52, scm.CcL)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
			} else {
				r53 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d80.Reg)
				ctx.W.EmitSetcc(r53, scm.CcL)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
			}
			ctx.FreeDesc(&d80)
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d82.Loc == scm.LocImm {
				if d82.Imm.Bool() {
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d82.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl22)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.FreeDesc(&d82)
			ctx.W.MarkLabel(lbl12)
			r54 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r54, 24)
			ctx.ProtectReg(r54)
			d83 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r54}
			ctx.UnprotectReg(r54)
			if d83.Loc == scm.LocReg { ctx.ProtectReg(d83.Reg) }
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0))
				if d83.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
					ctx.W.EmitStoreRegMem(scratch, baseReg, 0)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitStoreRegMem(d83.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0)
				if d83.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
					ctx.W.EmitStoreRegMem(scratch, thisptr.Reg, off)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitStoreRegMem(d83.Reg, thisptr.Reg, off)
				}
			}
			if d83.Loc == scm.LocReg { ctx.UnprotectReg(d83.Reg) }
			if d83.Loc == scm.LocReg { ctx.ProtectReg(d83.Reg) }
			r55 := ctx.AllocReg()
			lbl24 := ctx.W.ReserveLabel()
			var d86 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r56, thisptr.Reg, off)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			}
			var d88 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() * d86.Imm.Int())}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d86.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d83.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r57 := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegReg(r57, d83.Reg)
				ctx.W.EmitImulInt64(r57, d86.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
			}
			if d88.Loc == scm.LocReg && d83.Loc == scm.LocReg && d88.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r58, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
			}
			var d90 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() / 64)}
			} else {
				r59 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r59, d88.Reg)
				ctx.W.EmitShrRegImm8(r59, 6)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			}
			if d90.Loc == scm.LocReg && d88.Loc == scm.LocReg && d90.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			r60 := ctx.AllocReg()
			if d90.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r60, uint64(d90.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r60, d90.Reg)
				ctx.W.EmitShlRegImm8(r60, 3)
			}
			if d89.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(r60, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r60, d89.Reg)
			}
			r61 := ctx.AllocRegExcept(r60)
			ctx.W.EmitMovRegMem(r61, r60, 0)
			ctx.FreeReg(r60)
			d91 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
			ctx.FreeDesc(&d90)
			var d92 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				r62 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r62, d88.Reg)
				ctx.W.EmitAndRegImm32(r62, 63)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			}
			if d92.Loc == scm.LocReg && d88.Loc == scm.LocReg && d92.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			var d93 scm.JITValueDesc
			if d91.Loc == scm.LocImm && d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d91.Imm.Int()) << uint64(d92.Imm.Int())))}
			} else if d92.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d91.Reg, uint8(d92.Imm.Int()))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
			} else {
				{
					shiftSrc := d91.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d92.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d92.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d92.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d93.Loc == scm.LocReg && d91.Loc == scm.LocReg && d93.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r63, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d94.Loc == scm.LocImm {
				if d94.Imm.Bool() {
					ctx.W.EmitJmp(lbl25)
				} else {
					ctx.EmitStoreToStack(d93, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d94.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
				ctx.EmitStoreToStack(d93, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl25)
			}
			ctx.FreeDesc(&d94)
			ctx.W.MarkLabel(lbl26)
			r64 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r64, 88)
			ctx.ProtectReg(r64)
			d95 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r64}
			ctx.UnprotectReg(r64)
			var d96 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r65, thisptr.Reg, off)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r65}
			}
			d98 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d99 scm.JITValueDesc
			if d98.Loc == scm.LocImm && d96.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() - d96.Imm.Int())}
			} else if d96.Loc == scm.LocImm && d96.Imm.Int() == 0 {
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
			} else if d98.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d98.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d96.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d96.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d96.Imm.Int()))
				ctx.W.EmitSubInt64(d98.Reg, scm.RegR11)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
			} else {
				ctx.W.EmitSubInt64(d98.Reg, d96.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
			}
			if d99.Loc == scm.LocReg && d98.Loc == scm.LocReg && d99.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d96)
			var d100 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d99.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d95.Imm.Int()) >> uint64(d99.Imm.Int())))}
			} else if d99.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d95.Reg, uint8(d99.Imm.Int()))
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else {
				{
					shiftSrc := d95.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d99.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d99.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d99.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d100.Loc == scm.LocReg && d95.Loc == scm.LocReg && d100.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			ctx.FreeDesc(&d99)
			ctx.EmitMovToReg(r55, d100)
			ctx.W.EmitJmp(lbl24)
			ctx.FreeDesc(&d100)
			ctx.W.MarkLabel(lbl25)
			var d101 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				r66 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r66, d88.Reg)
				ctx.W.EmitAndRegImm32(r66, 63)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			}
			if d101.Loc == scm.LocReg && d88.Loc == scm.LocReg && d101.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r67, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r67}
			}
			var d104 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() + d102.Imm.Int())}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d102.Reg}
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d102.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d102.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(d101.Reg, scm.RegR11)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else {
				ctx.W.EmitAddInt64(d101.Reg, d102.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			}
			if d104.Loc == scm.LocReg && d101.Loc == scm.LocReg && d104.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d102)
			var d105 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d104.Imm.Int() > 64)}
			} else {
				r68 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d104.Reg, 64)
				ctx.W.EmitSetcc(r68, scm.CcG)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
			}
			ctx.FreeDesc(&d104)
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d105.Loc == scm.LocImm {
				if d105.Imm.Bool() {
					ctx.W.EmitJmp(lbl28)
				} else {
					ctx.EmitStoreToStack(d93, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d105.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
				ctx.EmitStoreToStack(d93, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d105)
			ctx.W.MarkLabel(lbl28)
			var d106 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() / 64)}
			} else {
				r69 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r69, d88.Reg)
				ctx.W.EmitShrRegImm8(r69, 6)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			}
			if d106.Loc == scm.LocReg && d88.Loc == scm.LocReg && d106.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d106.Reg, scm.RegR11)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d106.Reg}
			}
			if d107.Loc == scm.LocReg && d106.Loc == scm.LocReg && d107.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			r70 := ctx.AllocReg()
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r70, uint64(d107.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r70, d107.Reg)
				ctx.W.EmitShlRegImm8(r70, 3)
			}
			if d89.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(r70, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r70, d89.Reg)
			}
			r71 := ctx.AllocRegExcept(r70)
			ctx.W.EmitMovRegMem(r71, r70, 0)
			ctx.FreeReg(r70)
			d108 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
			ctx.FreeDesc(&d107)
			var d109 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d88.Reg, 63)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d88.Reg}
			}
			if d109.Loc == scm.LocReg && d88.Loc == scm.LocReg && d109.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			d110 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d111 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() - d109.Imm.Int())}
			} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d110.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d109.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d109.Imm.Int()))
				ctx.W.EmitSubInt64(d110.Reg, scm.RegR11)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
			} else {
				ctx.W.EmitSubInt64(d110.Reg, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
			}
			if d111.Loc == scm.LocReg && d110.Loc == scm.LocReg && d111.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			var d112 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d108.Imm.Int()) >> uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d108.Reg, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d108.Reg}
			} else {
				{
					shiftSrc := d108.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d111.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d111.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d111.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d112.Loc == scm.LocReg && d108.Loc == scm.LocReg && d112.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d111)
			var d113 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() | d112.Imm.Int())}
			} else if d93.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d93.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitOrInt64(d93.Reg, scratch)
				ctx.FreeReg(scratch)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d93.Reg}
			} else {
				ctx.W.EmitOrInt64(d93.Reg, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d93.Reg}
			}
			if d113.Loc == scm.LocReg && d93.Loc == scm.LocReg && d113.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.EmitStoreToStack(d113, 88)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl24)
			ctx.W.ResolveFixups()
			d114 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r55}
			if d83.Loc == scm.LocReg { ctx.UnprotectReg(d83.Reg) }
			var d116 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r72, thisptr.Reg, off)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			}
			var d117 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d114.Imm.Int() + d116.Imm.Int())}
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d114.Reg}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d114.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d116.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
				ctx.W.EmitAddInt64(d114.Reg, scm.RegR11)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d114.Reg}
			} else {
				ctx.W.EmitAddInt64(d114.Reg, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d114.Reg}
			}
			if d117.Loc == scm.LocReg && d114.Loc == scm.LocReg && d117.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			ctx.FreeDesc(&d116)
			var d118 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r73, thisptr.Reg, off)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r73}
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d118.Loc == scm.LocImm {
				if d118.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d118.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d118)
			ctx.W.MarkLabel(lbl22)
			ctx.EmitStoreToStack(d45, 56)
			ctx.EmitStoreToStack(d47, 64)
			lbl33 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl21)
			var d119 scm.JITValueDesc
			if d45.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(scratch, d45.Reg)
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitSubInt64(scratch, scm.RegR11)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d119.Loc == scm.LocReg && d45.Loc == scm.LocReg && d119.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.EmitStoreToStack(d46, 56)
			ctx.EmitStoreToStack(d119, 64)
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl31)
			if d83.Loc == scm.LocReg { ctx.ProtectReg(d83.Reg) }
			r74 := ctx.AllocReg()
			lbl34 := ctx.W.ReserveLabel()
			var d121 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r75 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r75, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			}
			var d123 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() * d121.Imm.Int())}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d121.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d83.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r76 := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegReg(r76, d83.Reg)
				ctx.W.EmitImulInt64(r76, d121.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			}
			if d123.Loc == scm.LocReg && d83.Loc == scm.LocReg && d123.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d121)
			var d124 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				r77 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r77, thisptr.Reg, off)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			}
			var d125 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() / 64)}
			} else {
				r78 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r78, d123.Reg)
				ctx.W.EmitShrRegImm8(r78, 6)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			}
			if d125.Loc == scm.LocReg && d123.Loc == scm.LocReg && d125.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			r79 := ctx.AllocReg()
			if d125.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r79, uint64(d125.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r79, d125.Reg)
				ctx.W.EmitShlRegImm8(r79, 3)
			}
			if d124.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
				ctx.W.EmitAddInt64(r79, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r79, d124.Reg)
			}
			r80 := ctx.AllocRegExcept(r79)
			ctx.W.EmitMovRegMem(r80, r79, 0)
			ctx.FreeReg(r79)
			d126 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r80}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() % 64)}
			} else {
				r81 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r81, d123.Reg)
				ctx.W.EmitAndRegImm32(r81, 63)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			}
			if d127.Loc == scm.LocReg && d123.Loc == scm.LocReg && d127.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d126.Imm.Int()) << uint64(d127.Imm.Int())))}
			} else if d127.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d126.Reg, uint8(d127.Imm.Int()))
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
			} else {
				{
					shiftSrc := d126.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d127.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d127.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d127.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d127)
			var d129 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r82 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r82, thisptr.Reg, off)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			if d129.Loc == scm.LocImm {
				if d129.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.EmitStoreToStack(d128, 96)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d129.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl37)
				ctx.EmitStoreToStack(d128, 96)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl37)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d129)
			ctx.W.MarkLabel(lbl36)
			r83 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r83, 96)
			ctx.ProtectReg(r83)
			d130 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r83}
			ctx.UnprotectReg(r83)
			var d131 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r84 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r84, thisptr.Reg, off)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r84}
			}
			d133 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() - d131.Imm.Int())}
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d131.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
				ctx.W.EmitSubInt64(d133.Reg, scm.RegR11)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			} else {
				ctx.W.EmitSubInt64(d133.Reg, d131.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			}
			if d134.Loc == scm.LocReg && d133.Loc == scm.LocReg && d134.Reg == d133.Reg {
				ctx.TransferReg(d133.Reg)
				d133.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			var d135 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) >> uint64(d134.Imm.Int())))}
			} else if d134.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d130.Reg, uint8(d134.Imm.Int()))
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d130.Reg}
			} else {
				{
					shiftSrc := d130.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d134.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d134.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d134.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d135.Loc == scm.LocReg && d130.Loc == scm.LocReg && d135.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d130)
			ctx.FreeDesc(&d134)
			ctx.EmitMovToReg(r74, d135)
			ctx.W.EmitJmp(lbl34)
			ctx.FreeDesc(&d135)
			ctx.W.MarkLabel(lbl35)
			var d136 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() % 64)}
			} else {
				r85 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r85, d123.Reg)
				ctx.W.EmitAndRegImm32(r85, 63)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			}
			if d136.Loc == scm.LocReg && d123.Loc == scm.LocReg && d136.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			var d137 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r86, thisptr.Reg, off)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			}
			var d139 scm.JITValueDesc
			if d136.Loc == scm.LocImm && d137.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() + d137.Imm.Int())}
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
			} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d137.Reg}
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d137.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(d136.Reg, scm.RegR11)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
			} else {
				ctx.W.EmitAddInt64(d136.Reg, d137.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
			}
			if d139.Loc == scm.LocReg && d136.Loc == scm.LocReg && d139.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			ctx.FreeDesc(&d137)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d139.Imm.Int() > 64)}
			} else {
				r87 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d139.Reg, 64)
				ctx.W.EmitSetcc(r87, scm.CcG)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r87}
			}
			ctx.FreeDesc(&d139)
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d140.Loc == scm.LocImm {
				if d140.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
					ctx.EmitStoreToStack(d128, 96)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d140.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.EmitStoreToStack(d128, 96)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d140)
			ctx.W.MarkLabel(lbl38)
			var d141 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() / 64)}
			} else {
				r88 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r88, d123.Reg)
				ctx.W.EmitShrRegImm8(r88, 6)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
			}
			if d141.Loc == scm.LocReg && d123.Loc == scm.LocReg && d141.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d141.Reg, scm.RegR11)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d141.Reg}
			}
			if d142.Loc == scm.LocReg && d141.Loc == scm.LocReg && d142.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			r89 := ctx.AllocReg()
			if d142.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r89, uint64(d142.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r89, d142.Reg)
				ctx.W.EmitShlRegImm8(r89, 3)
			}
			if d124.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
				ctx.W.EmitAddInt64(r89, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r89, d124.Reg)
			}
			r90 := ctx.AllocRegExcept(r89)
			ctx.W.EmitMovRegMem(r90, r89, 0)
			ctx.FreeReg(r89)
			d143 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r90}
			ctx.FreeDesc(&d142)
			var d144 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d123.Reg, 63)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d123.Reg}
			}
			if d144.Loc == scm.LocReg && d123.Loc == scm.LocReg && d144.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d123)
			d145 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() - d144.Imm.Int())}
			} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
			} else if d145.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d144.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d144.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitSubInt64(d145.Reg, scm.RegR11)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
			} else {
				ctx.W.EmitSubInt64(d145.Reg, d144.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
			}
			if d146.Loc == scm.LocReg && d145.Loc == scm.LocReg && d146.Reg == d145.Reg {
				ctx.TransferReg(d145.Reg)
				d145.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			var d147 scm.JITValueDesc
			if d143.Loc == scm.LocImm && d146.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d143.Imm.Int()) >> uint64(d146.Imm.Int())))}
			} else if d146.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d143.Reg, uint8(d146.Imm.Int()))
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d143.Reg}
			} else {
				{
					shiftSrc := d143.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d146.Reg != scm.RegRCX
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
				}
			}
			if d147.Loc == scm.LocReg && d143.Loc == scm.LocReg && d147.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d143)
			ctx.FreeDesc(&d146)
			var d148 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() | d147.Imm.Int())}
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d147.Imm.Int()))
				ctx.W.EmitOrInt64(d128.Reg, scratch)
				ctx.FreeReg(scratch)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			} else {
				ctx.W.EmitOrInt64(d128.Reg, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			}
			if d148.Loc == scm.LocReg && d128.Loc == scm.LocReg && d148.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.EmitStoreToStack(d148, 96)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl34)
			ctx.W.ResolveFixups()
			d149 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			if d83.Loc == scm.LocReg { ctx.UnprotectReg(d83.Reg) }
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r91, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			}
			var d152 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() + d151.Imm.Int())}
			} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d149.Reg}
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d151.Reg}
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d151.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d151.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
				ctx.W.EmitAddInt64(d149.Reg, scm.RegR11)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d149.Reg}
			} else {
				ctx.W.EmitAddInt64(d149.Reg, d151.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d149.Reg}
			}
			if d152.Loc == scm.LocReg && d149.Loc == scm.LocReg && d152.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			ctx.FreeDesc(&d151)
			r92 := ctx.AllocReg()
			lbl40 := ctx.W.ReserveLabel()
			var d154 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
			}
			var d156 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() * d154.Imm.Int())}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d154.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d154.Imm.Int()))
				ctx.W.EmitImulInt64(d83.Reg, scm.RegR11)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else {
				ctx.W.EmitImulInt64(d83.Reg, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			}
			if d156.Loc == scm.LocReg && d83.Loc == scm.LocReg && d156.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d83)
			ctx.FreeDesc(&d154)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() / 64)}
			} else {
				r94 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r94, d156.Reg)
				ctx.W.EmitShrRegImm8(r94, 6)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			}
			if d157.Loc == scm.LocReg && d156.Loc == scm.LocReg && d157.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			r95 := ctx.AllocReg()
			if d157.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r95, uint64(d157.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r95, d157.Reg)
				ctx.W.EmitShlRegImm8(r95, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r95, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r95, d11.Reg)
			}
			r96 := ctx.AllocRegExcept(r95)
			ctx.W.EmitMovRegMem(r96, r95, 0)
			ctx.FreeReg(r95)
			d158 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
			ctx.FreeDesc(&d157)
			var d159 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() % 64)}
			} else {
				r97 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r97, d156.Reg)
				ctx.W.EmitAndRegImm32(r97, 63)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			}
			if d159.Loc == scm.LocReg && d156.Loc == scm.LocReg && d159.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			var d160 scm.JITValueDesc
			if d158.Loc == scm.LocImm && d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d158.Imm.Int()) << uint64(d159.Imm.Int())))}
			} else if d159.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d158.Reg, uint8(d159.Imm.Int()))
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
			} else {
				{
					shiftSrc := d158.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d159.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d159.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d159.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d160.Loc == scm.LocReg && d158.Loc == scm.LocReg && d160.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.FreeDesc(&d159)
			var d161 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r98, thisptr.Reg, off)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			}
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d161.Loc == scm.LocImm {
				if d161.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.EmitStoreToStack(d160, 104)
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d161.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.EmitStoreToStack(d160, 104)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d161)
			ctx.W.MarkLabel(lbl42)
			r99 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r99, 104)
			ctx.ProtectReg(r99)
			d162 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r99}
			ctx.UnprotectReg(r99)
			var d163 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r100, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
			}
			d165 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm && d163.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() - d163.Imm.Int())}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			} else if d165.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d165.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d163.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d163.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
				ctx.W.EmitSubInt64(d165.Reg, scm.RegR11)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			} else {
				ctx.W.EmitSubInt64(d165.Reg, d163.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			var d167 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d162.Imm.Int()) >> uint64(d166.Imm.Int())))}
			} else if d166.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d162.Reg, uint8(d166.Imm.Int()))
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			} else {
				{
					shiftSrc := d162.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d166.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d166.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d166.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d167.Loc == scm.LocReg && d162.Loc == scm.LocReg && d167.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d166)
			ctx.EmitMovToReg(r92, d167)
			ctx.W.EmitJmp(lbl40)
			ctx.FreeDesc(&d167)
			ctx.W.MarkLabel(lbl41)
			var d168 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() % 64)}
			} else {
				r101 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r101, d156.Reg)
				ctx.W.EmitAndRegImm32(r101, 63)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			}
			if d168.Loc == scm.LocReg && d156.Loc == scm.LocReg && d168.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			var d169 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r102, thisptr.Reg, off)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r102}
			}
			var d171 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() + d169.Imm.Int())}
			} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d168.Reg}
			} else if d168.Loc == scm.LocImm && d168.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d169.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d169.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
				ctx.W.EmitAddInt64(d168.Reg, scm.RegR11)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d168.Reg}
			} else {
				ctx.W.EmitAddInt64(d168.Reg, d169.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d168.Reg}
			}
			if d171.Loc == scm.LocReg && d168.Loc == scm.LocReg && d171.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			ctx.FreeDesc(&d169)
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d171.Imm.Int() > 64)}
			} else {
				r103 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d171.Reg, 64)
				ctx.W.EmitSetcc(r103, scm.CcG)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r103}
			}
			ctx.FreeDesc(&d171)
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			if d172.Loc == scm.LocImm {
				if d172.Imm.Bool() {
					ctx.W.EmitJmp(lbl44)
				} else {
					ctx.EmitStoreToStack(d160, 104)
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d172.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.EmitStoreToStack(d160, 104)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d172)
			ctx.W.MarkLabel(lbl44)
			var d173 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() / 64)}
			} else {
				r104 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r104, d156.Reg)
				ctx.W.EmitShrRegImm8(r104, 6)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			}
			if d173.Loc == scm.LocReg && d156.Loc == scm.LocReg && d173.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d173.Reg, scm.RegR11)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
			}
			if d174.Loc == scm.LocReg && d173.Loc == scm.LocReg && d174.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			r105 := ctx.AllocReg()
			if d174.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r105, uint64(d174.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r105, d174.Reg)
				ctx.W.EmitShlRegImm8(r105, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r105, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r105, d11.Reg)
			}
			r106 := ctx.AllocRegExcept(r105)
			ctx.W.EmitMovRegMem(r106, r105, 0)
			ctx.FreeReg(r105)
			d175 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
			ctx.FreeDesc(&d174)
			var d176 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d156.Reg, 63)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d156.Reg}
			}
			if d176.Loc == scm.LocReg && d156.Loc == scm.LocReg && d176.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			d177 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d176.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() - d176.Imm.Int())}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d176.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d176.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
				ctx.W.EmitSubInt64(d177.Reg, scm.RegR11)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else {
				ctx.W.EmitSubInt64(d177.Reg, d176.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			}
			if d178.Loc == scm.LocReg && d177.Loc == scm.LocReg && d178.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			var d179 scm.JITValueDesc
			if d175.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d175.Imm.Int()) >> uint64(d178.Imm.Int())))}
			} else if d178.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d175.Reg, uint8(d178.Imm.Int()))
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
			} else {
				{
					shiftSrc := d175.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d178.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d178.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d178.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d179.Loc == scm.LocReg && d175.Loc == scm.LocReg && d179.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			ctx.FreeDesc(&d178)
			var d180 scm.JITValueDesc
			if d160.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d160.Imm.Int() | d179.Imm.Int())}
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d160.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d179.Imm.Int()))
				ctx.W.EmitOrInt64(d160.Reg, scratch)
				ctx.FreeReg(scratch)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			} else {
				ctx.W.EmitOrInt64(d160.Reg, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			}
			if d180.Loc == scm.LocReg && d160.Loc == scm.LocReg && d180.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.EmitStoreToStack(d180, 104)
			ctx.W.EmitJmp(lbl42)
			ctx.W.MarkLabel(lbl40)
			ctx.W.ResolveFixups()
			d181 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			ctx.FreeDesc(&d83)
			var d183 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r107, thisptr.Reg, off)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
			}
			var d184 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() + d183.Imm.Int())}
			} else if d183.Loc == scm.LocImm && d183.Imm.Int() == 0 {
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d181.Reg}
			} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d183.Reg}
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d181.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d183.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d183.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d183.Imm.Int()))
				ctx.W.EmitAddInt64(d181.Reg, scm.RegR11)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d181.Reg}
			} else {
				ctx.W.EmitAddInt64(d181.Reg, d183.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d181.Reg}
			}
			if d184.Loc == scm.LocReg && d181.Loc == scm.LocReg && d184.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d183)
			var d186 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() - d184.Imm.Int())}
			} else if d184.Loc == scm.LocImm && d184.Imm.Int() == 0 {
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d184.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d184.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitSubInt64(idxInt.Reg, scm.RegR11)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			} else {
				ctx.W.EmitSubInt64(idxInt.Reg, d184.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			}
			if d186.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d186.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&idxInt)
			ctx.FreeDesc(&d184)
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() * d152.Imm.Int())}
			} else if d186.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d186.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d152.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d152.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
				ctx.W.EmitImulInt64(d186.Reg, scm.RegR11)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d186.Reg}
			} else {
				ctx.W.EmitImulInt64(d186.Reg, d152.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d186.Reg}
			}
			if d187.Loc == scm.LocReg && d186.Loc == scm.LocReg && d187.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			ctx.FreeDesc(&d152)
			var d188 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() + d187.Imm.Int())}
			} else if d187.Loc == scm.LocImm && d187.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r108, d117.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			} else if d117.Loc == scm.LocImm && d117.Imm.Int() == 0 {
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d187.Reg}
			} else if d117.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d117.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d187.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d117.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r109 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r109, d117.Reg)
				ctx.W.EmitAddInt64(r109, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			}
			if d188.Loc == scm.LocReg && d117.Loc == scm.LocReg && d188.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d187)
			var d189 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d188.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(d188.Reg, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d188.Reg}
			}
			ctx.FreeDesc(&d188)
			ctx.W.EmitMakeFloat(result, d189)
			if d189.Loc == scm.LocReg { ctx.FreeReg(d189.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl30)
			var d190 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r110, thisptr.Reg, off)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
			}
			var d192 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d190.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d117.Imm.Int() == d190.Imm.Int())}
			} else if d190.Loc == scm.LocImm {
				r111 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d117.Reg, int32(d190.Imm.Int()))
				ctx.W.EmitSetcc(r111, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r111}
			} else if d117.Loc == scm.LocImm {
				r112 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d117.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d190.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r112, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r112}
			} else {
				r113 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d117.Reg, d190.Reg)
				ctx.W.EmitSetcc(r113, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r113}
			}
			ctx.FreeDesc(&d117)
			ctx.FreeDesc(&d190)
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d192.Loc == scm.LocImm {
				if d192.Imm.Bool() {
					ctx.W.EmitJmp(lbl46)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d192.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d192)
			ctx.W.MarkLabel(lbl33)
			r114 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r114, 56)
			ctx.ProtectReg(r114)
			d193 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r114}
			r115 := ctx.AllocRegExcept(r114)
			ctx.EmitLoadFromStack(r115, 64)
			ctx.ProtectReg(r115)
			d194 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r115}
			ctx.UnprotectReg(r114)
			ctx.UnprotectReg(r115)
			var d195 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d193.Imm.Int() == d194.Imm.Int())}
			} else if d194.Loc == scm.LocImm {
				r116 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitCmpRegImm32(d193.Reg, int32(d194.Imm.Int()))
				ctx.W.EmitSetcc(r116, scm.CcE)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
			} else if d193.Loc == scm.LocImm {
				r117 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d194.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r117, scm.CcE)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r117}
			} else {
				r118 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitCmpInt64(d193.Reg, d194.Reg)
				ctx.W.EmitSetcc(r118, scm.CcE)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
			}
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d195.Loc == scm.LocImm {
				if d195.Imm.Bool() {
					ctx.EmitStoreToStack(d193, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl48)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d195.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl49)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl49)
				ctx.EmitStoreToStack(d193, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d195)
			ctx.W.MarkLabel(lbl46)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl48)
			var d196 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d194.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d193.Imm.Int() + d194.Imm.Int())}
			} else if d194.Loc == scm.LocImm && d194.Imm.Int() == 0 {
				r119 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r119, d193.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d194.Reg}
			} else if d193.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d194.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d194.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d193.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r120 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r120, d193.Reg)
				ctx.W.EmitAddInt64(r120, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			}
			if d196.Loc == scm.LocReg && d193.Loc == scm.LocReg && d196.Reg == d193.Reg {
				ctx.TransferReg(d193.Reg)
				d193.Loc = scm.LocNone
			}
			var d197 scm.JITValueDesc
			if d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d196.Imm.Int() / 2)}
			} else {
				ctx.W.EmitShrRegImm8(d196.Reg, 1)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d196.Reg}
			}
			if d197.Loc == scm.LocReg && d196.Loc == scm.LocReg && d197.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.EmitStoreToStack(d197, 0)
			ctx.EmitStoreToStack(d193, 8)
			ctx.EmitStoreToStack(d194, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
