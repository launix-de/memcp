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
import "strings"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

type StorageString struct {
	// StorageInt for dictionary entries
	values StorageInt
	// the dictionary: bitcompress all start+end markers; use one big string for all values that is sliced of from
	dictionary string
	starts     StorageInt
	lens       StorageInt
	nodict     bool // disable values array

	// helpers
	sb         strings.Builder
	reverseMap map[string][3]uint
	count      uint
	allsize    int
	// prefix statistics
	prefixstat map[string]int
	laststr    string
}

func (s *StorageString) ComputeSize() uint {
	return s.values.ComputeSize() + 8 + uint(len(s.dictionary)) + 24 + s.starts.ComputeSize() + s.lens.ComputeSize() + 8*8
}

func (s *StorageString) String() string {
	if s.nodict {
		return fmt.Sprintf("string-buffer[%d bytes]", len(s.dictionary))
	} else {
		return fmt.Sprintf("string-dict[%d entries; %d bytes]", s.count, len(s.dictionary))
	}
}

func (s *StorageString) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(20)) // 20 = StorageString
	var nodict uint8 = 0
	if s.nodict {
		nodict = 1
	}
	binary.Write(f, binary.LittleEndian, uint8(nodict))
	io.WriteString(f, "123456") // dummy
	if s.nodict {
		binary.Write(f, binary.LittleEndian, uint64(s.starts.count))
	} else {
		binary.Write(f, binary.LittleEndian, uint64(s.values.count))
	}
	s.values.Serialize(f)
	s.starts.Serialize(f)
	s.lens.Serialize(f)
	binary.Write(f, binary.LittleEndian, uint64(len(s.dictionary)))
	io.WriteString(f, s.dictionary)
}

func (s *StorageString) Deserialize(f io.Reader) uint {
	var nodict uint8
	binary.Read(f, binary.LittleEndian, &nodict)
	if nodict == 1 {
		s.nodict = true
	}
	var dummy [6]byte
	f.Read(dummy[:])
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.values.DeserializeEx(f, true)
	s.count = s.starts.DeserializeEx(f, true)
	s.lens.DeserializeEx(f, true)
	var dictionarylength uint64
	binary.Read(f, binary.LittleEndian, &dictionarylength)
	if dictionarylength > 0 {
		rawdata := make([]byte, dictionarylength)
		f.Read(rawdata)
		s.dictionary = unsafe.String(&rawdata[0], dictionarylength)
	}
	return uint(l)
}

func (s *StorageString) GetCachedReader() ColumnReader { return s }

