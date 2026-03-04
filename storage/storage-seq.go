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
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			bbpos_0_9 := int32(-1)
			_ = bbpos_0_9
			bbpos_0_10 := int32(-1)
			_ = bbpos_0_10
			bbpos_0_11 := int32(-1)
			_ = bbpos_0_11
			bbpos_0_12 := int32(-1)
			_ = bbpos_0_12
			bbpos_0_13 := int32(-1)
			_ = bbpos_0_13
			bbpos_0_14 := int32(-1)
			_ = bbpos_0_14
			bbpos_0_15 := int32(-1)
			_ = bbpos_0_15
			bbpos_0_16 := int32(-1)
			_ = bbpos_0_16
			bbpos_0_17 := int32(-1)
			_ = bbpos_0_17
			bbpos_0_18 := int32(-1)
			_ = bbpos_0_18
			bbpos_0_19 := int32(-1)
			_ = bbpos_0_19
			bbpos_0_20 := int32(-1)
			_ = bbpos_0_20
			bbpos_0_21 := int32(-1)
			_ = bbpos_0_21
			bbpos_0_22 := int32(-1)
			_ = bbpos_0_22
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d4.Loc == scm.LocImm {
				if d4.Imm.Bool() {
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d3)
			bbpos_0_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl9)
			ctx.W.ResolveFixups()
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
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d6.Loc == scm.LocImm {
				if d6.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.MarkLabel(lbl14)
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
					ctx.W.EmitJmp(lbl1)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl14)
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
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d5)
			bbpos_0_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d11 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d12 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			}
			if d12.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: d12.Type, Imm: scm.NewInt(int64(uint64(d12.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d12.Reg, 32)
				ctx.W.EmitShrRegImm8(d12.Reg, 32)
			}
			if d12.Loc == scm.LocReg && d2.Loc == scm.LocReg && d12.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d13 := d11
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			d14 := d13
			if d14.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: d14.Type, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d14.Reg, 32)
				ctx.W.EmitShrRegImm8(d14.Reg, 32)
			}
			ctx.EmitStoreToStack(d14, 8)
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 16)
			d15 := d12
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			d16 := d15
			if d16.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: d16.Type, Imm: scm.NewInt(int64(uint64(d16.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d16.Reg, 32)
				ctx.W.EmitShrRegImm8(d16.Reg, 32)
			}
			ctx.EmitStoreToStack(d16, 24)
			bbpos_0_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d19 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d17)
			d20 := d17
			_ = d20
			r11 := d17.Loc == scm.LocReg
			r12 := d17.Reg
			if r11 { ctx.ProtectReg(r12) }
			lbl15 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d20.Imm.Int()))))}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d20.Reg)
				ctx.W.EmitShlRegImm8(r13, 32)
				ctx.W.EmitShrRegImm8(r13, 32)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d21)
			}
			var d22 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d22)
			}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d22.Imm.Int()))))}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r15, d22.Reg)
				ctx.W.EmitShlRegImm8(r15, 56)
				ctx.W.EmitShrRegImm8(r15, 56)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d23)
			}
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d21.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() * d23.Imm.Int())}
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else {
				r16 := ctx.AllocRegExcept(d21.Reg, d23.Reg)
				ctx.W.EmitMovRegReg(r16, d21.Reg)
				ctx.W.EmitImulInt64(r16, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d24)
			}
			if d24.Loc == scm.LocReg && d21.Loc == scm.LocReg && d24.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d23)
			var d25 scm.JITValueDesc
			r17 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r17, uint64(dataPtr))
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17, StackOff: int32(sliceLen)}
				ctx.BindReg(r17, &d25)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r17, thisptr.Reg, off)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d25)
			}
			ctx.BindReg(r17, &d25)
			ctx.EnsureDesc(&d24)
			var d26 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r18, d24.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d26)
			}
			if d26.Loc == scm.LocReg && d24.Loc == scm.LocReg && d26.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d26)
			r19 := ctx.AllocReg()
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d25)
			if d26.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d26.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d26.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d25.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d25.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d27 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.BindReg(r20, &d27)
			ctx.FreeDesc(&d26)
			ctx.EnsureDesc(&d24)
			var d28 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r21, d24.Reg)
				ctx.W.EmitAndRegImm32(r21, 63)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d28)
			}
			if d28.Loc == scm.LocReg && d24.Loc == scm.LocReg && d28.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d27.Imm.Int()) << uint64(d28.Imm.Int())))}
			} else if d28.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r22, d27.Reg)
				ctx.W.EmitShlRegImm8(r22, uint8(d28.Imm.Int()))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d29)
			} else {
				{
					shiftSrc := d27.Reg
					r23 := ctx.AllocRegExcept(d27.Reg)
					ctx.W.EmitMovRegReg(r23, d27.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d28.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d28.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d29)
				}
			}
			if d29.Loc == scm.LocReg && d27.Loc == scm.LocReg && d29.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.FreeDesc(&d28)
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d30)
			}
			d31 := d30
			ctx.EnsureDesc(&d31)
			if d31.Loc != scm.LocImm && d31.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d31.Loc == scm.LocImm {
				if d31.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
			d32 := d29
			if d32.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d32)
			ctx.EmitStoreToStack(d32, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d31.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
			d33 := d29
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 80)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d30)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl17)
			ctx.W.ResolveFixups()
			d34 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d35)
			}
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d35.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d35.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d36)
			}
			ctx.FreeDesc(&d35)
			d37 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d36)
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() - d36.Imm.Int())}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r27, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d38)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d36.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(scratch, d37.Reg)
				if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d36.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d36.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else {
				r28 := ctx.AllocRegExcept(d37.Reg, d36.Reg)
				ctx.W.EmitMovRegReg(r28, d37.Reg)
				ctx.W.EmitSubInt64(r28, d36.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d38)
			}
			if d38.Loc == scm.LocReg && d37.Loc == scm.LocReg && d38.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d38)
			var d39 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d34.Imm.Int()) >> uint64(d38.Imm.Int())))}
			} else if d38.Loc == scm.LocImm {
				r29 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r29, d34.Reg)
				ctx.W.EmitShrRegImm8(r29, uint8(d38.Imm.Int()))
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d39)
			} else {
				{
					shiftSrc := d34.Reg
					r30 := ctx.AllocRegExcept(d34.Reg)
					ctx.W.EmitMovRegReg(r30, d34.Reg)
					shiftSrc = r30
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d38.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d38.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d38.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d39)
				}
			}
			if d39.Loc == scm.LocReg && d34.Loc == scm.LocReg && d39.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d38)
			r31 := ctx.AllocReg()
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d39)
			if d39.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r31, d39)
			}
			ctx.W.EmitJmp(lbl15)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl16)
			ctx.W.ResolveFixups()
			d34 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d24)
			var d40 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r32, d24.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d40)
			}
			if d40.Loc == scm.LocReg && d24.Loc == scm.LocReg && d40.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r33, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
				ctx.BindReg(r33, &d41)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d41.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, d41.Reg)
				ctx.W.EmitShlRegImm8(r34, 56)
				ctx.W.EmitShrRegImm8(r34, 56)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d42)
			}
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() + d42.Imm.Int())}
			} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r35, d40.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d43)
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r36 := ctx.AllocRegExcept(d40.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r36, d40.Reg)
				ctx.W.EmitAddInt64(r36, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d43)
			}
			if d43.Loc == scm.LocReg && d40.Loc == scm.LocReg && d43.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d43.Imm.Int()) > uint64(64))}
			} else {
				r37 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitCmpRegImm32(d43.Reg, 64)
				ctx.W.EmitSetcc(r37, scm.CcA)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r37}
				ctx.BindReg(r37, &d44)
			}
			ctx.FreeDesc(&d43)
			d45 := d44
			ctx.EnsureDesc(&d45)
			if d45.Loc != scm.LocImm && d45.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.MarkLabel(lbl21)
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.MarkLabel(lbl22)
			d46 := d29
			if d46.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			ctx.EmitStoreToStack(d46, 80)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl21)
				ctx.W.EmitJmp(lbl22)
				ctx.W.MarkLabel(lbl21)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl22)
			d47 := d29
			if d47.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d47)
			ctx.EmitStoreToStack(d47, 80)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d44)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl20)
			ctx.W.ResolveFixups()
			d34 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(80)}
			ctx.EnsureDesc(&d24)
			var d48 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() / 64)}
			} else {
				r38 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r38, d24.Reg)
				ctx.W.EmitShrRegImm8(r38, 6)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d48)
			}
			if d48.Loc == scm.LocReg && d24.Loc == scm.LocReg && d48.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			}
			if d49.Loc == scm.LocReg && d48.Loc == scm.LocReg && d49.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d49)
			r39 := ctx.AllocReg()
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d25)
			if d49.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d49.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d49.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			if d25.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitAddInt64(r39, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r39, d25.Reg)
			}
			r40 := ctx.AllocRegExcept(r39)
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.FreeReg(r39)
			d50 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.BindReg(r40, &d50)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d24)
			var d51 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() % 64)}
			} else {
				r41 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r41, d24.Reg)
				ctx.W.EmitAndRegImm32(r41, 63)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d51)
			}
			if d51.Loc == scm.LocReg && d24.Loc == scm.LocReg && d51.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			d52 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d51)
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() - d51.Imm.Int())}
			} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r42, d52.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d53)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d51.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(scratch, d52.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d53)
			} else {
				r43 := ctx.AllocRegExcept(d52.Reg, d51.Reg)
				ctx.W.EmitMovRegReg(r43, d52.Reg)
				ctx.W.EmitSubInt64(r43, d51.Reg)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d53)
			}
			if d53.Loc == scm.LocReg && d52.Loc == scm.LocReg && d53.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d53)
			var d54 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) >> uint64(d53.Imm.Int())))}
			} else if d53.Loc == scm.LocImm {
				r44 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r44, d50.Reg)
				ctx.W.EmitShrRegImm8(r44, uint8(d53.Imm.Int()))
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d54)
			} else {
				{
					shiftSrc := d50.Reg
					r45 := ctx.AllocRegExcept(d50.Reg)
					ctx.W.EmitMovRegReg(r45, d50.Reg)
					shiftSrc = r45
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d53.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d53.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d53.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d54)
				}
			}
			if d54.Loc == scm.LocReg && d50.Loc == scm.LocReg && d54.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() | d54.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
				ctx.BindReg(d54.Reg, &d55)
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r46, d29.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d55)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else if d54.Loc == scm.LocImm {
				r47 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r47, d29.Reg)
				if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r47, int32(d54.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
					ctx.W.EmitOrInt64(r47, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d55)
			} else {
				r48 := ctx.AllocRegExcept(d29.Reg, d54.Reg)
				ctx.W.EmitMovRegReg(r48, d29.Reg)
				ctx.W.EmitOrInt64(r48, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d55)
			}
			if d55.Loc == scm.LocReg && d29.Loc == scm.LocReg && d55.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d54)
			d56 := d55
			if d56.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, 80)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl15)
			d57 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d57)
			ctx.BindReg(r31, &d57)
			if r11 { ctx.UnprotectReg(r12) }
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d57)
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d57.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d57.Reg)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d58)
			}
			ctx.FreeDesc(&d57)
			var d59 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d59)
			}
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() + d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r51 := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(r51, d58.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d60)
			} else if d58.Loc == scm.LocImm && d58.Imm.Int() == 0 {
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
				ctx.BindReg(d59.Reg, &d60)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(scratch, d58.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else {
				r52 := ctx.AllocRegExcept(d58.Reg, d59.Reg)
				ctx.W.EmitMovRegReg(r52, d58.Reg)
				ctx.W.EmitAddInt64(r52, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d60)
			}
			if d60.Loc == scm.LocReg && d58.Loc == scm.LocReg && d60.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d60.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d60.Reg)
				ctx.W.EmitShlRegImm8(r53, 32)
				ctx.W.EmitShrRegImm8(r53, 32)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d61)
			}
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d61.Imm.Int()))}
			} else if d61.Loc == scm.LocImm {
				r54 := ctx.AllocRegExcept(idxInt.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r54, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d62)
			} else if idxInt.Loc == scm.LocImm {
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d61.Reg)
				ctx.W.EmitSetcc(r55, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d62)
			} else {
				r56 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d61.Reg)
				ctx.W.EmitSetcc(r56, scm.CcB)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d62)
			}
			ctx.FreeDesc(&d61)
			d63 := d62
			ctx.EnsureDesc(&d63)
			if d63.Loc != scm.LocImm && d63.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d63.Loc == scm.LocImm {
				if d63.Imm.Bool() {
					ctx.W.MarkLabel(lbl25)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.MarkLabel(lbl26)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d63.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d62)
			bbpos_0_9 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl24)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			var d64 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
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
			if d64.Loc == scm.LocReg && d17.Loc == scm.LocReg && d64.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
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
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d66.Loc == scm.LocImm {
				if d66.Imm.Bool() {
					ctx.W.MarkLabel(lbl28)
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.MarkLabel(lbl29)
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
			d69 := d17
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
			d71 := d19
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
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d66.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl29)
			d73 := d64
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			d74 := d73
			if d74.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: d74.Type, Imm: scm.NewInt(int64(uint64(d74.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d74.Reg, 32)
				ctx.W.EmitShrRegImm8(d74.Reg, 32)
			}
			ctx.EmitStoreToStack(d74, 40)
			d75 := d17
			if d75.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d75)
			d76 := d75
			if d76.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: d76.Type, Imm: scm.NewInt(int64(uint64(d76.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d76.Reg, 32)
				ctx.W.EmitShrRegImm8(d76.Reg, 32)
			}
			ctx.EmitStoreToStack(d76, 48)
			d77 := d19
			if d77.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d77)
			d78 := d77
			if d78.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: d78.Type, Imm: scm.NewInt(int64(uint64(d78.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d78.Reg, 32)
				ctx.W.EmitShrRegImm8(d78.Reg, 32)
			}
			ctx.EmitStoreToStack(d78, 56)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d65)
			bbpos_0_8 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d80 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d81 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			var d82 scm.JITValueDesc
			if d80.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d80.Imm.Int()) == uint64(d81.Imm.Int()))}
			} else if d81.Loc == scm.LocImm {
				r60 := ctx.AllocRegExcept(d80.Reg)
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d80.Reg, int32(d81.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
					ctx.W.EmitCmpInt64(d80.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r60, scm.CcE)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d82)
			} else if d80.Loc == scm.LocImm {
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d81.Reg)
				ctx.W.EmitSetcc(r61, scm.CcE)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d82)
			} else {
				r62 := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitCmpInt64(d80.Reg, d81.Reg)
				ctx.W.EmitSetcc(r62, scm.CcE)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
				ctx.BindReg(r62, &d82)
			}
			d83 := d82
			ctx.EnsureDesc(&d83)
			if d83.Loc != scm.LocImm && d83.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d83.Loc == scm.LocImm {
				if d83.Imm.Bool() {
					ctx.W.MarkLabel(lbl31)
			d84 := d80
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
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl32)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d83.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl31)
			d86 := d80
			if d86.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d86)
			d87 := d86
			if d87.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: d87.Type, Imm: scm.NewInt(int64(uint64(d87.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d87.Reg, 32)
				ctx.W.EmitShrRegImm8(d87.Reg, 32)
			}
			ctx.EmitStoreToStack(d87, 32)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d82)
			bbpos_0_6 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl3)
			ctx.W.ResolveFixups()
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(d88.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d88.Reg)
				ctx.W.EmitShlRegImm8(r63, 32)
				ctx.W.EmitShrRegImm8(r63, 32)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d89)
			}
			ctx.EnsureDesc(&d89)
			if thisptr.Loc == scm.LocImm {
				baseReg := ctx.AllocReg()
				if d89.Loc == scm.LocReg {
					ctx.FreeReg(baseReg)
					baseReg = ctx.AllocRegExcept(d89.Reg)
				}
				ctx.W.EmitMovRegImm64(baseReg, uint64(uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).lastValue)))
				if d89.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, baseReg, 0)
				} else {
					ctx.W.EmitStoreRegMem(d89.Reg, baseReg, 0)
				}
				ctx.FreeReg(baseReg)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).lastValue))
				if d89.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
					ctx.W.EmitStoreRegMem(scm.RegR11, thisptr.Reg, off)
				} else {
					ctx.W.EmitStoreRegMem(d89.Reg, thisptr.Reg, off)
				}
			}
			ctx.FreeDesc(&d89)
			ctx.EnsureDesc(&d88)
			d90 := d88
			_ = d90
			r64 := d88.Loc == scm.LocReg
			r65 := d88.Reg
			if r64 { ctx.ProtectReg(r65) }
			lbl33 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d90.Imm.Int()))))}
			} else {
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r66, d90.Reg)
				ctx.W.EmitShlRegImm8(r66, 32)
				ctx.W.EmitShrRegImm8(r66, 32)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d91)
			}
			var d92 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r67, thisptr.Reg, off)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r67}
				ctx.BindReg(r67, &d92)
			}
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d92)
			var d93 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d92.Imm.Int()))))}
			} else {
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r68, d92.Reg)
				ctx.W.EmitShlRegImm8(r68, 56)
				ctx.W.EmitShrRegImm8(r68, 56)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d93)
			}
			ctx.FreeDesc(&d92)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d93)
			var d94 scm.JITValueDesc
			if d91.Loc == scm.LocImm && d93.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() * d93.Imm.Int())}
			} else if d91.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d91.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d93.Reg)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d94)
			} else if d93.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(scratch, d91.Reg)
				if d93.Imm.Int() >= -2147483648 && d93.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d93.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d93.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d94)
			} else {
				r69 := ctx.AllocRegExcept(d91.Reg, d93.Reg)
				ctx.W.EmitMovRegReg(r69, d91.Reg)
				ctx.W.EmitImulInt64(r69, d93.Reg)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d94)
			}
			if d94.Loc == scm.LocReg && d91.Loc == scm.LocReg && d94.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			ctx.FreeDesc(&d93)
			var d95 scm.JITValueDesc
			r70 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r70, uint64(dataPtr))
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70, StackOff: int32(sliceLen)}
				ctx.BindReg(r70, &d95)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 0)
				ctx.W.EmitMovRegMem(r70, thisptr.Reg, off)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d95)
			}
			ctx.BindReg(r70, &d95)
			ctx.EnsureDesc(&d94)
			var d96 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() / 64)}
			} else {
				r71 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r71, d94.Reg)
				ctx.W.EmitShrRegImm8(r71, 6)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d96)
			}
			if d96.Loc == scm.LocReg && d94.Loc == scm.LocReg && d96.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d96)
			r72 := ctx.AllocReg()
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d95)
			if d96.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r72, uint64(d96.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r72, d96.Reg)
				ctx.W.EmitShlRegImm8(r72, 3)
			}
			if d95.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
				ctx.W.EmitAddInt64(r72, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r72, d95.Reg)
			}
			r73 := ctx.AllocRegExcept(r72)
			ctx.W.EmitMovRegMem(r73, r72, 0)
			ctx.FreeReg(r72)
			d97 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r73}
			ctx.BindReg(r73, &d97)
			ctx.FreeDesc(&d96)
			ctx.EnsureDesc(&d94)
			var d98 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r74 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r74, d94.Reg)
				ctx.W.EmitAndRegImm32(r74, 63)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d98)
			}
			if d98.Loc == scm.LocReg && d94.Loc == scm.LocReg && d98.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d98)
			var d99 scm.JITValueDesc
			if d97.Loc == scm.LocImm && d98.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d97.Imm.Int()) << uint64(d98.Imm.Int())))}
			} else if d98.Loc == scm.LocImm {
				r75 := ctx.AllocRegExcept(d97.Reg)
				ctx.W.EmitMovRegReg(r75, d97.Reg)
				ctx.W.EmitShlRegImm8(r75, uint8(d98.Imm.Int()))
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d99)
			} else {
				{
					shiftSrc := d97.Reg
					r76 := ctx.AllocRegExcept(d97.Reg)
					ctx.W.EmitMovRegReg(r76, d97.Reg)
					shiftSrc = r76
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d98.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d98.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d98.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d99)
				}
			}
			if d99.Loc == scm.LocReg && d97.Loc == scm.LocReg && d99.Reg == d97.Reg {
				ctx.TransferReg(d97.Reg)
				d97.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d97)
			ctx.FreeDesc(&d98)
			var d100 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 25)
				r77 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r77, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
				ctx.BindReg(r77, &d100)
			}
			d101 := d100
			ctx.EnsureDesc(&d101)
			if d101.Loc != scm.LocImm && d101.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			if d101.Loc == scm.LocImm {
				if d101.Imm.Bool() {
					ctx.W.MarkLabel(lbl36)
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.MarkLabel(lbl37)
			d102 := d99
			if d102.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d102)
			ctx.EmitStoreToStack(d102, 88)
					ctx.W.EmitJmp(lbl35)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d101.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl34)
				ctx.W.MarkLabel(lbl37)
			d103 := d99
			if d103.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d103)
			ctx.EmitStoreToStack(d103, 88)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d100)
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl35)
			ctx.W.ResolveFixups()
			d104 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r78, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
				ctx.BindReg(r78, &d105)
			}
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d105)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r79 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r79, d105.Reg)
				ctx.W.EmitShlRegImm8(r79, 56)
				ctx.W.EmitShrRegImm8(r79, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d106)
			}
			ctx.FreeDesc(&d105)
			d107 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d106)
			var d108 scm.JITValueDesc
			if d107.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() - d106.Imm.Int())}
			} else if d106.Loc == scm.LocImm && d106.Imm.Int() == 0 {
				r80 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r80, d107.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d108)
			} else if d107.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d107.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d106.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else if d106.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(scratch, d107.Reg)
				if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d106.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d106.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else {
				r81 := ctx.AllocRegExcept(d107.Reg, d106.Reg)
				ctx.W.EmitMovRegReg(r81, d107.Reg)
				ctx.W.EmitSubInt64(r81, d106.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d108)
			}
			if d108.Loc == scm.LocReg && d107.Loc == scm.LocReg && d108.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d108)
			var d109 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d104.Imm.Int()) >> uint64(d108.Imm.Int())))}
			} else if d108.Loc == scm.LocImm {
				r82 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r82, d104.Reg)
				ctx.W.EmitShrRegImm8(r82, uint8(d108.Imm.Int()))
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d109)
			} else {
				{
					shiftSrc := d104.Reg
					r83 := ctx.AllocRegExcept(d104.Reg)
					ctx.W.EmitMovRegReg(r83, d104.Reg)
					shiftSrc = r83
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d108.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d108.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d108.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d109)
				}
			}
			if d109.Loc == scm.LocReg && d104.Loc == scm.LocReg && d109.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.FreeDesc(&d108)
			r84 := ctx.AllocReg()
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			if d109.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r84, d109)
			}
			ctx.W.EmitJmp(lbl33)
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl34)
			ctx.W.ResolveFixups()
			d104 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d94)
			var d110 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r85 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r85, d94.Reg)
				ctx.W.EmitAndRegImm32(r85, 63)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d110)
			}
			if d110.Loc == scm.LocReg && d94.Loc == scm.LocReg && d110.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			var d111 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 24)
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r86, thisptr.Reg, off)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
				ctx.BindReg(r86, &d111)
			}
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d111.Imm.Int()))))}
			} else {
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r87, d111.Reg)
				ctx.W.EmitShlRegImm8(r87, 56)
				ctx.W.EmitShrRegImm8(r87, 56)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d112)
			}
			ctx.FreeDesc(&d111)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d112)
			var d113 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() + d112.Imm.Int())}
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r88, d110.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d113)
			} else if d110.Loc == scm.LocImm && d110.Imm.Int() == 0 {
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d112.Reg}
				ctx.BindReg(d112.Reg, &d113)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d110.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(scratch, d110.Reg)
				if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d112.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else {
				r89 := ctx.AllocRegExcept(d110.Reg, d112.Reg)
				ctx.W.EmitMovRegReg(r89, d110.Reg)
				ctx.W.EmitAddInt64(r89, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d113)
			}
			if d113.Loc == scm.LocReg && d110.Loc == scm.LocReg && d113.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d110)
			ctx.FreeDesc(&d112)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d113.Imm.Int()) > uint64(64))}
			} else {
				r90 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitCmpRegImm32(d113.Reg, 64)
				ctx.W.EmitSetcc(r90, scm.CcA)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r90}
				ctx.BindReg(r90, &d114)
			}
			ctx.FreeDesc(&d113)
			d115 := d114
			ctx.EnsureDesc(&d115)
			if d115.Loc != scm.LocImm && d115.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d115.Loc == scm.LocImm {
				if d115.Imm.Bool() {
					ctx.W.MarkLabel(lbl39)
					ctx.W.EmitJmp(lbl38)
				} else {
					ctx.W.MarkLabel(lbl40)
			d116 := d99
			if d116.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d116)
			ctx.EmitStoreToStack(d116, 88)
					ctx.W.EmitJmp(lbl35)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d115.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.W.EmitJmp(lbl40)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl40)
			d117 := d99
			if d117.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d117)
			ctx.EmitStoreToStack(d117, 88)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d114)
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl38)
			ctx.W.ResolveFixups()
			d104 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(88)}
			ctx.EnsureDesc(&d94)
			var d118 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() / 64)}
			} else {
				r91 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r91, d94.Reg)
				ctx.W.EmitShrRegImm8(r91, 6)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d118)
			}
			if d118.Loc == scm.LocReg && d94.Loc == scm.LocReg && d118.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d118)
			var d119 scm.JITValueDesc
			if d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(scratch, d118.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d119)
			}
			if d119.Loc == scm.LocReg && d118.Loc == scm.LocReg && d119.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.EnsureDesc(&d119)
			r92 := ctx.AllocReg()
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d95)
			if d119.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r92, uint64(d119.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r92, d119.Reg)
				ctx.W.EmitShlRegImm8(r92, 3)
			}
			if d95.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
				ctx.W.EmitAddInt64(r92, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r92, d95.Reg)
			}
			r93 := ctx.AllocRegExcept(r92)
			ctx.W.EmitMovRegMem(r93, r92, 0)
			ctx.FreeReg(r92)
			d120 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
			ctx.BindReg(r93, &d120)
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d94)
			var d121 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r94 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r94, d94.Reg)
				ctx.W.EmitAndRegImm32(r94, 63)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d121)
			}
			if d121.Loc == scm.LocReg && d94.Loc == scm.LocReg && d121.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			d122 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d121)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() - d121.Imm.Int())}
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				r95 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r95, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d123)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d121.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(scratch, d122.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else {
				r96 := ctx.AllocRegExcept(d122.Reg, d121.Reg)
				ctx.W.EmitMovRegReg(r96, d122.Reg)
				ctx.W.EmitSubInt64(r96, d121.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d123)
			}
			if d123.Loc == scm.LocReg && d122.Loc == scm.LocReg && d123.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d121)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d123)
			var d124 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d120.Imm.Int()) >> uint64(d123.Imm.Int())))}
			} else if d123.Loc == scm.LocImm {
				r97 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r97, d120.Reg)
				ctx.W.EmitShrRegImm8(r97, uint8(d123.Imm.Int()))
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d124)
			} else {
				{
					shiftSrc := d120.Reg
					r98 := ctx.AllocRegExcept(d120.Reg)
					ctx.W.EmitMovRegReg(r98, d120.Reg)
					shiftSrc = r98
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d123.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d123.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d123.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d124)
				}
			}
			if d124.Loc == scm.LocReg && d120.Loc == scm.LocReg && d124.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			ctx.FreeDesc(&d123)
			ctx.EnsureDesc(&d99)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() | d124.Imm.Int())}
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
				ctx.BindReg(d124.Reg, &d125)
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				r99 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r99, d99.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d125)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else if d124.Loc == scm.LocImm {
				r100 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r100, d99.Reg)
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r100, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitOrInt64(r100, scm.RegR11)
				}
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d125)
			} else {
				r101 := ctx.AllocRegExcept(d99.Reg, d124.Reg)
				ctx.W.EmitMovRegReg(r101, d99.Reg)
				ctx.W.EmitOrInt64(r101, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d125)
			}
			if d125.Loc == scm.LocReg && d99.Loc == scm.LocReg && d125.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			d126 := d125
			if d126.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d126)
			ctx.EmitStoreToStack(d126, 88)
			ctx.W.EmitJmp(lbl35)
			ctx.W.MarkLabel(lbl33)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r84}
			ctx.BindReg(r84, &d127)
			ctx.BindReg(r84, &d127)
			if r64 { ctx.UnprotectReg(r65) }
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d127.Imm.Int()))))}
			} else {
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r102, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d128)
			}
			ctx.FreeDesc(&d127)
			var d129 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 32)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r103, thisptr.Reg, off)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
				ctx.BindReg(r103, &d129)
			}
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d129)
			var d130 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() + d129.Imm.Int())}
			} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
				r104 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r104, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d130)
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
				ctx.BindReg(d129.Reg, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(scratch, d128.Reg)
				if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d129.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r105 := ctx.AllocRegExcept(d128.Reg, d129.Reg)
				ctx.W.EmitMovRegReg(r105, d128.Reg)
				ctx.W.EmitAddInt64(r105, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d130)
			}
			if d130.Loc == scm.LocReg && d128.Loc == scm.LocReg && d130.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.FreeDesc(&d129)
			var d131 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 56)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d131)
			}
			d132 := d131
			ctx.EnsureDesc(&d132)
			if d132.Loc != scm.LocImm && d132.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d132.Loc == scm.LocImm {
				if d132.Imm.Bool() {
					ctx.W.MarkLabel(lbl42)
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.MarkLabel(lbl43)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d132.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d131)
			bbpos_0_21 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl7)
			ctx.W.ResolveFixups()
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d88)
			d133 := d88
			_ = d133
			r107 := d88.Loc == scm.LocReg
			r108 := d88.Reg
			if r107 { ctx.ProtectReg(r108) }
			lbl44 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d133)
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d133.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r109, d133.Reg)
				ctx.W.EmitShlRegImm8(r109, 32)
				ctx.W.EmitShrRegImm8(r109, 32)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d134)
			}
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r110, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
				ctx.BindReg(r110, &d135)
			}
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d135)
			var d136 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d135.Imm.Int()))))}
			} else {
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r111, d135.Reg)
				ctx.W.EmitShlRegImm8(r111, 56)
				ctx.W.EmitShrRegImm8(r111, 56)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d136)
			}
			ctx.FreeDesc(&d135)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d136)
			var d137 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() * d136.Imm.Int())}
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(scratch, d134.Reg)
				if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d136.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d137)
			} else {
				r112 := ctx.AllocRegExcept(d134.Reg, d136.Reg)
				ctx.W.EmitMovRegReg(r112, d134.Reg)
				ctx.W.EmitImulInt64(r112, d136.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d137)
			}
			if d137.Loc == scm.LocReg && d134.Loc == scm.LocReg && d137.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d134)
			ctx.FreeDesc(&d136)
			var d138 scm.JITValueDesc
			r113 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r113, uint64(dataPtr))
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r113, StackOff: int32(sliceLen)}
				ctx.BindReg(r113, &d138)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 0)
				ctx.W.EmitMovRegMem(r113, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r113}
				ctx.BindReg(r113, &d138)
			}
			ctx.BindReg(r113, &d138)
			ctx.EnsureDesc(&d137)
			var d139 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() / 64)}
			} else {
				r114 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r114, d137.Reg)
				ctx.W.EmitShrRegImm8(r114, 6)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
				ctx.BindReg(r114, &d139)
			}
			if d139.Loc == scm.LocReg && d137.Loc == scm.LocReg && d139.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d139)
			r115 := ctx.AllocReg()
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d138)
			if d139.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r115, uint64(d139.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r115, d139.Reg)
				ctx.W.EmitShlRegImm8(r115, 3)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
				ctx.W.EmitAddInt64(r115, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r115, d138.Reg)
			}
			r116 := ctx.AllocRegExcept(r115)
			ctx.W.EmitMovRegMem(r116, r115, 0)
			ctx.FreeReg(r115)
			d140 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
			ctx.BindReg(r116, &d140)
			ctx.FreeDesc(&d139)
			ctx.EnsureDesc(&d137)
			var d141 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				r117 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r117, d137.Reg)
				ctx.W.EmitAndRegImm32(r117, 63)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d141)
			}
			if d141.Loc == scm.LocReg && d137.Loc == scm.LocReg && d141.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d141)
			var d142 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d140.Imm.Int()) << uint64(d141.Imm.Int())))}
			} else if d141.Loc == scm.LocImm {
				r118 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r118, d140.Reg)
				ctx.W.EmitShlRegImm8(r118, uint8(d141.Imm.Int()))
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d142)
			} else {
				{
					shiftSrc := d140.Reg
					r119 := ctx.AllocRegExcept(d140.Reg)
					ctx.W.EmitMovRegReg(r119, d140.Reg)
					shiftSrc = r119
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
					ctx.BindReg(shiftSrc, &d142)
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
				r120 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r120, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r120}
				ctx.BindReg(r120, &d143)
			}
			d144 := d143
			ctx.EnsureDesc(&d144)
			if d144.Loc != scm.LocImm && d144.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			if d144.Loc == scm.LocImm {
				if d144.Imm.Bool() {
					ctx.W.MarkLabel(lbl47)
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.MarkLabel(lbl48)
			d145 := d142
			if d145.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d145)
			ctx.EmitStoreToStack(d145, 96)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d144.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
				ctx.W.MarkLabel(lbl48)
			d146 := d142
			if d146.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d146)
			ctx.EmitStoreToStack(d146, 96)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d143)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl46)
			ctx.W.ResolveFixups()
			d147 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			var d148 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
				ctx.BindReg(r121, &d148)
			}
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d148.Imm.Int()))))}
			} else {
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r122, d148.Reg)
				ctx.W.EmitShlRegImm8(r122, 56)
				ctx.W.EmitShrRegImm8(r122, 56)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d149)
			}
			ctx.FreeDesc(&d148)
			d150 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d149)
			var d151 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() - d149.Imm.Int())}
			} else if d149.Loc == scm.LocImm && d149.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r123, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d151)
			} else if d150.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d150.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(scratch, d150.Reg)
				if d149.Imm.Int() >= -2147483648 && d149.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d149.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d149.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else {
				r124 := ctx.AllocRegExcept(d150.Reg, d149.Reg)
				ctx.W.EmitMovRegReg(r124, d150.Reg)
				ctx.W.EmitSubInt64(r124, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d151)
			}
			if d151.Loc == scm.LocReg && d150.Loc == scm.LocReg && d151.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d147.Imm.Int()) >> uint64(d151.Imm.Int())))}
			} else if d151.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r125, d147.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d151.Imm.Int()))
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d152)
			} else {
				{
					shiftSrc := d147.Reg
					r126 := ctx.AllocRegExcept(d147.Reg)
					ctx.W.EmitMovRegReg(r126, d147.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d151.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d151.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d151.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d152)
				}
			}
			if d152.Loc == scm.LocReg && d147.Loc == scm.LocReg && d152.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.FreeDesc(&d151)
			r127 := ctx.AllocReg()
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d152)
			if d152.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r127, d152)
			}
			ctx.W.EmitJmp(lbl44)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl45)
			ctx.W.ResolveFixups()
			d147 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d137)
			var d153 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				r128 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r128, d137.Reg)
				ctx.W.EmitAndRegImm32(r128, 63)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d153)
			}
			if d153.Loc == scm.LocReg && d137.Loc == scm.LocReg && d153.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			var d154 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 24)
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r129, thisptr.Reg, off)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
				ctx.BindReg(r129, &d154)
			}
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d154.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d154.Reg)
				ctx.W.EmitShlRegImm8(r130, 56)
				ctx.W.EmitShrRegImm8(r130, 56)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d155)
			}
			ctx.FreeDesc(&d154)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d155)
			var d156 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() + d155.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r131 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r131, d153.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d156)
			} else if d153.Loc == scm.LocImm && d153.Imm.Int() == 0 {
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
				ctx.BindReg(d155.Reg, &d156)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(scratch, d153.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else {
				r132 := ctx.AllocRegExcept(d153.Reg, d155.Reg)
				ctx.W.EmitMovRegReg(r132, d153.Reg)
				ctx.W.EmitAddInt64(r132, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d156)
			}
			if d156.Loc == scm.LocReg && d153.Loc == scm.LocReg && d156.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d153)
			ctx.FreeDesc(&d155)
			ctx.EnsureDesc(&d156)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d156.Imm.Int()) > uint64(64))}
			} else {
				r133 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitCmpRegImm32(d156.Reg, 64)
				ctx.W.EmitSetcc(r133, scm.CcA)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r133}
				ctx.BindReg(r133, &d157)
			}
			ctx.FreeDesc(&d156)
			d158 := d157
			ctx.EnsureDesc(&d158)
			if d158.Loc != scm.LocImm && d158.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			if d158.Loc == scm.LocImm {
				if d158.Imm.Bool() {
					ctx.W.MarkLabel(lbl50)
					ctx.W.EmitJmp(lbl49)
				} else {
					ctx.W.MarkLabel(lbl51)
			d159 := d142
			if d159.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d159)
			ctx.EmitStoreToStack(d159, 96)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d158.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl51)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl51)
			d160 := d142
			if d160.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d160)
			ctx.EmitStoreToStack(d160, 96)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d157)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl49)
			ctx.W.ResolveFixups()
			d147 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(96)}
			ctx.EnsureDesc(&d137)
			var d161 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() / 64)}
			} else {
				r134 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r134, d137.Reg)
				ctx.W.EmitShrRegImm8(r134, 6)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d161)
			}
			if d161.Loc == scm.LocReg && d137.Loc == scm.LocReg && d161.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d161)
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
			ctx.EnsureDesc(&d162)
			r135 := ctx.AllocReg()
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d138)
			if d162.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r135, uint64(d162.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r135, d162.Reg)
				ctx.W.EmitShlRegImm8(r135, 3)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
				ctx.W.EmitAddInt64(r135, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r135, d138.Reg)
			}
			r136 := ctx.AllocRegExcept(r135)
			ctx.W.EmitMovRegMem(r136, r135, 0)
			ctx.FreeReg(r135)
			d163 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r136}
			ctx.BindReg(r136, &d163)
			ctx.FreeDesc(&d162)
			ctx.EnsureDesc(&d137)
			var d164 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() % 64)}
			} else {
				r137 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r137, d137.Reg)
				ctx.W.EmitAndRegImm32(r137, 63)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d164)
			}
			if d164.Loc == scm.LocReg && d137.Loc == scm.LocReg && d164.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			d165 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d164)
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() - d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r138 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(r138, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d166)
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
				r139 := ctx.AllocRegExcept(d165.Reg, d164.Reg)
				ctx.W.EmitMovRegReg(r139, d165.Reg)
				ctx.W.EmitSubInt64(r139, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d166)
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d166)
			var d167 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d163.Imm.Int()) >> uint64(d166.Imm.Int())))}
			} else if d166.Loc == scm.LocImm {
				r140 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r140, d163.Reg)
				ctx.W.EmitShrRegImm8(r140, uint8(d166.Imm.Int()))
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d167)
			} else {
				{
					shiftSrc := d163.Reg
					r141 := ctx.AllocRegExcept(d163.Reg)
					ctx.W.EmitMovRegReg(r141, d163.Reg)
					shiftSrc = r141
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
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d167)
			var d168 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() | d167.Imm.Int())}
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
				ctx.BindReg(d167.Reg, &d168)
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r142 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r142, d142.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d168)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			} else if d167.Loc == scm.LocImm {
				r143 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r143, d142.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r143, int32(d167.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.W.EmitOrInt64(r143, scm.RegR11)
				}
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d168)
			} else {
				r144 := ctx.AllocRegExcept(d142.Reg, d167.Reg)
				ctx.W.EmitMovRegReg(r144, d142.Reg)
				ctx.W.EmitOrInt64(r144, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d168)
			}
			if d168.Loc == scm.LocReg && d142.Loc == scm.LocReg && d168.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			d169 := d168
			if d169.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d169)
			ctx.EmitStoreToStack(d169, 96)
			ctx.W.EmitJmp(lbl46)
			ctx.W.MarkLabel(lbl44)
			d170 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r127}
			ctx.BindReg(r127, &d170)
			ctx.BindReg(r127, &d170)
			if r107 { ctx.UnprotectReg(r108) }
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d170.Imm.Int()))))}
			} else {
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r145, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d171)
			}
			ctx.FreeDesc(&d170)
			var d172 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).stride) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).stride) + 32)
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r146, thisptr.Reg, off)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
				ctx.BindReg(r146, &d172)
			}
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() + d172.Imm.Int())}
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				r147 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r147, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d173)
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
				ctx.BindReg(d172.Reg, &d173)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(scratch, d171.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else {
				r148 := ctx.AllocRegExcept(d171.Reg, d172.Reg)
				ctx.W.EmitMovRegReg(r148, d171.Reg)
				ctx.W.EmitAddInt64(r148, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d173)
			}
			if d173.Loc == scm.LocReg && d171.Loc == scm.LocReg && d173.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			ctx.FreeDesc(&d172)
			ctx.EnsureDesc(&d88)
			d174 := d88
			_ = d174
			r149 := d88.Loc == scm.LocReg
			r150 := d88.Reg
			if r149 { ctx.ProtectReg(r150) }
			lbl52 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d174)
			var d175 scm.JITValueDesc
			if d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d174.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d174.Reg)
				ctx.W.EmitShlRegImm8(r151, 32)
				ctx.W.EmitShrRegImm8(r151, 32)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d175)
			}
			var d176 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r152, thisptr.Reg, off)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
				ctx.BindReg(r152, &d176)
			}
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d176.Imm.Int()))))}
			} else {
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r153, d176.Reg)
				ctx.W.EmitShlRegImm8(r153, 56)
				ctx.W.EmitShrRegImm8(r153, 56)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d177)
			}
			ctx.FreeDesc(&d176)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d177)
			var d178 scm.JITValueDesc
			if d175.Loc == scm.LocImm && d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() * d177.Imm.Int())}
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d175.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(scratch, d175.Reg)
				if d177.Imm.Int() >= -2147483648 && d177.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d177.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else {
				r154 := ctx.AllocRegExcept(d175.Reg, d177.Reg)
				ctx.W.EmitMovRegReg(r154, d175.Reg)
				ctx.W.EmitImulInt64(r154, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d178)
			}
			if d178.Loc == scm.LocReg && d175.Loc == scm.LocReg && d178.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			ctx.FreeDesc(&d177)
			var d179 scm.JITValueDesc
			r155 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r155, uint64(dataPtr))
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155, StackOff: int32(sliceLen)}
				ctx.BindReg(r155, &d179)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r155, thisptr.Reg, off)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
				ctx.BindReg(r155, &d179)
			}
			ctx.BindReg(r155, &d179)
			ctx.EnsureDesc(&d178)
			var d180 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() / 64)}
			} else {
				r156 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r156, d178.Reg)
				ctx.W.EmitShrRegImm8(r156, 6)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d180)
			}
			if d180.Loc == scm.LocReg && d178.Loc == scm.LocReg && d180.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d180)
			r157 := ctx.AllocReg()
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d179)
			if d180.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r157, uint64(d180.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r157, d180.Reg)
				ctx.W.EmitShlRegImm8(r157, 3)
			}
			if d179.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
				ctx.W.EmitAddInt64(r157, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r157, d179.Reg)
			}
			r158 := ctx.AllocRegExcept(r157)
			ctx.W.EmitMovRegMem(r158, r157, 0)
			ctx.FreeReg(r157)
			d181 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
			ctx.BindReg(r158, &d181)
			ctx.FreeDesc(&d180)
			ctx.EnsureDesc(&d178)
			var d182 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() % 64)}
			} else {
				r159 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r159, d178.Reg)
				ctx.W.EmitAndRegImm32(r159, 63)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d182)
			}
			if d182.Loc == scm.LocReg && d178.Loc == scm.LocReg && d182.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d181.Imm.Int()) << uint64(d182.Imm.Int())))}
			} else if d182.Loc == scm.LocImm {
				r160 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r160, d181.Reg)
				ctx.W.EmitShlRegImm8(r160, uint8(d182.Imm.Int()))
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d183)
			} else {
				{
					shiftSrc := d181.Reg
					r161 := ctx.AllocRegExcept(d181.Reg)
					ctx.W.EmitMovRegReg(r161, d181.Reg)
					shiftSrc = r161
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d182.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d182.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d182.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d183)
				}
			}
			if d183.Loc == scm.LocReg && d181.Loc == scm.LocReg && d183.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d182)
			var d184 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r162 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r162, thisptr.Reg, off)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r162}
				ctx.BindReg(r162, &d184)
			}
			d185 := d184
			ctx.EnsureDesc(&d185)
			if d185.Loc != scm.LocImm && d185.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d185.Loc == scm.LocImm {
				if d185.Imm.Bool() {
					ctx.W.MarkLabel(lbl55)
					ctx.W.EmitJmp(lbl53)
				} else {
					ctx.W.MarkLabel(lbl56)
			d186 := d183
			if d186.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d186)
			ctx.EmitStoreToStack(d186, 104)
					ctx.W.EmitJmp(lbl54)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d185.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl55)
				ctx.W.EmitJmp(lbl56)
				ctx.W.MarkLabel(lbl55)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl56)
			d187 := d183
			if d187.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d187)
			ctx.EmitStoreToStack(d187, 104)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.FreeDesc(&d184)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl54)
			ctx.W.ResolveFixups()
			d188 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			var d189 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r163, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
				ctx.BindReg(r163, &d189)
			}
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d189)
			var d190 scm.JITValueDesc
			if d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d189.Imm.Int()))))}
			} else {
				r164 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r164, d189.Reg)
				ctx.W.EmitShlRegImm8(r164, 56)
				ctx.W.EmitShrRegImm8(r164, 56)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d190)
			}
			ctx.FreeDesc(&d189)
			d191 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d190)
			var d192 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d190.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() - d190.Imm.Int())}
			} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r165, d191.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d192)
			} else if d191.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d191.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d190.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d192)
			} else if d190.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(scratch, d191.Reg)
				if d190.Imm.Int() >= -2147483648 && d190.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d190.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d190.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d192)
			} else {
				r166 := ctx.AllocRegExcept(d191.Reg, d190.Reg)
				ctx.W.EmitMovRegReg(r166, d191.Reg)
				ctx.W.EmitSubInt64(r166, d190.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d192)
			}
			if d192.Loc == scm.LocReg && d191.Loc == scm.LocReg && d192.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d190)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d192)
			var d193 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d188.Imm.Int()) >> uint64(d192.Imm.Int())))}
			} else if d192.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r167, d188.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d192.Imm.Int()))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d193)
			} else {
				{
					shiftSrc := d188.Reg
					r168 := ctx.AllocRegExcept(d188.Reg)
					ctx.W.EmitMovRegReg(r168, d188.Reg)
					shiftSrc = r168
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
					ctx.BindReg(shiftSrc, &d193)
				}
			}
			if d193.Loc == scm.LocReg && d188.Loc == scm.LocReg && d193.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			ctx.FreeDesc(&d192)
			r169 := ctx.AllocReg()
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d193)
			if d193.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r169, d193)
			}
			ctx.W.EmitJmp(lbl52)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl53)
			ctx.W.ResolveFixups()
			d188 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d178)
			var d194 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() % 64)}
			} else {
				r170 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r170, d178.Reg)
				ctx.W.EmitAndRegImm32(r170, 63)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d194)
			}
			if d194.Loc == scm.LocReg && d178.Loc == scm.LocReg && d194.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r171 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r171, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
				ctx.BindReg(r171, &d195)
			}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d195.Reg)
				ctx.W.EmitShlRegImm8(r172, 56)
				ctx.W.EmitShrRegImm8(r172, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d196)
			}
			ctx.FreeDesc(&d195)
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d196)
			var d197 scm.JITValueDesc
			if d194.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d194.Imm.Int() + d196.Imm.Int())}
			} else if d196.Loc == scm.LocImm && d196.Imm.Int() == 0 {
				r173 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(r173, d194.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d197)
			} else if d194.Loc == scm.LocImm && d194.Imm.Int() == 0 {
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d196.Reg}
				ctx.BindReg(d196.Reg, &d197)
			} else if d194.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d194.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(scratch, d194.Reg)
				if d196.Imm.Int() >= -2147483648 && d196.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d196.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d196.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else {
				r174 := ctx.AllocRegExcept(d194.Reg, d196.Reg)
				ctx.W.EmitMovRegReg(r174, d194.Reg)
				ctx.W.EmitAddInt64(r174, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d197)
			}
			if d197.Loc == scm.LocReg && d194.Loc == scm.LocReg && d197.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			ctx.FreeDesc(&d196)
			ctx.EnsureDesc(&d197)
			var d198 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d197.Imm.Int()) > uint64(64))}
			} else {
				r175 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitCmpRegImm32(d197.Reg, 64)
				ctx.W.EmitSetcc(r175, scm.CcA)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r175}
				ctx.BindReg(r175, &d198)
			}
			ctx.FreeDesc(&d197)
			d199 := d198
			ctx.EnsureDesc(&d199)
			if d199.Loc != scm.LocImm && d199.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			if d199.Loc == scm.LocImm {
				if d199.Imm.Bool() {
					ctx.W.MarkLabel(lbl58)
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.MarkLabel(lbl59)
			d200 := d183
			if d200.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d200)
			ctx.EmitStoreToStack(d200, 104)
					ctx.W.EmitJmp(lbl54)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d199.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl58)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl58)
				ctx.W.EmitJmp(lbl57)
				ctx.W.MarkLabel(lbl59)
			d201 := d183
			if d201.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d201)
			ctx.EmitStoreToStack(d201, 104)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.FreeDesc(&d198)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl57)
			ctx.W.ResolveFixups()
			d188 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(104)}
			ctx.EnsureDesc(&d178)
			var d202 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() / 64)}
			} else {
				r176 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r176, d178.Reg)
				ctx.W.EmitShrRegImm8(r176, 6)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d202)
			}
			if d202.Loc == scm.LocReg && d178.Loc == scm.LocReg && d202.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(scratch, d202.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			}
			if d203.Loc == scm.LocReg && d202.Loc == scm.LocReg && d203.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			r177 := ctx.AllocReg()
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d179)
			if d203.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r177, uint64(d203.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r177, d203.Reg)
				ctx.W.EmitShlRegImm8(r177, 3)
			}
			if d179.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
				ctx.W.EmitAddInt64(r177, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r177, d179.Reg)
			}
			r178 := ctx.AllocRegExcept(r177)
			ctx.W.EmitMovRegMem(r178, r177, 0)
			ctx.FreeReg(r177)
			d204 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r178}
			ctx.BindReg(r178, &d204)
			ctx.FreeDesc(&d203)
			ctx.EnsureDesc(&d178)
			var d205 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d178.Imm.Int() % 64)}
			} else {
				r179 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r179, d178.Reg)
				ctx.W.EmitAndRegImm32(r179, 63)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d205)
			}
			if d205.Loc == scm.LocReg && d178.Loc == scm.LocReg && d205.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d178)
			d206 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d205)
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() - d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r180 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r180, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d207)
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
				r181 := ctx.AllocRegExcept(d206.Reg, d205.Reg)
				ctx.W.EmitMovRegReg(r181, d206.Reg)
				ctx.W.EmitSubInt64(r181, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d207)
			}
			if d207.Loc == scm.LocReg && d206.Loc == scm.LocReg && d207.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d207)
			var d208 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d204.Imm.Int()) >> uint64(d207.Imm.Int())))}
			} else if d207.Loc == scm.LocImm {
				r182 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(r182, d204.Reg)
				ctx.W.EmitShrRegImm8(r182, uint8(d207.Imm.Int()))
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d208)
			} else {
				{
					shiftSrc := d204.Reg
					r183 := ctx.AllocRegExcept(d204.Reg)
					ctx.W.EmitMovRegReg(r183, d204.Reg)
					shiftSrc = r183
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d207.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d207.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d207.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d208)
				}
			}
			if d208.Loc == scm.LocReg && d204.Loc == scm.LocReg && d208.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d207)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() | d208.Imm.Int())}
			} else if d183.Loc == scm.LocImm && d183.Imm.Int() == 0 {
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
				ctx.BindReg(d208.Reg, &d209)
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r184, d183.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d209)
			} else if d183.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d183.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == scm.LocImm {
				r185 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r185, d183.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r185, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitOrInt64(r185, scm.RegR11)
				}
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d209)
			} else {
				r186 := ctx.AllocRegExcept(d183.Reg, d208.Reg)
				ctx.W.EmitMovRegReg(r186, d183.Reg)
				ctx.W.EmitOrInt64(r186, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d209)
			}
			if d209.Loc == scm.LocReg && d183.Loc == scm.LocReg && d209.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			d210 := d209
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, 104)
			ctx.W.EmitJmp(lbl54)
			ctx.W.MarkLabel(lbl52)
			d211 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d211)
			ctx.BindReg(r169, &d211)
			if r149 { ctx.UnprotectReg(r150) }
			ctx.FreeDesc(&d88)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d211)
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d211.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r187, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d212)
			}
			ctx.FreeDesc(&d211)
			var d213 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r188, thisptr.Reg, off)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d213)
			}
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() + d213.Imm.Int())}
			} else if d213.Loc == scm.LocImm && d213.Imm.Int() == 0 {
				r189 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r189, d212.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d214)
			} else if d212.Loc == scm.LocImm && d212.Imm.Int() == 0 {
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d213.Reg}
				ctx.BindReg(d213.Reg, &d214)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else if d213.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				if d213.Imm.Int() >= -2147483648 && d213.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d213.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else {
				r190 := ctx.AllocRegExcept(d212.Reg, d213.Reg)
				ctx.W.EmitMovRegReg(r190, d212.Reg)
				ctx.W.EmitAddInt64(r190, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d214)
			}
			if d214.Loc == scm.LocReg && d212.Loc == scm.LocReg && d214.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d215 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint32(idxInt.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d215)
			}
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d214)
			var d216 scm.JITValueDesc
			if d215.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d215.Imm.Int() - d214.Imm.Int())}
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r192 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r192, d215.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d216)
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
				r193 := ctx.AllocRegExcept(d215.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r193, d215.Reg)
				ctx.W.EmitSubInt64(r193, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d216)
			}
			if d216.Loc == scm.LocReg && d215.Loc == scm.LocReg && d216.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d173)
			var d217 scm.JITValueDesc
			if d216.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d216.Imm.Int() * d173.Imm.Int())}
			} else if d216.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d216.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d173.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d216.Reg)
				ctx.W.EmitMovRegReg(scratch, d216.Reg)
				if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d173.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else {
				r194 := ctx.AllocRegExcept(d216.Reg, d173.Reg)
				ctx.W.EmitMovRegReg(r194, d216.Reg)
				ctx.W.EmitImulInt64(r194, d173.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d217)
			}
			if d217.Loc == scm.LocReg && d216.Loc == scm.LocReg && d217.Reg == d216.Reg {
				ctx.TransferReg(d216.Reg)
				d216.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d216)
			ctx.FreeDesc(&d173)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() + d217.Imm.Int())}
			} else if d217.Loc == scm.LocImm && d217.Imm.Int() == 0 {
				r195 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r195, d130.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d218)
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d217.Reg}
				ctx.BindReg(d217.Reg, &d218)
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d218)
			} else if d217.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(scratch, d130.Reg)
				if d217.Imm.Int() >= -2147483648 && d217.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d217.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d217.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d218)
			} else {
				r196 := ctx.AllocRegExcept(d130.Reg, d217.Reg)
				ctx.W.EmitMovRegReg(r196, d130.Reg)
				ctx.W.EmitAddInt64(r196, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d218)
			}
			if d218.Loc == scm.LocReg && d130.Loc == scm.LocReg && d218.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d217)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d218)
			var d219 scm.JITValueDesc
			if d218.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagFloat, Imm: scm.NewFloat(float64(d218.Imm.Int()))}
			} else {
				ctx.W.EmitCvtInt64ToFloat64(scm.RegX0, d218.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagFloat, Reg: d218.Reg}
				ctx.BindReg(d218.Reg, &d219)
			}
			ctx.FreeDesc(&d218)
			ctx.EnsureDesc(&d219)
			d220 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d220)
			ctx.BindReg(r2, &d220)
			ctx.EnsureDesc(&d219)
			ctx.W.EmitMakeFloat(d220, d219)
			if d219.Loc == scm.LocReg { ctx.FreeReg(d219.Reg) }
			ctx.W.EmitJmp(lbl0)
			bbpos_0_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl8)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d221 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d221)
			ctx.BindReg(r2, &d221)
			ctx.W.EmitMakeNil(d221)
			ctx.W.EmitJmp(lbl0)
			bbpos_0_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl12)
			ctx.W.ResolveFixups()
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d222 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			}
			if d222.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: d222.Type, Imm: scm.NewInt(int64(uint64(d222.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d222.Reg, 32)
				ctx.W.EmitShrRegImm8(d222.Reg, 32)
			}
			if d222.Loc == scm.LocReg && d2.Loc == scm.LocReg && d222.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d223 := d222
			if d223.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d223)
			d224 := d223
			if d224.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: d224.Type, Imm: scm.NewInt(int64(uint64(d224.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d224.Reg, 32)
				ctx.W.EmitShrRegImm8(d224.Reg, 32)
			}
			ctx.EmitStoreToStack(d224, 0)
			ctx.W.EmitJmp(lbl1)
			bbpos_0_7 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			ctx.EnsureDesc(&d17)
			var d225 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d17.Imm.Int()) == uint64(0))}
			} else {
				r197 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitCmpRegImm32(d17.Reg, 0)
				ctx.W.EmitSetcc(r197, scm.CcE)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r197}
				ctx.BindReg(r197, &d225)
			}
			d226 := d225
			ctx.EnsureDesc(&d226)
			if d226.Loc != scm.LocImm && d226.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			if d226.Loc == scm.LocImm {
				if d226.Imm.Bool() {
					ctx.W.MarkLabel(lbl62)
					ctx.W.EmitJmp(lbl60)
				} else {
					ctx.W.MarkLabel(lbl63)
					ctx.W.EmitJmp(lbl61)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d226.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl62)
				ctx.W.EmitJmp(lbl63)
				ctx.W.MarkLabel(lbl62)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl63)
				ctx.W.EmitJmp(lbl61)
			}
			ctx.FreeDesc(&d225)
			bbpos_0_11 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl61)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			var d227 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			}
			if d227.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: d227.Type, Imm: scm.NewInt(int64(uint64(d227.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d227.Reg, 32)
				ctx.W.EmitShrRegImm8(d227.Reg, 32)
			}
			if d227.Loc == scm.LocReg && d17.Loc == scm.LocReg && d227.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			var d228 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d228)
			}
			if d228.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: d228.Type, Imm: scm.NewInt(int64(uint64(d228.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d228.Reg, 32)
				ctx.W.EmitShrRegImm8(d228.Reg, 32)
			}
			if d228.Loc == scm.LocReg && d17.Loc == scm.LocReg && d228.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			d229 := d228
			if d229.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d229)
			d230 := d229
			if d230.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: d230.Type, Imm: scm.NewInt(int64(uint64(d230.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d230.Reg, 32)
				ctx.W.EmitShrRegImm8(d230.Reg, 32)
			}
			ctx.EmitStoreToStack(d230, 40)
			d231 := d18
			if d231.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d231)
			d232 := d231
			if d232.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: d232.Type, Imm: scm.NewInt(int64(uint64(d232.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d232.Reg, 32)
				ctx.W.EmitShrRegImm8(d232.Reg, 32)
			}
			ctx.EmitStoreToStack(d232, 48)
			d233 := d227
			if d233.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d233)
			d234 := d233
			if d234.Loc == scm.LocImm {
				d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: d234.Type, Imm: scm.NewInt(int64(uint64(d234.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d234.Reg, 32)
				ctx.W.EmitShrRegImm8(d234.Reg, 32)
			}
			ctx.EmitStoreToStack(d234, 56)
			ctx.W.EmitJmp(lbl4)
			bbpos_0_12 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl27)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d235 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d235)
			}
			if d235.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: d235.Type, Imm: scm.NewInt(int64(uint64(d235.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d235.Reg, 32)
				ctx.W.EmitShrRegImm8(d235.Reg, 32)
			}
			if d235.Loc == scm.LocReg && d2.Loc == scm.LocReg && d235.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			d236 := d235
			if d236.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d236)
			d237 := d236
			if d237.Loc == scm.LocImm {
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: d237.Type, Imm: scm.NewInt(int64(uint64(d237.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d237.Reg, 32)
				ctx.W.EmitShrRegImm8(d237.Reg, 32)
			}
			ctx.EmitStoreToStack(d237, 40)
			d238 := d17
			if d238.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d238)
			d239 := d238
			if d239.Loc == scm.LocImm {
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: d239.Type, Imm: scm.NewInt(int64(uint64(d239.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d239.Reg, 32)
				ctx.W.EmitShrRegImm8(d239.Reg, 32)
			}
			ctx.EmitStoreToStack(d239, 48)
			d240 := d19
			if d240.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d240)
			d241 := d240
			if d241.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: d241.Type, Imm: scm.NewInt(int64(uint64(d241.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d241.Reg, 32)
				ctx.W.EmitShrRegImm8(d241.Reg, 32)
			}
			ctx.EmitStoreToStack(d241, 56)
			ctx.W.EmitJmp(lbl4)
			bbpos_0_13 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl30)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			ctx.EnsureDesc(&d79)
			d242 := d79
			_ = d242
			r198 := d79.Loc == scm.LocReg
			r199 := d79.Reg
			if r198 { ctx.ProtectReg(r199) }
			lbl64 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d242)
			var d243 scm.JITValueDesc
			if d242.Loc == scm.LocImm {
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d242.Imm.Int()))))}
			} else {
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r200, d242.Reg)
				ctx.W.EmitShlRegImm8(r200, 32)
				ctx.W.EmitShrRegImm8(r200, 32)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d243)
			}
			var d244 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r201 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r201, thisptr.Reg, off)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
				ctx.BindReg(r201, &d244)
			}
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d244)
			var d245 scm.JITValueDesc
			if d244.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d244.Imm.Int()))))}
			} else {
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r202, d244.Reg)
				ctx.W.EmitShlRegImm8(r202, 56)
				ctx.W.EmitShrRegImm8(r202, 56)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d245)
			}
			ctx.FreeDesc(&d244)
			ctx.EnsureDesc(&d243)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d243)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d243)
			ctx.EnsureDesc(&d245)
			var d246 scm.JITValueDesc
			if d243.Loc == scm.LocImm && d245.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d243.Imm.Int() * d245.Imm.Int())}
			} else if d243.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d245.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d243.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d245.Reg)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d246)
			} else if d245.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegReg(scratch, d243.Reg)
				if d245.Imm.Int() >= -2147483648 && d245.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d245.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d245.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d246)
			} else {
				r203 := ctx.AllocRegExcept(d243.Reg, d245.Reg)
				ctx.W.EmitMovRegReg(r203, d243.Reg)
				ctx.W.EmitImulInt64(r203, d245.Reg)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d246)
			}
			if d246.Loc == scm.LocReg && d243.Loc == scm.LocReg && d246.Reg == d243.Reg {
				ctx.TransferReg(d243.Reg)
				d243.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d243)
			ctx.FreeDesc(&d245)
			var d247 scm.JITValueDesc
			r204 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r204, uint64(dataPtr))
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204, StackOff: int32(sliceLen)}
				ctx.BindReg(r204, &d247)
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 0)
				ctx.W.EmitMovRegMem(r204, thisptr.Reg, off)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d247)
			}
			ctx.BindReg(r204, &d247)
			ctx.EnsureDesc(&d246)
			var d248 scm.JITValueDesc
			if d246.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() / 64)}
			} else {
				r205 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r205, d246.Reg)
				ctx.W.EmitShrRegImm8(r205, 6)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d248)
			}
			if d248.Loc == scm.LocReg && d246.Loc == scm.LocReg && d248.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d248)
			r206 := ctx.AllocReg()
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d247)
			if d248.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r206, uint64(d248.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r206, d248.Reg)
				ctx.W.EmitShlRegImm8(r206, 3)
			}
			if d247.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d247.Imm.Int()))
				ctx.W.EmitAddInt64(r206, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r206, d247.Reg)
			}
			r207 := ctx.AllocRegExcept(r206)
			ctx.W.EmitMovRegMem(r207, r206, 0)
			ctx.FreeReg(r206)
			d249 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
			ctx.BindReg(r207, &d249)
			ctx.FreeDesc(&d248)
			ctx.EnsureDesc(&d246)
			var d250 scm.JITValueDesc
			if d246.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() % 64)}
			} else {
				r208 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r208, d246.Reg)
				ctx.W.EmitAndRegImm32(r208, 63)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d250)
			}
			if d250.Loc == scm.LocReg && d246.Loc == scm.LocReg && d250.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d250)
			var d251 scm.JITValueDesc
			if d249.Loc == scm.LocImm && d250.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d249.Imm.Int()) << uint64(d250.Imm.Int())))}
			} else if d250.Loc == scm.LocImm {
				r209 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r209, d249.Reg)
				ctx.W.EmitShlRegImm8(r209, uint8(d250.Imm.Int()))
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d251)
			} else {
				{
					shiftSrc := d249.Reg
					r210 := ctx.AllocRegExcept(d249.Reg)
					ctx.W.EmitMovRegReg(r210, d249.Reg)
					shiftSrc = r210
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d250.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d250.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d250.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d251)
				}
			}
			if d251.Loc == scm.LocReg && d249.Loc == scm.LocReg && d251.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			ctx.FreeDesc(&d250)
			var d252 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 25)
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r211, thisptr.Reg, off)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
				ctx.BindReg(r211, &d252)
			}
			d253 := d252
			ctx.EnsureDesc(&d253)
			if d253.Loc != scm.LocImm && d253.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			lbl68 := ctx.W.ReserveLabel()
			if d253.Loc == scm.LocImm {
				if d253.Imm.Bool() {
					ctx.W.MarkLabel(lbl67)
					ctx.W.EmitJmp(lbl65)
				} else {
					ctx.W.MarkLabel(lbl68)
			d254 := d251
			if d254.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d254)
			ctx.EmitStoreToStack(d254, 112)
					ctx.W.EmitJmp(lbl66)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d253.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl67)
				ctx.W.EmitJmp(lbl68)
				ctx.W.MarkLabel(lbl67)
				ctx.W.EmitJmp(lbl65)
				ctx.W.MarkLabel(lbl68)
			d255 := d251
			if d255.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d255)
			ctx.EmitStoreToStack(d255, 112)
				ctx.W.EmitJmp(lbl66)
			}
			ctx.FreeDesc(&d252)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl66)
			ctx.W.ResolveFixups()
			d256 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			var d257 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r212 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r212, thisptr.Reg, off)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r212}
				ctx.BindReg(r212, &d257)
			}
			ctx.EnsureDesc(&d257)
			ctx.EnsureDesc(&d257)
			var d258 scm.JITValueDesc
			if d257.Loc == scm.LocImm {
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d257.Imm.Int()))))}
			} else {
				r213 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r213, d257.Reg)
				ctx.W.EmitShlRegImm8(r213, 56)
				ctx.W.EmitShrRegImm8(r213, 56)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d258)
			}
			ctx.FreeDesc(&d257)
			d259 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d258)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d258)
			var d260 scm.JITValueDesc
			if d259.Loc == scm.LocImm && d258.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d259.Imm.Int() - d258.Imm.Int())}
			} else if d258.Loc == scm.LocImm && d258.Imm.Int() == 0 {
				r214 := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(r214, d259.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d260)
			} else if d259.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d259.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d258.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d260)
			} else if d258.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(scratch, d259.Reg)
				if d258.Imm.Int() >= -2147483648 && d258.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d258.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d258.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d260)
			} else {
				r215 := ctx.AllocRegExcept(d259.Reg, d258.Reg)
				ctx.W.EmitMovRegReg(r215, d259.Reg)
				ctx.W.EmitSubInt64(r215, d258.Reg)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d260)
			}
			if d260.Loc == scm.LocReg && d259.Loc == scm.LocReg && d260.Reg == d259.Reg {
				ctx.TransferReg(d259.Reg)
				d259.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d258)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d260)
			var d261 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d260.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) >> uint64(d260.Imm.Int())))}
			} else if d260.Loc == scm.LocImm {
				r216 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r216, d256.Reg)
				ctx.W.EmitShrRegImm8(r216, uint8(d260.Imm.Int()))
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d261)
			} else {
				{
					shiftSrc := d256.Reg
					r217 := ctx.AllocRegExcept(d256.Reg)
					ctx.W.EmitMovRegReg(r217, d256.Reg)
					shiftSrc = r217
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d260.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d260.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d260.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d261)
				}
			}
			if d261.Loc == scm.LocReg && d256.Loc == scm.LocReg && d261.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d256)
			ctx.FreeDesc(&d260)
			r218 := ctx.AllocReg()
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d261)
			if d261.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r218, d261)
			}
			ctx.W.EmitJmp(lbl64)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl65)
			ctx.W.ResolveFixups()
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d246)
			var d262 scm.JITValueDesc
			if d246.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() % 64)}
			} else {
				r219 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r219, d246.Reg)
				ctx.W.EmitAndRegImm32(r219, 63)
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d262)
			}
			if d262.Loc == scm.LocReg && d246.Loc == scm.LocReg && d262.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			var d263 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 24)
				r220 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r220, thisptr.Reg, off)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r220}
				ctx.BindReg(r220, &d263)
			}
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d263)
			var d264 scm.JITValueDesc
			if d263.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d263.Imm.Int()))))}
			} else {
				r221 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r221, d263.Reg)
				ctx.W.EmitShlRegImm8(r221, 56)
				ctx.W.EmitShrRegImm8(r221, 56)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d264)
			}
			ctx.FreeDesc(&d263)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d264)
			var d265 scm.JITValueDesc
			if d262.Loc == scm.LocImm && d264.Loc == scm.LocImm {
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d262.Imm.Int() + d264.Imm.Int())}
			} else if d264.Loc == scm.LocImm && d264.Imm.Int() == 0 {
				r222 := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(r222, d262.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d265)
			} else if d262.Loc == scm.LocImm && d262.Imm.Int() == 0 {
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d264.Reg}
				ctx.BindReg(d264.Reg, &d265)
			} else if d262.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d264.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d262.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d264.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d265)
			} else if d264.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(scratch, d262.Reg)
				if d264.Imm.Int() >= -2147483648 && d264.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d264.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d264.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d265)
			} else {
				r223 := ctx.AllocRegExcept(d262.Reg, d264.Reg)
				ctx.W.EmitMovRegReg(r223, d262.Reg)
				ctx.W.EmitAddInt64(r223, d264.Reg)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d265)
			}
			if d265.Loc == scm.LocReg && d262.Loc == scm.LocReg && d265.Reg == d262.Reg {
				ctx.TransferReg(d262.Reg)
				d262.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d262)
			ctx.FreeDesc(&d264)
			ctx.EnsureDesc(&d265)
			var d266 scm.JITValueDesc
			if d265.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d265.Imm.Int()) > uint64(64))}
			} else {
				r224 := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitCmpRegImm32(d265.Reg, 64)
				ctx.W.EmitSetcc(r224, scm.CcA)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r224}
				ctx.BindReg(r224, &d266)
			}
			ctx.FreeDesc(&d265)
			d267 := d266
			ctx.EnsureDesc(&d267)
			if d267.Loc != scm.LocImm && d267.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			lbl71 := ctx.W.ReserveLabel()
			if d267.Loc == scm.LocImm {
				if d267.Imm.Bool() {
					ctx.W.MarkLabel(lbl70)
					ctx.W.EmitJmp(lbl69)
				} else {
					ctx.W.MarkLabel(lbl71)
			d268 := d251
			if d268.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d268)
			ctx.EmitStoreToStack(d268, 112)
					ctx.W.EmitJmp(lbl66)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d267.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl70)
				ctx.W.EmitJmp(lbl71)
				ctx.W.MarkLabel(lbl70)
				ctx.W.EmitJmp(lbl69)
				ctx.W.MarkLabel(lbl71)
			d269 := d251
			if d269.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d269)
			ctx.EmitStoreToStack(d269, 112)
				ctx.W.EmitJmp(lbl66)
			}
			ctx.FreeDesc(&d266)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl69)
			ctx.W.ResolveFixups()
			d256 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(112)}
			ctx.EnsureDesc(&d246)
			var d270 scm.JITValueDesc
			if d246.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() / 64)}
			} else {
				r225 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r225, d246.Reg)
				ctx.W.EmitShrRegImm8(r225, 6)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d270)
			}
			if d270.Loc == scm.LocReg && d246.Loc == scm.LocReg && d270.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d270)
			ctx.EnsureDesc(&d270)
			var d271 scm.JITValueDesc
			if d270.Loc == scm.LocImm {
				d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d270.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d270.Reg)
				ctx.W.EmitMovRegReg(scratch, d270.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d271)
			}
			if d271.Loc == scm.LocReg && d270.Loc == scm.LocReg && d271.Reg == d270.Reg {
				ctx.TransferReg(d270.Reg)
				d270.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d270)
			ctx.EnsureDesc(&d271)
			r226 := ctx.AllocReg()
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d247)
			if d271.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r226, uint64(d271.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r226, d271.Reg)
				ctx.W.EmitShlRegImm8(r226, 3)
			}
			if d247.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d247.Imm.Int()))
				ctx.W.EmitAddInt64(r226, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r226, d247.Reg)
			}
			r227 := ctx.AllocRegExcept(r226)
			ctx.W.EmitMovRegMem(r227, r226, 0)
			ctx.FreeReg(r226)
			d272 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r227}
			ctx.BindReg(r227, &d272)
			ctx.FreeDesc(&d271)
			ctx.EnsureDesc(&d246)
			var d273 scm.JITValueDesc
			if d246.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() % 64)}
			} else {
				r228 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r228, d246.Reg)
				ctx.W.EmitAndRegImm32(r228, 63)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d273)
			}
			if d273.Loc == scm.LocReg && d246.Loc == scm.LocReg && d273.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			d274 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d273)
			var d275 scm.JITValueDesc
			if d274.Loc == scm.LocImm && d273.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d274.Imm.Int() - d273.Imm.Int())}
			} else if d273.Loc == scm.LocImm && d273.Imm.Int() == 0 {
				r229 := ctx.AllocRegExcept(d274.Reg)
				ctx.W.EmitMovRegReg(r229, d274.Reg)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d275)
			} else if d274.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d273.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d274.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d273.Reg)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d275)
			} else if d273.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d274.Reg)
				ctx.W.EmitMovRegReg(scratch, d274.Reg)
				if d273.Imm.Int() >= -2147483648 && d273.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d273.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d273.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d275)
			} else {
				r230 := ctx.AllocRegExcept(d274.Reg, d273.Reg)
				ctx.W.EmitMovRegReg(r230, d274.Reg)
				ctx.W.EmitSubInt64(r230, d273.Reg)
				d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d275)
			}
			if d275.Loc == scm.LocReg && d274.Loc == scm.LocReg && d275.Reg == d274.Reg {
				ctx.TransferReg(d274.Reg)
				d274.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d273)
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d275)
			var d276 scm.JITValueDesc
			if d272.Loc == scm.LocImm && d275.Loc == scm.LocImm {
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d272.Imm.Int()) >> uint64(d275.Imm.Int())))}
			} else if d275.Loc == scm.LocImm {
				r231 := ctx.AllocRegExcept(d272.Reg)
				ctx.W.EmitMovRegReg(r231, d272.Reg)
				ctx.W.EmitShrRegImm8(r231, uint8(d275.Imm.Int()))
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d276)
			} else {
				{
					shiftSrc := d272.Reg
					r232 := ctx.AllocRegExcept(d272.Reg)
					ctx.W.EmitMovRegReg(r232, d272.Reg)
					shiftSrc = r232
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d275.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d275.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d275.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d276)
				}
			}
			if d276.Loc == scm.LocReg && d272.Loc == scm.LocReg && d276.Reg == d272.Reg {
				ctx.TransferReg(d272.Reg)
				d272.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d272)
			ctx.FreeDesc(&d275)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d276)
			var d277 scm.JITValueDesc
			if d251.Loc == scm.LocImm && d276.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d251.Imm.Int() | d276.Imm.Int())}
			} else if d251.Loc == scm.LocImm && d251.Imm.Int() == 0 {
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d276.Reg}
				ctx.BindReg(d276.Reg, &d277)
			} else if d276.Loc == scm.LocImm && d276.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(r233, d251.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d277)
			} else if d251.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d276.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d251.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d276.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d277)
			} else if d276.Loc == scm.LocImm {
				r234 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(r234, d251.Reg)
				if d276.Imm.Int() >= -2147483648 && d276.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r234, int32(d276.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d276.Imm.Int()))
					ctx.W.EmitOrInt64(r234, scm.RegR11)
				}
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d277)
			} else {
				r235 := ctx.AllocRegExcept(d251.Reg, d276.Reg)
				ctx.W.EmitMovRegReg(r235, d251.Reg)
				ctx.W.EmitOrInt64(r235, d276.Reg)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d277)
			}
			if d277.Loc == scm.LocReg && d251.Loc == scm.LocReg && d277.Reg == d251.Reg {
				ctx.TransferReg(d251.Reg)
				d251.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d276)
			d278 := d277
			if d278.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d278)
			ctx.EmitStoreToStack(d278, 112)
			ctx.W.EmitJmp(lbl66)
			ctx.W.MarkLabel(lbl64)
			d279 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
			ctx.BindReg(r218, &d279)
			ctx.BindReg(r218, &d279)
			if r198 { ctx.UnprotectReg(r199) }
			ctx.EnsureDesc(&d279)
			ctx.EnsureDesc(&d279)
			var d280 scm.JITValueDesc
			if d279.Loc == scm.LocImm {
				d280 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d279.Imm.Int()))))}
			} else {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r236, d279.Reg)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d280)
			}
			ctx.FreeDesc(&d279)
			var d281 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d281 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).recordId) + 32)
				r237 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r237, thisptr.Reg, off)
				d281 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d281)
			}
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d280)
			ctx.EnsureDesc(&d281)
			var d282 scm.JITValueDesc
			if d280.Loc == scm.LocImm && d281.Loc == scm.LocImm {
				d282 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d280.Imm.Int() + d281.Imm.Int())}
			} else if d281.Loc == scm.LocImm && d281.Imm.Int() == 0 {
				r238 := ctx.AllocRegExcept(d280.Reg)
				ctx.W.EmitMovRegReg(r238, d280.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d282)
			} else if d280.Loc == scm.LocImm && d280.Imm.Int() == 0 {
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d281.Reg}
				ctx.BindReg(d281.Reg, &d282)
			} else if d280.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d281.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d280.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d281.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d282)
			} else if d281.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d280.Reg)
				ctx.W.EmitMovRegReg(scratch, d280.Reg)
				if d281.Imm.Int() >= -2147483648 && d281.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d281.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d281.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d282)
			} else {
				r239 := ctx.AllocRegExcept(d280.Reg, d281.Reg)
				ctx.W.EmitMovRegReg(r239, d280.Reg)
				ctx.W.EmitAddInt64(r239, d281.Reg)
				d282 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d282)
			}
			if d282.Loc == scm.LocReg && d280.Loc == scm.LocReg && d282.Reg == d280.Reg {
				ctx.TransferReg(d280.Reg)
				d280.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d280)
			ctx.FreeDesc(&d281)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d282)
			var d283 scm.JITValueDesc
			if d282.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d282.Imm.Int()))))}
			} else {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r240, d282.Reg)
				ctx.W.EmitShlRegImm8(r240, 32)
				ctx.W.EmitShrRegImm8(r240, 32)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d283)
			}
			ctx.FreeDesc(&d282)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d283)
			var d284 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d283.Loc == scm.LocImm {
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(idxInt.Imm.Int()) < uint64(d283.Imm.Int()))}
			} else if d283.Loc == scm.LocImm {
				r241 := ctx.AllocRegExcept(idxInt.Reg)
				if d283.Imm.Int() >= -2147483648 && d283.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(idxInt.Reg, int32(d283.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d283.Imm.Int()))
					ctx.W.EmitCmpInt64(idxInt.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r241, scm.CcB)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d284)
			} else if idxInt.Loc == scm.LocImm {
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(idxInt.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d283.Reg)
				ctx.W.EmitSetcc(r242, scm.CcB)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r242}
				ctx.BindReg(r242, &d284)
			} else {
				r243 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitCmpInt64(idxInt.Reg, d283.Reg)
				ctx.W.EmitSetcc(r243, scm.CcB)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d284)
			}
			ctx.FreeDesc(&idxInt)
			ctx.FreeDesc(&d283)
			d285 := d284
			ctx.EnsureDesc(&d285)
			if d285.Loc != scm.LocImm && d285.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl72 := ctx.W.ReserveLabel()
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			lbl75 := ctx.W.ReserveLabel()
			if d285.Loc == scm.LocImm {
				if d285.Imm.Bool() {
					ctx.W.MarkLabel(lbl74)
					ctx.W.EmitJmp(lbl72)
				} else {
					ctx.W.MarkLabel(lbl75)
					ctx.W.EmitJmp(lbl73)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d285.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl74)
				ctx.W.EmitJmp(lbl75)
				ctx.W.MarkLabel(lbl74)
				ctx.W.EmitJmp(lbl72)
				ctx.W.MarkLabel(lbl75)
				ctx.W.EmitJmp(lbl73)
			}
			ctx.FreeDesc(&d284)
			bbpos_0_16 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl73)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d286 := d79
			if d286.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d286)
			d287 := d286
			if d287.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: d287.Type, Imm: scm.NewInt(int64(uint64(d287.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d287.Reg, 32)
				ctx.W.EmitShrRegImm8(d287.Reg, 32)
			}
			ctx.EmitStoreToStack(d287, 64)
			d288 := d81
			if d288.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d288)
			d289 := d288
			if d289.Loc == scm.LocImm {
				d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: d289.Type, Imm: scm.NewInt(int64(uint64(d289.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d289.Reg, 32)
				ctx.W.EmitShrRegImm8(d289.Reg, 32)
			}
			ctx.EmitStoreToStack(d289, 72)
			bbpos_0_15 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d291 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			var d292 scm.JITValueDesc
			if d290.Loc == scm.LocImm && d291.Loc == scm.LocImm {
				d292 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d290.Imm.Int()) == uint64(d291.Imm.Int()))}
			} else if d291.Loc == scm.LocImm {
				r244 := ctx.AllocRegExcept(d290.Reg)
				if d291.Imm.Int() >= -2147483648 && d291.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d290.Reg, int32(d291.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d291.Imm.Int()))
					ctx.W.EmitCmpInt64(d290.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r244, scm.CcE)
				d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d292)
			} else if d290.Loc == scm.LocImm {
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d290.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d291.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d292)
			} else {
				r246 := ctx.AllocRegExcept(d290.Reg)
				ctx.W.EmitCmpInt64(d290.Reg, d291.Reg)
				ctx.W.EmitSetcc(r246, scm.CcE)
				d292 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d292)
			}
			d293 := d292
			ctx.EnsureDesc(&d293)
			if d293.Loc != scm.LocImm && d293.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl76 := ctx.W.ReserveLabel()
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			if d293.Loc == scm.LocImm {
				if d293.Imm.Bool() {
					ctx.W.MarkLabel(lbl77)
			d294 := d290
			if d294.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d294)
			d295 := d294
			if d295.Loc == scm.LocImm {
				d295 = scm.JITValueDesc{Loc: scm.LocImm, Type: d295.Type, Imm: scm.NewInt(int64(uint64(d295.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d295.Reg, 32)
				ctx.W.EmitShrRegImm8(d295.Reg, 32)
			}
			ctx.EmitStoreToStack(d295, 32)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl78)
					ctx.W.EmitJmp(lbl76)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d293.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl77)
				ctx.W.EmitJmp(lbl78)
				ctx.W.MarkLabel(lbl77)
			d296 := d290
			if d296.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d296)
			d297 := d296
			if d297.Loc == scm.LocImm {
				d297 = scm.JITValueDesc{Loc: scm.LocImm, Type: d297.Type, Imm: scm.NewInt(int64(uint64(d297.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d297.Reg, 32)
				ctx.W.EmitShrRegImm8(d297.Reg, 32)
			}
			ctx.EmitStoreToStack(d297, 32)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl78)
				ctx.W.EmitJmp(lbl76)
			}
			ctx.FreeDesc(&d292)
			bbpos_0_22 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl41)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			var d298 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSeq)(nil).start) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d298 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSeq)(nil).start) + 64)
				r247 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r247, thisptr.Reg, off)
				d298 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r247}
				ctx.BindReg(r247, &d298)
			}
			ctx.EnsureDesc(&d298)
			ctx.EnsureDesc(&d298)
			var d299 scm.JITValueDesc
			if d298.Loc == scm.LocImm {
				d299 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d298.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r248, d298.Reg)
				d299 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d299)
			}
			ctx.FreeDesc(&d298)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d299)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d299)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d299)
			var d300 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d299.Loc == scm.LocImm {
				d300 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d130.Imm.Int() == d299.Imm.Int())}
			} else if d299.Loc == scm.LocImm {
				r249 := ctx.AllocRegExcept(d130.Reg)
				if d299.Imm.Int() >= -2147483648 && d299.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d130.Reg, int32(d299.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d299.Imm.Int()))
					ctx.W.EmitCmpInt64(d130.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r249, scm.CcE)
				d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r249}
				ctx.BindReg(r249, &d300)
			} else if d130.Loc == scm.LocImm {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d299.Reg)
				ctx.W.EmitSetcc(r250, scm.CcE)
				d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r250}
				ctx.BindReg(r250, &d300)
			} else {
				r251 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitCmpInt64(d130.Reg, d299.Reg)
				ctx.W.EmitSetcc(r251, scm.CcE)
				d300 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r251}
				ctx.BindReg(r251, &d300)
			}
			ctx.FreeDesc(&d130)
			ctx.FreeDesc(&d299)
			d301 := d300
			ctx.EnsureDesc(&d301)
			if d301.Loc != scm.LocImm && d301.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl79 := ctx.W.ReserveLabel()
			lbl80 := ctx.W.ReserveLabel()
			if d301.Loc == scm.LocImm {
				if d301.Imm.Bool() {
					ctx.W.MarkLabel(lbl79)
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.MarkLabel(lbl80)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d301.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl79)
				ctx.W.EmitJmp(lbl80)
				ctx.W.MarkLabel(lbl79)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl80)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d300)
			bbpos_0_10 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl60)
			ctx.W.ResolveFixups()
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl3)
			bbpos_0_14 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl72)
			ctx.W.ResolveFixups()
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d79)
			var d302 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d302 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) == uint64(0))}
			} else {
				r252 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitSetcc(r252, scm.CcE)
				d302 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r252}
				ctx.BindReg(r252, &d302)
			}
			d303 := d302
			ctx.EnsureDesc(&d303)
			if d303.Loc != scm.LocImm && d303.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			lbl83 := ctx.W.ReserveLabel()
			lbl84 := ctx.W.ReserveLabel()
			if d303.Loc == scm.LocImm {
				if d303.Imm.Bool() {
					ctx.W.MarkLabel(lbl83)
					ctx.W.EmitJmp(lbl81)
				} else {
					ctx.W.MarkLabel(lbl84)
					ctx.W.EmitJmp(lbl82)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d303.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl83)
				ctx.W.EmitJmp(lbl84)
				ctx.W.MarkLabel(lbl83)
				ctx.W.EmitJmp(lbl81)
				ctx.W.MarkLabel(lbl84)
				ctx.W.EmitJmp(lbl82)
			}
			ctx.FreeDesc(&d302)
			bbpos_0_18 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl82)
			ctx.W.ResolveFixups()
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d79)
			var d304 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() - 1)}
			} else {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				ctx.W.EmitSubRegImm32(scratch, int32(1))
				d304 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d304)
			}
			if d304.Loc == scm.LocImm {
				d304 = scm.JITValueDesc{Loc: scm.LocImm, Type: d304.Type, Imm: scm.NewInt(int64(uint64(d304.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d304.Reg, 32)
				ctx.W.EmitShrRegImm8(d304.Reg, 32)
			}
			if d304.Loc == scm.LocReg && d79.Loc == scm.LocReg && d304.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			d305 := d80
			if d305.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d305)
			d306 := d305
			if d306.Loc == scm.LocImm {
				d306 = scm.JITValueDesc{Loc: scm.LocImm, Type: d306.Type, Imm: scm.NewInt(int64(uint64(d306.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d306.Reg, 32)
				ctx.W.EmitShrRegImm8(d306.Reg, 32)
			}
			ctx.EmitStoreToStack(d306, 64)
			d307 := d304
			if d307.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d307)
			d308 := d307
			if d308.Loc == scm.LocImm {
				d308 = scm.JITValueDesc{Loc: scm.LocImm, Type: d308.Type, Imm: scm.NewInt(int64(uint64(d308.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d308.Reg, 32)
				ctx.W.EmitShrRegImm8(d308.Reg, 32)
			}
			ctx.EmitStoreToStack(d308, 72)
			ctx.W.EmitJmp(lbl5)
			bbpos_0_19 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl76)
			ctx.W.ResolveFixups()
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			ctx.EnsureDesc(&d290)
			ctx.EnsureDesc(&d291)
			var d309 scm.JITValueDesc
			if d290.Loc == scm.LocImm && d291.Loc == scm.LocImm {
				d309 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d290.Imm.Int() + d291.Imm.Int())}
			} else if d291.Loc == scm.LocImm && d291.Imm.Int() == 0 {
				r253 := ctx.AllocRegExcept(d290.Reg)
				ctx.W.EmitMovRegReg(r253, d290.Reg)
				d309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d309)
			} else if d290.Loc == scm.LocImm && d290.Imm.Int() == 0 {
				d309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d291.Reg}
				ctx.BindReg(d291.Reg, &d309)
			} else if d290.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d291.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d290.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d291.Reg)
				d309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d309)
			} else if d291.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d290.Reg)
				ctx.W.EmitMovRegReg(scratch, d290.Reg)
				if d291.Imm.Int() >= -2147483648 && d291.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d291.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d291.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d309)
			} else {
				r254 := ctx.AllocRegExcept(d290.Reg, d291.Reg)
				ctx.W.EmitMovRegReg(r254, d290.Reg)
				ctx.W.EmitAddInt64(r254, d291.Reg)
				d309 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d309)
			}
			if d309.Loc == scm.LocImm {
				d309 = scm.JITValueDesc{Loc: scm.LocImm, Type: d309.Type, Imm: scm.NewInt(int64(uint64(d309.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d309.Reg, 32)
				ctx.W.EmitShrRegImm8(d309.Reg, 32)
			}
			if d309.Loc == scm.LocReg && d290.Loc == scm.LocReg && d309.Reg == d290.Reg {
				ctx.TransferReg(d290.Reg)
				d290.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d309)
			var d310 scm.JITValueDesc
			if d309.Loc == scm.LocImm {
				d310 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d309.Imm.Int() / 2)}
			} else {
				r255 := ctx.AllocRegExcept(d309.Reg)
				ctx.W.EmitMovRegReg(r255, d309.Reg)
				ctx.W.EmitShrRegImm8(r255, 1)
				d310 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r255}
				ctx.BindReg(r255, &d310)
			}
			if d310.Loc == scm.LocImm {
				d310 = scm.JITValueDesc{Loc: scm.LocImm, Type: d310.Type, Imm: scm.NewInt(int64(uint64(d310.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d310.Reg, 32)
				ctx.W.EmitShrRegImm8(d310.Reg, 32)
			}
			if d310.Loc == scm.LocReg && d309.Loc == scm.LocReg && d310.Reg == d309.Reg {
				ctx.TransferReg(d309.Reg)
				d309.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d309)
			d311 := d310
			if d311.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d311)
			d312 := d311
			if d312.Loc == scm.LocImm {
				d312 = scm.JITValueDesc{Loc: scm.LocImm, Type: d312.Type, Imm: scm.NewInt(int64(uint64(d312.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d312.Reg, 32)
				ctx.W.EmitShrRegImm8(d312.Reg, 32)
			}
			ctx.EmitStoreToStack(d312, 8)
			d313 := d290
			if d313.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d313)
			d314 := d313
			if d314.Loc == scm.LocImm {
				d314 = scm.JITValueDesc{Loc: scm.LocImm, Type: d314.Type, Imm: scm.NewInt(int64(uint64(d314.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d314.Reg, 32)
				ctx.W.EmitShrRegImm8(d314.Reg, 32)
			}
			ctx.EmitStoreToStack(d314, 16)
			d315 := d291
			if d315.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d315)
			d316 := d315
			if d316.Loc == scm.LocImm {
				d316 = scm.JITValueDesc{Loc: scm.LocImm, Type: d316.Type, Imm: scm.NewInt(int64(uint64(d316.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d316.Reg, 32)
				ctx.W.EmitShrRegImm8(d316.Reg, 32)
			}
			ctx.EmitStoreToStack(d316, 24)
			ctx.W.EmitJmp(lbl2)
			bbpos_0_20 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl6)
			ctx.W.ResolveFixups()
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d317 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d317)
			ctx.BindReg(r2, &d317)
			ctx.W.EmitMakeNil(d317)
			ctx.W.EmitJmp(lbl0)
			bbpos_0_17 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl81)
			ctx.W.ResolveFixups()
			d19 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			d79 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			d81 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(56)}
			d290 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(64)}
			d11 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d18 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			d88 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			d80 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(48)}
			d291 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(72)}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 32)
			ctx.W.EmitJmp(lbl3)
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
