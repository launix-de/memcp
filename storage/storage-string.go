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
			if d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r66 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r66, d74.Reg)
				ctx.W.EmitShrRegImm8(r66, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			r67 := ctx.AllocReg()
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r67, uint64(d75.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r67, d75.Reg)
				ctx.W.EmitShlRegImm8(r67, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r67, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r67, d40.Reg)
			}
			r68 := ctx.AllocRegExcept(r67)
			ctx.W.EmitMovRegMem(r68, r67, 0)
			ctx.FreeReg(r67)
			d76 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r69, d74.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
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
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
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
			r71 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r71, 16)
			ctx.ProtectReg(r71)
			d80 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r71}
			ctx.UnprotectReg(r71)
			var d81 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			}
			var d82 scm.JITValueDesc
			if d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d81.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d81.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			}
			ctx.FreeDesc(&d81)
			d83 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d82.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() - d82.Imm.Int())}
			} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d82.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d82.Loc == scm.LocImm {
				if d82.Imm.Int() >= -2147483648 && d82.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d83.Reg, int32(d82.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d82.Imm.Int()))
				ctx.W.EmitSubInt64(d83.Reg, scm.RegR11)
				}
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else {
				ctx.W.EmitSubInt64(d83.Reg, d82.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			}
			if d84.Loc == scm.LocReg && d83.Loc == scm.LocReg && d84.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d82)
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
			ctx.EmitMovToReg(r62, d85)
			ctx.W.EmitJmp(lbl22)
			ctx.FreeDesc(&d85)
			ctx.W.MarkLabel(lbl23)
			var d86 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r74 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r74, d74.Reg)
				ctx.W.EmitAndRegImm32(r74, 63)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
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
				r75 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r75, thisptr.Reg, off)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			}
			var d88 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d87.Imm.Int()))))}
			} else {
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r76, d87.Reg)
				ctx.W.EmitShlRegImm8(r76, 56)
				ctx.W.EmitShrRegImm8(r76, 56)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
			}
			ctx.FreeDesc(&d87)
			var d89 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d88.Imm.Int())}
			} else if d88.Loc == scm.LocImm && d88.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d88.Reg}
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d88.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d88.Loc == scm.LocImm {
				if d88.Imm.Int() >= -2147483648 && d88.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d86.Reg, int32(d88.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
				ctx.W.EmitAddInt64(d86.Reg, scm.RegR11)
				}
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else {
				ctx.W.EmitAddInt64(d86.Reg, d88.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			}
			if d89.Loc == scm.LocReg && d86.Loc == scm.LocReg && d89.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d88)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d89.Imm.Int()) > uint64(64))}
			} else {
				r77 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d89.Reg, 64)
				ctx.W.EmitSetcc(r77, scm.CcA)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r77}
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
				r78 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r78, d74.Reg)
				ctx.W.EmitShrRegImm8(r78, 6)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
			}
			if d91.Loc == scm.LocReg && d74.Loc == scm.LocReg && d91.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d91.Reg, int32(1))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
			}
			if d92.Loc == scm.LocReg && d91.Loc == scm.LocReg && d92.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			r79 := ctx.AllocReg()
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r79, uint64(d92.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r79, d92.Reg)
				ctx.W.EmitShlRegImm8(r79, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r79, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r79, d40.Reg)
			}
			r80 := ctx.AllocRegExcept(r79)
			ctx.W.EmitMovRegMem(r80, r79, 0)
			ctx.FreeReg(r79)
			d93 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r80}
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
				if d94.Imm.Int() >= -2147483648 && d94.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d95.Reg, int32(d94.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.W.EmitSubInt64(d95.Reg, scm.RegR11)
				}
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
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d97.Reg}
			} else if d97.Loc == scm.LocImm && d97.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r81, d78.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d97.Loc == scm.LocImm {
				r82 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r82, d78.Reg)
				if d97.Imm.Int() >= -2147483648 && d97.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r82, int32(d97.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d97.Imm.Int()))
					ctx.W.EmitOrInt64(r82, scratch)
					ctx.FreeReg(scratch)
				}
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
			} else {
				r83 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r83, d78.Reg)
				ctx.W.EmitOrInt64(r83, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
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
			d99 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
			if r60 { ctx.UnprotectReg(r61) }
			var d100 scm.JITValueDesc
			if d99.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d99.Imm.Int()))))}
			} else {
				r84 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r84, d99.Reg)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
			}
			ctx.FreeDesc(&d99)
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r85 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r85, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			}
			var d102 scm.JITValueDesc
			if d100.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d100.Imm.Int() + d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d100.Reg}
			} else if d100.Loc == scm.LocImm && d100.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else if d100.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d100.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d101.Loc == scm.LocImm {
				if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d100.Reg, int32(d101.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(d100.Reg, scm.RegR11)
				}
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d100.Reg}
			} else {
				ctx.W.EmitAddInt64(d100.Reg, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d100.Reg}
			}
			if d102.Loc == scm.LocReg && d100.Loc == scm.LocReg && d102.Reg == d100.Reg {
				ctx.TransferReg(d100.Reg)
				d100.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d100)
			ctx.FreeDesc(&d101)
			r86 := d34.Loc == scm.LocReg
			r87 := d34.Reg
			if r86 { ctx.ProtectReg(r87) }
			r88 := ctx.AllocReg()
			lbl28 := ctx.W.ReserveLabel()
			var d103 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r89, d34.Reg)
				ctx.W.EmitShlRegImm8(r89, 32)
				ctx.W.EmitShrRegImm8(r89, 32)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
			}
			var d104 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r90, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r90}
			}
			var d105 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d104.Imm.Int()))))}
			} else {
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r91, d104.Reg)
				ctx.W.EmitShlRegImm8(r91, 56)
				ctx.W.EmitShrRegImm8(r91, 56)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
			}
			ctx.FreeDesc(&d104)
			var d106 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() * d105.Imm.Int())}
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d105.Loc == scm.LocImm {
				if d105.Imm.Int() >= -2147483648 && d105.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d103.Reg, int32(d105.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d105.Imm.Int()))
				ctx.W.EmitImulInt64(d103.Reg, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d103.Reg}
			} else {
				ctx.W.EmitImulInt64(d103.Reg, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d103.Reg}
			}
			if d106.Loc == scm.LocReg && d103.Loc == scm.LocReg && d106.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d103)
			ctx.FreeDesc(&d105)
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r92, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r92}
			}
			var d108 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() / 64)}
			} else {
				r93 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r93, d106.Reg)
				ctx.W.EmitShrRegImm8(r93, 6)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			}
			if d108.Loc == scm.LocReg && d106.Loc == scm.LocReg && d108.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			r94 := ctx.AllocReg()
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r94, uint64(d108.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r94, d108.Reg)
				ctx.W.EmitShlRegImm8(r94, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r94, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r94, d107.Reg)
			}
			r95 := ctx.AllocRegExcept(r94)
			ctx.W.EmitMovRegMem(r95, r94, 0)
			ctx.FreeReg(r94)
			d109 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r96 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r96, d106.Reg)
				ctx.W.EmitAndRegImm32(r96, 63)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
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
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r97, thisptr.Reg, off)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
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
			r98 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r98, 24)
			ctx.ProtectReg(r98)
			d113 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r98}
			ctx.UnprotectReg(r98)
			var d114 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r99 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r99, thisptr.Reg, off)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			}
			var d115 scm.JITValueDesc
			if d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d114.Imm.Int()))))}
			} else {
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r100, d114.Reg)
				ctx.W.EmitShlRegImm8(r100, 56)
				ctx.W.EmitShrRegImm8(r100, 56)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			}
			ctx.FreeDesc(&d114)
			d116 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d115.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() - d115.Imm.Int())}
			} else if d115.Loc == scm.LocImm && d115.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d115.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d115.Loc == scm.LocImm {
				if d115.Imm.Int() >= -2147483648 && d115.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d116.Reg, int32(d115.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d115.Imm.Int()))
				ctx.W.EmitSubInt64(d116.Reg, scm.RegR11)
				}
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else {
				ctx.W.EmitSubInt64(d116.Reg, d115.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
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
			ctx.EmitMovToReg(r88, d118)
			ctx.W.EmitJmp(lbl28)
			ctx.FreeDesc(&d118)
			ctx.W.MarkLabel(lbl29)
			var d119 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r101 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r101, d106.Reg)
				ctx.W.EmitAndRegImm32(r101, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
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
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r102, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r102}
			}
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d120.Imm.Int()))))}
			} else {
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r103, d120.Reg)
				ctx.W.EmitShlRegImm8(r103, 56)
				ctx.W.EmitShrRegImm8(r103, 56)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
			}
			ctx.FreeDesc(&d120)
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + d121.Imm.Int())}
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d121.Reg}
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d121.Loc == scm.LocImm {
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d119.Reg, int32(d121.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
				ctx.W.EmitAddInt64(d119.Reg, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else {
				ctx.W.EmitAddInt64(d119.Reg, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d122.Imm.Int()) > uint64(64))}
			} else {
				r104 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d122.Reg, 64)
				ctx.W.EmitSetcc(r104, scm.CcA)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r104}
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
				r105 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r105, d106.Reg)
				ctx.W.EmitShrRegImm8(r105, 6)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			}
			if d124.Loc == scm.LocReg && d106.Loc == scm.LocReg && d124.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d124.Reg, int32(1))
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
			}
			if d125.Loc == scm.LocReg && d124.Loc == scm.LocReg && d125.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			r106 := ctx.AllocReg()
			if d125.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r106, uint64(d125.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r106, d125.Reg)
				ctx.W.EmitShlRegImm8(r106, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r106, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r106, d107.Reg)
			}
			r107 := ctx.AllocRegExcept(r106)
			ctx.W.EmitMovRegMem(r107, r106, 0)
			ctx.FreeReg(r106)
			d126 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
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
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d128.Reg, int32(d127.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.W.EmitSubInt64(d128.Reg, scm.RegR11)
				}
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
			} else if d111.Loc == scm.LocImm && d111.Imm.Int() == 0 {
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d130.Reg}
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r108, d111.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
			} else if d111.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d111.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d130.Loc == scm.LocImm {
				r109 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r109, d111.Reg)
				if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r109, int32(d130.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
					ctx.W.EmitOrInt64(r109, scratch)
					ctx.FreeReg(scratch)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
			} else {
				r110 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r110, d111.Reg)
				ctx.W.EmitOrInt64(r110, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
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
			d132 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r88}
			if r86 { ctx.UnprotectReg(r87) }
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d132.Imm.Int()))))}
			} else {
				r111 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r111, d132.Reg)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
			}
			ctx.FreeDesc(&d132)
			var d134 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r112, thisptr.Reg, off)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			}
			var d135 scm.JITValueDesc
			if d133.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() + d134.Imm.Int())}
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d134.Loc == scm.LocImm {
				if d134.Imm.Int() >= -2147483648 && d134.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d133.Reg, int32(d134.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d134.Imm.Int()))
				ctx.W.EmitAddInt64(d133.Reg, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			} else {
				ctx.W.EmitAddInt64(d133.Reg, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d133.Reg}
			}
			if d135.Loc == scm.LocReg && d133.Loc == scm.LocReg && d135.Reg == d133.Reg {
				ctx.TransferReg(d133.Reg)
				d133.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d133)
			ctx.FreeDesc(&d134)
			var d137 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r113 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r113, d102.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(scratch, d102.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r114 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r114, d102.Reg)
				ctx.W.EmitAddInt64(r114, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
			}
			if d137.Loc == scm.LocReg && d102.Loc == scm.LocReg && d137.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r115 := ctx.AllocReg()
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r115, fieldAddr)
				ctx.W.EmitMovRegMem64(r116, fieldAddr+8)
				d139 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r115, Reg2: r116}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r117 := ctx.AllocReg()
				r118 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r117, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r118, thisptr.Reg, off+8)
				d139 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r117, Reg2: r118}
			}
			r119 := ctx.AllocRegExcept(d139.Reg, d139.Reg2, d102.Reg, d102.Reg2, d137.Reg, d137.Reg2)
			r120 := ctx.AllocRegExcept(d139.Reg, d139.Reg2, d102.Reg, d102.Reg2, d137.Reg, d137.Reg2, r119)
			if d139.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r119, uint64(d139.Imm.Int()))
			} else if d139.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r119, d139.Reg)
			} else {
				ctx.W.EmitMovRegReg(r119, d139.Reg)
			}
			if d102.Loc == scm.LocImm {
				if d102.Imm.Int() != 0 {
					if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r119, int32(d102.Imm.Int()))
					} else {
						scratch := ctx.AllocReg()
						ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
						ctx.W.EmitAddInt64(r119, scratch)
						ctx.FreeReg(scratch)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r119, d102.Reg)
			}
			if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d137.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r120, d137.Reg)
			}
			if d102.Loc == scm.LocImm {
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r120, int32(d102.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
					ctx.W.EmitSubInt64(r120, scratch)
					ctx.FreeReg(scratch)
				}
			} else {
				ctx.W.EmitSubInt64(r120, d102.Reg)
			}
			d140 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r119, Reg2: r120}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d137)
			d141 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d140}, 2)
			ctx.EmitMovPairToResult(&d141, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r121, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			}
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d142.Imm.Int()))))}
			} else {
				r122 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r122, d142.Reg)
				ctx.W.EmitShlRegImm8(r122, 32)
				ctx.W.EmitShrRegImm8(r122, 32)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
			}
			ctx.FreeDesc(&d142)
			var d144 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d34.Imm.Int()) == uint64(d143.Imm.Int()))}
			} else if d143.Loc == scm.LocImm {
				r123 := ctx.AllocReg()
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d34.Reg, int32(d143.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d143.Imm.Int()))
					ctx.W.EmitCmpInt64(d34.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r123, scm.CcE)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r123}
			} else if d34.Loc == scm.LocImm {
				r124 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d143.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r124, scm.CcE)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r124}
			} else {
				r125 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d34.Reg, d143.Reg)
				ctx.W.EmitSetcc(r125, scm.CcE)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r125}
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d143)
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d144.Loc == scm.LocImm {
				if d144.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d144.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d144)
			ctx.W.MarkLabel(lbl20)
			r126 := ctx.AllocReg()
			lbl36 := ctx.W.ReserveLabel()
			var d145 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r127 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r127, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r127, 32)
				ctx.W.EmitShrRegImm8(r127, 32)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			}
			ctx.FreeDesc(&idxInt)
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r128 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r128, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r128}
			}
			var d147 scm.JITValueDesc
			if d146.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d146.Imm.Int()))))}
			} else {
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r129, d146.Reg)
				ctx.W.EmitShlRegImm8(r129, 56)
				ctx.W.EmitShrRegImm8(r129, 56)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
			}
			ctx.FreeDesc(&d146)
			var d148 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() * d147.Imm.Int())}
			} else if d145.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d147.Loc == scm.LocImm {
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d145.Reg, int32(d147.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
				ctx.W.EmitImulInt64(d145.Reg, scm.RegR11)
				}
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
			} else {
				ctx.W.EmitImulInt64(d145.Reg, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d145.Reg}
			}
			if d148.Loc == scm.LocReg && d145.Loc == scm.LocReg && d148.Reg == d145.Reg {
				ctx.TransferReg(d145.Reg)
				d145.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d145)
			ctx.FreeDesc(&d147)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() / 64)}
			} else {
				r130 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r130, d148.Reg)
				ctx.W.EmitShrRegImm8(r130, 6)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			}
			if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			r131 := ctx.AllocReg()
			if d149.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r131, uint64(d149.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r131, d149.Reg)
				ctx.W.EmitShlRegImm8(r131, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r131, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r131, d107.Reg)
			}
			r132 := ctx.AllocRegExcept(r131)
			ctx.W.EmitMovRegMem(r132, r131, 0)
			ctx.FreeReg(r131)
			d150 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r132}
			ctx.FreeDesc(&d149)
			var d151 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() % 64)}
			} else {
				r133 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r133, d148.Reg)
				ctx.W.EmitAndRegImm32(r133, 63)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
			}
			if d151.Loc == scm.LocReg && d148.Loc == scm.LocReg && d151.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			var d152 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d150.Imm.Int()) << uint64(d151.Imm.Int())))}
			} else if d151.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d150.Reg, uint8(d151.Imm.Int()))
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
			} else {
				{
					shiftSrc := d150.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r134, thisptr.Reg, off)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r134}
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d153.Loc == scm.LocImm {
				if d153.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
			ctx.EmitStoreToStack(d152, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d153.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			ctx.EmitStoreToStack(d152, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d153)
			ctx.W.MarkLabel(lbl38)
			r135 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r135, 32)
			ctx.ProtectReg(r135)
			d154 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r135}
			ctx.UnprotectReg(r135)
			var d155 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r136, thisptr.Reg, off)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r136}
			}
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d155.Imm.Int()))))}
			} else {
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r137, d155.Reg)
				ctx.W.EmitShlRegImm8(r137, 56)
				ctx.W.EmitShrRegImm8(r137, 56)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			}
			ctx.FreeDesc(&d155)
			d157 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d157.Imm.Int() - d156.Imm.Int())}
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
			} else if d157.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d157.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d156.Loc == scm.LocImm {
				if d156.Imm.Int() >= -2147483648 && d156.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d157.Reg, int32(d156.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d156.Imm.Int()))
				ctx.W.EmitSubInt64(d157.Reg, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
			} else {
				ctx.W.EmitSubInt64(d157.Reg, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
			}
			if d158.Loc == scm.LocReg && d157.Loc == scm.LocReg && d158.Reg == d157.Reg {
				ctx.TransferReg(d157.Reg)
				d157.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			var d159 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d154.Imm.Int()) >> uint64(d158.Imm.Int())))}
			} else if d158.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d154.Reg, uint8(d158.Imm.Int()))
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d154.Reg}
			} else {
				{
					shiftSrc := d154.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d158.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d158.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d158.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d159.Loc == scm.LocReg && d154.Loc == scm.LocReg && d159.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.FreeDesc(&d158)
			ctx.EmitMovToReg(r126, d159)
			ctx.W.EmitJmp(lbl36)
			ctx.FreeDesc(&d159)
			ctx.W.MarkLabel(lbl37)
			var d160 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() % 64)}
			} else {
				r138 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r138, d148.Reg)
				ctx.W.EmitAndRegImm32(r138, 63)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
			}
			if d160.Loc == scm.LocReg && d148.Loc == scm.LocReg && d160.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			var d161 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r139, thisptr.Reg, off)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r139}
			}
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d161.Imm.Int()))))}
			} else {
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r140, d161.Reg)
				ctx.W.EmitShlRegImm8(r140, 56)
				ctx.W.EmitShrRegImm8(r140, 56)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
			}
			ctx.FreeDesc(&d161)
			var d163 scm.JITValueDesc
			if d160.Loc == scm.LocImm && d162.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d160.Imm.Int() + d162.Imm.Int())}
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d160.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d162.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d162.Loc == scm.LocImm {
				if d162.Imm.Int() >= -2147483648 && d162.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d160.Reg, int32(d162.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d162.Imm.Int()))
				ctx.W.EmitAddInt64(d160.Reg, scm.RegR11)
				}
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			} else {
				ctx.W.EmitAddInt64(d160.Reg, d162.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
			}
			if d163.Loc == scm.LocReg && d160.Loc == scm.LocReg && d163.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d160)
			ctx.FreeDesc(&d162)
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d163.Imm.Int()) > uint64(64))}
			} else {
				r141 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d163.Reg, 64)
				ctx.W.EmitSetcc(r141, scm.CcA)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r141}
			}
			ctx.FreeDesc(&d163)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d164.Loc == scm.LocImm {
				if d164.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
			ctx.EmitStoreToStack(d152, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d164.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
			ctx.EmitStoreToStack(d152, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d164)
			ctx.W.MarkLabel(lbl40)
			var d165 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() / 64)}
			} else {
				r142 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r142, d148.Reg)
				ctx.W.EmitShrRegImm8(r142, 6)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
			}
			if d165.Loc == scm.LocReg && d148.Loc == scm.LocReg && d165.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d165.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d165.Reg, int32(1))
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			}
			if d166.Loc == scm.LocReg && d165.Loc == scm.LocReg && d166.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d165)
			r143 := ctx.AllocReg()
			if d166.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r143, uint64(d166.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r143, d166.Reg)
				ctx.W.EmitShlRegImm8(r143, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r143, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r143, d107.Reg)
			}
			r144 := ctx.AllocRegExcept(r143)
			ctx.W.EmitMovRegMem(r144, r143, 0)
			ctx.FreeReg(r143)
			d167 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r144}
			ctx.FreeDesc(&d166)
			var d168 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d148.Reg, 63)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d148.Reg}
			}
			if d168.Loc == scm.LocReg && d148.Loc == scm.LocReg && d168.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d148)
			d169 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm && d168.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() - d168.Imm.Int())}
			} else if d168.Loc == scm.LocImm && d168.Imm.Int() == 0 {
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d169.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d168.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d168.Loc == scm.LocImm {
				if d168.Imm.Int() >= -2147483648 && d168.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d169.Reg, int32(d168.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d168.Imm.Int()))
				ctx.W.EmitSubInt64(d169.Reg, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			} else {
				ctx.W.EmitSubInt64(d169.Reg, d168.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			}
			if d170.Loc == scm.LocReg && d169.Loc == scm.LocReg && d170.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			var d171 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d167.Imm.Int()) >> uint64(d170.Imm.Int())))}
			} else if d170.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d167.Reg, uint8(d170.Imm.Int()))
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d167.Reg}
			} else {
				{
					shiftSrc := d167.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d171.Loc == scm.LocReg && d167.Loc == scm.LocReg && d171.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.FreeDesc(&d170)
			var d172 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() | d171.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r145 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r145, d152.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d152.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d171.Loc == scm.LocImm {
				r146 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r146, d152.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r146, int32(d171.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
					ctx.W.EmitOrInt64(r146, scratch)
					ctx.FreeReg(scratch)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
			} else {
				r147 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r147, d152.Reg)
				ctx.W.EmitOrInt64(r147, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			}
			if d172.Loc == scm.LocReg && d152.Loc == scm.LocReg && d172.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			ctx.EmitStoreToStack(d172, 32)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d173 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r126}
			ctx.FreeDesc(&idxInt)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d173.Imm.Int()))))}
			} else {
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r148, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
			}
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r149, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
			}
			var d176 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() + d175.Imm.Int())}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d174.Reg}
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d175.Loc == scm.LocImm {
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d174.Reg, int32(d175.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
				ctx.W.EmitAddInt64(d174.Reg, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d174.Reg}
			} else {
				ctx.W.EmitAddInt64(d174.Reg, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d174.Reg}
			}
			if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d176.Imm.Int()))))}
			} else {
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r150, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			}
			ctx.FreeDesc(&d176)
			var d178 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d69.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d69.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
			}
			var d179 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d177.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + d177.Imm.Int())}
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				r152 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r152, d69.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d177.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(scratch, d69.Reg)
				if d177.Imm.Int() >= -2147483648 && d177.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d177.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r153 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r153, d69.Reg)
				ctx.W.EmitAddInt64(r153, d177.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
			}
			if d179.Loc == scm.LocReg && d69.Loc == scm.LocReg && d179.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d177)
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d179.Imm.Int()))))}
			} else {
				r154 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r154, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			}
			ctx.FreeDesc(&d179)
			r155 := ctx.AllocRegExcept(d139.Reg, d139.Reg2, d178.Reg, d178.Reg2, d180.Reg, d180.Reg2)
			r156 := ctx.AllocRegExcept(d139.Reg, d139.Reg2, d178.Reg, d178.Reg2, d180.Reg, d180.Reg2, r155)
			if d139.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r155, uint64(d139.Imm.Int()))
			} else if d139.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r155, d139.Reg)
			} else {
				ctx.W.EmitMovRegReg(r155, d139.Reg)
			}
			if d178.Loc == scm.LocImm {
				if d178.Imm.Int() != 0 {
					if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r155, int32(d178.Imm.Int()))
					} else {
						scratch := ctx.AllocReg()
						ctx.W.EmitMovRegImm64(scratch, uint64(d178.Imm.Int()))
						ctx.W.EmitAddInt64(r155, scratch)
						ctx.FreeReg(scratch)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r155, d178.Reg)
			}
			if d180.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r156, uint64(d180.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r156, d180.Reg)
			}
			if d178.Loc == scm.LocImm {
				if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r156, int32(d178.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d178.Imm.Int()))
					ctx.W.EmitSubInt64(r156, scratch)
					ctx.FreeReg(scratch)
				}
			} else {
				ctx.W.EmitSubInt64(r156, d178.Reg)
			}
			d181 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r155, Reg2: r156}
			ctx.FreeDesc(&d178)
			ctx.FreeDesc(&d180)
			d182 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d181}, 2)
			ctx.EmitMovPairToResult(&d182, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			var d183 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r157, thisptr.Reg, off)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
			}
			var d184 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d69.Imm.Int()) == uint64(d183.Imm.Int()))}
			} else if d183.Loc == scm.LocImm {
				r158 := ctx.AllocReg()
				if d183.Imm.Int() >= -2147483648 && d183.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d69.Reg, int32(d183.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d183.Imm.Int()))
					ctx.W.EmitCmpInt64(d69.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r158, scm.CcE)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r158}
			} else if d69.Loc == scm.LocImm {
				r159 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d183.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r159, scm.CcE)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r159}
			} else {
				r160 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d69.Reg, d183.Reg)
				ctx.W.EmitSetcc(r160, scm.CcE)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
			}
			ctx.FreeDesc(&d69)
			ctx.FreeDesc(&d183)
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d184.Loc == scm.LocImm {
				if d184.Imm.Bool() {
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d184.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d184)
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