func (s *StorageString) GetValue(i uint32) scm.Scmer {
	if s.nodict {
		start := uint64(int64(s.starts.GetValueUInt(i)) + s.starts.offset)
		if s.starts.hasNull && start == s.starts.null {
			return scm.NewNil()
		}
		len_ := uint64(int64(s.lens.GetValueUInt(i)) + s.lens.offset)
		startIdx := int(start)
		endIdx := int(start + len_)
		return scm.NewString(s.dictionary[startIdx:endIdx])
	} else {
		idx := uint32(int64(s.values.GetValueUInt(i)) + s.values.offset)
		if s.values.hasNull && idx == uint32(s.values.null) {
			return scm.NewNil()
		}
		start := int64(s.starts.GetValueUInt(idx)) + s.starts.offset
		len_ := int64(s.lens.GetValueUInt(idx)) + s.lens.offset
		startIdx := int(start)
		endIdx := int(start + len_)
		return scm.NewString(s.dictionary[startIdx:endIdx])
	}
}
func (s *StorageString) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
				r0 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r0, fieldAddr)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
				r1 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r1, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
			}
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d0.Loc == scm.LocImm {
				if d0.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d0.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.W.MarkLabel(lbl2)
			if idxInt.Loc == scm.LocReg { ctx.ProtectReg(idxInt.Reg) }
			r2 := ctx.W.EmitSubRSP32Fixup()
			r3 := ctx.AllocReg()
			lbl4 := ctx.W.ReserveLabel()
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r4, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			}
			var d4 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d2.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d2.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, idxInt.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r5 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(r5, idxInt.Reg)
				ctx.W.EmitImulInt64(r5, d2.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			}
			if d4.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d4.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			var d5 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r6, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			}
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r7 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r7, d4.Reg)
				ctx.W.EmitShrRegImm8(r7, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			r8 := ctx.AllocReg()
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r8, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r8, d6.Reg)
				ctx.W.EmitShlRegImm8(r8, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r8, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r8, d5.Reg)
			}
			r9 := ctx.AllocRegExcept(r8)
			ctx.W.EmitMovRegMem(r9, r8, 0)
			ctx.FreeReg(r8)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.FreeDesc(&d6)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r10 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitAndRegImm32(r10, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) << uint64(d8.Imm.Int())))}
			} else if d8.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d7.Reg, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d7.Reg}
			} else {
				{
					shiftSrc := d7.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d8.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d8.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d8.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d9.Loc == scm.LocReg && d7.Loc == scm.LocReg && d9.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d7)
			ctx.FreeDesc(&d8)
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 25)
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r11, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d10.Loc == scm.LocImm {
				if d10.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.EmitStoreToStack(d9, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
				ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl6)
			r12 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r12, 0)
			ctx.ProtectReg(r12)
			d11 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r12}
			ctx.UnprotectReg(r12)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r13, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			}
			d14 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d12.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else if d14.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d12.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitSubInt64(d14.Reg, scm.RegR11)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else {
				ctx.W.EmitSubInt64(d14.Reg, d12.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			}
			if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			var d16 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) >> uint64(d15.Imm.Int())))}
			} else if d15.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d11.Reg, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d11.Reg}
			} else {
				{
					shiftSrc := d11.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d15.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d15.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d15.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d16.Loc == scm.LocReg && d11.Loc == scm.LocReg && d16.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d15)
			ctx.EmitMovToReg(r3, d16)
			ctx.W.EmitJmp(lbl4)
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl5)
			var d17 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r14, d4.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			}
			if d17.Loc == scm.LocReg && d4.Loc == scm.LocReg && d17.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			}
			var d20 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d18.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.W.EmitAddInt64(d17.Reg, scm.RegR11)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else {
				ctx.W.EmitAddInt64(d17.Reg, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			}
			if d20.Loc == scm.LocReg && d17.Loc == scm.LocReg && d20.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d18)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d20.Imm.Int() > 64)}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d20.Reg, 64)
				ctx.W.EmitSetcc(r16, scm.CcG)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r16}
			}
			ctx.FreeDesc(&d20)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d21.Loc == scm.LocImm {
				if d21.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.EmitStoreToStack(d9, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d21)
			ctx.W.MarkLabel(lbl8)
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r17 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r17, d4.Reg)
				ctx.W.EmitShrRegImm8(r17, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d22.Reg, scm.RegR11)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			r18 := ctx.AllocReg()
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r18, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r18, d23.Reg)
				ctx.W.EmitShlRegImm8(r18, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r18, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r18, d5.Reg)
			}
			r19 := ctx.AllocRegExcept(r18)
			ctx.W.EmitMovRegMem(r19, r18, 0)
			ctx.FreeReg(r18)
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			ctx.FreeDesc(&d23)
			var d25 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d4.Reg, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d4.Reg}
			}
			if d25.Loc == scm.LocReg && d4.Loc == scm.LocReg && d25.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d26 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d25.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(d26.Reg, scm.RegR11)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			} else {
				ctx.W.EmitSubInt64(d26.Reg, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			var d28 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d24.Reg, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else {
				{
					shiftSrc := d24.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d27.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d27.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d27.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d28.Loc == scm.LocReg && d24.Loc == scm.LocReg && d28.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d27)
			var d29 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d28.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
				ctx.W.EmitOrInt64(d9.Reg, scratch)
				ctx.FreeReg(scratch)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			} else {
				ctx.W.EmitOrInt64(d9.Reg, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d9.Reg}
			}
			if d29.Loc == scm.LocReg && d9.Loc == scm.LocReg && d29.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EmitStoreToStack(d29, 0)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
			if idxInt.Loc == scm.LocReg { ctx.UnprotectReg(idxInt.Reg) }
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r20, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			}
			var d33 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.W.EmitAddInt64(d30.Reg, scm.RegR11)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else {
				ctx.W.EmitAddInt64(d30.Reg, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			}
			if d33.Loc == scm.LocReg && d30.Loc == scm.LocReg && d33.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d32)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r21, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl1)
			if idxInt.Loc == scm.LocReg { ctx.ProtectReg(idxInt.Reg) }
			r22 := ctx.AllocReg()
			lbl13 := ctx.W.ReserveLabel()
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r23, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			}
			var d39 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d37.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, idxInt.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r24 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(r24, idxInt.Reg)
				ctx.W.EmitImulInt64(r24, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			}
			if d39.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d39.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r25, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			}
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r26 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r26, d39.Reg)
				ctx.W.EmitShrRegImm8(r26, 6)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			r27 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r27, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r27, d41.Reg)
				ctx.W.EmitShlRegImm8(r27, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r27, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r27, d40.Reg)
			}
			r28 := ctx.AllocRegExcept(r27)
			ctx.W.EmitMovRegMem(r28, r27, 0)
			ctx.FreeReg(r27)
			d42 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r29 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r29, d39.Reg)
				ctx.W.EmitAndRegImm32(r29, 63)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			}
			if d43.Loc == scm.LocReg && d39.Loc == scm.LocReg && d43.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d44 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) << uint64(d43.Imm.Int())))}
			} else if d43.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d42.Reg, uint8(d43.Imm.Int()))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
			} else {
				{
					shiftSrc := d42.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d43.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d43.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d43.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d44.Loc == scm.LocReg && d42.Loc == scm.LocReg && d44.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d42)
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r30 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r30, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.EmitStoreToStack(d44, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.EmitStoreToStack(d44, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl15)
			r31 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r31, 8)
			ctx.ProtectReg(r31)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r31}
			ctx.UnprotectReg(r31)
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r32 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r32, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			}
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d47.Imm.Int())}
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d47.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d47.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitSubInt64(d49.Reg, scm.RegR11)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else {
				ctx.W.EmitSubInt64(d49.Reg, d47.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			var d51 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d50.Imm.Int())))}
			} else if d50.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d46.Reg, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
			} else {
				{
					shiftSrc := d46.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d51.Loc == scm.LocReg && d46.Loc == scm.LocReg && d51.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d50)
			ctx.EmitMovToReg(r22, d51)
			ctx.W.EmitJmp(lbl13)
			ctx.FreeDesc(&d51)
			ctx.W.MarkLabel(lbl14)
			var d52 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r33, d39.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			}
			if d52.Loc == scm.LocReg && d39.Loc == scm.LocReg && d52.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d53 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			}
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d53.Imm.Int())}
			} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d53.Reg}
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(d52.Reg, scm.RegR11)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else {
				ctx.W.EmitAddInt64(d52.Reg, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			}
			if d55.Loc == scm.LocReg && d52.Loc == scm.LocReg && d55.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.FreeDesc(&d53)
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d55.Imm.Int() > 64)}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d55.Reg, 64)
				ctx.W.EmitSetcc(r35, scm.CcG)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
			}
			ctx.FreeDesc(&d55)
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d56.Loc == scm.LocImm {
				if d56.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.EmitStoreToStack(d44, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d56.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.EmitStoreToStack(d44, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d56)
			ctx.W.MarkLabel(lbl17)
			var d57 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r36 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r36, d39.Reg)
				ctx.W.EmitShrRegImm8(r36, 6)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			}
			if d57.Loc == scm.LocReg && d39.Loc == scm.LocReg && d57.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d57.Reg, scm.RegR11)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
			}
			if d58.Loc == scm.LocReg && d57.Loc == scm.LocReg && d58.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			r37 := ctx.AllocReg()
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r37, uint64(d58.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r37, d58.Reg)
				ctx.W.EmitShlRegImm8(r37, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r37, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r37, d40.Reg)
			}
			r38 := ctx.AllocRegExcept(r37)
			ctx.W.EmitMovRegMem(r38, r37, 0)
			ctx.FreeReg(r37)
			d59 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r38}
			ctx.FreeDesc(&d58)
			var d60 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d39.Reg, 63)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			}
			if d60.Loc == scm.LocReg && d39.Loc == scm.LocReg && d60.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			d61 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() - d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d61.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d60.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
				ctx.W.EmitSubInt64(d61.Reg, scm.RegR11)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else {
				ctx.W.EmitSubInt64(d61.Reg, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			var d63 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) >> uint64(d62.Imm.Int())))}
			} else if d62.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d59.Reg, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
			} else {
				{
					shiftSrc := d59.Reg
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
			if d63.Loc == scm.LocReg && d59.Loc == scm.LocReg && d63.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d62)
			var d64 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() | d63.Imm.Int())}
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
				ctx.W.EmitOrInt64(d44.Reg, scratch)
				ctx.FreeReg(scratch)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d44.Reg}
			} else {
				ctx.W.EmitOrInt64(d44.Reg, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d44.Reg}
			}
			if d64.Loc == scm.LocReg && d44.Loc == scm.LocReg && d64.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.EmitStoreToStack(d64, 8)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl13)
			ctx.W.ResolveFixups()
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
			if idxInt.Loc == scm.LocReg { ctx.UnprotectReg(idxInt.Reg) }
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r39 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r39, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			}
			var d68 scm.JITValueDesc
			if d65.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d65.Imm.Int() + d67.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
			} else if d65.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d65.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d67.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
				ctx.W.EmitAddInt64(d65.Reg, scm.RegR11)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			} else {
				ctx.W.EmitAddInt64(d65.Reg, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			}
			if d68.Loc == scm.LocReg && d65.Loc == scm.LocReg && d68.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.FreeDesc(&d67)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r40 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r40, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			if d70.Loc == scm.LocImm {
				if d70.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d70.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl21)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl21)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d70)
			ctx.W.MarkLabel(lbl11)
			if d33.Loc == scm.LocReg { ctx.ProtectReg(d33.Reg) }
			r41 := ctx.AllocReg()
			lbl22 := ctx.W.ReserveLabel()
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r42, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
			}
			var d74 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() * d72.Imm.Int())}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d33.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r43 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r43, d33.Reg)
				ctx.W.EmitImulInt64(r43, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
			}
			if d74.Loc == scm.LocReg && d33.Loc == scm.LocReg && d74.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			var d75 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r44 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r44, d74.Reg)
				ctx.W.EmitShrRegImm8(r44, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			r45 := ctx.AllocReg()
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r45, uint64(d75.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r45, d75.Reg)
				ctx.W.EmitShlRegImm8(r45, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r45, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r45, d40.Reg)
			}
			r46 := ctx.AllocRegExcept(r45)
			ctx.W.EmitMovRegMem(r46, r45, 0)
			ctx.FreeReg(r45)
			d76 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r47 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r47, d74.Reg)
				ctx.W.EmitAndRegImm32(r47, 63)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			}
			if d77.Loc == scm.LocReg && d74.Loc == scm.LocReg && d77.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d78 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d76.Imm.Int()) << uint64(d77.Imm.Int())))}
			} else if d77.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d76.Reg, uint8(d77.Imm.Int()))
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
			} else {
				{
					shiftSrc := d76.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d77.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d77.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d77.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d78.Loc == scm.LocReg && d76.Loc == scm.LocReg && d78.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			ctx.FreeDesc(&d77)
			var d79 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r48, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d79.Loc == scm.LocImm {
				if d79.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.EmitStoreToStack(d78, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.EmitStoreToStack(d78, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d79)
			ctx.W.MarkLabel(lbl24)
			r49 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r49, 16)
			ctx.ProtectReg(r49)
			d80 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r49}
			ctx.UnprotectReg(r49)
			var d81 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r50, thisptr.Reg, off)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
			}
			d83 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() - d81.Imm.Int())}
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d81.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d81.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
				ctx.W.EmitSubInt64(d83.Reg, scm.RegR11)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else {
				ctx.W.EmitSubInt64(d83.Reg, d81.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			}
			if d84.Loc == scm.LocReg && d83.Loc == scm.LocReg && d84.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			var d85 scm.JITValueDesc
			if d80.Loc == scm.LocImm && d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d80.Imm.Int()) >> uint64(d84.Imm.Int())))}
			} else if d84.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d80.Reg, uint8(d84.Imm.Int()))
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d80.Reg}
			} else {
				{
					shiftSrc := d80.Reg
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
			if d85.Loc == scm.LocReg && d80.Loc == scm.LocReg && d85.Reg == d80.Reg {
				ctx.TransferReg(d80.Reg)
				d80.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			ctx.FreeDesc(&d84)
			ctx.EmitMovToReg(r41, d85)
			ctx.W.EmitJmp(lbl22)
			ctx.FreeDesc(&d85)
			ctx.W.MarkLabel(lbl23)
			var d86 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r51 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r51, d74.Reg)
				ctx.W.EmitAndRegImm32(r51, 63)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			}
			if d86.Loc == scm.LocReg && d74.Loc == scm.LocReg && d86.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d87 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r52, thisptr.Reg, off)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
			}
			var d89 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d87.Imm.Int())}
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d87.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d87.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(d86.Reg, scm.RegR11)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else {
				ctx.W.EmitAddInt64(d86.Reg, d87.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			}
			if d89.Loc == scm.LocReg && d86.Loc == scm.LocReg && d89.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d87)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d89.Imm.Int() > 64)}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d89.Reg, 64)
				ctx.W.EmitSetcc(r53, scm.CcG)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
			}
			ctx.FreeDesc(&d89)
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d90.Loc == scm.LocImm {
				if d90.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
					ctx.EmitStoreToStack(d78, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d90.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
				ctx.EmitStoreToStack(d78, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl26)
			}
			ctx.FreeDesc(&d90)
			ctx.W.MarkLabel(lbl26)
			var d91 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r54 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r54, d74.Reg)
				ctx.W.EmitShrRegImm8(r54, 6)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			}
			if d91.Loc == scm.LocReg && d74.Loc == scm.LocReg && d91.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d91.Reg, scm.RegR11)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
			}
			if d92.Loc == scm.LocReg && d91.Loc == scm.LocReg && d92.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			r55 := ctx.AllocReg()
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r55, uint64(d92.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r55, d92.Reg)
				ctx.W.EmitShlRegImm8(r55, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r55, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r55, d40.Reg)
			}
			r56 := ctx.AllocRegExcept(r55)
			ctx.W.EmitMovRegMem(r56, r55, 0)
			ctx.FreeReg(r55)
			d93 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d74.Reg, 63)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			}
			if d94.Loc == scm.LocReg && d74.Loc == scm.LocReg && d94.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			d95 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d96 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d94.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() - d94.Imm.Int())}
			} else if d94.Loc == scm.LocImm && d94.Imm.Int() == 0 {
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d95.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d94.Reg)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d94.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.W.EmitSubInt64(d95.Reg, scm.RegR11)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else {
				ctx.W.EmitSubInt64(d95.Reg, d94.Reg)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			}
			if d96.Loc == scm.LocReg && d95.Loc == scm.LocReg && d96.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			var d97 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d96.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) >> uint64(d96.Imm.Int())))}
			} else if d96.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d93.Reg, uint8(d96.Imm.Int()))
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d93.Reg}
			} else {
				{
					shiftSrc := d93.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d96.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d96.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d96.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d97.Loc == scm.LocReg && d93.Loc == scm.LocReg && d97.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			ctx.FreeDesc(&d96)
			var d98 scm.JITValueDesc
			if d78.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() | d97.Imm.Int())}
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d97.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d97.Imm.Int()))
				ctx.W.EmitOrInt64(d78.Reg, scratch)
				ctx.FreeReg(scratch)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d78.Reg}
			} else {
				ctx.W.EmitOrInt64(d78.Reg, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d78.Reg}
			}
			if d98.Loc == scm.LocReg && d78.Loc == scm.LocReg && d98.Reg == d78.Reg {
				ctx.TransferReg(d78.Reg)
				d78.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d97)
			ctx.EmitStoreToStack(d98, 16)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
			d99 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			if d33.Loc == scm.LocReg { ctx.UnprotectReg(d33.Reg) }
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r57 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r57, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			}
			var d102 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() + d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d101.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(d99.Reg, scm.RegR11)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			} else {
				ctx.W.EmitAddInt64(d99.Reg, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			}
			if d102.Loc == scm.LocReg && d99.Loc == scm.LocReg && d102.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			ctx.FreeDesc(&d101)
			if d33.Loc == scm.LocReg { ctx.ProtectReg(d33.Reg) }
			r58 := ctx.AllocReg()
			lbl28 := ctx.W.ReserveLabel()
			var d104 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r59, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			}
			var d106 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() * d104.Imm.Int())}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d33.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r60 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r60, d33.Reg)
				ctx.W.EmitImulInt64(r60, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			}
			if d106.Loc == scm.LocReg && d33.Loc == scm.LocReg && d106.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r61, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
			}
			var d108 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() / 64)}
			} else {
				r62 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r62, d106.Reg)
				ctx.W.EmitShrRegImm8(r62, 6)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
			}
			if d108.Loc == scm.LocReg && d106.Loc == scm.LocReg && d108.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			r63 := ctx.AllocReg()
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r63, uint64(d108.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r63, d108.Reg)
				ctx.W.EmitShlRegImm8(r63, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r63, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r63, d107.Reg)
			}
			r64 := ctx.AllocRegExcept(r63)
			ctx.W.EmitMovRegMem(r64, r63, 0)
			ctx.FreeReg(r63)
			d109 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r65 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r65, d106.Reg)
				ctx.W.EmitAndRegImm32(r65, 63)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			}
			if d110.Loc == scm.LocReg && d106.Loc == scm.LocReg && d110.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d111 scm.JITValueDesc
			if d109.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d109.Imm.Int()) << uint64(d110.Imm.Int())))}
			} else if d110.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d109.Reg, uint8(d110.Imm.Int()))
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d109.Reg}
			} else {
				{
					shiftSrc := d109.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d110.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d110.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d110.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d111.Loc == scm.LocReg && d109.Loc == scm.LocReg && d111.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			ctx.FreeDesc(&d110)
			var d112 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r66, thisptr.Reg, off)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d112.Loc == scm.LocImm {
				if d112.Imm.Bool() {
					ctx.W.EmitJmp(lbl29)
				} else {
					ctx.EmitStoreToStack(d111, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d112.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
				ctx.EmitStoreToStack(d111, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d112)
			ctx.W.MarkLabel(lbl30)
			r67 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r67, 24)
			ctx.ProtectReg(r67)
			d113 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r67}
			ctx.UnprotectReg(r67)
			var d114 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			}
			d116 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() - d114.Imm.Int())}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d114.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d114.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
				ctx.W.EmitSubInt64(d116.Reg, scm.RegR11)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else {
				ctx.W.EmitSubInt64(d116.Reg, d114.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			var d118 scm.JITValueDesc
			if d113.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d113.Imm.Int()) >> uint64(d117.Imm.Int())))}
			} else if d117.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d113.Reg, uint8(d117.Imm.Int()))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d113.Reg}
			} else {
				{
					shiftSrc := d113.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d118.Loc == scm.LocReg && d113.Loc == scm.LocReg && d118.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			ctx.FreeDesc(&d117)
			ctx.EmitMovToReg(r58, d118)
			ctx.W.EmitJmp(lbl28)
			ctx.FreeDesc(&d118)
			ctx.W.MarkLabel(lbl29)
			var d119 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r69, d106.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
			}
			if d119.Loc == scm.LocReg && d106.Loc == scm.LocReg && d119.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			}
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d120.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
				ctx.W.EmitAddInt64(d119.Reg, scm.RegR11)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else {
				ctx.W.EmitAddInt64(d119.Reg, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d120)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d122.Imm.Int() > 64)}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d122.Reg, 64)
				ctx.W.EmitSetcc(r71, scm.CcG)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r71}
			}
			ctx.FreeDesc(&d122)
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d123.Loc == scm.LocImm {
				if d123.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.EmitStoreToStack(d111, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d123.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl33)
				ctx.EmitStoreToStack(d111, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d123)
			ctx.W.MarkLabel(lbl32)
			var d124 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() / 64)}
			} else {
				r72 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r72, d106.Reg)
				ctx.W.EmitShrRegImm8(r72, 6)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
			}
			if d124.Loc == scm.LocReg && d106.Loc == scm.LocReg && d124.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d124.Reg, scm.RegR11)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
			}
			if d125.Loc == scm.LocReg && d124.Loc == scm.LocReg && d125.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			r73 := ctx.AllocReg()
			if d125.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r73, uint64(d125.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r73, d125.Reg)
				ctx.W.EmitShlRegImm8(r73, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r73, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r73, d107.Reg)
			}
			r74 := ctx.AllocRegExcept(r73)
			ctx.W.EmitMovRegMem(r74, r73, 0)
			ctx.FreeReg(r73)
			d126 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r74}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d106.Reg, 63)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d106.Reg}
			}
			if d127.Loc == scm.LocReg && d106.Loc == scm.LocReg && d127.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			d128 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d129 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() - d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d127.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.W.EmitSubInt64(d128.Reg, scm.RegR11)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			} else {
				ctx.W.EmitSubInt64(d128.Reg, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			}
			if d129.Loc == scm.LocReg && d128.Loc == scm.LocReg && d129.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			var d130 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d126.Imm.Int()) >> uint64(d129.Imm.Int())))}
			} else if d129.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d126.Reg, uint8(d129.Imm.Int()))
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
			} else {
				{
					shiftSrc := d126.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d130.Loc == scm.LocReg && d126.Loc == scm.LocReg && d130.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d129)
			var d131 scm.JITValueDesc
			if d111.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d111.Imm.Int() | d130.Imm.Int())}
			} else if d111.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d111.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
				ctx.W.EmitOrInt64(d111.Reg, scratch)
				ctx.FreeReg(scratch)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d111.Reg}
			} else {
				ctx.W.EmitOrInt64(d111.Reg, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d111.Reg}
			}
			if d131.Loc == scm.LocReg && d111.Loc == scm.LocReg && d131.Reg == d111.Reg {
				ctx.TransferReg(d111.Reg)
				d111.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d130)
			ctx.EmitStoreToStack(d131, 24)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			ctx.W.ResolveFixups()
			d132 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
			if d33.Loc == scm.LocReg { ctx.UnprotectReg(d33.Reg) }
			var d134 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r75 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r75, thisptr.Reg, off)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			}
			var d135 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() + d134.Imm.Int())}
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d134.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d134.Imm.Int()))
				ctx.W.EmitAddInt64(d132.Reg, scm.RegR11)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			} else {
				ctx.W.EmitAddInt64(d132.Reg, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			}
			if d135.Loc == scm.LocReg && d132.Loc == scm.LocReg && d135.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.FreeDesc(&d134)
			var d137 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r76 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r76, d102.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d102.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r77 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r77, d102.Reg)
				ctx.W.EmitAddInt64(r77, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			}
			if d137.Loc == scm.LocReg && d102.Loc == scm.LocReg && d137.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r78 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r78, fieldAddr)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r78}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r79 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r79, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			}
			var d141 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() - d102.Imm.Int())}
			} else {
				r80 := ctx.AllocReg()
				if d137.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r80, uint64(d137.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r80, d137.Reg)
				}
				if d102.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
					ctx.W.EmitSubInt64(r80, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitSubInt64(r80, d102.Reg)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
			}
			var d142 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + d102.Imm.Int())}
			} else {
				r81 := ctx.AllocReg()
				if d139.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r81, uint64(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r81, d139.Reg)
				}
				if d102.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
					ctx.W.EmitAddInt64(r81, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitAddInt64(r81, d102.Reg)
				}
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			}
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewInt(d142.Imm.Int())}
				_ = d141
			} else {
				r82 := ctx.AllocReg()
				r83 := ctx.AllocReg()
				if d142.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r82, uint64(d142.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r82, d142.Reg)
					ctx.FreeReg(d142.Reg)
				}
				if d141.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r83, uint64(d141.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r83, d141.Reg)
					ctx.FreeReg(d141.Reg)
				}
				d143 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r82, Reg2: r83}
			}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d137)
			d144 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d143}, 2)
			ctx.EmitMovPairToResult(&d144, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r84 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r84, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r84}
			}
			var d147 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d145.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d33.Imm.Int() == d145.Imm.Int())}
			} else if d145.Loc == scm.LocImm {
				r85 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d33.Reg, int32(d145.Imm.Int()))
				ctx.W.EmitSetcc(r85, scm.CcE)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r85}
			} else if d33.Loc == scm.LocImm {
				r86 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d145.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r86, scm.CcE)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r86}
			} else {
				r87 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d33.Reg, d145.Reg)
				ctx.W.EmitSetcc(r87, scm.CcE)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r87}
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d145)
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d147.Loc == scm.LocImm {
				if d147.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d147.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d147)
			ctx.W.MarkLabel(lbl20)
			r88 := ctx.AllocReg()
			lbl36 := ctx.W.ReserveLabel()
			var d149 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r89, thisptr.Reg, off)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
			}
			var d151 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d149.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d149.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d149.Imm.Int()))
				ctx.W.EmitImulInt64(idxInt.Reg, scm.RegR11)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			} else {
				ctx.W.EmitImulInt64(idxInt.Reg, d149.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			}
			if d151.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d151.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&idxInt)
			ctx.FreeDesc(&d149)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() / 64)}
			} else {
				r90 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r90, d151.Reg)
				ctx.W.EmitShrRegImm8(r90, 6)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			}
			if d152.Loc == scm.LocReg && d151.Loc == scm.LocReg && d152.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			r91 := ctx.AllocReg()
			if d152.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r91, uint64(d152.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r91, d152.Reg)
				ctx.W.EmitShlRegImm8(r91, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r91, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r91, d107.Reg)
			}
			r92 := ctx.AllocRegExcept(r91)
			ctx.W.EmitMovRegMem(r92, r91, 0)
			ctx.FreeReg(r91)
			d153 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			ctx.FreeDesc(&d152)
			var d154 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() % 64)}
			} else {
				r93 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r93, d151.Reg)
				ctx.W.EmitAndRegImm32(r93, 63)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			}
			if d154.Loc == scm.LocReg && d151.Loc == scm.LocReg && d154.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			var d155 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d153.Imm.Int()) << uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d153.Reg, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d153.Reg}
			} else {
				{
					shiftSrc := d153.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d155.Loc == scm.LocReg && d153.Loc == scm.LocReg && d155.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d153)
			ctx.FreeDesc(&d154)
			var d156 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r94, thisptr.Reg, off)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r94}
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d156.Loc == scm.LocImm {
				if d156.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
					ctx.EmitStoreToStack(d155, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d156.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.EmitStoreToStack(d155, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d156)
			ctx.W.MarkLabel(lbl38)
			r95 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r95, 32)
			ctx.ProtectReg(r95)
			d157 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r95}
			ctx.UnprotectReg(r95)
			var d158 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r96, thisptr.Reg, off)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
			}
			d160 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d161 scm.JITValueDesc
			if d160.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d160.Imm.Int() - d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d160.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d158.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d158.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
				ctx.W.EmitSubInt64(d160.Reg, scm.RegR11)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			} else {
				ctx.W.EmitSubInt64(d160.Reg, d158.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			}
			if d161.Loc == scm.LocReg && d160.Loc == scm.LocReg && d161.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			var d162 scm.JITValueDesc
			if d157.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d157.Imm.Int()) >> uint64(d161.Imm.Int())))}
			} else if d161.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d157.Reg, uint8(d161.Imm.Int()))
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
			} else {
				{
					shiftSrc := d157.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d161.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d161.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d161.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d162.Loc == scm.LocReg && d157.Loc == scm.LocReg && d162.Reg == d157.Reg {
				ctx.TransferReg(d157.Reg)
				d157.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d157)
			ctx.FreeDesc(&d161)
			ctx.EmitMovToReg(r88, d162)
			ctx.W.EmitJmp(lbl36)
			ctx.FreeDesc(&d162)
			ctx.W.MarkLabel(lbl37)
			var d163 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() % 64)}
			} else {
				r97 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r97, d151.Reg)
				ctx.W.EmitAndRegImm32(r97, 63)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			}
			if d163.Loc == scm.LocReg && d151.Loc == scm.LocReg && d163.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			var d164 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r98, thisptr.Reg, off)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			}
			var d166 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() + d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d163.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d164.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
				ctx.W.EmitAddInt64(d163.Reg, scm.RegR11)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
			} else {
				ctx.W.EmitAddInt64(d163.Reg, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
			}
			if d166.Loc == scm.LocReg && d163.Loc == scm.LocReg && d166.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.FreeDesc(&d164)
			var d167 scm.JITValueDesc
			if d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d166.Imm.Int() > 64)}
			} else {
				r99 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d166.Reg, 64)
				ctx.W.EmitSetcc(r99, scm.CcG)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r99}
			}
			ctx.FreeDesc(&d166)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d167.Loc == scm.LocImm {
				if d167.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
					ctx.EmitStoreToStack(d155, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d167.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
				ctx.EmitStoreToStack(d155, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d167)
			ctx.W.MarkLabel(lbl40)
			var d168 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() / 64)}
			} else {
				r100 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r100, d151.Reg)
				ctx.W.EmitShrRegImm8(r100, 6)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			}
			if d168.Loc == scm.LocReg && d151.Loc == scm.LocReg && d168.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d168.Reg, scm.RegR11)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d168.Reg}
			}
			if d169.Loc == scm.LocReg && d168.Loc == scm.LocReg && d169.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			r101 := ctx.AllocReg()
			if d169.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r101, uint64(d169.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r101, d169.Reg)
				ctx.W.EmitShlRegImm8(r101, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r101, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r101, d107.Reg)
			}
			r102 := ctx.AllocRegExcept(r101)
			ctx.W.EmitMovRegMem(r102, r101, 0)
			ctx.FreeReg(r101)
			d170 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r102}
			ctx.FreeDesc(&d169)
			var d171 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d151.Reg, 63)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d151.Reg}
			}
			if d171.Loc == scm.LocReg && d151.Loc == scm.LocReg && d171.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			d172 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d173 scm.JITValueDesc
			if d172.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d172.Imm.Int() - d171.Imm.Int())}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d172.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d171.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
				ctx.W.EmitSubInt64(d172.Reg, scm.RegR11)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
			} else {
				ctx.W.EmitSubInt64(d172.Reg, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
			}
			if d173.Loc == scm.LocReg && d172.Loc == scm.LocReg && d173.Reg == d172.Reg {
				ctx.TransferReg(d172.Reg)
				d172.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			var d174 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d170.Imm.Int()) >> uint64(d173.Imm.Int())))}
			} else if d173.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d170.Reg, uint8(d173.Imm.Int()))
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
			} else {
				{
					shiftSrc := d170.Reg
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
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d174.Loc == scm.LocReg && d170.Loc == scm.LocReg && d174.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() | d174.Imm.Int())}
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitOrInt64(d155.Reg, scratch)
				ctx.FreeReg(scratch)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
			} else {
				ctx.W.EmitOrInt64(d155.Reg, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
			}
			if d175.Loc == scm.LocReg && d155.Loc == scm.LocReg && d175.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.EmitStoreToStack(d175, 32)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d176 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			ctx.FreeDesc(&idxInt)
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r103, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			}
			var d179 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() + d178.Imm.Int())}
			} else if d178.Loc == scm.LocImm && d178.Imm.Int() == 0 {
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d178.Reg}
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d178.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
				ctx.W.EmitAddInt64(d176.Reg, scm.RegR11)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			} else {
				ctx.W.EmitAddInt64(d176.Reg, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			}
			if d179.Loc == scm.LocReg && d176.Loc == scm.LocReg && d179.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d178)
			var d182 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() + d179.Imm.Int())}
			} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
				r104 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r104, d68.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d179.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d179.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d68.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r105 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r105, d68.Reg)
				ctx.W.EmitAddInt64(r105, d179.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			}
			if d182.Loc == scm.LocReg && d68.Loc == scm.LocReg && d182.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			var d185 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() - d68.Imm.Int())}
			} else {
				r106 := ctx.AllocReg()
				if d182.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r106, uint64(d182.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r106, d182.Reg)
				}
				if d68.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
					ctx.W.EmitSubInt64(r106, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitSubInt64(r106, d68.Reg)
				}
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			}
			var d186 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + d68.Imm.Int())}
			} else {
				r107 := ctx.AllocReg()
				if d139.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r107, uint64(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r107, d139.Reg)
				}
				if d68.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
					ctx.W.EmitAddInt64(r107, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitAddInt64(r107, d68.Reg)
				}
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
			}
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewInt(d186.Imm.Int())}
				_ = d185
			} else {
				r108 := ctx.AllocReg()
				r109 := ctx.AllocReg()
				if d186.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r108, uint64(d186.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r108, d186.Reg)
					ctx.FreeReg(d186.Reg)
				}
				if d185.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r109, uint64(d185.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r109, d185.Reg)
					ctx.FreeReg(d185.Reg)
				}
				d187 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r108, Reg2: r109}
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d182)
			d188 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d187}, 2)
			ctx.EmitMovPairToResult(&d188, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			var d189 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r110, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
			}
			var d190 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d68.Imm.Int() == d189.Imm.Int())}
			} else if d189.Loc == scm.LocImm {
				r111 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d68.Reg, int32(d189.Imm.Int()))
				ctx.W.EmitSetcc(r111, scm.CcE)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r111}
			} else if d68.Loc == scm.LocImm {
				r112 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d189.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r112, scm.CcE)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r112}
			} else {
				r113 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d68.Reg, d189.Reg)
				ctx.W.EmitSetcc(r113, scm.CcE)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r113}
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d189)
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d190.Loc == scm.LocImm {
				if d190.Imm.Bool() {
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d190.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d190)
			ctx.W.MarkLabel(lbl34)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl42)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r2, int32(40))
			ctx.W.EmitAddRSP32(int32(40))
			return result
}

