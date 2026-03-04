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
				ctx.EnsureDesc(&idxInt)
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
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			r3 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				ctx.W.EmitMovRegMem(r3, thisptr.Reg, off)
			}
			d0 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			ctx.BindReg(r3, &d0)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d0.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r4, d0.Reg)
				ctx.W.EmitShlRegImm8(r4, 32)
				ctx.W.EmitShrRegImm8(r4, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			ctx.FreeDesc(&d0)
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).seqCount)
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMem32(r5, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d2)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).seqCount))
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMemL(r6, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
				ctx.BindReg(r6, &d2)
			}
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d2.Imm.Int()) == uint64(0))}
			} else {
				r7 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d3)
			}
			d4 := d3
			ctx.EnsureDesc(&d4)
			if d4.Loc != scm.LocImm && d4.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d4.Loc == scm.LocImm {
				if d4.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl4)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl5)
			ctx.W.EmitJmp(lbl3)
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d5 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r8 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r8, scm.CcAE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d5)
			} else if d1.Loc == scm.LocImm {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r9, scm.CcAE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r9}
				ctx.BindReg(r9, &d5)
			} else {
				r10 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r10, scm.CcAE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
				ctx.BindReg(r10, &d5)
			}
			d6 := d5
			ctx.EnsureDesc(&d6)
			if d6.Loc != scm.LocImm && d6.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d6.Loc == scm.LocImm {
				if d6.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.W.MarkLabel(lbl8)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl9)
			d7 := d1
			if d7.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d7)
			d8 := d7
			if d8.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: d8.Type, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d8.Reg, 32)
				ctx.W.EmitShrRegImm8(d8.Reg, 32)
			}
			ctx.EmitStoreToStack(d8, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			d9 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d9)
			ctx.BindReg(r2, &d9)
			ctx.W.EmitMakeNil(d9)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			d10 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d11 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d11)
			}
			if d11.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: d11.Type, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d11.Reg, 32)
				ctx.W.EmitShrRegImm8(d11.Reg, 32)
			}
			if d11.Loc == scm.LocReg && d2.Loc == scm.LocReg && d11.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			lbl10 := ctx.W.ReserveLabel()
			d12 := d10
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			d13 := d12
			if d13.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: d13.Type, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d13.Reg, 32)
				ctx.W.EmitShrRegImm8(d13.Reg, 32)
			}
			ctx.EmitStoreToStack(d13, 8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 16)
			d14 := d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			d15 := d14
			if d15.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: d15.Type, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d15.Reg, 32)
				ctx.W.EmitShrRegImm8(d15.Reg, 32)
			}
			ctx.EmitStoreToStack(d15, 24)
			ctx.W.MarkLabel(lbl10)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d18 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d16)
			d19 := d16
			_ = d19
			r11 := d16.Loc == scm.LocReg
			r12 := d16.Reg
			if r11 { ctx.ProtectReg(r12) }
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl12)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d19.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d19.Reg)
				ctx.W.EmitShlRegImm8(r13, 32)
				ctx.W.EmitShrRegImm8(r13, 32)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d20)
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d21)
			}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r15, d21.Reg)
				ctx.W.EmitShlRegImm8(r15, 56)
				ctx.W.EmitShrRegImm8(r15, 56)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d22)
			}
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() * d22.Imm.Int())}
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r16 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.W.EmitMovRegReg(r16, d20.Reg)
				ctx.W.EmitImulInt64(r16, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d23)
			}
			if d23.Loc == scm.LocReg && d20.Loc == scm.LocReg && d23.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.FreeDesc(&d22)
			var d24 scm.JITValueDesc
			r17 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r17, uint64(dataPtr))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17, StackOff: int32(sliceLen)}
				ctx.BindReg(r17, &d24)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r17, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d24)
			}
			ctx.BindReg(r17, &d24)
			ctx.EnsureDesc(&d23)
			var d25 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r18, d23.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d25)
			}
			if d25.Loc == scm.LocReg && d23.Loc == scm.LocReg && d25.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d25)
			r19 := ctx.AllocReg()
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d24)
			if d25.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d25.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d25.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d24.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d24.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d26 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d26)
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d23)
			var d27 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r21, d23.Reg)
				ctx.W.EmitAndRegImm32(r21, 63)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d27)
			}
			if d27.Loc == scm.LocReg && d23.Loc == scm.LocReg && d27.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d26.Imm.Int()) << uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r22, d26.Reg)
				ctx.W.EmitShlRegImm8(r22, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d28)
			} else {
				{
					shiftSrc := d26.Reg
					r23 := ctx.AllocRegExcept(d26.Reg)
					ctx.W.EmitMovRegReg(r23, d26.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d27.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d27.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d27.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d28)
				}
			}
			if d28.Loc == scm.LocReg && d26.Loc == scm.LocReg && d28.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.FreeDesc(&d27)
			var d29 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d29)
			}
			d30 := d29
			ctx.EnsureDesc(&d30)
			if d30.Loc != scm.LocImm && d30.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d30.Loc == scm.LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.W.MarkLabel(lbl15)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl16)
			d31 := d28
			if d31.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			ctx.EmitStoreToStack(d31, 80)
			ctx.W.EmitJmp(lbl14)
			ctx.FreeDesc(&d29)
			ctx.W.MarkLabel(lbl14)
			d32 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			var d33 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d33)
			}
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d33.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d33.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d34)
			}
			ctx.FreeDesc(&d33)
			d35 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() - d34.Imm.Int())}
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(r27, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d36)
			} else if d35.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(scratch, d35.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r28 := ctx.AllocRegExcept(d35.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r28, d35.Reg)
				ctx.W.EmitSubInt64(r28, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) >> uint64(d36.Imm.Int())))}
			} else if d36.Loc == scm.LocImm {
				r29 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r29, d32.Reg)
				ctx.W.EmitShrRegImm8(r29, uint8(d36.Imm.Int()))
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d37)
			} else {
				{
					shiftSrc := d32.Reg
					r30 := ctx.AllocRegExcept(d32.Reg)
					ctx.W.EmitMovRegReg(r30, d32.Reg)
					shiftSrc = r30
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d36.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d36.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d36.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d37)
				}
			}
			if d37.Loc == scm.LocReg && d32.Loc == scm.LocReg && d37.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.FreeDesc(&d36)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			if d37.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r31, d37)
			}
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl13)
			d32 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d23)
			var d38 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r32, d23.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d38)
			}
			if d38.Loc == scm.LocReg && d23.Loc == scm.LocReg && d38.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			var d39 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r33, thisptr.Reg, off)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
				ctx.BindReg(r33, &d39)
			}
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d39.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, d39.Reg)
				ctx.W.EmitShlRegImm8(r34, 56)
				ctx.W.EmitShrRegImm8(r34, 56)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d40)
			}
			ctx.FreeDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + d40.Imm.Int())}
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r35, d38.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d41)
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
				ctx.BindReg(d40.Reg, &d41)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d40.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else {
				r36 := ctx.AllocRegExcept(d38.Reg, d40.Reg)
				ctx.W.EmitMovRegReg(r36, d38.Reg)
				ctx.W.EmitAddInt64(r36, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d41)
			}
			if d41.Loc == scm.LocReg && d38.Loc == scm.LocReg && d41.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d41.Imm.Int()) > uint64(64))}
			} else {
				r37 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitCmpRegImm32(d41.Reg, 64)
				ctx.W.EmitSetcc(r37, scm.CcA)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
				ctx.BindReg(r37, &d42)
			}
			ctx.FreeDesc(&d41)
			d43 := d42
			ctx.EnsureDesc(&d43)
			if d43.Loc != scm.LocImm && d43.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
					ctx.W.EmitJmp(lbl19)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl19)
			d44 := d28
			if d44.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d44)
			ctx.EmitStoreToStack(d44, 80)
			ctx.W.EmitJmp(lbl14)
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl17)
			d32 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d23)
			var d45 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() / 64)}
			} else {
				r38 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r38, d23.Reg)
				ctx.W.EmitShrRegImm8(r38, 6)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d45)
			}
			if d45.Loc == scm.LocReg && d23.Loc == scm.LocReg && d45.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d45)
			var d46 scm.JITValueDesc
			if d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(scratch, d45.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			}
			if d46.Loc == scm.LocReg && d45.Loc == scm.LocReg && d46.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.EnsureDesc(&d46)
			r39 := ctx.AllocReg()
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d24)
			if d46.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d46.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d46.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			if d24.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(r39, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r39, d24.Reg)
			}
			r40 := ctx.AllocRegExcept(r39)
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.FreeReg(r39)
			d47 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d47)
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d23)
			var d48 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() % 64)}
			} else {
				r41 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r41, d23.Reg)
				ctx.W.EmitAndRegImm32(r41, 63)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d48)
			}
			if d48.Loc == scm.LocReg && d23.Loc == scm.LocReg && d48.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r42, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r43 := ctx.AllocRegExcept(d49.Reg, d48.Reg)
				ctx.W.EmitMovRegReg(r43, d49.Reg)
				ctx.W.EmitSubInt64(r43, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d50)
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) >> uint64(d50.Imm.Int())))}
			} else if d50.Loc == scm.LocImm {
				r44 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r44, d47.Reg)
				ctx.W.EmitShrRegImm8(r44, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d51)
			} else {
				{
					shiftSrc := d47.Reg
					r45 := ctx.AllocRegExcept(d47.Reg)
					ctx.W.EmitMovRegReg(r45, d47.Reg)
					shiftSrc = r45
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d50.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d50.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d50.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d51)
				}
			}
			if d51.Loc == scm.LocReg && d47.Loc == scm.LocReg && d51.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() | d51.Imm.Int())}
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d51.Reg}
				ctx.BindReg(d51.Reg, &d52)
			} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r46, d28.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d52)
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d52)
			} else if d51.Loc == scm.LocImm {
				r47 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r47, d28.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r47, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitOrInt64(r47, scm.RegR11)
				}
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d52)
			} else {
				r48 := ctx.AllocRegExcept(d28.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r48, d28.Reg)
				ctx.W.EmitOrInt64(r48, d51.Reg)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d52)
			}
			if d52.Loc == scm.LocReg && d28.Loc == scm.LocReg && d52.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			d53 := d52
			if d53.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d53)
			ctx.EmitStoreToStack(d53, 80)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl11)
			d54 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d54)
			ctx.BindReg(r31, &d54)
			if r11 { ctx.UnprotectReg(r12) }
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d54.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d55)
			}
			ctx.FreeDesc(&d54)
			var d56 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d56)
			}
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d56)
			var d57 scm.JITValueDesc
			if d55.Loc == scm.LocImm && d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() + d56.Imm.Int())}
			} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
				r51 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(r51, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d57)
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d56.Reg}
				ctx.BindReg(d56.Reg, &d57)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(scratch, d55.Reg)
				if d56.Imm.Int() >= -2147483648 && d56.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d56.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else {
				r52 := ctx.AllocRegExcept(d55.Reg, d56.Reg)
				ctx.W.EmitMovRegReg(r52, d55.Reg)
				ctx.W.EmitAddInt64(r52, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d57)
			}
			if d57.Loc == scm.LocReg && d55.Loc == scm.LocReg && d57.Reg == d55.Reg {
				ctx.TransferReg(d55.Reg)
				d55.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			ctx.FreeDesc(&d56)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d57)
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d57.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d57.Reg)
				ctx.W.EmitShlRegImm8(r53, 32)
				ctx.W.EmitShrRegImm8(r53, 32)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d58)
			}
			ctx.FreeDesc(&d57)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d58)
			var d59 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d58.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d58.Imm.Int()))}
			} else if d58.Loc == scm.LocImm {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				if d58.Imm.Int() >= -2147483648 && d58.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d58.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r54, scm.CcB)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d59)
			} else if idxInt.Loc == scm.LocImm {
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d58.Reg)
				ctx.W.EmitSetcc(r55, scm.CcB)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d59)
			} else {
				r56 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d58.Reg)
				ctx.W.EmitSetcc(r56, scm.CcB)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d59)
			}
			ctx.FreeDesc(&d58)
			d60 := d59
			ctx.EnsureDesc(&d60)
			if d60.Loc != scm.LocImm && d60.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d60.Loc == scm.LocImm {
				if d60.Imm.Bool() {
					ctx.W.EmitJmp(lbl22)
				} else {
					ctx.W.EmitJmp(lbl23)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d60.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.W.MarkLabel(lbl22)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl23)
			ctx.W.EmitJmp(lbl21)
			ctx.FreeDesc(&d59)
			ctx.W.MarkLabel(lbl6)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d61 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			}
			if d61.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: d61.Type, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d61.Reg, 32)
				ctx.W.EmitShrRegImm8(d61.Reg, 32)
			}
			if d61.Loc == scm.LocReg && d2.Loc == scm.LocReg && d61.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d62 := d61
			if d62.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d62)
			d63 := d62
			if d63.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: d63.Type, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d63.Reg, 32)
				ctx.W.EmitShrRegImm8(d63.Reg, 32)
			}
			ctx.EmitStoreToStack(d63, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl21)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			var d64 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			}
			if d64.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: d64.Type, Imm: scm.NewInt(int64(uint64(d64.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d64.Reg, 32)
				ctx.W.EmitShrRegImm8(d64.Reg, 32)
			}
			if d64.Loc == scm.LocReg && d16.Loc == scm.LocReg && d64.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d2)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d64.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d64.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d64.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d64.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r57, scm.CcAE)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d65)
			} else if d64.Loc == scm.LocImm {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d64.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r58, scm.CcAE)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d65)
			} else {
				r59 := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitCmpInt64(d64.Reg, d2.Reg)
				ctx.W.EmitSetcc(r59, scm.CcAE)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d65)
			}
			d66 := d65
			ctx.EnsureDesc(&d66)
			if d66.Loc != scm.LocImm && d66.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d66.Loc == scm.LocImm {
				if d66.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
					ctx.W.EmitJmp(lbl27)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d66.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.W.MarkLabel(lbl26)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl27)
			d67 := d64
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d67)
			d68 := d67
			if d68.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: d68.Type, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d68.Reg, 32)
				ctx.W.EmitShrRegImm8(d68.Reg, 32)
			}
			ctx.EmitStoreToStack(d68, 40)
			d69 := d16
			if d69.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d69)
			d70 := d69
			if d70.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: d70.Type, Imm: scm.NewInt(int64(uint64(d70.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d70.Reg, 32)
				ctx.W.EmitShrRegImm8(d70.Reg, 32)
			}
			ctx.EmitStoreToStack(d70, 48)
			d71 := d18
			if d71.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d71)
			d72 := d71
			if d72.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: d72.Type, Imm: scm.NewInt(int64(uint64(d72.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d72.Reg, 32)
				ctx.W.EmitShrRegImm8(d72.Reg, 32)
			}
			ctx.EmitStoreToStack(d72, 56)
			ctx.W.EmitJmp(lbl25)
			ctx.FreeDesc(&d65)
			ctx.W.MarkLabel(lbl20)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d16)
			var d73 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d16.Imm.Int()) == uint64(0))}
			} else {
				r60 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitSetcc(r60, scm.CcE)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d73)
			}
			d74 := d73
			ctx.EnsureDesc(&d74)
			if d74.Loc != scm.LocImm && d74.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d74.Loc == scm.LocImm {
				if d74.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d74.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl30)
				ctx.W.EmitJmp(lbl31)
			}
			ctx.W.MarkLabel(lbl30)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl31)
			ctx.W.EmitJmp(lbl29)
			ctx.FreeDesc(&d73)
			ctx.W.MarkLabel(lbl25)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d75 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d76 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d77 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d77)
			var d78 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d76.Imm.Int()) == uint64(d77.Imm.Int()))}
			} else if d77.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d76.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d76.Reg, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitCmpInt64(d76.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r61, scm.CcE)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d78)
			} else if d76.Loc == scm.LocImm {
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d77.Reg)
				ctx.W.EmitSetcc(r62, scm.CcE)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
				ctx.BindReg(r62, &d78)
			} else {
				r63 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitCmpInt64(d76.Reg, d77.Reg)
				ctx.W.EmitSetcc(r63, scm.CcE)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r63}
				ctx.BindReg(r63, &d78)
			}
			d79 := d78
			ctx.EnsureDesc(&d79)
			if d79.Loc != scm.LocImm && d79.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d79.Loc == scm.LocImm {
				if d79.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl35)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.W.MarkLabel(lbl34)
			d80 := d76
			if d80.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d80)
			d81 := d80
			if d81.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: d81.Type, Imm: scm.NewInt(int64(uint64(d81.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d81.Reg, 32)
				ctx.W.EmitShrRegImm8(d81.Reg, 32)
			}
			ctx.EmitStoreToStack(d81, 32)
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl35)
			ctx.W.EmitJmp(lbl33)
			ctx.FreeDesc(&d78)
			ctx.W.MarkLabel(lbl24)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d82 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			}
			if d82.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: d82.Type, Imm: scm.NewInt(int64(uint64(d82.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d82.Reg, 32)
				ctx.W.EmitShrRegImm8(d82.Reg, 32)
			}
			if d82.Loc == scm.LocReg && d2.Loc == scm.LocReg && d82.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d83 := d82
			if d83.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			d84 := d83
			if d84.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: d84.Type, Imm: scm.NewInt(int64(uint64(d84.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d84.Reg, 32)
				ctx.W.EmitShrRegImm8(d84.Reg, 32)
			}
			ctx.EmitStoreToStack(d84, 40)
			d85 := d16
			if d85.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d85)
			d86 := d85
			if d86.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: d86.Type, Imm: scm.NewInt(int64(uint64(d86.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d86.Reg, 32)
				ctx.W.EmitShrRegImm8(d86.Reg, 32)
			}
			ctx.EmitStoreToStack(d86, 48)
			d87 := d18
			if d87.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d87)
			d88 := d87
			if d88.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: d88.Type, Imm: scm.NewInt(int64(uint64(d88.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d88.Reg, 32)
				ctx.W.EmitShrRegImm8(d88.Reg, 32)
			}
			ctx.EmitStoreToStack(d88, 56)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl29)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			var d89 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d89)
			}
			if d89.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: d89.Type, Imm: scm.NewInt(int64(uint64(d89.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d89.Reg, 32)
				ctx.W.EmitShrRegImm8(d89.Reg, 32)
			}
			if d89.Loc == scm.LocReg && d16.Loc == scm.LocReg && d89.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			var d90 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			}
			if d90.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: d90.Type, Imm: scm.NewInt(int64(uint64(d90.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d90.Reg, 32)
				ctx.W.EmitShrRegImm8(d90.Reg, 32)
			}
			if d90.Loc == scm.LocReg && d16.Loc == scm.LocReg && d90.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			d91 := d90
			if d91.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d91)
			d92 := d91
			if d92.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: d92.Type, Imm: scm.NewInt(int64(uint64(d92.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d92.Reg, 32)
				ctx.W.EmitShrRegImm8(d92.Reg, 32)
			}
			ctx.EmitStoreToStack(d92, 40)
			d93 := d17
			if d93.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d93)
			d94 := d93
			if d94.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: d94.Type, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d94.Reg, 32)
				ctx.W.EmitShrRegImm8(d94.Reg, 32)
			}
			ctx.EmitStoreToStack(d94, 48)
			d95 := d89
			if d95.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d95)
			d96 := d95
			if d96.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: d96.Type, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d96.Reg, 32)
				ctx.W.EmitShrRegImm8(d96.Reg, 32)
			}
			ctx.EmitStoreToStack(d96, 56)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl28)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.MarkLabel(lbl32)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d97 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d97)
			var d98 scm.JITValueDesc
			if d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d97.Imm.Int()))))}
			} else {
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r64, d97.Reg)
				ctx.W.EmitShlRegImm8(r64, 32)
				ctx.W.EmitShrRegImm8(r64, 32)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d98)
			}
			ctx.EnsureDesc(&d98)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d98.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d98.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d98.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d98.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d98.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d98.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d98)
			ctx.EnsureDesc(&d97)
			d99 := d97
			_ = d99
			r65 := d97.Loc == scm.LocReg
			r66 := d97.Reg
			if r65 { ctx.ProtectReg(r66) }
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl37)
			ctx.EnsureDesc(&d99)
			ctx.EnsureDesc(&d99)
			var d100 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d99.Imm.Int()))))}
			} else {
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r67, d99.Reg)
				ctx.W.EmitShlRegImm8(r67, 32)
				ctx.W.EmitShrRegImm8(r67, 32)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d100)
			}
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
				ctx.BindReg(r68, &d101)
			}
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d101)
			var d102 scm.JITValueDesc
			if d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d101.Imm.Int()))))}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r69, d101.Reg)
				ctx.W.EmitShlRegImm8(r69, 56)
				ctx.W.EmitShrRegImm8(r69, 56)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d102)
			}
			ctx.FreeDesc(&d101)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d102)
			var d103 scm.JITValueDesc
			if d100.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d100.Imm.Int() * d102.Imm.Int())}
			} else if d100.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d100.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(scratch, d100.Reg)
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d102.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else {
				r70 := ctx.AllocRegExcept(d100.Reg, d102.Reg)
				ctx.W.EmitMovRegReg(r70, d100.Reg)
				ctx.W.EmitImulInt64(r70, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d103)
			}
			if d103.Loc == scm.LocReg && d100.Loc == scm.LocReg && d103.Reg == d100.Reg {
				ctx.TransferReg(d100.Reg)
				d100.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d100)
			ctx.FreeDesc(&d102)
			var d104 scm.JITValueDesc
			r71 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r71, uint64(dataPtr))
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71, StackOff: int32(sliceLen)}
				ctx.BindReg(r71, &d104)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r71, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
				ctx.BindReg(r71, &d104)
			}
			ctx.BindReg(r71, &d104)
			ctx.EnsureDesc(&d103)
			var d105 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() / 64)}
			} else {
				r72 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r72, d103.Reg)
				ctx.W.EmitShrRegImm8(r72, 6)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d105)
			}
			if d105.Loc == scm.LocReg && d103.Loc == scm.LocReg && d105.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d105)
			r73 := ctx.AllocReg()
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			if d105.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r73, uint64(d105.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r73, d105.Reg)
				ctx.W.EmitShlRegImm8(r73, 3)
			}
			if d104.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
				ctx.W.EmitAddInt64(r73, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r73, d104.Reg)
			}
			r74 := ctx.AllocRegExcept(r73)
			ctx.W.EmitMovRegMem(r74, r73, 0)
			ctx.FreeReg(r73)
			d106 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.BindReg(r74, &d106)
			ctx.FreeDesc(&d105)
			ctx.EnsureDesc(&d103)
			var d107 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() % 64)}
			} else {
				r75 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r75, d103.Reg)
				ctx.W.EmitAndRegImm32(r75, 63)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d107)
			}
			if d107.Loc == scm.LocReg && d103.Loc == scm.LocReg && d107.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d107)
			var d108 scm.JITValueDesc
			if d106.Loc == scm.LocImm && d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d106.Imm.Int()) << uint64(d107.Imm.Int())))}
			} else if d107.Loc == scm.LocImm {
				r76 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r76, d106.Reg)
				ctx.W.EmitShlRegImm8(r76, uint8(d107.Imm.Int()))
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d108)
			} else {
				{
					shiftSrc := d106.Reg
					r77 := ctx.AllocRegExcept(d106.Reg)
					ctx.W.EmitMovRegReg(r77, d106.Reg)
					shiftSrc = r77
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d107.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d107.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d107.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d108)
				}
			}
			if d108.Loc == scm.LocReg && d106.Loc == scm.LocReg && d108.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.FreeDesc(&d107)
			var d109 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r78, thisptr.Reg, off)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
				ctx.BindReg(r78, &d109)
			}
			d110 := d109
			ctx.EnsureDesc(&d110)
			if d110.Loc != scm.LocImm && d110.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d110.Loc == scm.LocImm {
				if d110.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
					ctx.W.EmitJmp(lbl41)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d110.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.W.MarkLabel(lbl40)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl41)
			d111 := d108
			if d111.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d111)
			ctx.EmitStoreToStack(d111, 88)
			ctx.W.EmitJmp(lbl39)
			ctx.FreeDesc(&d109)
			ctx.W.MarkLabel(lbl39)
			d112 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d113 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r79 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r79, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
				ctx.BindReg(r79, &d113)
			}
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d113.Imm.Int()))))}
			} else {
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r80, d113.Reg)
				ctx.W.EmitShlRegImm8(r80, 56)
				ctx.W.EmitShrRegImm8(r80, 56)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d114)
			}
			ctx.FreeDesc(&d113)
			d115 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d114)
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() - d114.Imm.Int())}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r81, d115.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d116)
			} else if d115.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d115.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d114.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(scratch, d115.Reg)
				if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d114.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			} else {
				r82 := ctx.AllocRegExcept(d115.Reg, d114.Reg)
				ctx.W.EmitMovRegReg(r82, d115.Reg)
				ctx.W.EmitSubInt64(r82, d114.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d116)
			}
			if d116.Loc == scm.LocReg && d115.Loc == scm.LocReg && d116.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d116)
			var d117 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d112.Imm.Int()) >> uint64(d116.Imm.Int())))}
			} else if d116.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r83, d112.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d116.Imm.Int()))
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d117)
			} else {
				{
					shiftSrc := d112.Reg
					r84 := ctx.AllocRegExcept(d112.Reg)
					ctx.W.EmitMovRegReg(r84, d112.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d116.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d116.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d116.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d117)
				}
			}
			if d117.Loc == scm.LocReg && d112.Loc == scm.LocReg && d117.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.FreeDesc(&d116)
			r85 := ctx.AllocReg()
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d117)
			if d117.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r85, d117)
			}
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl38)
			d112 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d103)
			var d118 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() % 64)}
			} else {
				r86 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r86, d103.Reg)
				ctx.W.EmitAndRegImm32(r86, 63)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d118)
			}
			if d118.Loc == scm.LocReg && d103.Loc == scm.LocReg && d118.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			var d119 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r87, thisptr.Reg, off)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d119)
			}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d119.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d119.Reg)
				ctx.W.EmitShlRegImm8(r88, 56)
				ctx.W.EmitShrRegImm8(r88, 56)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d120)
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d120)
			var d121 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() + d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				r89 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r89, d118.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d121)
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
				ctx.BindReg(d120.Reg, &d121)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d118.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(scratch, d118.Reg)
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d120.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else {
				r90 := ctx.AllocRegExcept(d118.Reg, d120.Reg)
				ctx.W.EmitMovRegReg(r90, d118.Reg)
				ctx.W.EmitAddInt64(r90, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d121)
			}
			if d121.Loc == scm.LocReg && d118.Loc == scm.LocReg && d121.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d120)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d121.Imm.Int()) > uint64(64))}
			} else {
				r91 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitCmpRegImm32(d121.Reg, 64)
				ctx.W.EmitSetcc(r91, scm.CcA)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r91}
				ctx.BindReg(r91, &d122)
			}
			ctx.FreeDesc(&d121)
			d123 := d122
			ctx.EnsureDesc(&d123)
			if d123.Loc != scm.LocImm && d123.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d123.Loc == scm.LocImm {
				if d123.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d123.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.W.MarkLabel(lbl43)
			ctx.W.EmitJmp(lbl42)
			ctx.W.MarkLabel(lbl44)
			d124 := d108
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, 88)
			ctx.W.EmitJmp(lbl39)
			ctx.FreeDesc(&d122)
			ctx.W.MarkLabel(lbl42)
			d112 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d103)
			var d125 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() / 64)}
			} else {
				r92 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r92, d103.Reg)
				ctx.W.EmitShrRegImm8(r92, 6)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d125)
			}
			if d125.Loc == scm.LocReg && d103.Loc == scm.LocReg && d125.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(scratch, d125.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			}
			if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.EnsureDesc(&d126)
			r93 := ctx.AllocReg()
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d104)
			if d126.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r93, uint64(d126.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r93, d126.Reg)
				ctx.W.EmitShlRegImm8(r93, 3)
			}
			if d104.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
				ctx.W.EmitAddInt64(r93, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r93, d104.Reg)
			}
			r94 := ctx.AllocRegExcept(r93)
			ctx.W.EmitMovRegMem(r94, r93, 0)
			ctx.FreeReg(r93)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r94}
			ctx.BindReg(r94, &d127)
			ctx.FreeDesc(&d126)
			ctx.EnsureDesc(&d103)
			var d128 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() % 64)}
			} else {
				r95 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r95, d103.Reg)
				ctx.W.EmitAndRegImm32(r95, 63)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d128)
			}
			if d128.Loc == scm.LocReg && d103.Loc == scm.LocReg && d128.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d103)
			d129 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d128)
			var d130 scm.JITValueDesc
			if d129.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() - d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				r96 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r96, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d130)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d129.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(scratch, d129.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r97 := ctx.AllocRegExcept(d129.Reg, d128.Reg)
				ctx.W.EmitMovRegReg(r97, d129.Reg)
				ctx.W.EmitSubInt64(r97, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d130)
			}
			if d130.Loc == scm.LocReg && d129.Loc == scm.LocReg && d130.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d130)
			var d131 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d127.Imm.Int()) >> uint64(d130.Imm.Int())))}
			} else if d130.Loc == scm.LocImm {
				r98 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r98, d127.Reg)
				ctx.W.EmitShrRegImm8(r98, uint8(d130.Imm.Int()))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d131)
			} else {
				{
					shiftSrc := d127.Reg
					r99 := ctx.AllocRegExcept(d127.Reg)
					ctx.W.EmitMovRegReg(r99, d127.Reg)
					shiftSrc = r99
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d130.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d130.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d130.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d131)
				}
			}
			if d131.Loc == scm.LocReg && d127.Loc == scm.LocReg && d131.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.FreeDesc(&d130)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d131)
			var d132 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() | d131.Imm.Int())}
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d131.Reg}
				ctx.BindReg(d131.Reg, &d132)
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				r100 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r100, d108.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d132)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d132)
			} else if d131.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r101, d108.Reg)
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r101, int32(d131.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
					ctx.W.EmitOrInt64(r101, scm.RegR11)
				}
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d132)
			} else {
				r102 := ctx.AllocRegExcept(d108.Reg, d131.Reg)
				ctx.W.EmitMovRegReg(r102, d108.Reg)
				ctx.W.EmitOrInt64(r102, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d132)
			}
			if d132.Loc == scm.LocReg && d108.Loc == scm.LocReg && d132.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			d133 := d132
			if d133.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d133)
			ctx.EmitStoreToStack(d133, 88)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl36)
			d134 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.BindReg(r85, &d134)
			ctx.BindReg(r85, &d134)
			if r65 { ctx.UnprotectReg(r66) }
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d134.Imm.Int()))))}
			} else {
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r103, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d135)
			}
			ctx.FreeDesc(&d134)
			var d136 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r104, thisptr.Reg, off)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
				ctx.BindReg(r104, &d136)
			}
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d136)
			var d137 scm.JITValueDesc
			if d135.Loc == scm.LocImm && d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() + d136.Imm.Int())}
			} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
				r105 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r105, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d137)
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
				ctx.BindReg(d136.Reg, &d137)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(scratch, d135.Reg)
				if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d136.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else {
				r106 := ctx.AllocRegExcept(d135.Reg, d136.Reg)
				ctx.W.EmitMovRegReg(r106, d135.Reg)
				ctx.W.EmitAddInt64(r106, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d137)
			}
			if d137.Loc == scm.LocReg && d135.Loc == scm.LocReg && d137.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			ctx.FreeDesc(&d136)
			var d138 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d138)
			}
			d139 := d138
			ctx.EnsureDesc(&d139)
			if d139.Loc != scm.LocImm && d139.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			if d139.Loc == scm.LocImm {
				if d139.Imm.Bool() {
					ctx.W.EmitJmp(lbl47)
				} else {
					ctx.W.EmitJmp(lbl48)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d139.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.W.MarkLabel(lbl47)
			ctx.W.EmitJmp(lbl45)
			ctx.W.MarkLabel(lbl48)
			ctx.W.EmitJmp(lbl46)
			ctx.FreeDesc(&d138)
			ctx.W.MarkLabel(lbl33)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d75)
			d140 := d75
			_ = d140
			r108 := d75.Loc == scm.LocReg
			r109 := d75.Reg
			if r108 { ctx.ProtectReg(r109) }
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl50)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d140.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d140.Reg)
				ctx.W.EmitShlRegImm8(r110, 32)
				ctx.W.EmitShrRegImm8(r110, 32)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d141)
			}
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r111, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r111}
				ctx.BindReg(r111, &d142)
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d142)
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d142.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d142.Reg)
				ctx.W.EmitShlRegImm8(r112, 56)
				ctx.W.EmitShrRegImm8(r112, 56)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d143)
			}
			ctx.FreeDesc(&d142)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() * d143.Imm.Int())}
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(scratch, d141.Reg)
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d143.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d143.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r113 := ctx.AllocRegExcept(d141.Reg, d143.Reg)
				ctx.W.EmitMovRegReg(r113, d141.Reg)
				ctx.W.EmitImulInt64(r113, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d144)
			}
			if d144.Loc == scm.LocReg && d141.Loc == scm.LocReg && d144.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			ctx.FreeDesc(&d143)
			var d145 scm.JITValueDesc
			r114 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r114, uint64(dataPtr))
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114, StackOff: int32(sliceLen)}
				ctx.BindReg(r114, &d145)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r114, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d145)
			}
			ctx.BindReg(r114, &d145)
			ctx.EnsureDesc(&d144)
			var d146 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() / 64)}
			} else {
				r115 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r115, d144.Reg)
				ctx.W.EmitShrRegImm8(r115, 6)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d146)
			}
			if d146.Loc == scm.LocReg && d144.Loc == scm.LocReg && d146.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d146)
			r116 := ctx.AllocReg()
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d145)
			if d146.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r116, uint64(d146.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r116, d146.Reg)
				ctx.W.EmitShlRegImm8(r116, 3)
			}
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.W.EmitAddInt64(r116, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r116, d145.Reg)
			}
			r117 := ctx.AllocRegExcept(r116)
			ctx.W.EmitMovRegMem(r117, r116, 0)
			ctx.FreeReg(r116)
			d147 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
			ctx.BindReg(r117, &d147)
			ctx.FreeDesc(&d146)
			ctx.EnsureDesc(&d144)
			var d148 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r118 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r118, d144.Reg)
				ctx.W.EmitAndRegImm32(r118, 63)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d148)
			}
			if d148.Loc == scm.LocReg && d144.Loc == scm.LocReg && d148.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d147.Imm.Int()) << uint64(d148.Imm.Int())))}
			} else if d148.Loc == scm.LocImm {
				r119 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r119, d147.Reg)
				ctx.W.EmitShlRegImm8(r119, uint8(d148.Imm.Int()))
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d149)
			} else {
				{
					shiftSrc := d147.Reg
					r120 := ctx.AllocRegExcept(d147.Reg)
					ctx.W.EmitMovRegReg(r120, d147.Reg)
					shiftSrc = r120
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d148.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d148.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d148.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d149)
				}
			}
			if d149.Loc == scm.LocReg && d147.Loc == scm.LocReg && d149.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.FreeDesc(&d148)
			var d150 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
				ctx.BindReg(r121, &d150)
			}
			d151 := d150
			ctx.EnsureDesc(&d151)
			if d151.Loc != scm.LocImm && d151.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d151.Loc == scm.LocImm {
				if d151.Imm.Bool() {
					ctx.W.EmitJmp(lbl53)
				} else {
					ctx.W.EmitJmp(lbl54)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d151.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.W.MarkLabel(lbl53)
			ctx.W.EmitJmp(lbl51)
			ctx.W.MarkLabel(lbl54)
			d152 := d149
			if d152.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d152)
			ctx.EmitStoreToStack(d152, 96)
			ctx.W.EmitJmp(lbl52)
			ctx.FreeDesc(&d150)
			ctx.W.MarkLabel(lbl52)
			d153 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d154 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r122, thisptr.Reg, off)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
				ctx.BindReg(r122, &d154)
			}
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d154.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r123, d154.Reg)
				ctx.W.EmitShlRegImm8(r123, 56)
				ctx.W.EmitShrRegImm8(r123, 56)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d155)
			}
			ctx.FreeDesc(&d154)
			d156 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d155)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() - d155.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r124, d156.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d157)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d155.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d157)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(scratch, d156.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d157)
			} else {
				r125 := ctx.AllocRegExcept(d156.Reg, d155.Reg)
				ctx.W.EmitMovRegReg(r125, d156.Reg)
				ctx.W.EmitSubInt64(r125, d155.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d157)
			}
			if d157.Loc == scm.LocReg && d156.Loc == scm.LocReg && d157.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d153.Imm.Int()) >> uint64(d157.Imm.Int())))}
			} else if d157.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r126, d153.Reg)
				ctx.W.EmitShrRegImm8(r126, uint8(d157.Imm.Int()))
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d158)
			} else {
				{
					shiftSrc := d153.Reg
					r127 := ctx.AllocRegExcept(d153.Reg)
					ctx.W.EmitMovRegReg(r127, d153.Reg)
					shiftSrc = r127
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d157.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d157.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d157.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d158)
				}
			}
			if d158.Loc == scm.LocReg && d153.Loc == scm.LocReg && d158.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d153)
			ctx.FreeDesc(&d157)
			r128 := ctx.AllocReg()
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d158)
			if d158.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r128, d158)
			}
			ctx.W.EmitJmp(lbl49)
			ctx.W.MarkLabel(lbl51)
			d153 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d144)
			var d159 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r129 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r129, d144.Reg)
				ctx.W.EmitAndRegImm32(r129, 63)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d159)
			}
			if d159.Loc == scm.LocReg && d144.Loc == scm.LocReg && d159.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			var d160 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r130, thisptr.Reg, off)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r130}
				ctx.BindReg(r130, &d160)
			}
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d160)
			var d161 scm.JITValueDesc
			if d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d160.Imm.Int()))))}
			} else {
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r131, d160.Reg)
				ctx.W.EmitShlRegImm8(r131, 56)
				ctx.W.EmitShrRegImm8(r131, 56)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d161)
			}
			ctx.FreeDesc(&d160)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d161)
			var d162 scm.JITValueDesc
			if d159.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() + d161.Imm.Int())}
			} else if d161.Loc == scm.LocImm && d161.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(r132, d159.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d162)
			} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
				ctx.BindReg(d161.Reg, &d162)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d161.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			} else if d161.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(scratch, d159.Reg)
				if d161.Imm.Int() >= -2147483648 && d161.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d161.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d161.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			} else {
				r133 := ctx.AllocRegExcept(d159.Reg, d161.Reg)
				ctx.W.EmitMovRegReg(r133, d159.Reg)
				ctx.W.EmitAddInt64(r133, d161.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d162)
			}
			if d162.Loc == scm.LocReg && d159.Loc == scm.LocReg && d162.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d159)
			ctx.FreeDesc(&d161)
			ctx.EnsureDesc(&d162)
			var d163 scm.JITValueDesc
			if d162.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d162.Imm.Int()) > uint64(64))}
			} else {
				r134 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitCmpRegImm32(d162.Reg, 64)
				ctx.W.EmitSetcc(r134, scm.CcA)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r134}
				ctx.BindReg(r134, &d163)
			}
			ctx.FreeDesc(&d162)
			d164 := d163
			ctx.EnsureDesc(&d164)
			if d164.Loc != scm.LocImm && d164.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			if d164.Loc == scm.LocImm {
				if d164.Imm.Bool() {
					ctx.W.EmitJmp(lbl56)
				} else {
					ctx.W.EmitJmp(lbl57)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d164.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.W.MarkLabel(lbl56)
			ctx.W.EmitJmp(lbl55)
			ctx.W.MarkLabel(lbl57)
			d165 := d149
			if d165.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d165)
			ctx.EmitStoreToStack(d165, 96)
			ctx.W.EmitJmp(lbl52)
			ctx.FreeDesc(&d163)
			ctx.W.MarkLabel(lbl55)
			d153 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d144)
			var d166 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() / 64)}
			} else {
				r135 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r135, d144.Reg)
				ctx.W.EmitShrRegImm8(r135, 6)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d166)
			}
			if d166.Loc == scm.LocReg && d144.Loc == scm.LocReg && d166.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d166)
			var d167 scm.JITValueDesc
			if d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d166.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegReg(scratch, d166.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d167)
			}
			if d167.Loc == scm.LocReg && d166.Loc == scm.LocReg && d167.Reg == d166.Reg {
				ctx.TransferReg(d166.Reg)
				d166.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.EnsureDesc(&d167)
			r136 := ctx.AllocReg()
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d145)
			if d167.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r136, uint64(d167.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r136, d167.Reg)
				ctx.W.EmitShlRegImm8(r136, 3)
			}
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
				ctx.W.EmitAddInt64(r136, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r136, d145.Reg)
			}
			r137 := ctx.AllocRegExcept(r136)
			ctx.W.EmitMovRegMem(r137, r136, 0)
			ctx.FreeReg(r136)
			d168 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d168)
			ctx.FreeDesc(&d167)
			ctx.EnsureDesc(&d144)
			var d169 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() % 64)}
			} else {
				r138 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r138, d144.Reg)
				ctx.W.EmitAndRegImm32(r138, 63)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d169)
			}
			if d169.Loc == scm.LocReg && d144.Loc == scm.LocReg && d169.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			d170 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d169)
			var d171 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() - d169.Imm.Int())}
			} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
				r139 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r139, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d171)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d169.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(scratch, d170.Reg)
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d169.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else {
				r140 := ctx.AllocRegExcept(d170.Reg, d169.Reg)
				ctx.W.EmitMovRegReg(r140, d170.Reg)
				ctx.W.EmitSubInt64(r140, d169.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d171)
			}
			if d171.Loc == scm.LocReg && d170.Loc == scm.LocReg && d171.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d171)
			var d172 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d168.Imm.Int()) >> uint64(d171.Imm.Int())))}
			} else if d171.Loc == scm.LocImm {
				r141 := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(r141, d168.Reg)
				ctx.W.EmitShrRegImm8(r141, uint8(d171.Imm.Int()))
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d172)
			} else {
				{
					shiftSrc := d168.Reg
					r142 := ctx.AllocRegExcept(d168.Reg)
					ctx.W.EmitMovRegReg(r142, d168.Reg)
					shiftSrc = r142
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d171.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d171.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d171.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d172)
				}
			}
			if d172.Loc == scm.LocReg && d168.Loc == scm.LocReg && d172.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			ctx.FreeDesc(&d171)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() | d172.Imm.Int())}
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
				ctx.BindReg(d172.Reg, &d173)
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				r143 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r143, d149.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d173)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else if d172.Loc == scm.LocImm {
				r144 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r144, d149.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r144, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitOrInt64(r144, scm.RegR11)
				}
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d173)
			} else {
				r145 := ctx.AllocRegExcept(d149.Reg, d172.Reg)
				ctx.W.EmitMovRegReg(r145, d149.Reg)
				ctx.W.EmitOrInt64(r145, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d173)
			}
			if d173.Loc == scm.LocReg && d149.Loc == scm.LocReg && d173.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			d174 := d173
			if d174.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d174)
			ctx.EmitStoreToStack(d174, 96)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl49)
			d175 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			ctx.BindReg(r128, &d175)
			ctx.BindReg(r128, &d175)
			if r108 { ctx.UnprotectReg(r109) }
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d175.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d176)
			}
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r147, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d177)
			}
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d177)
			var d178 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() + d177.Imm.Int())}
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				r148 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r148, d176.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d178)
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
				ctx.BindReg(d177.Reg, &d178)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(scratch, d176.Reg)
				if d177.Imm.Int() >= -2147483648 && d177.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d177.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else {
				r149 := ctx.AllocRegExcept(d176.Reg, d177.Reg)
				ctx.W.EmitMovRegReg(r149, d176.Reg)
				ctx.W.EmitAddInt64(r149, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d178)
			}
			if d178.Loc == scm.LocReg && d176.Loc == scm.LocReg && d178.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d177)
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d178)
			var d179 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d178.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d178.Reg)
				ctx.W.EmitShlRegImm8(r150, 32)
				ctx.W.EmitShrRegImm8(r150, 32)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d179)
			}
			ctx.FreeDesc(&d178)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d179)
			var d180 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d179.Imm.Int()))}
			} else if d179.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(idxInt.Reg)
				if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d179.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r151, scm.CcB)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d180)
			} else if idxInt.Loc == scm.LocImm {
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d179.Reg)
				ctx.W.EmitSetcc(r152, scm.CcB)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d180)
			} else {
				r153 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d179.Reg)
				ctx.W.EmitSetcc(r153, scm.CcB)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
				ctx.BindReg(r153, &d180)
			}
			ctx.FreeDesc(&d179)
			d181 := d180
			ctx.EnsureDesc(&d181)
			if d181.Loc != scm.LocImm && d181.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d181.Loc == scm.LocImm {
				if d181.Imm.Bool() {
					ctx.W.EmitJmp(lbl60)
				} else {
					ctx.W.EmitJmp(lbl61)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d181.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
			}
			ctx.W.MarkLabel(lbl60)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl61)
			ctx.W.EmitJmp(lbl59)
			ctx.FreeDesc(&d180)
			ctx.W.MarkLabel(lbl46)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d97)
			d182 := d97
			_ = d182
			r154 := d97.Loc == scm.LocReg
			r155 := d97.Reg
			if r154 { ctx.ProtectReg(r155) }
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl63)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d182.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, d182.Reg)
				ctx.W.EmitShlRegImm8(r156, 32)
				ctx.W.EmitShrRegImm8(r156, 32)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d183)
			}
			var d184 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r157, thisptr.Reg, off)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d184)
			}
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d184.Imm.Int()))))}
			} else {
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r158, d184.Reg)
				ctx.W.EmitShlRegImm8(r158, 56)
				ctx.W.EmitShrRegImm8(r158, 56)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d185)
			}
			ctx.FreeDesc(&d184)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d185)
			var d186 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() * d185.Imm.Int())}
			} else if d183.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d183.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d185.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d186)
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(scratch, d183.Reg)
				if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d185.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d186)
			} else {
				r159 := ctx.AllocRegExcept(d183.Reg, d185.Reg)
				ctx.W.EmitMovRegReg(r159, d183.Reg)
				ctx.W.EmitImulInt64(r159, d185.Reg)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d186)
			}
			if d186.Loc == scm.LocReg && d183.Loc == scm.LocReg && d186.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d183)
			ctx.FreeDesc(&d185)
			var d187 scm.JITValueDesc
			r160 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r160, uint64(dataPtr))
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160, StackOff: int32(sliceLen)}
				ctx.BindReg(r160, &d187)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r160, thisptr.Reg, off)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d187)
			}
			ctx.BindReg(r160, &d187)
			ctx.EnsureDesc(&d186)
			var d188 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r161, d186.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d188)
			}
			if d188.Loc == scm.LocReg && d186.Loc == scm.LocReg && d188.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d188)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d187)
			if d188.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d188.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d188.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d187.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d187.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d189 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d189)
			ctx.FreeDesc(&d188)
			ctx.EnsureDesc(&d186)
			var d190 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r164, d186.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d190)
			}
			if d190.Loc == scm.LocReg && d186.Loc == scm.LocReg && d190.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d190)
			var d191 scm.JITValueDesc
			if d189.Loc == scm.LocImm && d190.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d189.Imm.Int()) << uint64(d190.Imm.Int())))}
			} else if d190.Loc == scm.LocImm {
				r165 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r165, d189.Reg)
				ctx.W.EmitShlRegImm8(r165, uint8(d190.Imm.Int()))
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d191)
			} else {
				{
					shiftSrc := d189.Reg
					r166 := ctx.AllocRegExcept(d189.Reg)
					ctx.W.EmitMovRegReg(r166, d189.Reg)
					shiftSrc = r166
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d190.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d190.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d190.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d191)
				}
			}
			if d191.Loc == scm.LocReg && d189.Loc == scm.LocReg && d191.Reg == d189.Reg {
				ctx.TransferReg(d189.Reg)
				d189.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d189)
			ctx.FreeDesc(&d190)
			var d192 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r167, thisptr.Reg, off)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
				ctx.BindReg(r167, &d192)
			}
			d193 := d192
			ctx.EnsureDesc(&d193)
			if d193.Loc != scm.LocImm && d193.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			if d193.Loc == scm.LocImm {
				if d193.Imm.Bool() {
					ctx.W.EmitJmp(lbl66)
				} else {
					ctx.W.EmitJmp(lbl67)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d193.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl66)
				ctx.W.EmitJmp(lbl67)
			}
			ctx.W.MarkLabel(lbl66)
			ctx.W.EmitJmp(lbl64)
			ctx.W.MarkLabel(lbl67)
			d194 := d191
			if d194.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d194)
			ctx.EmitStoreToStack(d194, 104)
			ctx.W.EmitJmp(lbl65)
			ctx.FreeDesc(&d192)
			ctx.W.MarkLabel(lbl65)
			d195 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d196 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r168 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r168, thisptr.Reg, off)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r168}
				ctx.BindReg(r168, &d196)
			}
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d196)
			var d197 scm.JITValueDesc
			if d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d196.Imm.Int()))))}
			} else {
				r169 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r169, d196.Reg)
				ctx.W.EmitShlRegImm8(r169, 56)
				ctx.W.EmitShrRegImm8(r169, 56)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d197)
			}
			ctx.FreeDesc(&d196)
			d198 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d197)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() - d197.Imm.Int())}
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				r170 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r170, d198.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d199)
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
				r171 := ctx.AllocRegExcept(d198.Reg, d197.Reg)
				ctx.W.EmitMovRegReg(r171, d198.Reg)
				ctx.W.EmitSubInt64(r171, d197.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d199)
			}
			if d199.Loc == scm.LocReg && d198.Loc == scm.LocReg && d199.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d199)
			var d200 scm.JITValueDesc
			if d195.Loc == scm.LocImm && d199.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d195.Imm.Int()) >> uint64(d199.Imm.Int())))}
			} else if d199.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r172, d195.Reg)
				ctx.W.EmitShrRegImm8(r172, uint8(d199.Imm.Int()))
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d200)
			} else {
				{
					shiftSrc := d195.Reg
					r173 := ctx.AllocRegExcept(d195.Reg)
					ctx.W.EmitMovRegReg(r173, d195.Reg)
					shiftSrc = r173
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
			if d200.Loc == scm.LocReg && d195.Loc == scm.LocReg && d200.Reg == d195.Reg {
				ctx.TransferReg(d195.Reg)
				d195.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d195)
			ctx.FreeDesc(&d199)
			r174 := ctx.AllocReg()
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d200)
			if d200.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r174, d200)
			}
			ctx.W.EmitJmp(lbl62)
			ctx.W.MarkLabel(lbl64)
			d195 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d186)
			var d201 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() % 64)}
			} else {
				r175 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r175, d186.Reg)
				ctx.W.EmitAndRegImm32(r175, 63)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d201)
			}
			if d201.Loc == scm.LocReg && d186.Loc == scm.LocReg && d201.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			var d202 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r176 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r176, thisptr.Reg, off)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r176}
				ctx.BindReg(r176, &d202)
			}
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d202.Imm.Int()))))}
			} else {
				r177 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r177, d202.Reg)
				ctx.W.EmitShlRegImm8(r177, 56)
				ctx.W.EmitShrRegImm8(r177, 56)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d203)
			}
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d201.Loc == scm.LocImm && d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() + d203.Imm.Int())}
			} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r178, d201.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d204)
			} else if d201.Loc == scm.LocImm && d201.Imm.Int() == 0 {
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d203.Reg}
				ctx.BindReg(d203.Reg, &d204)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d201.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			} else if d203.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(scratch, d201.Reg)
				if d203.Imm.Int() >= -2147483648 && d203.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d203.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			} else {
				r179 := ctx.AllocRegExcept(d201.Reg, d203.Reg)
				ctx.W.EmitMovRegReg(r179, d201.Reg)
				ctx.W.EmitAddInt64(r179, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d204)
			}
			if d204.Loc == scm.LocReg && d201.Loc == scm.LocReg && d204.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			ctx.FreeDesc(&d203)
			ctx.EnsureDesc(&d204)
			var d205 scm.JITValueDesc
			if d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d204.Imm.Int()) > uint64(64))}
			} else {
				r180 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitCmpRegImm32(d204.Reg, 64)
				ctx.W.EmitSetcc(r180, scm.CcA)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r180}
				ctx.BindReg(r180, &d205)
			}
			ctx.FreeDesc(&d204)
			d206 := d205
			ctx.EnsureDesc(&d206)
			if d206.Loc != scm.LocImm && d206.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			if d206.Loc == scm.LocImm {
				if d206.Imm.Bool() {
					ctx.W.EmitJmp(lbl69)
				} else {
					ctx.W.EmitJmp(lbl70)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d206.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl69)
				ctx.W.EmitJmp(lbl70)
			}
			ctx.W.MarkLabel(lbl69)
			ctx.W.EmitJmp(lbl68)
			ctx.W.MarkLabel(lbl70)
			d207 := d191
			if d207.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d207)
			ctx.EmitStoreToStack(d207, 104)
			ctx.W.EmitJmp(lbl65)
			ctx.FreeDesc(&d205)
			ctx.W.MarkLabel(lbl68)
			d195 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d186)
			var d208 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() / 64)}
			} else {
				r181 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r181, d186.Reg)
				ctx.W.EmitShrRegImm8(r181, 6)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d208)
			}
			if d208.Loc == scm.LocReg && d186.Loc == scm.LocReg && d208.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d208.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(scratch, d208.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			}
			if d209.Loc == scm.LocReg && d208.Loc == scm.LocReg && d209.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			r182 := ctx.AllocReg()
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d187)
			if d209.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d209.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r182, d209.Reg)
				ctx.W.EmitShlRegImm8(r182, 3)
			}
			if d187.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
				ctx.W.EmitAddInt64(r182, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r182, d187.Reg)
			}
			r183 := ctx.AllocRegExcept(r182)
			ctx.W.EmitMovRegMem(r183, r182, 0)
			ctx.FreeReg(r182)
			d210 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
			ctx.BindReg(r183, &d210)
			ctx.FreeDesc(&d209)
			ctx.EnsureDesc(&d186)
			var d211 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() % 64)}
			} else {
				r184 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r184, d186.Reg)
				ctx.W.EmitAndRegImm32(r184, 63)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d211)
			}
			if d211.Loc == scm.LocReg && d186.Loc == scm.LocReg && d211.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			d212 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() - d211.Imm.Int())}
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r185 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r185, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r186 := ctx.AllocRegExcept(d212.Reg, d211.Reg)
				ctx.W.EmitMovRegReg(r186, d212.Reg)
				ctx.W.EmitSubInt64(r186, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d213)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d210.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) >> uint64(d213.Imm.Int())))}
			} else if d213.Loc == scm.LocImm {
				r187 := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(r187, d210.Reg)
				ctx.W.EmitShrRegImm8(r187, uint8(d213.Imm.Int()))
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d214)
			} else {
				{
					shiftSrc := d210.Reg
					r188 := ctx.AllocRegExcept(d210.Reg)
					ctx.W.EmitMovRegReg(r188, d210.Reg)
					shiftSrc = r188
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d213.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d213.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d213.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d214)
				}
			}
			if d214.Loc == scm.LocReg && d210.Loc == scm.LocReg && d214.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d214)
			var d215 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() | d214.Imm.Int())}
			} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
				ctx.BindReg(d214.Reg, &d215)
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r189, d191.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d215)
			} else if d191.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d191.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else if d214.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r190, d191.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r190, int32(d214.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.W.EmitOrInt64(r190, scm.RegR11)
				}
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d215)
			} else {
				r191 := ctx.AllocRegExcept(d191.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r191, d191.Reg)
				ctx.W.EmitOrInt64(r191, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d215)
			}
			if d215.Loc == scm.LocReg && d191.Loc == scm.LocReg && d215.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			d216 := d215
			if d216.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d216)
			ctx.EmitStoreToStack(d216, 104)
			ctx.W.EmitJmp(lbl65)
			ctx.W.MarkLabel(lbl62)
			d217 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
			ctx.BindReg(r174, &d217)
			ctx.BindReg(r174, &d217)
			if r154 { ctx.UnprotectReg(r155) }
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d217.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r192, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d218)
			}
			ctx.FreeDesc(&d217)
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r193, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
				ctx.BindReg(r193, &d219)
			}
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() + d219.Imm.Int())}
			} else if d219.Loc == scm.LocImm && d219.Imm.Int() == 0 {
				r194 := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(r194, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d220)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d219.Reg}
				ctx.BindReg(d219.Reg, &d220)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(scratch, d218.Reg)
				if d219.Imm.Int() >= -2147483648 && d219.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d219.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d219.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else {
				r195 := ctx.AllocRegExcept(d218.Reg, d219.Reg)
				ctx.W.EmitMovRegReg(r195, d218.Reg)
				ctx.W.EmitAddInt64(r195, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d220)
			}
			if d220.Loc == scm.LocReg && d218.Loc == scm.LocReg && d220.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d97)
			d221 := d97
			_ = d221
			r196 := d97.Loc == scm.LocReg
			r197 := d97.Reg
			if r196 { ctx.ProtectReg(r197) }
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl72)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d221)
			var d222 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d221.Imm.Int()))))}
			} else {
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r198, d221.Reg)
				ctx.W.EmitShlRegImm8(r198, 32)
				ctx.W.EmitShrRegImm8(r198, 32)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d222)
			}
			var d223 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r199, thisptr.Reg, off)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d223)
			}
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d223.Imm.Int()))))}
			} else {
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r200, d223.Reg)
				ctx.W.EmitShlRegImm8(r200, 56)
				ctx.W.EmitShrRegImm8(r200, 56)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d224)
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			var d225 scm.JITValueDesc
			if d222.Loc == scm.LocImm && d224.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() * d224.Imm.Int())}
			} else if d222.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d222.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d224.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d225)
			} else if d224.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(scratch, d222.Reg)
				if d224.Imm.Int() >= -2147483648 && d224.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d224.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d224.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d225)
			} else {
				r201 := ctx.AllocRegExcept(d222.Reg, d224.Reg)
				ctx.W.EmitMovRegReg(r201, d222.Reg)
				ctx.W.EmitImulInt64(r201, d224.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d225)
			}
			if d225.Loc == scm.LocReg && d222.Loc == scm.LocReg && d225.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d222)
			ctx.FreeDesc(&d224)
			var d226 scm.JITValueDesc
			r202 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r202, uint64(dataPtr))
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202, StackOff: int32(sliceLen)}
				ctx.BindReg(r202, &d226)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r202, thisptr.Reg, off)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d226)
			}
			ctx.BindReg(r202, &d226)
			ctx.EnsureDesc(&d225)
			var d227 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d225.Imm.Int() / 64)}
			} else {
				r203 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r203, d225.Reg)
				ctx.W.EmitShrRegImm8(r203, 6)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d227)
			}
			if d227.Loc == scm.LocReg && d225.Loc == scm.LocReg && d227.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d227)
			r204 := ctx.AllocReg()
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d226)
			if d227.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r204, uint64(d227.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r204, d227.Reg)
				ctx.W.EmitShlRegImm8(r204, 3)
			}
			if d226.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
				ctx.W.EmitAddInt64(r204, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r204, d226.Reg)
			}
			r205 := ctx.AllocRegExcept(r204)
			ctx.W.EmitMovRegMem(r205, r204, 0)
			ctx.FreeReg(r204)
			d228 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
			ctx.BindReg(r205, &d228)
			ctx.FreeDesc(&d227)
			ctx.EnsureDesc(&d225)
			var d229 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d225.Imm.Int() % 64)}
			} else {
				r206 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r206, d225.Reg)
				ctx.W.EmitAndRegImm32(r206, 63)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d229)
			}
			if d229.Loc == scm.LocReg && d225.Loc == scm.LocReg && d229.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d229)
			var d230 scm.JITValueDesc
			if d228.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d228.Imm.Int()) << uint64(d229.Imm.Int())))}
			} else if d229.Loc == scm.LocImm {
				r207 := ctx.AllocRegExcept(d228.Reg)
				ctx.W.EmitMovRegReg(r207, d228.Reg)
				ctx.W.EmitShlRegImm8(r207, uint8(d229.Imm.Int()))
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d230)
			} else {
				{
					shiftSrc := d228.Reg
					r208 := ctx.AllocRegExcept(d228.Reg)
					ctx.W.EmitMovRegReg(r208, d228.Reg)
					shiftSrc = r208
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d229.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d229.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d229.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d230)
				}
			}
			if d230.Loc == scm.LocReg && d228.Loc == scm.LocReg && d230.Reg == d228.Reg {
				ctx.TransferReg(d228.Reg)
				d228.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d228)
			ctx.FreeDesc(&d229)
			var d231 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r209 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r209, thisptr.Reg, off)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
				ctx.BindReg(r209, &d231)
			}
			d232 := d231
			ctx.EnsureDesc(&d232)
			if d232.Loc != scm.LocImm && d232.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			if d232.Loc == scm.LocImm {
				if d232.Imm.Bool() {
					ctx.W.EmitJmp(lbl75)
				} else {
					ctx.W.EmitJmp(lbl76)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d232.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl75)
				ctx.W.EmitJmp(lbl76)
			}
			ctx.W.MarkLabel(lbl75)
			ctx.W.EmitJmp(lbl73)
			ctx.W.MarkLabel(lbl76)
			d233 := d230
			if d233.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d233)
			ctx.EmitStoreToStack(d233, 112)
			ctx.W.EmitJmp(lbl74)
			ctx.FreeDesc(&d231)
			ctx.W.MarkLabel(lbl74)
			d234 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d235 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d235)
			}
			ctx.EnsureDesc(&d235)
			ctx.EnsureDesc(&d235)
			var d236 scm.JITValueDesc
			if d235.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d235.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d235.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d236)
			}
			ctx.FreeDesc(&d235)
			d237 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d236)
			var d238 scm.JITValueDesc
			if d237.Loc == scm.LocImm && d236.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d237.Imm.Int() - d236.Imm.Int())}
			} else if d236.Loc == scm.LocImm && d236.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r212, d237.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d238)
			} else if d237.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d237.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d236.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d238)
			} else if d236.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(scratch, d237.Reg)
				if d236.Imm.Int() >= -2147483648 && d236.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d236.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d236.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d238)
			} else {
				r213 := ctx.AllocRegExcept(d237.Reg, d236.Reg)
				ctx.W.EmitMovRegReg(r213, d237.Reg)
				ctx.W.EmitSubInt64(r213, d236.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d238)
			}
			if d238.Loc == scm.LocReg && d237.Loc == scm.LocReg && d238.Reg == d237.Reg {
				ctx.TransferReg(d237.Reg)
				d237.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.EnsureDesc(&d234)
			ctx.EnsureDesc(&d238)
			var d239 scm.JITValueDesc
			if d234.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d234.Imm.Int()) >> uint64(d238.Imm.Int())))}
			} else if d238.Loc == scm.LocImm {
				r214 := ctx.AllocRegExcept(d234.Reg)
				ctx.W.EmitMovRegReg(r214, d234.Reg)
				ctx.W.EmitShrRegImm8(r214, uint8(d238.Imm.Int()))
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d239)
			} else {
				{
					shiftSrc := d234.Reg
					r215 := ctx.AllocRegExcept(d234.Reg)
					ctx.W.EmitMovRegReg(r215, d234.Reg)
					shiftSrc = r215
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d238.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d238.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d238.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d239)
				}
			}
			if d239.Loc == scm.LocReg && d234.Loc == scm.LocReg && d239.Reg == d234.Reg {
				ctx.TransferReg(d234.Reg)
				d234.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d234)
			ctx.FreeDesc(&d238)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d239)
			if d239.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r216, d239)
			}
			ctx.W.EmitJmp(lbl71)
			ctx.W.MarkLabel(lbl73)
			d234 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d225)
			var d240 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d225.Imm.Int() % 64)}
			} else {
				r217 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r217, d225.Reg)
				ctx.W.EmitAndRegImm32(r217, 63)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d240)
			}
			if d240.Loc == scm.LocReg && d225.Loc == scm.LocReg && d240.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			var d241 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r218 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r218, thisptr.Reg, off)
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
				ctx.BindReg(r218, &d241)
			}
			ctx.EnsureDesc(&d241)
			ctx.EnsureDesc(&d241)
			var d242 scm.JITValueDesc
			if d241.Loc == scm.LocImm {
				d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d241.Imm.Int()))))}
			} else {
				r219 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r219, d241.Reg)
				ctx.W.EmitShlRegImm8(r219, 56)
				ctx.W.EmitShrRegImm8(r219, 56)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d242)
			}
			ctx.FreeDesc(&d241)
			ctx.EnsureDesc(&d240)
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d240)
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d240)
			ctx.EnsureDesc(&d242)
			var d243 scm.JITValueDesc
			if d240.Loc == scm.LocImm && d242.Loc == scm.LocImm {
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d240.Imm.Int() + d242.Imm.Int())}
			} else if d242.Loc == scm.LocImm && d242.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d240.Reg)
				ctx.W.EmitMovRegReg(r220, d240.Reg)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d243)
			} else if d240.Loc == scm.LocImm && d240.Imm.Int() == 0 {
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d242.Reg}
				ctx.BindReg(d242.Reg, &d243)
			} else if d240.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d242.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d240.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d242.Reg)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d243)
			} else if d242.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d240.Reg)
				ctx.W.EmitMovRegReg(scratch, d240.Reg)
				if d242.Imm.Int() >= -2147483648 && d242.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d242.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d242.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d243)
			} else {
				r221 := ctx.AllocRegExcept(d240.Reg, d242.Reg)
				ctx.W.EmitMovRegReg(r221, d240.Reg)
				ctx.W.EmitAddInt64(r221, d242.Reg)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d243)
			}
			if d243.Loc == scm.LocReg && d240.Loc == scm.LocReg && d243.Reg == d240.Reg {
				ctx.TransferReg(d240.Reg)
				d240.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d240)
			ctx.FreeDesc(&d242)
			ctx.EnsureDesc(&d243)
			var d244 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d243.Imm.Int()) > uint64(64))}
			} else {
				r222 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitCmpRegImm32(d243.Reg, 64)
				ctx.W.EmitSetcc(r222, scm.CcA)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r222}
				ctx.BindReg(r222, &d244)
			}
			ctx.FreeDesc(&d243)
			d245 := d244
			ctx.EnsureDesc(&d245)
			if d245.Loc != scm.LocImm && d245.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			if d245.Loc == scm.LocImm {
				if d245.Imm.Bool() {
					ctx.W.EmitJmp(lbl78)
				} else {
					ctx.W.EmitJmp(lbl79)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d245.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl78)
				ctx.W.EmitJmp(lbl79)
			}
			ctx.W.MarkLabel(lbl78)
			ctx.W.EmitJmp(lbl77)
			ctx.W.MarkLabel(lbl79)
			d246 := d230
			if d246.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d246)
			ctx.EmitStoreToStack(d246, 112)
			ctx.W.EmitJmp(lbl74)
			ctx.FreeDesc(&d244)
			ctx.W.MarkLabel(lbl77)
			d234 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d225)
			var d247 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d225.Imm.Int() / 64)}
			} else {
				r223 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r223, d225.Reg)
				ctx.W.EmitShrRegImm8(r223, 6)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d247)
			}
			if d247.Loc == scm.LocReg && d225.Loc == scm.LocReg && d247.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d247)
			var d248 scm.JITValueDesc
			if d247.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d247.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegReg(scratch, d247.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d248)
			}
			if d248.Loc == scm.LocReg && d247.Loc == scm.LocReg && d248.Reg == d247.Reg {
				ctx.TransferReg(d247.Reg)
				d247.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d247)
			ctx.EnsureDesc(&d248)
			r224 := ctx.AllocReg()
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d226)
			if d248.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r224, uint64(d248.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r224, d248.Reg)
				ctx.W.EmitShlRegImm8(r224, 3)
			}
			if d226.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
				ctx.W.EmitAddInt64(r224, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r224, d226.Reg)
			}
			r225 := ctx.AllocRegExcept(r224)
			ctx.W.EmitMovRegMem(r225, r224, 0)
			ctx.FreeReg(r224)
			d249 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r225}
			ctx.BindReg(r225, &d249)
			ctx.FreeDesc(&d248)
			ctx.EnsureDesc(&d225)
			var d250 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d225.Imm.Int() % 64)}
			} else {
				r226 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r226, d225.Reg)
				ctx.W.EmitAndRegImm32(r226, 63)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d250)
			}
			if d250.Loc == scm.LocReg && d225.Loc == scm.LocReg && d250.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d225)
			d251 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			var d252 scm.JITValueDesc
			if d251.Loc == scm.LocImm && d250.Loc == scm.LocImm {
				d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d251.Imm.Int() - d250.Imm.Int())}
			} else if d250.Loc == scm.LocImm && d250.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(r227, d251.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d252)
			} else if d251.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d250.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d251.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d250.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d252)
			} else if d250.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(scratch, d251.Reg)
				if d250.Imm.Int() >= -2147483648 && d250.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d250.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d252)
			} else {
				r228 := ctx.AllocRegExcept(d251.Reg, d250.Reg)
				ctx.W.EmitMovRegReg(r228, d251.Reg)
				ctx.W.EmitSubInt64(r228, d250.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d252)
			}
			if d252.Loc == scm.LocReg && d251.Loc == scm.LocReg && d252.Reg == d251.Reg {
				ctx.TransferReg(d251.Reg)
				d251.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d250)
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d252)
			var d253 scm.JITValueDesc
			if d249.Loc == scm.LocImm && d252.Loc == scm.LocImm {
				d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d249.Imm.Int()) >> uint64(d252.Imm.Int())))}
			} else if d252.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r229, d249.Reg)
				ctx.W.EmitShrRegImm8(r229, uint8(d252.Imm.Int()))
				d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d253)
			} else {
				{
					shiftSrc := d249.Reg
					r230 := ctx.AllocRegExcept(d249.Reg)
					ctx.W.EmitMovRegReg(r230, d249.Reg)
					shiftSrc = r230
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d252.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d252.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d252.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d253)
				}
			}
			if d253.Loc == scm.LocReg && d249.Loc == scm.LocReg && d253.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			ctx.FreeDesc(&d252)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d253)
			var d254 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d253.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d230.Imm.Int() | d253.Imm.Int())}
			} else if d230.Loc == scm.LocImm && d230.Imm.Int() == 0 {
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d253.Reg}
				ctx.BindReg(d253.Reg, &d254)
			} else if d253.Loc == scm.LocImm && d253.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(r231, d230.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d254)
			} else if d230.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d230.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d253.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d254)
			} else if d253.Loc == scm.LocImm {
				r232 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(r232, d230.Reg)
				if d253.Imm.Int() >= -2147483648 && d253.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r232, int32(d253.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d253.Imm.Int()))
					ctx.W.EmitOrInt64(r232, scm.RegR11)
				}
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d254)
			} else {
				r233 := ctx.AllocRegExcept(d230.Reg, d253.Reg)
				ctx.W.EmitMovRegReg(r233, d230.Reg)
				ctx.W.EmitOrInt64(r233, d253.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d254)
			}
			if d254.Loc == scm.LocReg && d230.Loc == scm.LocReg && d254.Reg == d230.Reg {
				ctx.TransferReg(d230.Reg)
				d230.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d253)
			d255 := d254
			if d255.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d255)
			ctx.EmitStoreToStack(d255, 112)
			ctx.W.EmitJmp(lbl74)
			ctx.W.MarkLabel(lbl71)
			d256 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
			ctx.BindReg(r216, &d256)
			ctx.BindReg(r216, &d256)
			if r196 { ctx.UnprotectReg(r197) }
			ctx.FreeDesc(&d97)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d256)
			var d257 scm.JITValueDesc
			if d256.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d256.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d256.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d257)
			}
			ctx.FreeDesc(&d256)
			var d258 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r235, thisptr.Reg, off)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r235}
				ctx.BindReg(r235, &d258)
			}
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d258)
			var d259 scm.JITValueDesc
			if d257.Loc == scm.LocImm && d258.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d257.Imm.Int() + d258.Imm.Int())}
			} else if d258.Loc == scm.LocImm && d258.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegReg(r236, d257.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d259)
			} else if d257.Loc == scm.LocImm && d257.Imm.Int() == 0 {
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d258.Reg}
				ctx.BindReg(d258.Reg, &d259)
			} else if d257.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d257.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else if d258.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegReg(scratch, d257.Reg)
				if d258.Imm.Int() >= -2147483648 && d258.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d258.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d258.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else {
				r237 := ctx.AllocRegExcept(d257.Reg, d258.Reg)
				ctx.W.EmitMovRegReg(r237, d257.Reg)
				ctx.W.EmitAddInt64(r237, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d259)
			}
			if d259.Loc == scm.LocReg && d257.Loc == scm.LocReg && d259.Reg == d257.Reg {
				ctx.TransferReg(d257.Reg)
				d257.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d257)
			ctx.FreeDesc(&d258)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d260 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d260)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			var d261 scm.JITValueDesc
			if d260.Loc == scm.LocImm && d259.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d260.Imm.Int() - d259.Imm.Int())}
			} else if d259.Loc == scm.LocImm && d259.Imm.Int() == 0 {
				r239 := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(r239, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d261)
			} else if d260.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d260.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d259.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegReg(scratch, d260.Reg)
				if d259.Imm.Int() >= -2147483648 && d259.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d259.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d259.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else {
				r240 := ctx.AllocRegExcept(d260.Reg, d259.Reg)
				ctx.W.EmitMovRegReg(r240, d260.Reg)
				ctx.W.EmitSubInt64(r240, d259.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d261)
			}
			if d261.Loc == scm.LocReg && d260.Loc == scm.LocReg && d261.Reg == d260.Reg {
				ctx.TransferReg(d260.Reg)
				d260.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d260)
			ctx.FreeDesc(&d259)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d220)
			var d262 scm.JITValueDesc
			if d261.Loc == scm.LocImm && d220.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d261.Imm.Int() * d220.Imm.Int())}
			} else if d261.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d261.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d220.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else if d220.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.W.EmitMovRegReg(scratch, d261.Reg)
				if d220.Imm.Int() >= -2147483648 && d220.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d220.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d220.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else {
				r241 := ctx.AllocRegExcept(d261.Reg, d220.Reg)
				ctx.W.EmitMovRegReg(r241, d261.Reg)
				ctx.W.EmitImulInt64(r241, d220.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d262)
			}
			if d262.Loc == scm.LocReg && d261.Loc == scm.LocReg && d262.Reg == d261.Reg {
				ctx.TransferReg(d261.Reg)
				d261.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d261)
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d262)
			var d263 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d262.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() + d262.Imm.Int())}
			} else if d262.Loc == scm.LocImm && d262.Imm.Int() == 0 {
				r242 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r242, d137.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d263)
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d262.Reg}
				ctx.BindReg(d262.Reg, &d263)
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else if d262.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(scratch, d137.Reg)
				if d262.Imm.Int() >= -2147483648 && d262.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d262.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d262.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else {
				r243 := ctx.AllocRegExcept(d137.Reg, d262.Reg)
				ctx.W.EmitMovRegReg(r243, d137.Reg)
				ctx.W.EmitAddInt64(r243, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d263)
			}
			if d263.Loc == scm.LocReg && d137.Loc == scm.LocReg && d263.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d262)
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d263)
			var d264 scm.JITValueDesc
			if d263.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d263.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d263.Reg)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d263.Reg}
				ctx.BindReg(d263.Reg, &d264)
			}
			ctx.FreeDesc(&d263)
			ctx.EnsureDesc(&d264)
			d265 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d265)
			ctx.BindReg(r2, &d265)
			ctx.EnsureDesc(&d264)
			ctx.W.EmitMakeFloat(d265, d264)
			if d264.Loc == scm.LocReg { ctx.FreeReg(d264.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl45)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			var d266 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r244, thisptr.Reg, off)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r244}
				ctx.BindReg(r244, &d266)
			}
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d266)
			var d267 scm.JITValueDesc
			if d266.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d266.Imm.Int()))))}
			} else {
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r245, d266.Reg)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d267)
			}
			ctx.FreeDesc(&d266)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d137.Imm.Int() == d267.Imm.Int())}
			} else if d267.Loc == scm.LocImm {
				r246 := ctx.AllocRegExcept(d137.Reg)
				if d267.Imm.Int() >= -2147483648 && d267.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d137.Reg, int32(d267.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
					ctx.W.EmitCmpInt64(d137.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r246, scm.CcE)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d268)
			} else if d137.Loc == scm.LocImm {
				r247 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d267.Reg)
				ctx.W.EmitSetcc(r247, scm.CcE)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r247}
				ctx.BindReg(r247, &d268)
			} else {
				r248 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitCmpInt64(d137.Reg, d267.Reg)
				ctx.W.EmitSetcc(r248, scm.CcE)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r248}
				ctx.BindReg(r248, &d268)
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d267)
			d269 := d268
			ctx.EnsureDesc(&d269)
			if d269.Loc != scm.LocImm && d269.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl80 := ctx.W.ReserveLabel()
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			if d269.Loc == scm.LocImm {
				if d269.Imm.Bool() {
					ctx.W.EmitJmp(lbl81)
				} else {
					ctx.W.EmitJmp(lbl82)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d269.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl81)
				ctx.W.EmitJmp(lbl82)
			}
			ctx.W.MarkLabel(lbl81)
			ctx.W.EmitJmp(lbl80)
			ctx.W.MarkLabel(lbl82)
			ctx.W.EmitJmp(lbl46)
			ctx.FreeDesc(&d268)
			ctx.W.MarkLabel(lbl59)
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			lbl83 := ctx.W.ReserveLabel()
			d270 := d75
			if d270.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d270)
			d271 := d270
			if d271.Loc == scm.LocImm {
				d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: d271.Type, Imm: scm.NewInt(int64(uint64(d271.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d271.Reg, 32)
				ctx.W.EmitShrRegImm8(d271.Reg, 32)
			}
			ctx.EmitStoreToStack(d271, 64)
			d272 := d77
			if d272.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d272)
			d273 := d272
			if d273.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: d273.Type, Imm: scm.NewInt(int64(uint64(d273.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d273.Reg, 32)
				ctx.W.EmitShrRegImm8(d273.Reg, 32)
			}
			ctx.EmitStoreToStack(d273, 72)
			ctx.W.MarkLabel(lbl83)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d274 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d275 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			var d276 scm.JITValueDesc
			if d274.Loc == scm.LocImm && d275.Loc == scm.LocImm {
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d274.Imm.Int()) == uint64(d275.Imm.Int()))}
			} else if d275.Loc == scm.LocImm {
				r249 := ctx.AllocRegExcept(d274.Reg)
				if d275.Imm.Int() >= -2147483648 && d275.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d274.Reg, int32(d275.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d275.Imm.Int()))
					ctx.W.EmitCmpInt64(d274.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r249, scm.CcE)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r249}
				ctx.BindReg(r249, &d276)
			} else if d274.Loc == scm.LocImm {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d274.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d275.Reg)
				ctx.W.EmitSetcc(r250, scm.CcE)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r250}
				ctx.BindReg(r250, &d276)
			} else {
				r251 := ctx.AllocRegExcept(d274.Reg)
				ctx.W.EmitCmpInt64(d274.Reg, d275.Reg)
				ctx.W.EmitSetcc(r251, scm.CcE)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d276)
			}
			d277 := d276
			ctx.EnsureDesc(&d277)
			if d277.Loc != scm.LocImm && d277.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl84 := ctx.W.ReserveLabel()
			lbl85 := ctx.W.ReserveLabel()
			lbl86 := ctx.W.ReserveLabel()
			if d277.Loc == scm.LocImm {
				if d277.Imm.Bool() {
					ctx.W.EmitJmp(lbl85)
				} else {
					ctx.W.EmitJmp(lbl86)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d277.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl85)
				ctx.W.EmitJmp(lbl86)
			}
			ctx.W.MarkLabel(lbl85)
			d278 := d274
			if d278.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d278)
			d279 := d278
			if d279.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: d279.Type, Imm: scm.NewInt(int64(uint64(d279.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d279.Reg, 32)
				ctx.W.EmitShrRegImm8(d279.Reg, 32)
			}
			ctx.EmitStoreToStack(d279, 32)
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl86)
			ctx.W.EmitJmp(lbl84)
			ctx.FreeDesc(&d276)
			ctx.W.MarkLabel(lbl58)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d274 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d75)
			var d280 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d280 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d75.Imm.Int()) == uint64(0))}
			} else {
				r252 := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitCmpRegImm32(d75.Reg, 0)
				ctx.W.EmitSetcc(r252, scm.CcE)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d280)
			}
			d281 := d280
			ctx.EnsureDesc(&d281)
			if d281.Loc != scm.LocImm && d281.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl87 := ctx.W.ReserveLabel()
			lbl88 := ctx.W.ReserveLabel()
			lbl89 := ctx.W.ReserveLabel()
			lbl90 := ctx.W.ReserveLabel()
			if d281.Loc == scm.LocImm {
				if d281.Imm.Bool() {
					ctx.W.EmitJmp(lbl89)
				} else {
					ctx.W.EmitJmp(lbl90)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d281.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl89)
				ctx.W.EmitJmp(lbl90)
			}
			ctx.W.MarkLabel(lbl89)
			ctx.W.EmitJmp(lbl87)
			ctx.W.MarkLabel(lbl90)
			ctx.W.EmitJmp(lbl88)
			ctx.FreeDesc(&d280)
			ctx.W.MarkLabel(lbl80)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d274 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d282 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d282)
			ctx.BindReg(r2, &d282)
			ctx.W.EmitMakeNil(d282)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl84)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d274 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d275)
			var d283 scm.JITValueDesc
			if d274.Loc == scm.LocImm && d275.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d274.Imm.Int() + d275.Imm.Int())}
			} else if d275.Loc == scm.LocImm && d275.Imm.Int() == 0 {
				r253 := ctx.AllocRegExcept(d274.Reg)
				ctx.W.EmitMovRegReg(r253, d274.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d283)
			} else if d274.Loc == scm.LocImm && d274.Imm.Int() == 0 {
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d275.Reg}
				ctx.BindReg(d275.Reg, &d283)
			} else if d274.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d274.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d275.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d283)
			} else if d275.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d274.Reg)
				ctx.W.EmitMovRegReg(scratch, d274.Reg)
				if d275.Imm.Int() >= -2147483648 && d275.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d275.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d275.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d283)
			} else {
				r254 := ctx.AllocRegExcept(d274.Reg, d275.Reg)
				ctx.W.EmitMovRegReg(r254, d274.Reg)
				ctx.W.EmitAddInt64(r254, d275.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d283)
			}
			if d283.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: d283.Type, Imm: scm.NewInt(int64(uint64(d283.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d283.Reg, 32)
				ctx.W.EmitShrRegImm8(d283.Reg, 32)
			}
			if d283.Loc == scm.LocReg && d274.Loc == scm.LocReg && d283.Reg == d274.Reg {
				ctx.TransferReg(d274.Reg)
				d274.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d283)
			var d284 scm.JITValueDesc
			if d283.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d283.Imm.Int() / 2)}
			} else {
				r255 := ctx.AllocRegExcept(d283.Reg)
				ctx.W.EmitMovRegReg(r255, d283.Reg)
				ctx.W.EmitShrRegImm8(r255, 1)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
				ctx.BindReg(r255, &d284)
			}
			if d284.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: d284.Type, Imm: scm.NewInt(int64(uint64(d284.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d284.Reg, 32)
				ctx.W.EmitShrRegImm8(d284.Reg, 32)
			}
			if d284.Loc == scm.LocReg && d283.Loc == scm.LocReg && d284.Reg == d283.Reg {
				ctx.TransferReg(d283.Reg)
				d283.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d283)
			d285 := d284
			if d285.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d285)
			d286 := d285
			if d286.Loc == scm.LocImm {
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: d286.Type, Imm: scm.NewInt(int64(uint64(d286.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d286.Reg, 32)
				ctx.W.EmitShrRegImm8(d286.Reg, 32)
			}
			ctx.EmitStoreToStack(d286, 8)
			d287 := d274
			if d287.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d287)
			d288 := d287
			if d288.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: d288.Type, Imm: scm.NewInt(int64(uint64(d288.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d288.Reg, 32)
				ctx.W.EmitShrRegImm8(d288.Reg, 32)
			}
			ctx.EmitStoreToStack(d288, 16)
			d289 := d275
			if d289.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d289)
			d290 := d289
			if d290.Loc == scm.LocImm {
				d290 = scm.JITValueDesc{Loc: scm.LocImm, Type: d290.Type, Imm: scm.NewInt(int64(uint64(d290.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d290.Reg, 32)
				ctx.W.EmitShrRegImm8(d290.Reg, 32)
			}
			ctx.EmitStoreToStack(d290, 24)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl88)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d274 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d75)
			var d291 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(scratch, d75.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d291)
			}
			if d291.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: d291.Type, Imm: scm.NewInt(int64(uint64(d291.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d291.Reg, 32)
				ctx.W.EmitShrRegImm8(d291.Reg, 32)
			}
			if d291.Loc == scm.LocReg && d75.Loc == scm.LocReg && d291.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			d292 := d76
			if d292.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d292)
			d293 := d292
			if d293.Loc == scm.LocImm {
				d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: d293.Type, Imm: scm.NewInt(int64(uint64(d293.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d293.Reg, 32)
				ctx.W.EmitShrRegImm8(d293.Reg, 32)
			}
			ctx.EmitStoreToStack(d293, 64)
			d294 := d291
			if d294.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d294)
			d295 := d294
			if d295.Loc == scm.LocImm {
				d295 = scm.JITValueDesc{Loc: scm.LocImm, Type: d295.Type, Imm: scm.NewInt(int64(uint64(d295.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d295.Reg, 32)
				ctx.W.EmitShrRegImm8(d295.Reg, 32)
			}
			ctx.EmitStoreToStack(d295, 72)
			ctx.W.EmitJmp(lbl83)
			ctx.W.MarkLabel(lbl87)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d75 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d77 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d76 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d274 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl0)
			d296 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d296)
			ctx.BindReg(r2, &d296)
			ctx.EmitMovPairToResult(&d296, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
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
