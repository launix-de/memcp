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
			if idxInt.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
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
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d0.Imm.Int()))))}
			} else {
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r2, d0.Reg)
				ctx.W.EmitShlRegImm8(r2, 32)
				ctx.W.EmitShrRegImm8(r2, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r2}
			}
			ctx.FreeDesc(&d0)
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMem32(r3, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMemL(r4, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			}
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
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
			d4 := d1
			if d4.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: d4.Type, Imm: scm.NewInt(int64(uint64(d4.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d4.Reg, 32)
				ctx.W.EmitShrRegImm8(d4.Reg, 32)
			}
			ctx.EmitStoreToStack(d4, 0)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 8)
			d5 := d3
			if d5.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: d5.Type, Imm: scm.NewInt(int64(uint64(d5.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d5.Reg, 32)
				ctx.W.EmitShrRegImm8(d5.Reg, 32)
			}
			ctx.EmitStoreToStack(d5, 16)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r5 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r5, 0)
			ctx.ProtectReg(r5)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r5}
			r6 := ctx.AllocRegExcept(r5)
			ctx.EmitLoadFromStack(r6, 8)
			ctx.ProtectReg(r6)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r6}
			r7 := ctx.AllocRegExcept(r5, r6)
			ctx.EmitLoadFromStack(r7, 16)
			ctx.ProtectReg(r7)
			d8 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r7}
			ctx.UnprotectReg(r5)
			ctx.UnprotectReg(r6)
			ctx.UnprotectReg(r7)
			r8 := d6.Loc == scm.LocReg
			r9 := d6.Reg
			if r8 { ctx.ProtectReg(r9) }
			r10 := ctx.AllocReg()
			lbl2 := ctx.W.ReserveLabel()
			var d9 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d6.Imm.Int()))))}
			} else {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r11, d6.Reg)
				ctx.W.EmitShlRegImm8(r11, 32)
				ctx.W.EmitShrRegImm8(r11, 32)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			}
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r12, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			}
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d10.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d10.Reg)
				ctx.W.EmitShlRegImm8(r13, 56)
				ctx.W.EmitShrRegImm8(r13, 56)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			}
			ctx.FreeDesc(&d10)
			var d12 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() * d11.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d11.Loc == scm.LocImm {
				if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d9.Reg, int32(d11.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitImulInt64(d9.Reg, scm.RegR11)
				}
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			} else {
				ctx.W.EmitImulInt64(d9.Reg, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			}
			if d12.Loc == scm.LocReg && d9.Loc == scm.LocReg && d12.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d11)
			var d13 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r14, thisptr.Reg, off)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			}
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r15 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r15, d12.Reg)
				ctx.W.EmitShrRegImm8(r15, 6)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			r16 := ctx.AllocReg()
			if d14.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r16, uint64(d14.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r16, d14.Reg)
				ctx.W.EmitShlRegImm8(r16, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r16, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r16, d13.Reg)
			}
			r17 := ctx.AllocRegExcept(r16)
			ctx.W.EmitMovRegMem(r17, r16, 0)
			ctx.FreeReg(r16)
			d15 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
			ctx.FreeDesc(&d14)
			var d16 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r18 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r18, d12.Reg)
				ctx.W.EmitAndRegImm32(r18, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			}
			if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			var d17 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) << uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d15.Reg, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			} else {
				{
					shiftSrc := d15.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d16.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d16.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d16.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d17.Loc == scm.LocReg && d15.Loc == scm.LocReg && d17.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.FreeDesc(&d16)
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d18.Loc == scm.LocImm {
				if d18.Imm.Bool() {
					ctx.W.EmitJmp(lbl3)
				} else {
			ctx.EmitStoreToStack(d17, 72)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl5)
			ctx.EmitStoreToStack(d17, 72)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
			ctx.FreeDesc(&d18)
			ctx.W.MarkLabel(lbl4)
			r20 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r20, 72)
			ctx.ProtectReg(r20)
			d19 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r20}
			ctx.UnprotectReg(r20)
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r21, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			}
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d20.Imm.Int()))))}
			} else {
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r22, d20.Reg)
				ctx.W.EmitShlRegImm8(r22, 56)
				ctx.W.EmitShrRegImm8(r22, 56)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			}
			ctx.FreeDesc(&d20)
			d22 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() - d21.Imm.Int())}
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d21.Loc == scm.LocImm {
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d22.Reg, int32(d21.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
				ctx.W.EmitSubInt64(d22.Reg, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			} else {
				ctx.W.EmitSubInt64(d22.Reg, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			var d24 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d19.Imm.Int()) >> uint64(d23.Imm.Int())))}
			} else if d23.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d19.Reg, uint8(d23.Imm.Int()))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d19.Reg}
			} else {
				{
					shiftSrc := d19.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d23.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d23.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d23.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d24.Loc == scm.LocReg && d19.Loc == scm.LocReg && d24.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.FreeDesc(&d23)
			ctx.EmitMovToReg(r10, d24)
			ctx.W.EmitJmp(lbl2)
			ctx.FreeDesc(&d24)
			ctx.W.MarkLabel(lbl3)
			var d25 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r23 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r23, d12.Reg)
				ctx.W.EmitAndRegImm32(r23, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			}
			if d25.Loc == scm.LocReg && d12.Loc == scm.LocReg && d25.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			var d26 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			}
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d26.Reg)
				ctx.W.EmitShlRegImm8(r25, 56)
				ctx.W.EmitShrRegImm8(r25, 56)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			}
			ctx.FreeDesc(&d26)
			var d28 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() + d27.Imm.Int())}
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d27.Loc == scm.LocImm {
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d25.Reg, int32(d27.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(d25.Reg, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else {
				ctx.W.EmitAddInt64(d25.Reg, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			}
			if d28.Loc == scm.LocReg && d25.Loc == scm.LocReg && d28.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.FreeDesc(&d27)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d28.Imm.Int()) > uint64(64))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d28.Reg, 64)
				ctx.W.EmitSetcc(r26, scm.CcA)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r26}
			}
			ctx.FreeDesc(&d28)
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d29.Loc == scm.LocImm {
				if d29.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			ctx.EmitStoreToStack(d17, 72)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d29.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
			ctx.EmitStoreToStack(d17, 72)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d29)
			ctx.W.MarkLabel(lbl6)
			var d30 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r27 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r27, d12.Reg)
				ctx.W.EmitShrRegImm8(r27, 6)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			}
			if d30.Loc == scm.LocReg && d12.Loc == scm.LocReg && d30.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d30.Reg, int32(1))
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			r28 := ctx.AllocReg()
			if d31.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r28, uint64(d31.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r28, d31.Reg)
				ctx.W.EmitShlRegImm8(r28, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r28, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r28, d13.Reg)
			}
			r29 := ctx.AllocRegExcept(r28)
			ctx.W.EmitMovRegMem(r29, r28, 0)
			ctx.FreeReg(r28)
			d32 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.FreeDesc(&d31)
			var d33 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d12.Reg, 63)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d12.Reg}
			}
			if d33.Loc == scm.LocReg && d12.Loc == scm.LocReg && d33.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			d34 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d35 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() - d33.Imm.Int())}
			} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d33.Loc == scm.LocImm {
				if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d34.Reg, int32(d33.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
				ctx.W.EmitSubInt64(d34.Reg, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
			} else {
				ctx.W.EmitSubInt64(d34.Reg, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
			}
			if d35.Loc == scm.LocReg && d34.Loc == scm.LocReg && d35.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			var d36 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) >> uint64(d35.Imm.Int())))}
			} else if d35.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d32.Reg, uint8(d35.Imm.Int()))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else {
				{
					shiftSrc := d32.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d35.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d35.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d35.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d36.Loc == scm.LocReg && d32.Loc == scm.LocReg && d36.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.FreeDesc(&d35)
			var d37 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() | d36.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r30, d17.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d36.Loc == scm.LocImm {
				r31 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r31, d17.Reg)
				if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r31, int32(d36.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
					ctx.W.EmitOrInt64(r31, scratch)
					ctx.FreeReg(scratch)
				}
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
			} else {
				r32 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r32, d17.Reg)
				ctx.W.EmitOrInt64(r32, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
			}
			if d37.Loc == scm.LocReg && d17.Loc == scm.LocReg && d37.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.EmitStoreToStack(d37, 72)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			d38 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			if r8 { ctx.UnprotectReg(r9) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d38.Imm.Int()))))}
			} else {
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r33, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r34, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			}
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() + d40.Imm.Int())}
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d40.Loc == scm.LocImm {
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d39.Reg, int32(d40.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(d39.Reg, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			} else {
				ctx.W.EmitAddInt64(d39.Reg, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d41.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r35, d41.Reg)
				ctx.W.EmitShlRegImm8(r35, 32)
				ctx.W.EmitShrRegImm8(r35, 32)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			}
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d42.Imm.Int()))}
			} else if d42.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(idxInt.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d42.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d42.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r36, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
			} else if idxInt.Loc == scm.LocImm {
				r37 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d42.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r37, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
			} else {
				r38 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d42.Reg)
				ctx.W.EmitSetcc(r38, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
			}
			ctx.FreeDesc(&d42)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d43)
			ctx.W.MarkLabel(lbl9)
			var d44 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d44.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: d44.Type, Imm: scm.NewInt(int64(uint64(d44.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d44.Reg, 32)
				ctx.W.EmitShrRegImm8(d44.Reg, 32)
			}
			if d44.Loc == scm.LocReg && d6.Loc == scm.LocReg && d44.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			d45 := d44
			if d45.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: d45.Type, Imm: scm.NewInt(int64(uint64(d45.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d45.Reg, 32)
				ctx.W.EmitShrRegImm8(d45.Reg, 32)
			}
			ctx.EmitStoreToStack(d45, 32)
			d46 := d6
			if d46.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: d46.Type, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d46.Reg, 32)
				ctx.W.EmitShrRegImm8(d46.Reg, 32)
			}
			ctx.EmitStoreToStack(d46, 40)
			d47 := d8
			if d47.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: d47.Type, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d47.Reg, 32)
				ctx.W.EmitShrRegImm8(d47.Reg, 32)
			}
			ctx.EmitStoreToStack(d47, 48)
			lbl11 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			var d48 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d48.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: d48.Type, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d48.Reg, 32)
				ctx.W.EmitShrRegImm8(d48.Reg, 32)
			}
			if d48.Loc == scm.LocReg && d6.Loc == scm.LocReg && d48.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			var d49 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d49.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: d49.Type, Imm: scm.NewInt(int64(uint64(d49.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d49.Reg, 32)
				ctx.W.EmitShrRegImm8(d49.Reg, 32)
			}
			if d49.Loc == scm.LocReg && d6.Loc == scm.LocReg && d49.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			d50 := d49
			if d50.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: d50.Type, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d50.Reg, 32)
				ctx.W.EmitShrRegImm8(d50.Reg, 32)
			}
			ctx.EmitStoreToStack(d50, 32)
			d51 := d7
			if d51.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: d51.Type, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d51.Reg, 32)
				ctx.W.EmitShrRegImm8(d51.Reg, 32)
			}
			ctx.EmitStoreToStack(d51, 40)
			d52 := d48
			if d52.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: d52.Type, Imm: scm.NewInt(int64(uint64(d52.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d52.Reg, 32)
				ctx.W.EmitShrRegImm8(d52.Reg, 32)
			}
			ctx.EmitStoreToStack(d52, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl11)
			r39 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r39, 32)
			ctx.ProtectReg(r39)
			d53 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r39}
			r40 := ctx.AllocRegExcept(r39)
			ctx.EmitLoadFromStack(r40, 40)
			ctx.ProtectReg(r40)
			d54 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r40}
			r41 := ctx.AllocRegExcept(r39, r40)
			ctx.EmitLoadFromStack(r41, 48)
			ctx.ProtectReg(r41)
			d55 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r41}
			ctx.UnprotectReg(r39)
			ctx.UnprotectReg(r40)
			ctx.UnprotectReg(r41)
			var d56 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d54.Imm.Int()) == uint64(d55.Imm.Int()))}
			} else if d55.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d54.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d54.Reg, int32(d55.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
					ctx.W.EmitCmpInt64(d54.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r42, scm.CcE)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r42}
			} else if d54.Loc == scm.LocImm {
				r43 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d55.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r43, scm.CcE)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r43}
			} else {
				r44 := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitCmpInt64(d54.Reg, d55.Reg)
				ctx.W.EmitSetcc(r44, scm.CcE)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r44}
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d56.Loc == scm.LocImm {
				if d56.Imm.Bool() {
			d57 := d54
			if d57.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: d57.Type, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d57.Reg, 32)
				ctx.W.EmitShrRegImm8(d57.Reg, 32)
			}
			ctx.EmitStoreToStack(d57, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d56.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
			d58 := d54
			if d58.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: d58.Type, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d58.Reg, 32)
				ctx.W.EmitShrRegImm8(d58.Reg, 32)
			}
			ctx.EmitStoreToStack(d58, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d56)
			ctx.W.MarkLabel(lbl13)
			r45 := d53.Loc == scm.LocReg
			r46 := d53.Reg
			if r45 { ctx.ProtectReg(r46) }
			r47 := ctx.AllocReg()
			lbl15 := ctx.W.ReserveLabel()
			var d59 scm.JITValueDesc
			if d53.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d53.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d53.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			}
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
			}
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d60.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d60.Reg)
				ctx.W.EmitShlRegImm8(r50, 56)
				ctx.W.EmitShrRegImm8(r50, 56)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			}
			ctx.FreeDesc(&d60)
			var d62 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() * d61.Imm.Int())}
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d61.Loc == scm.LocImm {
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d59.Reg, int32(d61.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
				ctx.W.EmitImulInt64(d59.Reg, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
			} else {
				ctx.W.EmitImulInt64(d59.Reg, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
			}
			if d62.Loc == scm.LocReg && d59.Loc == scm.LocReg && d62.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d61)
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r51 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r51, d62.Reg)
				ctx.W.EmitShrRegImm8(r51, 6)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			}
			if d63.Loc == scm.LocReg && d62.Loc == scm.LocReg && d63.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			r52 := ctx.AllocReg()
			if d63.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r52, uint64(d63.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r52, d63.Reg)
				ctx.W.EmitShlRegImm8(r52, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r52, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r52, d13.Reg)
			}
			r53 := ctx.AllocRegExcept(r52)
			ctx.W.EmitMovRegMem(r53, r52, 0)
			ctx.FreeReg(r52)
			d64 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
			ctx.FreeDesc(&d63)
			var d65 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r54 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r54, d62.Reg)
				ctx.W.EmitAndRegImm32(r54, 63)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			}
			if d65.Loc == scm.LocReg && d62.Loc == scm.LocReg && d65.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			var d66 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d64.Imm.Int()) << uint64(d65.Imm.Int())))}
			} else if d65.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d64.Reg, uint8(d65.Imm.Int()))
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d64.Reg}
			} else {
				{
					shiftSrc := d64.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d65.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d65.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d65.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d66.Loc == scm.LocReg && d64.Loc == scm.LocReg && d66.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d64)
			ctx.FreeDesc(&d65)
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r55, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r55}
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d67.Loc == scm.LocImm {
				if d67.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
			ctx.EmitStoreToStack(d66, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d67.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
			ctx.EmitStoreToStack(d66, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d67)
			ctx.W.MarkLabel(lbl17)
			r56 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r56, 80)
			ctx.ProtectReg(r56)
			d68 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r56}
			ctx.UnprotectReg(r56)
			var d69 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r57 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r57, thisptr.Reg, off)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			}
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d69.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r58, d69.Reg)
				ctx.W.EmitShlRegImm8(r58, 56)
				ctx.W.EmitShrRegImm8(r58, 56)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			}
			ctx.FreeDesc(&d69)
			d71 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d72 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d70.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d71.Imm.Int() - d70.Imm.Int())}
			} else if d70.Loc == scm.LocImm && d70.Imm.Int() == 0 {
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			} else if d71.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d71.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d70.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d70.Loc == scm.LocImm {
				if d70.Imm.Int() >= -2147483648 && d70.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d71.Reg, int32(d70.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d70.Imm.Int()))
				ctx.W.EmitSubInt64(d71.Reg, scm.RegR11)
				}
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			} else {
				ctx.W.EmitSubInt64(d71.Reg, d70.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			}
			if d72.Loc == scm.LocReg && d71.Loc == scm.LocReg && d72.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d70)
			var d73 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) >> uint64(d72.Imm.Int())))}
			} else if d72.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d68.Reg, uint8(d72.Imm.Int()))
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d68.Reg}
			} else {
				{
					shiftSrc := d68.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d72.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d72.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d72.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d73.Loc == scm.LocReg && d68.Loc == scm.LocReg && d73.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d72)
			ctx.EmitMovToReg(r47, d73)
			ctx.W.EmitJmp(lbl15)
			ctx.FreeDesc(&d73)
			ctx.W.MarkLabel(lbl16)
			var d74 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r59 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r59, d62.Reg)
				ctx.W.EmitAndRegImm32(r59, 63)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			}
			if d74.Loc == scm.LocReg && d62.Loc == scm.LocReg && d74.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			var d75 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r60, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
			}
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d75.Imm.Int()))))}
			} else {
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r61, d75.Reg)
				ctx.W.EmitShlRegImm8(r61, 56)
				ctx.W.EmitShrRegImm8(r61, 56)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
			}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if d74.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() + d76.Imm.Int())}
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d76.Loc == scm.LocImm {
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d74.Reg, int32(d76.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
				ctx.W.EmitAddInt64(d74.Reg, scm.RegR11)
				}
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			} else {
				ctx.W.EmitAddInt64(d74.Reg, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			}
			if d77.Loc == scm.LocReg && d74.Loc == scm.LocReg && d77.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			ctx.FreeDesc(&d76)
			var d78 scm.JITValueDesc
			if d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d77.Imm.Int()) > uint64(64))}
			} else {
				r62 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d77.Reg, 64)
				ctx.W.EmitSetcc(r62, scm.CcA)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
			}
			ctx.FreeDesc(&d77)
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d78.Loc == scm.LocImm {
				if d78.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
			ctx.EmitStoreToStack(d66, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d78.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl20)
			ctx.EmitStoreToStack(d66, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d78)
			ctx.W.MarkLabel(lbl19)
			var d79 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r63 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r63, d62.Reg)
				ctx.W.EmitShrRegImm8(r63, 6)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			}
			if d79.Loc == scm.LocReg && d62.Loc == scm.LocReg && d79.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			var d80 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d79.Reg, int32(1))
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg}
			}
			if d80.Loc == scm.LocReg && d79.Loc == scm.LocReg && d80.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d79)
			r64 := ctx.AllocReg()
			if d80.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r64, uint64(d80.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r64, d80.Reg)
				ctx.W.EmitShlRegImm8(r64, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r64, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r64, d13.Reg)
			}
			r65 := ctx.AllocRegExcept(r64)
			ctx.W.EmitMovRegMem(r65, r64, 0)
			ctx.FreeReg(r64)
			d81 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r65}
			ctx.FreeDesc(&d80)
			var d82 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d62.Reg, 63)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d62.Reg}
			}
			if d82.Loc == scm.LocReg && d62.Loc == scm.LocReg && d82.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			d83 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d82.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() - d82.Imm.Int())}
			} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d82.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d82.Loc == scm.LocImm {
				if d82.Imm.Int() >= -2147483648 && d82.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d83.Reg, int32(d82.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d82.Imm.Int()))
				ctx.W.EmitSubInt64(d83.Reg, scm.RegR11)
				}
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else {
				ctx.W.EmitSubInt64(d83.Reg, d82.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			}
			if d84.Loc == scm.LocReg && d83.Loc == scm.LocReg && d84.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d82)
			var d85 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d81.Imm.Int()) >> uint64(d84.Imm.Int())))}
			} else if d84.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d81.Reg, uint8(d84.Imm.Int()))
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
			} else {
				{
					shiftSrc := d81.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d84.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d84.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d84.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d85.Loc == scm.LocReg && d81.Loc == scm.LocReg && d85.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d84)
			var d86 scm.JITValueDesc
			if d66.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() | d85.Imm.Int())}
			} else if d66.Loc == scm.LocImm && d66.Imm.Int() == 0 {
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d85.Reg}
			} else if d85.Loc == scm.LocImm && d85.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r66, d66.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			} else if d66.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d66.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d85.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d85.Loc == scm.LocImm {
				r67 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r67, d66.Reg)
				if d85.Imm.Int() >= -2147483648 && d85.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r67, int32(d85.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
					ctx.W.EmitOrInt64(r67, scratch)
					ctx.FreeReg(scratch)
				}
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			} else {
				r68 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r68, d66.Reg)
				ctx.W.EmitOrInt64(r68, d85.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
			}
			if d86.Loc == scm.LocReg && d66.Loc == scm.LocReg && d86.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.EmitStoreToStack(d86, 80)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl15)
			ctx.W.ResolveFixups()
			d87 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			if r45 { ctx.UnprotectReg(r46) }
			var d88 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d87.Imm.Int()))))}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r69, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			}
			ctx.FreeDesc(&d87)
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r70, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			}
			var d90 scm.JITValueDesc
			if d88.Loc == scm.LocImm && d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() + d89.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d88.Reg}
			} else if d88.Loc == scm.LocImm && d88.Imm.Int() == 0 {
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d89.Reg}
			} else if d88.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d88.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d89.Loc == scm.LocImm {
				if d89.Imm.Int() >= -2147483648 && d89.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d88.Reg, int32(d89.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(d88.Reg, scm.RegR11)
				}
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d88.Reg}
			} else {
				ctx.W.EmitAddInt64(d88.Reg, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d88.Reg}
			}
			if d90.Loc == scm.LocReg && d88.Loc == scm.LocReg && d90.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			ctx.FreeDesc(&d89)
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d90.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d90.Reg)
				ctx.W.EmitShlRegImm8(r71, 32)
				ctx.W.EmitShrRegImm8(r71, 32)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
			}
			ctx.FreeDesc(&d90)
			var d92 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d91.Imm.Int()))}
			} else if d91.Loc == scm.LocImm {
				r72 := ctx.AllocRegExcept(idxInt.Reg)
				if d91.Imm.Int() >= -2147483648 && d91.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d91.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d91.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r72, scm.CcB)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r72}
			} else if idxInt.Loc == scm.LocImm {
				r73 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d91.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r73, scm.CcB)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r73}
			} else {
				r74 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d91.Reg)
				ctx.W.EmitSetcc(r74, scm.CcB)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
			}
			ctx.FreeDesc(&d91)
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d92.Loc == scm.LocImm {
				if d92.Imm.Bool() {
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d92.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl22)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.FreeDesc(&d92)
			ctx.W.MarkLabel(lbl12)
			r75 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r75, 24)
			ctx.ProtectReg(r75)
			d93 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r75}
			ctx.UnprotectReg(r75)
			var d94 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d93.Imm.Int()))))}
			} else {
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r76, d93.Reg)
				ctx.W.EmitShlRegImm8(r76, 32)
				ctx.W.EmitShrRegImm8(r76, 32)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			}
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0))
				if d94.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d94.Imm.Int()))
					ctx.W.EmitStoreRegMem(scratch, baseReg, 0)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitStoreRegMem(d94.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue) + 0)
				if d94.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d94.Imm.Int()))
					ctx.W.EmitStoreRegMem(scratch, thisptr.Reg, off)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitStoreRegMem(d94.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d94)
			ctx.FreeDesc(&d94)
			r77 := d93.Loc == scm.LocReg
			r78 := d93.Reg
			if r77 { ctx.ProtectReg(r78) }
			r79 := ctx.AllocReg()
			lbl24 := ctx.W.ReserveLabel()
			var d95 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d93.Imm.Int()))))}
			} else {
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r80, d93.Reg)
				ctx.W.EmitShlRegImm8(r80, 32)
				ctx.W.EmitShrRegImm8(r80, 32)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			}
			var d96 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r81 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r81, thisptr.Reg, off)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r81}
			}
			var d97 scm.JITValueDesc
			if d96.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d96.Imm.Int()))))}
			} else {
				r82 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r82, d96.Reg)
				ctx.W.EmitShlRegImm8(r82, 56)
				ctx.W.EmitShrRegImm8(r82, 56)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			}
			ctx.FreeDesc(&d96)
			var d98 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() * d97.Imm.Int())}
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d95.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d97.Loc == scm.LocImm {
				if d97.Imm.Int() >= -2147483648 && d97.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d95.Reg, int32(d97.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d97.Imm.Int()))
				ctx.W.EmitImulInt64(d95.Reg, scm.RegR11)
				}
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else {
				ctx.W.EmitImulInt64(d95.Reg, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			}
			if d98.Loc == scm.LocReg && d95.Loc == scm.LocReg && d98.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			ctx.FreeDesc(&d97)
			var d99 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				r83 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r83, thisptr.Reg, off)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r83}
			}
			var d100 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() / 64)}
			} else {
				r84 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r84, d98.Reg)
				ctx.W.EmitShrRegImm8(r84, 6)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			}
			if d100.Loc == scm.LocReg && d98.Loc == scm.LocReg && d100.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			r85 := ctx.AllocReg()
			if d100.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r85, uint64(d100.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r85, d100.Reg)
				ctx.W.EmitShlRegImm8(r85, 3)
			}
			if d99.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(r85, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r85, d99.Reg)
			}
			r86 := ctx.AllocRegExcept(r85)
			ctx.W.EmitMovRegMem(r86, r85, 0)
			ctx.FreeReg(r85)
			d101 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			ctx.FreeDesc(&d100)
			var d102 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() % 64)}
			} else {
				r87 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r87, d98.Reg)
				ctx.W.EmitAndRegImm32(r87, 63)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
			}
			if d102.Loc == scm.LocReg && d98.Loc == scm.LocReg && d102.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			var d103 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d101.Imm.Int()) << uint64(d102.Imm.Int())))}
			} else if d102.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d101.Reg, uint8(d102.Imm.Int()))
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else {
				{
					shiftSrc := d101.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d102.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d102.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d102.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d103.Loc == scm.LocReg && d101.Loc == scm.LocReg && d103.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d102)
			var d104 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r88, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d104.Loc == scm.LocImm {
				if d104.Imm.Bool() {
					ctx.W.EmitJmp(lbl25)
				} else {
			ctx.EmitStoreToStack(d103, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d104.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
			ctx.EmitStoreToStack(d103, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl25)
			}
			ctx.FreeDesc(&d104)
			ctx.W.MarkLabel(lbl26)
			r89 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r89, 88)
			ctx.ProtectReg(r89)
			d105 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r89}
			ctx.UnprotectReg(r89)
			var d106 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r90, thisptr.Reg, off)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r90}
			}
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d106.Imm.Int()))))}
			} else {
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r91, d106.Reg)
				ctx.W.EmitShlRegImm8(r91, 56)
				ctx.W.EmitShrRegImm8(r91, 56)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			}
			ctx.FreeDesc(&d106)
			d108 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d109 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() - d107.Imm.Int())}
			} else if d107.Loc == scm.LocImm && d107.Imm.Int() == 0 {
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d108.Reg}
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d107.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d107.Loc == scm.LocImm {
				if d107.Imm.Int() >= -2147483648 && d107.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d108.Reg, int32(d107.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitSubInt64(d108.Reg, scm.RegR11)
				}
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d108.Reg}
			} else {
				ctx.W.EmitSubInt64(d108.Reg, d107.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d108.Reg}
			}
			if d109.Loc == scm.LocReg && d108.Loc == scm.LocReg && d109.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			var d110 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d105.Imm.Int()) >> uint64(d109.Imm.Int())))}
			} else if d109.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d105.Reg, uint8(d109.Imm.Int()))
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d105.Reg}
			} else {
				{
					shiftSrc := d105.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d109.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d109.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d109.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d110.Loc == scm.LocReg && d105.Loc == scm.LocReg && d110.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d105)
			ctx.FreeDesc(&d109)
			ctx.EmitMovToReg(r79, d110)
			ctx.W.EmitJmp(lbl24)
			ctx.FreeDesc(&d110)
			ctx.W.MarkLabel(lbl25)
			var d111 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() % 64)}
			} else {
				r92 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r92, d98.Reg)
				ctx.W.EmitAndRegImm32(r92, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			}
			if d111.Loc == scm.LocReg && d98.Loc == scm.LocReg && d111.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			var d112 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
			}
			var d113 scm.JITValueDesc
			if d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d112.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d112.Reg)
				ctx.W.EmitShlRegImm8(r94, 56)
				ctx.W.EmitShrRegImm8(r94, 56)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			}
			ctx.FreeDesc(&d112)
			var d114 scm.JITValueDesc
			if d111.Loc == scm.LocImm && d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d111.Imm.Int() + d113.Imm.Int())}
			} else if d113.Loc == scm.LocImm && d113.Imm.Int() == 0 {
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d111.Reg}
			} else if d111.Loc == scm.LocImm && d111.Imm.Int() == 0 {
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d113.Reg}
			} else if d111.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d111.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d113.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d113.Loc == scm.LocImm {
				if d113.Imm.Int() >= -2147483648 && d113.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d111.Reg, int32(d113.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d113.Imm.Int()))
				ctx.W.EmitAddInt64(d111.Reg, scm.RegR11)
				}
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d111.Reg}
			} else {
				ctx.W.EmitAddInt64(d111.Reg, d113.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d111.Reg}
			}
			if d114.Loc == scm.LocReg && d111.Loc == scm.LocReg && d114.Reg == d111.Reg {
				ctx.TransferReg(d111.Reg)
				d111.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d111)
			ctx.FreeDesc(&d113)
			var d115 scm.JITValueDesc
			if d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d114.Imm.Int()) > uint64(64))}
			} else {
				r95 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d114.Reg, 64)
				ctx.W.EmitSetcc(r95, scm.CcA)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r95}
			}
			ctx.FreeDesc(&d114)
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d115.Loc == scm.LocImm {
				if d115.Imm.Bool() {
					ctx.W.EmitJmp(lbl28)
				} else {
			ctx.EmitStoreToStack(d103, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d115.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
			ctx.EmitStoreToStack(d103, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d115)
			ctx.W.MarkLabel(lbl28)
			var d116 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() / 64)}
			} else {
				r96 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r96, d98.Reg)
				ctx.W.EmitShrRegImm8(r96, 6)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			}
			if d116.Loc == scm.LocReg && d98.Loc == scm.LocReg && d116.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d116.Reg, int32(1))
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			r97 := ctx.AllocReg()
			if d117.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r97, uint64(d117.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r97, d117.Reg)
				ctx.W.EmitShlRegImm8(r97, 3)
			}
			if d99.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(r97, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r97, d99.Reg)
			}
			r98 := ctx.AllocRegExcept(r97)
			ctx.W.EmitMovRegMem(r98, r97, 0)
			ctx.FreeReg(r97)
			d118 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			ctx.FreeDesc(&d117)
			var d119 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d98.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d98.Reg, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
			}
			if d119.Loc == scm.LocReg && d98.Loc == scm.LocReg && d119.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d98)
			d120 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() - d119.Imm.Int())}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d119.Loc == scm.LocImm {
				if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d120.Reg, int32(d119.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
				ctx.W.EmitSubInt64(d120.Reg, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else {
				ctx.W.EmitSubInt64(d120.Reg, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			}
			if d121.Loc == scm.LocReg && d120.Loc == scm.LocReg && d121.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			var d122 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d118.Imm.Int()) >> uint64(d121.Imm.Int())))}
			} else if d121.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d118.Reg, uint8(d121.Imm.Int()))
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d118.Reg}
			} else {
				{
					shiftSrc := d118.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d121.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d121.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d121.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d122.Loc == scm.LocReg && d118.Loc == scm.LocReg && d122.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() | d122.Imm.Int())}
			} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r99 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r99, d103.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d122.Loc == scm.LocImm {
				r100 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r100, d103.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r100, int32(d122.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
					ctx.W.EmitOrInt64(r100, scratch)
					ctx.FreeReg(scratch)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			} else {
				r101 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r101, d103.Reg)
				ctx.W.EmitOrInt64(r101, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			}
			if d123.Loc == scm.LocReg && d103.Loc == scm.LocReg && d123.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			ctx.EmitStoreToStack(d123, 88)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl24)
			ctx.W.ResolveFixups()
			d124 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			if r77 { ctx.UnprotectReg(r78) }
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d124.Imm.Int()))))}
			} else {
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r102, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			}
			ctx.FreeDesc(&d124)
			var d126 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r103, thisptr.Reg, off)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			}
			var d127 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d126.Imm.Int())}
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d125.Reg}
			} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
			} else if d125.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d125.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d126.Loc == scm.LocImm {
				if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d125.Reg, int32(d126.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
				ctx.W.EmitAddInt64(d125.Reg, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d125.Reg}
			} else {
				ctx.W.EmitAddInt64(d125.Reg, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d125.Reg}
			}
			if d127.Loc == scm.LocReg && d125.Loc == scm.LocReg && d127.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d126)
			var d128 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r104, thisptr.Reg, off)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d128.Loc == scm.LocImm {
				if d128.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d128.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d128)
			ctx.W.MarkLabel(lbl22)
			d129 := d53
			if d129.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: d129.Type, Imm: scm.NewInt(int64(uint64(d129.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d129.Reg, 32)
				ctx.W.EmitShrRegImm8(d129.Reg, 32)
			}
			ctx.EmitStoreToStack(d129, 56)
			d130 := d55
			if d130.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: d130.Type, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d130.Reg, 32)
				ctx.W.EmitShrRegImm8(d130.Reg, 32)
			}
			ctx.EmitStoreToStack(d130, 64)
			lbl33 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl21)
			var d131 scm.JITValueDesc
			if d53.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(scratch, d53.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d131.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: d131.Type, Imm: scm.NewInt(int64(uint64(d131.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d131.Reg, 32)
				ctx.W.EmitShrRegImm8(d131.Reg, 32)
			}
			if d131.Loc == scm.LocReg && d53.Loc == scm.LocReg && d131.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			d132 := d54
			if d132.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: d132.Type, Imm: scm.NewInt(int64(uint64(d132.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d132.Reg, 32)
				ctx.W.EmitShrRegImm8(d132.Reg, 32)
			}
			ctx.EmitStoreToStack(d132, 56)
			d133 := d131
			if d133.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: d133.Type, Imm: scm.NewInt(int64(uint64(d133.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d133.Reg, 32)
				ctx.W.EmitShrRegImm8(d133.Reg, 32)
			}
			ctx.EmitStoreToStack(d133, 64)
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl31)
			r105 := d93.Loc == scm.LocReg
			r106 := d93.Reg
			if r105 { ctx.ProtectReg(r106) }
			r107 := ctx.AllocReg()
			lbl34 := ctx.W.ReserveLabel()
			var d134 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d93.Imm.Int()))))}
			} else {
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r108, d93.Reg)
				ctx.W.EmitShlRegImm8(r108, 32)
				ctx.W.EmitShrRegImm8(r108, 32)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			}
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r109, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
			}
			var d136 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d135.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d135.Reg)
				ctx.W.EmitShlRegImm8(r110, 56)
				ctx.W.EmitShrRegImm8(r110, 56)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			}
			ctx.FreeDesc(&d135)
			var d137 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() * d136.Imm.Int())}
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d136.Loc == scm.LocImm {
				if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d134.Reg, int32(d136.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
				ctx.W.EmitImulInt64(d134.Reg, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else {
				ctx.W.EmitImulInt64(d134.Reg, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			}
			if d137.Loc == scm.LocReg && d134.Loc == scm.LocReg && d137.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d134)
			ctx.FreeDesc(&d136)
			var d138 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r111, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r111}
			}
			var d139 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() / 64)}
			} else {
				r112 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r112, d137.Reg)
				ctx.W.EmitShrRegImm8(r112, 6)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			}
			if d139.Loc == scm.LocReg && d137.Loc == scm.LocReg && d139.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			r113 := ctx.AllocReg()
			if d139.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r113, uint64(d139.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r113, d139.Reg)
				ctx.W.EmitShlRegImm8(r113, 3)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
				ctx.W.EmitAddInt64(r113, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r113, d138.Reg)
			}
			r114 := ctx.AllocRegExcept(r113)
			ctx.W.EmitMovRegMem(r114, r113, 0)
			ctx.FreeReg(r113)
			d140 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
			ctx.FreeDesc(&d139)
			var d141 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				r115 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r115, d137.Reg)
				ctx.W.EmitAndRegImm32(r115, 63)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			}
			if d141.Loc == scm.LocReg && d137.Loc == scm.LocReg && d141.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			var d142 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d140.Imm.Int()) << uint64(d141.Imm.Int())))}
			} else if d141.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d140.Reg, uint8(d141.Imm.Int()))
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d140.Reg}
			} else {
				{
					shiftSrc := d140.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d141.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d141.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d141.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d142.Loc == scm.LocReg && d140.Loc == scm.LocReg && d142.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			ctx.FreeDesc(&d141)
			var d143 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r116, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
			}
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			if d143.Loc == scm.LocImm {
				if d143.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
			ctx.EmitStoreToStack(d142, 96)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d143.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl37)
			ctx.EmitStoreToStack(d142, 96)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl37)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d143)
			ctx.W.MarkLabel(lbl36)
			r117 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r117, 96)
			ctx.ProtectReg(r117)
			d144 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r117}
			ctx.UnprotectReg(r117)
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r118 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r118, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r118}
			}
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d145.Imm.Int()))))}
			} else {
				r119 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r119, d145.Reg)
				ctx.W.EmitShlRegImm8(r119, 56)
				ctx.W.EmitShrRegImm8(r119, 56)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
			}
			ctx.FreeDesc(&d145)
			d147 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d148 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d146.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() - d146.Imm.Int())}
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d147.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d146.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d146.Loc == scm.LocImm {
				if d146.Imm.Int() >= -2147483648 && d146.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d147.Reg, int32(d146.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d146.Imm.Int()))
				ctx.W.EmitSubInt64(d147.Reg, scm.RegR11)
				}
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
			} else {
				ctx.W.EmitSubInt64(d147.Reg, d146.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
			}
			if d148.Loc == scm.LocReg && d147.Loc == scm.LocReg && d148.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			var d149 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d144.Imm.Int()) >> uint64(d148.Imm.Int())))}
			} else if d148.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d144.Reg, uint8(d148.Imm.Int()))
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d144.Reg}
			} else {
				{
					shiftSrc := d144.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d148.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d148.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d148.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d149.Loc == scm.LocReg && d144.Loc == scm.LocReg && d149.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d148)
			ctx.EmitMovToReg(r107, d149)
			ctx.W.EmitJmp(lbl34)
			ctx.FreeDesc(&d149)
			ctx.W.MarkLabel(lbl35)
			var d150 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				r120 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r120, d137.Reg)
				ctx.W.EmitAndRegImm32(r120, 63)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			}
			if d150.Loc == scm.LocReg && d137.Loc == scm.LocReg && d150.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			}
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d151.Imm.Int()))))}
			} else {
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r122, d151.Reg)
				ctx.W.EmitShlRegImm8(r122, 56)
				ctx.W.EmitShrRegImm8(r122, 56)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			}
			ctx.FreeDesc(&d151)
			var d153 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() + d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
			} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d152.Reg}
			} else if d150.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d150.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d152.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d152.Loc == scm.LocImm {
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d150.Reg, int32(d152.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
				ctx.W.EmitAddInt64(d150.Reg, scm.RegR11)
				}
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
			} else {
				ctx.W.EmitAddInt64(d150.Reg, d152.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
			}
			if d153.Loc == scm.LocReg && d150.Loc == scm.LocReg && d153.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d152)
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d153.Imm.Int()) > uint64(64))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d153.Reg, 64)
				ctx.W.EmitSetcc(r123, scm.CcA)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r123}
			}
			ctx.FreeDesc(&d153)
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d154.Loc == scm.LocImm {
				if d154.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
			ctx.EmitStoreToStack(d142, 96)
					ctx.W.EmitJmp(lbl36)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d154.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			ctx.EmitStoreToStack(d142, 96)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d154)
			ctx.W.MarkLabel(lbl38)
			var d155 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() / 64)}
			} else {
				r124 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r124, d137.Reg)
				ctx.W.EmitShrRegImm8(r124, 6)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			}
			if d155.Loc == scm.LocReg && d137.Loc == scm.LocReg && d155.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d155.Reg, int32(1))
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
			}
			if d156.Loc == scm.LocReg && d155.Loc == scm.LocReg && d156.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			r125 := ctx.AllocReg()
			if d156.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r125, uint64(d156.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r125, d156.Reg)
				ctx.W.EmitShlRegImm8(r125, 3)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
				ctx.W.EmitAddInt64(r125, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r125, d138.Reg)
			}
			r126 := ctx.AllocRegExcept(r125)
			ctx.W.EmitMovRegMem(r126, r125, 0)
			ctx.FreeReg(r125)
			d157 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.FreeDesc(&d156)
			var d158 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d137.Reg, 63)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d137.Reg}
			}
			if d158.Loc == scm.LocReg && d137.Loc == scm.LocReg && d158.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			d159 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() - d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d159.Reg}
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d158.Loc == scm.LocImm {
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d159.Reg, int32(d158.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
				ctx.W.EmitSubInt64(d159.Reg, scm.RegR11)
				}
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d159.Reg}
			} else {
				ctx.W.EmitSubInt64(d159.Reg, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d159.Reg}
			}
			if d160.Loc == scm.LocReg && d159.Loc == scm.LocReg && d160.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			var d161 scm.JITValueDesc
			if d157.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d157.Imm.Int()) >> uint64(d160.Imm.Int())))}
			} else if d160.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d157.Reg, uint8(d160.Imm.Int()))
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
			} else {
				{
					shiftSrc := d157.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d160.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d160.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d160.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d161.Loc == scm.LocReg && d157.Loc == scm.LocReg && d161.Reg == d157.Reg {
				ctx.TransferReg(d157.Reg)
				d157.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d157)
			ctx.FreeDesc(&d160)
			var d162 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() | d161.Imm.Int())}
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
			} else if d161.Loc == scm.LocImm && d161.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r127, d142.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d161.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d161.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r128, d142.Reg)
				if d161.Imm.Int() >= -2147483648 && d161.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d161.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d161.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scratch)
					ctx.FreeReg(scratch)
				}
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			} else {
				r129 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r129, d142.Reg)
				ctx.W.EmitOrInt64(r129, d161.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			}
			if d162.Loc == scm.LocReg && d142.Loc == scm.LocReg && d162.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.EmitStoreToStack(d162, 96)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl34)
			ctx.W.ResolveFixups()
			d163 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
			if r105 { ctx.UnprotectReg(r106) }
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d163.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d163.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			}
			ctx.FreeDesc(&d163)
			var d165 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
			}
			var d166 scm.JITValueDesc
			if d164.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + d165.Imm.Int())}
			} else if d165.Loc == scm.LocImm && d165.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d164.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d165.Loc == scm.LocImm {
				if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d164.Reg, int32(d165.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.W.EmitAddInt64(d164.Reg, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else {
				ctx.W.EmitAddInt64(d164.Reg, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			}
			if d166.Loc == scm.LocReg && d164.Loc == scm.LocReg && d166.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.FreeDesc(&d165)
			r132 := ctx.AllocReg()
			lbl40 := ctx.W.ReserveLabel()
			var d167 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d93.Imm.Int()))))}
			} else {
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r133, d93.Reg)
				ctx.W.EmitShlRegImm8(r133, 32)
				ctx.W.EmitShrRegImm8(r133, 32)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			}
			ctx.FreeDesc(&d93)
			var d168 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r134, thisptr.Reg, off)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r134}
			}
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d168.Imm.Int()))))}
			} else {
				r135 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r135, d168.Reg)
				ctx.W.EmitShlRegImm8(r135, 56)
				ctx.W.EmitShrRegImm8(r135, 56)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			}
			ctx.FreeDesc(&d168)
			var d170 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() * d169.Imm.Int())}
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d167.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d169.Loc == scm.LocImm {
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d167.Reg, int32(d169.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
				ctx.W.EmitImulInt64(d167.Reg, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
			} else {
				ctx.W.EmitImulInt64(d167.Reg, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
			}
			if d170.Loc == scm.LocReg && d167.Loc == scm.LocReg && d170.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.FreeDesc(&d169)
			var d171 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() / 64)}
			} else {
				r136 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r136, d170.Reg)
				ctx.W.EmitShrRegImm8(r136, 6)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			}
			if d171.Loc == scm.LocReg && d170.Loc == scm.LocReg && d171.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			r137 := ctx.AllocReg()
			if d171.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r137, uint64(d171.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r137, d171.Reg)
				ctx.W.EmitShlRegImm8(r137, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r137, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r137, d13.Reg)
			}
			r138 := ctx.AllocRegExcept(r137)
			ctx.W.EmitMovRegMem(r138, r137, 0)
			ctx.FreeReg(r137)
			d172 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r138}
			ctx.FreeDesc(&d171)
			var d173 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				r139 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r139, d170.Reg)
				ctx.W.EmitAndRegImm32(r139, 63)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			}
			if d173.Loc == scm.LocReg && d170.Loc == scm.LocReg && d173.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			var d174 scm.JITValueDesc
			if d172.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d172.Imm.Int()) << uint64(d173.Imm.Int())))}
			} else if d173.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d172.Reg, uint8(d173.Imm.Int()))
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
			} else {
				{
					shiftSrc := d172.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d173.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d173.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d173.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d174.Loc == scm.LocReg && d172.Loc == scm.LocReg && d174.Reg == d172.Reg {
				ctx.TransferReg(d172.Reg)
				d172.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r140, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
			}
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d175.Loc == scm.LocImm {
				if d175.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
			ctx.EmitStoreToStack(d174, 104)
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d175.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
			ctx.EmitStoreToStack(d174, 104)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d175)
			ctx.W.MarkLabel(lbl42)
			r141 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r141, 104)
			ctx.ProtectReg(r141)
			d176 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r141}
			ctx.UnprotectReg(r141)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r142 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r142, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142}
			}
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d177.Imm.Int()))))}
			} else {
				r143 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r143, d177.Reg)
				ctx.W.EmitShlRegImm8(r143, 56)
				ctx.W.EmitShrRegImm8(r143, 56)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
			}
			ctx.FreeDesc(&d177)
			d179 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() - d178.Imm.Int())}
			} else if d178.Loc == scm.LocImm && d178.Imm.Int() == 0 {
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d179.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d178.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d178.Loc == scm.LocImm {
				if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d179.Reg, int32(d178.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
				ctx.W.EmitSubInt64(d179.Reg, scm.RegR11)
				}
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			} else {
				ctx.W.EmitSubInt64(d179.Reg, d178.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			}
			if d180.Loc == scm.LocReg && d179.Loc == scm.LocReg && d180.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d178)
			var d181 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d176.Imm.Int()) >> uint64(d180.Imm.Int())))}
			} else if d180.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d176.Reg, uint8(d180.Imm.Int()))
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			} else {
				{
					shiftSrc := d176.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d180.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d180.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d180.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d181.Loc == scm.LocReg && d176.Loc == scm.LocReg && d181.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d180)
			ctx.EmitMovToReg(r132, d181)
			ctx.W.EmitJmp(lbl40)
			ctx.FreeDesc(&d181)
			ctx.W.MarkLabel(lbl41)
			var d182 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				r144 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r144, d170.Reg)
				ctx.W.EmitAndRegImm32(r144, 63)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			}
			if d182.Loc == scm.LocReg && d170.Loc == scm.LocReg && d182.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			var d183 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r145, thisptr.Reg, off)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			}
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d183.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d183.Reg)
				ctx.W.EmitShlRegImm8(r146, 56)
				ctx.W.EmitShrRegImm8(r146, 56)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			}
			ctx.FreeDesc(&d183)
			var d185 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() + d184.Imm.Int())}
			} else if d184.Loc == scm.LocImm && d184.Imm.Int() == 0 {
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d182.Reg}
			} else if d182.Loc == scm.LocImm && d182.Imm.Int() == 0 {
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d184.Reg}
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d182.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d184.Loc == scm.LocImm {
				if d184.Imm.Int() >= -2147483648 && d184.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d182.Reg, int32(d184.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(d182.Reg, scm.RegR11)
				}
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d182.Reg}
			} else {
				ctx.W.EmitAddInt64(d182.Reg, d184.Reg)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d182.Reg}
			}
			if d185.Loc == scm.LocReg && d182.Loc == scm.LocReg && d185.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.FreeDesc(&d184)
			var d186 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d185.Imm.Int()) > uint64(64))}
			} else {
				r147 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d185.Reg, 64)
				ctx.W.EmitSetcc(r147, scm.CcA)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r147}
			}
			ctx.FreeDesc(&d185)
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			if d186.Loc == scm.LocImm {
				if d186.Imm.Bool() {
					ctx.W.EmitJmp(lbl44)
				} else {
			ctx.EmitStoreToStack(d174, 104)
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d186.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
			ctx.EmitStoreToStack(d174, 104)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d186)
			ctx.W.MarkLabel(lbl44)
			var d187 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() / 64)}
			} else {
				r148 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r148, d170.Reg)
				ctx.W.EmitShrRegImm8(r148, 6)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			}
			if d187.Loc == scm.LocReg && d170.Loc == scm.LocReg && d187.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			var d188 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d187.Reg, int32(1))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d187.Reg}
			}
			if d188.Loc == scm.LocReg && d187.Loc == scm.LocReg && d188.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d187)
			r149 := ctx.AllocReg()
			if d188.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r149, uint64(d188.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r149, d188.Reg)
				ctx.W.EmitShlRegImm8(r149, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r149, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r149, d13.Reg)
			}
			r150 := ctx.AllocRegExcept(r149)
			ctx.W.EmitMovRegMem(r150, r149, 0)
			ctx.FreeReg(r149)
			d189 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
			ctx.FreeDesc(&d188)
			var d190 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d170.Reg, 63)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
			}
			if d190.Loc == scm.LocReg && d170.Loc == scm.LocReg && d190.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			d191 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d192 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d190.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() - d190.Imm.Int())}
			} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d191.Reg}
			} else if d191.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d191.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d190.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d190.Loc == scm.LocImm {
				if d190.Imm.Int() >= -2147483648 && d190.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d191.Reg, int32(d190.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d190.Imm.Int()))
				ctx.W.EmitSubInt64(d191.Reg, scm.RegR11)
				}
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d191.Reg}
			} else {
				ctx.W.EmitSubInt64(d191.Reg, d190.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d191.Reg}
			}
			if d192.Loc == scm.LocReg && d191.Loc == scm.LocReg && d192.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d190)
			var d193 scm.JITValueDesc
			if d189.Loc == scm.LocImm && d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d189.Imm.Int()) >> uint64(d192.Imm.Int())))}
			} else if d192.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d189.Reg, uint8(d192.Imm.Int()))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d189.Reg}
			} else {
				{
					shiftSrc := d189.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d192.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d192.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d192.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d193.Loc == scm.LocReg && d189.Loc == scm.LocReg && d193.Reg == d189.Reg {
				ctx.TransferReg(d189.Reg)
				d189.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d189)
			ctx.FreeDesc(&d192)
			var d194 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() | d193.Imm.Int())}
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d193.Reg}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				r151 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r151, d174.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d193.Loc == scm.LocImm {
				r152 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r152, d174.Reg)
				if d193.Imm.Int() >= -2147483648 && d193.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r152, int32(d193.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
					ctx.W.EmitOrInt64(r152, scratch)
					ctx.FreeReg(scratch)
				}
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			} else {
				r153 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r153, d174.Reg)
				ctx.W.EmitOrInt64(r153, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			}
			if d194.Loc == scm.LocReg && d174.Loc == scm.LocReg && d194.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d193)
			ctx.EmitStoreToStack(d194, 104)
			ctx.W.EmitJmp(lbl42)
			ctx.W.MarkLabel(lbl40)
			ctx.W.ResolveFixups()
			d195 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r132}
			ctx.FreeDesc(&d93)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d195.Imm.Int()))))}
			} else {
				r154 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r154, d195.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			}
			ctx.FreeDesc(&d195)
			var d197 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r155 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r155, thisptr.Reg, off)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
			}
			var d198 scm.JITValueDesc
			if d196.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d196.Imm.Int() + d197.Imm.Int())}
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d196.Reg}
			} else if d196.Loc == scm.LocImm && d196.Imm.Int() == 0 {
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d197.Reg}
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d196.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d197.Loc == scm.LocImm {
				if d197.Imm.Int() >= -2147483648 && d197.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d196.Reg, int32(d197.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d197.Imm.Int()))
				ctx.W.EmitAddInt64(d196.Reg, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d196.Reg}
			} else {
				ctx.W.EmitAddInt64(d196.Reg, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d196.Reg}
			}
			if d198.Loc == scm.LocReg && d196.Loc == scm.LocReg && d198.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.FreeDesc(&d197)
			var d199 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r156, 32)
				ctx.W.EmitShrRegImm8(r156, 32)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			}
			ctx.FreeDesc(&idxInt)
			var d200 scm.JITValueDesc
			if d199.Loc == scm.LocImm && d198.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d199.Imm.Int() - d198.Imm.Int())}
			} else if d198.Loc == scm.LocImm && d198.Imm.Int() == 0 {
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d199.Reg}
			} else if d199.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d199.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d198.Reg)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d198.Loc == scm.LocImm {
				if d198.Imm.Int() >= -2147483648 && d198.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d199.Reg, int32(d198.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d198.Imm.Int()))
				ctx.W.EmitSubInt64(d199.Reg, scm.RegR11)
				}
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d199.Reg}
			} else {
				ctx.W.EmitSubInt64(d199.Reg, d198.Reg)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d199.Reg}
			}
			if d200.Loc == scm.LocReg && d199.Loc == scm.LocReg && d200.Reg == d199.Reg {
				ctx.TransferReg(d199.Reg)
				d199.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d199)
			ctx.FreeDesc(&d198)
			var d201 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() * d166.Imm.Int())}
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d200.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d166.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d166.Loc == scm.LocImm {
				if d166.Imm.Int() >= -2147483648 && d166.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d200.Reg, int32(d166.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d166.Imm.Int()))
				ctx.W.EmitImulInt64(d200.Reg, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
			} else {
				ctx.W.EmitImulInt64(d200.Reg, d166.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
			}
			if d201.Loc == scm.LocReg && d200.Loc == scm.LocReg && d201.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d166)
			var d202 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() + d201.Imm.Int())}
			} else if d201.Loc == scm.LocImm && d201.Imm.Int() == 0 {
				r157 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r157, d127.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d201.Reg}
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(scratch, d127.Reg)
				if d201.Imm.Int() >= -2147483648 && d201.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d201.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d201.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r158 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r158, d127.Reg)
				ctx.W.EmitAddInt64(r158, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
			}
			if d202.Loc == scm.LocReg && d127.Loc == scm.LocReg && d202.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d202.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d202.Reg}
			}
			ctx.FreeDesc(&d202)
			ctx.W.EmitMakeFloat(result, d203)
			if d203.Loc == scm.LocReg { ctx.FreeReg(d203.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl30)
			var d204 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r159, thisptr.Reg, off)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r159}
			}
			var d205 scm.JITValueDesc
			if d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d204.Imm.Int()))))}
			} else {
				r160 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r160, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			}
			ctx.FreeDesc(&d204)
			var d206 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d127.Imm.Int() == d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm {
				r161 := ctx.AllocReg()
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d127.Reg, int32(d205.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d205.Imm.Int()))
					ctx.W.EmitCmpInt64(d127.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r161, scm.CcE)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r161}
			} else if d127.Loc == scm.LocImm {
				r162 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d205.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r162, scm.CcE)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
			} else {
				r163 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d127.Reg, d205.Reg)
				ctx.W.EmitSetcc(r163, scm.CcE)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r163}
			}
			ctx.FreeDesc(&d127)
			ctx.FreeDesc(&d205)
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d206.Loc == scm.LocImm {
				if d206.Imm.Bool() {
					ctx.W.EmitJmp(lbl46)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d206.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d206)
			ctx.W.MarkLabel(lbl33)
			r164 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r164, 56)
			ctx.ProtectReg(r164)
			d207 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r164}
			r165 := ctx.AllocRegExcept(r164)
			ctx.EmitLoadFromStack(r165, 64)
			ctx.ProtectReg(r165)
			d208 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r165}
			ctx.UnprotectReg(r164)
			ctx.UnprotectReg(r165)
			var d209 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d207.Imm.Int()) == uint64(d208.Imm.Int()))}
			} else if d208.Loc == scm.LocImm {
				r166 := ctx.AllocRegExcept(d207.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d207.Reg, int32(d208.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d208.Imm.Int()))
					ctx.W.EmitCmpInt64(d207.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r166, scm.CcE)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r166}
			} else if d207.Loc == scm.LocImm {
				r167 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d208.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r167, scm.CcE)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r167}
			} else {
				r168 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitCmpInt64(d207.Reg, d208.Reg)
				ctx.W.EmitSetcc(r168, scm.CcE)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r168}
			}
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d209.Loc == scm.LocImm {
				if d209.Imm.Bool() {
			d210 := d207
			if d210.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: d210.Type, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d210.Reg, 32)
				ctx.W.EmitShrRegImm8(d210.Reg, 32)
			}
			ctx.EmitStoreToStack(d210, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl48)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d209.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl49)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl49)
			d211 := d207
			if d211.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: d211.Type, Imm: scm.NewInt(int64(uint64(d211.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d211.Reg, 32)
				ctx.W.EmitShrRegImm8(d211.Reg, 32)
			}
			ctx.EmitStoreToStack(d211, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d209)
			ctx.W.MarkLabel(lbl46)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl48)
			var d212 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() + d208.Imm.Int())}
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r169 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r169, d207.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
			} else if d207.Loc == scm.LocImm && d207.Imm.Int() == 0 {
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
			} else if d207.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d208.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(scratch, d207.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r170 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r170, d207.Reg)
				ctx.W.EmitAddInt64(r170, d208.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
			}
			if d212.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: d212.Type, Imm: scm.NewInt(int64(uint64(d212.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d212.Reg, 32)
				ctx.W.EmitShrRegImm8(d212.Reg, 32)
			}
			if d212.Loc == scm.LocReg && d207.Loc == scm.LocReg && d212.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() / 2)}
			} else {
				ctx.W.EmitShrRegImm8(d212.Reg, 1)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d212.Reg}
			}
			if d213.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: d213.Type, Imm: scm.NewInt(int64(uint64(d213.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d213.Reg, 32)
				ctx.W.EmitShrRegImm8(d213.Reg, 32)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			d214 := d213
			if d214.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: d214.Type, Imm: scm.NewInt(int64(uint64(d214.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d214.Reg, 32)
				ctx.W.EmitShrRegImm8(d214.Reg, 32)
			}
			ctx.EmitStoreToStack(d214, 0)
			d215 := d207
			if d215.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: d215.Type, Imm: scm.NewInt(int64(uint64(d215.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d215.Reg, 32)
				ctx.W.EmitShrRegImm8(d215.Reg, 32)
			}
			ctx.EmitStoreToStack(d215, 8)
			d216 := d208
			if d216.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: d216.Type, Imm: scm.NewInt(int64(uint64(d216.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d216.Reg, 32)
				ctx.W.EmitShrRegImm8(d216.Reg, 32)
			}
			ctx.EmitStoreToStack(d216, 16)
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
