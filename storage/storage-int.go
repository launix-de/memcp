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
	chunk   []uint64
	bitsize uint8
	offset  int64
	max     int64  // only of statistic use
	count   uint64 // only stored for serialization purposes
	hasNull bool
	null    uint64 // which value is null
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
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r2, fieldAddr)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			} else {
				panic("FieldAddr: thisptr not scm.LocImm")
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
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				r3 := ctx.AllocReg()
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
				ctx.W.EmitMovRegMem64(r4, fieldAddr+8)
				d4 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r3, Reg2: r4}
			} else {
				panic("FieldAddr: thisptr not scm.LocImm")
			}
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r5, d3.Reg)
				ctx.W.EmitShrRegImm8(r5, 6)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			}
			if d5.Loc == scm.LocReg && d3.Loc == scm.LocReg && d5.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			r6 := ctx.AllocReg()
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r6, uint64(d5.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r6, d5.Reg)
				ctx.W.EmitShlRegImm8(r6, 3)
			}
			ctx.W.EmitAddInt64(r6, d4.Reg)
			r7 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r7, r6, 0)
			ctx.FreeReg(r6)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d3.Reg)
				ctx.W.EmitAndRegImm32(r8, 63)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
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
			if d3.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				ctx.W.EmitAndRegImm32(r9, 63)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			}
			if d9.Loc == scm.LocReg && d3.Loc == scm.LocReg && d9.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() + d1.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d1.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitAddInt64(d9.Reg, scratch)
				ctx.FreeReg(scratch)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			} else {
				ctx.W.EmitAddInt64(d9.Reg, d1.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			}
			if d11.Loc == scm.LocReg && d9.Loc == scm.LocReg && d11.Reg == d9.Reg {
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			var d12 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d11.Imm.Int() > 64)}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d11.Reg, 64)
				ctx.W.EmitSetcc(r10, scm.CcG)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
			}
			ctx.FreeDesc(&d11)
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.EmitMovToReg(r0, d8)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.EmitMovToReg(r0, d8)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d12)
			ctx.W.MarkLabel(lbl3)
			d13 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r0}
			d15 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d16 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() - d1.Imm.Int())}
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d15.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d1.Reg)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitSubInt64(d15.Reg, scratch)
				ctx.FreeReg(scratch)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			} else {
				ctx.W.EmitSubInt64(d15.Reg, d1.Reg)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			}
			if d16.Loc == scm.LocReg && d15.Loc == scm.LocReg && d16.Reg == d15.Reg {
				d15.Loc = scm.LocNone
			}
			var d17 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) >> uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d13.Reg, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else {
				{
					shiftSrc := d13.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d16.Reg != scm.RegRCX
					var rcxSave scm.Reg
					if rcxUsed {
						rcxSave = ctx.AllocReg()
						ctx.W.EmitMovRegReg(rcxSave, scm.RegRCX)
					}
					if d16.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d16.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, rcxSave)
						ctx.FreeReg(rcxSave)
					}
					d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d17.Loc == scm.LocReg && d13.Loc == scm.LocReg && d17.Reg == d13.Reg {
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d16)
			ctx.EmitMovToReg(r1, d17)
			ctx.W.EmitJmp(lbl1)
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl2)
			var d18 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r11, d3.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			}
			if d18.Loc == scm.LocReg && d3.Loc == scm.LocReg && d18.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(d18.Reg, scratch)
				ctx.FreeReg(scratch)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			}
			if d19.Loc == scm.LocReg && d18.Loc == scm.LocReg && d19.Reg == d18.Reg {
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			r12 := ctx.AllocReg()
			if d19.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d19.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d19.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			ctx.W.EmitAddInt64(r12, d4.Reg)
			r13 := ctx.AllocReg()
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d20 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.FreeDesc(&d19)
			var d21 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d3.Reg, 63)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d3.Reg}
			}
			if d21.Loc == scm.LocReg && d3.Loc == scm.LocReg && d21.Reg == d3.Reg {
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			d22 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() - d21.Imm.Int())}
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitSubInt64(d22.Reg, scratch)
				ctx.FreeReg(scratch)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			} else {
				ctx.W.EmitSubInt64(d22.Reg, d21.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			var d24 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d20.Imm.Int()) >> uint64(d23.Imm.Int())))}
			} else if d23.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d20.Reg, uint8(d23.Imm.Int()))
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			} else {
				{
					shiftSrc := d20.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d23.Reg != scm.RegRCX
					var rcxSave scm.Reg
					if rcxUsed {
						rcxSave = ctx.AllocReg()
						ctx.W.EmitMovRegReg(rcxSave, scm.RegRCX)
					}
					if d23.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d23.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, rcxSave)
						ctx.FreeReg(rcxSave)
					}
					d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d24.Loc == scm.LocReg && d20.Loc == scm.LocReg && d24.Reg == d20.Reg {
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.FreeDesc(&d23)
			var d25 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() | d24.Imm.Int())}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitOrInt64(d8.Reg, scratch)
				ctx.FreeReg(scratch)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
			} else {
				ctx.W.EmitOrInt64(d8.Reg, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
			}
			if d25.Loc == scm.LocReg && d8.Loc == scm.LocReg && d25.Reg == d8.Reg {
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d24)
			ctx.EmitMovToReg(r0, d25)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d26 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			ctx.FreeDesc(&idxInt)
			var d27 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r14, fieldAddr)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			} else {
				panic("FieldAddr: thisptr not scm.LocImm")
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl6)
			var d29 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r15, fieldAddr)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			} else {
				panic("FieldAddr: thisptr not scm.LocImm")
			}
			var d30 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() + d29.Imm.Int())}
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d26.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d26.Reg)
				ctx.W.EmitAddInt64(r16, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			if d30.Loc == scm.LocReg && d26.Loc == scm.LocReg && d30.Reg == d26.Reg {
				d26.Loc = scm.LocNone
			}
			ctx.W.EmitMakeInt(result, d30)
			if d30.Loc == scm.LocReg { ctx.FreeReg(d30.Reg) }
			result.Type = scm.TagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl5)
			var d31 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r17, fieldAddr)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
			} else {
				panic("FieldAddr: thisptr not scm.LocImm")
			}
			var d32 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d26.Imm.Int() == d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm {
				r18 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d26.Reg, int32(d31.Imm.Int()))
				ctx.W.EmitSetcc(r18, scm.CcE)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r18}
			} else if d26.Loc == scm.LocImm {
				r19 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d31.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r19, scm.CcE)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r19}
			} else {
				r20 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d26.Reg, d31.Reg)
				ctx.W.EmitSetcc(r20, scm.CcE)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r20}
			}
			ctx.FreeDesc(&d26)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d32.Loc == scm.LocImm {
				if d32.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d32.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d32)
			ctx.W.MarkLabel(lbl8)
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
	if bitpos%64+uint(s.bitsize) > 64 {
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
