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
import "unsafe"
import "math/bits"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type StorageInt struct {
	chunk      []uint64 `jit:"immutable-after-finish"`
	bitsize    uint8    `jit:"immutable-after-finish"`
	crossWord  bool     `jit:"immutable-after-finish"` // true when 64 % bitsize != 0 (cross-word spanning possible)
	offset     int64    `jit:"immutable-after-finish"`
	max        int64    // only of statistic use
	count      uint64   // only stored for serialization purposes
	hasNull    bool     `jit:"immutable-after-finish"`
	null       uint64   `jit:"immutable-after-finish"`
}

func (s *StorageInt) Serialize(f io.Writer) {
	var hasNull uint8
	if s.hasNull {
		hasNull = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(10))            // 10 = StorageInt
	binary.Write(f, binary.LittleEndian, uint8(s.bitsize))     // len=2
	binary.Write(f, binary.LittleEndian, uint8(hasNull))       // len=3
	binary.Write(f, binary.LittleEndian, uint8(0))             // len=4
	binary.Write(f, binary.LittleEndian, uint32(0))            // len=8
	binary.Write(f, binary.LittleEndian, uint64(len(s.chunk))) // chunk size so we know how many data is left
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(s.offset))
	binary.Write(f, binary.LittleEndian, uint64(s.null))
	if len(s.chunk) > 0 {
		f.Write(unsafe.Slice((*byte)(unsafe.Pointer(&s.chunk[0])), 8*len(s.chunk)))
	}
}
func (s *StorageInt) Deserialize(f io.Reader) uint {
	return s.DeserializeEx(f, false)
}

func (s *StorageInt) DeserializeEx(f io.Reader, readMagicbyte bool) uint {
	var dummy8 uint8
	var dummy32 uint32
	if readMagicbyte {
		binary.Read(f, binary.LittleEndian, &dummy8)
		if dummy8 != 10 {
			panic(fmt.Sprintf("Tried to deserialize StorageInt(10) from file but found %d", dummy8))
		}
	}
	binary.Read(f, binary.LittleEndian, &s.bitsize)
	var hasNull uint8
	binary.Read(f, binary.LittleEndian, &hasNull)
	s.hasNull = hasNull != 0
	binary.Read(f, binary.LittleEndian, &dummy8)
	binary.Read(f, binary.LittleEndian, &dummy32)
	var chunkcount uint64
	binary.Read(f, binary.LittleEndian, &chunkcount)
	binary.Read(f, binary.LittleEndian, &s.count)
	binary.Read(f, binary.LittleEndian, &s.offset)
	binary.Read(f, binary.LittleEndian, &s.null)
	if chunkcount > 0 {
		rawdata := make([]byte, chunkcount*8)
		f.Read(rawdata)
		s.chunk = unsafe.Slice((*uint64)(unsafe.Pointer(&rawdata[0])), chunkcount)
	}
	s.crossWord = s.bitsize > 0 && 64%uint(s.bitsize) != 0
	return uint(s.count)
}

func (s *StorageInt) ComputeSize() uint {
	return 8*uint(len(s.chunk)) + 64 // management overhead
}

func (s *StorageInt) String() string {
	if s.hasNull {
		return fmt.Sprintf("int[%d]NULL", s.bitsize)
	} else {
		return fmt.Sprintf("int[%d]", s.bitsize)
	}
}

func (s *StorageInt) GetCachedReader() ColumnReader { return s }

