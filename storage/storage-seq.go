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
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			lbl0 := ctx.W.ReserveLabel()
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
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d2)
			var d4 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d1.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r8 := ctx.AllocRegExcept(d1.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r8, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d4)
			} else if d1.Loc == scm.LocImm {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r9, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r9}
				ctx.BindReg(r9, &d4)
			} else {
				r10 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d2.Reg)
				ctx.W.EmitSetcc(r10, scm.CcAE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
				ctx.BindReg(r10, &d4)
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
			ctx.EnsureDesc(&d5)
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
			ctx.EnsureDesc(&d7)
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
			d9 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d9)
			ctx.BindReg(r2, &d9)
			ctx.W.EmitMakeNil(d9)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			d10 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
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
			lbl7 := ctx.W.ReserveLabel()
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
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl7)
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
			lbl8 := ctx.W.ReserveLabel()
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
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d29.Loc == scm.LocImm {
				if d29.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d30 := d28
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, 80)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d29.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
			d31 := d28
			if d31.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d31)
			ctx.EmitStoreToStack(d31, 80)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d29)
			ctx.W.MarkLabel(lbl10)
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
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl9)
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
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d42.Loc == scm.LocImm {
				if d42.Imm.Bool() {
					ctx.W.EmitJmp(lbl12)
				} else {
			d43 := d28
			if d43.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, 80)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d42.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
			d44 := d28
			if d44.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d44)
			ctx.EmitStoreToStack(d44, 80)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl12)
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
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl8)
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
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d59.Loc == scm.LocImm {
				if d59.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d59.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d59)
			ctx.W.MarkLabel(lbl4)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d2)
			var d60 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			}
			if d60.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: d60.Type, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d60.Reg, 32)
				ctx.W.EmitShrRegImm8(d60.Reg, 32)
			}
			if d60.Loc == scm.LocReg && d2.Loc == scm.LocReg && d60.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d61 := d60
			if d61.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			d62 := d61
			if d62.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: d62.Type, Imm: scm.NewInt(int64(uint64(d62.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d62.Reg, 32)
				ctx.W.EmitShrRegImm8(d62.Reg, 32)
			}
			ctx.EmitStoreToStack(d62, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl15)
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d16)
			var d63 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d63)
			}
			if d63.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: d63.Type, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d63.Reg, 32)
				ctx.W.EmitShrRegImm8(d63.Reg, 32)
			}
			if d63.Loc == scm.LocReg && d16.Loc == scm.LocReg && d63.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d2)
			var d64 scm.JITValueDesc
			if d63.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d63.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d63.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d63.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d63.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r57, scm.CcAE)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d64)
			} else if d63.Loc == scm.LocImm {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r58, scm.CcAE)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d64)
			} else {
				r59 := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitCmpInt64(d63.Reg, d2.Reg)
				ctx.W.EmitSetcc(r59, scm.CcAE)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d64)
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d64.Loc == scm.LocImm {
				if d64.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
			d65 := d63
			if d65.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d65)
			d66 := d65
			if d66.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: d66.Type, Imm: scm.NewInt(int64(uint64(d66.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d66.Reg, 32)
				ctx.W.EmitShrRegImm8(d66.Reg, 32)
			}
			ctx.EmitStoreToStack(d66, 40)
			d67 := d16
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d67)
			d68 := d67
			if d68.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: d68.Type, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d68.Reg, 32)
				ctx.W.EmitShrRegImm8(d68.Reg, 32)
			}
			ctx.EmitStoreToStack(d68, 48)
			d69 := d18
			if d69.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d69)
			d70 := d69
			if d70.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: d70.Type, Imm: scm.NewInt(int64(uint64(d70.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d70.Reg, 32)
				ctx.W.EmitShrRegImm8(d70.Reg, 32)
			}
			ctx.EmitStoreToStack(d70, 56)
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d64.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
			d71 := d63
			if d71.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d71)
			d72 := d71
			if d72.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: d72.Type, Imm: scm.NewInt(int64(uint64(d72.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d72.Reg, 32)
				ctx.W.EmitShrRegImm8(d72.Reg, 32)
			}
			ctx.EmitStoreToStack(d72, 40)
			d73 := d16
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			d74 := d73
			if d74.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: d74.Type, Imm: scm.NewInt(int64(uint64(d74.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d74.Reg, 32)
				ctx.W.EmitShrRegImm8(d74.Reg, 32)
			}
			ctx.EmitStoreToStack(d74, 48)
			d75 := d18
			if d75.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d75)
			d76 := d75
			if d76.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: d76.Type, Imm: scm.NewInt(int64(uint64(d76.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d76.Reg, 32)
				ctx.W.EmitShrRegImm8(d76.Reg, 32)
			}
			ctx.EmitStoreToStack(d76, 56)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d64)
			ctx.W.MarkLabel(lbl14)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d16)
			var d77 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d16.Imm.Int()) == uint64(0))}
			} else {
				r60 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitSetcc(r60, scm.CcE)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d77)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d77.Loc == scm.LocImm {
				if d77.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d77.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d77)
			ctx.W.MarkLabel(lbl18)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d78 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d79 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d80 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) == uint64(d80.Imm.Int()))}
			} else if d80.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d79.Reg)
				if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d79.Reg, int32(d80.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
					ctx.W.EmitCmpInt64(d79.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r61, scm.CcE)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d81)
			} else if d79.Loc == scm.LocImm {
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d80.Reg)
				ctx.W.EmitSetcc(r62, scm.CcE)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
				ctx.BindReg(r62, &d81)
			} else {
				r63 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitCmpInt64(d79.Reg, d80.Reg)
				ctx.W.EmitSetcc(r63, scm.CcE)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r63}
				ctx.BindReg(r63, &d81)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d81.Loc == scm.LocImm {
				if d81.Imm.Bool() {
			d82 := d79
			if d82.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d82)
			d83 := d82
			if d83.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: d83.Type, Imm: scm.NewInt(int64(uint64(d83.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d83.Reg, 32)
				ctx.W.EmitShrRegImm8(d83.Reg, 32)
			}
			ctx.EmitStoreToStack(d83, 32)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d81.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
			d84 := d79
			if d84.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d84)
			d85 := d84
			if d85.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: d85.Type, Imm: scm.NewInt(int64(uint64(d85.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d85.Reg, 32)
				ctx.W.EmitShrRegImm8(d85.Reg, 32)
			}
			ctx.EmitStoreToStack(d85, 32)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d81)
			ctx.W.MarkLabel(lbl17)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d2)
			var d86 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d86)
			}
			if d86.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: d86.Type, Imm: scm.NewInt(int64(uint64(d86.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d86.Reg, 32)
				ctx.W.EmitShrRegImm8(d86.Reg, 32)
			}
			if d86.Loc == scm.LocReg && d2.Loc == scm.LocReg && d86.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d87 := d86
			if d87.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d87)
			d88 := d87
			if d88.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: d88.Type, Imm: scm.NewInt(int64(uint64(d88.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d88.Reg, 32)
				ctx.W.EmitShrRegImm8(d88.Reg, 32)
			}
			ctx.EmitStoreToStack(d88, 40)
			d89 := d16
			if d89.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d89)
			d90 := d89
			if d90.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: d90.Type, Imm: scm.NewInt(int64(uint64(d90.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d90.Reg, 32)
				ctx.W.EmitShrRegImm8(d90.Reg, 32)
			}
			ctx.EmitStoreToStack(d90, 48)
			d91 := d18
			if d91.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d91)
			d92 := d91
			if d92.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: d92.Type, Imm: scm.NewInt(int64(uint64(d92.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d92.Reg, 32)
				ctx.W.EmitShrRegImm8(d92.Reg, 32)
			}
			ctx.EmitStoreToStack(d92, 56)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl21)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d16)
			var d93 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			}
			if d93.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: d93.Type, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d93.Reg, 32)
				ctx.W.EmitShrRegImm8(d93.Reg, 32)
			}
			if d93.Loc == scm.LocReg && d16.Loc == scm.LocReg && d93.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d16)
			var d94 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d94)
			}
			if d94.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: d94.Type, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d94.Reg, 32)
				ctx.W.EmitShrRegImm8(d94.Reg, 32)
			}
			if d94.Loc == scm.LocReg && d16.Loc == scm.LocReg && d94.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			d95 := d94
			if d95.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d95)
			d96 := d95
			if d96.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: d96.Type, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d96.Reg, 32)
				ctx.W.EmitShrRegImm8(d96.Reg, 32)
			}
			ctx.EmitStoreToStack(d96, 40)
			d97 := d17
			if d97.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d97)
			d98 := d97
			if d98.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: d98.Type, Imm: scm.NewInt(int64(uint64(d98.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d98.Reg, 32)
				ctx.W.EmitShrRegImm8(d98.Reg, 32)
			}
			ctx.EmitStoreToStack(d98, 48)
			d99 := d93
			if d99.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d99)
			d100 := d99
			if d100.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: d100.Type, Imm: scm.NewInt(int64(uint64(d100.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d100.Reg, 32)
				ctx.W.EmitShrRegImm8(d100.Reg, 32)
			}
			ctx.EmitStoreToStack(d100, 56)
			ctx.W.EmitJmp(lbl18)
			ctx.W.MarkLabel(lbl20)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl23)
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d101 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d101)
			var d102 scm.JITValueDesc
			if d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d101.Imm.Int()))))}
			} else {
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r64, d101.Reg)
				ctx.W.EmitShlRegImm8(r64, 32)
				ctx.W.EmitShrRegImm8(r64, 32)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d102)
			}
			ctx.EnsureDesc(&d102)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d102.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d102.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d102.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d102.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d102.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d102.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d102)
			ctx.EnsureDesc(&d101)
			d103 := d101
			_ = d103
			r65 := d101.Loc == scm.LocReg
			r66 := d101.Reg
			if r65 { ctx.ProtectReg(r66) }
			lbl26 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d103.Imm.Int()))))}
			} else {
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r67, d103.Reg)
				ctx.W.EmitShlRegImm8(r67, 32)
				ctx.W.EmitShrRegImm8(r67, 32)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d104)
			}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
				ctx.BindReg(r68, &d105)
			}
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d105)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r69, d105.Reg)
				ctx.W.EmitShlRegImm8(r69, 56)
				ctx.W.EmitShrRegImm8(r69, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d106)
			}
			ctx.FreeDesc(&d105)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() * d106.Imm.Int())}
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			} else if d106.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(scratch, d104.Reg)
				if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d106.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d106.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			} else {
				r70 := ctx.AllocRegExcept(d104.Reg, d106.Reg)
				ctx.W.EmitMovRegReg(r70, d104.Reg)
				ctx.W.EmitImulInt64(r70, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d107)
			}
			if d107.Loc == scm.LocReg && d104.Loc == scm.LocReg && d107.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.FreeDesc(&d106)
			var d108 scm.JITValueDesc
			r71 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r71, uint64(dataPtr))
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71, StackOff: int32(sliceLen)}
				ctx.BindReg(r71, &d108)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r71, thisptr.Reg, off)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
				ctx.BindReg(r71, &d108)
			}
			ctx.BindReg(r71, &d108)
			ctx.EnsureDesc(&d107)
			var d109 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r72 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r72, d107.Reg)
				ctx.W.EmitShrRegImm8(r72, 6)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d109)
			}
			if d109.Loc == scm.LocReg && d107.Loc == scm.LocReg && d109.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d109)
			r73 := ctx.AllocReg()
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d108)
			if d109.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r73, uint64(d109.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r73, d109.Reg)
				ctx.W.EmitShlRegImm8(r73, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r73, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r73, d108.Reg)
			}
			r74 := ctx.AllocRegExcept(r73)
			ctx.W.EmitMovRegMem(r74, r73, 0)
			ctx.FreeReg(r73)
			d110 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.BindReg(r74, &d110)
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d107)
			var d111 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r75 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r75, d107.Reg)
				ctx.W.EmitAndRegImm32(r75, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d111)
			}
			if d111.Loc == scm.LocReg && d107.Loc == scm.LocReg && d111.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d110.Imm.Int()) << uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				r76 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r76, d110.Reg)
				ctx.W.EmitShlRegImm8(r76, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d112)
			} else {
				{
					shiftSrc := d110.Reg
					r77 := ctx.AllocRegExcept(d110.Reg)
					ctx.W.EmitMovRegReg(r77, d110.Reg)
					shiftSrc = r77
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d111.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d111.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d111.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d112)
				}
			}
			if d112.Loc == scm.LocReg && d110.Loc == scm.LocReg && d112.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d110)
			ctx.FreeDesc(&d111)
			var d113 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r78, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
				ctx.BindReg(r78, &d113)
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
			d114 := d112
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, 88)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d113.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl29)
			d115 := d112
			if d115.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d115)
			ctx.EmitStoreToStack(d115, 88)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl29)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d113)
			ctx.W.MarkLabel(lbl28)
			d116 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d117 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r79 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r79, thisptr.Reg, off)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
				ctx.BindReg(r79, &d117)
			}
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d117)
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d117.Imm.Int()))))}
			} else {
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r80, d117.Reg)
				ctx.W.EmitShlRegImm8(r80, 56)
				ctx.W.EmitShrRegImm8(r80, 56)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d118)
			}
			ctx.FreeDesc(&d117)
			d119 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d118)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() - d118.Imm.Int())}
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r81, d119.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d120)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d118.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d118.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			} else {
				r82 := ctx.AllocRegExcept(d119.Reg, d118.Reg)
				ctx.W.EmitMovRegReg(r82, d119.Reg)
				ctx.W.EmitSubInt64(r82, d118.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d120)
			}
			if d120.Loc == scm.LocReg && d119.Loc == scm.LocReg && d120.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d120)
			var d121 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d116.Imm.Int()) >> uint64(d120.Imm.Int())))}
			} else if d120.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r83, d116.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d120.Imm.Int()))
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d121)
			} else {
				{
					shiftSrc := d116.Reg
					r84 := ctx.AllocRegExcept(d116.Reg)
					ctx.W.EmitMovRegReg(r84, d116.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d120.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d120.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d120.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d121)
				}
			}
			if d121.Loc == scm.LocReg && d116.Loc == scm.LocReg && d121.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.FreeDesc(&d120)
			r85 := ctx.AllocReg()
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d121)
			if d121.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r85, d121)
			}
			ctx.W.EmitJmp(lbl26)
			ctx.W.MarkLabel(lbl27)
			d116 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d107)
			var d122 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r86 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r86, d107.Reg)
				ctx.W.EmitAndRegImm32(r86, 63)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d122)
			}
			if d122.Loc == scm.LocReg && d107.Loc == scm.LocReg && d122.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			var d123 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r87, thisptr.Reg, off)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d123)
			}
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d123)
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d123.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d123.Reg)
				ctx.W.EmitShlRegImm8(r88, 56)
				ctx.W.EmitShrRegImm8(r88, 56)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d124)
			}
			ctx.FreeDesc(&d123)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d122.Loc == scm.LocImm && d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() + d124.Imm.Int())}
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				r89 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r89, d122.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d125)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
				ctx.BindReg(d124.Reg, &d125)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(scratch, d122.Reg)
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else {
				r90 := ctx.AllocRegExcept(d122.Reg, d124.Reg)
				ctx.W.EmitMovRegReg(r90, d122.Reg)
				ctx.W.EmitAddInt64(r90, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d125)
			}
			if d125.Loc == scm.LocReg && d122.Loc == scm.LocReg && d125.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			ctx.FreeDesc(&d124)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d125.Imm.Int()) > uint64(64))}
			} else {
				r91 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitCmpRegImm32(d125.Reg, 64)
				ctx.W.EmitSetcc(r91, scm.CcA)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r91}
				ctx.BindReg(r91, &d126)
			}
			ctx.FreeDesc(&d125)
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d126.Loc == scm.LocImm {
				if d126.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
			d127 := d112
			if d127.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d127)
			ctx.EmitStoreToStack(d127, 88)
					ctx.W.EmitJmp(lbl28)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d126.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
			d128 := d112
			if d128.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d128)
			ctx.EmitStoreToStack(d128, 88)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d126)
			ctx.W.MarkLabel(lbl30)
			d116 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d107)
			var d129 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r92 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r92, d107.Reg)
				ctx.W.EmitShrRegImm8(r92, 6)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d129)
			}
			if d129.Loc == scm.LocReg && d107.Loc == scm.LocReg && d129.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d129)
			var d130 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(scratch, d129.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			}
			if d130.Loc == scm.LocReg && d129.Loc == scm.LocReg && d130.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d129)
			ctx.EnsureDesc(&d130)
			r93 := ctx.AllocReg()
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d108)
			if d130.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r93, uint64(d130.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r93, d130.Reg)
				ctx.W.EmitShlRegImm8(r93, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r93, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r93, d108.Reg)
			}
			r94 := ctx.AllocRegExcept(r93)
			ctx.W.EmitMovRegMem(r94, r93, 0)
			ctx.FreeReg(r93)
			d131 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r94}
			ctx.BindReg(r94, &d131)
			ctx.FreeDesc(&d130)
			ctx.EnsureDesc(&d107)
			var d132 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r95 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r95, d107.Reg)
				ctx.W.EmitAndRegImm32(r95, 63)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d132)
			}
			if d132.Loc == scm.LocReg && d107.Loc == scm.LocReg && d132.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			d133 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d132)
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm && d132.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() - d132.Imm.Int())}
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				r96 := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegReg(r96, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d134)
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d132.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d134)
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegReg(scratch, d133.Reg)
				if d132.Imm.Int() >= -2147483648 && d132.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d132.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d132.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d134)
			} else {
				r97 := ctx.AllocRegExcept(d133.Reg, d132.Reg)
				ctx.W.EmitMovRegReg(r97, d133.Reg)
				ctx.W.EmitSubInt64(r97, d132.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d134)
			}
			if d134.Loc == scm.LocReg && d133.Loc == scm.LocReg && d134.Reg == d133.Reg {
				ctx.TransferReg(d133.Reg)
				d133.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d131.Imm.Int()) >> uint64(d134.Imm.Int())))}
			} else if d134.Loc == scm.LocImm {
				r98 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r98, d131.Reg)
				ctx.W.EmitShrRegImm8(r98, uint8(d134.Imm.Int()))
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d135)
			} else {
				{
					shiftSrc := d131.Reg
					r99 := ctx.AllocRegExcept(d131.Reg)
					ctx.W.EmitMovRegReg(r99, d131.Reg)
					shiftSrc = r99
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
					ctx.BindReg(shiftSrc, &d135)
				}
			}
			if d135.Loc == scm.LocReg && d131.Loc == scm.LocReg && d135.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.FreeDesc(&d134)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d135)
			var d136 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() | d135.Imm.Int())}
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
				ctx.BindReg(d135.Reg, &d136)
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r100 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r100, d112.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d136)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d136)
			} else if d135.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r101, d112.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r101, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitOrInt64(r101, scm.RegR11)
				}
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d136)
			} else {
				r102 := ctx.AllocRegExcept(d112.Reg, d135.Reg)
				ctx.W.EmitMovRegReg(r102, d112.Reg)
				ctx.W.EmitOrInt64(r102, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d136)
			}
			if d136.Loc == scm.LocReg && d112.Loc == scm.LocReg && d136.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			d137 := d136
			if d137.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d137)
			ctx.EmitStoreToStack(d137, 88)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl26)
			d138 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.BindReg(r85, &d138)
			ctx.BindReg(r85, &d138)
			if r65 { ctx.UnprotectReg(r66) }
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d138)
			var d139 scm.JITValueDesc
			if d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d138.Imm.Int()))))}
			} else {
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r103, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d139)
			}
			ctx.FreeDesc(&d138)
			var d140 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r104, thisptr.Reg, off)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
				ctx.BindReg(r104, &d140)
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + d140.Imm.Int())}
			} else if d140.Loc == scm.LocImm && d140.Imm.Int() == 0 {
				r105 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r105, d139.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d141)
			} else if d139.Loc == scm.LocImm && d139.Imm.Int() == 0 {
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d140.Reg}
				ctx.BindReg(d140.Reg, &d141)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d139.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(scratch, d139.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d140.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d141)
			} else {
				r106 := ctx.AllocRegExcept(d139.Reg, d140.Reg)
				ctx.W.EmitMovRegReg(r106, d139.Reg)
				ctx.W.EmitAddInt64(r106, d140.Reg)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d141)
			}
			if d141.Loc == scm.LocReg && d139.Loc == scm.LocReg && d141.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			ctx.FreeDesc(&d140)
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d142)
			}
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d142.Loc == scm.LocImm {
				if d142.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d142.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d142)
			ctx.W.MarkLabel(lbl24)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d78)
			d143 := d78
			_ = d143
			r108 := d78.Loc == scm.LocReg
			r109 := d78.Reg
			if r108 { ctx.ProtectReg(r109) }
			lbl35 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d143.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d143.Reg)
				ctx.W.EmitShlRegImm8(r110, 32)
				ctx.W.EmitShrRegImm8(r110, 32)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d144)
			}
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r111, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r111}
				ctx.BindReg(r111, &d145)
			}
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d145.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d145.Reg)
				ctx.W.EmitShlRegImm8(r112, 56)
				ctx.W.EmitShrRegImm8(r112, 56)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d146)
			}
			ctx.FreeDesc(&d145)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d146)
			var d147 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d146.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() * d146.Imm.Int())}
			} else if d144.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d144.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d146.Reg)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d147)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(scratch, d144.Reg)
				if d146.Imm.Int() >= -2147483648 && d146.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d146.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d146.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d147)
			} else {
				r113 := ctx.AllocRegExcept(d144.Reg, d146.Reg)
				ctx.W.EmitMovRegReg(r113, d144.Reg)
				ctx.W.EmitImulInt64(r113, d146.Reg)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d147)
			}
			if d147.Loc == scm.LocReg && d144.Loc == scm.LocReg && d147.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d146)
			var d148 scm.JITValueDesc
			r114 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r114, uint64(dataPtr))
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114, StackOff: int32(sliceLen)}
				ctx.BindReg(r114, &d148)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r114, thisptr.Reg, off)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d148)
			}
			ctx.BindReg(r114, &d148)
			ctx.EnsureDesc(&d147)
			var d149 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() / 64)}
			} else {
				r115 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r115, d147.Reg)
				ctx.W.EmitShrRegImm8(r115, 6)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d149)
			}
			if d149.Loc == scm.LocReg && d147.Loc == scm.LocReg && d149.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d149)
			r116 := ctx.AllocReg()
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d148)
			if d149.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r116, uint64(d149.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r116, d149.Reg)
				ctx.W.EmitShlRegImm8(r116, 3)
			}
			if d148.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
				ctx.W.EmitAddInt64(r116, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r116, d148.Reg)
			}
			r117 := ctx.AllocRegExcept(r116)
			ctx.W.EmitMovRegMem(r117, r116, 0)
			ctx.FreeReg(r116)
			d150 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
			ctx.BindReg(r117, &d150)
			ctx.FreeDesc(&d149)
			ctx.EnsureDesc(&d147)
			var d151 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() % 64)}
			} else {
				r118 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r118, d147.Reg)
				ctx.W.EmitAndRegImm32(r118, 63)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d151)
			}
			if d151.Loc == scm.LocReg && d147.Loc == scm.LocReg && d151.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d150.Imm.Int()) << uint64(d151.Imm.Int())))}
			} else if d151.Loc == scm.LocImm {
				r119 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r119, d150.Reg)
				ctx.W.EmitShlRegImm8(r119, uint8(d151.Imm.Int()))
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d152)
			} else {
				{
					shiftSrc := d150.Reg
					r120 := ctx.AllocRegExcept(d150.Reg)
					ctx.W.EmitMovRegReg(r120, d150.Reg)
					shiftSrc = r120
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d151.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d151.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d151.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d152)
				}
			}
			if d152.Loc == scm.LocReg && d150.Loc == scm.LocReg && d152.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d151)
			var d153 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
				ctx.BindReg(r121, &d153)
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			if d153.Loc == scm.LocImm {
				if d153.Imm.Bool() {
					ctx.W.EmitJmp(lbl36)
				} else {
			d154 := d152
			if d154.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d154)
			ctx.EmitStoreToStack(d154, 96)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d153.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl38)
			d155 := d152
			if d155.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d155)
			ctx.EmitStoreToStack(d155, 96)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl36)
			}
			ctx.FreeDesc(&d153)
			ctx.W.MarkLabel(lbl37)
			d156 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r122, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
				ctx.BindReg(r122, &d157)
			}
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r123, d157.Reg)
				ctx.W.EmitShlRegImm8(r123, 56)
				ctx.W.EmitShrRegImm8(r123, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d158)
			}
			ctx.FreeDesc(&d157)
			d159 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d158)
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() - d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(r124, d159.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d160)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(scratch, d159.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else {
				r125 := ctx.AllocRegExcept(d159.Reg, d158.Reg)
				ctx.W.EmitMovRegReg(r125, d159.Reg)
				ctx.W.EmitSubInt64(r125, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d160)
			}
			if d160.Loc == scm.LocReg && d159.Loc == scm.LocReg && d160.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d160)
			var d161 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d156.Imm.Int()) >> uint64(d160.Imm.Int())))}
			} else if d160.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r126, d156.Reg)
				ctx.W.EmitShrRegImm8(r126, uint8(d160.Imm.Int()))
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d161)
			} else {
				{
					shiftSrc := d156.Reg
					r127 := ctx.AllocRegExcept(d156.Reg)
					ctx.W.EmitMovRegReg(r127, d156.Reg)
					shiftSrc = r127
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
					ctx.BindReg(shiftSrc, &d161)
				}
			}
			if d161.Loc == scm.LocReg && d156.Loc == scm.LocReg && d161.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d160)
			r128 := ctx.AllocReg()
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d161)
			if d161.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r128, d161)
			}
			ctx.W.EmitJmp(lbl35)
			ctx.W.MarkLabel(lbl36)
			d156 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d147)
			var d162 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() % 64)}
			} else {
				r129 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r129, d147.Reg)
				ctx.W.EmitAndRegImm32(r129, 63)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d162)
			}
			if d162.Loc == scm.LocReg && d147.Loc == scm.LocReg && d162.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			var d163 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r130, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r130}
				ctx.BindReg(r130, &d163)
			}
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d163)
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d163.Imm.Int()))))}
			} else {
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r131, d163.Reg)
				ctx.W.EmitShlRegImm8(r131, 56)
				ctx.W.EmitShrRegImm8(r131, 56)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d164)
			}
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(r132, d162.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d165)
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
				ctx.BindReg(d164.Reg, &d165)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(scratch, d162.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d164.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else {
				r133 := ctx.AllocRegExcept(d162.Reg, d164.Reg)
				ctx.W.EmitMovRegReg(r133, d162.Reg)
				ctx.W.EmitAddInt64(r133, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d165)
			}
			if d165.Loc == scm.LocReg && d162.Loc == scm.LocReg && d165.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d165)
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d165.Imm.Int()) > uint64(64))}
			} else {
				r134 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitCmpRegImm32(d165.Reg, 64)
				ctx.W.EmitSetcc(r134, scm.CcA)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r134}
				ctx.BindReg(r134, &d166)
			}
			ctx.FreeDesc(&d165)
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d166.Loc == scm.LocImm {
				if d166.Imm.Bool() {
					ctx.W.EmitJmp(lbl39)
				} else {
			d167 := d152
			if d167.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d167)
			ctx.EmitStoreToStack(d167, 96)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d166.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
			d168 := d152
			if d168.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d168)
			ctx.EmitStoreToStack(d168, 96)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl39)
			}
			ctx.FreeDesc(&d166)
			ctx.W.MarkLabel(lbl39)
			d156 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d147)
			var d169 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() / 64)}
			} else {
				r135 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r135, d147.Reg)
				ctx.W.EmitShrRegImm8(r135, 6)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d169)
			}
			if d169.Loc == scm.LocReg && d147.Loc == scm.LocReg && d169.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(scratch, d169.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			}
			if d170.Loc == scm.LocReg && d169.Loc == scm.LocReg && d170.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d170)
			r136 := ctx.AllocReg()
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d148)
			if d170.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r136, uint64(d170.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r136, d170.Reg)
				ctx.W.EmitShlRegImm8(r136, 3)
			}
			if d148.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
				ctx.W.EmitAddInt64(r136, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r136, d148.Reg)
			}
			r137 := ctx.AllocRegExcept(r136)
			ctx.W.EmitMovRegMem(r137, r136, 0)
			ctx.FreeReg(r136)
			d171 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d171)
			ctx.FreeDesc(&d170)
			ctx.EnsureDesc(&d147)
			var d172 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() % 64)}
			} else {
				r138 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r138, d147.Reg)
				ctx.W.EmitAndRegImm32(r138, 63)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d172)
			}
			if d172.Loc == scm.LocReg && d147.Loc == scm.LocReg && d172.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			d173 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d172)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() - d172.Imm.Int())}
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				r139 := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(r139, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d174)
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d174)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(scratch, d173.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d174)
			} else {
				r140 := ctx.AllocRegExcept(d173.Reg, d172.Reg)
				ctx.W.EmitMovRegReg(r140, d173.Reg)
				ctx.W.EmitSubInt64(r140, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d174)
			}
			if d174.Loc == scm.LocReg && d173.Loc == scm.LocReg && d174.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d174)
			var d175 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d171.Imm.Int()) >> uint64(d174.Imm.Int())))}
			} else if d174.Loc == scm.LocImm {
				r141 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r141, d171.Reg)
				ctx.W.EmitShrRegImm8(r141, uint8(d174.Imm.Int()))
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d175)
			} else {
				{
					shiftSrc := d171.Reg
					r142 := ctx.AllocRegExcept(d171.Reg)
					ctx.W.EmitMovRegReg(r142, d171.Reg)
					shiftSrc = r142
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d174.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d174.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d174.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d175)
				}
			}
			if d175.Loc == scm.LocReg && d171.Loc == scm.LocReg && d175.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			ctx.FreeDesc(&d174)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() | d175.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
				ctx.BindReg(d175.Reg, &d176)
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				r143 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r143, d152.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d176)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d152.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				r144 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r144, d152.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r144, int32(d175.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.W.EmitOrInt64(r144, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d176)
			} else {
				r145 := ctx.AllocRegExcept(d152.Reg, d175.Reg)
				ctx.W.EmitMovRegReg(r145, d152.Reg)
				ctx.W.EmitOrInt64(r145, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d176)
			}
			if d176.Loc == scm.LocReg && d152.Loc == scm.LocReg && d176.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			d177 := d176
			if d177.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d177)
			ctx.EmitStoreToStack(d177, 96)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl35)
			d178 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			ctx.BindReg(r128, &d178)
			ctx.BindReg(r128, &d178)
			if r108 { ctx.UnprotectReg(r109) }
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d178)
			var d179 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d178.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d179)
			}
			ctx.FreeDesc(&d178)
			var d180 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r147, thisptr.Reg, off)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d180)
			}
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d180)
			var d181 scm.JITValueDesc
			if d179.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() + d180.Imm.Int())}
			} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
				r148 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r148, d179.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d181)
			} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d180.Reg}
				ctx.BindReg(d180.Reg, &d181)
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d179.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d180.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d181)
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(scratch, d179.Reg)
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d181)
			} else {
				r149 := ctx.AllocRegExcept(d179.Reg, d180.Reg)
				ctx.W.EmitMovRegReg(r149, d179.Reg)
				ctx.W.EmitAddInt64(r149, d180.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d181)
			}
			if d181.Loc == scm.LocReg && d179.Loc == scm.LocReg && d181.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.FreeDesc(&d180)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d181)
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d181.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d181.Reg)
				ctx.W.EmitShlRegImm8(r150, 32)
				ctx.W.EmitShrRegImm8(r150, 32)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d182)
			}
			ctx.FreeDesc(&d181)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d182.Imm.Int()))}
			} else if d182.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(idxInt.Reg)
				if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d182.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r151, scm.CcB)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d183)
			} else if idxInt.Loc == scm.LocImm {
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d182.Reg)
				ctx.W.EmitSetcc(r152, scm.CcB)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d183)
			} else {
				r153 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d182.Reg)
				ctx.W.EmitSetcc(r153, scm.CcB)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
				ctx.BindReg(r153, &d183)
			}
			ctx.FreeDesc(&d182)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d183.Loc == scm.LocImm {
				if d183.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d183.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d183)
			ctx.W.MarkLabel(lbl33)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d101)
			d184 := d101
			_ = d184
			r154 := d101.Loc == scm.LocReg
			r155 := d101.Reg
			if r154 { ctx.ProtectReg(r155) }
			lbl44 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d184.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, d184.Reg)
				ctx.W.EmitShlRegImm8(r156, 32)
				ctx.W.EmitShrRegImm8(r156, 32)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d185)
			}
			var d186 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r157, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d186)
			}
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d186)
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d186.Imm.Int()))))}
			} else {
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r158, d186.Reg)
				ctx.W.EmitShlRegImm8(r158, 56)
				ctx.W.EmitShrRegImm8(r158, 56)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d187)
			}
			ctx.FreeDesc(&d186)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			var d188 scm.JITValueDesc
			if d185.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() * d187.Imm.Int())}
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d185.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(scratch, d185.Reg)
				if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else {
				r159 := ctx.AllocRegExcept(d185.Reg, d187.Reg)
				ctx.W.EmitMovRegReg(r159, d185.Reg)
				ctx.W.EmitImulInt64(r159, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d188)
			}
			if d188.Loc == scm.LocReg && d185.Loc == scm.LocReg && d188.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			ctx.FreeDesc(&d187)
			var d189 scm.JITValueDesc
			r160 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r160, uint64(dataPtr))
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160, StackOff: int32(sliceLen)}
				ctx.BindReg(r160, &d189)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r160, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d189)
			}
			ctx.BindReg(r160, &d189)
			ctx.EnsureDesc(&d188)
			var d190 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r161, d188.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d190)
			}
			if d190.Loc == scm.LocReg && d188.Loc == scm.LocReg && d190.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d190)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d189)
			if d190.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d190.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d190.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d189.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d189.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d189.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d191 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d191)
			ctx.FreeDesc(&d190)
			ctx.EnsureDesc(&d188)
			var d192 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r164, d188.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d192)
			}
			if d192.Loc == scm.LocReg && d188.Loc == scm.LocReg && d192.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d192)
			var d193 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d191.Imm.Int()) << uint64(d192.Imm.Int())))}
			} else if d192.Loc == scm.LocImm {
				r165 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r165, d191.Reg)
				ctx.W.EmitShlRegImm8(r165, uint8(d192.Imm.Int()))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d193)
			} else {
				{
					shiftSrc := d191.Reg
					r166 := ctx.AllocRegExcept(d191.Reg)
					ctx.W.EmitMovRegReg(r166, d191.Reg)
					shiftSrc = r166
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d192.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d192.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d192.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d193)
				}
			}
			if d193.Loc == scm.LocReg && d191.Loc == scm.LocReg && d193.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d191)
			ctx.FreeDesc(&d192)
			var d194 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r167, thisptr.Reg, off)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
				ctx.BindReg(r167, &d194)
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d194.Loc == scm.LocImm {
				if d194.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
			d195 := d193
			if d195.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d195)
			ctx.EmitStoreToStack(d195, 104)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d194.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
			d196 := d193
			if d196.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d196)
			ctx.EmitStoreToStack(d196, 104)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d194)
			ctx.W.MarkLabel(lbl46)
			d197 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d198 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r168 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r168, thisptr.Reg, off)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r168}
				ctx.BindReg(r168, &d198)
			}
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d198.Imm.Int()))))}
			} else {
				r169 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r169, d198.Reg)
				ctx.W.EmitShlRegImm8(r169, 56)
				ctx.W.EmitShrRegImm8(r169, 56)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d199)
			}
			ctx.FreeDesc(&d198)
			d200 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d199)
			var d201 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d199.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() - d199.Imm.Int())}
			} else if d199.Loc == scm.LocImm && d199.Imm.Int() == 0 {
				r170 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r170, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d201)
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
				r171 := ctx.AllocRegExcept(d200.Reg, d199.Reg)
				ctx.W.EmitMovRegReg(r171, d200.Reg)
				ctx.W.EmitSubInt64(r171, d199.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d201)
			}
			if d201.Loc == scm.LocReg && d200.Loc == scm.LocReg && d201.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d199)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d197.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d197.Imm.Int()) >> uint64(d201.Imm.Int())))}
			} else if d201.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r172, d197.Reg)
				ctx.W.EmitShrRegImm8(r172, uint8(d201.Imm.Int()))
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d202)
			} else {
				{
					shiftSrc := d197.Reg
					r173 := ctx.AllocRegExcept(d197.Reg)
					ctx.W.EmitMovRegReg(r173, d197.Reg)
					shiftSrc = r173
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d201.Reg != scm.RegRCX
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
			if d202.Loc == scm.LocReg && d197.Loc == scm.LocReg && d202.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			ctx.FreeDesc(&d201)
			r174 := ctx.AllocReg()
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d202)
			if d202.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r174, d202)
			}
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl45)
			d197 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d188)
			var d203 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r175 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r175, d188.Reg)
				ctx.W.EmitAndRegImm32(r175, 63)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d203)
			}
			if d203.Loc == scm.LocReg && d188.Loc == scm.LocReg && d203.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			var d204 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r176 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r176, thisptr.Reg, off)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r176}
				ctx.BindReg(r176, &d204)
			}
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d204)
			var d205 scm.JITValueDesc
			if d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d204.Imm.Int()))))}
			} else {
				r177 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r177, d204.Reg)
				ctx.W.EmitShlRegImm8(r177, 56)
				ctx.W.EmitShrRegImm8(r177, 56)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d205)
			}
			ctx.FreeDesc(&d204)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d205)
			var d206 scm.JITValueDesc
			if d203.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d203.Imm.Int() + d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(r178, d203.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d206)
			} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d205.Reg}
				ctx.BindReg(d205.Reg, &d206)
			} else if d203.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d203.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(scratch, d203.Reg)
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else {
				r179 := ctx.AllocRegExcept(d203.Reg, d205.Reg)
				ctx.W.EmitMovRegReg(r179, d203.Reg)
				ctx.W.EmitAddInt64(r179, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d206)
			}
			if d206.Loc == scm.LocReg && d203.Loc == scm.LocReg && d206.Reg == d203.Reg {
				ctx.TransferReg(d203.Reg)
				d203.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d203)
			ctx.FreeDesc(&d205)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d206.Imm.Int()) > uint64(64))}
			} else {
				r180 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitCmpRegImm32(d206.Reg, 64)
				ctx.W.EmitSetcc(r180, scm.CcA)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r180}
				ctx.BindReg(r180, &d207)
			}
			ctx.FreeDesc(&d206)
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d207.Loc == scm.LocImm {
				if d207.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
			d208 := d193
			if d208.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d208)
			ctx.EmitStoreToStack(d208, 104)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d207.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl49)
			d209 := d193
			if d209.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d209)
			ctx.EmitStoreToStack(d209, 104)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl49)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d207)
			ctx.W.MarkLabel(lbl48)
			d197 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d188)
			var d210 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() / 64)}
			} else {
				r181 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r181, d188.Reg)
				ctx.W.EmitShrRegImm8(r181, 6)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d210)
			}
			if d210.Loc == scm.LocReg && d188.Loc == scm.LocReg && d210.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d210)
			var d211 scm.JITValueDesc
			if d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d210.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(scratch, d210.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d211)
			}
			if d211.Loc == scm.LocReg && d210.Loc == scm.LocReg && d211.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.EnsureDesc(&d211)
			r182 := ctx.AllocReg()
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d189)
			if d211.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d211.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r182, d211.Reg)
				ctx.W.EmitShlRegImm8(r182, 3)
			}
			if d189.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d189.Imm.Int()))
				ctx.W.EmitAddInt64(r182, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r182, d189.Reg)
			}
			r183 := ctx.AllocRegExcept(r182)
			ctx.W.EmitMovRegMem(r183, r182, 0)
			ctx.FreeReg(r182)
			d212 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
			ctx.BindReg(r183, &d212)
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d188)
			var d213 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r184 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r184, d188.Reg)
				ctx.W.EmitAndRegImm32(r184, 63)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d213)
			}
			if d213.Loc == scm.LocReg && d188.Loc == scm.LocReg && d213.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			d214 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d213)
			var d215 scm.JITValueDesc
			if d214.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d214.Imm.Int() - d213.Imm.Int())}
			} else if d213.Loc == scm.LocImm && d213.Imm.Int() == 0 {
				r185 := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegReg(r185, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d215)
			} else if d214.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d214.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d213.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else if d213.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegReg(scratch, d214.Reg)
				if d213.Imm.Int() >= -2147483648 && d213.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d213.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else {
				r186 := ctx.AllocRegExcept(d214.Reg, d213.Reg)
				ctx.W.EmitMovRegReg(r186, d214.Reg)
				ctx.W.EmitSubInt64(r186, d213.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d215)
			}
			if d215.Loc == scm.LocReg && d214.Loc == scm.LocReg && d215.Reg == d214.Reg {
				ctx.TransferReg(d214.Reg)
				d214.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d215)
			var d216 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d215.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d212.Imm.Int()) >> uint64(d215.Imm.Int())))}
			} else if d215.Loc == scm.LocImm {
				r187 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r187, d212.Reg)
				ctx.W.EmitShrRegImm8(r187, uint8(d215.Imm.Int()))
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d216)
			} else {
				{
					shiftSrc := d212.Reg
					r188 := ctx.AllocRegExcept(d212.Reg)
					ctx.W.EmitMovRegReg(r188, d212.Reg)
					shiftSrc = r188
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d215.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d215.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d215.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d216)
				}
			}
			if d216.Loc == scm.LocReg && d212.Loc == scm.LocReg && d216.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			ctx.FreeDesc(&d215)
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d193.Imm.Int() | d216.Imm.Int())}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d216.Reg}
				ctx.BindReg(d216.Reg, &d217)
			} else if d216.Loc == scm.LocImm && d216.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r189, d193.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d217)
			} else if d193.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d216.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else if d216.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r190, d193.Reg)
				if d216.Imm.Int() >= -2147483648 && d216.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r190, int32(d216.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d216.Imm.Int()))
					ctx.W.EmitOrInt64(r190, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d217)
			} else {
				r191 := ctx.AllocRegExcept(d193.Reg, d216.Reg)
				ctx.W.EmitMovRegReg(r191, d193.Reg)
				ctx.W.EmitOrInt64(r191, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d217)
			}
			if d217.Loc == scm.LocReg && d193.Loc == scm.LocReg && d217.Reg == d193.Reg {
				ctx.TransferReg(d193.Reg)
				d193.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d216)
			d218 := d217
			if d218.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d218)
			ctx.EmitStoreToStack(d218, 104)
			ctx.W.EmitJmp(lbl46)
			ctx.W.MarkLabel(lbl44)
			d219 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
			ctx.BindReg(r174, &d219)
			ctx.BindReg(r174, &d219)
			if r154 { ctx.UnprotectReg(r155) }
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d219.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r192, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d220)
			}
			ctx.FreeDesc(&d219)
			var d221 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r193, thisptr.Reg, off)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
				ctx.BindReg(r193, &d221)
			}
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d221)
			var d222 scm.JITValueDesc
			if d220.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d220.Imm.Int() + d221.Imm.Int())}
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				r194 := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegReg(r194, d220.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d222)
			} else if d220.Loc == scm.LocImm && d220.Imm.Int() == 0 {
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
				ctx.BindReg(d221.Reg, &d222)
			} else if d220.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d220.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegReg(scratch, d220.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else {
				r195 := ctx.AllocRegExcept(d220.Reg, d221.Reg)
				ctx.W.EmitMovRegReg(r195, d220.Reg)
				ctx.W.EmitAddInt64(r195, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d222)
			}
			if d222.Loc == scm.LocReg && d220.Loc == scm.LocReg && d222.Reg == d220.Reg {
				ctx.TransferReg(d220.Reg)
				d220.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d220)
			ctx.FreeDesc(&d221)
			ctx.EnsureDesc(&d101)
			d223 := d101
			_ = d223
			r196 := d101.Loc == scm.LocReg
			r197 := d101.Reg
			if r196 { ctx.ProtectReg(r197) }
			lbl50 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d223.Imm.Int()))))}
			} else {
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r198, d223.Reg)
				ctx.W.EmitShlRegImm8(r198, 32)
				ctx.W.EmitShrRegImm8(r198, 32)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d224)
			}
			var d225 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r199, thisptr.Reg, off)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d225)
			}
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d225)
			var d226 scm.JITValueDesc
			if d225.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d225.Imm.Int()))))}
			} else {
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r200, d225.Reg)
				ctx.W.EmitShlRegImm8(r200, 56)
				ctx.W.EmitShrRegImm8(r200, 56)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d226)
			}
			ctx.FreeDesc(&d225)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d226)
			var d227 scm.JITValueDesc
			if d224.Loc == scm.LocImm && d226.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d224.Imm.Int() * d226.Imm.Int())}
			} else if d224.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d224.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d226.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			} else if d226.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(scratch, d224.Reg)
				if d226.Imm.Int() >= -2147483648 && d226.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d226.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			} else {
				r201 := ctx.AllocRegExcept(d224.Reg, d226.Reg)
				ctx.W.EmitMovRegReg(r201, d224.Reg)
				ctx.W.EmitImulInt64(r201, d226.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d227)
			}
			if d227.Loc == scm.LocReg && d224.Loc == scm.LocReg && d227.Reg == d224.Reg {
				ctx.TransferReg(d224.Reg)
				d224.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d224)
			ctx.FreeDesc(&d226)
			var d228 scm.JITValueDesc
			r202 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r202, uint64(dataPtr))
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202, StackOff: int32(sliceLen)}
				ctx.BindReg(r202, &d228)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r202, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d228)
			}
			ctx.BindReg(r202, &d228)
			ctx.EnsureDesc(&d227)
			var d229 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() / 64)}
			} else {
				r203 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r203, d227.Reg)
				ctx.W.EmitShrRegImm8(r203, 6)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d229)
			}
			if d229.Loc == scm.LocReg && d227.Loc == scm.LocReg && d229.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d229)
			r204 := ctx.AllocReg()
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d228)
			if d229.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r204, uint64(d229.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r204, d229.Reg)
				ctx.W.EmitShlRegImm8(r204, 3)
			}
			if d228.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d228.Imm.Int()))
				ctx.W.EmitAddInt64(r204, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r204, d228.Reg)
			}
			r205 := ctx.AllocRegExcept(r204)
			ctx.W.EmitMovRegMem(r205, r204, 0)
			ctx.FreeReg(r204)
			d230 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
			ctx.BindReg(r205, &d230)
			ctx.FreeDesc(&d229)
			ctx.EnsureDesc(&d227)
			var d231 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() % 64)}
			} else {
				r206 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r206, d227.Reg)
				ctx.W.EmitAndRegImm32(r206, 63)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d231)
			}
			if d231.Loc == scm.LocReg && d227.Loc == scm.LocReg && d231.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d231)
			var d232 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d230.Imm.Int()) << uint64(d231.Imm.Int())))}
			} else if d231.Loc == scm.LocImm {
				r207 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(r207, d230.Reg)
				ctx.W.EmitShlRegImm8(r207, uint8(d231.Imm.Int()))
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d232)
			} else {
				{
					shiftSrc := d230.Reg
					r208 := ctx.AllocRegExcept(d230.Reg)
					ctx.W.EmitMovRegReg(r208, d230.Reg)
					shiftSrc = r208
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d231.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d231.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d231.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d232)
				}
			}
			if d232.Loc == scm.LocReg && d230.Loc == scm.LocReg && d232.Reg == d230.Reg {
				ctx.TransferReg(d230.Reg)
				d230.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d230)
			ctx.FreeDesc(&d231)
			var d233 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r209 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r209, thisptr.Reg, off)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
				ctx.BindReg(r209, &d233)
			}
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			if d233.Loc == scm.LocImm {
				if d233.Imm.Bool() {
					ctx.W.EmitJmp(lbl51)
				} else {
			d234 := d232
			if d234.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d234)
			ctx.EmitStoreToStack(d234, 112)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d233.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
			d235 := d232
			if d235.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d235)
			ctx.EmitStoreToStack(d235, 112)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl51)
			}
			ctx.FreeDesc(&d233)
			ctx.W.MarkLabel(lbl52)
			d236 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d237 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d237)
			}
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d237)
			var d238 scm.JITValueDesc
			if d237.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d237.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d237.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d238)
			}
			ctx.FreeDesc(&d237)
			d239 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d238)
			var d240 scm.JITValueDesc
			if d239.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() - d238.Imm.Int())}
			} else if d238.Loc == scm.LocImm && d238.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r212, d239.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d240)
			} else if d239.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d239.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d238.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(scratch, d239.Reg)
				if d238.Imm.Int() >= -2147483648 && d238.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d238.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d238.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else {
				r213 := ctx.AllocRegExcept(d239.Reg, d238.Reg)
				ctx.W.EmitMovRegReg(r213, d239.Reg)
				ctx.W.EmitSubInt64(r213, d238.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d240)
			}
			if d240.Loc == scm.LocReg && d239.Loc == scm.LocReg && d240.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d238)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d240)
			var d241 scm.JITValueDesc
			if d236.Loc == scm.LocImm && d240.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d236.Imm.Int()) >> uint64(d240.Imm.Int())))}
			} else if d240.Loc == scm.LocImm {
				r214 := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(r214, d236.Reg)
				ctx.W.EmitShrRegImm8(r214, uint8(d240.Imm.Int()))
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d241)
			} else {
				{
					shiftSrc := d236.Reg
					r215 := ctx.AllocRegExcept(d236.Reg)
					ctx.W.EmitMovRegReg(r215, d236.Reg)
					shiftSrc = r215
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d240.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d240.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d240.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d241)
				}
			}
			if d241.Loc == scm.LocReg && d236.Loc == scm.LocReg && d241.Reg == d236.Reg {
				ctx.TransferReg(d236.Reg)
				d236.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.FreeDesc(&d240)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d241)
			ctx.EnsureDesc(&d241)
			if d241.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r216, d241)
			}
			ctx.W.EmitJmp(lbl50)
			ctx.W.MarkLabel(lbl51)
			d236 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d227)
			var d242 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() % 64)}
			} else {
				r217 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r217, d227.Reg)
				ctx.W.EmitAndRegImm32(r217, 63)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d242)
			}
			if d242.Loc == scm.LocReg && d227.Loc == scm.LocReg && d242.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			var d243 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r218 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r218, thisptr.Reg, off)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
				ctx.BindReg(r218, &d243)
			}
			ctx.EnsureDesc(&d243)
			ctx.EnsureDesc(&d243)
			var d244 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d243.Imm.Int()))))}
			} else {
				r219 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r219, d243.Reg)
				ctx.W.EmitShlRegImm8(r219, 56)
				ctx.W.EmitShrRegImm8(r219, 56)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d244)
			}
			ctx.FreeDesc(&d243)
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d244)
			var d245 scm.JITValueDesc
			if d242.Loc == scm.LocImm && d244.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d242.Imm.Int() + d244.Imm.Int())}
			} else if d244.Loc == scm.LocImm && d244.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d242.Reg)
				ctx.W.EmitMovRegReg(r220, d242.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d245)
			} else if d242.Loc == scm.LocImm && d242.Imm.Int() == 0 {
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d244.Reg}
				ctx.BindReg(d244.Reg, &d245)
			} else if d242.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d242.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d244.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d245)
			} else if d244.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d242.Reg)
				ctx.W.EmitMovRegReg(scratch, d242.Reg)
				if d244.Imm.Int() >= -2147483648 && d244.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d244.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d244.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d245)
			} else {
				r221 := ctx.AllocRegExcept(d242.Reg, d244.Reg)
				ctx.W.EmitMovRegReg(r221, d242.Reg)
				ctx.W.EmitAddInt64(r221, d244.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d245)
			}
			if d245.Loc == scm.LocReg && d242.Loc == scm.LocReg && d245.Reg == d242.Reg {
				ctx.TransferReg(d242.Reg)
				d242.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d242)
			ctx.FreeDesc(&d244)
			ctx.EnsureDesc(&d245)
			var d246 scm.JITValueDesc
			if d245.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d245.Imm.Int()) > uint64(64))}
			} else {
				r222 := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitCmpRegImm32(d245.Reg, 64)
				ctx.W.EmitSetcc(r222, scm.CcA)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r222}
				ctx.BindReg(r222, &d246)
			}
			ctx.FreeDesc(&d245)
			lbl54 := ctx.W.ReserveLabel()
			lbl55 := ctx.W.ReserveLabel()
			if d246.Loc == scm.LocImm {
				if d246.Imm.Bool() {
					ctx.W.EmitJmp(lbl54)
				} else {
			d247 := d232
			if d247.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d247)
			ctx.EmitStoreToStack(d247, 112)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d246.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl55)
			d248 := d232
			if d248.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d248)
			ctx.EmitStoreToStack(d248, 112)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl55)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.FreeDesc(&d246)
			ctx.W.MarkLabel(lbl54)
			d236 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d227)
			var d249 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() / 64)}
			} else {
				r223 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r223, d227.Reg)
				ctx.W.EmitShrRegImm8(r223, 6)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d249)
			}
			if d249.Loc == scm.LocReg && d227.Loc == scm.LocReg && d249.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d249)
			var d250 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(scratch, d249.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d250)
			}
			if d250.Loc == scm.LocReg && d249.Loc == scm.LocReg && d250.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			ctx.EnsureDesc(&d250)
			r224 := ctx.AllocReg()
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d228)
			if d250.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r224, uint64(d250.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r224, d250.Reg)
				ctx.W.EmitShlRegImm8(r224, 3)
			}
			if d228.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d228.Imm.Int()))
				ctx.W.EmitAddInt64(r224, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r224, d228.Reg)
			}
			r225 := ctx.AllocRegExcept(r224)
			ctx.W.EmitMovRegMem(r225, r224, 0)
			ctx.FreeReg(r224)
			d251 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r225}
			ctx.BindReg(r225, &d251)
			ctx.FreeDesc(&d250)
			ctx.EnsureDesc(&d227)
			var d252 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() % 64)}
			} else {
				r226 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r226, d227.Reg)
				ctx.W.EmitAndRegImm32(r226, 63)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d252)
			}
			if d252.Loc == scm.LocReg && d227.Loc == scm.LocReg && d252.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d227)
			d253 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d252)
			ctx.EnsureDesc(&d253)
			ctx.EnsureDesc(&d252)
			ctx.EnsureDesc(&d253)
			ctx.EnsureDesc(&d252)
			var d254 scm.JITValueDesc
			if d253.Loc == scm.LocImm && d252.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d253.Imm.Int() - d252.Imm.Int())}
			} else if d252.Loc == scm.LocImm && d252.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(r227, d253.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d254)
			} else if d253.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d252.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d253.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d252.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d254)
			} else if d252.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(scratch, d253.Reg)
				if d252.Imm.Int() >= -2147483648 && d252.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d252.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d252.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d254)
			} else {
				r228 := ctx.AllocRegExcept(d253.Reg, d252.Reg)
				ctx.W.EmitMovRegReg(r228, d253.Reg)
				ctx.W.EmitSubInt64(r228, d252.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d254)
			}
			if d254.Loc == scm.LocReg && d253.Loc == scm.LocReg && d254.Reg == d253.Reg {
				ctx.TransferReg(d253.Reg)
				d253.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d252)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d254)
			var d255 scm.JITValueDesc
			if d251.Loc == scm.LocImm && d254.Loc == scm.LocImm {
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d251.Imm.Int()) >> uint64(d254.Imm.Int())))}
			} else if d254.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(r229, d251.Reg)
				ctx.W.EmitShrRegImm8(r229, uint8(d254.Imm.Int()))
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d255)
			} else {
				{
					shiftSrc := d251.Reg
					r230 := ctx.AllocRegExcept(d251.Reg)
					ctx.W.EmitMovRegReg(r230, d251.Reg)
					shiftSrc = r230
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d254.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d254.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d254.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d255)
				}
			}
			if d255.Loc == scm.LocReg && d251.Loc == scm.LocReg && d255.Reg == d251.Reg {
				ctx.TransferReg(d251.Reg)
				d251.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d251)
			ctx.FreeDesc(&d254)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d255)
			var d256 scm.JITValueDesc
			if d232.Loc == scm.LocImm && d255.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() | d255.Imm.Int())}
			} else if d232.Loc == scm.LocImm && d232.Imm.Int() == 0 {
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d255.Reg}
				ctx.BindReg(d255.Reg, &d256)
			} else if d255.Loc == scm.LocImm && d255.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r231, d232.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d256)
			} else if d232.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d255.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d232.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d255.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d256)
			} else if d255.Loc == scm.LocImm {
				r232 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r232, d232.Reg)
				if d255.Imm.Int() >= -2147483648 && d255.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r232, int32(d255.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d255.Imm.Int()))
					ctx.W.EmitOrInt64(r232, scm.RegR11)
				}
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d256)
			} else {
				r233 := ctx.AllocRegExcept(d232.Reg, d255.Reg)
				ctx.W.EmitMovRegReg(r233, d232.Reg)
				ctx.W.EmitOrInt64(r233, d255.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d256)
			}
			if d256.Loc == scm.LocReg && d232.Loc == scm.LocReg && d256.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d255)
			d257 := d256
			if d257.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d257)
			ctx.EmitStoreToStack(d257, 112)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl50)
			d258 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
			ctx.BindReg(r216, &d258)
			ctx.BindReg(r216, &d258)
			if r196 { ctx.UnprotectReg(r197) }
			ctx.FreeDesc(&d101)
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d258)
			var d259 scm.JITValueDesc
			if d258.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d258.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d259)
			}
			ctx.FreeDesc(&d258)
			var d260 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r235, thisptr.Reg, off)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r235}
				ctx.BindReg(r235, &d260)
			}
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d260)
			var d261 scm.JITValueDesc
			if d259.Loc == scm.LocImm && d260.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d259.Imm.Int() + d260.Imm.Int())}
			} else if d260.Loc == scm.LocImm && d260.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(r236, d259.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d261)
			} else if d259.Loc == scm.LocImm && d259.Imm.Int() == 0 {
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d260.Reg}
				ctx.BindReg(d260.Reg, &d261)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d259.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else if d260.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(scratch, d259.Reg)
				if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d260.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d260.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else {
				r237 := ctx.AllocRegExcept(d259.Reg, d260.Reg)
				ctx.W.EmitMovRegReg(r237, d259.Reg)
				ctx.W.EmitAddInt64(r237, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d261)
			}
			if d261.Loc == scm.LocReg && d259.Loc == scm.LocReg && d261.Reg == d259.Reg {
				ctx.TransferReg(d259.Reg)
				d259.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d259)
			ctx.FreeDesc(&d260)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d262 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d262)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			var d263 scm.JITValueDesc
			if d262.Loc == scm.LocImm && d261.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d262.Imm.Int() - d261.Imm.Int())}
			} else if d261.Loc == scm.LocImm && d261.Imm.Int() == 0 {
				r239 := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(r239, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d263)
			} else if d262.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d262.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d261.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else if d261.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(scratch, d262.Reg)
				if d261.Imm.Int() >= -2147483648 && d261.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d261.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d261.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else {
				r240 := ctx.AllocRegExcept(d262.Reg, d261.Reg)
				ctx.W.EmitMovRegReg(r240, d262.Reg)
				ctx.W.EmitSubInt64(r240, d261.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d263)
			}
			if d263.Loc == scm.LocReg && d262.Loc == scm.LocReg && d263.Reg == d262.Reg {
				ctx.TransferReg(d262.Reg)
				d262.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d262)
			ctx.FreeDesc(&d261)
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d222)
			var d264 scm.JITValueDesc
			if d263.Loc == scm.LocImm && d222.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d263.Imm.Int() * d222.Imm.Int())}
			} else if d263.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d263.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d222.Reg)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d264)
			} else if d222.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d263.Reg)
				ctx.W.EmitMovRegReg(scratch, d263.Reg)
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d222.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d264)
			} else {
				r241 := ctx.AllocRegExcept(d263.Reg, d222.Reg)
				ctx.W.EmitMovRegReg(r241, d263.Reg)
				ctx.W.EmitImulInt64(r241, d222.Reg)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d264)
			}
			if d264.Loc == scm.LocReg && d263.Loc == scm.LocReg && d264.Reg == d263.Reg {
				ctx.TransferReg(d263.Reg)
				d263.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d263)
			ctx.FreeDesc(&d222)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d264)
			var d265 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d264.Loc == scm.LocImm {
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() + d264.Imm.Int())}
			} else if d264.Loc == scm.LocImm && d264.Imm.Int() == 0 {
				r242 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r242, d141.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d265)
			} else if d141.Loc == scm.LocImm && d141.Imm.Int() == 0 {
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d264.Reg}
				ctx.BindReg(d264.Reg, &d265)
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d264.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d264.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d265)
			} else if d264.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(scratch, d141.Reg)
				if d264.Imm.Int() >= -2147483648 && d264.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d264.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d264.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d265)
			} else {
				r243 := ctx.AllocRegExcept(d141.Reg, d264.Reg)
				ctx.W.EmitMovRegReg(r243, d141.Reg)
				ctx.W.EmitAddInt64(r243, d264.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d265)
			}
			if d265.Loc == scm.LocReg && d141.Loc == scm.LocReg && d265.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d264)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d265)
			var d266 scm.JITValueDesc
			if d265.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d265.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d265.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d265.Reg}
				ctx.BindReg(d265.Reg, &d266)
			}
			ctx.FreeDesc(&d265)
			ctx.EnsureDesc(&d266)
			d267 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d267)
			ctx.BindReg(r2, &d267)
			ctx.EnsureDesc(&d266)
			ctx.W.EmitMakeFloat(d267, d266)
			if d266.Loc == scm.LocReg { ctx.FreeReg(d266.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl32)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			var d268 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r244, thisptr.Reg, off)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r244}
				ctx.BindReg(r244, &d268)
			}
			ctx.EnsureDesc(&d268)
			ctx.EnsureDesc(&d268)
			var d269 scm.JITValueDesc
			if d268.Loc == scm.LocImm {
				d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d268.Imm.Int()))))}
			} else {
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r245, d268.Reg)
				d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d269)
			}
			ctx.FreeDesc(&d268)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d269)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d269)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d269)
			var d270 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d269.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d141.Imm.Int() == d269.Imm.Int())}
			} else if d269.Loc == scm.LocImm {
				r246 := ctx.AllocRegExcept(d141.Reg)
				if d269.Imm.Int() >= -2147483648 && d269.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d141.Reg, int32(d269.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d269.Imm.Int()))
					ctx.W.EmitCmpInt64(d141.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r246, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d270)
			} else if d141.Loc == scm.LocImm {
				r247 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d269.Reg)
				ctx.W.EmitSetcc(r247, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r247}
				ctx.BindReg(r247, &d270)
			} else {
				r248 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitCmpInt64(d141.Reg, d269.Reg)
				ctx.W.EmitSetcc(r248, scm.CcE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r248}
				ctx.BindReg(r248, &d270)
			}
			ctx.FreeDesc(&d141)
			ctx.FreeDesc(&d269)
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			if d270.Loc == scm.LocImm {
				if d270.Imm.Bool() {
					ctx.W.EmitJmp(lbl56)
				} else {
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d270.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d270)
			ctx.W.MarkLabel(lbl42)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			lbl58 := ctx.W.ReserveLabel()
			d271 := d78
			if d271.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d271)
			d272 := d271
			if d272.Loc == scm.LocImm {
				d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: d272.Type, Imm: scm.NewInt(int64(uint64(d272.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d272.Reg, 32)
				ctx.W.EmitShrRegImm8(d272.Reg, 32)
			}
			ctx.EmitStoreToStack(d272, 64)
			d273 := d80
			if d273.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d273)
			d274 := d273
			if d274.Loc == scm.LocImm {
				d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: d274.Type, Imm: scm.NewInt(int64(uint64(d274.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d274.Reg, 32)
				ctx.W.EmitShrRegImm8(d274.Reg, 32)
			}
			ctx.EmitStoreToStack(d274, 72)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl58)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d275 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d276 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			var d277 scm.JITValueDesc
			if d275.Loc == scm.LocImm && d276.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d275.Imm.Int()) == uint64(d276.Imm.Int()))}
			} else if d276.Loc == scm.LocImm {
				r249 := ctx.AllocRegExcept(d275.Reg)
				if d276.Imm.Int() >= -2147483648 && d276.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d275.Reg, int32(d276.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d276.Imm.Int()))
					ctx.W.EmitCmpInt64(d275.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r249, scm.CcE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r249}
				ctx.BindReg(r249, &d277)
			} else if d275.Loc == scm.LocImm {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d275.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d276.Reg)
				ctx.W.EmitSetcc(r250, scm.CcE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r250}
				ctx.BindReg(r250, &d277)
			} else {
				r251 := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitCmpInt64(d275.Reg, d276.Reg)
				ctx.W.EmitSetcc(r251, scm.CcE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d277)
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			if d277.Loc == scm.LocImm {
				if d277.Imm.Bool() {
			d278 := d275
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
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl59)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d277.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl60)
			d280 := d275
			if d280.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d280)
			d281 := d280
			if d281.Loc == scm.LocImm {
				d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: d281.Type, Imm: scm.NewInt(int64(uint64(d281.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d281.Reg, 32)
				ctx.W.EmitShrRegImm8(d281.Reg, 32)
			}
			ctx.EmitStoreToStack(d281, 32)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d277)
			ctx.W.MarkLabel(lbl41)
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d276 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d78)
			var d282 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d78.Imm.Int()) == uint64(0))}
			} else {
				r252 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitCmpRegImm32(d78.Reg, 0)
				ctx.W.EmitSetcc(r252, scm.CcE)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d282)
			}
			lbl61 := ctx.W.ReserveLabel()
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			if d282.Loc == scm.LocImm {
				if d282.Imm.Bool() {
					ctx.W.EmitJmp(lbl61)
				} else {
					ctx.W.EmitJmp(lbl62)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d282.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl63)
				ctx.W.EmitJmp(lbl62)
				ctx.W.MarkLabel(lbl63)
				ctx.W.EmitJmp(lbl61)
			}
			ctx.FreeDesc(&d282)
			ctx.W.MarkLabel(lbl56)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d276 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d283 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d283)
			ctx.BindReg(r2, &d283)
			ctx.W.EmitMakeNil(d283)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl59)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d276 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d276)
			var d284 scm.JITValueDesc
			if d275.Loc == scm.LocImm && d276.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d275.Imm.Int() + d276.Imm.Int())}
			} else if d276.Loc == scm.LocImm && d276.Imm.Int() == 0 {
				r253 := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegReg(r253, d275.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d284)
			} else if d275.Loc == scm.LocImm && d275.Imm.Int() == 0 {
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d276.Reg}
				ctx.BindReg(d276.Reg, &d284)
			} else if d275.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d276.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d275.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d276.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			} else if d276.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegReg(scratch, d275.Reg)
				if d276.Imm.Int() >= -2147483648 && d276.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d276.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d276.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d284)
			} else {
				r254 := ctx.AllocRegExcept(d275.Reg, d276.Reg)
				ctx.W.EmitMovRegReg(r254, d275.Reg)
				ctx.W.EmitAddInt64(r254, d276.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d284)
			}
			if d284.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: d284.Type, Imm: scm.NewInt(int64(uint64(d284.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d284.Reg, 32)
				ctx.W.EmitShrRegImm8(d284.Reg, 32)
			}
			if d284.Loc == scm.LocReg && d275.Loc == scm.LocReg && d284.Reg == d275.Reg {
				ctx.TransferReg(d275.Reg)
				d275.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d284)
			var d285 scm.JITValueDesc
			if d284.Loc == scm.LocImm {
				d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d284.Imm.Int() / 2)}
			} else {
				r255 := ctx.AllocRegExcept(d284.Reg)
				ctx.W.EmitMovRegReg(r255, d284.Reg)
				ctx.W.EmitShrRegImm8(r255, 1)
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
				ctx.BindReg(r255, &d285)
			}
			if d285.Loc == scm.LocImm {
				d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: d285.Type, Imm: scm.NewInt(int64(uint64(d285.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d285.Reg, 32)
				ctx.W.EmitShrRegImm8(d285.Reg, 32)
			}
			if d285.Loc == scm.LocReg && d284.Loc == scm.LocReg && d285.Reg == d284.Reg {
				ctx.TransferReg(d284.Reg)
				d284.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d284)
			d286 := d285
			if d286.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d286)
			d287 := d286
			if d287.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: d287.Type, Imm: scm.NewInt(int64(uint64(d287.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d287.Reg, 32)
				ctx.W.EmitShrRegImm8(d287.Reg, 32)
			}
			ctx.EmitStoreToStack(d287, 8)
			d288 := d275
			if d288.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d288)
			d289 := d288
			if d289.Loc == scm.LocImm {
				d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: d289.Type, Imm: scm.NewInt(int64(uint64(d289.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d289.Reg, 32)
				ctx.W.EmitShrRegImm8(d289.Reg, 32)
			}
			ctx.EmitStoreToStack(d289, 16)
			d290 := d276
			if d290.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d290)
			d291 := d290
			if d291.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: d291.Type, Imm: scm.NewInt(int64(uint64(d291.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d291.Reg, 32)
				ctx.W.EmitShrRegImm8(d291.Reg, 32)
			}
			ctx.EmitStoreToStack(d291, 24)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl62)
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d276 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d78)
			var d292 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d292 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(scratch, d78.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d292)
			}
			if d292.Loc == scm.LocImm {
				d292 = scm.JITValueDesc{Loc: scm.LocImm, Type: d292.Type, Imm: scm.NewInt(int64(uint64(d292.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d292.Reg, 32)
				ctx.W.EmitShrRegImm8(d292.Reg, 32)
			}
			if d292.Loc == scm.LocReg && d78.Loc == scm.LocReg && d292.Reg == d78.Reg {
				ctx.TransferReg(d78.Reg)
				d78.Loc = scm.LocNone
			}
			d293 := d79
			if d293.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d293)
			d294 := d293
			if d294.Loc == scm.LocImm {
				d294 = scm.JITValueDesc{Loc: scm.LocImm, Type: d294.Type, Imm: scm.NewInt(int64(uint64(d294.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d294.Reg, 32)
				ctx.W.EmitShrRegImm8(d294.Reg, 32)
			}
			ctx.EmitStoreToStack(d294, 64)
			d295 := d292
			if d295.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d295)
			d296 := d295
			if d296.Loc == scm.LocImm {
				d296 = scm.JITValueDesc{Loc: scm.LocImm, Type: d296.Type, Imm: scm.NewInt(int64(uint64(d296.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d296.Reg, 32)
				ctx.W.EmitShrRegImm8(d296.Reg, 32)
			}
			ctx.EmitStoreToStack(d296, 72)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl61)
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d276 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d10 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d78 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d275 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl0)
			d297 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d297)
			ctx.BindReg(r2, &d297)
			ctx.EmitMovPairToResult(&d297, &result)
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
