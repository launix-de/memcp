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
				if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
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
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d2.Imm.Int()) == uint64(0))}
			} else {
				r5 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitSetcc(r5, scm.CcE)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
				ctx.BindReg(r5, &d3)
			}
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d3.Loc == scm.LocImm {
				if d3.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl2)
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d4 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r6, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d4)
			} else if d1.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r7, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d4)
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r8, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d4)
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d4.Loc == scm.LocImm {
				if d4.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
			d5 := d1
			if d5.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			d6 := d5
			if d6.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: d6.Type, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d6.Reg, 32)
				ctx.W.EmitShrRegImm8(d6.Reg, 32)
			}
			ctx.EmitStoreToStack(d6, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
			d7 := d1
			if d7.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			d8 := d7
			if d8.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: d8.Type, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d8.Reg, 32)
				ctx.W.EmitShrRegImm8(d8.Reg, 32)
			}
			ctx.EmitStoreToStack(d8, 0)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d9 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d10 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d10)
			}
			if d10.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: d10.Type, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d10.Reg, 32)
				ctx.W.EmitShrRegImm8(d10.Reg, 32)
			}
			if d10.Loc == scm.LocReg && d2.Loc == scm.LocReg && d10.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			lbl7 := ctx.W.ReserveLabel()
			d11 := d9
			if d11.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			d12 := d11
			if d12.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: d12.Type, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d12.Reg, 32)
				ctx.W.EmitShrRegImm8(d12.Reg, 32)
			}
			ctx.EmitStoreToStack(d12, 8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 16)
			d13 := d10
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			d14 := d13
			if d14.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: d14.Type, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d14.Reg, 32)
				ctx.W.EmitShrRegImm8(d14.Reg, 32)
			}
			ctx.EmitStoreToStack(d14, 24)
			ctx.W.MarkLabel(lbl7)
			d15 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d16 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d17 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			r9 := d15.Loc == scm.LocReg
			r10 := d15.Reg
			if r9 { ctx.ProtectReg(r10) }
			lbl8 := ctx.W.ReserveLabel()
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d18 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d15.Imm.Int()))))}
			} else {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r11, d15.Reg)
				ctx.W.EmitShlRegImm8(r11, 32)
				ctx.W.EmitShrRegImm8(r11, 32)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d18)
			}
			var d19 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r12, thisptr.Reg, off)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d19)
			}
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d19.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d19.Reg)
				ctx.W.EmitShlRegImm8(r13, 56)
				ctx.W.EmitShrRegImm8(r13, 56)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d20)
			}
			ctx.FreeDesc(&d19)
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() * d20.Imm.Int())}
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r14 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r14, d18.Reg)
				ctx.W.EmitImulInt64(r14, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d21)
			}
			if d21.Loc == scm.LocReg && d18.Loc == scm.LocReg && d21.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.FreeDesc(&d20)
			var d22 scm.JITValueDesc
			r15 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r15, uint64(dataPtr))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15, StackOff: int32(sliceLen)}
				ctx.BindReg(r15, &d22)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r15, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d22)
			}
			ctx.BindReg(r15, &d22)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d23 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() / 64)}
			} else {
				r16 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r16, d21.Reg)
				ctx.W.EmitShrRegImm8(r16, 6)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d23)
			}
			if d23.Loc == scm.LocReg && d21.Loc == scm.LocReg && d23.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			r17 := ctx.AllocReg()
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r17, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r17, d23.Reg)
				ctx.W.EmitShlRegImm8(r17, 3)
			}
			if d22.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
				ctx.W.EmitAddInt64(r17, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r17, d22.Reg)
			}
			r18 := ctx.AllocRegExcept(r17)
			ctx.W.EmitMovRegMem(r18, r17, 0)
			ctx.FreeReg(r17)
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			ctx.BindReg(r18, &d24)
			ctx.FreeDesc(&d23)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d25 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() % 64)}
			} else {
				r19 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r19, d21.Reg)
				ctx.W.EmitAndRegImm32(r19, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d25)
			}
			if d25.Loc == scm.LocReg && d21.Loc == scm.LocReg && d25.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			var d26 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) << uint64(d25.Imm.Int())))}
			} else if d25.Loc == scm.LocImm {
				r20 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r20, d24.Reg)
				ctx.W.EmitShlRegImm8(r20, uint8(d25.Imm.Int()))
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d26)
			} else {
				{
					shiftSrc := d24.Reg
					r21 := ctx.AllocRegExcept(d24.Reg)
					ctx.W.EmitMovRegReg(r21, d24.Reg)
					shiftSrc = r21
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d25.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d25.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d25.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d26)
				}
			}
			if d26.Loc == scm.LocReg && d24.Loc == scm.LocReg && d26.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d25)
			var d27 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r22, thisptr.Reg, off)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
				ctx.BindReg(r22, &d27)
			}
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d28 := d26
			if d28.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			ctx.EmitStoreToStack(d28, 80)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
			d29 := d26
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			ctx.EmitStoreToStack(d29, 80)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d27)
			ctx.W.MarkLabel(lbl10)
			d30 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			var d31 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r23, thisptr.Reg, off)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
				ctx.BindReg(r23, &d31)
			}
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
			} else {
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r24, d31.Reg)
				ctx.W.EmitShlRegImm8(r24, 56)
				ctx.W.EmitShrRegImm8(r24, 56)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d32)
			}
			ctx.FreeDesc(&d31)
			d33 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() - d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r25 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r25, d33.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d34)
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(scratch, d33.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			} else {
				r26 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r26, d33.Reg)
				ctx.W.EmitSubInt64(r26, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d34)
			}
			if d34.Loc == scm.LocReg && d33.Loc == scm.LocReg && d34.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d35 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d34.Imm.Int())))}
			} else if d34.Loc == scm.LocImm {
				r27 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r27, d30.Reg)
				ctx.W.EmitShrRegImm8(r27, uint8(d34.Imm.Int()))
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d35)
			} else {
				{
					shiftSrc := d30.Reg
					r28 := ctx.AllocRegExcept(d30.Reg)
					ctx.W.EmitMovRegReg(r28, d30.Reg)
					shiftSrc = r28
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d34.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d34.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d34.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d35)
				}
			}
			if d35.Loc == scm.LocReg && d30.Loc == scm.LocReg && d35.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d34)
			r29 := ctx.AllocReg()
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			ctx.EmitMovToReg(r29, d35)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl9)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d36 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() % 64)}
			} else {
				r30 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r30, d21.Reg)
				ctx.W.EmitAndRegImm32(r30, 63)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d36)
			}
			if d36.Loc == scm.LocReg && d21.Loc == scm.LocReg && d36.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r31 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r31, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
				ctx.BindReg(r31, &d37)
			}
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d37.Imm.Int()))))}
			} else {
				r32 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r32, d37.Reg)
				ctx.W.EmitShlRegImm8(r32, 56)
				ctx.W.EmitShrRegImm8(r32, 56)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d38)
			}
			ctx.FreeDesc(&d37)
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				r33 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r33, d36.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d39)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
				ctx.BindReg(d38.Reg, &d39)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r34 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r34, d36.Reg)
				ctx.W.EmitAddInt64(r34, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d39)
			}
			if d39.Loc == scm.LocReg && d36.Loc == scm.LocReg && d39.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d38)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d39.Imm.Int()) > uint64(64))}
			} else {
				r35 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpRegImm32(d39.Reg, 64)
				ctx.W.EmitSetcc(r35, scm.CcA)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
				ctx.BindReg(r35, &d40)
			}
			ctx.FreeDesc(&d39)
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d40.Loc == scm.LocImm {
				if d40.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
			d41 := d26
			if d41.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			ctx.EmitStoreToStack(d41, 80)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d40.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
			d42 := d26
			if d42.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			ctx.EmitStoreToStack(d42, 80)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d40)
			ctx.W.MarkLabel(lbl12)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d43 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() / 64)}
			} else {
				r36 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r36, d21.Reg)
				ctx.W.EmitShrRegImm8(r36, 6)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d43)
			}
			if d43.Loc == scm.LocReg && d21.Loc == scm.LocReg && d43.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(scratch, d43.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			}
			if d44.Loc == scm.LocReg && d43.Loc == scm.LocReg && d44.Reg == d43.Reg {
				ctx.TransferReg(d43.Reg)
				d43.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d43)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			r37 := ctx.AllocReg()
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			if d44.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r37, uint64(d44.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r37, d44.Reg)
				ctx.W.EmitShlRegImm8(r37, 3)
			}
			if d22.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
				ctx.W.EmitAddInt64(r37, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r37, d22.Reg)
			}
			r38 := ctx.AllocRegExcept(r37)
			ctx.W.EmitMovRegMem(r38, r37, 0)
			ctx.FreeReg(r37)
			d45 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.BindReg(r38, &d45)
			ctx.FreeDesc(&d44)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d46 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() % 64)}
			} else {
				r39 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r39, d21.Reg)
				ctx.W.EmitAndRegImm32(r39, 63)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d46)
			}
			if d46.Loc == scm.LocReg && d21.Loc == scm.LocReg && d46.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			d47 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() - d46.Imm.Int())}
			} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r40, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d48)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d46.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d46.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else {
				r41 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r41, d47.Reg)
				ctx.W.EmitSubInt64(r41, d46.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d48)
			}
			if d48.Loc == scm.LocReg && d47.Loc == scm.LocReg && d48.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d49 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d45.Imm.Int()) >> uint64(d48.Imm.Int())))}
			} else if d48.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(r42, d45.Reg)
				ctx.W.EmitShrRegImm8(r42, uint8(d48.Imm.Int()))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d49)
			} else {
				{
					shiftSrc := d45.Reg
					r43 := ctx.AllocRegExcept(d45.Reg)
					ctx.W.EmitMovRegReg(r43, d45.Reg)
					shiftSrc = r43
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d48.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d48.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d48.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d49)
				}
			}
			if d49.Loc == scm.LocReg && d45.Loc == scm.LocReg && d49.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.FreeDesc(&d48)
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			var d50 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() | d49.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
				ctx.BindReg(d49.Reg, &d50)
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r44, d26.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d50)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d49.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r45, d26.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r45, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitOrInt64(r45, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d50)
			} else {
				r46 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r46, d26.Reg)
				ctx.W.EmitOrInt64(r46, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d50)
			}
			if d50.Loc == scm.LocReg && d26.Loc == scm.LocReg && d50.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			d51 := d50
			if d51.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitStoreToStack(d51, 80)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl8)
			ctx.W.ResolveFixups()
			d52 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			ctx.BindReg(r29, &d52)
			ctx.BindReg(r29, &d52)
			if r9 { ctx.UnprotectReg(r10) }
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d52.Imm.Int()))))}
			} else {
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r47, d52.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d53)
			}
			ctx.FreeDesc(&d52)
			var d54 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r48, thisptr.Reg, off)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d54)
			}
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			var d55 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() + d54.Imm.Int())}
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				r49 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r49, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d55)
			} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
				ctx.BindReg(d54.Reg, &d55)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else if d54.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(scratch, d53.Reg)
				if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d54.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else {
				r50 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r50, d53.Reg)
				ctx.W.EmitAddInt64(r50, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d55)
			}
			if d55.Loc == scm.LocReg && d53.Loc == scm.LocReg && d55.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.FreeDesc(&d54)
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d55.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d55.Reg)
				ctx.W.EmitShlRegImm8(r51, 32)
				ctx.W.EmitShrRegImm8(r51, 32)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d56)
			}
			ctx.FreeDesc(&d55)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if d56.Loc == scm.LocStack || d56.Loc == scm.LocStackPair { ctx.EnsureDesc(&d56) }
			var d57 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d56.Imm.Int()))}
			} else if d56.Loc == scm.LocImm {
				r52 := ctx.AllocRegExcept(idxInt.Reg)
				if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d56.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r52, scm.CcB)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r52}
				ctx.BindReg(r52, &d57)
			} else if idxInt.Loc == scm.LocImm {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d56.Reg)
				ctx.W.EmitSetcc(r53, scm.CcB)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d57)
			} else {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d56.Reg)
				ctx.W.EmitSetcc(r54, scm.CcB)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d57)
			}
			ctx.FreeDesc(&d56)
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d57.Loc == scm.LocImm {
				if d57.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d57.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d57)
			ctx.W.MarkLabel(lbl4)
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d58 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			}
			if d58.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: d58.Type, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d58.Reg, 32)
				ctx.W.EmitShrRegImm8(d58.Reg, 32)
			}
			if d58.Loc == scm.LocReg && d2.Loc == scm.LocReg && d58.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d59 := d58
			if d59.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			d60 := d59
			if d60.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: d60.Type, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d60.Reg, 32)
				ctx.W.EmitShrRegImm8(d60.Reg, 32)
			}
			ctx.EmitStoreToStack(d60, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl15)
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d61 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			}
			if d61.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: d61.Type, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d61.Reg, 32)
				ctx.W.EmitShrRegImm8(d61.Reg, 32)
			}
			if d61.Loc == scm.LocReg && d15.Loc == scm.LocReg && d61.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d61.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d61.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d61.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d61.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcAE)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d62)
			} else if d61.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r56, scm.CcAE)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d62)
			} else {
				r57 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitCmpInt64(d61.Reg, d2.Reg)
				ctx.W.EmitSetcc(r57, scm.CcAE)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d62)
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d62.Loc == scm.LocImm {
				if d62.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
			d63 := d61
			if d63.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d63.Loc == scm.LocStack || d63.Loc == scm.LocStackPair { ctx.EnsureDesc(&d63) }
			d64 := d63
			if d64.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: d64.Type, Imm: scm.NewInt(int64(uint64(d64.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d64.Reg, 32)
				ctx.W.EmitShrRegImm8(d64.Reg, 32)
			}
			ctx.EmitStoreToStack(d64, 40)
			d65 := d15
			if d65.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			d66 := d65
			if d66.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: d66.Type, Imm: scm.NewInt(int64(uint64(d66.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d66.Reg, 32)
				ctx.W.EmitShrRegImm8(d66.Reg, 32)
			}
			ctx.EmitStoreToStack(d66, 48)
			d67 := d17
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			d68 := d67
			if d68.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: d68.Type, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d68.Reg, 32)
				ctx.W.EmitShrRegImm8(d68.Reg, 32)
			}
			ctx.EmitStoreToStack(d68, 56)
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d62.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
			d69 := d61
			if d69.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			d70 := d69
			if d70.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: d70.Type, Imm: scm.NewInt(int64(uint64(d70.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d70.Reg, 32)
				ctx.W.EmitShrRegImm8(d70.Reg, 32)
			}
			ctx.EmitStoreToStack(d70, 40)
			d71 := d15
			if d71.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			d72 := d71
			if d72.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: d72.Type, Imm: scm.NewInt(int64(uint64(d72.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d72.Reg, 32)
				ctx.W.EmitShrRegImm8(d72.Reg, 32)
			}
			ctx.EmitStoreToStack(d72, 48)
			d73 := d17
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d73.Loc == scm.LocStack || d73.Loc == scm.LocStackPair { ctx.EnsureDesc(&d73) }
			d74 := d73
			if d74.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: d74.Type, Imm: scm.NewInt(int64(uint64(d74.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d74.Reg, 32)
				ctx.W.EmitShrRegImm8(d74.Reg, 32)
			}
			ctx.EmitStoreToStack(d74, 56)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d62)
			ctx.W.MarkLabel(lbl14)
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d75 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d15.Imm.Int()) == uint64(0))}
			} else {
				r58 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitCmpRegImm32(d15.Reg, 0)
				ctx.W.EmitSetcc(r58, scm.CcE)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d75)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d75.Loc == scm.LocImm {
				if d75.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d75.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d75)
			ctx.W.MarkLabel(lbl18)
			d76 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d78 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d77.Imm.Int()) == uint64(d78.Imm.Int()))}
			} else if d78.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d77.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d77.Reg, int32(d78.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.W.EmitCmpInt64(d77.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r59, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d79)
			} else if d77.Loc == scm.LocImm {
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d78.Reg)
				ctx.W.EmitSetcc(r60, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d79)
			} else {
				r61 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitCmpInt64(d77.Reg, d78.Reg)
				ctx.W.EmitSetcc(r61, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d79)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d79.Loc == scm.LocImm {
				if d79.Imm.Bool() {
			d80 := d77
			if d80.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair { ctx.EnsureDesc(&d80) }
			d81 := d80
			if d81.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: d81.Type, Imm: scm.NewInt(int64(uint64(d81.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d81.Reg, 32)
				ctx.W.EmitShrRegImm8(d81.Reg, 32)
			}
			ctx.EmitStoreToStack(d81, 32)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
			d82 := d77
			if d82.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			d83 := d82
			if d83.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: d83.Type, Imm: scm.NewInt(int64(uint64(d83.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d83.Reg, 32)
				ctx.W.EmitShrRegImm8(d83.Reg, 32)
			}
			ctx.EmitStoreToStack(d83, 32)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d79)
			ctx.W.MarkLabel(lbl17)
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d84 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			}
			if d84.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: d84.Type, Imm: scm.NewInt(int64(uint64(d84.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d84.Reg, 32)
				ctx.W.EmitShrRegImm8(d84.Reg, 32)
			}
			if d84.Loc == scm.LocReg && d2.Loc == scm.LocReg && d84.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d85 := d84
			if d85.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			d86 := d85
			if d86.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: d86.Type, Imm: scm.NewInt(int64(uint64(d86.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d86.Reg, 32)
				ctx.W.EmitShrRegImm8(d86.Reg, 32)
			}
			ctx.EmitStoreToStack(d86, 40)
			d87 := d15
			if d87.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			d88 := d87
			if d88.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: d88.Type, Imm: scm.NewInt(int64(uint64(d88.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d88.Reg, 32)
				ctx.W.EmitShrRegImm8(d88.Reg, 32)
			}
			ctx.EmitStoreToStack(d88, 48)
			d89 := d17
			if d89.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			d90 := d89
			if d90.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: d90.Type, Imm: scm.NewInt(int64(uint64(d90.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d90.Reg, 32)
				ctx.W.EmitShrRegImm8(d90.Reg, 32)
			}
			ctx.EmitStoreToStack(d90, 56)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl21)
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d91 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d91)
			}
			if d91.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: d91.Type, Imm: scm.NewInt(int64(uint64(d91.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d91.Reg, 32)
				ctx.W.EmitShrRegImm8(d91.Reg, 32)
			}
			if d91.Loc == scm.LocReg && d15.Loc == scm.LocReg && d91.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d92 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(scratch, d15.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			}
			if d92.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: d92.Type, Imm: scm.NewInt(int64(uint64(d92.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d92.Reg, 32)
				ctx.W.EmitShrRegImm8(d92.Reg, 32)
			}
			if d92.Loc == scm.LocReg && d15.Loc == scm.LocReg && d92.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			d93 := d92
			if d93.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair { ctx.EnsureDesc(&d93) }
			d94 := d93
			if d94.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: d94.Type, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d94.Reg, 32)
				ctx.W.EmitShrRegImm8(d94.Reg, 32)
			}
			ctx.EmitStoreToStack(d94, 40)
			d95 := d16
			if d95.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			d96 := d95
			if d96.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: d96.Type, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d96.Reg, 32)
				ctx.W.EmitShrRegImm8(d96.Reg, 32)
			}
			ctx.EmitStoreToStack(d96, 48)
			d97 := d91
			if d97.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			d98 := d97
			if d98.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: d98.Type, Imm: scm.NewInt(int64(uint64(d98.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d98.Reg, 32)
				ctx.W.EmitShrRegImm8(d98.Reg, 32)
			}
			ctx.EmitStoreToStack(d98, 56)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl20)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.MarkLabel(lbl23)
			d99 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d100 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d99.Imm.Int()))))}
			} else {
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r62, d99.Reg)
				ctx.W.EmitShlRegImm8(r62, 32)
				ctx.W.EmitShrRegImm8(r62, 32)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
				ctx.BindReg(r62, &d100)
			}
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d100.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d100.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d100.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d100.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d100.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d100.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d100.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d100.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d100)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			r63 := d99.Loc == scm.LocReg
			r64 := d99.Reg
			if r63 { ctx.ProtectReg(r64) }
			lbl26 := ctx.W.ReserveLabel()
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d101 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d99.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d99.Reg)
				ctx.W.EmitShlRegImm8(r65, 32)
				ctx.W.EmitShrRegImm8(r65, 32)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d101)
			}
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r66, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
				ctx.BindReg(r66, &d102)
			}
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			var d103 scm.JITValueDesc
			if d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d102.Imm.Int()))))}
			} else {
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r67, d102.Reg)
				ctx.W.EmitShlRegImm8(r67, 56)
				ctx.W.EmitShrRegImm8(r67, 56)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d103)
			}
			ctx.FreeDesc(&d102)
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			var d104 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() * d103.Imm.Int())}
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d101.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d103.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d104)
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(scratch, d101.Reg)
				if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d103.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d104)
			} else {
				r68 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r68, d101.Reg)
				ctx.W.EmitImulInt64(r68, d103.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d104)
			}
			if d104.Loc == scm.LocReg && d101.Loc == scm.LocReg && d104.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d103)
			var d105 scm.JITValueDesc
			r69 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r69, uint64(dataPtr))
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69, StackOff: int32(sliceLen)}
				ctx.BindReg(r69, &d105)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r69, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
				ctx.BindReg(r69, &d105)
			}
			ctx.BindReg(r69, &d105)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d106 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() / 64)}
			} else {
				r70 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r70, d104.Reg)
				ctx.W.EmitShrRegImm8(r70, 6)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d106)
			}
			if d106.Loc == scm.LocReg && d104.Loc == scm.LocReg && d106.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			r71 := ctx.AllocReg()
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			if d106.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r71, uint64(d106.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r71, d106.Reg)
				ctx.W.EmitShlRegImm8(r71, 3)
			}
			if d105.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d105.Imm.Int()))
				ctx.W.EmitAddInt64(r71, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r71, d105.Reg)
			}
			r72 := ctx.AllocRegExcept(r71)
			ctx.W.EmitMovRegMem(r72, r71, 0)
			ctx.FreeReg(r71)
			d107 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			ctx.BindReg(r72, &d107)
			ctx.FreeDesc(&d106)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d108 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() % 64)}
			} else {
				r73 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r73, d104.Reg)
				ctx.W.EmitAndRegImm32(r73, 63)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d108)
			}
			if d108.Loc == scm.LocReg && d104.Loc == scm.LocReg && d108.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair { ctx.EnsureDesc(&d108) }
			var d109 scm.JITValueDesc
			if d107.Loc == scm.LocImm && d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d107.Imm.Int()) << uint64(d108.Imm.Int())))}
			} else if d108.Loc == scm.LocImm {
				r74 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r74, d107.Reg)
				ctx.W.EmitShlRegImm8(r74, uint8(d108.Imm.Int()))
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d109)
			} else {
				{
					shiftSrc := d107.Reg
					r75 := ctx.AllocRegExcept(d107.Reg)
					ctx.W.EmitMovRegReg(r75, d107.Reg)
					shiftSrc = r75
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d108.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d108.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d108.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d109)
				}
			}
			if d109.Loc == scm.LocReg && d107.Loc == scm.LocReg && d109.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r76, thisptr.Reg, off)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
				ctx.BindReg(r76, &d110)
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d110.Loc == scm.LocImm {
				if d110.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
			d111 := d109
			if d111.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			ctx.EmitStoreToStack(d111, 88)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d110.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
			d112 := d109
			if d112.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			ctx.EmitStoreToStack(d112, 88)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d110)
			ctx.W.MarkLabel(lbl28)
			d113 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d114 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r77 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r77, thisptr.Reg, off)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
				ctx.BindReg(r77, &d114)
			}
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			var d115 scm.JITValueDesc
			if d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d114.Imm.Int()))))}
			} else {
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r78, d114.Reg)
				ctx.W.EmitShlRegImm8(r78, 56)
				ctx.W.EmitShrRegImm8(r78, 56)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d115)
			}
			ctx.FreeDesc(&d114)
			d116 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d115.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() - d115.Imm.Int())}
			} else if d115.Loc == scm.LocImm && d115.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r79, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d117)
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d115.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			} else if d115.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(scratch, d116.Reg)
				if d115.Imm.Int() >= -2147483648 && d115.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d115.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d115.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			} else {
				r80 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r80, d116.Reg)
				ctx.W.EmitSubInt64(r80, d115.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d117)
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			if d113.Loc == scm.LocStack || d113.Loc == scm.LocStackPair { ctx.EnsureDesc(&d113) }
			if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair { ctx.EnsureDesc(&d117) }
			var d118 scm.JITValueDesc
			if d113.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d113.Imm.Int()) >> uint64(d117.Imm.Int())))}
			} else if d117.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r81, d113.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d117.Imm.Int()))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d118)
			} else {
				{
					shiftSrc := d113.Reg
					r82 := ctx.AllocRegExcept(d113.Reg)
					ctx.W.EmitMovRegReg(r82, d113.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d117.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d117.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d117.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d118)
				}
			}
			if d118.Loc == scm.LocReg && d113.Loc == scm.LocReg && d118.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			ctx.FreeDesc(&d117)
			r83 := ctx.AllocReg()
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			ctx.EmitMovToReg(r83, d118)
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl27)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d119 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() % 64)}
			} else {
				r84 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r84, d104.Reg)
				ctx.W.EmitAndRegImm32(r84, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d119)
			}
			if d119.Loc == scm.LocReg && d104.Loc == scm.LocReg && d119.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r85 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r85, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
				ctx.BindReg(r85, &d120)
			}
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d120.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d120.Reg)
				ctx.W.EmitShlRegImm8(r86, 56)
				ctx.W.EmitShrRegImm8(r86, 56)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d121)
			}
			ctx.FreeDesc(&d120)
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + d121.Imm.Int())}
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				r87 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r87, d119.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d122)
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d121.Reg}
				ctx.BindReg(d121.Reg, &d122)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r88 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r88, d119.Reg)
				ctx.W.EmitAddInt64(r88, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d122)
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d121)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d122.Imm.Int()) > uint64(64))}
			} else {
				r89 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitCmpRegImm32(d122.Reg, 64)
				ctx.W.EmitSetcc(r89, scm.CcA)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r89}
				ctx.BindReg(r89, &d123)
			}
			ctx.FreeDesc(&d122)
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d123.Loc == scm.LocImm {
				if d123.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
			d124 := d109
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			ctx.EmitStoreToStack(d124, 88)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d123.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
			d125 := d109
			if d125.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			ctx.EmitStoreToStack(d125, 88)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d123)
			ctx.W.MarkLabel(lbl30)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d126 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() / 64)}
			} else {
				r90 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r90, d104.Reg)
				ctx.W.EmitShrRegImm8(r90, 6)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d126)
			}
			if d126.Loc == scm.LocReg && d104.Loc == scm.LocReg && d126.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			var d127 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(scratch, d126.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			}
			if d127.Loc == scm.LocReg && d126.Loc == scm.LocReg && d127.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			r91 := ctx.AllocReg()
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			if d127.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r91, uint64(d127.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r91, d127.Reg)
				ctx.W.EmitShlRegImm8(r91, 3)
			}
			if d105.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d105.Imm.Int()))
				ctx.W.EmitAddInt64(r91, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r91, d105.Reg)
			}
			r92 := ctx.AllocRegExcept(r91)
			ctx.W.EmitMovRegMem(r92, r91, 0)
			ctx.FreeReg(r91)
			d128 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			ctx.BindReg(r92, &d128)
			ctx.FreeDesc(&d127)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d129 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() % 64)}
			} else {
				r93 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r93, d104.Reg)
				ctx.W.EmitAndRegImm32(r93, 63)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
				ctx.BindReg(r93, &d129)
			}
			if d129.Loc == scm.LocReg && d104.Loc == scm.LocReg && d129.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			d130 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair { ctx.EnsureDesc(&d129) }
			var d131 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() - d129.Imm.Int())}
			} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
				r94 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r94, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d131)
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d129.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(scratch, d130.Reg)
				if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d129.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else {
				r95 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r95, d130.Reg)
				ctx.W.EmitSubInt64(r95, d129.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d131)
			}
			if d131.Loc == scm.LocReg && d130.Loc == scm.LocReg && d131.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d129)
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			if d131.Loc == scm.LocStack || d131.Loc == scm.LocStackPair { ctx.EnsureDesc(&d131) }
			var d132 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d128.Imm.Int()) >> uint64(d131.Imm.Int())))}
			} else if d131.Loc == scm.LocImm {
				r96 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r96, d128.Reg)
				ctx.W.EmitShrRegImm8(r96, uint8(d131.Imm.Int()))
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d132)
			} else {
				{
					shiftSrc := d128.Reg
					r97 := ctx.AllocRegExcept(d128.Reg)
					ctx.W.EmitMovRegReg(r97, d128.Reg)
					shiftSrc = r97
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d131.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d131.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d131.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d132)
				}
			}
			if d132.Loc == scm.LocReg && d128.Loc == scm.LocReg && d132.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.FreeDesc(&d131)
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			var d133 scm.JITValueDesc
			if d109.Loc == scm.LocImm && d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() | d132.Imm.Int())}
			} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
				ctx.BindReg(d132.Reg, &d133)
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				r98 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r98, d109.Reg)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d133)
			} else if d109.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d109.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d132.Reg)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d133)
			} else if d132.Loc == scm.LocImm {
				r99 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r99, d109.Reg)
				if d132.Imm.Int() >= -2147483648 && d132.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r99, int32(d132.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d132.Imm.Int()))
					ctx.W.EmitOrInt64(r99, scm.RegR11)
				}
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d133)
			} else {
				r100 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(r100, d109.Reg)
				ctx.W.EmitOrInt64(r100, d132.Reg)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d133)
			}
			if d133.Loc == scm.LocReg && d109.Loc == scm.LocReg && d133.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			d134 := d133
			if d134.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d134.Loc == scm.LocStack || d134.Loc == scm.LocStackPair { ctx.EnsureDesc(&d134) }
			ctx.EmitStoreToStack(d134, 88)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl26)
			ctx.W.ResolveFixups()
			d135 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r83}
			ctx.BindReg(r83, &d135)
			ctx.BindReg(r83, &d135)
			if r63 { ctx.UnprotectReg(r64) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			var d136 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d135.Imm.Int()))))}
			} else {
				r101 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r101, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d136)
			}
			ctx.FreeDesc(&d135)
			var d137 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r102, thisptr.Reg, off)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r102}
				ctx.BindReg(r102, &d137)
			}
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair { ctx.EnsureDesc(&d137) }
			var d138 scm.JITValueDesc
			if d136.Loc == scm.LocImm && d137.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() + d137.Imm.Int())}
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				r103 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r103, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d138)
			} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d137.Reg}
				ctx.BindReg(d137.Reg, &d138)
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d137.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d138)
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(scratch, d136.Reg)
				if d137.Imm.Int() >= -2147483648 && d137.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d137.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d138)
			} else {
				r104 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r104, d136.Reg)
				ctx.W.EmitAddInt64(r104, d137.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d138)
			}
			if d138.Loc == scm.LocReg && d136.Loc == scm.LocReg && d138.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			ctx.FreeDesc(&d137)
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d139)
			}
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d139.Loc == scm.LocImm {
				if d139.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d139.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d139)
			ctx.W.MarkLabel(lbl24)
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			r106 := d76.Loc == scm.LocReg
			r107 := d76.Reg
			if r106 { ctx.ProtectReg(r107) }
			lbl35 := ctx.W.ReserveLabel()
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			var d140 scm.JITValueDesc
			if d76.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d76.Imm.Int()))))}
			} else {
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r108, d76.Reg)
				ctx.W.EmitShlRegImm8(r108, 32)
				ctx.W.EmitShrRegImm8(r108, 32)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d140)
			}
			var d141 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r109, thisptr.Reg, off)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
				ctx.BindReg(r109, &d141)
			}
			if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair { ctx.EnsureDesc(&d141) }
			if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair { ctx.EnsureDesc(&d141) }
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d141.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d141.Reg)
				ctx.W.EmitShlRegImm8(r110, 56)
				ctx.W.EmitShrRegImm8(r110, 56)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d142)
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
				r111 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r111, d140.Reg)
				ctx.W.EmitImulInt64(r111, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d143)
			}
			if d143.Loc == scm.LocReg && d140.Loc == scm.LocReg && d143.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			ctx.FreeDesc(&d142)
			var d144 scm.JITValueDesc
			r112 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r112, uint64(dataPtr))
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112, StackOff: int32(sliceLen)}
				ctx.BindReg(r112, &d144)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r112, thisptr.Reg, off)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
				ctx.BindReg(r112, &d144)
			}
			ctx.BindReg(r112, &d144)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d145 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() / 64)}
			} else {
				r113 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r113, d143.Reg)
				ctx.W.EmitShrRegImm8(r113, 6)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d145)
			}
			if d145.Loc == scm.LocReg && d143.Loc == scm.LocReg && d145.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			r114 := ctx.AllocReg()
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r114, uint64(d145.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r114, d145.Reg)
				ctx.W.EmitShlRegImm8(r114, 3)
			}
			if d144.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(r114, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r114, d144.Reg)
			}
			r115 := ctx.AllocRegExcept(r114)
			ctx.W.EmitMovRegMem(r115, r114, 0)
			ctx.FreeReg(r114)
			d146 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
			ctx.BindReg(r115, &d146)
			ctx.FreeDesc(&d145)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d147 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r116 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r116, d143.Reg)
				ctx.W.EmitAndRegImm32(r116, 63)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d147)
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
				r117 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r117, d146.Reg)
				ctx.W.EmitShlRegImm8(r117, uint8(d147.Imm.Int()))
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d148)
			} else {
				{
					shiftSrc := d146.Reg
					r118 := ctx.AllocRegExcept(d146.Reg)
					ctx.W.EmitMovRegReg(r118, d146.Reg)
					shiftSrc = r118
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r119 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r119, thisptr.Reg, off)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r119}
				ctx.BindReg(r119, &d149)
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			if d149.Loc == scm.LocImm {
				if d149.Imm.Bool() {
					ctx.W.EmitJmp(lbl36)
				} else {
			d150 := d148
			if d150.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			ctx.EmitStoreToStack(d150, 96)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d149.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl38)
			d151 := d148
			if d151.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d151.Loc == scm.LocStack || d151.Loc == scm.LocStackPair { ctx.EnsureDesc(&d151) }
			ctx.EmitStoreToStack(d151, 96)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d149)
			ctx.W.MarkLabel(lbl37)
			d152 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d153 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r120 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r120, thisptr.Reg, off)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r120}
				ctx.BindReg(r120, &d153)
			}
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d153.Imm.Int()))))}
			} else {
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r121, d153.Reg)
				ctx.W.EmitShlRegImm8(r121, 56)
				ctx.W.EmitShrRegImm8(r121, 56)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d154)
			}
			ctx.FreeDesc(&d153)
			d155 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() - d154.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				r122 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r122, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d156)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(scratch, d155.Reg)
				if d154.Imm.Int() >= -2147483648 && d154.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d154.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d154.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else {
				r123 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r123, d155.Reg)
				ctx.W.EmitSubInt64(r123, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d156)
			}
			if d156.Loc == scm.LocReg && d155.Loc == scm.LocReg && d156.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			if d156.Loc == scm.LocStack || d156.Loc == scm.LocStackPair { ctx.EnsureDesc(&d156) }
			var d157 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d152.Imm.Int()) >> uint64(d156.Imm.Int())))}
			} else if d156.Loc == scm.LocImm {
				r124 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r124, d152.Reg)
				ctx.W.EmitShrRegImm8(r124, uint8(d156.Imm.Int()))
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d157)
			} else {
				{
					shiftSrc := d152.Reg
					r125 := ctx.AllocRegExcept(d152.Reg)
					ctx.W.EmitMovRegReg(r125, d152.Reg)
					shiftSrc = r125
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d156.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d156.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d156.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d157)
				}
			}
			if d157.Loc == scm.LocReg && d152.Loc == scm.LocReg && d157.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.FreeDesc(&d156)
			r126 := ctx.AllocReg()
			if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair { ctx.EnsureDesc(&d157) }
			ctx.EmitMovToReg(r126, d157)
			ctx.W.EmitJmp(lbl35)
			ctx.W.MarkLabel(lbl36)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d158 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r127 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r127, d143.Reg)
				ctx.W.EmitAndRegImm32(r127, 63)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d158)
			}
			if d158.Loc == scm.LocReg && d143.Loc == scm.LocReg && d158.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			var d159 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r128 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r128, thisptr.Reg, off)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
				ctx.BindReg(r128, &d159)
			}
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d159.Imm.Int()))))}
			} else {
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r129, d159.Reg)
				ctx.W.EmitShlRegImm8(r129, 56)
				ctx.W.EmitShrRegImm8(r129, 56)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d160)
			}
			ctx.FreeDesc(&d159)
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			var d161 scm.JITValueDesc
			if d158.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() + d160.Imm.Int())}
			} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
				r130 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r130, d158.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d161)
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
				ctx.BindReg(d160.Reg, &d161)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d158.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(scratch, d158.Reg)
				if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d160.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else {
				r131 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r131, d158.Reg)
				ctx.W.EmitAddInt64(r131, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d161)
			}
			if d161.Loc == scm.LocReg && d158.Loc == scm.LocReg && d161.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.FreeDesc(&d160)
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d161.Imm.Int()) > uint64(64))}
			} else {
				r132 := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitCmpRegImm32(d161.Reg, 64)
				ctx.W.EmitSetcc(r132, scm.CcA)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r132}
				ctx.BindReg(r132, &d162)
			}
			ctx.FreeDesc(&d161)
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d162.Loc == scm.LocImm {
				if d162.Imm.Bool() {
					ctx.W.EmitJmp(lbl39)
				} else {
			d163 := d148
			if d163.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			ctx.EmitStoreToStack(d163, 96)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d162.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
			d164 := d148
			if d164.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			ctx.EmitStoreToStack(d164, 96)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl39)
			}
			ctx.FreeDesc(&d162)
			ctx.W.MarkLabel(lbl39)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d165 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() / 64)}
			} else {
				r133 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r133, d143.Reg)
				ctx.W.EmitShrRegImm8(r133, 6)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d165)
			}
			if d165.Loc == scm.LocReg && d143.Loc == scm.LocReg && d165.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(scratch, d165.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d165)
			if d166.Loc == scm.LocStack || d166.Loc == scm.LocStackPair { ctx.EnsureDesc(&d166) }
			r134 := ctx.AllocReg()
			if d166.Loc == scm.LocStack || d166.Loc == scm.LocStackPair { ctx.EnsureDesc(&d166) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			if d166.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r134, uint64(d166.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r134, d166.Reg)
				ctx.W.EmitShlRegImm8(r134, 3)
			}
			if d144.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(r134, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r134, d144.Reg)
			}
			r135 := ctx.AllocRegExcept(r134)
			ctx.W.EmitMovRegMem(r135, r134, 0)
			ctx.FreeReg(r134)
			d167 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r135}
			ctx.BindReg(r135, &d167)
			ctx.FreeDesc(&d166)
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d168 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() % 64)}
			} else {
				r136 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r136, d143.Reg)
				ctx.W.EmitAndRegImm32(r136, 63)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d168)
			}
			if d168.Loc == scm.LocReg && d143.Loc == scm.LocReg && d168.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d143)
			d169 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm && d168.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() - d168.Imm.Int())}
			} else if d168.Loc == scm.LocImm && d168.Imm.Int() == 0 {
				r137 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r137, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d170)
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d169.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d168.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(scratch, d169.Reg)
				if d168.Imm.Int() >= -2147483648 && d168.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d168.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d168.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else {
				r138 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r138, d169.Reg)
				ctx.W.EmitSubInt64(r138, d168.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d170)
			}
			if d170.Loc == scm.LocReg && d169.Loc == scm.LocReg && d170.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d171 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d167.Imm.Int()) >> uint64(d170.Imm.Int())))}
			} else if d170.Loc == scm.LocImm {
				r139 := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(r139, d167.Reg)
				ctx.W.EmitShrRegImm8(r139, uint8(d170.Imm.Int()))
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d171)
			} else {
				{
					shiftSrc := d167.Reg
					r140 := ctx.AllocRegExcept(d167.Reg)
					ctx.W.EmitMovRegReg(r140, d167.Reg)
					shiftSrc = r140
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d170.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d170.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d170.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d171)
				}
			}
			if d171.Loc == scm.LocReg && d167.Loc == scm.LocReg && d171.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.FreeDesc(&d170)
			if d148.Loc == scm.LocStack || d148.Loc == scm.LocStackPair { ctx.EnsureDesc(&d148) }
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			var d172 scm.JITValueDesc
			if d148.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() | d171.Imm.Int())}
			} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
				ctx.BindReg(d171.Reg, &d172)
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r141 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r141, d148.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d172)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d148.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else if d171.Loc == scm.LocImm {
				r142 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r142, d148.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r142, int32(d171.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
					ctx.W.EmitOrInt64(r142, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d172)
			} else {
				r143 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r143, d148.Reg)
				ctx.W.EmitOrInt64(r143, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d172)
			}
			if d172.Loc == scm.LocReg && d148.Loc == scm.LocReg && d172.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			d173 := d172
			if d173.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			ctx.EmitStoreToStack(d173, 96)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl35)
			ctx.W.ResolveFixups()
			d174 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.BindReg(r126, &d174)
			ctx.BindReg(r126, &d174)
			if r106 { ctx.UnprotectReg(r107) }
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			var d175 scm.JITValueDesc
			if d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d174.Imm.Int()))))}
			} else {
				r144 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r144, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d175)
			}
			ctx.FreeDesc(&d174)
			var d176 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r145, thisptr.Reg, off)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
				ctx.BindReg(r145, &d176)
			}
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d177 scm.JITValueDesc
			if d175.Loc == scm.LocImm && d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() + d176.Imm.Int())}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				r146 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r146, d175.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d177)
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
				ctx.BindReg(d176.Reg, &d177)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d175.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d177)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(scratch, d175.Reg)
				if d176.Imm.Int() >= -2147483648 && d176.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d176.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d177)
			} else {
				r147 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r147, d175.Reg)
				ctx.W.EmitAddInt64(r147, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d177)
			}
			if d177.Loc == scm.LocReg && d175.Loc == scm.LocReg && d177.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			ctx.FreeDesc(&d176)
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d177.Imm.Int()))))}
			} else {
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r148, d177.Reg)
				ctx.W.EmitShlRegImm8(r148, 32)
				ctx.W.EmitShrRegImm8(r148, 32)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d178)
			}
			ctx.FreeDesc(&d177)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			var d179 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d178.Imm.Int()))}
			} else if d178.Loc == scm.LocImm {
				r149 := ctx.AllocRegExcept(idxInt.Reg)
				if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d178.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r149, scm.CcB)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r149}
				ctx.BindReg(r149, &d179)
			} else if idxInt.Loc == scm.LocImm {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d178.Reg)
				ctx.W.EmitSetcc(r150, scm.CcB)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r150}
				ctx.BindReg(r150, &d179)
			} else {
				r151 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d178.Reg)
				ctx.W.EmitSetcc(r151, scm.CcB)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d179)
			}
			ctx.FreeDesc(&d178)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d179.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d179)
			ctx.W.MarkLabel(lbl33)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			r152 := d99.Loc == scm.LocReg
			r153 := d99.Reg
			if r152 { ctx.ProtectReg(r153) }
			lbl44 := ctx.W.ReserveLabel()
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d180 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d99.Imm.Int()))))}
			} else {
				r154 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r154, d99.Reg)
				ctx.W.EmitShlRegImm8(r154, 32)
				ctx.W.EmitShrRegImm8(r154, 32)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d180)
			}
			var d181 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r155 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r155, thisptr.Reg, off)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
				ctx.BindReg(r155, &d181)
			}
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d181.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, d181.Reg)
				ctx.W.EmitShlRegImm8(r156, 56)
				ctx.W.EmitShrRegImm8(r156, 56)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d182)
			}
			ctx.FreeDesc(&d181)
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			var d183 scm.JITValueDesc
			if d180.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() * d182.Imm.Int())}
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(scratch, d180.Reg)
				if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d182.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else {
				r157 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r157, d180.Reg)
				ctx.W.EmitImulInt64(r157, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d183)
			}
			if d183.Loc == scm.LocReg && d180.Loc == scm.LocReg && d183.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			ctx.FreeDesc(&d182)
			var d184 scm.JITValueDesc
			r158 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r158, uint64(dataPtr))
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158, StackOff: int32(sliceLen)}
				ctx.BindReg(r158, &d184)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r158, thisptr.Reg, off)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d184)
			}
			ctx.BindReg(r158, &d184)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d185 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() / 64)}
			} else {
				r159 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r159, d183.Reg)
				ctx.W.EmitShrRegImm8(r159, 6)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d185)
			}
			if d185.Loc == scm.LocReg && d183.Loc == scm.LocReg && d185.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			r160 := ctx.AllocReg()
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			if d185.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r160, uint64(d185.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r160, d185.Reg)
				ctx.W.EmitShlRegImm8(r160, 3)
			}
			if d184.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(r160, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r160, d184.Reg)
			}
			r161 := ctx.AllocRegExcept(r160)
			ctx.W.EmitMovRegMem(r161, r160, 0)
			ctx.FreeReg(r160)
			d186 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
			ctx.BindReg(r161, &d186)
			ctx.FreeDesc(&d185)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d187 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r162 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r162, d183.Reg)
				ctx.W.EmitAndRegImm32(r162, 63)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d187)
			}
			if d187.Loc == scm.LocReg && d183.Loc == scm.LocReg && d187.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d188 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d186.Imm.Int()) << uint64(d187.Imm.Int())))}
			} else if d187.Loc == scm.LocImm {
				r163 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r163, d186.Reg)
				ctx.W.EmitShlRegImm8(r163, uint8(d187.Imm.Int()))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d188)
			} else {
				{
					shiftSrc := d186.Reg
					r164 := ctx.AllocRegExcept(d186.Reg)
					ctx.W.EmitMovRegReg(r164, d186.Reg)
					shiftSrc = r164
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d187.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d187.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d187.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d188)
				}
			}
			if d188.Loc == scm.LocReg && d186.Loc == scm.LocReg && d188.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			ctx.FreeDesc(&d187)
			var d189 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r165 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r165, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
				ctx.BindReg(r165, &d189)
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d189.Loc == scm.LocImm {
				if d189.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
			d190 := d188
			if d190.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			ctx.EmitStoreToStack(d190, 104)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d189.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
			d191 := d188
			if d191.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			ctx.EmitStoreToStack(d191, 104)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d189)
			ctx.W.MarkLabel(lbl46)
			d192 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d193 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r166 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r166, thisptr.Reg, off)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r166}
				ctx.BindReg(r166, &d193)
			}
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			var d194 scm.JITValueDesc
			if d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d193.Imm.Int()))))}
			} else {
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r167, d193.Reg)
				ctx.W.EmitShlRegImm8(r167, 56)
				ctx.W.EmitShrRegImm8(r167, 56)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d194)
			}
			ctx.FreeDesc(&d193)
			d195 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm && d194.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d195.Imm.Int() - d194.Imm.Int())}
			} else if d194.Loc == scm.LocImm && d194.Imm.Int() == 0 {
				r168 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r168, d195.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d196)
			} else if d195.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d195.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d196)
			} else if d194.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(scratch, d195.Reg)
				if d194.Imm.Int() >= -2147483648 && d194.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d194.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d194.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d196)
			} else {
				r169 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r169, d195.Reg)
				ctx.W.EmitSubInt64(r169, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d196)
			}
			if d196.Loc == scm.LocReg && d195.Loc == scm.LocReg && d196.Reg == d195.Reg {
				ctx.TransferReg(d195.Reg)
				d195.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair { ctx.EnsureDesc(&d196) }
			var d197 scm.JITValueDesc
			if d192.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d192.Imm.Int()) >> uint64(d196.Imm.Int())))}
			} else if d196.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(r170, d192.Reg)
				ctx.W.EmitShrRegImm8(r170, uint8(d196.Imm.Int()))
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d197)
			} else {
				{
					shiftSrc := d192.Reg
					r171 := ctx.AllocRegExcept(d192.Reg)
					ctx.W.EmitMovRegReg(r171, d192.Reg)
					shiftSrc = r171
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d196.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d196.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d196.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d197)
				}
			}
			if d197.Loc == scm.LocReg && d192.Loc == scm.LocReg && d197.Reg == d192.Reg {
				ctx.TransferReg(d192.Reg)
				d192.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d192)
			ctx.FreeDesc(&d196)
			r172 := ctx.AllocReg()
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			ctx.EmitMovToReg(r172, d197)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl45)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d198 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r173 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r173, d183.Reg)
				ctx.W.EmitAndRegImm32(r173, 63)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d198)
			}
			if d198.Loc == scm.LocReg && d183.Loc == scm.LocReg && d198.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			var d199 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r174, thisptr.Reg, off)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
				ctx.BindReg(r174, &d199)
			}
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			var d200 scm.JITValueDesc
			if d199.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d199.Imm.Int()))))}
			} else {
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r175, d199.Reg)
				ctx.W.EmitShlRegImm8(r175, 56)
				ctx.W.EmitShrRegImm8(r175, 56)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d200)
			}
			ctx.FreeDesc(&d199)
			if d198.Loc == scm.LocStack || d198.Loc == scm.LocStackPair { ctx.EnsureDesc(&d198) }
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			var d201 scm.JITValueDesc
			if d198.Loc == scm.LocImm && d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() + d200.Imm.Int())}
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r176, d198.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d201)
			} else if d198.Loc == scm.LocImm && d198.Imm.Int() == 0 {
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d200.Reg}
				ctx.BindReg(d200.Reg, &d201)
			} else if d198.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d198.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d201)
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(scratch, d198.Reg)
				if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d200.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d200.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d201)
			} else {
				r177 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r177, d198.Reg)
				ctx.W.EmitAddInt64(r177, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d201)
			}
			if d201.Loc == scm.LocReg && d198.Loc == scm.LocReg && d201.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d198)
			ctx.FreeDesc(&d200)
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d201.Imm.Int()) > uint64(64))}
			} else {
				r178 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitCmpRegImm32(d201.Reg, 64)
				ctx.W.EmitSetcc(r178, scm.CcA)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r178}
				ctx.BindReg(r178, &d202)
			}
			ctx.FreeDesc(&d201)
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d202.Loc == scm.LocImm {
				if d202.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
			d203 := d188
			if d203.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			ctx.EmitStoreToStack(d203, 104)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d202.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl49)
			d204 := d188
			if d204.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			ctx.EmitStoreToStack(d204, 104)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl49)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d202)
			ctx.W.MarkLabel(lbl48)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d205 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() / 64)}
			} else {
				r179 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r179, d183.Reg)
				ctx.W.EmitShrRegImm8(r179, 6)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d205)
			}
			if d205.Loc == scm.LocReg && d183.Loc == scm.LocReg && d205.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			var d206 scm.JITValueDesc
			if d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d205.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegReg(scratch, d205.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			}
			if d206.Loc == scm.LocReg && d205.Loc == scm.LocReg && d206.Reg == d205.Reg {
				ctx.TransferReg(d205.Reg)
				d205.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			r180 := ctx.AllocReg()
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			if d206.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r180, uint64(d206.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r180, d206.Reg)
				ctx.W.EmitShlRegImm8(r180, 3)
			}
			if d184.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(r180, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r180, d184.Reg)
			}
			r181 := ctx.AllocRegExcept(r180)
			ctx.W.EmitMovRegMem(r181, r180, 0)
			ctx.FreeReg(r180)
			d207 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r181}
			ctx.BindReg(r181, &d207)
			ctx.FreeDesc(&d206)
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d208 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r182 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r182, d183.Reg)
				ctx.W.EmitAndRegImm32(r182, 63)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d208)
			}
			if d208.Loc == scm.LocReg && d183.Loc == scm.LocReg && d208.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d183)
			d209 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			var d210 scm.JITValueDesc
			if d209.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d209.Imm.Int() - d208.Imm.Int())}
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r183 := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegReg(r183, d209.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
				ctx.BindReg(r183, &d210)
			} else if d209.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d209.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d208.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d210)
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegReg(scratch, d209.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d210)
			} else {
				r184 := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegReg(r184, d209.Reg)
				ctx.W.EmitSubInt64(r184, d208.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d210)
			}
			if d210.Loc == scm.LocReg && d209.Loc == scm.LocReg && d210.Reg == d209.Reg {
				ctx.TransferReg(d209.Reg)
				d209.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			var d211 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d207.Imm.Int()) >> uint64(d210.Imm.Int())))}
			} else if d210.Loc == scm.LocImm {
				r185 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r185, d207.Reg)
				ctx.W.EmitShrRegImm8(r185, uint8(d210.Imm.Int()))
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d211)
			} else {
				{
					shiftSrc := d207.Reg
					r186 := ctx.AllocRegExcept(d207.Reg)
					ctx.W.EmitMovRegReg(r186, d207.Reg)
					shiftSrc = r186
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d210.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d210.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d210.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d211)
				}
			}
			if d211.Loc == scm.LocReg && d207.Loc == scm.LocReg && d211.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d207)
			ctx.FreeDesc(&d210)
			if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair { ctx.EnsureDesc(&d188) }
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			var d212 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() | d211.Imm.Int())}
			} else if d188.Loc == scm.LocImm && d188.Imm.Int() == 0 {
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d211.Reg}
				ctx.BindReg(d211.Reg, &d212)
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r187 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r187, d188.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d212)
			} else if d188.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d188.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else if d211.Loc == scm.LocImm {
				r188 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r188, d188.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r188, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitOrInt64(r188, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
				ctx.BindReg(r188, &d212)
			} else {
				r189 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r189, d188.Reg)
				ctx.W.EmitOrInt64(r189, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d212)
			}
			if d212.Loc == scm.LocReg && d188.Loc == scm.LocReg && d212.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			d213 := d212
			if d213.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair { ctx.EnsureDesc(&d213) }
			ctx.EmitStoreToStack(d213, 104)
			ctx.W.EmitJmp(lbl46)
			ctx.W.MarkLabel(lbl44)
			ctx.W.ResolveFixups()
			d214 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r172}
			ctx.BindReg(r172, &d214)
			ctx.BindReg(r172, &d214)
			if r152 { ctx.UnprotectReg(r153) }
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			var d215 scm.JITValueDesc
			if d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d214.Imm.Int()))))}
			} else {
				r190 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r190, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d215)
			}
			ctx.FreeDesc(&d214)
			var d216 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r191, thisptr.Reg, off)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
				ctx.BindReg(r191, &d216)
			}
			if d215.Loc == scm.LocStack || d215.Loc == scm.LocStackPair { ctx.EnsureDesc(&d215) }
			if d216.Loc == scm.LocStack || d216.Loc == scm.LocStackPair { ctx.EnsureDesc(&d216) }
			var d217 scm.JITValueDesc
			if d215.Loc == scm.LocImm && d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d215.Imm.Int() + d216.Imm.Int())}
			} else if d216.Loc == scm.LocImm && d216.Imm.Int() == 0 {
				r192 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r192, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d217)
			} else if d215.Loc == scm.LocImm && d215.Imm.Int() == 0 {
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d216.Reg}
				ctx.BindReg(d216.Reg, &d217)
			} else if d215.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d216.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d215.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else if d216.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(scratch, d215.Reg)
				if d216.Imm.Int() >= -2147483648 && d216.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d216.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d216.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else {
				r193 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r193, d215.Reg)
				ctx.W.EmitAddInt64(r193, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d217)
			}
			if d217.Loc == scm.LocReg && d215.Loc == scm.LocReg && d217.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			ctx.FreeDesc(&d216)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			lbl50 := ctx.W.ReserveLabel()
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d218 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d99.Imm.Int()))))}
			} else {
				r194 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r194, d99.Reg)
				ctx.W.EmitShlRegImm8(r194, 32)
				ctx.W.EmitShrRegImm8(r194, 32)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d218)
			}
			ctx.FreeDesc(&d99)
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r195, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195}
				ctx.BindReg(r195, &d219)
			}
			if d219.Loc == scm.LocStack || d219.Loc == scm.LocStackPair { ctx.EnsureDesc(&d219) }
			if d219.Loc == scm.LocStack || d219.Loc == scm.LocStackPair { ctx.EnsureDesc(&d219) }
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d219.Imm.Int()))))}
			} else {
				r196 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r196, d219.Reg)
				ctx.W.EmitShlRegImm8(r196, 56)
				ctx.W.EmitShrRegImm8(r196, 56)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d220)
			}
			ctx.FreeDesc(&d219)
			if d218.Loc == scm.LocStack || d218.Loc == scm.LocStackPair { ctx.EnsureDesc(&d218) }
			if d220.Loc == scm.LocStack || d220.Loc == scm.LocStackPair { ctx.EnsureDesc(&d220) }
			var d221 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() * d220.Imm.Int())}
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d221)
			} else if d220.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(scratch, d218.Reg)
				if d220.Imm.Int() >= -2147483648 && d220.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d220.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d220.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d221)
			} else {
				r197 := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(r197, d218.Reg)
				ctx.W.EmitImulInt64(r197, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d221)
			}
			if d221.Loc == scm.LocReg && d218.Loc == scm.LocReg && d221.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d220)
			var d222 scm.JITValueDesc
			r198 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r198, uint64(dataPtr))
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198, StackOff: int32(sliceLen)}
				ctx.BindReg(r198, &d222)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r198, thisptr.Reg, off)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
				ctx.BindReg(r198, &d222)
			}
			ctx.BindReg(r198, &d222)
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d223 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() / 64)}
			} else {
				r199 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r199, d221.Reg)
				ctx.W.EmitShrRegImm8(r199, 6)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d223)
			}
			if d223.Loc == scm.LocReg && d221.Loc == scm.LocReg && d223.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair { ctx.EnsureDesc(&d223) }
			r200 := ctx.AllocReg()
			if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair { ctx.EnsureDesc(&d223) }
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			if d223.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r200, uint64(d223.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r200, d223.Reg)
				ctx.W.EmitShlRegImm8(r200, 3)
			}
			if d222.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
				ctx.W.EmitAddInt64(r200, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r200, d222.Reg)
			}
			r201 := ctx.AllocRegExcept(r200)
			ctx.W.EmitMovRegMem(r201, r200, 0)
			ctx.FreeReg(r200)
			d224 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
			ctx.BindReg(r201, &d224)
			ctx.FreeDesc(&d223)
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d225 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() % 64)}
			} else {
				r202 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r202, d221.Reg)
				ctx.W.EmitAndRegImm32(r202, 63)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d225)
			}
			if d225.Loc == scm.LocReg && d221.Loc == scm.LocReg && d225.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			if d224.Loc == scm.LocStack || d224.Loc == scm.LocStackPair { ctx.EnsureDesc(&d224) }
			if d225.Loc == scm.LocStack || d225.Loc == scm.LocStackPair { ctx.EnsureDesc(&d225) }
			var d226 scm.JITValueDesc
			if d224.Loc == scm.LocImm && d225.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d224.Imm.Int()) << uint64(d225.Imm.Int())))}
			} else if d225.Loc == scm.LocImm {
				r203 := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(r203, d224.Reg)
				ctx.W.EmitShlRegImm8(r203, uint8(d225.Imm.Int()))
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d226)
			} else {
				{
					shiftSrc := d224.Reg
					r204 := ctx.AllocRegExcept(d224.Reg)
					ctx.W.EmitMovRegReg(r204, d224.Reg)
					shiftSrc = r204
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d225.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d225.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d225.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d226)
				}
			}
			if d226.Loc == scm.LocReg && d224.Loc == scm.LocReg && d226.Reg == d224.Reg {
				ctx.TransferReg(d224.Reg)
				d224.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d224)
			ctx.FreeDesc(&d225)
			var d227 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r205 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r205, thisptr.Reg, off)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
				ctx.BindReg(r205, &d227)
			}
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			if d227.Loc == scm.LocImm {
				if d227.Imm.Bool() {
					ctx.W.EmitJmp(lbl51)
				} else {
			d228 := d226
			if d228.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair { ctx.EnsureDesc(&d228) }
			ctx.EmitStoreToStack(d228, 112)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d227.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
			d229 := d226
			if d229.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair { ctx.EnsureDesc(&d229) }
			ctx.EmitStoreToStack(d229, 112)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d227)
			ctx.W.MarkLabel(lbl52)
			d230 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d231 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d231)
			}
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			var d232 scm.JITValueDesc
			if d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d231.Imm.Int()))))}
			} else {
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r207, d231.Reg)
				ctx.W.EmitShlRegImm8(r207, 56)
				ctx.W.EmitShrRegImm8(r207, 56)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d232)
			}
			ctx.FreeDesc(&d231)
			d233 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d234 scm.JITValueDesc
			if d233.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d233.Imm.Int() - d232.Imm.Int())}
			} else if d232.Loc == scm.LocImm && d232.Imm.Int() == 0 {
				r208 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r208, d233.Reg)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d234)
			} else if d233.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d233.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d232.Reg)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d234)
			} else if d232.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(scratch, d233.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d234)
			} else {
				r209 := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegReg(r209, d233.Reg)
				ctx.W.EmitSubInt64(r209, d232.Reg)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d234)
			}
			if d234.Loc == scm.LocReg && d233.Loc == scm.LocReg && d234.Reg == d233.Reg {
				ctx.TransferReg(d233.Reg)
				d233.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d232)
			if d230.Loc == scm.LocStack || d230.Loc == scm.LocStackPair { ctx.EnsureDesc(&d230) }
			if d234.Loc == scm.LocStack || d234.Loc == scm.LocStackPair { ctx.EnsureDesc(&d234) }
			var d235 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d234.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d230.Imm.Int()) >> uint64(d234.Imm.Int())))}
			} else if d234.Loc == scm.LocImm {
				r210 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(r210, d230.Reg)
				ctx.W.EmitShrRegImm8(r210, uint8(d234.Imm.Int()))
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d235)
			} else {
				{
					shiftSrc := d230.Reg
					r211 := ctx.AllocRegExcept(d230.Reg)
					ctx.W.EmitMovRegReg(r211, d230.Reg)
					shiftSrc = r211
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d234.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d234.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d234.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d235)
				}
			}
			if d235.Loc == scm.LocReg && d230.Loc == scm.LocReg && d235.Reg == d230.Reg {
				ctx.TransferReg(d230.Reg)
				d230.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d230)
			ctx.FreeDesc(&d234)
			r212 := ctx.AllocReg()
			if d235.Loc == scm.LocStack || d235.Loc == scm.LocStackPair { ctx.EnsureDesc(&d235) }
			ctx.EmitMovToReg(r212, d235)
			ctx.W.EmitJmp(lbl50)
			ctx.W.MarkLabel(lbl51)
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d236 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() % 64)}
			} else {
				r213 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r213, d221.Reg)
				ctx.W.EmitAndRegImm32(r213, 63)
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d236)
			}
			if d236.Loc == scm.LocReg && d221.Loc == scm.LocReg && d236.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			var d237 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r214 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r214, thisptr.Reg, off)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r214}
				ctx.BindReg(r214, &d237)
			}
			if d237.Loc == scm.LocStack || d237.Loc == scm.LocStackPair { ctx.EnsureDesc(&d237) }
			if d237.Loc == scm.LocStack || d237.Loc == scm.LocStackPair { ctx.EnsureDesc(&d237) }
			var d238 scm.JITValueDesc
			if d237.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d237.Imm.Int()))))}
			} else {
				r215 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r215, d237.Reg)
				ctx.W.EmitShlRegImm8(r215, 56)
				ctx.W.EmitShrRegImm8(r215, 56)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d238)
			}
			ctx.FreeDesc(&d237)
			if d236.Loc == scm.LocStack || d236.Loc == scm.LocStackPair { ctx.EnsureDesc(&d236) }
			if d238.Loc == scm.LocStack || d238.Loc == scm.LocStackPair { ctx.EnsureDesc(&d238) }
			var d239 scm.JITValueDesc
			if d236.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d236.Imm.Int() + d238.Imm.Int())}
			} else if d238.Loc == scm.LocImm && d238.Imm.Int() == 0 {
				r216 := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(r216, d236.Reg)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d239)
			} else if d236.Loc == scm.LocImm && d236.Imm.Int() == 0 {
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d238.Reg}
				ctx.BindReg(d238.Reg, &d239)
			} else if d236.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d236.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d238.Reg)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d239)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(scratch, d236.Reg)
				if d238.Imm.Int() >= -2147483648 && d238.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d238.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d238.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d239)
			} else {
				r217 := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(r217, d236.Reg)
				ctx.W.EmitAddInt64(r217, d238.Reg)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d239)
			}
			if d239.Loc == scm.LocReg && d236.Loc == scm.LocReg && d239.Reg == d236.Reg {
				ctx.TransferReg(d236.Reg)
				d236.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.FreeDesc(&d238)
			if d239.Loc == scm.LocStack || d239.Loc == scm.LocStackPair { ctx.EnsureDesc(&d239) }
			var d240 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d239.Imm.Int()) > uint64(64))}
			} else {
				r218 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitCmpRegImm32(d239.Reg, 64)
				ctx.W.EmitSetcc(r218, scm.CcA)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r218}
				ctx.BindReg(r218, &d240)
			}
			ctx.FreeDesc(&d239)
			lbl54 := ctx.W.ReserveLabel()
			lbl55 := ctx.W.ReserveLabel()
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
					ctx.W.EmitJmp(lbl54)
				} else {
			d241 := d226
			if d241.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d241.Loc == scm.LocStack || d241.Loc == scm.LocStackPair { ctx.EnsureDesc(&d241) }
			ctx.EmitStoreToStack(d241, 112)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d240.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl55)
			d242 := d226
			if d242.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d242.Loc == scm.LocStack || d242.Loc == scm.LocStackPair { ctx.EnsureDesc(&d242) }
			ctx.EmitStoreToStack(d242, 112)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl55)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.FreeDesc(&d240)
			ctx.W.MarkLabel(lbl54)
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d243 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() / 64)}
			} else {
				r219 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r219, d221.Reg)
				ctx.W.EmitShrRegImm8(r219, 6)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d243)
			}
			if d243.Loc == scm.LocReg && d221.Loc == scm.LocReg && d243.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair { ctx.EnsureDesc(&d243) }
			var d244 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(scratch, d243.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d244)
			}
			if d244.Loc == scm.LocReg && d243.Loc == scm.LocReg && d244.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d243)
			if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair { ctx.EnsureDesc(&d244) }
			r220 := ctx.AllocReg()
			if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair { ctx.EnsureDesc(&d244) }
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			if d244.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r220, uint64(d244.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r220, d244.Reg)
				ctx.W.EmitShlRegImm8(r220, 3)
			}
			if d222.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
				ctx.W.EmitAddInt64(r220, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r220, d222.Reg)
			}
			r221 := ctx.AllocRegExcept(r220)
			ctx.W.EmitMovRegMem(r221, r220, 0)
			ctx.FreeReg(r220)
			d245 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r221}
			ctx.BindReg(r221, &d245)
			ctx.FreeDesc(&d244)
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d246 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() % 64)}
			} else {
				r222 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r222, d221.Reg)
				ctx.W.EmitAndRegImm32(r222, 63)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d246)
			}
			if d246.Loc == scm.LocReg && d221.Loc == scm.LocReg && d246.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			d247 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d246.Loc == scm.LocStack || d246.Loc == scm.LocStackPair { ctx.EnsureDesc(&d246) }
			var d248 scm.JITValueDesc
			if d247.Loc == scm.LocImm && d246.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d247.Imm.Int() - d246.Imm.Int())}
			} else if d246.Loc == scm.LocImm && d246.Imm.Int() == 0 {
				r223 := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegReg(r223, d247.Reg)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d248)
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
				r224 := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegReg(r224, d247.Reg)
				ctx.W.EmitSubInt64(r224, d246.Reg)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d248)
			}
			if d248.Loc == scm.LocReg && d247.Loc == scm.LocReg && d248.Reg == d247.Reg {
				ctx.TransferReg(d247.Reg)
				d247.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			if d245.Loc == scm.LocStack || d245.Loc == scm.LocStackPair { ctx.EnsureDesc(&d245) }
			if d248.Loc == scm.LocStack || d248.Loc == scm.LocStackPair { ctx.EnsureDesc(&d248) }
			var d249 scm.JITValueDesc
			if d245.Loc == scm.LocImm && d248.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d245.Imm.Int()) >> uint64(d248.Imm.Int())))}
			} else if d248.Loc == scm.LocImm {
				r225 := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegReg(r225, d245.Reg)
				ctx.W.EmitShrRegImm8(r225, uint8(d248.Imm.Int()))
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d249)
			} else {
				{
					shiftSrc := d245.Reg
					r226 := ctx.AllocRegExcept(d245.Reg)
					ctx.W.EmitMovRegReg(r226, d245.Reg)
					shiftSrc = r226
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d248.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d248.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d248.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d249)
				}
			}
			if d249.Loc == scm.LocReg && d245.Loc == scm.LocReg && d249.Reg == d245.Reg {
				ctx.TransferReg(d245.Reg)
				d245.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d245)
			ctx.FreeDesc(&d248)
			if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair { ctx.EnsureDesc(&d226) }
			if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair { ctx.EnsureDesc(&d249) }
			var d250 scm.JITValueDesc
			if d226.Loc == scm.LocImm && d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d226.Imm.Int() | d249.Imm.Int())}
			} else if d226.Loc == scm.LocImm && d226.Imm.Int() == 0 {
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d249.Reg}
				ctx.BindReg(d249.Reg, &d250)
			} else if d249.Loc == scm.LocImm && d249.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitMovRegReg(r227, d226.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d250)
			} else if d226.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d226.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d249.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d250)
			} else if d249.Loc == scm.LocImm {
				r228 := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitMovRegReg(r228, d226.Reg)
				if d249.Imm.Int() >= -2147483648 && d249.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r228, int32(d249.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d249.Imm.Int()))
					ctx.W.EmitOrInt64(r228, scm.RegR11)
				}
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d250)
			} else {
				r229 := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitMovRegReg(r229, d226.Reg)
				ctx.W.EmitOrInt64(r229, d249.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d250)
			}
			if d250.Loc == scm.LocReg && d226.Loc == scm.LocReg && d250.Reg == d226.Reg {
				ctx.TransferReg(d226.Reg)
				d226.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			d251 := d250
			if d251.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d251.Loc == scm.LocStack || d251.Loc == scm.LocStackPair { ctx.EnsureDesc(&d251) }
			ctx.EmitStoreToStack(d251, 112)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl50)
			ctx.W.ResolveFixups()
			d252 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r212}
			ctx.BindReg(r212, &d252)
			ctx.BindReg(r212, &d252)
			ctx.FreeDesc(&d99)
			if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair { ctx.EnsureDesc(&d252) }
			if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair { ctx.EnsureDesc(&d252) }
			var d253 scm.JITValueDesc
			if d252.Loc == scm.LocImm {
				d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d252.Imm.Int()))))}
			} else {
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r230, d252.Reg)
				d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d253)
			}
			ctx.FreeDesc(&d252)
			var d254 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r231, thisptr.Reg, off)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r231}
				ctx.BindReg(r231, &d254)
			}
			if d253.Loc == scm.LocStack || d253.Loc == scm.LocStackPair { ctx.EnsureDesc(&d253) }
			if d254.Loc == scm.LocStack || d254.Loc == scm.LocStackPair { ctx.EnsureDesc(&d254) }
			var d255 scm.JITValueDesc
			if d253.Loc == scm.LocImm && d254.Loc == scm.LocImm {
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d253.Imm.Int() + d254.Imm.Int())}
			} else if d254.Loc == scm.LocImm && d254.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(r232, d253.Reg)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d255)
			} else if d253.Loc == scm.LocImm && d253.Imm.Int() == 0 {
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d254.Reg}
				ctx.BindReg(d254.Reg, &d255)
			} else if d253.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d253.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d254.Reg)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d255)
			} else if d254.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(scratch, d253.Reg)
				if d254.Imm.Int() >= -2147483648 && d254.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d254.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d254.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d255)
			} else {
				r233 := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(r233, d253.Reg)
				ctx.W.EmitAddInt64(r233, d254.Reg)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d255)
			}
			if d255.Loc == scm.LocReg && d253.Loc == scm.LocReg && d255.Reg == d253.Reg {
				ctx.TransferReg(d253.Reg)
				d253.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d253)
			ctx.FreeDesc(&d254)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d256 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r234, 32)
				ctx.W.EmitShrRegImm8(r234, 32)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d256)
			}
			ctx.FreeDesc(&idxInt)
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			if d255.Loc == scm.LocStack || d255.Loc == scm.LocStackPair { ctx.EnsureDesc(&d255) }
			var d257 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d255.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d256.Imm.Int() - d255.Imm.Int())}
			} else if d255.Loc == scm.LocImm && d255.Imm.Int() == 0 {
				r235 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r235, d256.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d257)
			} else if d256.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d255.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d256.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d255.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d257)
			} else if d255.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(scratch, d256.Reg)
				if d255.Imm.Int() >= -2147483648 && d255.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d255.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d255.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d257)
			} else {
				r236 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r236, d256.Reg)
				ctx.W.EmitSubInt64(r236, d255.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d257)
			}
			if d257.Loc == scm.LocReg && d256.Loc == scm.LocReg && d257.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d256)
			ctx.FreeDesc(&d255)
			if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair { ctx.EnsureDesc(&d257) }
			if d217.Loc == scm.LocStack || d217.Loc == scm.LocStackPair { ctx.EnsureDesc(&d217) }
			var d258 scm.JITValueDesc
			if d257.Loc == scm.LocImm && d217.Loc == scm.LocImm {
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d257.Imm.Int() * d217.Imm.Int())}
			} else if d257.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d257.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d217.Reg)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d258)
			} else if d217.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegReg(scratch, d257.Reg)
				if d217.Imm.Int() >= -2147483648 && d217.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d217.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d217.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d258)
			} else {
				r237 := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegReg(r237, d257.Reg)
				ctx.W.EmitImulInt64(r237, d217.Reg)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d258)
			}
			if d258.Loc == scm.LocReg && d257.Loc == scm.LocReg && d258.Reg == d257.Reg {
				ctx.TransferReg(d257.Reg)
				d257.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d257)
			ctx.FreeDesc(&d217)
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			var d259 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d258.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d138.Imm.Int() + d258.Imm.Int())}
			} else if d258.Loc == scm.LocImm && d258.Imm.Int() == 0 {
				r238 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r238, d138.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d259)
			} else if d138.Loc == scm.LocImm && d138.Imm.Int() == 0 {
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d258.Reg}
				ctx.BindReg(d258.Reg, &d259)
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else if d258.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(scratch, d138.Reg)
				if d258.Imm.Int() >= -2147483648 && d258.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d258.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d258.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else {
				r239 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r239, d138.Reg)
				ctx.W.EmitAddInt64(r239, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d259)
			}
			if d259.Loc == scm.LocReg && d138.Loc == scm.LocReg && d259.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d258)
			if d259.Loc == scm.LocStack || d259.Loc == scm.LocStackPair { ctx.EnsureDesc(&d259) }
			if d259.Loc == scm.LocStack || d259.Loc == scm.LocStackPair { ctx.EnsureDesc(&d259) }
			var d260 scm.JITValueDesc
			if d259.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d259.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d259.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d259.Reg}
				ctx.BindReg(d259.Reg, &d260)
			}
			ctx.FreeDesc(&d259)
			if d260.Loc == scm.LocStack || d260.Loc == scm.LocStackPair { ctx.EnsureDesc(&d260) }
			ctx.W.EmitMakeFloat(result, d260)
			if d260.Loc == scm.LocReg { ctx.FreeReg(d260.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl32)
			var d261 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r240, thisptr.Reg, off)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r240}
				ctx.BindReg(r240, &d261)
			}
			if d261.Loc == scm.LocStack || d261.Loc == scm.LocStackPair { ctx.EnsureDesc(&d261) }
			if d261.Loc == scm.LocStack || d261.Loc == scm.LocStackPair { ctx.EnsureDesc(&d261) }
			var d262 scm.JITValueDesc
			if d261.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d261.Imm.Int()))))}
			} else {
				r241 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r241, d261.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d262)
			}
			ctx.FreeDesc(&d261)
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d262.Loc == scm.LocStack || d262.Loc == scm.LocStackPair { ctx.EnsureDesc(&d262) }
			var d263 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d262.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d138.Imm.Int() == d262.Imm.Int())}
			} else if d262.Loc == scm.LocImm {
				r242 := ctx.AllocRegExcept(d138.Reg)
				if d262.Imm.Int() >= -2147483648 && d262.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d138.Reg, int32(d262.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d262.Imm.Int()))
					ctx.W.EmitCmpInt64(d138.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r242, scm.CcE)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r242}
				ctx.BindReg(r242, &d263)
			} else if d138.Loc == scm.LocImm {
				r243 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d262.Reg)
				ctx.W.EmitSetcc(r243, scm.CcE)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d263)
			} else {
				r244 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitCmpInt64(d138.Reg, d262.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d263)
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d262)
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			if d263.Loc == scm.LocImm {
				if d263.Imm.Bool() {
					ctx.W.EmitJmp(lbl56)
				} else {
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d263.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d263)
			ctx.W.MarkLabel(lbl42)
			lbl58 := ctx.W.ReserveLabel()
			d264 := d76
			if d264.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d264.Loc == scm.LocStack || d264.Loc == scm.LocStackPair { ctx.EnsureDesc(&d264) }
			d265 := d264
			if d265.Loc == scm.LocImm {
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: d265.Type, Imm: scm.NewInt(int64(uint64(d265.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d265.Reg, 32)
				ctx.W.EmitShrRegImm8(d265.Reg, 32)
			}
			ctx.EmitStoreToStack(d265, 64)
			d266 := d78
			if d266.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			d267 := d266
			if d267.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: d267.Type, Imm: scm.NewInt(int64(uint64(d267.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d267.Reg, 32)
				ctx.W.EmitShrRegImm8(d267.Reg, 32)
			}
			ctx.EmitStoreToStack(d267, 72)
			ctx.W.MarkLabel(lbl58)
			d268 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d269 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d269.Loc == scm.LocStack || d269.Loc == scm.LocStackPair { ctx.EnsureDesc(&d269) }
			var d270 scm.JITValueDesc
			if d268.Loc == scm.LocImm && d269.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d268.Imm.Int()) == uint64(d269.Imm.Int()))}
			} else if d269.Loc == scm.LocImm {
				r245 := ctx.AllocRegExcept(d268.Reg)
				if d269.Imm.Int() >= -2147483648 && d269.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d268.Reg, int32(d269.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d269.Imm.Int()))
					ctx.W.EmitCmpInt64(d268.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r245, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d270)
			} else if d268.Loc == scm.LocImm {
				r246 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d268.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d269.Reg)
				ctx.W.EmitSetcc(r246, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d270)
			} else {
				r247 := ctx.AllocRegExcept(d268.Reg)
				ctx.W.EmitCmpInt64(d268.Reg, d269.Reg)
				ctx.W.EmitSetcc(r247, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r247}
				ctx.BindReg(r247, &d270)
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			if d270.Loc == scm.LocImm {
				if d270.Imm.Bool() {
			d271 := d268
			if d271.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d271.Loc == scm.LocStack || d271.Loc == scm.LocStackPair { ctx.EnsureDesc(&d271) }
			d272 := d271
			if d272.Loc == scm.LocImm {
				d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: d272.Type, Imm: scm.NewInt(int64(uint64(d272.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d272.Reg, 32)
				ctx.W.EmitShrRegImm8(d272.Reg, 32)
			}
			ctx.EmitStoreToStack(d272, 32)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl59)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d270.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl60)
			d273 := d268
			if d273.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d273.Loc == scm.LocStack || d273.Loc == scm.LocStackPair { ctx.EnsureDesc(&d273) }
			d274 := d273
			if d274.Loc == scm.LocImm {
				d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: d274.Type, Imm: scm.NewInt(int64(uint64(d274.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d274.Reg, 32)
				ctx.W.EmitShrRegImm8(d274.Reg, 32)
			}
			ctx.EmitStoreToStack(d274, 32)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d270)
			ctx.W.MarkLabel(lbl41)
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			var d275 scm.JITValueDesc
			if d76.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d76.Imm.Int()) == uint64(0))}
			} else {
				r248 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitCmpRegImm32(d76.Reg, 0)
				ctx.W.EmitSetcc(r248, scm.CcE)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r248}
				ctx.BindReg(r248, &d275)
			}
			lbl61 := ctx.W.ReserveLabel()
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			if d275.Loc == scm.LocImm {
				if d275.Imm.Bool() {
					ctx.W.EmitJmp(lbl61)
				} else {
					ctx.W.EmitJmp(lbl62)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d275.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl63)
				ctx.W.EmitJmp(lbl62)
				ctx.W.MarkLabel(lbl63)
				ctx.W.EmitJmp(lbl61)
			}
			ctx.FreeDesc(&d275)
			ctx.W.MarkLabel(lbl56)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl59)
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d269.Loc == scm.LocStack || d269.Loc == scm.LocStackPair { ctx.EnsureDesc(&d269) }
			var d276 scm.JITValueDesc
			if d268.Loc == scm.LocImm && d269.Loc == scm.LocImm {
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d268.Imm.Int() + d269.Imm.Int())}
			} else if d269.Loc == scm.LocImm && d269.Imm.Int() == 0 {
				r249 := ctx.AllocRegExcept(d268.Reg)
				ctx.W.EmitMovRegReg(r249, d268.Reg)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r249}
				ctx.BindReg(r249, &d276)
			} else if d268.Loc == scm.LocImm && d268.Imm.Int() == 0 {
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d269.Reg}
				ctx.BindReg(d269.Reg, &d276)
			} else if d268.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d269.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d268.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d269.Reg)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d276)
			} else if d269.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d268.Reg)
				ctx.W.EmitMovRegReg(scratch, d268.Reg)
				if d269.Imm.Int() >= -2147483648 && d269.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d269.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d269.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d276)
			} else {
				r250 := ctx.AllocRegExcept(d268.Reg)
				ctx.W.EmitMovRegReg(r250, d268.Reg)
				ctx.W.EmitAddInt64(r250, d269.Reg)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d276)
			}
			if d276.Loc == scm.LocImm {
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: d276.Type, Imm: scm.NewInt(int64(uint64(d276.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d276.Reg, 32)
				ctx.W.EmitShrRegImm8(d276.Reg, 32)
			}
			if d276.Loc == scm.LocReg && d268.Loc == scm.LocReg && d276.Reg == d268.Reg {
				ctx.TransferReg(d268.Reg)
				d268.Loc = scm.LocNone
			}
			if d276.Loc == scm.LocStack || d276.Loc == scm.LocStackPair { ctx.EnsureDesc(&d276) }
			var d277 scm.JITValueDesc
			if d276.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d276.Imm.Int() / 2)}
			} else {
				r251 := ctx.AllocRegExcept(d276.Reg)
				ctx.W.EmitMovRegReg(r251, d276.Reg)
				ctx.W.EmitShrRegImm8(r251, 1)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d277)
			}
			if d277.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: d277.Type, Imm: scm.NewInt(int64(uint64(d277.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d277.Reg, 32)
				ctx.W.EmitShrRegImm8(d277.Reg, 32)
			}
			if d277.Loc == scm.LocReg && d276.Loc == scm.LocReg && d277.Reg == d276.Reg {
				ctx.TransferReg(d276.Reg)
				d276.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d276)
			d278 := d277
			if d278.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d278.Loc == scm.LocStack || d278.Loc == scm.LocStackPair { ctx.EnsureDesc(&d278) }
			d279 := d278
			if d279.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: d279.Type, Imm: scm.NewInt(int64(uint64(d279.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d279.Reg, 32)
				ctx.W.EmitShrRegImm8(d279.Reg, 32)
			}
			ctx.EmitStoreToStack(d279, 8)
			d280 := d268
			if d280.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d280.Loc == scm.LocStack || d280.Loc == scm.LocStackPair { ctx.EnsureDesc(&d280) }
			d281 := d280
			if d281.Loc == scm.LocImm {
				d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: d281.Type, Imm: scm.NewInt(int64(uint64(d281.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d281.Reg, 32)
				ctx.W.EmitShrRegImm8(d281.Reg, 32)
			}
			ctx.EmitStoreToStack(d281, 16)
			d282 := d269
			if d282.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d282.Loc == scm.LocStack || d282.Loc == scm.LocStackPair { ctx.EnsureDesc(&d282) }
			d283 := d282
			if d283.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: d283.Type, Imm: scm.NewInt(int64(uint64(d283.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d283.Reg, 32)
				ctx.W.EmitShrRegImm8(d283.Reg, 32)
			}
			ctx.EmitStoreToStack(d283, 24)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl62)
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			var d284 scm.JITValueDesc
			if d76.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(scratch, d76.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			}
			if d284.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: d284.Type, Imm: scm.NewInt(int64(uint64(d284.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d284.Reg, 32)
				ctx.W.EmitShrRegImm8(d284.Reg, 32)
			}
			if d284.Loc == scm.LocReg && d76.Loc == scm.LocReg && d284.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			d285 := d77
			if d285.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d285.Loc == scm.LocStack || d285.Loc == scm.LocStackPair { ctx.EnsureDesc(&d285) }
			d286 := d285
			if d286.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: d286.Type, Imm: scm.NewInt(int64(uint64(d286.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d286.Reg, 32)
				ctx.W.EmitShrRegImm8(d286.Reg, 32)
			}
			ctx.EmitStoreToStack(d286, 64)
			d287 := d284
			if d287.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d287.Loc == scm.LocStack || d287.Loc == scm.LocStackPair { ctx.EnsureDesc(&d287) }
			d288 := d287
			if d288.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: d288.Type, Imm: scm.NewInt(int64(uint64(d288.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d288.Reg, 32)
				ctx.W.EmitShrRegImm8(d288.Reg, 32)
			}
			ctx.EmitStoreToStack(d288, 72)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl61)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r0, int32(120))
			ctx.W.EmitAddRSP32(int32(120))
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
