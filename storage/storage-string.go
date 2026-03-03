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
	nodict     bool `jit:"immutable-after-finish"` // disable values array

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
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
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
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
				r0 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r0, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r0}
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
			ctx.FreeDesc(&d0)
			ctx.W.MarkLabel(lbl2)
			r1 := idxInt.Loc == scm.LocReg
			r2 := idxInt.Reg
			if r1 { ctx.ProtectReg(r2) }
			r3 := ctx.W.EmitSubRSP32Fixup()
			r4 := ctx.AllocReg()
			lbl4 := ctx.W.ReserveLabel()
			var d1 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r5, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r5, 32)
				ctx.W.EmitShrRegImm8(r5, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
			}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r6, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r6}
			}
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r7, d2.Reg)
				ctx.W.EmitShlRegImm8(r7, 56)
				ctx.W.EmitShrRegImm8(r7, 56)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
			}
			ctx.FreeDesc(&d2)
			var d4 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() * d3.Imm.Int())}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d3.Loc == scm.LocImm {
				if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d1.Reg, int32(d3.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
				ctx.W.EmitImulInt64(d1.Reg, scm.RegR11)
				}
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1.Reg}
			} else {
				ctx.W.EmitImulInt64(d1.Reg, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1.Reg}
			}
			if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d3)
			var d5 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r8, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
			}
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				ctx.W.EmitShrRegImm8(r9, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			r10 := ctx.AllocReg()
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r10, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r10, d6.Reg)
				ctx.W.EmitShlRegImm8(r10, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r10, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r10, d5.Reg)
			}
			r11 := ctx.AllocRegExcept(r10)
			ctx.W.EmitMovRegMem(r11, r10, 0)
			ctx.FreeReg(r10)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
			ctx.FreeDesc(&d6)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r12 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r12, d4.Reg)
				ctx.W.EmitAndRegImm32(r12, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
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
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r13, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
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
			r14 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r14, 0)
			ctx.ProtectReg(r14)
			d11 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r14}
			ctx.UnprotectReg(r14)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			}
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d12.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			ctx.FreeDesc(&d12)
			d14 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d13.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else if d14.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d13.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d13.Loc == scm.LocImm {
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d14.Reg, int32(d13.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitSubInt64(d14.Reg, scm.RegR11)
				}
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else {
				ctx.W.EmitSubInt64(d14.Reg, d13.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			}
			if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
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
			ctx.EmitMovToReg(r4, d16)
			ctx.W.EmitJmp(lbl4)
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl5)
			var d17 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r17 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r17, d4.Reg)
				ctx.W.EmitAndRegImm32(r17, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
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
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			}
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d18.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			}
			ctx.FreeDesc(&d18)
			var d20 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d19.Reg}
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d19.Loc == scm.LocImm {
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d17.Reg, int32(d19.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
				ctx.W.EmitAddInt64(d17.Reg, scm.RegR11)
				}
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else {
				ctx.W.EmitAddInt64(d17.Reg, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			}
			if d20.Loc == scm.LocReg && d17.Loc == scm.LocReg && d20.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d19)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d20.Imm.Int()) > uint64(64))}
			} else {
				r20 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d20.Reg, 64)
				ctx.W.EmitSetcc(r20, scm.CcA)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r20}
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
				r21 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r21, d4.Reg)
				ctx.W.EmitShrRegImm8(r21, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d22.Reg, int32(1))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			r22 := ctx.AllocReg()
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r22, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r22, d23.Reg)
				ctx.W.EmitShlRegImm8(r22, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r22, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r22, d5.Reg)
			}
			r23 := ctx.AllocRegExcept(r22)
			ctx.W.EmitMovRegMem(r23, r22, 0)
			ctx.FreeReg(r22)
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
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
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d26.Reg, int32(d25.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(d26.Reg, scm.RegR11)
				}
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
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				r24 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r24, d9.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d28.Loc == scm.LocImm {
				r25 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r25, d9.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r25, int32(d28.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
					ctx.W.EmitOrInt64(r25, scratch)
					ctx.FreeReg(scratch)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			} else {
				r26 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r26, d9.Reg)
				ctx.W.EmitOrInt64(r26, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
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
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			if r1 { ctx.UnprotectReg(r2) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d30.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			}
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r28, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			}
			var d33 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d31.Reg, int32(d32.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.W.EmitAddInt64(d31.Reg, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			} else {
				ctx.W.EmitAddInt64(d31.Reg, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			}
			if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.FreeDesc(&d32)
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d33.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r29, d33.Reg)
				ctx.W.EmitShlRegImm8(r29, 32)
				ctx.W.EmitShrRegImm8(r29, 32)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
			}
			ctx.FreeDesc(&d33)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r30 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r30, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
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
			r31 := idxInt.Loc == scm.LocReg
			r32 := idxInt.Reg
			if r31 { ctx.ProtectReg(r32) }
			r33 := ctx.AllocReg()
			lbl13 := ctx.W.ReserveLabel()
			var d36 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r34, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r34, 32)
				ctx.W.EmitShrRegImm8(r34, 32)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			}
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r35, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			}
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d37.Imm.Int()))))}
			} else {
				r36 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r36, d37.Reg)
				ctx.W.EmitShlRegImm8(r36, 56)
				ctx.W.EmitShrRegImm8(r36, 56)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
			}
			ctx.FreeDesc(&d37)
			var d39 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() * d38.Imm.Int())}
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d38.Loc == scm.LocImm {
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d36.Reg, int32(d38.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
				ctx.W.EmitImulInt64(d36.Reg, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else {
				ctx.W.EmitImulInt64(d36.Reg, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			}
			if d39.Loc == scm.LocReg && d36.Loc == scm.LocReg && d39.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				r37 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r37, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r37}
			}
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r38 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r38, d39.Reg)
				ctx.W.EmitShrRegImm8(r38, 6)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			r39 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r39, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r39, d41.Reg)
				ctx.W.EmitShlRegImm8(r39, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r39, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r39, d40.Reg)
			}
			r40 := ctx.AllocRegExcept(r39)
			ctx.W.EmitMovRegMem(r40, r39, 0)
			ctx.FreeReg(r39)
			d42 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r41 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r41, d39.Reg)
				ctx.W.EmitAndRegImm32(r41, 63)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
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
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r42, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
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
			r43 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r43, 8)
			ctx.ProtectReg(r43)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r43}
			ctx.UnprotectReg(r43)
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r44, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
			}
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d47.Imm.Int()))))}
			} else {
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r45, d47.Reg)
				ctx.W.EmitShlRegImm8(r45, 56)
				ctx.W.EmitShrRegImm8(r45, 56)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
			}
			ctx.FreeDesc(&d47)
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d48.Loc == scm.LocImm {
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d49.Reg, int32(d48.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitSubInt64(d49.Reg, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else {
				ctx.W.EmitSubInt64(d49.Reg, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
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
			ctx.EmitMovToReg(r33, d51)
			ctx.W.EmitJmp(lbl13)
			ctx.FreeDesc(&d51)
			ctx.W.MarkLabel(lbl14)
			var d52 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r46 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r46, d39.Reg)
				ctx.W.EmitAndRegImm32(r46, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
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
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
			}
			var d54 scm.JITValueDesc
			if d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d53.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d53.Reg)
				ctx.W.EmitShlRegImm8(r48, 56)
				ctx.W.EmitShrRegImm8(r48, 56)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
			}
			ctx.FreeDesc(&d53)
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d54.Imm.Int())}
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d54.Loc == scm.LocImm {
				if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d52.Reg, int32(d54.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
				ctx.W.EmitAddInt64(d52.Reg, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else {
				ctx.W.EmitAddInt64(d52.Reg, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			}
			if d55.Loc == scm.LocReg && d52.Loc == scm.LocReg && d55.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.FreeDesc(&d54)
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d55.Imm.Int()) > uint64(64))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d55.Reg, 64)
				ctx.W.EmitSetcc(r49, scm.CcA)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
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
				r50 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r50, d39.Reg)
				ctx.W.EmitShrRegImm8(r50, 6)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			}
			if d57.Loc == scm.LocReg && d39.Loc == scm.LocReg && d57.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d57.Reg, int32(1))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
			}
			if d58.Loc == scm.LocReg && d57.Loc == scm.LocReg && d58.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			r51 := ctx.AllocReg()
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r51, uint64(d58.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r51, d58.Reg)
				ctx.W.EmitShlRegImm8(r51, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r51, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r51, d40.Reg)
			}
			r52 := ctx.AllocRegExcept(r51)
			ctx.W.EmitMovRegMem(r52, r51, 0)
			ctx.FreeReg(r51)
			d59 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
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
				if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d61.Reg, int32(d60.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
				ctx.W.EmitSubInt64(d61.Reg, scm.RegR11)
				}
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
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d63.Reg}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r53 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r53, d44.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d63.Loc == scm.LocImm {
				r54 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r54, d44.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r54, int32(d63.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
					ctx.W.EmitOrInt64(r54, scratch)
					ctx.FreeReg(scratch)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
			} else {
				r55 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r55, d44.Reg)
				ctx.W.EmitOrInt64(r55, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
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
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			if r31 { ctx.UnprotectReg(r32) }
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d65.Imm.Int()))))}
			} else {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r56, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
			}
			ctx.FreeDesc(&d65)
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r57 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r57, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			}
			var d68 scm.JITValueDesc
			if d66.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + d67.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d66.Reg}
			} else if d66.Loc == scm.LocImm && d66.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
			} else if d66.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d66.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d67.Loc == scm.LocImm {
				if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d66.Reg, int32(d67.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
				ctx.W.EmitAddInt64(d66.Reg, scm.RegR11)
				}
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d66.Reg}
			} else {
				ctx.W.EmitAddInt64(d66.Reg, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d66.Reg}
			}
			if d68.Loc == scm.LocReg && d66.Loc == scm.LocReg && d68.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d66)
			ctx.FreeDesc(&d67)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d68.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r58, d68.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
			}
			ctx.FreeDesc(&d68)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r59, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
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
			r60 := d34.Loc == scm.LocReg
			r61 := d34.Reg
			if r60 { ctx.ProtectReg(r61) }
			r62 := ctx.AllocReg()
			lbl22 := ctx.W.ReserveLabel()
			var d71 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d34.Reg)
				ctx.W.EmitShlRegImm8(r63, 32)
				ctx.W.EmitShrRegImm8(r63, 32)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			}
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
			}
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d72.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
			}
			ctx.FreeDesc(&d72)
			var d74 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d71.Imm.Int() * d73.Imm.Int())}
			} else if d71.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d71.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d73.Loc == scm.LocImm {
				if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d71.Reg, int32(d73.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
				ctx.W.EmitImulInt64(d71.Reg, scm.RegR11)
				}
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			} else {
				ctx.W.EmitImulInt64(d71.Reg, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
			}
			if d74.Loc == scm.LocReg && d71.Loc == scm.LocReg && d74.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.FreeDesc(&d73)
			var d75 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r66, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			}
			var d76 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r67 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r67, d74.Reg)
				ctx.W.EmitShrRegImm8(r67, 6)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			}
			if d76.Loc == scm.LocReg && d74.Loc == scm.LocReg && d76.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			r68 := ctx.AllocReg()
			if d76.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r68, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r68, d76.Reg)
				ctx.W.EmitShlRegImm8(r68, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r68, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r68, d75.Reg)
			}
			r69 := ctx.AllocRegExcept(r68)
			ctx.W.EmitMovRegMem(r69, r68, 0)
			ctx.FreeReg(r68)
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r69}
			ctx.FreeDesc(&d76)
			var d78 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r70 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r70, d74.Reg)
				ctx.W.EmitAndRegImm32(r70, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			}
			if d78.Loc == scm.LocReg && d74.Loc == scm.LocReg && d78.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d79 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) << uint64(d78.Imm.Int())))}
			} else if d78.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d77.Reg, uint8(d78.Imm.Int()))
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
			} else {
				{
					shiftSrc := d77.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d78.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d78.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d78.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d79.Loc == scm.LocReg && d77.Loc == scm.LocReg && d79.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d78)
			var d80 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r71, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r71}
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
			ctx.EmitStoreToStack(d79, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d80.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
			ctx.EmitStoreToStack(d79, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d80)
			ctx.W.MarkLabel(lbl24)
			r72 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r72, 16)
			ctx.ProtectReg(r72)
			d81 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r72}
			ctx.UnprotectReg(r72)
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r73, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r73}
			}
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d82.Imm.Int()))))}
			} else {
				r74 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r74, d82.Reg)
				ctx.W.EmitShlRegImm8(r74, 56)
				ctx.W.EmitShrRegImm8(r74, 56)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			}
			ctx.FreeDesc(&d82)
			d84 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() - d83.Imm.Int())}
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d84.Reg}
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d83.Loc == scm.LocImm {
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d84.Reg, int32(d83.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
				ctx.W.EmitSubInt64(d84.Reg, scm.RegR11)
				}
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d84.Reg}
			} else {
				ctx.W.EmitSubInt64(d84.Reg, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d84.Reg}
			}
			if d85.Loc == scm.LocReg && d84.Loc == scm.LocReg && d85.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d83)
			var d86 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d81.Imm.Int()) >> uint64(d85.Imm.Int())))}
			} else if d85.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d81.Reg, uint8(d85.Imm.Int()))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
			} else {
				{
					shiftSrc := d81.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d86.Loc == scm.LocReg && d81.Loc == scm.LocReg && d86.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d85)
			ctx.EmitMovToReg(r62, d86)
			ctx.W.EmitJmp(lbl22)
			ctx.FreeDesc(&d86)
			ctx.W.MarkLabel(lbl23)
			var d87 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r75 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r75, d74.Reg)
				ctx.W.EmitAndRegImm32(r75, 63)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			}
			if d87.Loc == scm.LocReg && d74.Loc == scm.LocReg && d87.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d88 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r76, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
			}
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d88.Imm.Int()))))}
			} else {
				r77 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r77, d88.Reg)
				ctx.W.EmitShlRegImm8(r77, 56)
				ctx.W.EmitShrRegImm8(r77, 56)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
			}
			ctx.FreeDesc(&d88)
			var d90 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d89.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d89.Reg}
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d89.Loc == scm.LocImm {
				if d89.Imm.Int() >= -2147483648 && d89.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d87.Reg, int32(d89.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(d87.Reg, scm.RegR11)
				}
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
			} else {
				ctx.W.EmitAddInt64(d87.Reg, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
			}
			if d90.Loc == scm.LocReg && d87.Loc == scm.LocReg && d90.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d89)
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d90.Imm.Int()) > uint64(64))}
			} else {
				r78 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d90.Reg, 64)
				ctx.W.EmitSetcc(r78, scm.CcA)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r78}
			}
			ctx.FreeDesc(&d90)
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d91.Loc == scm.LocImm {
				if d91.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
			ctx.EmitStoreToStack(d79, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d91.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
			ctx.EmitStoreToStack(d79, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl26)
			}
			ctx.FreeDesc(&d91)
			ctx.W.MarkLabel(lbl26)
			var d92 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r79 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r79, d74.Reg)
				ctx.W.EmitShrRegImm8(r79, 6)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
			}
			if d92.Loc == scm.LocReg && d74.Loc == scm.LocReg && d92.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d93 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d92.Reg, int32(1))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d92.Reg}
			}
			if d93.Loc == scm.LocReg && d92.Loc == scm.LocReg && d93.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d92)
			r80 := ctx.AllocReg()
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r80, uint64(d93.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r80, d93.Reg)
				ctx.W.EmitShlRegImm8(r80, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r80, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r80, d75.Reg)
			}
			r81 := ctx.AllocRegExcept(r80)
			ctx.W.EmitMovRegMem(r81, r80, 0)
			ctx.FreeReg(r80)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r81}
			ctx.FreeDesc(&d93)
			var d95 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d74.Reg, 63)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			}
			if d95.Loc == scm.LocReg && d74.Loc == scm.LocReg && d95.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			d96 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d97 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d95.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() - d95.Imm.Int())}
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d96.Reg}
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d95.Loc == scm.LocImm {
				if d95.Imm.Int() >= -2147483648 && d95.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d96.Reg, int32(d95.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
				ctx.W.EmitSubInt64(d96.Reg, scm.RegR11)
				}
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d96.Reg}
			} else {
				ctx.W.EmitSubInt64(d96.Reg, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d96.Reg}
			}
			if d97.Loc == scm.LocReg && d96.Loc == scm.LocReg && d97.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			var d98 scm.JITValueDesc
			if d94.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) >> uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d94.Reg, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d94.Reg}
			} else {
				{
					shiftSrc := d94.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d97.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d97.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d97.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d98.Loc == scm.LocReg && d94.Loc == scm.LocReg && d98.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			ctx.FreeDesc(&d97)
			var d99 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d98.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() | d98.Imm.Int())}
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
			} else if d98.Loc == scm.LocImm && d98.Imm.Int() == 0 {
				r82 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r82, d79.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d98.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r83, d79.Reg)
				if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r83, int32(d98.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d98.Imm.Int()))
					ctx.W.EmitOrInt64(r83, scratch)
					ctx.FreeReg(scratch)
				}
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			} else {
				r84 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r84, d79.Reg)
				ctx.W.EmitOrInt64(r84, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			}
			if d99.Loc == scm.LocReg && d79.Loc == scm.LocReg && d99.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d98)
			ctx.EmitStoreToStack(d99, 16)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
			d100 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
			if r60 { ctx.UnprotectReg(r61) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d100.Imm.Int()))))}
			} else {
				r85 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r85, d100.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
			}
			ctx.FreeDesc(&d100)
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r86, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r86}
			}
			var d103 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() + d102.Imm.Int())}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d102.Reg}
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d102.Loc == scm.LocImm {
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d101.Reg, int32(d102.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(d101.Reg, scm.RegR11)
				}
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else {
				ctx.W.EmitAddInt64(d101.Reg, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			}
			if d103.Loc == scm.LocReg && d101.Loc == scm.LocReg && d103.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d102)
			r87 := d34.Loc == scm.LocReg
			r88 := d34.Reg
			if r87 { ctx.ProtectReg(r88) }
			r89 := ctx.AllocReg()
			lbl28 := ctx.W.ReserveLabel()
			var d104 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d34.Reg)
				ctx.W.EmitShlRegImm8(r90, 32)
				ctx.W.EmitShrRegImm8(r90, 32)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			}
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d105.Reg)
				ctx.W.EmitShlRegImm8(r92, 56)
				ctx.W.EmitShrRegImm8(r92, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
			}
			ctx.FreeDesc(&d105)
			var d107 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() * d106.Imm.Int())}
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d106.Loc == scm.LocImm {
				if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d104.Reg, int32(d106.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d106.Imm.Int()))
				ctx.W.EmitImulInt64(d104.Reg, scm.RegR11)
				}
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d104.Reg}
			} else {
				ctx.W.EmitImulInt64(d104.Reg, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d104.Reg}
			}
			if d107.Loc == scm.LocReg && d104.Loc == scm.LocReg && d107.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.FreeDesc(&d106)
			var d108 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r93, thisptr.Reg, off)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
			}
			var d109 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r94 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r94, d107.Reg)
				ctx.W.EmitShrRegImm8(r94, 6)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
			}
			if d109.Loc == scm.LocReg && d107.Loc == scm.LocReg && d109.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			r95 := ctx.AllocReg()
			if d109.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r95, uint64(d109.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r95, d109.Reg)
				ctx.W.EmitShlRegImm8(r95, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r95, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r95, d108.Reg)
			}
			r96 := ctx.AllocRegExcept(r95)
			ctx.W.EmitMovRegMem(r96, r95, 0)
			ctx.FreeReg(r95)
			d110 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
			ctx.FreeDesc(&d109)
			var d111 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r97 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r97, d107.Reg)
				ctx.W.EmitAndRegImm32(r97, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			}
			if d111.Loc == scm.LocReg && d107.Loc == scm.LocReg && d111.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			var d112 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d110.Imm.Int()) << uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d110.Reg, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
			} else {
				{
					shiftSrc := d110.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r98, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.W.EmitJmp(lbl29)
				} else {
			ctx.EmitStoreToStack(d112, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d113.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
			ctx.EmitStoreToStack(d112, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d113)
			ctx.W.MarkLabel(lbl30)
			r99 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r99, 24)
			ctx.ProtectReg(r99)
			d114 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r99}
			ctx.UnprotectReg(r99)
			var d115 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r100, thisptr.Reg, off)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
			}
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
			} else {
				r101 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r101, d115.Reg)
				ctx.W.EmitShlRegImm8(r101, 56)
				ctx.W.EmitShrRegImm8(r101, 56)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			}
			ctx.FreeDesc(&d115)
			d117 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() - d116.Imm.Int())}
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d117.Reg}
			} else if d117.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d117.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d116.Loc == scm.LocImm {
				if d116.Imm.Int() >= -2147483648 && d116.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d117.Reg, int32(d116.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
				ctx.W.EmitSubInt64(d117.Reg, scm.RegR11)
				}
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d117.Reg}
			} else {
				ctx.W.EmitSubInt64(d117.Reg, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d117.Reg}
			}
			if d118.Loc == scm.LocReg && d117.Loc == scm.LocReg && d118.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			var d119 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d114.Imm.Int()) >> uint64(d118.Imm.Int())))}
			} else if d118.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d114.Reg, uint8(d118.Imm.Int()))
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d114.Reg}
			} else {
				{
					shiftSrc := d114.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d118.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d118.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d118.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d119.Loc == scm.LocReg && d114.Loc == scm.LocReg && d119.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			ctx.FreeDesc(&d118)
			ctx.EmitMovToReg(r89, d119)
			ctx.W.EmitJmp(lbl28)
			ctx.FreeDesc(&d119)
			ctx.W.MarkLabel(lbl29)
			var d120 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r102, d107.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
			}
			if d120.Loc == scm.LocReg && d107.Loc == scm.LocReg && d120.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			var d121 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r103, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			}
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d121.Imm.Int()))))}
			} else {
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r104, d121.Reg)
				ctx.W.EmitShlRegImm8(r104, 56)
				ctx.W.EmitShrRegImm8(r104, 56)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			}
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() + d122.Imm.Int())}
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d122.Loc == scm.LocImm {
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d120.Reg, int32(d122.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
				ctx.W.EmitAddInt64(d120.Reg, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else {
				ctx.W.EmitAddInt64(d120.Reg, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			}
			if d123.Loc == scm.LocReg && d120.Loc == scm.LocReg && d123.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			ctx.FreeDesc(&d122)
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d123.Imm.Int()) > uint64(64))}
			} else {
				r105 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d123.Reg, 64)
				ctx.W.EmitSetcc(r105, scm.CcA)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r105}
			}
			ctx.FreeDesc(&d123)
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d124.Loc == scm.LocImm {
				if d124.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
			ctx.EmitStoreToStack(d112, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d124.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl33)
			ctx.EmitStoreToStack(d112, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d124)
			ctx.W.MarkLabel(lbl32)
			var d125 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r106 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r106, d107.Reg)
				ctx.W.EmitShrRegImm8(r106, 6)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
			}
			if d125.Loc == scm.LocReg && d107.Loc == scm.LocReg && d125.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d125.Reg, int32(1))
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d125.Reg}
			}
			if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			r107 := ctx.AllocReg()
			if d126.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r107, uint64(d126.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r107, d126.Reg)
				ctx.W.EmitShlRegImm8(r107, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r107, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r107, d108.Reg)
			}
			r108 := ctx.AllocRegExcept(r107)
			ctx.W.EmitMovRegMem(r108, r107, 0)
			ctx.FreeReg(r107)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
			ctx.FreeDesc(&d126)
			var d128 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d107.Reg, 63)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d107.Reg}
			}
			if d128.Loc == scm.LocReg && d107.Loc == scm.LocReg && d128.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			d129 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d130 scm.JITValueDesc
			if d129.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() - d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d129.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d128.Loc == scm.LocImm {
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d129.Reg, int32(d128.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
				ctx.W.EmitSubInt64(d129.Reg, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
			} else {
				ctx.W.EmitSubInt64(d129.Reg, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d129.Reg}
			}
			if d130.Loc == scm.LocReg && d129.Loc == scm.LocReg && d130.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			var d131 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d127.Imm.Int()) >> uint64(d130.Imm.Int())))}
			} else if d130.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d127.Reg, uint8(d130.Imm.Int()))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
			} else {
				{
					shiftSrc := d127.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d131.Loc == scm.LocReg && d127.Loc == scm.LocReg && d131.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.FreeDesc(&d130)
			var d132 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() | d131.Imm.Int())}
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d131.Reg}
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				r109 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r109, d112.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d131.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r110, d112.Reg)
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r110, int32(d131.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
					ctx.W.EmitOrInt64(r110, scratch)
					ctx.FreeReg(scratch)
				}
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
			} else {
				r111 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r111, d112.Reg)
				ctx.W.EmitOrInt64(r111, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			}
			if d132.Loc == scm.LocReg && d112.Loc == scm.LocReg && d132.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.EmitStoreToStack(d132, 24)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			ctx.W.ResolveFixups()
			d133 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
			if r87 { ctx.UnprotectReg(r88) }
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d133.Imm.Int()))))}
			} else {
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r112, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
			}
			ctx.FreeDesc(&d133)
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r113 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r113, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r113}
			}
			var d136 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d135.Loc == scm.LocImm {
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d134.Reg, int32(d135.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
				ctx.W.EmitAddInt64(d134.Reg, scm.RegR11)
				}
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else {
				ctx.W.EmitAddInt64(d134.Reg, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			}
			if d136.Loc == scm.LocReg && d134.Loc == scm.LocReg && d136.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d134)
			ctx.FreeDesc(&d135)
			var d138 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() + d136.Imm.Int())}
			} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
				r114 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r114, d103.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(scratch, d103.Reg)
				if d136.Imm.Int() >= -2147483648 && d136.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d136.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r115 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r115, d103.Reg)
				ctx.W.EmitAddInt64(r115, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
			}
			if d138.Loc == scm.LocReg && d103.Loc == scm.LocReg && d138.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			var d140 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r116 := ctx.AllocReg()
				r117 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r116, fieldAddr)
				ctx.W.EmitMovRegMem64(r117, fieldAddr+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r116, Reg2: r117}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r118 := ctx.AllocReg()
				r119 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r118, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r119, thisptr.Reg, off+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r118, Reg2: r119}
			}
			r120 := ctx.AllocRegExcept(d140.Reg, d140.Reg2, d103.Reg, d103.Reg2, d138.Reg, d138.Reg2)
			r121 := ctx.AllocRegExcept(d140.Reg, d140.Reg2, d103.Reg, d103.Reg2, d138.Reg, d138.Reg2, r120)
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r120, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r120, d140.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() != 0 {
					if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r120, int32(d103.Imm.Int()))
					} else {
						scratch := ctx.AllocReg()
						ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
						ctx.W.EmitAddInt64(r120, scratch)
						ctx.FreeReg(scratch)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r120, d103.Reg)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r121, uint64(d138.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r121, d138.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r121, int32(d103.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
					ctx.W.EmitSubInt64(r121, scratch)
					ctx.FreeReg(scratch)
				}
			} else {
				ctx.W.EmitSubInt64(r121, d103.Reg)
			}
			d141 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r120, Reg2: r121}
			ctx.FreeDesc(&d103)
			ctx.FreeDesc(&d138)
			d142 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d141}, 2)
			ctx.EmitMovPairToResult(&d142, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			var d143 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r122, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
			}
			var d144 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d143.Imm.Int()))))}
			} else {
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r123, d143.Reg)
				ctx.W.EmitShlRegImm8(r123, 32)
				ctx.W.EmitShrRegImm8(r123, 32)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
			}
			ctx.FreeDesc(&d143)
			var d145 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d34.Imm.Int()) == uint64(d144.Imm.Int()))}
			} else if d144.Loc == scm.LocImm {
				r124 := ctx.AllocReg()
				if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d34.Reg, int32(d144.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d144.Imm.Int()))
					ctx.W.EmitCmpInt64(d34.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r124, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r124}
			} else if d34.Loc == scm.LocImm {
				r125 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d144.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r125, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r125}
			} else {
				r126 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d34.Reg, d144.Reg)
				ctx.W.EmitSetcc(r126, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r126}
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d144)
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d145.Loc == scm.LocImm {
				if d145.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d145.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d145)
			ctx.W.MarkLabel(lbl20)
			r127 := ctx.AllocReg()
			lbl36 := ctx.W.ReserveLabel()
			var d146 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r128 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r128, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r128, 32)
				ctx.W.EmitShrRegImm8(r128, 32)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
			}
			ctx.FreeDesc(&idxInt)
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r129, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
			}
			var d148 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d147.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d147.Reg)
				ctx.W.EmitShlRegImm8(r130, 56)
				ctx.W.EmitShrRegImm8(r130, 56)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			}
			ctx.FreeDesc(&d147)
			var d149 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() * d148.Imm.Int())}
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d148.Loc == scm.LocImm {
				if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d146.Reg, int32(d148.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
				ctx.W.EmitImulInt64(d146.Reg, scm.RegR11)
				}
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d146.Reg}
			} else {
				ctx.W.EmitImulInt64(d146.Reg, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d146.Reg}
			}
			if d149.Loc == scm.LocReg && d146.Loc == scm.LocReg && d149.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d148)
			var d150 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
			}
			var d151 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() / 64)}
			} else {
				r132 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r132, d149.Reg)
				ctx.W.EmitShrRegImm8(r132, 6)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			}
			if d151.Loc == scm.LocReg && d149.Loc == scm.LocReg && d151.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			r133 := ctx.AllocReg()
			if d151.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r133, uint64(d151.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r133, d151.Reg)
				ctx.W.EmitShlRegImm8(r133, 3)
			}
			if d150.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
				ctx.W.EmitAddInt64(r133, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r133, d150.Reg)
			}
			r134 := ctx.AllocRegExcept(r133)
			ctx.W.EmitMovRegMem(r134, r133, 0)
			ctx.FreeReg(r133)
			d152 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r134}
			ctx.FreeDesc(&d151)
			var d153 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				r135 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r135, d149.Reg)
				ctx.W.EmitAndRegImm32(r135, 63)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			}
			if d153.Loc == scm.LocReg && d149.Loc == scm.LocReg && d153.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			var d154 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d152.Imm.Int()) << uint64(d153.Imm.Int())))}
			} else if d153.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d152.Reg, uint8(d153.Imm.Int()))
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d152.Reg}
			} else {
				{
					shiftSrc := d152.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d153.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d153.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d153.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d154.Loc == scm.LocReg && d152.Loc == scm.LocReg && d154.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.FreeDesc(&d153)
			var d155 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r136, thisptr.Reg, off)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r136}
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d155.Loc == scm.LocImm {
				if d155.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
			ctx.EmitStoreToStack(d154, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d155.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			ctx.EmitStoreToStack(d154, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d155)
			ctx.W.MarkLabel(lbl38)
			r137 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r137, 32)
			ctx.ProtectReg(r137)
			d156 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r137}
			ctx.UnprotectReg(r137)
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r138, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r138}
			}
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r139, d157.Reg)
				ctx.W.EmitShlRegImm8(r139, 56)
				ctx.W.EmitShrRegImm8(r139, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
			}
			ctx.FreeDesc(&d157)
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
			if d156.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d156.Imm.Int()) >> uint64(d160.Imm.Int())))}
			} else if d160.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d156.Reg, uint8(d160.Imm.Int()))
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d156.Reg}
			} else {
				{
					shiftSrc := d156.Reg
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
			if d161.Loc == scm.LocReg && d156.Loc == scm.LocReg && d161.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d160)
			ctx.EmitMovToReg(r127, d161)
			ctx.W.EmitJmp(lbl36)
			ctx.FreeDesc(&d161)
			ctx.W.MarkLabel(lbl37)
			var d162 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				r140 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r140, d149.Reg)
				ctx.W.EmitAndRegImm32(r140, 63)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			}
			if d162.Loc == scm.LocReg && d149.Loc == scm.LocReg && d162.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			var d163 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r141 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r141, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r141}
			}
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d163.Imm.Int()))))}
			} else {
				r142 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r142, d163.Reg)
				ctx.W.EmitShlRegImm8(r142, 56)
				ctx.W.EmitShrRegImm8(r142, 56)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			}
			ctx.FreeDesc(&d163)
			var d165 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d164.Loc == scm.LocImm {
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d162.Reg, int32(d164.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
				ctx.W.EmitAddInt64(d162.Reg, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			} else {
				ctx.W.EmitAddInt64(d162.Reg, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			}
			if d165.Loc == scm.LocReg && d162.Loc == scm.LocReg && d165.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d164)
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d165.Imm.Int()) > uint64(64))}
			} else {
				r143 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d165.Reg, 64)
				ctx.W.EmitSetcc(r143, scm.CcA)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r143}
			}
			ctx.FreeDesc(&d165)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d166.Loc == scm.LocImm {
				if d166.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
			ctx.EmitStoreToStack(d154, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d166.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
			ctx.EmitStoreToStack(d154, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d166)
			ctx.W.MarkLabel(lbl40)
			var d167 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() / 64)}
			} else {
				r144 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r144, d149.Reg)
				ctx.W.EmitShrRegImm8(r144, 6)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
			}
			if d167.Loc == scm.LocReg && d149.Loc == scm.LocReg && d167.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d167.Reg, int32(1))
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
			}
			if d168.Loc == scm.LocReg && d167.Loc == scm.LocReg && d168.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			r145 := ctx.AllocReg()
			if d168.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r145, uint64(d168.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r145, d168.Reg)
				ctx.W.EmitShlRegImm8(r145, 3)
			}
			if d150.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
				ctx.W.EmitAddInt64(r145, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r145, d150.Reg)
			}
			r146 := ctx.AllocRegExcept(r145)
			ctx.W.EmitMovRegMem(r146, r145, 0)
			ctx.FreeReg(r145)
			d169 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
			ctx.FreeDesc(&d168)
			var d170 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d149.Reg, 63)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d149.Reg}
			}
			if d170.Loc == scm.LocReg && d149.Loc == scm.LocReg && d170.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			d171 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() - d170.Imm.Int())}
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d170.Loc == scm.LocImm {
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d171.Reg, int32(d170.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
				ctx.W.EmitSubInt64(d171.Reg, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
			} else {
				ctx.W.EmitSubInt64(d171.Reg, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
			}
			if d172.Loc == scm.LocReg && d171.Loc == scm.LocReg && d172.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			var d173 scm.JITValueDesc
			if d169.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d169.Imm.Int()) >> uint64(d172.Imm.Int())))}
			} else if d172.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d169.Reg, uint8(d172.Imm.Int()))
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			} else {
				{
					shiftSrc := d169.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d173.Loc == scm.LocReg && d169.Loc == scm.LocReg && d173.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			ctx.FreeDesc(&d172)
			var d174 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d154.Imm.Int() | d173.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
			} else if d173.Loc == scm.LocImm && d173.Imm.Int() == 0 {
				r147 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r147, d154.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d154.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d173.Loc == scm.LocImm {
				r148 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r148, d154.Reg)
				if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r148, int32(d173.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
					ctx.W.EmitOrInt64(r148, scratch)
					ctx.FreeReg(scratch)
				}
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			} else {
				r149 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r149, d154.Reg)
				ctx.W.EmitOrInt64(r149, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
			}
			if d174.Loc == scm.LocReg && d154.Loc == scm.LocReg && d174.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			ctx.EmitStoreToStack(d174, 32)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d175 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r127}
			ctx.FreeDesc(&idxInt)
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d175.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			}
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r151, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r151}
			}
			var d178 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() + d177.Imm.Int())}
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d177.Loc == scm.LocImm {
				if d177.Imm.Int() >= -2147483648 && d177.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d176.Reg, int32(d177.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(d176.Reg, scm.RegR11)
				}
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			} else {
				ctx.W.EmitAddInt64(d176.Reg, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d176.Reg}
			}
			if d178.Loc == scm.LocReg && d176.Loc == scm.LocReg && d178.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d177)
			var d179 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d178.Imm.Int()))))}
			} else {
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r152, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			}
			ctx.FreeDesc(&d178)
			var d180 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d69.Imm.Int()))))}
			} else {
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r153, d69.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			}
			var d181 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + d179.Imm.Int())}
			} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
				r154 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r154, d69.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d179.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(scratch, d69.Reg)
				if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d179.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r155 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r155, d69.Reg)
				ctx.W.EmitAddInt64(r155, d179.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
			}
			if d181.Loc == scm.LocReg && d69.Loc == scm.LocReg && d181.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d181.Imm.Int()))))}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r156, d181.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
			}
			ctx.FreeDesc(&d181)
			r157 := ctx.AllocRegExcept(d140.Reg, d140.Reg2, d180.Reg, d180.Reg2, d182.Reg, d182.Reg2)
			r158 := ctx.AllocRegExcept(d140.Reg, d140.Reg2, d180.Reg, d180.Reg2, d182.Reg, d182.Reg2, r157)
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r157, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r157, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r157, d140.Reg)
			}
			if d180.Loc == scm.LocImm {
				if d180.Imm.Int() != 0 {
					if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r157, int32(d180.Imm.Int()))
					} else {
						scratch := ctx.AllocReg()
						ctx.W.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
						ctx.W.EmitAddInt64(r157, scratch)
						ctx.FreeReg(scratch)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r157, d180.Reg)
			}
			if d182.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r158, uint64(d182.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r158, d182.Reg)
			}
			if d180.Loc == scm.LocImm {
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r158, int32(d180.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
					ctx.W.EmitSubInt64(r158, scratch)
					ctx.FreeReg(scratch)
				}
			} else {
				ctx.W.EmitSubInt64(r158, d180.Reg)
			}
			d183 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r157, Reg2: r158}
			ctx.FreeDesc(&d180)
			ctx.FreeDesc(&d182)
			d184 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d183}, 2)
			ctx.EmitMovPairToResult(&d184, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			var d185 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r159, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r159}
			}
			var d186 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d69.Imm.Int()) == uint64(d185.Imm.Int()))}
			} else if d185.Loc == scm.LocImm {
				r160 := ctx.AllocReg()
				if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d69.Reg, int32(d185.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d185.Imm.Int()))
					ctx.W.EmitCmpInt64(d69.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r160, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
			} else if d69.Loc == scm.LocImm {
				r161 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d185.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r161, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r161}
			} else {
				r162 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d69.Reg, d185.Reg)
				ctx.W.EmitSetcc(r162, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
			}
			ctx.FreeDesc(&d69)
			ctx.FreeDesc(&d185)
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d186.Loc == scm.LocImm {
				if d186.Imm.Bool() {
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d186.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d186)
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
			ctx.W.PatchInt32(r3, int32(40))
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
