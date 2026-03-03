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
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
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
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
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
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
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
			ctx.W.MarkLabel(lbl1)
			r5 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r5, 0)
			ctx.ProtectReg(r5)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r5}
			ctx.BindReg(r5, &d6)
			r6 := ctx.AllocRegExcept(r5)
			ctx.EmitLoadFromStack(r6, 8)
			ctx.ProtectReg(r6)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r6}
			ctx.BindReg(r6, &d7)
			r7 := ctx.AllocRegExcept(r5, r6)
			ctx.EmitLoadFromStack(r7, 16)
			ctx.ProtectReg(r7)
			d8 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r7}
			ctx.BindReg(r7, &d8)
			ctx.UnprotectReg(r5)
			ctx.UnprotectReg(r6)
			ctx.UnprotectReg(r7)
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			r8 := d6.Loc == scm.LocReg
			r9 := d6.Reg
			if r8 { ctx.ProtectReg(r9) }
			lbl2 := ctx.W.ReserveLabel()
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d9 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d6.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r10, d6.Reg)
				ctx.W.EmitShlRegImm8(r10, 32)
				ctx.W.EmitShrRegImm8(r10, 32)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d9)
			}
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r11, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
				ctx.BindReg(r11, &d10)
			}
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d10.Imm.Int()))))}
			} else {
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r12, d10.Reg)
				ctx.W.EmitShlRegImm8(r12, 56)
				ctx.W.EmitShrRegImm8(r12, 56)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d11)
			}
			ctx.FreeDesc(&d10)
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d12 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() * d11.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(scratch, d9.Reg)
				if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d11.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else {
				r13 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r13, d9.Reg)
				ctx.W.EmitImulInt64(r13, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d12)
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
				ctx.BindReg(r14, &d13)
			}
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r15 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r15, d12.Reg)
				ctx.W.EmitShrRegImm8(r15, 6)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d14)
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
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
			ctx.BindReg(r17, &d15)
			ctx.FreeDesc(&d14)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d16 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r18 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r18, d12.Reg)
				ctx.W.EmitAndRegImm32(r18, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d16)
			}
			if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) << uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				r19 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r19, d15.Reg)
				ctx.W.EmitShlRegImm8(r19, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d17)
			} else {
				{
					shiftSrc := d15.Reg
					r20 := ctx.AllocRegExcept(d15.Reg)
					ctx.W.EmitMovRegReg(r20, d15.Reg)
					shiftSrc = r20
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
					ctx.BindReg(shiftSrc, &d17)
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
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r21, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
				ctx.BindReg(r21, &d18)
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
			r22 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r22, 72)
			ctx.ProtectReg(r22)
			d19 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r22}
			ctx.BindReg(r22, &d19)
			ctx.UnprotectReg(r22)
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r23, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
				ctx.BindReg(r23, &d20)
			}
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d20.Imm.Int()))))}
			} else {
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r24, d20.Reg)
				ctx.W.EmitShlRegImm8(r24, 56)
				ctx.W.EmitShrRegImm8(r24, 56)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d21)
			}
			ctx.FreeDesc(&d20)
			d22 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() - d21.Imm.Int())}
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r25, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r26 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r26, d22.Reg)
				ctx.W.EmitSubInt64(r26, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d23)
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			var d24 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d19.Imm.Int()) >> uint64(d23.Imm.Int())))}
			} else if d23.Loc == scm.LocImm {
				r27 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r27, d19.Reg)
				ctx.W.EmitShrRegImm8(r27, uint8(d23.Imm.Int()))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d24)
			} else {
				{
					shiftSrc := d19.Reg
					r28 := ctx.AllocRegExcept(d19.Reg)
					ctx.W.EmitMovRegReg(r28, d19.Reg)
					shiftSrc = r28
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
					ctx.BindReg(shiftSrc, &d24)
				}
			}
			if d24.Loc == scm.LocReg && d19.Loc == scm.LocReg && d24.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.FreeDesc(&d23)
			r29 := ctx.AllocReg()
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			ctx.EmitMovToReg(r29, d24)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl3)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d25 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r30 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r30, d12.Reg)
				ctx.W.EmitAndRegImm32(r30, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d25)
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
				r31 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r31, thisptr.Reg, off)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
				ctx.BindReg(r31, &d26)
			}
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d26.Imm.Int()))))}
			} else {
				r32 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r32, d26.Reg)
				ctx.W.EmitShlRegImm8(r32, 56)
				ctx.W.EmitShrRegImm8(r32, 56)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d27)
			}
			ctx.FreeDesc(&d26)
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			var d28 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() + d27.Imm.Int())}
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				r33 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r33, d25.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d28)
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
				ctx.BindReg(d27.Reg, &d28)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(scratch, d25.Reg)
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d27.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r34 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r34, d25.Reg)
				ctx.W.EmitAddInt64(r34, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d28)
			}
			if d28.Loc == scm.LocReg && d25.Loc == scm.LocReg && d28.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.FreeDesc(&d27)
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d28.Imm.Int()) > uint64(64))}
			} else {
				r35 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitCmpRegImm32(d28.Reg, 64)
				ctx.W.EmitSetcc(r35, scm.CcA)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
				ctx.BindReg(r35, &d29)
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
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d30 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r36 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r36, d12.Reg)
				ctx.W.EmitShrRegImm8(r36, 6)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d30)
			}
			if d30.Loc == scm.LocReg && d12.Loc == scm.LocReg && d30.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			r37 := ctx.AllocReg()
			if d31.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r37, uint64(d31.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r37, d31.Reg)
				ctx.W.EmitShlRegImm8(r37, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r37, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r37, d13.Reg)
			}
			r38 := ctx.AllocRegExcept(r37)
			ctx.W.EmitMovRegMem(r38, r37, 0)
			ctx.FreeReg(r37)
			d32 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d32)
			ctx.FreeDesc(&d31)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d33 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r39 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r39, d12.Reg)
				ctx.W.EmitAndRegImm32(r39, 63)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d33)
			}
			if d33.Loc == scm.LocReg && d12.Loc == scm.LocReg && d33.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			d34 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d33.Loc == scm.LocStack || d33.Loc == scm.LocStackPair { ctx.EnsureDesc(&d33) }
			var d35 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() - d33.Imm.Int())}
			} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r40, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(scratch, d34.Reg)
				if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d33.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else {
				r41 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r41, d34.Reg)
				ctx.W.EmitSubInt64(r41, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d35)
			}
			if d35.Loc == scm.LocReg && d34.Loc == scm.LocReg && d35.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			var d36 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) >> uint64(d35.Imm.Int())))}
			} else if d35.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r42, d32.Reg)
				ctx.W.EmitShrRegImm8(r42, uint8(d35.Imm.Int()))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d36)
			} else {
				{
					shiftSrc := d32.Reg
					r43 := ctx.AllocRegExcept(d32.Reg)
					ctx.W.EmitMovRegReg(r43, d32.Reg)
					shiftSrc = r43
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
					ctx.BindReg(shiftSrc, &d36)
				}
			}
			if d36.Loc == scm.LocReg && d32.Loc == scm.LocReg && d36.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.FreeDesc(&d35)
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			var d37 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() | d36.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
				ctx.BindReg(d36.Reg, &d37)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r44, d17.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d37)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			} else if d36.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r45, d17.Reg)
				if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r45, int32(d36.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d36.Imm.Int()))
					ctx.W.EmitOrInt64(r45, scm.RegR11)
				}
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d37)
			} else {
				r46 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r46, d17.Reg)
				ctx.W.EmitOrInt64(r46, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d37)
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
			d38 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d38)
			ctx.BindReg(r29, &d38)
			if r8 { ctx.UnprotectReg(r9) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d38.Imm.Int()))))}
			} else {
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r47, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d39)
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r48, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d40)
			}
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() + d40.Imm.Int())}
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				r49 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r49, d39.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d41)
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
				ctx.BindReg(d40.Reg, &d41)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(scratch, d39.Reg)
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d40.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else {
				r50 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r50, d39.Reg)
				ctx.W.EmitAddInt64(r50, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d41)
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d40)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d41.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d41.Reg)
				ctx.W.EmitShlRegImm8(r51, 32)
				ctx.W.EmitShrRegImm8(r51, 32)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d42)
			}
			ctx.FreeDesc(&d41)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			var d43 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d42.Imm.Int()))}
			} else if d42.Loc == scm.LocImm {
				r52 := ctx.AllocRegExcept(idxInt.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r52, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d43)
			} else if idxInt.Loc == scm.LocImm {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d42.Reg)
				ctx.W.EmitSetcc(r53, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d43)
			} else {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d42.Reg)
				ctx.W.EmitSetcc(r54, scm.CcB)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d43)
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
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d44 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
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
			lbl11 := ctx.W.ReserveLabel()
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
			ctx.W.MarkLabel(lbl11)
			r55 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r55, 32)
			ctx.ProtectReg(r55)
			d48 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r55}
			ctx.BindReg(r55, &d48)
			r56 := ctx.AllocRegExcept(r55)
			ctx.EmitLoadFromStack(r56, 40)
			ctx.ProtectReg(r56)
			d49 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r56}
			ctx.BindReg(r56, &d49)
			r57 := ctx.AllocRegExcept(r55, r56)
			ctx.EmitLoadFromStack(r57, 48)
			ctx.ProtectReg(r57)
			d50 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r57}
			ctx.BindReg(r57, &d50)
			ctx.UnprotectReg(r55)
			ctx.UnprotectReg(r56)
			ctx.UnprotectReg(r57)
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			if d50.Loc == scm.LocStack || d50.Loc == scm.LocStackPair { ctx.EnsureDesc(&d50) }
			var d51 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d49.Imm.Int()) == uint64(d50.Imm.Int()))}
			} else if d50.Loc == scm.LocImm {
				r58 := ctx.AllocRegExcept(d49.Reg)
				if d50.Imm.Int() >= -2147483648 && d50.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d49.Reg, int32(d50.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
					ctx.W.EmitCmpInt64(d49.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r58, scm.CcE)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d51)
			} else if d49.Loc == scm.LocImm {
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d50.Reg)
				ctx.W.EmitSetcc(r59, scm.CcE)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d51)
			} else {
				r60 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitCmpInt64(d49.Reg, d50.Reg)
				ctx.W.EmitSetcc(r60, scm.CcE)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d51)
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d51.Loc == scm.LocImm {
				if d51.Imm.Bool() {
			d52 := d49
			if d52.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: d52.Type, Imm: scm.NewInt(int64(uint64(d52.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d52.Reg, 32)
				ctx.W.EmitShrRegImm8(d52.Reg, 32)
			}
			ctx.EmitStoreToStack(d52, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d51.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl14)
			d53 := d49
			if d53.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: d53.Type, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d53.Reg, 32)
				ctx.W.EmitShrRegImm8(d53.Reg, 32)
			}
			ctx.EmitStoreToStack(d53, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d51)
			ctx.W.MarkLabel(lbl8)
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d54 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			}
			if d54.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: d54.Type, Imm: scm.NewInt(int64(uint64(d54.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d54.Reg, 32)
				ctx.W.EmitShrRegImm8(d54.Reg, 32)
			}
			if d54.Loc == scm.LocReg && d6.Loc == scm.LocReg && d54.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d55 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(scratch, d6.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			}
			if d55.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: d55.Type, Imm: scm.NewInt(int64(uint64(d55.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d55.Reg, 32)
				ctx.W.EmitShrRegImm8(d55.Reg, 32)
			}
			if d55.Loc == scm.LocReg && d6.Loc == scm.LocReg && d55.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			d56 := d55
			if d56.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: d56.Type, Imm: scm.NewInt(int64(uint64(d56.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d56.Reg, 32)
				ctx.W.EmitShrRegImm8(d56.Reg, 32)
			}
			ctx.EmitStoreToStack(d56, 32)
			d57 := d7
			if d57.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: d57.Type, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d57.Reg, 32)
				ctx.W.EmitShrRegImm8(d57.Reg, 32)
			}
			ctx.EmitStoreToStack(d57, 40)
			d58 := d54
			if d58.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: d58.Type, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d58.Reg, 32)
				ctx.W.EmitShrRegImm8(d58.Reg, 32)
			}
			ctx.EmitStoreToStack(d58, 48)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl13)
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			r61 := d48.Loc == scm.LocReg
			r62 := d48.Reg
			if r61 { ctx.ProtectReg(r62) }
			lbl15 := ctx.W.ReserveLabel()
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d59 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d48.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d48.Reg)
				ctx.W.EmitShlRegImm8(r63, 32)
				ctx.W.EmitShrRegImm8(r63, 32)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d59)
			}
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d60)
			}
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d60.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d60.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d61)
			}
			ctx.FreeDesc(&d60)
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			var d62 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() * d61.Imm.Int())}
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else {
				r66 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r66, d59.Reg)
				ctx.W.EmitImulInt64(r66, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d62)
			}
			if d62.Loc == scm.LocReg && d59.Loc == scm.LocReg && d62.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d61)
			var d63 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r67, thisptr.Reg, off)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r67}
				ctx.BindReg(r67, &d63)
			}
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d64 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r68 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r68, d62.Reg)
				ctx.W.EmitShrRegImm8(r68, 6)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d64)
			}
			if d64.Loc == scm.LocReg && d62.Loc == scm.LocReg && d64.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			r69 := ctx.AllocReg()
			if d64.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r69, uint64(d64.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r69, d64.Reg)
				ctx.W.EmitShlRegImm8(r69, 3)
			}
			if d63.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.W.EmitAddInt64(r69, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r69, d63.Reg)
			}
			r70 := ctx.AllocRegExcept(r69)
			ctx.W.EmitMovRegMem(r70, r69, 0)
			ctx.FreeReg(r69)
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d65)
			ctx.FreeDesc(&d64)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d66 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r71, d62.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d66)
			}
			if d66.Loc == scm.LocReg && d62.Loc == scm.LocReg && d66.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			if d66.Loc == scm.LocStack || d66.Loc == scm.LocStackPair { ctx.EnsureDesc(&d66) }
			var d67 scm.JITValueDesc
			if d65.Loc == scm.LocImm && d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) << uint64(d66.Imm.Int())))}
			} else if d66.Loc == scm.LocImm {
				r72 := ctx.AllocRegExcept(d65.Reg)
				ctx.W.EmitMovRegReg(r72, d65.Reg)
				ctx.W.EmitShlRegImm8(r72, uint8(d66.Imm.Int()))
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d67)
			} else {
				{
					shiftSrc := d65.Reg
					r73 := ctx.AllocRegExcept(d65.Reg)
					ctx.W.EmitMovRegReg(r73, d65.Reg)
					shiftSrc = r73
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d66.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d66.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d66.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d67)
				}
			}
			if d67.Loc == scm.LocReg && d65.Loc == scm.LocReg && d67.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.FreeDesc(&d66)
			var d68 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r74 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r74, thisptr.Reg, off)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
				ctx.BindReg(r74, &d68)
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d68.Loc == scm.LocImm {
				if d68.Imm.Bool() {
					ctx.W.EmitJmp(lbl16)
				} else {
			ctx.EmitStoreToStack(d67, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d68.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
			ctx.EmitStoreToStack(d67, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d68)
			ctx.W.MarkLabel(lbl17)
			r75 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r75, 80)
			ctx.ProtectReg(r75)
			d69 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r75}
			ctx.BindReg(r75, &d69)
			ctx.UnprotectReg(r75)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r76, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
				ctx.BindReg(r76, &d70)
			}
			if d70.Loc == scm.LocStack || d70.Loc == scm.LocStackPair { ctx.EnsureDesc(&d70) }
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d70.Imm.Int()))))}
			} else {
				r77 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r77, d70.Reg)
				ctx.W.EmitShlRegImm8(r77, 56)
				ctx.W.EmitShrRegImm8(r77, 56)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d71)
			}
			ctx.FreeDesc(&d70)
			d72 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm && d71.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d72.Imm.Int() - d71.Imm.Int())}
			} else if d71.Loc == scm.LocImm && d71.Imm.Int() == 0 {
				r78 := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(r78, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d73)
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d71.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d71.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(scratch, d72.Reg)
				if d71.Imm.Int() >= -2147483648 && d71.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d71.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else {
				r79 := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(r79, d72.Reg)
				ctx.W.EmitSubInt64(r79, d71.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d73)
			}
			if d73.Loc == scm.LocReg && d72.Loc == scm.LocReg && d73.Reg == d72.Reg {
				ctx.TransferReg(d72.Reg)
				d72.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			if d73.Loc == scm.LocStack || d73.Loc == scm.LocStackPair { ctx.EnsureDesc(&d73) }
			var d74 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d69.Imm.Int()) >> uint64(d73.Imm.Int())))}
			} else if d73.Loc == scm.LocImm {
				r80 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r80, d69.Reg)
				ctx.W.EmitShrRegImm8(r80, uint8(d73.Imm.Int()))
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d74)
			} else {
				{
					shiftSrc := d69.Reg
					r81 := ctx.AllocRegExcept(d69.Reg)
					ctx.W.EmitMovRegReg(r81, d69.Reg)
					shiftSrc = r81
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d73.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d73.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d73.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d74)
				}
			}
			if d74.Loc == scm.LocReg && d69.Loc == scm.LocReg && d74.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d69)
			ctx.FreeDesc(&d73)
			r82 := ctx.AllocReg()
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			ctx.EmitMovToReg(r82, d74)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl16)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d75 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r83 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r83, d62.Reg)
				ctx.W.EmitAndRegImm32(r83, 63)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d75)
			}
			if d75.Loc == scm.LocReg && d62.Loc == scm.LocReg && d75.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			var d76 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r84 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r84, thisptr.Reg, off)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r84}
				ctx.BindReg(r84, &d76)
			}
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			var d77 scm.JITValueDesc
			if d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d76.Imm.Int()))))}
			} else {
				r85 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r85, d76.Reg)
				ctx.W.EmitShlRegImm8(r85, 56)
				ctx.W.EmitShrRegImm8(r85, 56)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d77)
			}
			ctx.FreeDesc(&d76)
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			var d78 scm.JITValueDesc
			if d75.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + d77.Imm.Int())}
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				r86 := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(r86, d75.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d78)
			} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
				ctx.BindReg(d77.Reg, &d78)
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(scratch, d75.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else {
				r87 := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(r87, d75.Reg)
				ctx.W.EmitAddInt64(r87, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d78)
			}
			if d78.Loc == scm.LocReg && d75.Loc == scm.LocReg && d78.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.FreeDesc(&d77)
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d78.Imm.Int()) > uint64(64))}
			} else {
				r88 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitCmpRegImm32(d78.Reg, 64)
				ctx.W.EmitSetcc(r88, scm.CcA)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r88}
				ctx.BindReg(r88, &d79)
			}
			ctx.FreeDesc(&d78)
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d79.Loc == scm.LocImm {
				if d79.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
			ctx.EmitStoreToStack(d67, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl20)
			ctx.EmitStoreToStack(d67, 80)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d79)
			ctx.W.MarkLabel(lbl19)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d80 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() / 64)}
			} else {
				r89 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r89, d62.Reg)
				ctx.W.EmitShrRegImm8(r89, 6)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d80)
			}
			if d80.Loc == scm.LocReg && d62.Loc == scm.LocReg && d80.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair { ctx.EnsureDesc(&d80) }
			var d81 scm.JITValueDesc
			if d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d80.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegReg(scratch, d80.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			}
			if d81.Loc == scm.LocReg && d80.Loc == scm.LocReg && d81.Reg == d80.Reg {
				ctx.TransferReg(d80.Reg)
				d80.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			r90 := ctx.AllocReg()
			if d81.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r90, uint64(d81.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r90, d81.Reg)
				ctx.W.EmitShlRegImm8(r90, 3)
			}
			if d63.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.W.EmitAddInt64(r90, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r90, d63.Reg)
			}
			r91 := ctx.AllocRegExcept(r90)
			ctx.W.EmitMovRegMem(r91, r90, 0)
			ctx.FreeReg(r90)
			d82 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			ctx.BindReg(r91, &d82)
			ctx.FreeDesc(&d81)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d83 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() % 64)}
			} else {
				r92 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r92, d62.Reg)
				ctx.W.EmitAndRegImm32(r92, 63)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d83)
			}
			if d83.Loc == scm.LocReg && d62.Loc == scm.LocReg && d83.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			d84 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d83.Loc == scm.LocStack || d83.Loc == scm.LocStackPair { ctx.EnsureDesc(&d83) }
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() - d83.Imm.Int())}
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				r93 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r93, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
				ctx.BindReg(r93, &d85)
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else {
				r94 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r94, d84.Reg)
				ctx.W.EmitSubInt64(r94, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d85)
			}
			if d85.Loc == scm.LocReg && d84.Loc == scm.LocReg && d85.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d83)
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			var d86 scm.JITValueDesc
			if d82.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d82.Imm.Int()) >> uint64(d85.Imm.Int())))}
			} else if d85.Loc == scm.LocImm {
				r95 := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitMovRegReg(r95, d82.Reg)
				ctx.W.EmitShrRegImm8(r95, uint8(d85.Imm.Int()))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d86)
			} else {
				{
					shiftSrc := d82.Reg
					r96 := ctx.AllocRegExcept(d82.Reg)
					ctx.W.EmitMovRegReg(r96, d82.Reg)
					shiftSrc = r96
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d85.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d85.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d85.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d86)
				}
			}
			if d86.Loc == scm.LocReg && d82.Loc == scm.LocReg && d86.Reg == d82.Reg {
				ctx.TransferReg(d82.Reg)
				d82.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d82)
			ctx.FreeDesc(&d85)
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			var d87 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() | d86.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
				ctx.BindReg(d86.Reg, &d87)
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				r97 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r97, d67.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d87)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				r98 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r98, d67.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r98, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitOrInt64(r98, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d87)
			} else {
				r99 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r99, d67.Reg)
				ctx.W.EmitOrInt64(r99, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d87)
			}
			if d87.Loc == scm.LocReg && d67.Loc == scm.LocReg && d87.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.EmitStoreToStack(d87, 80)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl15)
			ctx.W.ResolveFixups()
			d88 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			ctx.BindReg(r82, &d88)
			ctx.BindReg(r82, &d88)
			if r61 { ctx.UnprotectReg(r62) }
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d88.Imm.Int()))))}
			} else {
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r100, d88.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d89)
			}
			ctx.FreeDesc(&d88)
			var d90 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r101 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r101, thisptr.Reg, off)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
				ctx.BindReg(r101, &d90)
			}
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			if d90.Loc == scm.LocStack || d90.Loc == scm.LocStackPair { ctx.EnsureDesc(&d90) }
			var d91 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() + d90.Imm.Int())}
			} else if d90.Loc == scm.LocImm && d90.Imm.Int() == 0 {
				r102 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r102, d89.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d91)
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d90.Reg}
				ctx.BindReg(d90.Reg, &d91)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d90.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d91)
			} else if d90.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(scratch, d89.Reg)
				if d90.Imm.Int() >= -2147483648 && d90.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d90.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d90.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d91)
			} else {
				r103 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r103, d89.Reg)
				ctx.W.EmitAddInt64(r103, d90.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d91)
			}
			if d91.Loc == scm.LocReg && d89.Loc == scm.LocReg && d91.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d89)
			ctx.FreeDesc(&d90)
			if d91.Loc == scm.LocStack || d91.Loc == scm.LocStackPair { ctx.EnsureDesc(&d91) }
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d91.Imm.Int()))))}
			} else {
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r104, d91.Reg)
				ctx.W.EmitShlRegImm8(r104, 32)
				ctx.W.EmitShrRegImm8(r104, 32)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d92)
			}
			ctx.FreeDesc(&d91)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair { ctx.EnsureDesc(&d92) }
			var d93 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d92.Imm.Int()))}
			} else if d92.Loc == scm.LocImm {
				r105 := ctx.AllocRegExcept(idxInt.Reg)
				if d92.Imm.Int() >= -2147483648 && d92.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d92.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r105, scm.CcB)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r105}
				ctx.BindReg(r105, &d93)
			} else if idxInt.Loc == scm.LocImm {
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d92.Reg)
				ctx.W.EmitSetcc(r106, scm.CcB)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r106}
				ctx.BindReg(r106, &d93)
			} else {
				r107 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d92.Reg)
				ctx.W.EmitSetcc(r107, scm.CcB)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r107}
				ctx.BindReg(r107, &d93)
			}
			ctx.FreeDesc(&d92)
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d93.Loc == scm.LocImm {
				if d93.Imm.Bool() {
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d93.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl22)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.FreeDesc(&d93)
			ctx.W.MarkLabel(lbl12)
			r108 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r108, 24)
			ctx.ProtectReg(r108)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r108}
			ctx.BindReg(r108, &d94)
			ctx.UnprotectReg(r108)
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d95 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d94.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r109, d94.Reg)
				ctx.W.EmitShlRegImm8(r109, 32)
				ctx.W.EmitShrRegImm8(r109, 32)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d95)
			}
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d95.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d95.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d95.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d95.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d95.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d95.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d95)
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			r110 := d94.Loc == scm.LocReg
			r111 := d94.Reg
			if r110 { ctx.ProtectReg(r111) }
			lbl24 := ctx.W.ReserveLabel()
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d96 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d94.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d94.Reg)
				ctx.W.EmitShlRegImm8(r112, 32)
				ctx.W.EmitShrRegImm8(r112, 32)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d96)
			}
			var d97 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r113 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r113, thisptr.Reg, off)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r113}
				ctx.BindReg(r113, &d97)
			}
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			var d98 scm.JITValueDesc
			if d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d97.Imm.Int()))))}
			} else {
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r114, d97.Reg)
				ctx.W.EmitShlRegImm8(r114, 56)
				ctx.W.EmitShrRegImm8(r114, 56)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
				ctx.BindReg(r114, &d98)
			}
			ctx.FreeDesc(&d97)
			if d96.Loc == scm.LocStack || d96.Loc == scm.LocStackPair { ctx.EnsureDesc(&d96) }
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			var d99 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d98.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() * d98.Imm.Int())}
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d99)
			} else if d98.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(scratch, d96.Reg)
				if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d98.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d99)
			} else {
				r115 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r115, d96.Reg)
				ctx.W.EmitImulInt64(r115, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d99)
			}
			if d99.Loc == scm.LocReg && d96.Loc == scm.LocReg && d99.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d96)
			ctx.FreeDesc(&d98)
			var d100 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r116, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
				ctx.BindReg(r116, &d100)
			}
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d101 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() / 64)}
			} else {
				r117 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r117, d99.Reg)
				ctx.W.EmitShrRegImm8(r117, 6)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d101)
			}
			if d101.Loc == scm.LocReg && d99.Loc == scm.LocReg && d101.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			r118 := ctx.AllocReg()
			if d101.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r118, uint64(d101.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r118, d101.Reg)
				ctx.W.EmitShlRegImm8(r118, 3)
			}
			if d100.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d100.Imm.Int()))
				ctx.W.EmitAddInt64(r118, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r118, d100.Reg)
			}
			r119 := ctx.AllocRegExcept(r118)
			ctx.W.EmitMovRegMem(r119, r118, 0)
			ctx.FreeReg(r118)
			d102 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r119}
			ctx.BindReg(r119, &d102)
			ctx.FreeDesc(&d101)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d103 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() % 64)}
			} else {
				r120 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r120, d99.Reg)
				ctx.W.EmitAndRegImm32(r120, 63)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d103)
			}
			if d103.Loc == scm.LocReg && d99.Loc == scm.LocReg && d103.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			var d104 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d102.Imm.Int()) << uint64(d103.Imm.Int())))}
			} else if d103.Loc == scm.LocImm {
				r121 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r121, d102.Reg)
				ctx.W.EmitShlRegImm8(r121, uint8(d103.Imm.Int()))
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d104)
			} else {
				{
					shiftSrc := d102.Reg
					r122 := ctx.AllocRegExcept(d102.Reg)
					ctx.W.EmitMovRegReg(r122, d102.Reg)
					shiftSrc = r122
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d103.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d103.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d103.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d104)
				}
			}
			if d104.Loc == scm.LocReg && d102.Loc == scm.LocReg && d104.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d103)
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r123, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
				ctx.BindReg(r123, &d105)
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d105.Loc == scm.LocImm {
				if d105.Imm.Bool() {
					ctx.W.EmitJmp(lbl25)
				} else {
			ctx.EmitStoreToStack(d104, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d105.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
			ctx.EmitStoreToStack(d104, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl25)
			}
			ctx.FreeDesc(&d105)
			ctx.W.MarkLabel(lbl26)
			r124 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r124, 88)
			ctx.ProtectReg(r124)
			d106 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r124}
			ctx.BindReg(r124, &d106)
			ctx.UnprotectReg(r124)
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r125 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r125, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r125}
				ctx.BindReg(r125, &d107)
			}
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d108 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d107.Imm.Int()))))}
			} else {
				r126 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r126, d107.Reg)
				ctx.W.EmitShlRegImm8(r126, 56)
				ctx.W.EmitShrRegImm8(r126, 56)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d108)
			}
			ctx.FreeDesc(&d107)
			d109 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair { ctx.EnsureDesc(&d108) }
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm && d108.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() - d108.Imm.Int())}
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r127, d109.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d110)
			} else if d109.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d109.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d108.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(scratch, d109.Reg)
				if d108.Imm.Int() >= -2147483648 && d108.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d108.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			} else {
				r128 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r128, d109.Reg)
				ctx.W.EmitSubInt64(r128, d108.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d110)
			}
			if d110.Loc == scm.LocReg && d109.Loc == scm.LocReg && d110.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			if d110.Loc == scm.LocStack || d110.Loc == scm.LocStackPair { ctx.EnsureDesc(&d110) }
			var d111 scm.JITValueDesc
			if d106.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d106.Imm.Int()) >> uint64(d110.Imm.Int())))}
			} else if d110.Loc == scm.LocImm {
				r129 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r129, d106.Reg)
				ctx.W.EmitShrRegImm8(r129, uint8(d110.Imm.Int()))
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d111)
			} else {
				{
					shiftSrc := d106.Reg
					r130 := ctx.AllocRegExcept(d106.Reg)
					ctx.W.EmitMovRegReg(r130, d106.Reg)
					shiftSrc = r130
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d110.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d110.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d110.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d111)
				}
			}
			if d111.Loc == scm.LocReg && d106.Loc == scm.LocReg && d111.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.FreeDesc(&d110)
			r131 := ctx.AllocReg()
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			ctx.EmitMovToReg(r131, d111)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl25)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d112 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() % 64)}
			} else {
				r132 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r132, d99.Reg)
				ctx.W.EmitAndRegImm32(r132, 63)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d112)
			}
			if d112.Loc == scm.LocReg && d99.Loc == scm.LocReg && d112.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			var d113 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r133, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
				ctx.BindReg(r133, &d113)
			}
			if d113.Loc == scm.LocStack || d113.Loc == scm.LocStackPair { ctx.EnsureDesc(&d113) }
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d113.Imm.Int()))))}
			} else {
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r134, d113.Reg)
				ctx.W.EmitShlRegImm8(r134, 56)
				ctx.W.EmitShrRegImm8(r134, 56)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d114)
			}
			ctx.FreeDesc(&d113)
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			var d115 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() + d114.Imm.Int())}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				r135 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r135, d112.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d115)
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d114.Reg}
				ctx.BindReg(d114.Reg, &d115)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d114.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(scratch, d112.Reg)
				if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d114.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else {
				r136 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r136, d112.Reg)
				ctx.W.EmitAddInt64(r136, d114.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d115)
			}
			if d115.Loc == scm.LocReg && d112.Loc == scm.LocReg && d115.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.FreeDesc(&d114)
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d115.Imm.Int()) > uint64(64))}
			} else {
				r137 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitCmpRegImm32(d115.Reg, 64)
				ctx.W.EmitSetcc(r137, scm.CcA)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r137}
				ctx.BindReg(r137, &d116)
			}
			ctx.FreeDesc(&d115)
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d116.Loc == scm.LocImm {
				if d116.Imm.Bool() {
					ctx.W.EmitJmp(lbl28)
				} else {
			ctx.EmitStoreToStack(d104, 88)
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d116.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
			ctx.EmitStoreToStack(d104, 88)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl28)
			}
			ctx.FreeDesc(&d116)
			ctx.W.MarkLabel(lbl28)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d117 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() / 64)}
			} else {
				r138 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r138, d99.Reg)
				ctx.W.EmitShrRegImm8(r138, 6)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d117)
			}
			if d117.Loc == scm.LocReg && d99.Loc == scm.LocReg && d117.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair { ctx.EnsureDesc(&d117) }
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(scratch, d117.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			}
			if d118.Loc == scm.LocReg && d117.Loc == scm.LocReg && d118.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d117)
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			r139 := ctx.AllocReg()
			if d118.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r139, uint64(d118.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r139, d118.Reg)
				ctx.W.EmitShlRegImm8(r139, 3)
			}
			if d100.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d100.Imm.Int()))
				ctx.W.EmitAddInt64(r139, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r139, d100.Reg)
			}
			r140 := ctx.AllocRegExcept(r139)
			ctx.W.EmitMovRegMem(r140, r139, 0)
			ctx.FreeReg(r139)
			d119 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
			ctx.BindReg(r140, &d119)
			ctx.FreeDesc(&d118)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d120 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() % 64)}
			} else {
				r141 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r141, d99.Reg)
				ctx.W.EmitAndRegImm32(r141, 63)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d120)
			}
			if d120.Loc == scm.LocReg && d99.Loc == scm.LocReg && d120.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			d121 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d121.Imm.Int() - d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				r142 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(r142, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(scratch, d121.Reg)
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d120.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r143 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(r143, d121.Reg)
				ctx.W.EmitSubInt64(r143, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d122)
			}
			if d122.Loc == scm.LocReg && d121.Loc == scm.LocReg && d122.Reg == d121.Reg {
				ctx.TransferReg(d121.Reg)
				d121.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d123 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d119.Imm.Int()) >> uint64(d122.Imm.Int())))}
			} else if d122.Loc == scm.LocImm {
				r144 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r144, d119.Reg)
				ctx.W.EmitShrRegImm8(r144, uint8(d122.Imm.Int()))
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d123)
			} else {
				{
					shiftSrc := d119.Reg
					r145 := ctx.AllocRegExcept(d119.Reg)
					ctx.W.EmitMovRegReg(r145, d119.Reg)
					shiftSrc = r145
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d122.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d122.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d122.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d123)
				}
			}
			if d123.Loc == scm.LocReg && d119.Loc == scm.LocReg && d123.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d122)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			var d124 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() | d123.Imm.Int())}
			} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d123.Reg}
				ctx.BindReg(d123.Reg, &d124)
			} else if d123.Loc == scm.LocImm && d123.Imm.Int() == 0 {
				r146 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r146, d104.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d124)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d123.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d124)
			} else if d123.Loc == scm.LocImm {
				r147 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r147, d104.Reg)
				if d123.Imm.Int() >= -2147483648 && d123.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r147, int32(d123.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
					ctx.W.EmitOrInt64(r147, scm.RegR11)
				}
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d124)
			} else {
				r148 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r148, d104.Reg)
				ctx.W.EmitOrInt64(r148, d123.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d124)
			}
			if d124.Loc == scm.LocReg && d104.Loc == scm.LocReg && d124.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d123)
			ctx.EmitStoreToStack(d124, 88)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl24)
			ctx.W.ResolveFixups()
			d125 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
			ctx.BindReg(r131, &d125)
			ctx.BindReg(r131, &d125)
			if r110 { ctx.UnprotectReg(r111) }
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d125.Imm.Int()))))}
			} else {
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r149, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d126)
			}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r150, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d127)
			}
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r151 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r151, d126.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d128)
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d128)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d126.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(scratch, d126.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else {
				r152 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r152, d126.Reg)
				ctx.W.EmitAddInt64(r152, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d128)
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d127)
			var d129 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r153, thisptr.Reg, off)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
				ctx.BindReg(r153, &d129)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d129.Loc == scm.LocImm {
				if d129.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d129.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d129)
			ctx.W.MarkLabel(lbl22)
			lbl33 := ctx.W.ReserveLabel()
			d130 := d48
			if d130.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: d130.Type, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d130.Reg, 32)
				ctx.W.EmitShrRegImm8(d130.Reg, 32)
			}
			ctx.EmitStoreToStack(d130, 56)
			d131 := d50
			if d131.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: d131.Type, Imm: scm.NewInt(int64(uint64(d131.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d131.Reg, 32)
				ctx.W.EmitShrRegImm8(d131.Reg, 32)
			}
			ctx.EmitStoreToStack(d131, 64)
			ctx.W.MarkLabel(lbl33)
			r154 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r154, 56)
			ctx.ProtectReg(r154)
			d132 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r154}
			ctx.BindReg(r154, &d132)
			r155 := ctx.AllocRegExcept(r154)
			ctx.EmitLoadFromStack(r155, 64)
			ctx.ProtectReg(r155)
			d133 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r155}
			ctx.BindReg(r155, &d133)
			ctx.UnprotectReg(r154)
			ctx.UnprotectReg(r155)
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d134 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d132.Imm.Int()) == uint64(d133.Imm.Int()))}
			} else if d133.Loc == scm.LocImm {
				r156 := ctx.AllocRegExcept(d132.Reg)
				if d133.Imm.Int() >= -2147483648 && d133.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d132.Reg, int32(d133.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
					ctx.W.EmitCmpInt64(d132.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r156, scm.CcE)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r156}
				ctx.BindReg(r156, &d134)
			} else if d132.Loc == scm.LocImm {
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d132.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d133.Reg)
				ctx.W.EmitSetcc(r157, scm.CcE)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r157}
				ctx.BindReg(r157, &d134)
			} else {
				r158 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitCmpInt64(d132.Reg, d133.Reg)
				ctx.W.EmitSetcc(r158, scm.CcE)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r158}
				ctx.BindReg(r158, &d134)
			}
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d134.Loc == scm.LocImm {
				if d134.Imm.Bool() {
			d135 := d132
			if d135.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: d135.Type, Imm: scm.NewInt(int64(uint64(d135.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d135.Reg, 32)
				ctx.W.EmitShrRegImm8(d135.Reg, 32)
			}
			ctx.EmitStoreToStack(d135, 24)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl34)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d134.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl34)
				ctx.W.MarkLabel(lbl35)
			d136 := d132
			if d136.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: d136.Type, Imm: scm.NewInt(int64(uint64(d136.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d136.Reg, 32)
				ctx.W.EmitShrRegImm8(d136.Reg, 32)
			}
			ctx.EmitStoreToStack(d136, 24)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d134)
			ctx.W.MarkLabel(lbl21)
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d137 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			}
			if d137.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: d137.Type, Imm: scm.NewInt(int64(uint64(d137.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d137.Reg, 32)
				ctx.W.EmitShrRegImm8(d137.Reg, 32)
			}
			if d137.Loc == scm.LocReg && d48.Loc == scm.LocReg && d137.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			d138 := d49
			if d138.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: d138.Type, Imm: scm.NewInt(int64(uint64(d138.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d138.Reg, 32)
				ctx.W.EmitShrRegImm8(d138.Reg, 32)
			}
			ctx.EmitStoreToStack(d138, 56)
			d139 := d137
			if d139.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: d139.Type, Imm: scm.NewInt(int64(uint64(d139.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d139.Reg, 32)
				ctx.W.EmitShrRegImm8(d139.Reg, 32)
			}
			ctx.EmitStoreToStack(d139, 64)
			ctx.W.EmitJmp(lbl33)
			ctx.W.MarkLabel(lbl31)
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			r159 := d94.Loc == scm.LocReg
			r160 := d94.Reg
			if r159 { ctx.ProtectReg(r160) }
			lbl36 := ctx.W.ReserveLabel()
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d140 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d94.Imm.Int()))))}
			} else {
				r161 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r161, d94.Reg)
				ctx.W.EmitShlRegImm8(r161, 32)
				ctx.W.EmitShrRegImm8(r161, 32)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d140)
			}
			var d141 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r162 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r162, thisptr.Reg, off)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r162}
				ctx.BindReg(r162, &d141)
			}
			if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair { ctx.EnsureDesc(&d141) }
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d141.Imm.Int()))))}
			} else {
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r163, d141.Reg)
				ctx.W.EmitShlRegImm8(r163, 56)
				ctx.W.EmitShrRegImm8(r163, 56)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d142)
			}
			ctx.FreeDesc(&d141)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			var d143 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() * d142.Imm.Int())}
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d140.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d143)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(scratch, d140.Reg)
				if d142.Imm.Int() >= -2147483648 && d142.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d142.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d143)
			} else {
				r164 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r164, d140.Reg)
				ctx.W.EmitImulInt64(r164, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d143)
			}
			if d143.Loc == scm.LocReg && d140.Loc == scm.LocReg && d143.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			ctx.FreeDesc(&d142)
			var d144 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				r165 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r165, thisptr.Reg, off)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
				ctx.BindReg(r165, &d144)
			}
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d145 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() / 64)}
			} else {
				r166 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r166, d143.Reg)
				ctx.W.EmitShrRegImm8(r166, 6)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d145)
			}
			if d145.Loc == scm.LocReg && d143.Loc == scm.LocReg && d145.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			r167 := ctx.AllocReg()
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r167, uint64(d145.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r167, d145.Reg)
				ctx.W.EmitShlRegImm8(r167, 3)
			}
			if d144.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(r167, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r167, d144.Reg)
			}
			r168 := ctx.AllocRegExcept(r167)
			ctx.W.EmitMovRegMem(r168, r167, 0)
			ctx.FreeReg(r167)
			d146 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r168}
			ctx.BindReg(r168, &d146)
			ctx.FreeDesc(&d145)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d147 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r169 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r169, d143.Reg)
				ctx.W.EmitAndRegImm32(r169, 63)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d147)
			}
			if d147.Loc == scm.LocReg && d143.Loc == scm.LocReg && d147.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			if d146.Loc == scm.LocStack || d146.Loc == scm.LocStackPair { ctx.EnsureDesc(&d146) }
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			var d148 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d146.Imm.Int()) << uint64(d147.Imm.Int())))}
			} else if d147.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r170, d146.Reg)
				ctx.W.EmitShlRegImm8(r170, uint8(d147.Imm.Int()))
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d148)
			} else {
				{
					shiftSrc := d146.Reg
					r171 := ctx.AllocRegExcept(d146.Reg)
					ctx.W.EmitMovRegReg(r171, d146.Reg)
					shiftSrc = r171
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d147.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d147.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d147.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d148)
				}
			}
			if d148.Loc == scm.LocReg && d146.Loc == scm.LocReg && d148.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d147)
			var d149 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r172, thisptr.Reg, off)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r172}
				ctx.BindReg(r172, &d149)
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d149.Loc == scm.LocImm {
				if d149.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
			ctx.EmitStoreToStack(d148, 96)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d149.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			ctx.EmitStoreToStack(d148, 96)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d149)
			ctx.W.MarkLabel(lbl38)
			r173 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r173, 96)
			ctx.ProtectReg(r173)
			d150 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r173}
			ctx.BindReg(r173, &d150)
			ctx.UnprotectReg(r173)
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r174, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
				ctx.BindReg(r174, &d151)
			}
			if d151.Loc == scm.LocStack || d151.Loc == scm.LocStackPair { ctx.EnsureDesc(&d151) }
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d151.Imm.Int()))))}
			} else {
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r175, d151.Reg)
				ctx.W.EmitShlRegImm8(r175, 56)
				ctx.W.EmitShrRegImm8(r175, 56)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d152)
			}
			ctx.FreeDesc(&d151)
			d153 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() - d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r176, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d154)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(scratch, d153.Reg)
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d152.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else {
				r177 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r177, d153.Reg)
				ctx.W.EmitSubInt64(r177, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d154)
			}
			if d154.Loc == scm.LocReg && d153.Loc == scm.LocReg && d154.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			var d155 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d150.Imm.Int()) >> uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r178 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r178, d150.Reg)
				ctx.W.EmitShrRegImm8(r178, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d155)
			} else {
				{
					shiftSrc := d150.Reg
					r179 := ctx.AllocRegExcept(d150.Reg)
					ctx.W.EmitMovRegReg(r179, d150.Reg)
					shiftSrc = r179
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d155)
				}
			}
			if d155.Loc == scm.LocReg && d150.Loc == scm.LocReg && d155.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d154)
			r180 := ctx.AllocReg()
			if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair { ctx.EnsureDesc(&d155) }
			ctx.EmitMovToReg(r180, d155)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl37)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d156 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r181 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r181, d143.Reg)
				ctx.W.EmitAndRegImm32(r181, 63)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d156)
			}
			if d156.Loc == scm.LocReg && d143.Loc == scm.LocReg && d156.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r182 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r182, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r182}
				ctx.BindReg(r182, &d157)
			}
			if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair { ctx.EnsureDesc(&d157) }
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r183, d157.Reg)
				ctx.W.EmitShlRegImm8(r183, 56)
				ctx.W.EmitShrRegImm8(r183, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
				ctx.BindReg(r183, &d158)
			}
			ctx.FreeDesc(&d157)
			if d156.Loc == scm.LocStack || d156.Loc == scm.LocStackPair { ctx.EnsureDesc(&d156) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d159 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() + d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r184, d156.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d159)
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
				ctx.BindReg(d158.Reg, &d159)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(scratch, d156.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else {
				r185 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r185, d156.Reg)
				ctx.W.EmitAddInt64(r185, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d159)
			}
			if d159.Loc == scm.LocReg && d156.Loc == scm.LocReg && d159.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d158)
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d159.Imm.Int()) > uint64(64))}
			} else {
				r186 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitCmpRegImm32(d159.Reg, 64)
				ctx.W.EmitSetcc(r186, scm.CcA)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r186}
				ctx.BindReg(r186, &d160)
			}
			ctx.FreeDesc(&d159)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d160.Loc == scm.LocImm {
				if d160.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
			ctx.EmitStoreToStack(d148, 96)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d160.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
			ctx.EmitStoreToStack(d148, 96)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d160)
			ctx.W.MarkLabel(lbl40)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d161 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() / 64)}
			} else {
				r187 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r187, d143.Reg)
				ctx.W.EmitShrRegImm8(r187, 6)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d161)
			}
			if d161.Loc == scm.LocReg && d143.Loc == scm.LocReg && d161.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d161.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegReg(scratch, d161.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			}
			if d162.Loc == scm.LocReg && d161.Loc == scm.LocReg && d162.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			if d162.Loc == scm.LocStack || d162.Loc == scm.LocStackPair { ctx.EnsureDesc(&d162) }
			r188 := ctx.AllocReg()
			if d162.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r188, uint64(d162.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r188, d162.Reg)
				ctx.W.EmitShlRegImm8(r188, 3)
			}
			if d144.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(r188, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r188, d144.Reg)
			}
			r189 := ctx.AllocRegExcept(r188)
			ctx.W.EmitMovRegMem(r189, r188, 0)
			ctx.FreeReg(r188)
			d163 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r189}
			ctx.BindReg(r189, &d163)
			ctx.FreeDesc(&d162)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d164 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r190 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r190, d143.Reg)
				ctx.W.EmitAndRegImm32(r190, 63)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d164)
			}
			if d164.Loc == scm.LocReg && d143.Loc == scm.LocReg && d164.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d143)
			d165 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() - d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r191 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(r191, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d166)
			} else if d165.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d165.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(scratch, d165.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d164.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else {
				r192 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(r192, d165.Reg)
				ctx.W.EmitSubInt64(r192, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d166)
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d166.Loc == scm.LocStack || d166.Loc == scm.LocStackPair { ctx.EnsureDesc(&d166) }
			var d167 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d163.Imm.Int()) >> uint64(d166.Imm.Int())))}
			} else if d166.Loc == scm.LocImm {
				r193 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r193, d163.Reg)
				ctx.W.EmitShrRegImm8(r193, uint8(d166.Imm.Int()))
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d167)
			} else {
				{
					shiftSrc := d163.Reg
					r194 := ctx.AllocRegExcept(d163.Reg)
					ctx.W.EmitMovRegReg(r194, d163.Reg)
					shiftSrc = r194
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
					ctx.BindReg(shiftSrc, &d167)
				}
			}
			if d167.Loc == scm.LocReg && d163.Loc == scm.LocReg && d167.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.FreeDesc(&d166)
			if d148.Loc == scm.LocStack || d148.Loc == scm.LocStackPair { ctx.EnsureDesc(&d148) }
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			var d168 scm.JITValueDesc
			if d148.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() | d167.Imm.Int())}
			} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
				ctx.BindReg(d167.Reg, &d168)
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r195 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r195, d148.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d168)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d148.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			} else if d167.Loc == scm.LocImm {
				r196 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r196, d148.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r196, int32(d167.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.W.EmitOrInt64(r196, scm.RegR11)
				}
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d168)
			} else {
				r197 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r197, d148.Reg)
				ctx.W.EmitOrInt64(r197, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d168)
			}
			if d168.Loc == scm.LocReg && d148.Loc == scm.LocReg && d168.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.EmitStoreToStack(d168, 96)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d169 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r180}
			ctx.BindReg(r180, &d169)
			ctx.BindReg(r180, &d169)
			if r159 { ctx.UnprotectReg(r160) }
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d169.Imm.Int()))))}
			} else {
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r198, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d170)
			}
			ctx.FreeDesc(&d169)
			var d171 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r199, thisptr.Reg, off)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d171)
			}
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			var d172 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() + d171.Imm.Int())}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r200 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r200, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d172)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
				ctx.BindReg(d171.Reg, &d172)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(scratch, d170.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d171.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else {
				r201 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r201, d170.Reg)
				ctx.W.EmitAddInt64(r201, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d172)
			}
			if d172.Loc == scm.LocReg && d170.Loc == scm.LocReg && d172.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.FreeDesc(&d171)
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			lbl42 := ctx.W.ReserveLabel()
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d173 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d94.Imm.Int()))))}
			} else {
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r202, d94.Reg)
				ctx.W.EmitShlRegImm8(r202, 32)
				ctx.W.EmitShrRegImm8(r202, 32)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d173)
			}
			ctx.FreeDesc(&d94)
			var d174 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r203, thisptr.Reg, off)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r203}
				ctx.BindReg(r203, &d174)
			}
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			var d175 scm.JITValueDesc
			if d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d174.Imm.Int()))))}
			} else {
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r204, d174.Reg)
				ctx.W.EmitShlRegImm8(r204, 56)
				ctx.W.EmitShrRegImm8(r204, 56)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d175)
			}
			ctx.FreeDesc(&d174)
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			var d176 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() * d175.Imm.Int())}
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(scratch, d173.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d175.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else {
				r205 := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(r205, d173.Reg)
				ctx.W.EmitImulInt64(r205, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d176)
			}
			if d176.Loc == scm.LocReg && d173.Loc == scm.LocReg && d176.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r206, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d177)
			}
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d178 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() / 64)}
			} else {
				r207 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r207, d176.Reg)
				ctx.W.EmitShrRegImm8(r207, 6)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d178)
			}
			if d178.Loc == scm.LocReg && d176.Loc == scm.LocReg && d178.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			r208 := ctx.AllocReg()
			if d178.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r208, uint64(d178.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r208, d178.Reg)
				ctx.W.EmitShlRegImm8(r208, 3)
			}
			if d177.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(r208, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r208, d177.Reg)
			}
			r209 := ctx.AllocRegExcept(r208)
			ctx.W.EmitMovRegMem(r209, r208, 0)
			ctx.FreeReg(r208)
			d179 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
			ctx.BindReg(r209, &d179)
			ctx.FreeDesc(&d178)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d180 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r210 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r210, d176.Reg)
				ctx.W.EmitAndRegImm32(r210, 63)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d180)
			}
			if d180.Loc == scm.LocReg && d176.Loc == scm.LocReg && d180.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			var d181 scm.JITValueDesc
			if d179.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d179.Imm.Int()) << uint64(d180.Imm.Int())))}
			} else if d180.Loc == scm.LocImm {
				r211 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r211, d179.Reg)
				ctx.W.EmitShlRegImm8(r211, uint8(d180.Imm.Int()))
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d181)
			} else {
				{
					shiftSrc := d179.Reg
					r212 := ctx.AllocRegExcept(d179.Reg)
					ctx.W.EmitMovRegReg(r212, d179.Reg)
					shiftSrc = r212
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d180.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d180.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d180.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d181)
				}
			}
			if d181.Loc == scm.LocReg && d179.Loc == scm.LocReg && d181.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.FreeDesc(&d180)
			var d182 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r213 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r213, thisptr.Reg, off)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
				ctx.BindReg(r213, &d182)
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			if d182.Loc == scm.LocImm {
				if d182.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
			ctx.EmitStoreToStack(d181, 104)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d182.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
			ctx.EmitStoreToStack(d181, 104)
				ctx.W.EmitJmp(lbl44)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl43)
			}
			ctx.FreeDesc(&d182)
			ctx.W.MarkLabel(lbl44)
			r214 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r214, 104)
			ctx.ProtectReg(r214)
			d183 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r214}
			ctx.BindReg(r214, &d183)
			ctx.UnprotectReg(r214)
			var d184 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r215 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r215, thisptr.Reg, off)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r215}
				ctx.BindReg(r215, &d184)
			}
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			var d185 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d184.Imm.Int()))))}
			} else {
				r216 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r216, d184.Reg)
				ctx.W.EmitShlRegImm8(r216, 56)
				ctx.W.EmitShrRegImm8(r216, 56)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d185)
			}
			ctx.FreeDesc(&d184)
			d186 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() - d185.Imm.Int())}
			} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
				r217 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r217, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d187)
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
				r218 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r218, d186.Reg)
				ctx.W.EmitSubInt64(r218, d185.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d187)
			}
			if d187.Loc == scm.LocReg && d186.Loc == scm.LocReg && d187.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d188 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d183.Imm.Int()) >> uint64(d187.Imm.Int())))}
			} else if d187.Loc == scm.LocImm {
				r219 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r219, d183.Reg)
				ctx.W.EmitShrRegImm8(r219, uint8(d187.Imm.Int()))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d188)
			} else {
				{
					shiftSrc := d183.Reg
					r220 := ctx.AllocRegExcept(d183.Reg)
					ctx.W.EmitMovRegReg(r220, d183.Reg)
					shiftSrc = r220
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d187.Reg != scm.RegRCX
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
			r221 := ctx.AllocReg()
			if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair { ctx.EnsureDesc(&d188) }
			ctx.EmitMovToReg(r221, d188)
			ctx.W.EmitJmp(lbl42)
			ctx.W.MarkLabel(lbl43)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d189 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r222 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r222, d176.Reg)
				ctx.W.EmitAndRegImm32(r222, 63)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d189)
			}
			if d189.Loc == scm.LocReg && d176.Loc == scm.LocReg && d189.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			var d190 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r223 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r223, thisptr.Reg, off)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r223}
				ctx.BindReg(r223, &d190)
			}
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			var d191 scm.JITValueDesc
			if d190.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d190.Imm.Int()))))}
			} else {
				r224 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r224, d190.Reg)
				ctx.W.EmitShlRegImm8(r224, 56)
				ctx.W.EmitShrRegImm8(r224, 56)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d191)
			}
			ctx.FreeDesc(&d190)
			if d189.Loc == scm.LocStack || d189.Loc == scm.LocStackPair { ctx.EnsureDesc(&d189) }
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			var d192 scm.JITValueDesc
			if d189.Loc == scm.LocImm && d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d189.Imm.Int() + d191.Imm.Int())}
			} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
				r225 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r225, d189.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d192)
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
				r226 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r226, d189.Reg)
				ctx.W.EmitAddInt64(r226, d191.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d192)
			}
			if d192.Loc == scm.LocReg && d189.Loc == scm.LocReg && d192.Reg == d189.Reg {
				ctx.TransferReg(d189.Reg)
				d189.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d189)
			ctx.FreeDesc(&d191)
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			var d193 scm.JITValueDesc
			if d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d192.Imm.Int()) > uint64(64))}
			} else {
				r227 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitCmpRegImm32(d192.Reg, 64)
				ctx.W.EmitSetcc(r227, scm.CcA)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r227}
				ctx.BindReg(r227, &d193)
			}
			ctx.FreeDesc(&d192)
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d193.Loc == scm.LocImm {
				if d193.Imm.Bool() {
					ctx.W.EmitJmp(lbl46)
				} else {
			ctx.EmitStoreToStack(d181, 104)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d193.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
			ctx.EmitStoreToStack(d181, 104)
				ctx.W.EmitJmp(lbl44)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d193)
			ctx.W.MarkLabel(lbl46)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d194 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() / 64)}
			} else {
				r228 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r228, d176.Reg)
				ctx.W.EmitShrRegImm8(r228, 6)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d194)
			}
			if d194.Loc == scm.LocReg && d176.Loc == scm.LocReg && d194.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			var d195 scm.JITValueDesc
			if d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d194.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(scratch, d194.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d195)
			}
			if d195.Loc == scm.LocReg && d194.Loc == scm.LocReg && d195.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			if d195.Loc == scm.LocStack || d195.Loc == scm.LocStackPair { ctx.EnsureDesc(&d195) }
			r229 := ctx.AllocReg()
			if d195.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r229, uint64(d195.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r229, d195.Reg)
				ctx.W.EmitShlRegImm8(r229, 3)
			}
			if d177.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(r229, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r229, d177.Reg)
			}
			r230 := ctx.AllocRegExcept(r229)
			ctx.W.EmitMovRegMem(r230, r229, 0)
			ctx.FreeReg(r229)
			d196 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r230}
			ctx.BindReg(r230, &d196)
			ctx.FreeDesc(&d195)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d197 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r231 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r231, d176.Reg)
				ctx.W.EmitAndRegImm32(r231, 63)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d197)
			}
			if d197.Loc == scm.LocReg && d176.Loc == scm.LocReg && d197.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			d198 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() - d197.Imm.Int())}
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r232, d198.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d199)
			} else if d198.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d198.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d197.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(scratch, d198.Reg)
				if d197.Imm.Int() >= -2147483648 && d197.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d197.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d197.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			} else {
				r233 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r233, d198.Reg)
				ctx.W.EmitSubInt64(r233, d197.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d199)
			}
			if d199.Loc == scm.LocReg && d198.Loc == scm.LocReg && d199.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair { ctx.EnsureDesc(&d196) }
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			var d200 scm.JITValueDesc
			if d196.Loc == scm.LocImm && d199.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d196.Imm.Int()) >> uint64(d199.Imm.Int())))}
			} else if d199.Loc == scm.LocImm {
				r234 := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(r234, d196.Reg)
				ctx.W.EmitShrRegImm8(r234, uint8(d199.Imm.Int()))
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d200)
			} else {
				{
					shiftSrc := d196.Reg
					r235 := ctx.AllocRegExcept(d196.Reg)
					ctx.W.EmitMovRegReg(r235, d196.Reg)
					shiftSrc = r235
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d199.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d199.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d199.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d200)
				}
			}
			if d200.Loc == scm.LocReg && d196.Loc == scm.LocReg && d200.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.FreeDesc(&d199)
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			var d201 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() | d200.Imm.Int())}
			} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
				ctx.BindReg(d200.Reg, &d201)
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r236, d181.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d201)
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d181.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d201)
			} else if d200.Loc == scm.LocImm {
				r237 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r237, d181.Reg)
				if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r237, int32(d200.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d200.Imm.Int()))
					ctx.W.EmitOrInt64(r237, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d201)
			} else {
				r238 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r238, d181.Reg)
				ctx.W.EmitOrInt64(r238, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d201)
			}
			if d201.Loc == scm.LocReg && d181.Loc == scm.LocReg && d201.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.EmitStoreToStack(d201, 104)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl42)
			ctx.W.ResolveFixups()
			d202 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r221}
			ctx.BindReg(r221, &d202)
			ctx.BindReg(r221, &d202)
			ctx.FreeDesc(&d94)
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d202.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r239, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d203)
			}
			ctx.FreeDesc(&d202)
			var d204 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r240, thisptr.Reg, off)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r240}
				ctx.BindReg(r240, &d204)
			}
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			var d205 scm.JITValueDesc
			if d203.Loc == scm.LocImm && d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d203.Imm.Int() + d204.Imm.Int())}
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				r241 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(r241, d203.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d205)
			} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d204.Reg}
				ctx.BindReg(d204.Reg, &d205)
			} else if d203.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d203.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d205)
			} else if d204.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(scratch, d203.Reg)
				if d204.Imm.Int() >= -2147483648 && d204.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d204.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d204.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d205)
			} else {
				r242 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(r242, d203.Reg)
				ctx.W.EmitAddInt64(r242, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d205)
			}
			if d205.Loc == scm.LocReg && d203.Loc == scm.LocReg && d205.Reg == d203.Reg {
				ctx.TransferReg(d203.Reg)
				d203.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d203)
			ctx.FreeDesc(&d204)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d206 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r243 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r243, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r243, 32)
				ctx.W.EmitShrRegImm8(r243, 32)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d206)
			}
			ctx.FreeDesc(&idxInt)
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() - d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r244 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r244, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
				ctx.BindReg(r244, &d207)
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d206.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(scratch, d206.Reg)
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else {
				r245 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r245, d206.Reg)
				ctx.W.EmitSubInt64(r245, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d207)
			}
			if d207.Loc == scm.LocReg && d206.Loc == scm.LocReg && d207.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d206)
			ctx.FreeDesc(&d205)
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() * d172.Imm.Int())}
			} else if d207.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d172.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(scratch, d207.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			} else {
				r246 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r246, d207.Reg)
				ctx.W.EmitImulInt64(r246, d172.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
				ctx.BindReg(r246, &d208)
			}
			if d208.Loc == scm.LocReg && d207.Loc == scm.LocReg && d208.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d207)
			ctx.FreeDesc(&d172)
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			var d209 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() + d208.Imm.Int())}
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r247 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r247, d128.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
				ctx.BindReg(r247, &d209)
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
				ctx.BindReg(d208.Reg, &d209)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(scratch, d128.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else {
				r248 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r248, d128.Reg)
				ctx.W.EmitAddInt64(r248, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d209)
			}
			if d209.Loc == scm.LocReg && d128.Loc == scm.LocReg && d209.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair { ctx.EnsureDesc(&d209) }
			var d210 scm.JITValueDesc
			if d209.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d209.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d209.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d209.Reg}
				ctx.BindReg(d209.Reg, &d210)
			}
			ctx.FreeDesc(&d209)
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			ctx.W.EmitMakeFloat(result, d210)
			if d210.Loc == scm.LocReg { ctx.FreeReg(d210.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl30)
			var d211 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r249, thisptr.Reg, off)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d211)
			}
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d211.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r250, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d212)
			}
			ctx.FreeDesc(&d211)
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			var d213 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d128.Imm.Int() == d212.Imm.Int())}
			} else if d212.Loc == scm.LocImm {
				r251 := ctx.AllocRegExcept(d128.Reg)
				if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d128.Reg, int32(d212.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
					ctx.W.EmitCmpInt64(d128.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r251, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d213)
			} else if d128.Loc == scm.LocImm {
				r252 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d212.Reg)
				ctx.W.EmitSetcc(r252, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d213)
			} else {
				r253 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitCmpInt64(d128.Reg, d212.Reg)
				ctx.W.EmitSetcc(r253, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r253}
				ctx.BindReg(r253, &d213)
			}
			ctx.FreeDesc(&d128)
			ctx.FreeDesc(&d212)
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d213.Loc == scm.LocImm {
				if d213.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d213.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl49)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl49)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d213)
			ctx.W.MarkLabel(lbl34)
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d214 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() + d133.Imm.Int())}
			} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
				r254 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(r254, d132.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d214)
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
				ctx.BindReg(d133.Reg, &d214)
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d133.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(scratch, d132.Reg)
				if d133.Imm.Int() >= -2147483648 && d133.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d133.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else {
				r255 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(r255, d132.Reg)
				ctx.W.EmitAddInt64(r255, d133.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
				ctx.BindReg(r255, &d214)
			}
			if d214.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: d214.Type, Imm: scm.NewInt(int64(uint64(d214.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d214.Reg, 32)
				ctx.W.EmitShrRegImm8(d214.Reg, 32)
			}
			if d214.Loc == scm.LocReg && d132.Loc == scm.LocReg && d214.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			var d215 scm.JITValueDesc
			if d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d214.Imm.Int() / 2)}
			} else {
				r256 := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegReg(r256, d214.Reg)
				ctx.W.EmitShrRegImm8(r256, 1)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d215)
			}
			if d215.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: d215.Type, Imm: scm.NewInt(int64(uint64(d215.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d215.Reg, 32)
				ctx.W.EmitShrRegImm8(d215.Reg, 32)
			}
			if d215.Loc == scm.LocReg && d214.Loc == scm.LocReg && d215.Reg == d214.Reg {
				ctx.TransferReg(d214.Reg)
				d214.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			d216 := d215
			if d216.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: d216.Type, Imm: scm.NewInt(int64(uint64(d216.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d216.Reg, 32)
				ctx.W.EmitShrRegImm8(d216.Reg, 32)
			}
			ctx.EmitStoreToStack(d216, 0)
			d217 := d132
			if d217.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: d217.Type, Imm: scm.NewInt(int64(uint64(d217.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d217.Reg, 32)
				ctx.W.EmitShrRegImm8(d217.Reg, 32)
			}
			ctx.EmitStoreToStack(d217, 8)
			d218 := d133
			if d218.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: d218.Type, Imm: scm.NewInt(int64(uint64(d218.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d218.Reg, 32)
				ctx.W.EmitShrRegImm8(d218.Reg, 32)
			}
			ctx.EmitStoreToStack(d218, 16)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl48)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
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
