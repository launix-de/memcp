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
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).nodict)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).nodict))
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r2, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d0)
			}
			d1 := d0
			ctx.EnsureDesc(&d1)
			if d1.Loc != scm.LocImm && d1.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			if d1.Loc == scm.LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.W.MarkLabel(lbl4)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl5)
			ctx.W.EmitJmp(lbl3)
			ctx.FreeDesc(&d0)
			ctx.W.MarkLabel(lbl3)
			ctx.EnsureDesc(&idxInt)
			d2 := idxInt
			_ = d2
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 := ctx.W.EmitSubRSP32Fixup()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl7)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d2.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r6, d2.Reg)
				ctx.W.EmitShlRegImm8(r6, 32)
				ctx.W.EmitShrRegImm8(r6, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d3)
			}
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r7, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d4)
			}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d4)
			var d5 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d4.Reg)
				ctx.W.EmitShlRegImm8(r8, 56)
				ctx.W.EmitShrRegImm8(r8, 56)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			ctx.FreeDesc(&d4)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d5)
			var d6 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() * d5.Imm.Int())}
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(scratch, d3.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else {
				r9 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				ctx.W.EmitImulInt64(r9, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d6)
			}
			if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			r10 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r10, uint64(dataPtr))
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10, StackOff: int32(sliceLen)}
				ctx.BindReg(r10, &d7)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d7)
			}
			ctx.BindReg(r10, &d7)
			ctx.EnsureDesc(&d6)
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r11, d6.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d8)
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			r12 := ctx.AllocReg()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d7)
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d8.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d8.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.W.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r12, d7.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d9 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d9)
			ctx.FreeDesc(&d8)
			ctx.EnsureDesc(&d6)
			var d10 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r14, d6.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d10)
			}
			if d10.Loc == scm.LocReg && d6.Loc == scm.LocReg && d10.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) << uint64(d10.Imm.Int())))}
			} else if d10.Loc == scm.LocImm {
				r15 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r15, d9.Reg)
				ctx.W.EmitShlRegImm8(r15, uint8(d10.Imm.Int()))
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d11)
			} else {
				{
					shiftSrc := d9.Reg
					r16 := ctx.AllocRegExcept(d9.Reg)
					ctx.W.EmitMovRegReg(r16, d9.Reg)
					shiftSrc = r16
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d10.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d10.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d10.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d11)
				}
			}
			if d11.Loc == scm.LocReg && d9.Loc == scm.LocReg && d11.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d10)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 25)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d12)
			}
			d13 := d12
			ctx.EnsureDesc(&d13)
			if d13.Loc != scm.LocImm && d13.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d13.Loc == scm.LocImm {
				if d13.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl11)
			d14 := d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
			ctx.W.EmitJmp(lbl9)
			ctx.FreeDesc(&d12)
			ctx.W.MarkLabel(lbl9)
			d15 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d16 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d16)
			}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d16)
			var d17 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d16.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d16.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d17)
			}
			ctx.FreeDesc(&d16)
			d18 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d17)
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() - d17.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r20, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d19)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d17.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else {
				r21 := ctx.AllocRegExcept(d18.Reg, d17.Reg)
				ctx.W.EmitMovRegReg(r21, d18.Reg)
				ctx.W.EmitSubInt64(r21, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			}
			if d19.Loc == scm.LocReg && d18.Loc == scm.LocReg && d19.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d19)
			var d20 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) >> uint64(d19.Imm.Int())))}
			} else if d19.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r22, d15.Reg)
				ctx.W.EmitShrRegImm8(r22, uint8(d19.Imm.Int()))
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d20)
			} else {
				{
					shiftSrc := d15.Reg
					r23 := ctx.AllocRegExcept(d15.Reg)
					ctx.W.EmitMovRegReg(r23, d15.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d19.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d19.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d19.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d20)
				}
			}
			if d20.Loc == scm.LocReg && d15.Loc == scm.LocReg && d20.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.FreeDesc(&d19)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			if d20.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r24, d20)
			}
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl8)
			d15 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d21 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r25, d6.Reg)
				ctx.W.EmitAndRegImm32(r25, 63)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d21)
			}
			if d21.Loc == scm.LocReg && d6.Loc == scm.LocReg && d21.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			var d22 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d22)
			}
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d22.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d22.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
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
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() + d23.Imm.Int())}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r28, d21.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d24)
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
				ctx.BindReg(d23.Reg, &d24)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else {
				r29 := ctx.AllocRegExcept(d21.Reg, d23.Reg)
				ctx.W.EmitMovRegReg(r29, d21.Reg)
				ctx.W.EmitAddInt64(r29, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d24)
			}
			if d24.Loc == scm.LocReg && d21.Loc == scm.LocReg && d24.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d23)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d24.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitCmpRegImm32(d24.Reg, 64)
				ctx.W.EmitSetcc(r30, scm.CcA)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d25)
			}
			ctx.FreeDesc(&d24)
			d26 := d25
			ctx.EnsureDesc(&d26)
			if d26.Loc != scm.LocImm && d26.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d26.Loc == scm.LocImm {
				if d26.Imm.Bool() {
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.EmitJmp(lbl14)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d26.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.W.MarkLabel(lbl13)
			ctx.W.EmitJmp(lbl12)
			ctx.W.MarkLabel(lbl14)
			d27 := d11
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, 0)
			ctx.W.EmitJmp(lbl9)
			ctx.FreeDesc(&d25)
			ctx.W.MarkLabel(lbl12)
			d15 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d28 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r31, d6.Reg)
				ctx.W.EmitShrRegImm8(r31, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d28)
			}
			if d28.Loc == scm.LocReg && d6.Loc == scm.LocReg && d28.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(scratch, d28.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			}
			if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d29)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d7)
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r32, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r32, d29.Reg)
				ctx.W.EmitShlRegImm8(r32, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.W.EmitAddInt64(r32, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r32, d7.Reg)
			}
			r33 := ctx.AllocRegExcept(r32)
			ctx.W.EmitMovRegMem(r33, r32, 0)
			ctx.FreeReg(r32)
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d30)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d6)
			var d31 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r34, d6.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d31)
			}
			if d31.Loc == scm.LocReg && d6.Loc == scm.LocReg && d31.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			d32 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d31)
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r35, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(scratch, d32.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r36 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.W.EmitMovRegReg(r36, d32.Reg)
				ctx.W.EmitSubInt64(r36, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d33)
			}
			if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r37, d30.Reg)
				ctx.W.EmitShrRegImm8(r37, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r38 := ctx.AllocRegExcept(d30.Reg)
					ctx.W.EmitMovRegReg(r38, d30.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d33.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d33.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d34)
				}
			}
			if d34.Loc == scm.LocReg && d30.Loc == scm.LocReg && d34.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() | d34.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r39, d11.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d35)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r40, d11.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r40, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitOrInt64(r40, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			} else {
				r41 := ctx.AllocRegExcept(d11.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r41, d11.Reg)
				ctx.W.EmitOrInt64(r41, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d35)
			}
			if d35.Loc == scm.LocReg && d11.Loc == scm.LocReg && d35.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			d36 := d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 0)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl6)
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d37)
			ctx.BindReg(r24, &d37)
			if r3 { ctx.UnprotectReg(r4) }
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			}
			ctx.FreeDesc(&d37)
			var d39 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d39)
			}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r44, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d40)
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
				ctx.BindReg(d39.Reg, &d40)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r45 := ctx.AllocRegExcept(d38.Reg, d39.Reg)
				ctx.W.EmitMovRegReg(r45, d38.Reg)
				ctx.W.EmitAddInt64(r45, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d40)
			}
			if d40.Loc == scm.LocReg && d38.Loc == scm.LocReg && d40.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d39)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d40.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r46, d40.Reg)
				ctx.W.EmitShlRegImm8(r46, 32)
				ctx.W.EmitShrRegImm8(r46, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d41)
			}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d42)
			}
			d43 := d42
			ctx.EnsureDesc(&d43)
			if d43.Loc != scm.LocImm && d43.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.W.MarkLabel(lbl17)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitJmp(lbl16)
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl2)
			ctx.EnsureDesc(&idxInt)
			d44 := idxInt
			_ = d44
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl20)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d44.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d44.Reg)
				ctx.W.EmitShlRegImm8(r50, 32)
				ctx.W.EmitShrRegImm8(r50, 32)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d45)
			}
			var d46 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r51, thisptr.Reg, off)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d46)
			}
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d46.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d46.Reg)
				ctx.W.EmitShlRegImm8(r52, 56)
				ctx.W.EmitShrRegImm8(r52, 56)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d47)
			}
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			var d48 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() * d47.Imm.Int())}
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(scratch, d45.Reg)
				if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d47.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else {
				r53 := ctx.AllocRegExcept(d45.Reg, d47.Reg)
				ctx.W.EmitMovRegReg(r53, d45.Reg)
				ctx.W.EmitImulInt64(r53, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d48)
			}
			if d48.Loc == scm.LocReg && d45.Loc == scm.LocReg && d48.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.FreeDesc(&d47)
			var d49 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r54, uint64(dataPtr))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d49)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d49)
			}
			ctx.BindReg(r54, &d49)
			ctx.EnsureDesc(&d48)
			var d50 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r55, d48.Reg)
				ctx.W.EmitShrRegImm8(r55, 6)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d50)
			}
			if d50.Loc == scm.LocReg && d48.Loc == scm.LocReg && d50.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d50)
			r56 := ctx.AllocReg()
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d49)
			if d50.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r56, uint64(d50.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r56, d50.Reg)
				ctx.W.EmitShlRegImm8(r56, 3)
			}
			if d49.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.W.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r56, d49.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.W.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d51 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d51)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d48)
			var d52 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r58, d48.Reg)
				ctx.W.EmitAndRegImm32(r58, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d52)
			}
			if d52.Loc == scm.LocReg && d48.Loc == scm.LocReg && d52.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d52)
			var d53 scm.JITValueDesc
			if d51.Loc == scm.LocImm && d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) << uint64(d52.Imm.Int())))}
			} else if d52.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r59, d51.Reg)
				ctx.W.EmitShlRegImm8(r59, uint8(d52.Imm.Int()))
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d53)
			} else {
				{
					shiftSrc := d51.Reg
					r60 := ctx.AllocRegExcept(d51.Reg)
					ctx.W.EmitMovRegReg(r60, d51.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d52.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d52.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d52.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d53)
				}
			}
			if d53.Loc == scm.LocReg && d51.Loc == scm.LocReg && d53.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			ctx.FreeDesc(&d52)
			var d54 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d54)
			}
			d55 := d54
			ctx.EnsureDesc(&d55)
			if d55.Loc != scm.LocImm && d55.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d55.Loc == scm.LocImm {
				if d55.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d55.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.W.MarkLabel(lbl23)
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl24)
			d56 := d53
			if d56.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			ctx.EmitStoreToStack(d56, 8)
			ctx.W.EmitJmp(lbl22)
			ctx.FreeDesc(&d54)
			ctx.W.MarkLabel(lbl22)
			d57 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d58 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d58)
			}
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d58)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d58.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d58.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d59)
			}
			ctx.FreeDesc(&d58)
			d60 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d59)
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d60.Imm.Int() - d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegReg(r64, d60.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d61)
			} else if d60.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d60.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d59.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegReg(scratch, d60.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else {
				r65 := ctx.AllocRegExcept(d60.Reg, d59.Reg)
				ctx.W.EmitMovRegReg(r65, d60.Reg)
				ctx.W.EmitSubInt64(r65, d59.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d61)
			}
			if d61.Loc == scm.LocReg && d60.Loc == scm.LocReg && d61.Reg == d60.Reg {
				ctx.TransferReg(d60.Reg)
				d60.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d57.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) >> uint64(d61.Imm.Int())))}
			} else if d61.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(r66, d57.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d61.Imm.Int()))
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d62)
			} else {
				{
					shiftSrc := d57.Reg
					r67 := ctx.AllocRegExcept(d57.Reg)
					ctx.W.EmitMovRegReg(r67, d57.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d61.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d61.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d61.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d62)
				}
			}
			if d62.Loc == scm.LocReg && d57.Loc == scm.LocReg && d62.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			ctx.FreeDesc(&d61)
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d62)
			if d62.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r68, d62)
			}
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl21)
			d57 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d48)
			var d63 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r69, d48.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d63)
			}
			if d63.Loc == scm.LocReg && d48.Loc == scm.LocReg && d63.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			var d64 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d64)
			}
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d64)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d64.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d64.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d65)
			}
			ctx.FreeDesc(&d64)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d63.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() + d65.Imm.Int())}
			} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegReg(r72, d63.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d66)
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
				ctx.BindReg(d65.Reg, &d66)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d65.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			} else if d65.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegReg(scratch, d63.Reg)
				if d65.Imm.Int() >= -2147483648 && d65.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d65.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d65.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			} else {
				r73 := ctx.AllocRegExcept(d63.Reg, d65.Reg)
				ctx.W.EmitMovRegReg(r73, d63.Reg)
				ctx.W.EmitAddInt64(r73, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d66)
			}
			if d66.Loc == scm.LocReg && d63.Loc == scm.LocReg && d66.Reg == d63.Reg {
				ctx.TransferReg(d63.Reg)
				d63.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.FreeDesc(&d65)
			ctx.EnsureDesc(&d66)
			var d67 scm.JITValueDesc
			if d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d66.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitCmpRegImm32(d66.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d67)
			}
			ctx.FreeDesc(&d66)
			d68 := d67
			ctx.EnsureDesc(&d68)
			if d68.Loc != scm.LocImm && d68.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d68.Loc == scm.LocImm {
				if d68.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
					ctx.W.EmitJmp(lbl27)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d68.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.W.MarkLabel(lbl26)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl27)
			d69 := d53
			if d69.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d69)
			ctx.EmitStoreToStack(d69, 8)
			ctx.W.EmitJmp(lbl22)
			ctx.FreeDesc(&d67)
			ctx.W.MarkLabel(lbl25)
			d57 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d48)
			var d70 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r75, d48.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d70)
			}
			if d70.Loc == scm.LocReg && d48.Loc == scm.LocReg && d70.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d70.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(scratch, d70.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			}
			if d71.Loc == scm.LocReg && d70.Loc == scm.LocReg && d71.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d70)
			ctx.EnsureDesc(&d71)
			r76 := ctx.AllocReg()
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d49)
			if d71.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d71.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d71.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d49.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d49.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d72 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d72)
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d48)
			var d73 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(r78, d48.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d73)
			}
			if d73.Loc == scm.LocReg && d48.Loc == scm.LocReg && d73.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			d74 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d73)
			var d75 scm.JITValueDesc
			if d74.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() - d73.Imm.Int())}
			} else if d73.Loc == scm.LocImm && d73.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r79, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d75)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d73.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else if d73.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(scratch, d74.Reg)
				if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d73.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else {
				r80 := ctx.AllocRegExcept(d74.Reg, d73.Reg)
				ctx.W.EmitMovRegReg(r80, d74.Reg)
				ctx.W.EmitSubInt64(r80, d73.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d75)
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d73)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d72.Loc == scm.LocImm && d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d72.Imm.Int()) >> uint64(d75.Imm.Int())))}
			} else if d75.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(r81, d72.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d75.Imm.Int()))
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d76)
			} else {
				{
					shiftSrc := d72.Reg
					r82 := ctx.AllocRegExcept(d72.Reg)
					ctx.W.EmitMovRegReg(r82, d72.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d75.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d75.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d75.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d76)
				}
			}
			if d76.Loc == scm.LocReg && d72.Loc == scm.LocReg && d76.Reg == d72.Reg {
				ctx.TransferReg(d72.Reg)
				d72.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d76)
			var d77 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() | d76.Imm.Int())}
			} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
				ctx.BindReg(d76.Reg, &d77)
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r83, d53.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d77)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else if d76.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r84, d53.Reg)
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d76.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d77)
			} else {
				r85 := ctx.AllocRegExcept(d53.Reg, d76.Reg)
				ctx.W.EmitMovRegReg(r85, d53.Reg)
				ctx.W.EmitOrInt64(r85, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d77)
			}
			if d77.Loc == scm.LocReg && d53.Loc == scm.LocReg && d77.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			d78 := d77
			if d78.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d78)
			ctx.EmitStoreToStack(d78, 8)
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl19)
			d79 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d79)
			ctx.BindReg(r68, &d79)
			if r48 { ctx.UnprotectReg(r49) }
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d79)
			var d80 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d79.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d80)
			}
			ctx.FreeDesc(&d79)
			var d81 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d81)
			}
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d81)
			var d82 scm.JITValueDesc
			if d80.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d80.Imm.Int() + d81.Imm.Int())}
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegReg(r88, d80.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d82)
			} else if d80.Loc == scm.LocImm && d80.Imm.Int() == 0 {
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
				ctx.BindReg(d81.Reg, &d82)
			} else if d80.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d80.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			} else if d81.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegReg(scratch, d80.Reg)
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d81.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			} else {
				r89 := ctx.AllocRegExcept(d80.Reg, d81.Reg)
				ctx.W.EmitMovRegReg(r89, d80.Reg)
				ctx.W.EmitAddInt64(r89, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d82)
			}
			if d82.Loc == scm.LocReg && d80.Loc == scm.LocReg && d82.Reg == d80.Reg {
				ctx.TransferReg(d80.Reg)
				d80.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			ctx.FreeDesc(&d81)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d82)
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d82.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d82.Reg)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d83)
			}
			ctx.FreeDesc(&d82)
			var d84 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d84)
			}
			d85 := d84
			ctx.EnsureDesc(&d85)
			if d85.Loc != scm.LocImm && d85.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d85.Loc == scm.LocImm {
				if d85.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d85.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl30)
				ctx.W.EmitJmp(lbl31)
			}
			ctx.W.MarkLabel(lbl30)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl31)
			ctx.W.EmitJmp(lbl29)
			ctx.FreeDesc(&d84)
			ctx.W.MarkLabel(lbl16)
			ctx.EnsureDesc(&d41)
			d86 := d41
			_ = d86
			r92 := d41.Loc == scm.LocReg
			r93 := d41.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl33)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d86.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d86.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d87)
			}
			var d88 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d88)
			}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d88.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d88.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d89)
			}
			ctx.FreeDesc(&d88)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d89)
			var d90 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() * d89.Imm.Int())}
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(scratch, d87.Reg)
				if d89.Imm.Int() >= -2147483648 && d89.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d89.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else {
				r97 := ctx.AllocRegExcept(d87.Reg, d89.Reg)
				ctx.W.EmitMovRegReg(r97, d87.Reg)
				ctx.W.EmitImulInt64(r97, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d90)
			}
			if d90.Loc == scm.LocReg && d87.Loc == scm.LocReg && d90.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d89)
			var d91 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d91)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d91)
			}
			ctx.BindReg(r98, &d91)
			ctx.EnsureDesc(&d90)
			var d92 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r99, d90.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d92)
			}
			if d92.Loc == scm.LocReg && d90.Loc == scm.LocReg && d92.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d92)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d91)
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d92.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d92.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d91.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d91.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d91.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d93 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d93)
			ctx.FreeDesc(&d92)
			ctx.EnsureDesc(&d90)
			var d94 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r102, d90.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d94)
			}
			if d94.Loc == scm.LocReg && d90.Loc == scm.LocReg && d94.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d94)
			var d95 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) << uint64(d94.Imm.Int())))}
			} else if d94.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r103, d93.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d94.Imm.Int()))
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d95)
			} else {
				{
					shiftSrc := d93.Reg
					r104 := ctx.AllocRegExcept(d93.Reg)
					ctx.W.EmitMovRegReg(r104, d93.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d94.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d94.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d94.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d95)
				}
			}
			if d95.Loc == scm.LocReg && d93.Loc == scm.LocReg && d95.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			ctx.FreeDesc(&d94)
			var d96 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d96)
			}
			d97 := d96
			ctx.EnsureDesc(&d97)
			if d97.Loc != scm.LocImm && d97.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			if d97.Loc == scm.LocImm {
				if d97.Imm.Bool() {
					ctx.W.EmitJmp(lbl36)
				} else {
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d97.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.W.MarkLabel(lbl36)
			ctx.W.EmitJmp(lbl34)
			ctx.W.MarkLabel(lbl37)
			d98 := d95
			if d98.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d98)
			ctx.EmitStoreToStack(d98, 16)
			ctx.W.EmitJmp(lbl35)
			ctx.FreeDesc(&d96)
			ctx.W.MarkLabel(lbl35)
			d99 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d100 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d100)
			}
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d100)
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d100.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d100.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d101)
			}
			ctx.FreeDesc(&d100)
			d102 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d101)
			var d103 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() - d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r108, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d103)
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d101.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(scratch, d102.Reg)
				if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d101.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else {
				r109 := ctx.AllocRegExcept(d102.Reg, d101.Reg)
				ctx.W.EmitMovRegReg(r109, d102.Reg)
				ctx.W.EmitSubInt64(r109, d101.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d103)
			}
			if d103.Loc == scm.LocReg && d102.Loc == scm.LocReg && d103.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.EnsureDesc(&d99)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d99.Imm.Int()) >> uint64(d103.Imm.Int())))}
			} else if d103.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r110, d99.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d103.Imm.Int()))
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d104)
			} else {
				{
					shiftSrc := d99.Reg
					r111 := ctx.AllocRegExcept(d99.Reg)
					ctx.W.EmitMovRegReg(r111, d99.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d103.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d103.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d103.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d104)
				}
			}
			if d104.Loc == scm.LocReg && d99.Loc == scm.LocReg && d104.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			ctx.FreeDesc(&d103)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d104)
			if d104.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d104)
			}
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl34)
			d99 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d90)
			var d105 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r113, d90.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d105)
			}
			if d105.Loc == scm.LocReg && d90.Loc == scm.LocReg && d105.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			var d106 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d106)
			}
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d106.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d106.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d107)
			}
			ctx.FreeDesc(&d106)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d107)
			var d108 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d105.Imm.Int() + d107.Imm.Int())}
			} else if d107.Loc == scm.LocImm && d107.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(r116, d105.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d108)
			} else if d105.Loc == scm.LocImm && d105.Imm.Int() == 0 {
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d107.Reg}
				ctx.BindReg(d107.Reg, &d108)
			} else if d105.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d105.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d107.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else if d107.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(scratch, d105.Reg)
				if d107.Imm.Int() >= -2147483648 && d107.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d107.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d108)
			} else {
				r117 := ctx.AllocRegExcept(d105.Reg, d107.Reg)
				ctx.W.EmitMovRegReg(r117, d105.Reg)
				ctx.W.EmitAddInt64(r117, d107.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d108)
			}
			if d108.Loc == scm.LocReg && d105.Loc == scm.LocReg && d108.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d105)
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d108)
			var d109 scm.JITValueDesc
			if d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d108.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitCmpRegImm32(d108.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d109)
			}
			ctx.FreeDesc(&d108)
			d110 := d109
			ctx.EnsureDesc(&d110)
			if d110.Loc != scm.LocImm && d110.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d110.Loc == scm.LocImm {
				if d110.Imm.Bool() {
					ctx.W.EmitJmp(lbl39)
				} else {
					ctx.W.EmitJmp(lbl40)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d110.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.W.MarkLabel(lbl39)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl40)
			d111 := d95
			if d111.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d111)
			ctx.EmitStoreToStack(d111, 16)
			ctx.W.EmitJmp(lbl35)
			ctx.FreeDesc(&d109)
			ctx.W.MarkLabel(lbl38)
			d99 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d90)
			var d112 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r119, d90.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d112)
			}
			if d112.Loc == scm.LocReg && d90.Loc == scm.LocReg && d112.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d112)
			var d113 scm.JITValueDesc
			if d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(scratch, d112.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			}
			if d113.Loc == scm.LocReg && d112.Loc == scm.LocReg && d113.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.EnsureDesc(&d113)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d91)
			if d113.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d113.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d113.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d91.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d91.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d91.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d114 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d114)
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d90)
			var d115 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d90.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r122, d90.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d115)
			}
			if d115.Loc == scm.LocReg && d90.Loc == scm.LocReg && d115.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d90)
			d116 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d115)
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d115.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() - d115.Imm.Int())}
			} else if d115.Loc == scm.LocImm && d115.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r123, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d117)
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
				r124 := ctx.AllocRegExcept(d116.Reg, d115.Reg)
				ctx.W.EmitMovRegReg(r124, d116.Reg)
				ctx.W.EmitSubInt64(r124, d115.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d117)
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d117)
			var d118 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d114.Imm.Int()) >> uint64(d117.Imm.Int())))}
			} else if d117.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r125, d114.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d117.Imm.Int()))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d118)
			} else {
				{
					shiftSrc := d114.Reg
					r126 := ctx.AllocRegExcept(d114.Reg)
					ctx.W.EmitMovRegReg(r126, d114.Reg)
					shiftSrc = r126
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
			if d118.Loc == scm.LocReg && d114.Loc == scm.LocReg && d118.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			ctx.FreeDesc(&d117)
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d118)
			var d119 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() | d118.Imm.Int())}
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d118.Reg}
				ctx.BindReg(d118.Reg, &d119)
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r127, d95.Reg)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d119)
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d95.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d118.Reg)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d119)
			} else if d118.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r128, d95.Reg)
				if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d118.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d119)
			} else {
				r129 := ctx.AllocRegExcept(d95.Reg, d118.Reg)
				ctx.W.EmitMovRegReg(r129, d95.Reg)
				ctx.W.EmitOrInt64(r129, d118.Reg)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d119)
			}
			if d119.Loc == scm.LocReg && d95.Loc == scm.LocReg && d119.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			d120 := d119
			if d120.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d120)
			ctx.EmitStoreToStack(d120, 16)
			ctx.W.EmitJmp(lbl35)
			ctx.W.MarkLabel(lbl32)
			d121 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d121)
			ctx.BindReg(r112, &d121)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d121.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d122)
			}
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d123)
			}
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d123)
			var d124 scm.JITValueDesc
			if d122.Loc == scm.LocImm && d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() + d123.Imm.Int())}
			} else if d123.Loc == scm.LocImm && d123.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r132, d122.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d124)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d123.Reg}
				ctx.BindReg(d123.Reg, &d124)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d123.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d124)
			} else if d123.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(scratch, d122.Reg)
				if d123.Imm.Int() >= -2147483648 && d123.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d123.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d124)
			} else {
				r133 := ctx.AllocRegExcept(d122.Reg, d123.Reg)
				ctx.W.EmitMovRegReg(r133, d122.Reg)
				ctx.W.EmitAddInt64(r133, d123.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d124)
			}
			if d124.Loc == scm.LocReg && d122.Loc == scm.LocReg && d124.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			ctx.FreeDesc(&d123)
			ctx.EnsureDesc(&d41)
			d125 := d41
			_ = d125
			r134 := d41.Loc == scm.LocReg
			r135 := d41.Reg
			if r134 { ctx.ProtectReg(r135) }
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl42)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d125.Imm.Int()))))}
			} else {
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r136, d125.Reg)
				ctx.W.EmitShlRegImm8(r136, 32)
				ctx.W.EmitShrRegImm8(r136, 32)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d126)
			}
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r137, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
				ctx.BindReg(r137, &d127)
			}
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d127.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d127.Reg)
				ctx.W.EmitShlRegImm8(r138, 56)
				ctx.W.EmitShrRegImm8(r138, 56)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d128)
			}
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d128)
			var d129 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() * d128.Imm.Int())}
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d126.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d128.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(scratch, d126.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else {
				r139 := ctx.AllocRegExcept(d126.Reg, d128.Reg)
				ctx.W.EmitMovRegReg(r139, d126.Reg)
				ctx.W.EmitImulInt64(r139, d128.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d129)
			}
			if d129.Loc == scm.LocReg && d126.Loc == scm.LocReg && d129.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d128)
			var d130 scm.JITValueDesc
			r140 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r140, uint64(dataPtr))
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140, StackOff: int32(sliceLen)}
				ctx.BindReg(r140, &d130)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
				ctx.BindReg(r140, &d130)
			}
			ctx.BindReg(r140, &d130)
			ctx.EnsureDesc(&d129)
			var d131 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() / 64)}
			} else {
				r141 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r141, d129.Reg)
				ctx.W.EmitShrRegImm8(r141, 6)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d131)
			}
			if d131.Loc == scm.LocReg && d129.Loc == scm.LocReg && d131.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d131)
			r142 := ctx.AllocReg()
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d130)
			if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r142, uint64(d131.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r142, d131.Reg)
				ctx.W.EmitShlRegImm8(r142, 3)
			}
			if d130.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
				ctx.W.EmitAddInt64(r142, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r142, d130.Reg)
			}
			r143 := ctx.AllocRegExcept(r142)
			ctx.W.EmitMovRegMem(r143, r142, 0)
			ctx.FreeReg(r142)
			d132 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
			ctx.BindReg(r143, &d132)
			ctx.FreeDesc(&d131)
			ctx.EnsureDesc(&d129)
			var d133 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() % 64)}
			} else {
				r144 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r144, d129.Reg)
				ctx.W.EmitAndRegImm32(r144, 63)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d133)
			}
			if d133.Loc == scm.LocReg && d129.Loc == scm.LocReg && d133.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d133)
			var d134 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d132.Imm.Int()) << uint64(d133.Imm.Int())))}
			} else if d133.Loc == scm.LocImm {
				r145 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(r145, d132.Reg)
				ctx.W.EmitShlRegImm8(r145, uint8(d133.Imm.Int()))
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d134)
			} else {
				{
					shiftSrc := d132.Reg
					r146 := ctx.AllocRegExcept(d132.Reg)
					ctx.W.EmitMovRegReg(r146, d132.Reg)
					shiftSrc = r146
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d133.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d133.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d133.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d134)
				}
			}
			if d134.Loc == scm.LocReg && d132.Loc == scm.LocReg && d134.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.FreeDesc(&d133)
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d135)
			}
			d136 := d135
			ctx.EnsureDesc(&d136)
			if d136.Loc != scm.LocImm && d136.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			if d136.Loc == scm.LocImm {
				if d136.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d136.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.W.MarkLabel(lbl45)
			ctx.W.EmitJmp(lbl43)
			ctx.W.MarkLabel(lbl46)
			d137 := d134
			if d137.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d137)
			ctx.EmitStoreToStack(d137, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.FreeDesc(&d135)
			ctx.W.MarkLabel(lbl44)
			d138 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r148, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
				ctx.BindReg(r148, &d139)
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d139.Imm.Int()))))}
			} else {
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r149, d139.Reg)
				ctx.W.EmitShlRegImm8(r149, 56)
				ctx.W.EmitShrRegImm8(r149, 56)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d140)
			}
			ctx.FreeDesc(&d139)
			d141 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d140)
			var d142 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() - d140.Imm.Int())}
			} else if d140.Loc == scm.LocImm && d140.Imm.Int() == 0 {
				r150 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r150, d141.Reg)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d142)
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d140.Reg)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d142)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(scratch, d141.Reg)
				if d140.Imm.Int() >= -2147483648 && d140.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d140.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d142)
			} else {
				r151 := ctx.AllocRegExcept(d141.Reg, d140.Reg)
				ctx.W.EmitMovRegReg(r151, d141.Reg)
				ctx.W.EmitSubInt64(r151, d140.Reg)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d142)
			}
			if d142.Loc == scm.LocReg && d141.Loc == scm.LocReg && d142.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d142)
			var d143 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d138.Imm.Int()) >> uint64(d142.Imm.Int())))}
			} else if d142.Loc == scm.LocImm {
				r152 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r152, d138.Reg)
				ctx.W.EmitShrRegImm8(r152, uint8(d142.Imm.Int()))
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d143)
			} else {
				{
					shiftSrc := d138.Reg
					r153 := ctx.AllocRegExcept(d138.Reg)
					ctx.W.EmitMovRegReg(r153, d138.Reg)
					shiftSrc = r153
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d142.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d142.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d142.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d143)
				}
			}
			if d143.Loc == scm.LocReg && d138.Loc == scm.LocReg && d143.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d142)
			r154 := ctx.AllocReg()
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d143)
			if d143.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r154, d143)
			}
			ctx.W.EmitJmp(lbl41)
			ctx.W.MarkLabel(lbl43)
			d138 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d129)
			var d144 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() % 64)}
			} else {
				r155 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r155, d129.Reg)
				ctx.W.EmitAndRegImm32(r155, 63)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d144)
			}
			if d144.Loc == scm.LocReg && d129.Loc == scm.LocReg && d144.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r156, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
				ctx.BindReg(r156, &d145)
			}
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d145)
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d145.Imm.Int()))))}
			} else {
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r157, d145.Reg)
				ctx.W.EmitShlRegImm8(r157, 56)
				ctx.W.EmitShrRegImm8(r157, 56)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d146)
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
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() + d146.Imm.Int())}
			} else if d146.Loc == scm.LocImm && d146.Imm.Int() == 0 {
				r158 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r158, d144.Reg)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d147)
			} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d146.Reg}
				ctx.BindReg(d146.Reg, &d147)
			} else if d144.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d144.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d146.Reg)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d147)
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(scratch, d144.Reg)
				if d146.Imm.Int() >= -2147483648 && d146.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d146.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d146.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d147)
			} else {
				r159 := ctx.AllocRegExcept(d144.Reg, d146.Reg)
				ctx.W.EmitMovRegReg(r159, d144.Reg)
				ctx.W.EmitAddInt64(r159, d146.Reg)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d147)
			}
			if d147.Loc == scm.LocReg && d144.Loc == scm.LocReg && d147.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			ctx.FreeDesc(&d146)
			ctx.EnsureDesc(&d147)
			var d148 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d147.Imm.Int()) > uint64(64))}
			} else {
				r160 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitCmpRegImm32(d147.Reg, 64)
				ctx.W.EmitSetcc(r160, scm.CcA)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
				ctx.BindReg(r160, &d148)
			}
			ctx.FreeDesc(&d147)
			d149 := d148
			ctx.EnsureDesc(&d149)
			if d149.Loc != scm.LocImm && d149.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d149.Loc == scm.LocImm {
				if d149.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d149.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl48)
				ctx.W.EmitJmp(lbl49)
			}
			ctx.W.MarkLabel(lbl48)
			ctx.W.EmitJmp(lbl47)
			ctx.W.MarkLabel(lbl49)
			d150 := d134
			if d150.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d150)
			ctx.EmitStoreToStack(d150, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.FreeDesc(&d148)
			ctx.W.MarkLabel(lbl47)
			d138 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d129)
			var d151 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r161, d129.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d151)
			}
			if d151.Loc == scm.LocReg && d129.Loc == scm.LocReg && d151.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(scratch, d151.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d152)
			}
			if d152.Loc == scm.LocReg && d151.Loc == scm.LocReg && d152.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			ctx.EnsureDesc(&d152)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d130)
			if d152.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d152.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d152.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d130.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d130.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d153 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d153)
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d129)
			var d154 scm.JITValueDesc
			if d129.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r164, d129.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d154)
			}
			if d154.Loc == scm.LocReg && d129.Loc == scm.LocReg && d154.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d129)
			d155 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d154)
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() - d154.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r165, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d156)
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
				r166 := ctx.AllocRegExcept(d155.Reg, d154.Reg)
				ctx.W.EmitMovRegReg(r166, d155.Reg)
				ctx.W.EmitSubInt64(r166, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d156)
			}
			if d156.Loc == scm.LocReg && d155.Loc == scm.LocReg && d156.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d156)
			var d157 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d153.Imm.Int()) >> uint64(d156.Imm.Int())))}
			} else if d156.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r167, d153.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d156.Imm.Int()))
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d157)
			} else {
				{
					shiftSrc := d153.Reg
					r168 := ctx.AllocRegExcept(d153.Reg)
					ctx.W.EmitMovRegReg(r168, d153.Reg)
					shiftSrc = r168
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
			if d157.Loc == scm.LocReg && d153.Loc == scm.LocReg && d157.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d153)
			ctx.FreeDesc(&d156)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() | d157.Imm.Int())}
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
				ctx.BindReg(d157.Reg, &d158)
			} else if d157.Loc == scm.LocImm && d157.Imm.Int() == 0 {
				r169 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r169, d134.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d158)
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d157.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else if d157.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r170, d134.Reg)
				if d157.Imm.Int() >= -2147483648 && d157.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r170, int32(d157.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d157.Imm.Int()))
					ctx.W.EmitOrInt64(r170, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d158)
			} else {
				r171 := ctx.AllocRegExcept(d134.Reg, d157.Reg)
				ctx.W.EmitMovRegReg(r171, d134.Reg)
				ctx.W.EmitOrInt64(r171, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d158)
			}
			if d158.Loc == scm.LocReg && d134.Loc == scm.LocReg && d158.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d157)
			d159 := d158
			if d159.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d159)
			ctx.EmitStoreToStack(d159, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl41)
			d160 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d160)
			ctx.BindReg(r154, &d160)
			if r134 { ctx.UnprotectReg(r135) }
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d160)
			var d161 scm.JITValueDesc
			if d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d160.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d161)
			}
			ctx.FreeDesc(&d160)
			var d162 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r173 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r173, thisptr.Reg, off)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
				ctx.BindReg(r173, &d162)
			}
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d162)
			var d163 scm.JITValueDesc
			if d161.Loc == scm.LocImm && d162.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d161.Imm.Int() + d162.Imm.Int())}
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				r174 := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegReg(r174, d161.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d163)
			} else if d161.Loc == scm.LocImm && d161.Imm.Int() == 0 {
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d162.Reg}
				ctx.BindReg(d162.Reg, &d163)
			} else if d161.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d161.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d162.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegReg(scratch, d161.Reg)
				if d162.Imm.Int() >= -2147483648 && d162.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d162.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d162.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else {
				r175 := ctx.AllocRegExcept(d161.Reg, d162.Reg)
				ctx.W.EmitMovRegReg(r175, d161.Reg)
				ctx.W.EmitAddInt64(r175, d162.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d163)
			}
			if d163.Loc == scm.LocReg && d161.Loc == scm.LocReg && d163.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.FreeDesc(&d162)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d163)
			var d165 scm.JITValueDesc
			if d124.Loc == scm.LocImm && d163.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + d163.Imm.Int())}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegReg(r176, d124.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d165)
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
				ctx.BindReg(d163.Reg, &d165)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d124.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d163.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegReg(scratch, d124.Reg)
				if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d163.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else {
				r177 := ctx.AllocRegExcept(d124.Reg, d163.Reg)
				ctx.W.EmitMovRegReg(r177, d124.Reg)
				ctx.W.EmitAddInt64(r177, d163.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d165)
			}
			if d165.Loc == scm.LocReg && d124.Loc == scm.LocReg && d165.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d165)
			var d167 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r178 := ctx.AllocReg()
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r178, fieldAddr)
				ctx.W.EmitMovRegMem64(r179, fieldAddr+8)
				d167 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r178, Reg2: r179}
				ctx.BindReg(r178, &d167)
				ctx.BindReg(r179, &d167)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r180, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r181, thisptr.Reg, off+8)
				d167 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d167)
				ctx.BindReg(r181, &d167)
			}
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d165)
			r182 := ctx.AllocReg()
			r183 := ctx.AllocRegExcept(r182)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d165)
			if d167.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d167.Imm.Int()))
			} else if d167.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r182, d167.Reg)
			} else {
				ctx.W.EmitMovRegReg(r182, d167.Reg)
			}
			if d124.Loc == scm.LocImm {
				if d124.Imm.Int() != 0 {
					if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r182, int32(d124.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
						ctx.W.EmitAddInt64(r182, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r182, d124.Reg)
			}
			if d165.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r183, uint64(d165.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r183, d165.Reg)
			}
			if d124.Loc == scm.LocImm {
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r183, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitSubInt64(r183, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r183, d124.Reg)
			}
			d168 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
			ctx.BindReg(r182, &d168)
			ctx.BindReg(r183, &d168)
			ctx.FreeDesc(&d124)
			ctx.FreeDesc(&d165)
			d169 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d169)
			ctx.BindReg(r1, &d169)
			d170 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d168}, 2)
			ctx.EmitMovPairToResult(&d170, &d169)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			var d171 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r184 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r184, thisptr.Reg, off)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r184}
				ctx.BindReg(r184, &d171)
			}
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d171)
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d171.Imm.Int()))))}
			} else {
				r185 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r185, d171.Reg)
				ctx.W.EmitShlRegImm8(r185, 32)
				ctx.W.EmitShrRegImm8(r185, 32)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d172)
			}
			ctx.FreeDesc(&d171)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d41.Imm.Int()) == uint64(d172.Imm.Int()))}
			} else if d172.Loc == scm.LocImm {
				r186 := ctx.AllocRegExcept(d41.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d41.Reg, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitCmpInt64(d41.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r186, scm.CcE)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r186}
				ctx.BindReg(r186, &d173)
			} else if d41.Loc == scm.LocImm {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d172.Reg)
				ctx.W.EmitSetcc(r187, scm.CcE)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r187}
				ctx.BindReg(r187, &d173)
			} else {
				r188 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitCmpInt64(d41.Reg, d172.Reg)
				ctx.W.EmitSetcc(r188, scm.CcE)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r188}
				ctx.BindReg(r188, &d173)
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d172)
			d174 := d173
			ctx.EnsureDesc(&d174)
			if d174.Loc != scm.LocImm && d174.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			if d174.Loc == scm.LocImm {
				if d174.Imm.Bool() {
					ctx.W.EmitJmp(lbl51)
				} else {
					ctx.W.EmitJmp(lbl52)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d174.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl51)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.W.MarkLabel(lbl51)
			ctx.W.EmitJmp(lbl50)
			ctx.W.MarkLabel(lbl52)
			ctx.W.EmitJmp(lbl16)
			ctx.FreeDesc(&d173)
			ctx.W.MarkLabel(lbl29)
			ctx.EnsureDesc(&idxInt)
			d175 := idxInt
			_ = d175
			r189 := idxInt.Loc == scm.LocReg
			r190 := idxInt.Reg
			if r189 { ctx.ProtectReg(r190) }
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl54)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d175.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, d175.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d176)
			}
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r192, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r192}
				ctx.BindReg(r192, &d177)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d177)
			var d178 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d177.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, d177.Reg)
				ctx.W.EmitShlRegImm8(r193, 56)
				ctx.W.EmitShrRegImm8(r193, 56)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d178)
			}
			ctx.FreeDesc(&d177)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d178)
			var d179 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() * d178.Imm.Int())}
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d179)
			} else if d178.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(scratch, d176.Reg)
				if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d178.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d179)
			} else {
				r194 := ctx.AllocRegExcept(d176.Reg, d178.Reg)
				ctx.W.EmitMovRegReg(r194, d176.Reg)
				ctx.W.EmitImulInt64(r194, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d179)
			}
			if d179.Loc == scm.LocReg && d176.Loc == scm.LocReg && d179.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d178)
			var d180 scm.JITValueDesc
			r195 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r195, uint64(dataPtr))
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195, StackOff: int32(sliceLen)}
				ctx.BindReg(r195, &d180)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r195, thisptr.Reg, off)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195}
				ctx.BindReg(r195, &d180)
			}
			ctx.BindReg(r195, &d180)
			ctx.EnsureDesc(&d179)
			var d181 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() / 64)}
			} else {
				r196 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r196, d179.Reg)
				ctx.W.EmitShrRegImm8(r196, 6)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d181)
			}
			if d181.Loc == scm.LocReg && d179.Loc == scm.LocReg && d181.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d181)
			r197 := ctx.AllocReg()
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d180)
			if d181.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r197, uint64(d181.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r197, d181.Reg)
				ctx.W.EmitShlRegImm8(r197, 3)
			}
			if d180.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
				ctx.W.EmitAddInt64(r197, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r197, d180.Reg)
			}
			r198 := ctx.AllocRegExcept(r197)
			ctx.W.EmitMovRegMem(r198, r197, 0)
			ctx.FreeReg(r197)
			d182 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
			ctx.BindReg(r198, &d182)
			ctx.FreeDesc(&d181)
			ctx.EnsureDesc(&d179)
			var d183 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() % 64)}
			} else {
				r199 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r199, d179.Reg)
				ctx.W.EmitAndRegImm32(r199, 63)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d183)
			}
			if d183.Loc == scm.LocReg && d179.Loc == scm.LocReg && d183.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d182.Imm.Int()) << uint64(d183.Imm.Int())))}
			} else if d183.Loc == scm.LocImm {
				r200 := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(r200, d182.Reg)
				ctx.W.EmitShlRegImm8(r200, uint8(d183.Imm.Int()))
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d184)
			} else {
				{
					shiftSrc := d182.Reg
					r201 := ctx.AllocRegExcept(d182.Reg)
					ctx.W.EmitMovRegReg(r201, d182.Reg)
					shiftSrc = r201
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d183.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d183.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d183.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d184)
				}
			}
			if d184.Loc == scm.LocReg && d182.Loc == scm.LocReg && d184.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d182)
			ctx.FreeDesc(&d183)
			var d185 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d185)
			}
			d186 := d185
			ctx.EnsureDesc(&d186)
			if d186.Loc != scm.LocImm && d186.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d186.Loc == scm.LocImm {
				if d186.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d186.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
			}
			ctx.W.MarkLabel(lbl57)
			ctx.W.EmitJmp(lbl55)
			ctx.W.MarkLabel(lbl58)
			d187 := d184
			if d187.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d187)
			ctx.EmitStoreToStack(d187, 32)
			ctx.W.EmitJmp(lbl56)
			ctx.FreeDesc(&d185)
			ctx.W.MarkLabel(lbl56)
			d188 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d189 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r203, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r203}
				ctx.BindReg(r203, &d189)
			}
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d189)
			var d190 scm.JITValueDesc
			if d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d189.Imm.Int()))))}
			} else {
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r204, d189.Reg)
				ctx.W.EmitShlRegImm8(r204, 56)
				ctx.W.EmitShrRegImm8(r204, 56)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d190)
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
				r205 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r205, d191.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d192)
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
				r206 := ctx.AllocRegExcept(d191.Reg, d190.Reg)
				ctx.W.EmitMovRegReg(r206, d191.Reg)
				ctx.W.EmitSubInt64(r206, d190.Reg)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d192)
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
				r207 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r207, d188.Reg)
				ctx.W.EmitShrRegImm8(r207, uint8(d192.Imm.Int()))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d193)
			} else {
				{
					shiftSrc := d188.Reg
					r208 := ctx.AllocRegExcept(d188.Reg)
					ctx.W.EmitMovRegReg(r208, d188.Reg)
					shiftSrc = r208
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
			r209 := ctx.AllocReg()
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d193)
			if d193.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r209, d193)
			}
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl55)
			d188 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d179)
			var d194 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() % 64)}
			} else {
				r210 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r210, d179.Reg)
				ctx.W.EmitAndRegImm32(r210, 63)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d194)
			}
			if d194.Loc == scm.LocReg && d179.Loc == scm.LocReg && d194.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r211, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
				ctx.BindReg(r211, &d195)
			}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r212 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r212, d195.Reg)
				ctx.W.EmitShlRegImm8(r212, 56)
				ctx.W.EmitShrRegImm8(r212, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d196)
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
				r213 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(r213, d194.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d197)
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
				r214 := ctx.AllocRegExcept(d194.Reg, d196.Reg)
				ctx.W.EmitMovRegReg(r214, d194.Reg)
				ctx.W.EmitAddInt64(r214, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d197)
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
				r215 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitCmpRegImm32(d197.Reg, 64)
				ctx.W.EmitSetcc(r215, scm.CcA)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r215}
				ctx.BindReg(r215, &d198)
			}
			ctx.FreeDesc(&d197)
			d199 := d198
			ctx.EnsureDesc(&d199)
			if d199.Loc != scm.LocImm && d199.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d199.Loc == scm.LocImm {
				if d199.Imm.Bool() {
					ctx.W.EmitJmp(lbl60)
				} else {
					ctx.W.EmitJmp(lbl61)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d199.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
			}
			ctx.W.MarkLabel(lbl60)
			ctx.W.EmitJmp(lbl59)
			ctx.W.MarkLabel(lbl61)
			d200 := d184
			if d200.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d200)
			ctx.EmitStoreToStack(d200, 32)
			ctx.W.EmitJmp(lbl56)
			ctx.FreeDesc(&d198)
			ctx.W.MarkLabel(lbl59)
			d188 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d179)
			var d201 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() / 64)}
			} else {
				r216 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r216, d179.Reg)
				ctx.W.EmitShrRegImm8(r216, 6)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d201)
			}
			if d201.Loc == scm.LocReg && d179.Loc == scm.LocReg && d201.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(scratch, d201.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			}
			if d202.Loc == scm.LocReg && d201.Loc == scm.LocReg && d202.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			ctx.EnsureDesc(&d202)
			r217 := ctx.AllocReg()
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d180)
			if d202.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r217, uint64(d202.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r217, d202.Reg)
				ctx.W.EmitShlRegImm8(r217, 3)
			}
			if d180.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
				ctx.W.EmitAddInt64(r217, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r217, d180.Reg)
			}
			r218 := ctx.AllocRegExcept(r217)
			ctx.W.EmitMovRegMem(r218, r217, 0)
			ctx.FreeReg(r217)
			d203 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
			ctx.BindReg(r218, &d203)
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d179)
			var d204 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d179.Imm.Int() % 64)}
			} else {
				r219 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r219, d179.Reg)
				ctx.W.EmitAndRegImm32(r219, 63)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d204)
			}
			if d204.Loc == scm.LocReg && d179.Loc == scm.LocReg && d204.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			d205 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d204)
			var d206 scm.JITValueDesc
			if d205.Loc == scm.LocImm && d204.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d205.Imm.Int() - d204.Imm.Int())}
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegReg(r220, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d206)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d205.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d204.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else if d204.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegReg(scratch, d205.Reg)
				if d204.Imm.Int() >= -2147483648 && d204.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d204.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d204.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else {
				r221 := ctx.AllocRegExcept(d205.Reg, d204.Reg)
				ctx.W.EmitMovRegReg(r221, d205.Reg)
				ctx.W.EmitSubInt64(r221, d204.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d206)
			}
			if d206.Loc == scm.LocReg && d205.Loc == scm.LocReg && d206.Reg == d205.Reg {
				ctx.TransferReg(d205.Reg)
				d205.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d203.Loc == scm.LocImm && d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d203.Imm.Int()) >> uint64(d206.Imm.Int())))}
			} else if d206.Loc == scm.LocImm {
				r222 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(r222, d203.Reg)
				ctx.W.EmitShrRegImm8(r222, uint8(d206.Imm.Int()))
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d207)
			} else {
				{
					shiftSrc := d203.Reg
					r223 := ctx.AllocRegExcept(d203.Reg)
					ctx.W.EmitMovRegReg(r223, d203.Reg)
					shiftSrc = r223
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d206.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d206.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d206.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d207)
				}
			}
			if d207.Loc == scm.LocReg && d203.Loc == scm.LocReg && d207.Reg == d203.Reg {
				ctx.TransferReg(d203.Reg)
				d203.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d203)
			ctx.FreeDesc(&d206)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d207)
			var d208 scm.JITValueDesc
			if d184.Loc == scm.LocImm && d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() | d207.Imm.Int())}
			} else if d184.Loc == scm.LocImm && d184.Imm.Int() == 0 {
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d207.Reg}
				ctx.BindReg(d207.Reg, &d208)
			} else if d207.Loc == scm.LocImm && d207.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r224, d184.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d208)
			} else if d184.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d184.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d207.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			} else if d207.Loc == scm.LocImm {
				r225 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r225, d184.Reg)
				if d207.Imm.Int() >= -2147483648 && d207.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r225, int32(d207.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d207.Imm.Int()))
					ctx.W.EmitOrInt64(r225, scm.RegR11)
				}
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d208)
			} else {
				r226 := ctx.AllocRegExcept(d184.Reg, d207.Reg)
				ctx.W.EmitMovRegReg(r226, d184.Reg)
				ctx.W.EmitOrInt64(r226, d207.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d208)
			}
			if d208.Loc == scm.LocReg && d184.Loc == scm.LocReg && d208.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d207)
			d209 := d208
			if d209.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d209)
			ctx.EmitStoreToStack(d209, 32)
			ctx.W.EmitJmp(lbl56)
			ctx.W.MarkLabel(lbl53)
			d210 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
			ctx.BindReg(r209, &d210)
			ctx.BindReg(r209, &d210)
			if r189 { ctx.UnprotectReg(r190) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d210)
			var d211 scm.JITValueDesc
			if d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d210.Imm.Int()))))}
			} else {
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r227, d210.Reg)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d211)
			}
			ctx.FreeDesc(&d210)
			var d212 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r228 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r228, thisptr.Reg, off)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r228}
				ctx.BindReg(r228, &d212)
			}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			var d213 scm.JITValueDesc
			if d211.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d211.Imm.Int() + d212.Imm.Int())}
			} else if d212.Loc == scm.LocImm && d212.Imm.Int() == 0 {
				r229 := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegReg(r229, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d213)
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d212.Reg}
				ctx.BindReg(d212.Reg, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d211.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegReg(scratch, d211.Reg)
				if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d212.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r230 := ctx.AllocRegExcept(d211.Reg, d212.Reg)
				ctx.W.EmitMovRegReg(r230, d211.Reg)
				ctx.W.EmitAddInt64(r230, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d213)
			}
			if d213.Loc == scm.LocReg && d211.Loc == scm.LocReg && d213.Reg == d211.Reg {
				ctx.TransferReg(d211.Reg)
				d211.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.FreeDesc(&d212)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d213.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d214)
			}
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d83)
			var d215 scm.JITValueDesc
			if d83.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d83.Imm.Int()))))}
			} else {
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r232, d83.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d215)
			}
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d214)
			var d216 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() + d214.Imm.Int())}
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegReg(r233, d83.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d216)
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
				ctx.BindReg(d214.Reg, &d216)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d216)
			} else if d214.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegReg(scratch, d83.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d214.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d216)
			} else {
				r234 := ctx.AllocRegExcept(d83.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r234, d83.Reg)
				ctx.W.EmitAddInt64(r234, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d216)
			}
			if d216.Loc == scm.LocReg && d83.Loc == scm.LocReg && d216.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d216.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d217)
			}
			ctx.FreeDesc(&d216)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d217)
			r236 := ctx.AllocReg()
			r237 := ctx.AllocRegExcept(r236)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d217)
			if d167.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d167.Imm.Int()))
			} else if d167.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r236, d167.Reg)
			} else {
				ctx.W.EmitMovRegReg(r236, d167.Reg)
			}
			if d215.Loc == scm.LocImm {
				if d215.Imm.Int() != 0 {
					if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r236, int32(d215.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
						ctx.W.EmitAddInt64(r236, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r236, d215.Reg)
			}
			if d217.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r237, uint64(d217.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r237, d217.Reg)
			}
			if d215.Loc == scm.LocImm {
				if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r237, int32(d215.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
					ctx.W.EmitSubInt64(r237, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r237, d215.Reg)
			}
			d218 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r236, Reg2: r237}
			ctx.BindReg(r236, &d218)
			ctx.BindReg(r237, &d218)
			ctx.FreeDesc(&d215)
			ctx.FreeDesc(&d217)
			d219 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d219)
			ctx.BindReg(r1, &d219)
			d220 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d218}, 2)
			ctx.EmitMovPairToResult(&d220, &d219)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl28)
			var d221 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r238, thisptr.Reg, off)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r238}
				ctx.BindReg(r238, &d221)
			}
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d221)
			var d222 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d83.Imm.Int()) == uint64(d221.Imm.Int()))}
			} else if d221.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d83.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d83.Reg, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitCmpInt64(d83.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d222)
			} else if d83.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d221.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d222)
			} else {
				r241 := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitCmpInt64(d83.Reg, d221.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d222)
			}
			ctx.FreeDesc(&d83)
			ctx.FreeDesc(&d221)
			d223 := d222
			ctx.EnsureDesc(&d223)
			if d223.Loc != scm.LocImm && d223.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			lbl64 := ctx.W.ReserveLabel()
			if d223.Loc == scm.LocImm {
				if d223.Imm.Bool() {
					ctx.W.EmitJmp(lbl63)
				} else {
					ctx.W.EmitJmp(lbl64)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d223.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl63)
				ctx.W.EmitJmp(lbl64)
			}
			ctx.W.MarkLabel(lbl63)
			ctx.W.EmitJmp(lbl62)
			ctx.W.MarkLabel(lbl64)
			ctx.W.EmitJmp(lbl29)
			ctx.FreeDesc(&d222)
			ctx.W.MarkLabel(lbl50)
			d224 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d224)
			ctx.BindReg(r1, &d224)
			ctx.W.EmitMakeNil(d224)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl62)
			d225 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d225)
			ctx.BindReg(r1, &d225)
			ctx.W.EmitMakeNil(d225)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d226 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d226)
			ctx.BindReg(r1, &d226)
			ctx.EmitMovPairToResult(&d226, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r5, int32(40))
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