func (s *StorageString) prepare() {
	// set up scan
	s.starts.prepare()
	s.lens.prepare()
	s.values.prepare()
	s.reverseMap = make(map[string][3]uint)
	s.prefixstat = make(map[string]int)
}
func (s *StorageString) scan(i uint32, value scm.Scmer) {
	// storage is so simple, dont need scan
	var v string
	if value.IsNil() {
		if s.nodict {
			s.starts.scan(i, scm.NewNil())
		} else {
			s.values.scan(i, scm.NewNil())
		}
		return
	}
	v = scm.String(value)

	// check if we have common prefix (but ignore duplicates because they are compressed by dictionary)
	if s.laststr != v {
		commonlen := 0
		for commonlen < len(s.laststr) && commonlen < len(v) && s.laststr[commonlen] == v[commonlen] {
			s.prefixstat[v[0:commonlen]] = s.prefixstat[v[0:commonlen]] + 1
			commonlen++
		}
		if v != "" {
			s.laststr = v
		}
	}

	// check for dictionary
	if i == 100 && len(s.reverseMap) > 99 {
		// nearly no repetition in the first 100 items: save the time to build reversemap
		s.nodict = true
		s.reverseMap = nil
		s.sb.Reset()
		if s.values.hasNull {
			s.starts.scan(0, scm.NewNil()) // learn NULL
		}
		// build will fill our stringbuffer
	}
	s.allsize = s.allsize + len(v)
	if s.nodict {
		s.starts.scan(i, scm.NewInt(int64(s.allsize)))
		s.lens.scan(i, scm.NewInt(int64(len(v))))
	} else {
		start, ok := s.reverseMap[v]
		if ok {
			// reuse of string
		} else {
			// learn
			start[0] = s.count
			start[1] = uint(s.sb.Len())
			start[2] = uint(len(v))
			s.sb.WriteString(v)
			s.starts.scan(uint32(start[0]), scm.NewInt(int64(start[1])))
			s.lens.scan(uint32(start[0]), scm.NewInt(int64(start[2])))
			s.reverseMap[v] = start
			s.count = s.count + 1
		}
		s.values.scan(i, scm.NewInt(int64(start[0])))
	}
}
func (s *StorageString) init(i uint32) {
	s.prefixstat = nil // free memory
	if s.nodict {
		// do not init values, sb andsoon
		s.starts.init(i)
		s.lens.init(i)
	} else {
		// allocate
		s.dictionary = s.sb.String() // extract one big slice with all strings (no extra memory structure)
		s.sb.Reset()                 // free the memory
		// prefixed strings are not accounted with that, but maybe this could be checked later??
		s.values.init(i)
		// take over dictionary
		s.starts.init(uint32(s.count))
		s.lens.init(uint32(s.count))
		for _, start := range s.reverseMap {
			// we read the value from dictionary, so we can free up all the single-strings
			s.starts.build(uint32(start[0]), scm.NewInt(int64(start[1])))
			s.lens.build(uint32(start[0]), scm.NewInt(int64(start[2])))
		}
	}
}
func (s *StorageString) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		if s.nodict {
			s.starts.build(i, scm.NewNil())
		} else {
			s.values.build(i, scm.NewNil())
		}
		return
	}
	v := scm.String(value)
	if s.nodict {
		s.starts.build(i, scm.NewInt(int64(s.sb.Len())))
		s.lens.build(i, scm.NewInt(int64(len(v))))
		s.sb.WriteString(v)
	} else {
		start := s.reverseMap[v]
		// write start+end into sub storage maps
		s.values.build(i, scm.NewInt(int64(start[0])))
	}
}
func (s *StorageString) finish() {
	if s.nodict {
		s.dictionary = s.sb.String()
		s.sb.Reset()
	} else {
		s.reverseMap = nil
		s.values.finish()
	}
	s.starts.finish()
	s.lens.finish()
}
func (s *StorageString) proposeCompression(i uint32) ColumnStorage {
	// build prefix map (maybe prefix trees later?)
	/* TODO: reactivate as soon as StoragePrefix has a proper implementation for Serialize/Deserialize
	mostprefixscore := 0
	mostprefix := ""
	for k, v := range s.prefixstat {
		if len(k) * v > mostprefixscore {
			mostprefix = k
			mostprefixscore = len(k) * v // cost saving of prefix = len(prefix) * occurance
		}
	}
	if uint(mostprefixscore) > i / 8 + 100 {
		// built a 1-bit prefix (TODO: maybe later more)
		stor := new(StoragePrefix)
		stor.prefixdictionary = []string{"", mostprefix}
		return stor
	}

	Prefix tree index:
	rootnodes = []
	for each s := range string {
		foreach k, v := rootnodes {
			pfx := commonPrefix(s, k)
			if pfx == k {
				// insert into subtree
				v.insert(s[len(pfx):], value)
			} else {
				// split the tree
				delete(rootnodes, k)
				rootnodes[pfx] = {k[len(pfx):]: v, s[len(pfx):]: value}
			}
		}
		rootnodes[s] = value
		cont:
	}
	implementation: byte stream of id, len, byte[len] + array:id->*treenode; encode bigger ids similar to utf-8: for { result = result < 7 | (byte & 127) if byte & 128 == 0 {break}}

	prefix compression: multi-stage storage
	type prefixTree struct { text string, children []prefixTree }
	type prefixTreeStorage struct { childIndexes ColumnStorage, recordIdTranslation ColumnStorage, children []prefixTreeStorage } -> Seq-compression should be very effective

	*/
	// dont't propose another pass
	return nil
}