func (s *StorageInt) GetValue(i uint32) scm.Scmer {
	v := s.GetValueUInt(i)
	if s.hasNull && v == s.null {
		return scm.NewNil()
	}
	return scm.NewInt(int64(v) + s.offset)
}
func (s *StorageInt) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
			} else {
				idxInt = idx
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			r0 := ctx.AllocReg()
			r1 := ctx.AllocReg()
			lbl1 := ctx.W.ReserveLabel()
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r2, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			}
			var d3 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d1.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d1.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(idxInt.Reg, scratch)
				ctx.FreeReg(scratch)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			} else {
				ctx.W.EmitImulInt64(idxInt.Reg, d1.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			}
			if d3.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d3.Reg == idxInt.Reg {
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&idxInt)
			ctx.FreeDesc(&d1)
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(r3, uint64(dataPtr))
				ctx.W.EmitMovRegImm64(r4, uint64(sliceLen))
				d4 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				r5 := ctx.AllocReg()
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r5, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r6, thisptr.Reg, off+8)
				d4 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r5, Reg2: r6}
			}
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r7, d3.Reg)
				ctx.W.EmitShrRegImm8(r7, 6)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			}
			if d5.Loc == scm.LocReg && d3.Loc == scm.LocReg && d5.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			r8 := ctx.AllocReg()
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r8, uint64(d5.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r8, d5.Reg)
				ctx.W.EmitShlRegImm8(r8, 3)
			}
			ctx.W.EmitAddInt64(r8, d4.Reg)
			r9 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r9, r8, 0)
			ctx.FreeReg(r8)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r10, d3.Reg)
				ctx.W.EmitAndRegImm32(r10, 63)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			}
			if d7.Loc == scm.LocReg && d3.Loc == scm.LocReg && d7.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) << uint64(d7.Imm.Int())))}
			} else if d7.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d6.Reg, uint8(d7.Imm.Int()))
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d6.Reg}
			} else {
				{
					shiftSrc := d6.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d7.Reg != scm.RegRCX
					var rcxSave scm.Reg
					if rcxUsed {
						rcxSave = ctx.AllocReg()
						ctx.W.EmitMovRegReg(rcxSave, scm.RegRCX)
					}
					if d7.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d7.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, rcxSave)
						ctx.FreeReg(rcxSave)
					}
					d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			ctx.FreeDesc(&d7)
			var d9 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).crossWord)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).crossWord))
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r11, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d9.Loc == scm.LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.EmitMovToReg(r0, d8)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.EmitMovToReg(r0, d8)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl3)
			d10 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r0}
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r12, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			}
			d13 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d14 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() - d11.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d11.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.W.EmitSubInt64(d13.Reg, scratch)
				ctx.FreeReg(scratch)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else {
				ctx.W.EmitSubInt64(d13.Reg, d11.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			}
			if d14.Loc == scm.LocReg && d13.Loc == scm.LocReg && d14.Reg == d13.Reg {
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			var d15 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) >> uint64(d14.Imm.Int())))}
			} else if d14.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d10.Reg, uint8(d14.Imm.Int()))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d10.Reg}
			} else {
				{
					shiftSrc := d10.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d14.Reg != scm.RegRCX
					var rcxSave scm.Reg
					if rcxUsed {
						rcxSave = ctx.AllocReg()
						ctx.W.EmitMovRegReg(rcxSave, scm.RegRCX)
					}
					if d14.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d14.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, rcxSave)
						ctx.FreeReg(rcxSave)
					}
					d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d15.Loc == scm.LocReg && d10.Loc == scm.LocReg && d15.Reg == d10.Reg {
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d14)
			ctx.EmitMovToReg(r1, d15)
			ctx.W.EmitJmp(lbl1)
			ctx.FreeDesc(&d15)
			ctx.W.MarkLabel(lbl2)
			var d16 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r13, d3.Reg)
				ctx.W.EmitAndRegImm32(r13, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			}
			if d16.Loc == scm.LocReg && d3.Loc == scm.LocReg && d16.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			}
			var d19 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() + d17.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitAddInt64(d16.Reg, scratch)
				ctx.FreeReg(scratch)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			} else {
				ctx.W.EmitAddInt64(d16.Reg, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d16.Reg}
			}
			if d19.Loc == scm.LocReg && d16.Loc == scm.LocReg && d19.Reg == d16.Reg {
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.FreeDesc(&d17)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d19.Imm.Int() > 64)}
			} else {
				r15 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d19.Reg, 64)
				ctx.W.EmitSetcc(r15, scm.CcG)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r15}
			}
			ctx.FreeDesc(&d19)
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d20.Loc == scm.LocImm {
				if d20.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.EmitMovToReg(r0, d8)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d20.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.EmitMovToReg(r0, d8)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl5)
			var d21 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d3.Reg)
				ctx.W.EmitShrRegImm8(r16, 6)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			if d21.Loc == scm.LocReg && d3.Loc == scm.LocReg && d21.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d21.Reg, scratch)
				ctx.FreeReg(scratch)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
			}
			if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			r17 := ctx.AllocReg()
			if d22.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r17, uint64(d22.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r17, d22.Reg)
				ctx.W.EmitShlRegImm8(r17, 3)
			}
			ctx.W.EmitAddInt64(r17, d4.Reg)
			r18 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r18, r17, 0)
			ctx.FreeReg(r17)
			d23 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			ctx.FreeDesc(&d22)
			var d24 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d3.Reg, 63)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d3.Reg}
			}
			if d24.Loc == scm.LocReg && d3.Loc == scm.LocReg && d24.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			d25 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitSubInt64(d25.Reg, scratch)
				ctx.FreeReg(scratch)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			} else {
				ctx.W.EmitSubInt64(d25.Reg, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
			}
			if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			var d27 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d26.Imm.Int())))}
			} else if d26.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d23.Reg, uint8(d26.Imm.Int()))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else {
				{
					shiftSrc := d23.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d26.Reg != scm.RegRCX
					var rcxSave scm.Reg
					if rcxUsed {
						rcxSave = ctx.AllocReg()
						ctx.W.EmitMovRegReg(rcxSave, scm.RegRCX)
					}
					if d26.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d26.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, rcxSave)
						ctx.FreeReg(rcxSave)
					}
					d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d27.Loc == scm.LocReg && d23.Loc == scm.LocReg && d27.Reg == d23.Reg {
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d26)
			var d28 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() | d27.Imm.Int())}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.W.EmitOrInt64(d8.Reg, scratch)
				ctx.FreeReg(scratch)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
			} else {
				ctx.W.EmitOrInt64(d8.Reg, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
			}
			if d28.Loc == scm.LocReg && d8.Loc == scm.LocReg && d28.Reg == d8.Reg {
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d27)
			ctx.EmitMovToReg(r0, d28)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d29 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.FreeDesc(&idxInt)
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d30.Loc == scm.LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d30)
			ctx.W.MarkLabel(lbl8)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r20, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			}
			var d33 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r21, d29.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d29.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r22, d29.Reg)
				ctx.W.EmitAddInt64(r22, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
			}
			if d33.Loc == scm.LocReg && d29.Loc == scm.LocReg && d33.Reg == d29.Reg {
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.W.EmitMakeInt(result, d33)
			if d33.Loc == scm.LocReg { ctx.FreeReg(d33.Reg) }
			result.Type = scm.TagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			var d34 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r23, thisptr.Reg, off)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			}
			var d35 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d29.Imm.Int() == d34.Imm.Int())}
			} else if d34.Loc == scm.LocImm {
				r24 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d29.Reg, int32(d34.Imm.Int()))
				ctx.W.EmitSetcc(r24, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r24}
			} else if d29.Loc == scm.LocImm {
				r25 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d34.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r25, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d29.Reg, d34.Reg)
				ctx.W.EmitSetcc(r26, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r26}
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d34)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			return result
}

