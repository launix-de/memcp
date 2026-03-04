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
					ctx.W.MarkLabel(lbl4)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
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
					ctx.W.MarkLabel(lbl8)
					ctx.W.EmitJmp(lbl6)
				} else {
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
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
			d9 := d1
			if d9.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d9)
			d10 := d9
			if d10.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: d10.Type, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d10.Reg, 32)
				ctx.W.EmitShrRegImm8(d10.Reg, 32)
			}
			ctx.EmitStoreToStack(d10, 0)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl2)
			d11 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d11)
			ctx.BindReg(r2, &d11)
			ctx.W.EmitMakeNil(d11)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			d12 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d13 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d13)
			}
			if d13.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: d13.Type, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d13.Reg, 32)
				ctx.W.EmitShrRegImm8(d13.Reg, 32)
			}
			if d13.Loc == scm.LocReg && d2.Loc == scm.LocReg && d13.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			lbl10 := ctx.W.ReserveLabel()
			d14 := d12
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			d15 := d14
			if d15.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: d15.Type, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d15.Reg, 32)
				ctx.W.EmitShrRegImm8(d15.Reg, 32)
			}
			ctx.EmitStoreToStack(d15, 8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 16)
			d16 := d13
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			d17 := d16
			if d17.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: d17.Type, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d17.Reg, 32)
				ctx.W.EmitShrRegImm8(d17.Reg, 32)
			}
			ctx.EmitStoreToStack(d17, 24)
			ctx.W.MarkLabel(lbl10)
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d18 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d19 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d18)
			d21 := d18
			_ = d21
			r11 := d18.Loc == scm.LocReg
			r12 := d18.Reg
			if r11 { ctx.ProtectReg(r12) }
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl12)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d21.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d21.Reg)
				ctx.W.EmitShlRegImm8(r13, 32)
				ctx.W.EmitShrRegImm8(r13, 32)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d22)
			}
			var d23 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d23)
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d23.Imm.Int()))))}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r15, d23.Reg)
				ctx.W.EmitShlRegImm8(r15, 56)
				ctx.W.EmitShrRegImm8(r15, 56)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d24)
			}
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() * d24.Imm.Int())}
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r16 := ctx.AllocRegExcept(d22.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r16, d22.Reg)
				ctx.W.EmitImulInt64(r16, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d25)
			}
			if d25.Loc == scm.LocReg && d22.Loc == scm.LocReg && d25.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.FreeDesc(&d24)
			var d26 scm.JITValueDesc
			r17 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r17, uint64(dataPtr))
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17, StackOff: int32(sliceLen)}
				ctx.BindReg(r17, &d26)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r17, thisptr.Reg, off)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d26)
			}
			ctx.BindReg(r17, &d26)
			ctx.EnsureDesc(&d25)
			var d27 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r18, d25.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d27)
			}
			if d27.Loc == scm.LocReg && d25.Loc == scm.LocReg && d27.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d27)
			r19 := ctx.AllocReg()
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d26)
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d27.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d27.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d26.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d26.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d28 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d28)
			ctx.FreeDesc(&d27)
			ctx.EnsureDesc(&d25)
			var d29 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r21, d25.Reg)
				ctx.W.EmitAndRegImm32(r21, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d29)
			}
			if d29.Loc == scm.LocReg && d25.Loc == scm.LocReg && d29.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) << uint64(d29.Imm.Int())))}
			} else if d29.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r22, d28.Reg)
				ctx.W.EmitShlRegImm8(r22, uint8(d29.Imm.Int()))
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d30)
			} else {
				{
					shiftSrc := d28.Reg
					r23 := ctx.AllocRegExcept(d28.Reg)
					ctx.W.EmitMovRegReg(r23, d28.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d29.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d29.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d29.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d30)
				}
			}
			if d30.Loc == scm.LocReg && d28.Loc == scm.LocReg && d30.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d29)
			var d31 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d31)
			}
			d32 := d31
			ctx.EnsureDesc(&d32)
			if d32.Loc != scm.LocImm && d32.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d32.Loc == scm.LocImm {
				if d32.Imm.Bool() {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.MarkLabel(lbl16)
			d33 := d30
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 80)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d32.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl16)
			d34 := d30
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d34)
			ctx.EmitStoreToStack(d34, 80)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d31)
			ctx.W.MarkLabel(lbl14)
			d35 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			var d36 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d36)
			}
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d36.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d36.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d37)
			}
			ctx.FreeDesc(&d36)
			d38 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() - d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r27, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r28 := ctx.AllocRegExcept(d38.Reg, d37.Reg)
				ctx.W.EmitMovRegReg(r28, d38.Reg)
				ctx.W.EmitSubInt64(r28, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d39)
			}
			if d39.Loc == scm.LocReg && d38.Loc == scm.LocReg && d39.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d35.Imm.Int()) >> uint64(d39.Imm.Int())))}
			} else if d39.Loc == scm.LocImm {
				r29 := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(r29, d35.Reg)
				ctx.W.EmitShrRegImm8(r29, uint8(d39.Imm.Int()))
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d40)
			} else {
				{
					shiftSrc := d35.Reg
					r30 := ctx.AllocRegExcept(d35.Reg)
					ctx.W.EmitMovRegReg(r30, d35.Reg)
					shiftSrc = r30
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d39.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d39.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d39.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d40)
				}
			}
			if d40.Loc == scm.LocReg && d35.Loc == scm.LocReg && d40.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.FreeDesc(&d39)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			if d40.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r31, d40)
			}
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl13)
			d35 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d25)
			var d41 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r32, d25.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d41)
			}
			if d41.Loc == scm.LocReg && d25.Loc == scm.LocReg && d41.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r33, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
				ctx.BindReg(r33, &d42)
			}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, d42.Reg)
				ctx.W.EmitShlRegImm8(r34, 56)
				ctx.W.EmitShrRegImm8(r34, 56)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d43)
			}
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + d43.Imm.Int())}
			} else if d43.Loc == scm.LocImm && d43.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r35, d41.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d44)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d43.Reg}
				ctx.BindReg(d43.Reg, &d44)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r36 := ctx.AllocRegExcept(d41.Reg, d43.Reg)
				ctx.W.EmitMovRegReg(r36, d41.Reg)
				ctx.W.EmitAddInt64(r36, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d44)
			}
			if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d43)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) > uint64(64))}
			} else {
				r37 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitCmpRegImm32(d44.Reg, 64)
				ctx.W.EmitSetcc(r37, scm.CcA)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
				ctx.BindReg(r37, &d45)
			}
			ctx.FreeDesc(&d44)
			d46 := d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.MarkLabel(lbl19)
			d47 := d30
			if d47.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d47)
			ctx.EmitStoreToStack(d47, 80)
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl19)
			d48 := d30
			if d48.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d48)
			ctx.EmitStoreToStack(d48, 80)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl17)
			d35 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d25)
			var d49 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() / 64)}
			} else {
				r38 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r38, d25.Reg)
				ctx.W.EmitShrRegImm8(r38, 6)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d49)
			}
			if d49.Loc == scm.LocReg && d25.Loc == scm.LocReg && d49.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d50)
			r39 := ctx.AllocReg()
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d26)
			if d50.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d50.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d50.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			if d26.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
				ctx.W.EmitAddInt64(r39, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r39, d26.Reg)
			}
			r40 := ctx.AllocRegExcept(r39)
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.FreeReg(r39)
			d51 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d51)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d25)
			var d52 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() % 64)}
			} else {
				r41 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r41, d25.Reg)
				ctx.W.EmitAndRegImm32(r41, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d52)
			}
			if d52.Loc == scm.LocReg && d25.Loc == scm.LocReg && d52.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			d53 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d52)
			var d54 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d52.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() - d52.Imm.Int())}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r42, d53.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d54)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d52.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(scratch, d53.Reg)
				if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d52.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else {
				r43 := ctx.AllocRegExcept(d53.Reg, d52.Reg)
				ctx.W.EmitMovRegReg(r43, d53.Reg)
				ctx.W.EmitSubInt64(r43, d52.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d54)
			}
			if d54.Loc == scm.LocReg && d53.Loc == scm.LocReg && d54.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d51.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) >> uint64(d54.Imm.Int())))}
			} else if d54.Loc == scm.LocImm {
				r44 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r44, d51.Reg)
				ctx.W.EmitShrRegImm8(r44, uint8(d54.Imm.Int()))
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d55)
			} else {
				{
					shiftSrc := d51.Reg
					r45 := ctx.AllocRegExcept(d51.Reg)
					ctx.W.EmitMovRegReg(r45, d51.Reg)
					shiftSrc = r45
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d54.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d54.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d54.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d55)
				}
			}
			if d55.Loc == scm.LocReg && d51.Loc == scm.LocReg && d55.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			ctx.FreeDesc(&d54)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() | d55.Imm.Int())}
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
				ctx.BindReg(d55.Reg, &d56)
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r46, d30.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d56)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d56)
			} else if d55.Loc == scm.LocImm {
				r47 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r47, d30.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r47, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitOrInt64(r47, scm.RegR11)
				}
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d56)
			} else {
				r48 := ctx.AllocRegExcept(d30.Reg, d55.Reg)
				ctx.W.EmitMovRegReg(r48, d30.Reg)
				ctx.W.EmitOrInt64(r48, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d56)
			}
			if d56.Loc == scm.LocReg && d30.Loc == scm.LocReg && d56.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			d57 := d56
			if d57.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 80)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl11)
			d58 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d58)
			ctx.BindReg(r31, &d58)
			if r11 { ctx.UnprotectReg(r12) }
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d58)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d58.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d58.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d59)
			}
			ctx.FreeDesc(&d58)
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d60)
			}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				r51 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r51, d59.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d61)
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d60.Reg}
				ctx.BindReg(d60.Reg, &d61)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d60.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else if d60.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d60.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else {
				r52 := ctx.AllocRegExcept(d59.Reg, d60.Reg)
				ctx.W.EmitMovRegReg(r52, d59.Reg)
				ctx.W.EmitAddInt64(r52, d60.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d61)
			}
			if d61.Loc == scm.LocReg && d59.Loc == scm.LocReg && d61.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d61.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d61.Reg)
				ctx.W.EmitShlRegImm8(r53, 32)
				ctx.W.EmitShrRegImm8(r53, 32)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d62)
			}
			ctx.FreeDesc(&d61)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d62.Imm.Int()))}
			} else if d62.Loc == scm.LocImm {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d62.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d62.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r54, scm.CcB)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d63)
			} else if idxInt.Loc == scm.LocImm {
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d62.Reg)
				ctx.W.EmitSetcc(r55, scm.CcB)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d63)
			} else {
				r56 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d62.Reg)
				ctx.W.EmitSetcc(r56, scm.CcB)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d63)
			}
			ctx.FreeDesc(&d62)
			d64 := d63
			ctx.EnsureDesc(&d64)
			if d64.Loc != scm.LocImm && d64.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d64.Loc == scm.LocImm {
				if d64.Imm.Bool() {
					ctx.W.MarkLabel(lbl22)
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d64.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl23)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.FreeDesc(&d63)
			ctx.W.MarkLabel(lbl6)
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d65 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			}
			if d65.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: d65.Type, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d65.Reg, 32)
				ctx.W.EmitShrRegImm8(d65.Reg, 32)
			}
			if d65.Loc == scm.LocReg && d2.Loc == scm.LocReg && d65.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d66 := d65
			if d66.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			d67 := d66
			if d67.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: d67.Type, Imm: scm.NewInt(int64(uint64(d67.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d67.Reg, 32)
				ctx.W.EmitShrRegImm8(d67.Reg, 32)
			}
			ctx.EmitStoreToStack(d67, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl21)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d68 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d68)
			}
			if d68.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: d68.Type, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d68.Reg, 32)
				ctx.W.EmitShrRegImm8(d68.Reg, 32)
			}
			if d68.Loc == scm.LocReg && d18.Loc == scm.LocReg && d68.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d2)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d68.Imm.Int()) >= uint64(d2.Imm.Int()))}
			} else if d2.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d68.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d68.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d68.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r57, scm.CcAE)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d69)
			} else if d68.Loc == scm.LocImm {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d68.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r58, scm.CcAE)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d69)
			} else {
				r59 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitCmpInt64(d68.Reg, d2.Reg)
				ctx.W.EmitSetcc(r59, scm.CcAE)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d69)
			}
			d70 := d69
			ctx.EnsureDesc(&d70)
			if d70.Loc != scm.LocImm && d70.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d70.Loc == scm.LocImm {
				if d70.Imm.Bool() {
					ctx.W.MarkLabel(lbl26)
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.W.MarkLabel(lbl27)
			d71 := d68
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
			d73 := d18
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
			d75 := d20
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
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d70.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl27)
			d77 := d68
			if d77.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d77)
			d78 := d77
			if d78.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: d78.Type, Imm: scm.NewInt(int64(uint64(d78.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d78.Reg, 32)
				ctx.W.EmitShrRegImm8(d78.Reg, 32)
			}
			ctx.EmitStoreToStack(d78, 40)
			d79 := d18
			if d79.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d79)
			d80 := d79
			if d80.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: d80.Type, Imm: scm.NewInt(int64(uint64(d80.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d80.Reg, 32)
				ctx.W.EmitShrRegImm8(d80.Reg, 32)
			}
			ctx.EmitStoreToStack(d80, 48)
			d81 := d20
			if d81.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d81)
			d82 := d81
			if d82.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: d82.Type, Imm: scm.NewInt(int64(uint64(d82.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d82.Reg, 32)
				ctx.W.EmitShrRegImm8(d82.Reg, 32)
			}
			ctx.EmitStoreToStack(d82, 56)
				ctx.W.EmitJmp(lbl25)
			}
			ctx.FreeDesc(&d69)
			ctx.W.MarkLabel(lbl20)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d18)
			var d83 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d18.Imm.Int()) == uint64(0))}
			} else {
				r60 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitSetcc(r60, scm.CcE)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d83)
			}
			d84 := d83
			ctx.EnsureDesc(&d84)
			if d84.Loc != scm.LocImm && d84.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d84.Loc == scm.LocImm {
				if d84.Imm.Bool() {
					ctx.W.MarkLabel(lbl30)
					ctx.W.EmitJmp(lbl28)
				} else {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl29)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d84.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl30)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl30)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d83)
			ctx.W.MarkLabel(lbl25)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d85 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d86.Imm.Int()) == uint64(d87.Imm.Int()))}
			} else if d87.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d86.Reg)
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d86.Reg, int32(d87.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.W.EmitCmpInt64(d86.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r61, scm.CcE)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d88)
			} else if d86.Loc == scm.LocImm {
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d87.Reg)
				ctx.W.EmitSetcc(r62, scm.CcE)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
				ctx.BindReg(r62, &d88)
			} else {
				r63 := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitCmpInt64(d86.Reg, d87.Reg)
				ctx.W.EmitSetcc(r63, scm.CcE)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r63}
				ctx.BindReg(r63, &d88)
			}
			d89 := d88
			ctx.EnsureDesc(&d89)
			if d89.Loc != scm.LocImm && d89.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d89.Loc == scm.LocImm {
				if d89.Imm.Bool() {
					ctx.W.MarkLabel(lbl34)
			d90 := d86
			if d90.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d90)
			d91 := d90
			if d91.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: d91.Type, Imm: scm.NewInt(int64(uint64(d91.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d91.Reg, 32)
				ctx.W.EmitShrRegImm8(d91.Reg, 32)
			}
			ctx.EmitStoreToStack(d91, 32)
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.W.MarkLabel(lbl35)
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d89.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl34)
			d92 := d86
			if d92.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d92)
			d93 := d92
			if d93.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: d93.Type, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d93.Reg, 32)
				ctx.W.EmitShrRegImm8(d93.Reg, 32)
			}
			ctx.EmitStoreToStack(d93, 32)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d88)
			ctx.W.MarkLabel(lbl24)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d94 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
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
			if d94.Loc == scm.LocReg && d2.Loc == scm.LocReg && d94.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
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
			d97 := d18
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
			d99 := d20
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
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl29)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d101 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d101)
			}
			if d101.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: d101.Type, Imm: scm.NewInt(int64(uint64(d101.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d101.Reg, 32)
				ctx.W.EmitShrRegImm8(d101.Reg, 32)
			}
			if d101.Loc == scm.LocReg && d18.Loc == scm.LocReg && d101.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d102 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d102)
			}
			if d102.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: d102.Type, Imm: scm.NewInt(int64(uint64(d102.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d102.Reg, 32)
				ctx.W.EmitShrRegImm8(d102.Reg, 32)
			}
			if d102.Loc == scm.LocReg && d18.Loc == scm.LocReg && d102.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			d103 := d102
			if d103.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d103)
			d104 := d103
			if d104.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: d104.Type, Imm: scm.NewInt(int64(uint64(d104.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d104.Reg, 32)
				ctx.W.EmitShrRegImm8(d104.Reg, 32)
			}
			ctx.EmitStoreToStack(d104, 40)
			d105 := d19
			if d105.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d105)
			d106 := d105
			if d106.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: d106.Type, Imm: scm.NewInt(int64(uint64(d106.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d106.Reg, 32)
				ctx.W.EmitShrRegImm8(d106.Reg, 32)
			}
			ctx.EmitStoreToStack(d106, 48)
			d107 := d101
			if d107.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d107)
			d108 := d107
			if d108.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: d108.Type, Imm: scm.NewInt(int64(uint64(d108.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d108.Reg, 32)
				ctx.W.EmitShrRegImm8(d108.Reg, 32)
			}
			ctx.EmitStoreToStack(d108, 56)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl28)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.MarkLabel(lbl32)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d109.Imm.Int()))))}
			} else {
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r64, d109.Reg)
				ctx.W.EmitShlRegImm8(r64, 32)
				ctx.W.EmitShrRegImm8(r64, 32)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d110)
			}
			ctx.EnsureDesc(&d110)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d110.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d110.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d110.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d110.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d110.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d110.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d110)
			ctx.EnsureDesc(&d109)
			d111 := d109
			_ = d111
			r65 := d109.Loc == scm.LocReg
			r66 := d109.Reg
			if r65 { ctx.ProtectReg(r66) }
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl37)
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d111.Imm.Int()))))}
			} else {
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r67, d111.Reg)
				ctx.W.EmitShlRegImm8(r67, 32)
				ctx.W.EmitShrRegImm8(r67, 32)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d112)
			}
			var d113 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
				ctx.BindReg(r68, &d113)
			}
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d113.Imm.Int()))))}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r69, d113.Reg)
				ctx.W.EmitShlRegImm8(r69, 56)
				ctx.W.EmitShrRegImm8(r69, 56)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d114)
			}
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d114)
			var d115 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() * d114.Imm.Int())}
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d114.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(scratch, d112.Reg)
				if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d114.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else {
				r70 := ctx.AllocRegExcept(d112.Reg, d114.Reg)
				ctx.W.EmitMovRegReg(r70, d112.Reg)
				ctx.W.EmitImulInt64(r70, d114.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d115)
			}
			if d115.Loc == scm.LocReg && d112.Loc == scm.LocReg && d115.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.FreeDesc(&d114)
			var d116 scm.JITValueDesc
			r71 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r71, uint64(dataPtr))
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71, StackOff: int32(sliceLen)}
				ctx.BindReg(r71, &d116)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r71, thisptr.Reg, off)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
				ctx.BindReg(r71, &d116)
			}
			ctx.BindReg(r71, &d116)
			ctx.EnsureDesc(&d115)
			var d117 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() / 64)}
			} else {
				r72 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r72, d115.Reg)
				ctx.W.EmitShrRegImm8(r72, 6)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d117)
			}
			if d117.Loc == scm.LocReg && d115.Loc == scm.LocReg && d117.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d117)
			r73 := ctx.AllocReg()
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d116)
			if d117.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r73, uint64(d117.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r73, d117.Reg)
				ctx.W.EmitShlRegImm8(r73, 3)
			}
			if d116.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
				ctx.W.EmitAddInt64(r73, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r73, d116.Reg)
			}
			r74 := ctx.AllocRegExcept(r73)
			ctx.W.EmitMovRegMem(r74, r73, 0)
			ctx.FreeReg(r73)
			d118 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.BindReg(r74, &d118)
			ctx.FreeDesc(&d117)
			ctx.EnsureDesc(&d115)
			var d119 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() % 64)}
			} else {
				r75 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r75, d115.Reg)
				ctx.W.EmitAndRegImm32(r75, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d119)
			}
			if d119.Loc == scm.LocReg && d115.Loc == scm.LocReg && d119.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d118.Imm.Int()) << uint64(d119.Imm.Int())))}
			} else if d119.Loc == scm.LocImm {
				r76 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r76, d118.Reg)
				ctx.W.EmitShlRegImm8(r76, uint8(d119.Imm.Int()))
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d120)
			} else {
				{
					shiftSrc := d118.Reg
					r77 := ctx.AllocRegExcept(d118.Reg)
					ctx.W.EmitMovRegReg(r77, d118.Reg)
					shiftSrc = r77
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d119.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d119.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d119.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d120)
				}
			}
			if d120.Loc == scm.LocReg && d118.Loc == scm.LocReg && d120.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d119)
			var d121 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r78, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
				ctx.BindReg(r78, &d121)
			}
			d122 := d121
			ctx.EnsureDesc(&d122)
			if d122.Loc != scm.LocImm && d122.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d122.Loc == scm.LocImm {
				if d122.Imm.Bool() {
					ctx.W.MarkLabel(lbl40)
					ctx.W.EmitJmp(lbl38)
				} else {
					ctx.W.MarkLabel(lbl41)
			d123 := d120
			if d123.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d123)
			ctx.EmitStoreToStack(d123, 88)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d122.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
			d124 := d120
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, 88)
				ctx.W.EmitJmp(lbl39)
			}
			ctx.FreeDesc(&d121)
			ctx.W.MarkLabel(lbl39)
			d125 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d126 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r79 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r79, thisptr.Reg, off)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
				ctx.BindReg(r79, &d126)
			}
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d126)
			var d127 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d126.Imm.Int()))))}
			} else {
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r80, d126.Reg)
				ctx.W.EmitShlRegImm8(r80, 56)
				ctx.W.EmitShrRegImm8(r80, 56)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d127)
			}
			ctx.FreeDesc(&d126)
			d128 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d127)
			var d129 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() - d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r81, d128.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d129)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(scratch, d128.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else {
				r82 := ctx.AllocRegExcept(d128.Reg, d127.Reg)
				ctx.W.EmitMovRegReg(r82, d128.Reg)
				ctx.W.EmitSubInt64(r82, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d129)
			}
			if d129.Loc == scm.LocReg && d128.Loc == scm.LocReg && d129.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d129)
			var d130 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d125.Imm.Int()) >> uint64(d129.Imm.Int())))}
			} else if d129.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r83, d125.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d129.Imm.Int()))
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d130)
			} else {
				{
					shiftSrc := d125.Reg
					r84 := ctx.AllocRegExcept(d125.Reg)
					ctx.W.EmitMovRegReg(r84, d125.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d129.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d129.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d129.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d130)
				}
			}
			if d130.Loc == scm.LocReg && d125.Loc == scm.LocReg && d130.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d129)
			r85 := ctx.AllocReg()
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			if d130.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r85, d130)
			}
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl38)
			d125 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d115)
			var d131 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() % 64)}
			} else {
				r86 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r86, d115.Reg)
				ctx.W.EmitAndRegImm32(r86, 63)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d131)
			}
			if d131.Loc == scm.LocReg && d115.Loc == scm.LocReg && d131.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r87, thisptr.Reg, off)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d132)
			}
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d132)
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d132.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d132.Reg)
				ctx.W.EmitShlRegImm8(r88, 56)
				ctx.W.EmitShrRegImm8(r88, 56)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d133)
			}
			ctx.FreeDesc(&d132)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d133)
			var d134 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d131.Imm.Int() + d133.Imm.Int())}
			} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
				r89 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r89, d131.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d134)
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
				ctx.BindReg(d133.Reg, &d134)
			} else if d131.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d134)
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(scratch, d131.Reg)
				if d133.Imm.Int() >= -2147483648 && d133.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d133.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d134)
			} else {
				r90 := ctx.AllocRegExcept(d131.Reg, d133.Reg)
				ctx.W.EmitMovRegReg(r90, d131.Reg)
				ctx.W.EmitAddInt64(r90, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d134)
			}
			if d134.Loc == scm.LocReg && d131.Loc == scm.LocReg && d134.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.FreeDesc(&d133)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d134.Imm.Int()) > uint64(64))}
			} else {
				r91 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitCmpRegImm32(d134.Reg, 64)
				ctx.W.EmitSetcc(r91, scm.CcA)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r91}
				ctx.BindReg(r91, &d135)
			}
			ctx.FreeDesc(&d134)
			d136 := d135
			ctx.EnsureDesc(&d136)
			if d136.Loc != scm.LocImm && d136.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d136.Loc == scm.LocImm {
				if d136.Imm.Bool() {
					ctx.W.MarkLabel(lbl43)
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.MarkLabel(lbl44)
			d137 := d120
			if d137.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d137)
			ctx.EmitStoreToStack(d137, 88)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d136.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl44)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl44)
			d138 := d120
			if d138.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d138)
			ctx.EmitStoreToStack(d138, 88)
				ctx.W.EmitJmp(lbl39)
			}
			ctx.FreeDesc(&d135)
			ctx.W.MarkLabel(lbl42)
			d125 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d115)
			var d139 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() / 64)}
			} else {
				r92 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r92, d115.Reg)
				ctx.W.EmitShrRegImm8(r92, 6)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d139)
			}
			if d139.Loc == scm.LocReg && d115.Loc == scm.LocReg && d139.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(scratch, d139.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			}
			if d140.Loc == scm.LocReg && d139.Loc == scm.LocReg && d140.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			ctx.EnsureDesc(&d140)
			r93 := ctx.AllocReg()
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d116)
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r93, uint64(d140.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r93, d140.Reg)
				ctx.W.EmitShlRegImm8(r93, 3)
			}
			if d116.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
				ctx.W.EmitAddInt64(r93, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r93, d116.Reg)
			}
			r94 := ctx.AllocRegExcept(r93)
			ctx.W.EmitMovRegMem(r94, r93, 0)
			ctx.FreeReg(r93)
			d141 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r94}
			ctx.BindReg(r94, &d141)
			ctx.FreeDesc(&d140)
			ctx.EnsureDesc(&d115)
			var d142 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() % 64)}
			} else {
				r95 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r95, d115.Reg)
				ctx.W.EmitAndRegImm32(r95, 63)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d142)
			}
			if d142.Loc == scm.LocReg && d115.Loc == scm.LocReg && d142.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			d143 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d142)
			var d144 scm.JITValueDesc
			if d143.Loc == scm.LocImm && d142.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d143.Imm.Int() - d142.Imm.Int())}
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				r96 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r96, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d143.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d142.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(scratch, d143.Reg)
				if d142.Imm.Int() >= -2147483648 && d142.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d142.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d142.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r97 := ctx.AllocRegExcept(d143.Reg, d142.Reg)
				ctx.W.EmitMovRegReg(r97, d143.Reg)
				ctx.W.EmitSubInt64(r97, d142.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d144)
			}
			if d144.Loc == scm.LocReg && d143.Loc == scm.LocReg && d144.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d144)
			var d145 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d141.Imm.Int()) >> uint64(d144.Imm.Int())))}
			} else if d144.Loc == scm.LocImm {
				r98 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r98, d141.Reg)
				ctx.W.EmitShrRegImm8(r98, uint8(d144.Imm.Int()))
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d145)
			} else {
				{
					shiftSrc := d141.Reg
					r99 := ctx.AllocRegExcept(d141.Reg)
					ctx.W.EmitMovRegReg(r99, d141.Reg)
					shiftSrc = r99
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d144.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d144.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d144.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d145)
				}
			}
			if d145.Loc == scm.LocReg && d141.Loc == scm.LocReg && d145.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			ctx.FreeDesc(&d144)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() | d145.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
				ctx.BindReg(d145.Reg, &d146)
			} else if d145.Loc == scm.LocImm && d145.Imm.Int() == 0 {
				r100 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r100, d120.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d146)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d145.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d146)
			} else if d145.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r101, d120.Reg)
				if d145.Imm.Int() >= -2147483648 && d145.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r101, int32(d145.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
					ctx.W.EmitOrInt64(r101, scm.RegR11)
				}
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d146)
			} else {
				r102 := ctx.AllocRegExcept(d120.Reg, d145.Reg)
				ctx.W.EmitMovRegReg(r102, d120.Reg)
				ctx.W.EmitOrInt64(r102, d145.Reg)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d146)
			}
			if d146.Loc == scm.LocReg && d120.Loc == scm.LocReg && d146.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d145)
			d147 := d146
			if d147.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d147)
			ctx.EmitStoreToStack(d147, 88)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl36)
			d148 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.BindReg(r85, &d148)
			ctx.BindReg(r85, &d148)
			if r65 { ctx.UnprotectReg(r66) }
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d148.Imm.Int()))))}
			} else {
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r103, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d149)
			}
			ctx.FreeDesc(&d148)
			var d150 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r104, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
				ctx.BindReg(r104, &d150)
			}
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d150)
			var d151 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() + d150.Imm.Int())}
			} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
				r105 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r105, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d151)
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
				ctx.BindReg(d150.Reg, &d151)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else if d150.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(scratch, d149.Reg)
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d150.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else {
				r106 := ctx.AllocRegExcept(d149.Reg, d150.Reg)
				ctx.W.EmitMovRegReg(r106, d149.Reg)
				ctx.W.EmitAddInt64(r106, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d151)
			}
			if d151.Loc == scm.LocReg && d149.Loc == scm.LocReg && d151.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			ctx.FreeDesc(&d150)
			var d152 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d152)
			}
			d153 := d152
			ctx.EnsureDesc(&d153)
			if d153.Loc != scm.LocImm && d153.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			if d153.Loc == scm.LocImm {
				if d153.Imm.Bool() {
					ctx.W.MarkLabel(lbl47)
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.MarkLabel(lbl48)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d153.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
				ctx.W.MarkLabel(lbl48)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d152)
			ctx.W.MarkLabel(lbl33)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d85)
			d154 := d85
			_ = d154
			r108 := d85.Loc == scm.LocReg
			r109 := d85.Reg
			if r108 { ctx.ProtectReg(r109) }
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl50)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d154.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d154.Reg)
				ctx.W.EmitShlRegImm8(r110, 32)
				ctx.W.EmitShrRegImm8(r110, 32)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d155)
			}
			var d156 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r111, thisptr.Reg, off)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r111}
				ctx.BindReg(r111, &d156)
			}
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d156)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d156.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d156.Reg)
				ctx.W.EmitShlRegImm8(r112, 56)
				ctx.W.EmitShrRegImm8(r112, 56)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d157)
			}
			ctx.FreeDesc(&d156)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() * d157.Imm.Int())}
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d157.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else if d157.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(scratch, d155.Reg)
				if d157.Imm.Int() >= -2147483648 && d157.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d157.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d157.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else {
				r113 := ctx.AllocRegExcept(d155.Reg, d157.Reg)
				ctx.W.EmitMovRegReg(r113, d155.Reg)
				ctx.W.EmitImulInt64(r113, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d158)
			}
			if d158.Loc == scm.LocReg && d155.Loc == scm.LocReg && d158.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			ctx.FreeDesc(&d157)
			var d159 scm.JITValueDesc
			r114 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r114, uint64(dataPtr))
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114, StackOff: int32(sliceLen)}
				ctx.BindReg(r114, &d159)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r114, thisptr.Reg, off)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d159)
			}
			ctx.BindReg(r114, &d159)
			ctx.EnsureDesc(&d158)
			var d160 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() / 64)}
			} else {
				r115 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r115, d158.Reg)
				ctx.W.EmitShrRegImm8(r115, 6)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d160)
			}
			if d160.Loc == scm.LocReg && d158.Loc == scm.LocReg && d160.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d160)
			r116 := ctx.AllocReg()
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d159)
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r116, uint64(d160.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r116, d160.Reg)
				ctx.W.EmitShlRegImm8(r116, 3)
			}
			if d159.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
				ctx.W.EmitAddInt64(r116, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r116, d159.Reg)
			}
			r117 := ctx.AllocRegExcept(r116)
			ctx.W.EmitMovRegMem(r117, r116, 0)
			ctx.FreeReg(r116)
			d161 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
			ctx.BindReg(r117, &d161)
			ctx.FreeDesc(&d160)
			ctx.EnsureDesc(&d158)
			var d162 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() % 64)}
			} else {
				r118 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r118, d158.Reg)
				ctx.W.EmitAndRegImm32(r118, 63)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d162)
			}
			if d162.Loc == scm.LocReg && d158.Loc == scm.LocReg && d162.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			var d163 scm.JITValueDesc
			if d161.Loc == scm.LocImm && d162.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d161.Imm.Int()) << uint64(d162.Imm.Int())))}
			} else if d162.Loc == scm.LocImm {
				r119 := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegReg(r119, d161.Reg)
				ctx.W.EmitShlRegImm8(r119, uint8(d162.Imm.Int()))
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d163)
			} else {
				{
					shiftSrc := d161.Reg
					r120 := ctx.AllocRegExcept(d161.Reg)
					ctx.W.EmitMovRegReg(r120, d161.Reg)
					shiftSrc = r120
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d162.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d162.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d162.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d163)
				}
			}
			if d163.Loc == scm.LocReg && d161.Loc == scm.LocReg && d163.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.FreeDesc(&d162)
			var d164 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
				ctx.BindReg(r121, &d164)
			}
			d165 := d164
			ctx.EnsureDesc(&d165)
			if d165.Loc != scm.LocImm && d165.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d165.Loc == scm.LocImm {
				if d165.Imm.Bool() {
					ctx.W.MarkLabel(lbl53)
					ctx.W.EmitJmp(lbl51)
				} else {
					ctx.W.MarkLabel(lbl54)
			d166 := d163
			if d166.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d166)
			ctx.EmitStoreToStack(d166, 96)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d165.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl51)
				ctx.W.MarkLabel(lbl54)
			d167 := d163
			if d167.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d167)
			ctx.EmitStoreToStack(d167, 96)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d164)
			ctx.W.MarkLabel(lbl52)
			d168 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d169 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r122, thisptr.Reg, off)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
				ctx.BindReg(r122, &d169)
			}
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d169.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r123, d169.Reg)
				ctx.W.EmitShlRegImm8(r123, 56)
				ctx.W.EmitShrRegImm8(r123, 56)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d170)
			}
			ctx.FreeDesc(&d169)
			d171 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d170)
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() - d170.Imm.Int())}
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r124, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d172)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(scratch, d171.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else {
				r125 := ctx.AllocRegExcept(d171.Reg, d170.Reg)
				ctx.W.EmitMovRegReg(r125, d171.Reg)
				ctx.W.EmitSubInt64(r125, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d172)
			}
			if d172.Loc == scm.LocReg && d171.Loc == scm.LocReg && d172.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d168.Imm.Int()) >> uint64(d172.Imm.Int())))}
			} else if d172.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(r126, d168.Reg)
				ctx.W.EmitShrRegImm8(r126, uint8(d172.Imm.Int()))
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d173)
			} else {
				{
					shiftSrc := d168.Reg
					r127 := ctx.AllocRegExcept(d168.Reg)
					ctx.W.EmitMovRegReg(r127, d168.Reg)
					shiftSrc = r127
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d172.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d172.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d172.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d173)
				}
			}
			if d173.Loc == scm.LocReg && d168.Loc == scm.LocReg && d173.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			ctx.FreeDesc(&d172)
			r128 := ctx.AllocReg()
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			if d173.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r128, d173)
			}
			ctx.W.EmitJmp(lbl49)
			ctx.W.MarkLabel(lbl51)
			d168 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d158)
			var d174 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() % 64)}
			} else {
				r129 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r129, d158.Reg)
				ctx.W.EmitAndRegImm32(r129, 63)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d174)
			}
			if d174.Loc == scm.LocReg && d158.Loc == scm.LocReg && d174.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r130, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r130}
				ctx.BindReg(r130, &d175)
			}
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d175.Imm.Int()))))}
			} else {
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r131, d175.Reg)
				ctx.W.EmitShlRegImm8(r131, 56)
				ctx.W.EmitShrRegImm8(r131, 56)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d176)
			}
			ctx.FreeDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() + d176.Imm.Int())}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r132, d174.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d177)
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
				ctx.BindReg(d176.Reg, &d177)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d177)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(scratch, d174.Reg)
				if d176.Imm.Int() >= -2147483648 && d176.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d176.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d177)
			} else {
				r133 := ctx.AllocRegExcept(d174.Reg, d176.Reg)
				ctx.W.EmitMovRegReg(r133, d174.Reg)
				ctx.W.EmitAddInt64(r133, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d177)
			}
			if d177.Loc == scm.LocReg && d174.Loc == scm.LocReg && d177.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.FreeDesc(&d176)
			ctx.EnsureDesc(&d177)
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d177.Imm.Int()) > uint64(64))}
			} else {
				r134 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitCmpRegImm32(d177.Reg, 64)
				ctx.W.EmitSetcc(r134, scm.CcA)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r134}
				ctx.BindReg(r134, &d178)
			}
			ctx.FreeDesc(&d177)
			d179 := d178
			ctx.EnsureDesc(&d179)
			if d179.Loc != scm.LocImm && d179.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
					ctx.W.MarkLabel(lbl56)
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.W.MarkLabel(lbl57)
			d180 := d163
			if d180.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d180)
			ctx.EmitStoreToStack(d180, 96)
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d179.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
				ctx.W.EmitJmp(lbl57)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl57)
			d181 := d163
			if d181.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d181)
			ctx.EmitStoreToStack(d181, 96)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d178)
			ctx.W.MarkLabel(lbl55)
			d168 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d158)
			var d182 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() / 64)}
			} else {
				r135 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r135, d158.Reg)
				ctx.W.EmitShrRegImm8(r135, 6)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d182)
			}
			if d182.Loc == scm.LocReg && d158.Loc == scm.LocReg && d182.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(scratch, d182.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			}
			if d183.Loc == scm.LocReg && d182.Loc == scm.LocReg && d183.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.EnsureDesc(&d183)
			r136 := ctx.AllocReg()
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d159)
			if d183.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r136, uint64(d183.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r136, d183.Reg)
				ctx.W.EmitShlRegImm8(r136, 3)
			}
			if d159.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
				ctx.W.EmitAddInt64(r136, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r136, d159.Reg)
			}
			r137 := ctx.AllocRegExcept(r136)
			ctx.W.EmitMovRegMem(r137, r136, 0)
			ctx.FreeReg(r136)
			d184 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
			ctx.BindReg(r137, &d184)
			ctx.FreeDesc(&d183)
			ctx.EnsureDesc(&d158)
			var d185 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() % 64)}
			} else {
				r138 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r138, d158.Reg)
				ctx.W.EmitAndRegImm32(r138, 63)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d185)
			}
			if d185.Loc == scm.LocReg && d158.Loc == scm.LocReg && d185.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			d186 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d185)
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d186.Imm.Int() - d185.Imm.Int())}
			} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
				r139 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r139, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d187)
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
				r140 := ctx.AllocRegExcept(d186.Reg, d185.Reg)
				ctx.W.EmitMovRegReg(r140, d186.Reg)
				ctx.W.EmitSubInt64(r140, d185.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d187)
			}
			if d187.Loc == scm.LocReg && d186.Loc == scm.LocReg && d187.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d187)
			var d188 scm.JITValueDesc
			if d184.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d184.Imm.Int()) >> uint64(d187.Imm.Int())))}
			} else if d187.Loc == scm.LocImm {
				r141 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r141, d184.Reg)
				ctx.W.EmitShrRegImm8(r141, uint8(d187.Imm.Int()))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d188)
			} else {
				{
					shiftSrc := d184.Reg
					r142 := ctx.AllocRegExcept(d184.Reg)
					ctx.W.EmitMovRegReg(r142, d184.Reg)
					shiftSrc = r142
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
			if d188.Loc == scm.LocReg && d184.Loc == scm.LocReg && d188.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d184)
			ctx.FreeDesc(&d187)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d188)
			var d189 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() | d188.Imm.Int())}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d188.Reg}
				ctx.BindReg(d188.Reg, &d189)
			} else if d188.Loc == scm.LocImm && d188.Imm.Int() == 0 {
				r143 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r143, d163.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d189)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d163.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d189)
			} else if d188.Loc == scm.LocImm {
				r144 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r144, d163.Reg)
				if d188.Imm.Int() >= -2147483648 && d188.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r144, int32(d188.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d188.Imm.Int()))
					ctx.W.EmitOrInt64(r144, scm.RegR11)
				}
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d189)
			} else {
				r145 := ctx.AllocRegExcept(d163.Reg, d188.Reg)
				ctx.W.EmitMovRegReg(r145, d163.Reg)
				ctx.W.EmitOrInt64(r145, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d189)
			}
			if d189.Loc == scm.LocReg && d163.Loc == scm.LocReg && d189.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			d190 := d189
			if d190.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d190)
			ctx.EmitStoreToStack(d190, 96)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl49)
			d191 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			ctx.BindReg(r128, &d191)
			ctx.BindReg(r128, &d191)
			if r108 { ctx.UnprotectReg(r109) }
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d191)
			var d192 scm.JITValueDesc
			if d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d191.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d191.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d192)
			}
			ctx.FreeDesc(&d191)
			var d193 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r147, thisptr.Reg, off)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d193)
			}
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d193)
			var d194 scm.JITValueDesc
			if d192.Loc == scm.LocImm && d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d192.Imm.Int() + d193.Imm.Int())}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				r148 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(r148, d192.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d194)
			} else if d192.Loc == scm.LocImm && d192.Imm.Int() == 0 {
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d193.Reg}
				ctx.BindReg(d193.Reg, &d194)
			} else if d192.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d192.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d194)
			} else if d193.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(scratch, d192.Reg)
				if d193.Imm.Int() >= -2147483648 && d193.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d193.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d193.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d194)
			} else {
				r149 := ctx.AllocRegExcept(d192.Reg, d193.Reg)
				ctx.W.EmitMovRegReg(r149, d192.Reg)
				ctx.W.EmitAddInt64(r149, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d194)
			}
			if d194.Loc == scm.LocReg && d192.Loc == scm.LocReg && d194.Reg == d192.Reg {
				ctx.TransferReg(d192.Reg)
				d192.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d192)
			ctx.FreeDesc(&d193)
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d194)
			var d195 scm.JITValueDesc
			if d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d194.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d194.Reg)
				ctx.W.EmitShlRegImm8(r150, 32)
				ctx.W.EmitShrRegImm8(r150, 32)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d195)
			}
			ctx.FreeDesc(&d194)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d195.Imm.Int()))}
			} else if d195.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(idxInt.Reg)
				if d195.Imm.Int() >= -2147483648 && d195.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d195.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d195.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r151, scm.CcB)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r151}
				ctx.BindReg(r151, &d196)
			} else if idxInt.Loc == scm.LocImm {
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d195.Reg)
				ctx.W.EmitSetcc(r152, scm.CcB)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r152}
				ctx.BindReg(r152, &d196)
			} else {
				r153 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d195.Reg)
				ctx.W.EmitSetcc(r153, scm.CcB)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r153}
				ctx.BindReg(r153, &d196)
			}
			ctx.FreeDesc(&d195)
			d197 := d196
			ctx.EnsureDesc(&d197)
			if d197.Loc != scm.LocImm && d197.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d197.Loc == scm.LocImm {
				if d197.Imm.Bool() {
					ctx.W.MarkLabel(lbl60)
					ctx.W.EmitJmp(lbl58)
				} else {
					ctx.W.MarkLabel(lbl61)
					ctx.W.EmitJmp(lbl59)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d197.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl60)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl59)
			}
			ctx.FreeDesc(&d196)
			ctx.W.MarkLabel(lbl46)
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d109)
			d198 := d109
			_ = d198
			r154 := d109.Loc == scm.LocReg
			r155 := d109.Reg
			if r154 { ctx.ProtectReg(r155) }
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl63)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d198.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, d198.Reg)
				ctx.W.EmitShlRegImm8(r156, 32)
				ctx.W.EmitShrRegImm8(r156, 32)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d199)
			}
			var d200 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r157, thisptr.Reg, off)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d200)
			}
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d200)
			var d201 scm.JITValueDesc
			if d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d200.Imm.Int()))))}
			} else {
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r158, d200.Reg)
				ctx.W.EmitShlRegImm8(r158, 56)
				ctx.W.EmitShrRegImm8(r158, 56)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d201)
			}
			ctx.FreeDesc(&d200)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d199.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d199.Imm.Int() * d201.Imm.Int())}
			} else if d199.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d199.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d199.Reg)
				ctx.W.EmitMovRegReg(scratch, d199.Reg)
				if d201.Imm.Int() >= -2147483648 && d201.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d201.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d201.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else {
				r159 := ctx.AllocRegExcept(d199.Reg, d201.Reg)
				ctx.W.EmitMovRegReg(r159, d199.Reg)
				ctx.W.EmitImulInt64(r159, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d202)
			}
			if d202.Loc == scm.LocReg && d199.Loc == scm.LocReg && d202.Reg == d199.Reg {
				ctx.TransferReg(d199.Reg)
				d199.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d199)
			ctx.FreeDesc(&d201)
			var d203 scm.JITValueDesc
			r160 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r160, uint64(dataPtr))
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160, StackOff: int32(sliceLen)}
				ctx.BindReg(r160, &d203)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r160, thisptr.Reg, off)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d203)
			}
			ctx.BindReg(r160, &d203)
			ctx.EnsureDesc(&d202)
			var d204 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r161, d202.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d204)
			}
			if d204.Loc == scm.LocReg && d202.Loc == scm.LocReg && d204.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d204)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d203)
			if d204.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d204.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d204.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d203.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d203.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d205 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d205)
			ctx.FreeDesc(&d204)
			ctx.EnsureDesc(&d202)
			var d206 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r164, d202.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d206)
			}
			if d206.Loc == scm.LocReg && d202.Loc == scm.LocReg && d206.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d205.Loc == scm.LocImm && d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d205.Imm.Int()) << uint64(d206.Imm.Int())))}
			} else if d206.Loc == scm.LocImm {
				r165 := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegReg(r165, d205.Reg)
				ctx.W.EmitShlRegImm8(r165, uint8(d206.Imm.Int()))
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d207)
			} else {
				{
					shiftSrc := d205.Reg
					r166 := ctx.AllocRegExcept(d205.Reg)
					ctx.W.EmitMovRegReg(r166, d205.Reg)
					shiftSrc = r166
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d206.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d206.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d206.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d207)
				}
			}
			if d207.Loc == scm.LocReg && d205.Loc == scm.LocReg && d207.Reg == d205.Reg {
				ctx.TransferReg(d205.Reg)
				d205.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			ctx.FreeDesc(&d206)
			var d208 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 25)
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r167, thisptr.Reg, off)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
				ctx.BindReg(r167, &d208)
			}
			d209 := d208
			ctx.EnsureDesc(&d209)
			if d209.Loc != scm.LocImm && d209.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			if d209.Loc == scm.LocImm {
				if d209.Imm.Bool() {
					ctx.W.MarkLabel(lbl66)
					ctx.W.EmitJmp(lbl64)
				} else {
					ctx.W.MarkLabel(lbl67)
			d210 := d207
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, 104)
					ctx.W.EmitJmp(lbl65)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d209.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl66)
				ctx.W.EmitJmp(lbl67)
				ctx.W.MarkLabel(lbl66)
				ctx.W.EmitJmp(lbl64)
				ctx.W.MarkLabel(lbl67)
			d211 := d207
			if d211.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d211)
			ctx.EmitStoreToStack(d211, 104)
				ctx.W.EmitJmp(lbl65)
			}
			ctx.FreeDesc(&d208)
			ctx.W.MarkLabel(lbl65)
			d212 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d213 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r168 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r168, thisptr.Reg, off)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r168}
				ctx.BindReg(r168, &d213)
			}
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d213.Imm.Int()))))}
			} else {
				r169 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r169, d213.Reg)
				ctx.W.EmitShlRegImm8(r169, 56)
				ctx.W.EmitShrRegImm8(r169, 56)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d214)
			}
			ctx.FreeDesc(&d213)
			d215 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d214)
			var d216 scm.JITValueDesc
			if d215.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d215.Imm.Int() - d214.Imm.Int())}
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r170 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r170, d215.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d216)
			} else if d215.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d215.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d216)
			} else if d214.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(scratch, d215.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d214.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d216)
			} else {
				r171 := ctx.AllocRegExcept(d215.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r171, d215.Reg)
				ctx.W.EmitSubInt64(r171, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d216)
			}
			if d216.Loc == scm.LocReg && d215.Loc == scm.LocReg && d216.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d212.Imm.Int()) >> uint64(d216.Imm.Int())))}
			} else if d216.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r172, d212.Reg)
				ctx.W.EmitShrRegImm8(r172, uint8(d216.Imm.Int()))
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d217)
			} else {
				{
					shiftSrc := d212.Reg
					r173 := ctx.AllocRegExcept(d212.Reg)
					ctx.W.EmitMovRegReg(r173, d212.Reg)
					shiftSrc = r173
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d216.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d216.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d216.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d217)
				}
			}
			if d217.Loc == scm.LocReg && d212.Loc == scm.LocReg && d217.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			ctx.FreeDesc(&d216)
			r174 := ctx.AllocReg()
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			if d217.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r174, d217)
			}
			ctx.W.EmitJmp(lbl62)
			ctx.W.MarkLabel(lbl64)
			d212 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d202)
			var d218 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() % 64)}
			} else {
				r175 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r175, d202.Reg)
				ctx.W.EmitAndRegImm32(r175, 63)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d218)
			}
			if d218.Loc == scm.LocReg && d202.Loc == scm.LocReg && d218.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r176 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r176, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r176}
				ctx.BindReg(r176, &d219)
			}
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d219.Imm.Int()))))}
			} else {
				r177 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r177, d219.Reg)
				ctx.W.EmitShlRegImm8(r177, 56)
				ctx.W.EmitShrRegImm8(r177, 56)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d220)
			}
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d220)
			var d221 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() + d220.Imm.Int())}
			} else if d220.Loc == scm.LocImm && d220.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(r178, d218.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d221)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d220.Reg}
				ctx.BindReg(d220.Reg, &d221)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d221)
			} else if d220.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(scratch, d218.Reg)
				if d220.Imm.Int() >= -2147483648 && d220.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d220.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d220.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d221)
			} else {
				r179 := ctx.AllocRegExcept(d218.Reg, d220.Reg)
				ctx.W.EmitMovRegReg(r179, d218.Reg)
				ctx.W.EmitAddInt64(r179, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d221)
			}
			if d221.Loc == scm.LocReg && d218.Loc == scm.LocReg && d221.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d221)
			var d222 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d221.Imm.Int()) > uint64(64))}
			} else {
				r180 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitCmpRegImm32(d221.Reg, 64)
				ctx.W.EmitSetcc(r180, scm.CcA)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r180}
				ctx.BindReg(r180, &d222)
			}
			ctx.FreeDesc(&d221)
			d223 := d222
			ctx.EnsureDesc(&d223)
			if d223.Loc != scm.LocImm && d223.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			if d223.Loc == scm.LocImm {
				if d223.Imm.Bool() {
					ctx.W.MarkLabel(lbl69)
					ctx.W.EmitJmp(lbl68)
				} else {
					ctx.W.MarkLabel(lbl70)
			d224 := d207
			if d224.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d224)
			ctx.EmitStoreToStack(d224, 104)
					ctx.W.EmitJmp(lbl65)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d223.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl69)
				ctx.W.EmitJmp(lbl70)
				ctx.W.MarkLabel(lbl69)
				ctx.W.EmitJmp(lbl68)
				ctx.W.MarkLabel(lbl70)
			d225 := d207
			if d225.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d225)
			ctx.EmitStoreToStack(d225, 104)
				ctx.W.EmitJmp(lbl65)
			}
			ctx.FreeDesc(&d222)
			ctx.W.MarkLabel(lbl68)
			d212 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d202)
			var d226 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() / 64)}
			} else {
				r181 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r181, d202.Reg)
				ctx.W.EmitShrRegImm8(r181, 6)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d226)
			}
			if d226.Loc == scm.LocReg && d202.Loc == scm.LocReg && d226.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d226)
			var d227 scm.JITValueDesc
			if d226.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d226.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitMovRegReg(scratch, d226.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			}
			if d227.Loc == scm.LocReg && d226.Loc == scm.LocReg && d227.Reg == d226.Reg {
				ctx.TransferReg(d226.Reg)
				d226.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d226)
			ctx.EnsureDesc(&d227)
			r182 := ctx.AllocReg()
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d203)
			if d227.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d227.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r182, d227.Reg)
				ctx.W.EmitShlRegImm8(r182, 3)
			}
			if d203.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
				ctx.W.EmitAddInt64(r182, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r182, d203.Reg)
			}
			r183 := ctx.AllocRegExcept(r182)
			ctx.W.EmitMovRegMem(r183, r182, 0)
			ctx.FreeReg(r182)
			d228 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
			ctx.BindReg(r183, &d228)
			ctx.FreeDesc(&d227)
			ctx.EnsureDesc(&d202)
			var d229 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() % 64)}
			} else {
				r184 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r184, d202.Reg)
				ctx.W.EmitAndRegImm32(r184, 63)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d229)
			}
			if d229.Loc == scm.LocReg && d202.Loc == scm.LocReg && d229.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d202)
			d230 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d230)
			ctx.EnsureDesc(&d229)
			var d231 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d230.Imm.Int() - d229.Imm.Int())}
			} else if d229.Loc == scm.LocImm && d229.Imm.Int() == 0 {
				r185 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(r185, d230.Reg)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d231)
			} else if d230.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d229.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d230.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d229.Reg)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d231)
			} else if d229.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitMovRegReg(scratch, d230.Reg)
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d229.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d231)
			} else {
				r186 := ctx.AllocRegExcept(d230.Reg, d229.Reg)
				ctx.W.EmitMovRegReg(r186, d230.Reg)
				ctx.W.EmitSubInt64(r186, d229.Reg)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d231)
			}
			if d231.Loc == scm.LocReg && d230.Loc == scm.LocReg && d231.Reg == d230.Reg {
				ctx.TransferReg(d230.Reg)
				d230.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d229)
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d231)
			var d232 scm.JITValueDesc
			if d228.Loc == scm.LocImm && d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d228.Imm.Int()) >> uint64(d231.Imm.Int())))}
			} else if d231.Loc == scm.LocImm {
				r187 := ctx.AllocRegExcept(d228.Reg)
				ctx.W.EmitMovRegReg(r187, d228.Reg)
				ctx.W.EmitShrRegImm8(r187, uint8(d231.Imm.Int()))
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d232)
			} else {
				{
					shiftSrc := d228.Reg
					r188 := ctx.AllocRegExcept(d228.Reg)
					ctx.W.EmitMovRegReg(r188, d228.Reg)
					shiftSrc = r188
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d231.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d231.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d231.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d232)
				}
			}
			if d232.Loc == scm.LocReg && d228.Loc == scm.LocReg && d232.Reg == d228.Reg {
				ctx.TransferReg(d228.Reg)
				d228.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d228)
			ctx.FreeDesc(&d231)
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() | d232.Imm.Int())}
			} else if d207.Loc == scm.LocImm && d207.Imm.Int() == 0 {
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d232.Reg}
				ctx.BindReg(d232.Reg, &d233)
			} else if d232.Loc == scm.LocImm && d232.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r189, d207.Reg)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d233)
			} else if d207.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d232.Reg)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d233)
			} else if d232.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r190, d207.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r190, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitOrInt64(r190, scm.RegR11)
				}
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d233)
			} else {
				r191 := ctx.AllocRegExcept(d207.Reg, d232.Reg)
				ctx.W.EmitMovRegReg(r191, d207.Reg)
				ctx.W.EmitOrInt64(r191, d232.Reg)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d233)
			}
			if d233.Loc == scm.LocReg && d207.Loc == scm.LocReg && d233.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d232)
			d234 := d233
			if d234.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d234)
			ctx.EmitStoreToStack(d234, 104)
			ctx.W.EmitJmp(lbl65)
			ctx.W.MarkLabel(lbl62)
			d235 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r174}
			ctx.BindReg(r174, &d235)
			ctx.BindReg(r174, &d235)
			if r154 { ctx.UnprotectReg(r155) }
			ctx.EnsureDesc(&d235)
			ctx.EnsureDesc(&d235)
			var d236 scm.JITValueDesc
			if d235.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d235.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r192, d235.Reg)
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d236)
			}
			ctx.FreeDesc(&d235)
			var d237 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r193, thisptr.Reg, off)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
				ctx.BindReg(r193, &d237)
			}
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d237)
			var d238 scm.JITValueDesc
			if d236.Loc == scm.LocImm && d237.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d236.Imm.Int() + d237.Imm.Int())}
			} else if d237.Loc == scm.LocImm && d237.Imm.Int() == 0 {
				r194 := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(r194, d236.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d238)
			} else if d236.Loc == scm.LocImm && d236.Imm.Int() == 0 {
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d237.Reg}
				ctx.BindReg(d237.Reg, &d238)
			} else if d236.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d236.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d237.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d238)
			} else if d237.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(scratch, d236.Reg)
				if d237.Imm.Int() >= -2147483648 && d237.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d237.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d237.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d238)
			} else {
				r195 := ctx.AllocRegExcept(d236.Reg, d237.Reg)
				ctx.W.EmitMovRegReg(r195, d236.Reg)
				ctx.W.EmitAddInt64(r195, d237.Reg)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d238)
			}
			if d238.Loc == scm.LocReg && d236.Loc == scm.LocReg && d238.Reg == d236.Reg {
				ctx.TransferReg(d236.Reg)
				d236.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.FreeDesc(&d237)
			ctx.EnsureDesc(&d109)
			d239 := d109
			_ = d239
			r196 := d109.Loc == scm.LocReg
			r197 := d109.Reg
			if r196 { ctx.ProtectReg(r197) }
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl72)
			ctx.EnsureDesc(&d239)
			ctx.EnsureDesc(&d239)
			var d240 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d239.Imm.Int()))))}
			} else {
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r198, d239.Reg)
				ctx.W.EmitShlRegImm8(r198, 32)
				ctx.W.EmitShrRegImm8(r198, 32)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d240)
			}
			var d241 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r199, thisptr.Reg, off)
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d241)
			}
			ctx.EnsureDesc(&d241)
			ctx.EnsureDesc(&d241)
			var d242 scm.JITValueDesc
			if d241.Loc == scm.LocImm {
				d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d241.Imm.Int()))))}
			} else {
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r200, d241.Reg)
				ctx.W.EmitShlRegImm8(r200, 56)
				ctx.W.EmitShrRegImm8(r200, 56)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d242)
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
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d240.Imm.Int() * d242.Imm.Int())}
			} else if d240.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d242.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d240.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d242.Reg)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d243)
			} else if d242.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d240.Reg)
				ctx.W.EmitMovRegReg(scratch, d240.Reg)
				if d242.Imm.Int() >= -2147483648 && d242.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d242.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d242.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d243)
			} else {
				r201 := ctx.AllocRegExcept(d240.Reg, d242.Reg)
				ctx.W.EmitMovRegReg(r201, d240.Reg)
				ctx.W.EmitImulInt64(r201, d242.Reg)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d243)
			}
			if d243.Loc == scm.LocReg && d240.Loc == scm.LocReg && d243.Reg == d240.Reg {
				ctx.TransferReg(d240.Reg)
				d240.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d240)
			ctx.FreeDesc(&d242)
			var d244 scm.JITValueDesc
			r202 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r202, uint64(dataPtr))
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202, StackOff: int32(sliceLen)}
				ctx.BindReg(r202, &d244)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r202, thisptr.Reg, off)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d244)
			}
			ctx.BindReg(r202, &d244)
			ctx.EnsureDesc(&d243)
			var d245 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() / 64)}
			} else {
				r203 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(r203, d243.Reg)
				ctx.W.EmitShrRegImm8(r203, 6)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d245)
			}
			if d245.Loc == scm.LocReg && d243.Loc == scm.LocReg && d245.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d245)
			r204 := ctx.AllocReg()
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d244)
			if d245.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r204, uint64(d245.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r204, d245.Reg)
				ctx.W.EmitShlRegImm8(r204, 3)
			}
			if d244.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d244.Imm.Int()))
				ctx.W.EmitAddInt64(r204, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r204, d244.Reg)
			}
			r205 := ctx.AllocRegExcept(r204)
			ctx.W.EmitMovRegMem(r205, r204, 0)
			ctx.FreeReg(r204)
			d246 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
			ctx.BindReg(r205, &d246)
			ctx.FreeDesc(&d245)
			ctx.EnsureDesc(&d243)
			var d247 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() % 64)}
			} else {
				r206 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(r206, d243.Reg)
				ctx.W.EmitAndRegImm32(r206, 63)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d247)
			}
			if d247.Loc == scm.LocReg && d243.Loc == scm.LocReg && d247.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d247)
			var d248 scm.JITValueDesc
			if d246.Loc == scm.LocImm && d247.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d246.Imm.Int()) << uint64(d247.Imm.Int())))}
			} else if d247.Loc == scm.LocImm {
				r207 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r207, d246.Reg)
				ctx.W.EmitShlRegImm8(r207, uint8(d247.Imm.Int()))
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d248)
			} else {
				{
					shiftSrc := d246.Reg
					r208 := ctx.AllocRegExcept(d246.Reg)
					ctx.W.EmitMovRegReg(r208, d246.Reg)
					shiftSrc = r208
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d247.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d247.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d247.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d248)
				}
			}
			if d248.Loc == scm.LocReg && d246.Loc == scm.LocReg && d248.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			ctx.FreeDesc(&d247)
			var d249 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r209 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r209, thisptr.Reg, off)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
				ctx.BindReg(r209, &d249)
			}
			d250 := d249
			ctx.EnsureDesc(&d250)
			if d250.Loc != scm.LocImm && d250.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			if d250.Loc == scm.LocImm {
				if d250.Imm.Bool() {
					ctx.W.MarkLabel(lbl75)
					ctx.W.EmitJmp(lbl73)
				} else {
					ctx.W.MarkLabel(lbl76)
			d251 := d248
			if d251.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d251)
			ctx.EmitStoreToStack(d251, 112)
					ctx.W.EmitJmp(lbl74)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d250.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl75)
				ctx.W.EmitJmp(lbl76)
				ctx.W.MarkLabel(lbl75)
				ctx.W.EmitJmp(lbl73)
				ctx.W.MarkLabel(lbl76)
			d252 := d248
			if d252.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d252)
			ctx.EmitStoreToStack(d252, 112)
				ctx.W.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d249)
			ctx.W.MarkLabel(lbl74)
			d253 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d254 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d254)
			}
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d254)
			var d255 scm.JITValueDesc
			if d254.Loc == scm.LocImm {
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d254.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d254.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d255)
			}
			ctx.FreeDesc(&d254)
			d256 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d255)
			var d257 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d255.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d256.Imm.Int() - d255.Imm.Int())}
			} else if d255.Loc == scm.LocImm && d255.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r212, d256.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d257)
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
				r213 := ctx.AllocRegExcept(d256.Reg, d255.Reg)
				ctx.W.EmitMovRegReg(r213, d256.Reg)
				ctx.W.EmitSubInt64(r213, d255.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d257)
			}
			if d257.Loc == scm.LocReg && d256.Loc == scm.LocReg && d257.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d255)
			ctx.EnsureDesc(&d253)
			ctx.EnsureDesc(&d257)
			var d258 scm.JITValueDesc
			if d253.Loc == scm.LocImm && d257.Loc == scm.LocImm {
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d253.Imm.Int()) >> uint64(d257.Imm.Int())))}
			} else if d257.Loc == scm.LocImm {
				r214 := ctx.AllocRegExcept(d253.Reg)
				ctx.W.EmitMovRegReg(r214, d253.Reg)
				ctx.W.EmitShrRegImm8(r214, uint8(d257.Imm.Int()))
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d258)
			} else {
				{
					shiftSrc := d253.Reg
					r215 := ctx.AllocRegExcept(d253.Reg)
					ctx.W.EmitMovRegReg(r215, d253.Reg)
					shiftSrc = r215
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d257.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d257.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d257.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d258)
				}
			}
			if d258.Loc == scm.LocReg && d253.Loc == scm.LocReg && d258.Reg == d253.Reg {
				ctx.TransferReg(d253.Reg)
				d253.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d253)
			ctx.FreeDesc(&d257)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d258)
			if d258.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r216, d258)
			}
			ctx.W.EmitJmp(lbl71)
			ctx.W.MarkLabel(lbl73)
			d253 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d243)
			var d259 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() % 64)}
			} else {
				r217 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(r217, d243.Reg)
				ctx.W.EmitAndRegImm32(r217, 63)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d259)
			}
			if d259.Loc == scm.LocReg && d243.Loc == scm.LocReg && d259.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			var d260 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r218 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r218, thisptr.Reg, off)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
				ctx.BindReg(r218, &d260)
			}
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d260)
			var d261 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d260.Imm.Int()))))}
			} else {
				r219 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r219, d260.Reg)
				ctx.W.EmitShlRegImm8(r219, 56)
				ctx.W.EmitShrRegImm8(r219, 56)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d261)
			}
			ctx.FreeDesc(&d260)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d261)
			var d262 scm.JITValueDesc
			if d259.Loc == scm.LocImm && d261.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d259.Imm.Int() + d261.Imm.Int())}
			} else if d261.Loc == scm.LocImm && d261.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(r220, d259.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d262)
			} else if d259.Loc == scm.LocImm && d259.Imm.Int() == 0 {
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d261.Reg}
				ctx.BindReg(d261.Reg, &d262)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d259.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d261.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else if d261.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(scratch, d259.Reg)
				if d261.Imm.Int() >= -2147483648 && d261.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d261.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d261.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			} else {
				r221 := ctx.AllocRegExcept(d259.Reg, d261.Reg)
				ctx.W.EmitMovRegReg(r221, d259.Reg)
				ctx.W.EmitAddInt64(r221, d261.Reg)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d262)
			}
			if d262.Loc == scm.LocReg && d259.Loc == scm.LocReg && d262.Reg == d259.Reg {
				ctx.TransferReg(d259.Reg)
				d259.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d259)
			ctx.FreeDesc(&d261)
			ctx.EnsureDesc(&d262)
			var d263 scm.JITValueDesc
			if d262.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d262.Imm.Int()) > uint64(64))}
			} else {
				r222 := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitCmpRegImm32(d262.Reg, 64)
				ctx.W.EmitSetcc(r222, scm.CcA)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r222}
				ctx.BindReg(r222, &d263)
			}
			ctx.FreeDesc(&d262)
			d264 := d263
			ctx.EnsureDesc(&d264)
			if d264.Loc != scm.LocImm && d264.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			if d264.Loc == scm.LocImm {
				if d264.Imm.Bool() {
					ctx.W.MarkLabel(lbl78)
					ctx.W.EmitJmp(lbl77)
				} else {
					ctx.W.MarkLabel(lbl79)
			d265 := d248
			if d265.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d265)
			ctx.EmitStoreToStack(d265, 112)
					ctx.W.EmitJmp(lbl74)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d264.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl78)
				ctx.W.EmitJmp(lbl79)
				ctx.W.MarkLabel(lbl78)
				ctx.W.EmitJmp(lbl77)
				ctx.W.MarkLabel(lbl79)
			d266 := d248
			if d266.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d266)
			ctx.EmitStoreToStack(d266, 112)
				ctx.W.EmitJmp(lbl74)
			}
			ctx.FreeDesc(&d263)
			ctx.W.MarkLabel(lbl77)
			d253 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d243)
			var d267 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() / 64)}
			} else {
				r223 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(r223, d243.Reg)
				ctx.W.EmitShrRegImm8(r223, 6)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d267)
			}
			if d267.Loc == scm.LocReg && d243.Loc == scm.LocReg && d267.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d267.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d267.Reg)
				ctx.W.EmitMovRegReg(scratch, d267.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			}
			if d268.Loc == scm.LocReg && d267.Loc == scm.LocReg && d268.Reg == d267.Reg {
				ctx.TransferReg(d267.Reg)
				d267.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d267)
			ctx.EnsureDesc(&d268)
			r224 := ctx.AllocReg()
			ctx.EnsureDesc(&d268)
			ctx.EnsureDesc(&d244)
			if d268.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r224, uint64(d268.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r224, d268.Reg)
				ctx.W.EmitShlRegImm8(r224, 3)
			}
			if d244.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d244.Imm.Int()))
				ctx.W.EmitAddInt64(r224, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r224, d244.Reg)
			}
			r225 := ctx.AllocRegExcept(r224)
			ctx.W.EmitMovRegMem(r225, r224, 0)
			ctx.FreeReg(r224)
			d269 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r225}
			ctx.BindReg(r225, &d269)
			ctx.FreeDesc(&d268)
			ctx.EnsureDesc(&d243)
			var d270 scm.JITValueDesc
			if d243.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() % 64)}
			} else {
				r226 := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(r226, d243.Reg)
				ctx.W.EmitAndRegImm32(r226, 63)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d270)
			}
			if d270.Loc == scm.LocReg && d243.Loc == scm.LocReg && d270.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d243)
			d271 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d270)
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d270)
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d270)
			var d272 scm.JITValueDesc
			if d271.Loc == scm.LocImm && d270.Loc == scm.LocImm {
				d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d271.Imm.Int() - d270.Imm.Int())}
			} else if d270.Loc == scm.LocImm && d270.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d271.Reg)
				ctx.W.EmitMovRegReg(r227, d271.Reg)
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d272)
			} else if d271.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d270.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d271.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d270.Reg)
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d272)
			} else if d270.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d271.Reg)
				ctx.W.EmitMovRegReg(scratch, d271.Reg)
				if d270.Imm.Int() >= -2147483648 && d270.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d270.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d270.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d272)
			} else {
				r228 := ctx.AllocRegExcept(d271.Reg, d270.Reg)
				ctx.W.EmitMovRegReg(r228, d271.Reg)
				ctx.W.EmitSubInt64(r228, d270.Reg)
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d272)
			}
			if d272.Loc == scm.LocReg && d271.Loc == scm.LocReg && d272.Reg == d271.Reg {
				ctx.TransferReg(d271.Reg)
				d271.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d270)
			ctx.EnsureDesc(&d269)
			ctx.EnsureDesc(&d272)
			var d273 scm.JITValueDesc
			if d269.Loc == scm.LocImm && d272.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d269.Imm.Int()) >> uint64(d272.Imm.Int())))}
			} else if d272.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d269.Reg)
				ctx.W.EmitMovRegReg(r229, d269.Reg)
				ctx.W.EmitShrRegImm8(r229, uint8(d272.Imm.Int()))
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d273)
			} else {
				{
					shiftSrc := d269.Reg
					r230 := ctx.AllocRegExcept(d269.Reg)
					ctx.W.EmitMovRegReg(r230, d269.Reg)
					shiftSrc = r230
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d272.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d272.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d272.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d273)
				}
			}
			if d273.Loc == scm.LocReg && d269.Loc == scm.LocReg && d273.Reg == d269.Reg {
				ctx.TransferReg(d269.Reg)
				d269.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d269)
			ctx.FreeDesc(&d272)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d273)
			var d274 scm.JITValueDesc
			if d248.Loc == scm.LocImm && d273.Loc == scm.LocImm {
				d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d248.Imm.Int() | d273.Imm.Int())}
			} else if d248.Loc == scm.LocImm && d248.Imm.Int() == 0 {
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d273.Reg}
				ctx.BindReg(d273.Reg, &d274)
			} else if d273.Loc == scm.LocImm && d273.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r231, d248.Reg)
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d274)
			} else if d248.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d273.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d248.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d273.Reg)
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d274)
			} else if d273.Loc == scm.LocImm {
				r232 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r232, d248.Reg)
				if d273.Imm.Int() >= -2147483648 && d273.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r232, int32(d273.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d273.Imm.Int()))
					ctx.W.EmitOrInt64(r232, scm.RegR11)
				}
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d274)
			} else {
				r233 := ctx.AllocRegExcept(d248.Reg, d273.Reg)
				ctx.W.EmitMovRegReg(r233, d248.Reg)
				ctx.W.EmitOrInt64(r233, d273.Reg)
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d274)
			}
			if d274.Loc == scm.LocReg && d248.Loc == scm.LocReg && d274.Reg == d248.Reg {
				ctx.TransferReg(d248.Reg)
				d248.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d273)
			d275 := d274
			if d275.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d275)
			ctx.EmitStoreToStack(d275, 112)
			ctx.W.EmitJmp(lbl74)
			ctx.W.MarkLabel(lbl71)
			d276 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
			ctx.BindReg(r216, &d276)
			ctx.BindReg(r216, &d276)
			if r196 { ctx.UnprotectReg(r197) }
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d276)
			var d277 scm.JITValueDesc
			if d276.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d276.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d276.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d277)
			}
			ctx.FreeDesc(&d276)
			var d278 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r235, thisptr.Reg, off)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r235}
				ctx.BindReg(r235, &d278)
			}
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d278)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d278)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d278)
			var d279 scm.JITValueDesc
			if d277.Loc == scm.LocImm && d278.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d277.Imm.Int() + d278.Imm.Int())}
			} else if d278.Loc == scm.LocImm && d278.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d277.Reg)
				ctx.W.EmitMovRegReg(r236, d277.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d279)
			} else if d277.Loc == scm.LocImm && d277.Imm.Int() == 0 {
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d278.Reg}
				ctx.BindReg(d278.Reg, &d279)
			} else if d277.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d278.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d277.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d278.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d279)
			} else if d278.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d277.Reg)
				ctx.W.EmitMovRegReg(scratch, d277.Reg)
				if d278.Imm.Int() >= -2147483648 && d278.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d278.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d278.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d279)
			} else {
				r237 := ctx.AllocRegExcept(d277.Reg, d278.Reg)
				ctx.W.EmitMovRegReg(r237, d277.Reg)
				ctx.W.EmitAddInt64(r237, d278.Reg)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d279)
			}
			if d279.Loc == scm.LocReg && d277.Loc == scm.LocReg && d279.Reg == d277.Reg {
				ctx.TransferReg(d277.Reg)
				d277.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d277)
			ctx.FreeDesc(&d278)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d280 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d280 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d280)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d279)
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d279)
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d279)
			var d281 scm.JITValueDesc
			if d280.Loc == scm.LocImm && d279.Loc == scm.LocImm {
				d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d280.Imm.Int() - d279.Imm.Int())}
			} else if d279.Loc == scm.LocImm && d279.Imm.Int() == 0 {
				r239 := ctx.AllocRegExcept(d280.Reg)
				ctx.W.EmitMovRegReg(r239, d280.Reg)
				d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d281)
			} else if d280.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d279.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d280.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d279.Reg)
				d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d281)
			} else if d279.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d280.Reg)
				ctx.W.EmitMovRegReg(scratch, d280.Reg)
				if d279.Imm.Int() >= -2147483648 && d279.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d279.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d279.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d281)
			} else {
				r240 := ctx.AllocRegExcept(d280.Reg, d279.Reg)
				ctx.W.EmitMovRegReg(r240, d280.Reg)
				ctx.W.EmitSubInt64(r240, d279.Reg)
				d281 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d281)
			}
			if d281.Loc == scm.LocReg && d280.Loc == scm.LocReg && d281.Reg == d280.Reg {
				ctx.TransferReg(d280.Reg)
				d280.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d280)
			ctx.FreeDesc(&d279)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d238)
			var d282 scm.JITValueDesc
			if d281.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d281.Imm.Int() * d238.Imm.Int())}
			} else if d281.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d281.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d238.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d282)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d281.Reg)
				ctx.W.EmitMovRegReg(scratch, d281.Reg)
				if d238.Imm.Int() >= -2147483648 && d238.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d238.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d238.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d282)
			} else {
				r241 := ctx.AllocRegExcept(d281.Reg, d238.Reg)
				ctx.W.EmitMovRegReg(r241, d281.Reg)
				ctx.W.EmitImulInt64(r241, d238.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d282)
			}
			if d282.Loc == scm.LocReg && d281.Loc == scm.LocReg && d282.Reg == d281.Reg {
				ctx.TransferReg(d281.Reg)
				d281.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d281)
			ctx.FreeDesc(&d238)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d282)
			var d283 scm.JITValueDesc
			if d151.Loc == scm.LocImm && d282.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() + d282.Imm.Int())}
			} else if d282.Loc == scm.LocImm && d282.Imm.Int() == 0 {
				r242 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r242, d151.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d283)
			} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d282.Reg}
				ctx.BindReg(d282.Reg, &d283)
			} else if d151.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d282.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d151.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d282.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d283)
			} else if d282.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(scratch, d151.Reg)
				if d282.Imm.Int() >= -2147483648 && d282.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d282.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d282.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d283)
			} else {
				r243 := ctx.AllocRegExcept(d151.Reg, d282.Reg)
				ctx.W.EmitMovRegReg(r243, d151.Reg)
				ctx.W.EmitAddInt64(r243, d282.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r243}
				ctx.BindReg(r243, &d283)
			}
			if d283.Loc == scm.LocReg && d151.Loc == scm.LocReg && d283.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d282)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d283)
			var d284 scm.JITValueDesc
			if d283.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d283.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d283.Reg)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d283.Reg}
				ctx.BindReg(d283.Reg, &d284)
			}
			ctx.FreeDesc(&d283)
			ctx.EnsureDesc(&d284)
			d285 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d285)
			ctx.BindReg(r2, &d285)
			ctx.EnsureDesc(&d284)
			ctx.W.EmitMakeFloat(d285, d284)
			if d284.Loc == scm.LocReg { ctx.FreeReg(d284.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl45)
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			var d286 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d286 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r244, thisptr.Reg, off)
				d286 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r244}
				ctx.BindReg(r244, &d286)
			}
			ctx.EnsureDesc(&d286)
			ctx.EnsureDesc(&d286)
			var d287 scm.JITValueDesc
			if d286.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d286.Imm.Int()))))}
			} else {
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r245, d286.Reg)
				d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r245}
				ctx.BindReg(r245, &d287)
			}
			ctx.FreeDesc(&d286)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d287)
			var d288 scm.JITValueDesc
			if d151.Loc == scm.LocImm && d287.Loc == scm.LocImm {
				d288 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d151.Imm.Int() == d287.Imm.Int())}
			} else if d287.Loc == scm.LocImm {
				r246 := ctx.AllocRegExcept(d151.Reg)
				if d287.Imm.Int() >= -2147483648 && d287.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d151.Reg, int32(d287.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d287.Imm.Int()))
					ctx.W.EmitCmpInt64(d151.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r246, scm.CcE)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d288)
			} else if d151.Loc == scm.LocImm {
				r247 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d287.Reg)
				ctx.W.EmitSetcc(r247, scm.CcE)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r247}
				ctx.BindReg(r247, &d288)
			} else {
				r248 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitCmpInt64(d151.Reg, d287.Reg)
				ctx.W.EmitSetcc(r248, scm.CcE)
				d288 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r248}
				ctx.BindReg(r248, &d288)
			}
			ctx.FreeDesc(&d151)
			ctx.FreeDesc(&d287)
			d289 := d288
			ctx.EnsureDesc(&d289)
			if d289.Loc != scm.LocImm && d289.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl80 := ctx.W.ReserveLabel()
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			if d289.Loc == scm.LocImm {
				if d289.Imm.Bool() {
					ctx.W.MarkLabel(lbl81)
					ctx.W.EmitJmp(lbl80)
				} else {
					ctx.W.MarkLabel(lbl82)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d289.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl81)
				ctx.W.EmitJmp(lbl82)
				ctx.W.MarkLabel(lbl81)
				ctx.W.EmitJmp(lbl80)
				ctx.W.MarkLabel(lbl82)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d288)
			ctx.W.MarkLabel(lbl59)
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			lbl83 := ctx.W.ReserveLabel()
			d290 := d85
			if d290.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d290)
			d291 := d290
			if d291.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: d291.Type, Imm: scm.NewInt(int64(uint64(d291.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d291.Reg, 32)
				ctx.W.EmitShrRegImm8(d291.Reg, 32)
			}
			ctx.EmitStoreToStack(d291, 64)
			d292 := d87
			if d292.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d292)
			d293 := d292
			if d293.Loc == scm.LocImm {
				d293 = scm.JITValueDesc{Loc: scm.LocImm, Type: d293.Type, Imm: scm.NewInt(int64(uint64(d293.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d293.Reg, 32)
				ctx.W.EmitShrRegImm8(d293.Reg, 32)
			}
			ctx.EmitStoreToStack(d293, 72)
			ctx.W.MarkLabel(lbl83)
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d294 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d295 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			var d296 scm.JITValueDesc
			if d294.Loc == scm.LocImm && d295.Loc == scm.LocImm {
				d296 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d294.Imm.Int()) == uint64(d295.Imm.Int()))}
			} else if d295.Loc == scm.LocImm {
				r249 := ctx.AllocRegExcept(d294.Reg)
				if d295.Imm.Int() >= -2147483648 && d295.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d294.Reg, int32(d295.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d295.Imm.Int()))
					ctx.W.EmitCmpInt64(d294.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r249, scm.CcE)
				d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r249}
				ctx.BindReg(r249, &d296)
			} else if d294.Loc == scm.LocImm {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d294.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d295.Reg)
				ctx.W.EmitSetcc(r250, scm.CcE)
				d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r250}
				ctx.BindReg(r250, &d296)
			} else {
				r251 := ctx.AllocRegExcept(d294.Reg)
				ctx.W.EmitCmpInt64(d294.Reg, d295.Reg)
				ctx.W.EmitSetcc(r251, scm.CcE)
				d296 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d296)
			}
			d297 := d296
			ctx.EnsureDesc(&d297)
			if d297.Loc != scm.LocImm && d297.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl84 := ctx.W.ReserveLabel()
			lbl85 := ctx.W.ReserveLabel()
			lbl86 := ctx.W.ReserveLabel()
			if d297.Loc == scm.LocImm {
				if d297.Imm.Bool() {
					ctx.W.MarkLabel(lbl85)
			d298 := d294
			if d298.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d298)
			d299 := d298
			if d299.Loc == scm.LocImm {
				d299 = scm.JITValueDesc{Loc: scm.LocImm, Type: d299.Type, Imm: scm.NewInt(int64(uint64(d299.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d299.Reg, 32)
				ctx.W.EmitShrRegImm8(d299.Reg, 32)
			}
			ctx.EmitStoreToStack(d299, 32)
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.W.MarkLabel(lbl86)
					ctx.W.EmitJmp(lbl84)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d297.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl85)
				ctx.W.EmitJmp(lbl86)
				ctx.W.MarkLabel(lbl85)
			d300 := d294
			if d300.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d300)
			d301 := d300
			if d301.Loc == scm.LocImm {
				d301 = scm.JITValueDesc{Loc: scm.LocImm, Type: d301.Type, Imm: scm.NewInt(int64(uint64(d301.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d301.Reg, 32)
				ctx.W.EmitShrRegImm8(d301.Reg, 32)
			}
			ctx.EmitStoreToStack(d301, 32)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl86)
				ctx.W.EmitJmp(lbl84)
			}
			ctx.FreeDesc(&d296)
			ctx.W.MarkLabel(lbl58)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d295 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d294 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EnsureDesc(&d85)
			var d302 scm.JITValueDesc
			if d85.Loc == scm.LocImm {
				d302 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d85.Imm.Int()) == uint64(0))}
			} else {
				r252 := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitCmpRegImm32(d85.Reg, 0)
				ctx.W.EmitSetcc(r252, scm.CcE)
				d302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d302)
			}
			d303 := d302
			ctx.EnsureDesc(&d303)
			if d303.Loc != scm.LocImm && d303.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl87 := ctx.W.ReserveLabel()
			lbl88 := ctx.W.ReserveLabel()
			lbl89 := ctx.W.ReserveLabel()
			lbl90 := ctx.W.ReserveLabel()
			if d303.Loc == scm.LocImm {
				if d303.Imm.Bool() {
					ctx.W.MarkLabel(lbl89)
					ctx.W.EmitJmp(lbl87)
				} else {
					ctx.W.MarkLabel(lbl90)
					ctx.W.EmitJmp(lbl88)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d303.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl89)
				ctx.W.EmitJmp(lbl90)
				ctx.W.MarkLabel(lbl89)
				ctx.W.EmitJmp(lbl87)
				ctx.W.MarkLabel(lbl90)
				ctx.W.EmitJmp(lbl88)
			}
			ctx.FreeDesc(&d302)
			ctx.W.MarkLabel(lbl80)
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d294 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d295 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d304 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d304)
			ctx.BindReg(r2, &d304)
			ctx.W.EmitMakeNil(d304)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl84)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d295 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d294 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			ctx.EnsureDesc(&d294)
			ctx.EnsureDesc(&d295)
			var d305 scm.JITValueDesc
			if d294.Loc == scm.LocImm && d295.Loc == scm.LocImm {
				d305 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d294.Imm.Int() + d295.Imm.Int())}
			} else if d295.Loc == scm.LocImm && d295.Imm.Int() == 0 {
				r253 := ctx.AllocRegExcept(d294.Reg)
				ctx.W.EmitMovRegReg(r253, d294.Reg)
				d305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d305)
			} else if d294.Loc == scm.LocImm && d294.Imm.Int() == 0 {
				d305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d295.Reg}
				ctx.BindReg(d295.Reg, &d305)
			} else if d294.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d295.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d294.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d295.Reg)
				d305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d305)
			} else if d295.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d294.Reg)
				ctx.W.EmitMovRegReg(scratch, d294.Reg)
				if d295.Imm.Int() >= -2147483648 && d295.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d295.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d295.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d305)
			} else {
				r254 := ctx.AllocRegExcept(d294.Reg, d295.Reg)
				ctx.W.EmitMovRegReg(r254, d294.Reg)
				ctx.W.EmitAddInt64(r254, d295.Reg)
				d305 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d305)
			}
			if d305.Loc == scm.LocImm {
				d305 = scm.JITValueDesc{Loc: scm.LocImm, Type: d305.Type, Imm: scm.NewInt(int64(uint64(d305.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d305.Reg, 32)
				ctx.W.EmitShrRegImm8(d305.Reg, 32)
			}
			if d305.Loc == scm.LocReg && d294.Loc == scm.LocReg && d305.Reg == d294.Reg {
				ctx.TransferReg(d294.Reg)
				d294.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d305)
			var d306 scm.JITValueDesc
			if d305.Loc == scm.LocImm {
				d306 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d305.Imm.Int() / 2)}
			} else {
				r255 := ctx.AllocRegExcept(d305.Reg)
				ctx.W.EmitMovRegReg(r255, d305.Reg)
				ctx.W.EmitShrRegImm8(r255, 1)
				d306 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
				ctx.BindReg(r255, &d306)
			}
			if d306.Loc == scm.LocImm {
				d306 = scm.JITValueDesc{Loc: scm.LocImm, Type: d306.Type, Imm: scm.NewInt(int64(uint64(d306.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d306.Reg, 32)
				ctx.W.EmitShrRegImm8(d306.Reg, 32)
			}
			if d306.Loc == scm.LocReg && d305.Loc == scm.LocReg && d306.Reg == d305.Reg {
				ctx.TransferReg(d305.Reg)
				d305.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d305)
			d307 := d306
			if d307.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d307)
			d308 := d307
			if d308.Loc == scm.LocImm {
				d308 = scm.JITValueDesc{Loc: scm.LocImm, Type: d308.Type, Imm: scm.NewInt(int64(uint64(d308.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d308.Reg, 32)
				ctx.W.EmitShrRegImm8(d308.Reg, 32)
			}
			ctx.EmitStoreToStack(d308, 8)
			d309 := d294
			if d309.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d309)
			d310 := d309
			if d310.Loc == scm.LocImm {
				d310 = scm.JITValueDesc{Loc: scm.LocImm, Type: d310.Type, Imm: scm.NewInt(int64(uint64(d310.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d310.Reg, 32)
				ctx.W.EmitShrRegImm8(d310.Reg, 32)
			}
			ctx.EmitStoreToStack(d310, 16)
			d311 := d295
			if d311.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d311)
			d312 := d311
			if d312.Loc == scm.LocImm {
				d312 = scm.JITValueDesc{Loc: scm.LocImm, Type: d312.Type, Imm: scm.NewInt(int64(uint64(d312.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d312.Reg, 32)
				ctx.W.EmitShrRegImm8(d312.Reg, 32)
			}
			ctx.EmitStoreToStack(d312, 24)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl88)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d295 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d294 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d85)
			var d313 scm.JITValueDesc
			if d85.Loc == scm.LocImm {
				d313 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(scratch, d85.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d313 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d313)
			}
			if d313.Loc == scm.LocImm {
				d313 = scm.JITValueDesc{Loc: scm.LocImm, Type: d313.Type, Imm: scm.NewInt(int64(uint64(d313.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d313.Reg, 32)
				ctx.W.EmitShrRegImm8(d313.Reg, 32)
			}
			if d313.Loc == scm.LocReg && d85.Loc == scm.LocReg && d313.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			d314 := d86
			if d314.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d314)
			d315 := d314
			if d315.Loc == scm.LocImm {
				d315 = scm.JITValueDesc{Loc: scm.LocImm, Type: d315.Type, Imm: scm.NewInt(int64(uint64(d315.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d315.Reg, 32)
				ctx.W.EmitShrRegImm8(d315.Reg, 32)
			}
			ctx.EmitStoreToStack(d315, 64)
			d316 := d313
			if d316.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d316)
			d317 := d316
			if d317.Loc == scm.LocImm {
				d317 = scm.JITValueDesc{Loc: scm.LocImm, Type: d317.Type, Imm: scm.NewInt(int64(uint64(d317.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d317.Reg, 32)
				ctx.W.EmitShrRegImm8(d317.Reg, 32)
			}
			ctx.EmitStoreToStack(d317, 72)
			ctx.W.EmitJmp(lbl83)
			ctx.W.MarkLabel(lbl87)
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d85 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d86 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d87 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d295 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d12 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d20 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d109 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d294 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl0)
			d318 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d318)
			ctx.BindReg(r2, &d318)
			ctx.EmitMovPairToResult(&d318, &result)
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