// SetValue overwrites a single element in the bit-packed array.
// The new value must fit within the existing [offset, offset+2^bitsize) range.
// Caller must hold the shard write lock.
func (s *StorageInt) SetValue(i uint32, value scm.Scmer) {
	var vi int64
	if value.IsNil() {
		vi = int64(s.null)
	} else {
		vi = value.Int() - s.offset
	}
	bitpos := uint(i) * uint(s.bitsize)
	mask := uint64((1<<uint(s.bitsize))-1) << (64 - uint(s.bitsize)) // bitsize ones at MSB
	v := uint64(vi) << (64 - uint(s.bitsize))
	// clear old bits then set new bits in first chunk
	shifted := mask >> (bitpos % 64)
	s.chunk[bitpos/64] = (s.chunk[bitpos/64] & ^shifted) | (v >> (bitpos % 64))
	if bitpos%64+uint(s.bitsize) > 64 {
		// spans two chunks
		shifted2 := mask << (64 - bitpos%64)
		s.chunk[bitpos/64+1] = (s.chunk[bitpos/64+1] & ^shifted2) | (v << (64 - bitpos%64))
	}
}

func (s *StorageInt) GetValueUInt(i uint32) uint64 {
	bitpos := uint(i) * uint(s.bitsize)

	v := s.chunk[bitpos/64] << (bitpos % 64) // align to leftmost position
	if s.crossWord && bitpos%64+uint(s.bitsize) > 64 {
		v = v | s.chunk[bitpos/64+1]>>(64-bitpos%64)
	}

	return uint64(v) >> (64 - uint(s.bitsize)) // shift right without sign
}

func (s *StorageInt) prepare() {
	// set up scan
	s.bitsize = 0
	s.offset = int64(1<<63 - 1)
	s.max = -s.offset - 1
	s.hasNull = false
}
func (s *StorageInt) scan(i uint32, value scm.Scmer) {
	// storage is so simple, dont need scan
	if value.IsNil() {
		s.hasNull = true
		return
	}
	v := value.Int()
	if v < s.offset {
		s.offset = v
	}
	if v > s.max {
		s.max = v
	}
}
func (s *StorageInt) init(i uint32) {
	v := s.max - s.offset
	if s.hasNull {
		// store the value
		v = v + 1
		s.null = uint64(v)
	}
	if v == -1 {
		// no values at all
		v = 0
		s.offset = 0
		s.null = 0
	}
	s.bitsize = uint8(bits.Len64(uint64(v)))
	if s.bitsize == 0 {
		s.bitsize = 1
	}
	s.crossWord = 64%uint(s.bitsize) != 0
	// allocate
	s.chunk = make([]uint64, ((uint(i)-1)*uint(s.bitsize)+65)/64+1)
	s.count = uint64(i)
	// fmt.Println("storing bitsize", s.bitsize,"null",s.null,"offset",s.offset)
}
func (s *StorageInt) build(i uint32, value scm.Scmer) {
	if i >= uint32(s.count) {
		panic("tried to build StorageInt outside of range")
	}
	// store
	vi := value.Int()
	if value.IsNil() {
		// null value
		vi = int64(s.null)
	} else {
		vi = vi - s.offset
	}
	bitpos := uint(i) * uint(s.bitsize)
	v := uint64(vi) << (64 - uint(s.bitsize))                      // shift value to the leftmost position of 64bit int
	s.chunk[bitpos/64] = s.chunk[bitpos/64] | (v >> (bitpos % 64)) // first chunk
	if bitpos%64+uint(s.bitsize) > 64 {
		s.chunk[bitpos/64+1] = s.chunk[bitpos/64+1] | v<<(64-bitpos%64) // second chunk
	}
}
func (s *StorageInt) finish() {
}
func (s *StorageInt) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}
