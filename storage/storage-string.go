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
					ctx.W.MarkLabel(lbl4)
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
			}
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
					ctx.W.MarkLabel(lbl10)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl11)
			d14 := d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
			d15 := d11
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d12)
			ctx.W.MarkLabel(lbl9)
			d16 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d17)
			}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d17.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d17.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d18)
			}
			ctx.FreeDesc(&d17)
			d19 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d18)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() - d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r20, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d20)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d20)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(scratch, d19.Reg)
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d18.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d20)
			} else {
				r21 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
				ctx.W.EmitMovRegReg(r21, d19.Reg)
				ctx.W.EmitSubInt64(r21, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d20)
			}
			if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d20)
			var d21 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d16.Imm.Int()) >> uint64(d20.Imm.Int())))}
			} else if d20.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r22, d16.Reg)
				ctx.W.EmitShrRegImm8(r22, uint8(d20.Imm.Int()))
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d21)
			} else {
				{
					shiftSrc := d16.Reg
					r23 := ctx.AllocRegExcept(d16.Reg)
					ctx.W.EmitMovRegReg(r23, d16.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d20.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d20.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d20.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d21)
				}
			}
			if d21.Loc == scm.LocReg && d16.Loc == scm.LocReg && d21.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.FreeDesc(&d20)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			if d21.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r24, d21)
			}
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl8)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d22 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r25, d6.Reg)
				ctx.W.EmitAndRegImm32(r25, 63)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d22)
			}
			if d22.Loc == scm.LocReg && d6.Loc == scm.LocReg && d22.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			var d23 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d23)
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d23.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d23.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d24)
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
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r28, d22.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d25)
			} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
				ctx.BindReg(d24.Reg, &d25)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d22.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r29 := ctx.AllocRegExcept(d22.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r29, d22.Reg)
				ctx.W.EmitAddInt64(r29, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d25)
			}
			if d25.Loc == scm.LocReg && d22.Loc == scm.LocReg && d25.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d25.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitCmpRegImm32(d25.Reg, 64)
				ctx.W.EmitSetcc(r30, scm.CcA)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d26)
			}
			ctx.FreeDesc(&d25)
			d27 := d26
			ctx.EnsureDesc(&d27)
			if d27.Loc != scm.LocImm && d27.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.MarkLabel(lbl14)
			d28 := d11
			if d28.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 0)
					ctx.W.EmitJmp(lbl9)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl14)
			d29 := d11
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d26)
			ctx.W.MarkLabel(lbl12)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d6)
			var d30 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r31, d6.Reg)
				ctx.W.EmitShrRegImm8(r31, 6)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d30)
			}
			if d30.Loc == scm.LocReg && d6.Loc == scm.LocReg && d30.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d31)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d7)
			if d31.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r32, uint64(d31.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r32, d31.Reg)
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
			d32 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d32)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d6)
			var d33 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r34, d6.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d33)
			}
			if d33.Loc == scm.LocReg && d6.Loc == scm.LocReg && d33.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			d34 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d33)
			var d35 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() - d33.Imm.Int())}
			} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(r35, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d35)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(scratch, d34.Reg)
				if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d33.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else {
				r36 := ctx.AllocRegExcept(d34.Reg, d33.Reg)
				ctx.W.EmitMovRegReg(r36, d34.Reg)
				ctx.W.EmitSubInt64(r36, d33.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d35)
			}
			if d35.Loc == scm.LocReg && d34.Loc == scm.LocReg && d35.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d32.Imm.Int()) >> uint64(d35.Imm.Int())))}
			} else if d35.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r37, d32.Reg)
				ctx.W.EmitShrRegImm8(r37, uint8(d35.Imm.Int()))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d36)
			} else {
				{
					shiftSrc := d32.Reg
					r38 := ctx.AllocRegExcept(d32.Reg)
					ctx.W.EmitMovRegReg(r38, d32.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d35.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d35.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d35.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d36)
				}
			}
			if d36.Loc == scm.LocReg && d32.Loc == scm.LocReg && d36.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() | d36.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
				ctx.BindReg(d36.Reg, &d37)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r39, d11.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d37)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			} else if d36.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r40, d11.Reg)
				if d36.Imm.Int() >= -2147483648 && d36.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r40, int32(d36.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d36.Imm.Int()))
					ctx.W.EmitOrInt64(r40, scm.RegR11)
				}
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d37)
			} else {
				r41 := ctx.AllocRegExcept(d11.Reg, d36.Reg)
				ctx.W.EmitMovRegReg(r41, d11.Reg)
				ctx.W.EmitOrInt64(r41, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d37)
			}
			if d37.Loc == scm.LocReg && d11.Loc == scm.LocReg && d37.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			d38 := d37
			if d38.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d38)
			ctx.EmitStoreToStack(d38, 0)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl6)
			d39 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d39)
			ctx.BindReg(r24, &d39)
			if r3 { ctx.UnprotectReg(r4) }
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d39.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d40)
			}
			ctx.FreeDesc(&d39)
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d41)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() + d41.Imm.Int())}
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r44, d40.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d42)
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else {
				r45 := ctx.AllocRegExcept(d40.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r45, d40.Reg)
				ctx.W.EmitAddInt64(r45, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d42)
			}
			if d42.Loc == scm.LocReg && d40.Loc == scm.LocReg && d42.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d42.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r46, d42.Reg)
				ctx.W.EmitShlRegImm8(r46, 32)
				ctx.W.EmitShrRegImm8(r46, 32)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d43)
			}
			ctx.FreeDesc(&d42)
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d44)
			}
			d45 := d44
			ctx.EnsureDesc(&d45)
			if d45.Loc != scm.LocImm && d45.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.MarkLabel(lbl17)
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d44)
			ctx.W.MarkLabel(lbl2)
			ctx.EnsureDesc(&idxInt)
			d46 := idxInt
			_ = d46
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl20)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d46.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d46.Reg)
				ctx.W.EmitShlRegImm8(r50, 32)
				ctx.W.EmitShrRegImm8(r50, 32)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d47)
			}
			var d48 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r51, thisptr.Reg, off)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d48)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d48.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d48.Reg)
				ctx.W.EmitShlRegImm8(r52, 56)
				ctx.W.EmitShrRegImm8(r52, 56)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d49)
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() * d49.Imm.Int())}
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r53 := ctx.AllocRegExcept(d47.Reg, d49.Reg)
				ctx.W.EmitMovRegReg(r53, d47.Reg)
				ctx.W.EmitImulInt64(r53, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d50)
			}
			if d50.Loc == scm.LocReg && d47.Loc == scm.LocReg && d50.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d49)
			var d51 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r54, uint64(dataPtr))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d51)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d51)
			}
			ctx.BindReg(r54, &d51)
			ctx.EnsureDesc(&d50)
			var d52 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r55, d50.Reg)
				ctx.W.EmitShrRegImm8(r55, 6)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d52)
			}
			if d52.Loc == scm.LocReg && d50.Loc == scm.LocReg && d52.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d52)
			r56 := ctx.AllocReg()
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d51)
			if d52.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r56, uint64(d52.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r56, d52.Reg)
				ctx.W.EmitShlRegImm8(r56, 3)
			}
			if d51.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
				ctx.W.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r56, d51.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.W.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d53 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d53)
			ctx.FreeDesc(&d52)
			ctx.EnsureDesc(&d50)
			var d54 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r58, d50.Reg)
				ctx.W.EmitAndRegImm32(r58, 63)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d54)
			}
			if d54.Loc == scm.LocReg && d50.Loc == scm.LocReg && d54.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) << uint64(d54.Imm.Int())))}
			} else if d54.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r59, d53.Reg)
				ctx.W.EmitShlRegImm8(r59, uint8(d54.Imm.Int()))
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d55)
			} else {
				{
					shiftSrc := d53.Reg
					r60 := ctx.AllocRegExcept(d53.Reg)
					ctx.W.EmitMovRegReg(r60, d53.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d54.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d54.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d54.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d55)
				}
			}
			if d55.Loc == scm.LocReg && d53.Loc == scm.LocReg && d55.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.FreeDesc(&d54)
			var d56 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d56)
			}
			d57 := d56
			ctx.EnsureDesc(&d57)
			if d57.Loc != scm.LocImm && d57.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d57.Loc == scm.LocImm {
				if d57.Imm.Bool() {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.MarkLabel(lbl24)
			d58 := d55
			if d58.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d58)
			ctx.EmitStoreToStack(d58, 8)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d57.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl24)
			d59 := d55
			if d59.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			ctx.EmitStoreToStack(d59, 8)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d56)
			ctx.W.MarkLabel(lbl22)
			d60 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d61 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d61)
			}
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d61)
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d61.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d61.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d62)
			}
			ctx.FreeDesc(&d61)
			d63 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d62)
			var d64 scm.JITValueDesc
			if d63.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d63.Imm.Int() - d62.Imm.Int())}
			} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegReg(r64, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d64)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d62.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else if d62.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegReg(scratch, d63.Reg)
				if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d62.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d62.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else {
				r65 := ctx.AllocRegExcept(d63.Reg, d62.Reg)
				ctx.W.EmitMovRegReg(r65, d63.Reg)
				ctx.W.EmitSubInt64(r65, d62.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d64)
			}
			if d64.Loc == scm.LocReg && d63.Loc == scm.LocReg && d64.Reg == d63.Reg {
				ctx.TransferReg(d63.Reg)
				d63.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d64)
			var d65 scm.JITValueDesc
			if d60.Loc == scm.LocImm && d64.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) >> uint64(d64.Imm.Int())))}
			} else if d64.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegReg(r66, d60.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d64.Imm.Int()))
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d65)
			} else {
				{
					shiftSrc := d60.Reg
					r67 := ctx.AllocRegExcept(d60.Reg)
					ctx.W.EmitMovRegReg(r67, d60.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d64.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d64.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d64.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d65)
				}
			}
			if d65.Loc == scm.LocReg && d60.Loc == scm.LocReg && d65.Reg == d60.Reg {
				ctx.TransferReg(d60.Reg)
				d60.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			ctx.FreeDesc(&d64)
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d65)
			if d65.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r68, d65)
			}
			ctx.W.EmitJmp(lbl19)
			ctx.W.MarkLabel(lbl21)
			d60 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d50)
			var d66 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r69, d50.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d66)
			}
			if d66.Loc == scm.LocReg && d50.Loc == scm.LocReg && d66.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d67)
			}
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d67)
			var d68 scm.JITValueDesc
			if d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d67.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d67.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d68)
			}
			ctx.FreeDesc(&d67)
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d68)
			var d69 scm.JITValueDesc
			if d66.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + d68.Imm.Int())}
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r72, d66.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d69)
			} else if d66.Loc == scm.LocImm && d66.Imm.Int() == 0 {
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d68.Reg}
				ctx.BindReg(d68.Reg, &d69)
			} else if d66.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d66.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d68.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d69)
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(scratch, d66.Reg)
				if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d68.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d68.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d69)
			} else {
				r73 := ctx.AllocRegExcept(d66.Reg, d68.Reg)
				ctx.W.EmitMovRegReg(r73, d66.Reg)
				ctx.W.EmitAddInt64(r73, d68.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d69)
			}
			if d69.Loc == scm.LocReg && d66.Loc == scm.LocReg && d69.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d66)
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d69)
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d69.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitCmpRegImm32(d69.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d70)
			}
			ctx.FreeDesc(&d69)
			d71 := d70
			ctx.EnsureDesc(&d71)
			if d71.Loc != scm.LocImm && d71.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d71.Loc == scm.LocImm {
				if d71.Imm.Bool() {
					ctx.W.MarkLabel(lbl26)
					ctx.W.EmitJmp(lbl25)
				} else {
					ctx.W.MarkLabel(lbl27)
			d72 := d55
			if d72.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d72)
			ctx.EmitStoreToStack(d72, 8)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d71.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl27)
			d73 := d55
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 8)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d70)
			ctx.W.MarkLabel(lbl25)
			d60 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d50)
			var d74 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r75, d50.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d74)
			}
			if d74.Loc == scm.LocReg && d50.Loc == scm.LocReg && d74.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d74)
			var d75 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(scratch, d74.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			ctx.EnsureDesc(&d75)
			r76 := ctx.AllocReg()
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d51)
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d75.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d75.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d51.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d51.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d76 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d76)
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d50)
			var d77 scm.JITValueDesc
			if d50.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d50.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r78, d50.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d77)
			}
			if d77.Loc == scm.LocReg && d50.Loc == scm.LocReg && d77.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			d78 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d77)
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() - d77.Imm.Int())}
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r79, d78.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d79)
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d77.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d79)
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(scratch, d78.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d79)
			} else {
				r80 := ctx.AllocRegExcept(d78.Reg, d77.Reg)
				ctx.W.EmitMovRegReg(r80, d78.Reg)
				ctx.W.EmitSubInt64(r80, d77.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d79)
			}
			if d79.Loc == scm.LocReg && d78.Loc == scm.LocReg && d79.Reg == d78.Reg {
				ctx.TransferReg(d78.Reg)
				d78.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d79)
			var d80 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d76.Imm.Int()) >> uint64(d79.Imm.Int())))}
			} else if d79.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(r81, d76.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d79.Imm.Int()))
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d80)
			} else {
				{
					shiftSrc := d76.Reg
					r82 := ctx.AllocRegExcept(d76.Reg)
					ctx.W.EmitMovRegReg(r82, d76.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d79.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d79.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d79.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d80)
				}
			}
			if d80.Loc == scm.LocReg && d76.Loc == scm.LocReg && d80.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			ctx.FreeDesc(&d79)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d55.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d55.Imm.Int() | d80.Imm.Int())}
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d80.Reg}
				ctx.BindReg(d80.Reg, &d81)
			} else if d80.Loc == scm.LocImm && d80.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(r83, d55.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d81)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d55.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			} else if d80.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(r84, d55.Reg)
				if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d80.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d81)
			} else {
				r85 := ctx.AllocRegExcept(d55.Reg, d80.Reg)
				ctx.W.EmitMovRegReg(r85, d55.Reg)
				ctx.W.EmitOrInt64(r85, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d81)
			}
			if d81.Loc == scm.LocReg && d55.Loc == scm.LocReg && d81.Reg == d55.Reg {
				ctx.TransferReg(d55.Reg)
				d55.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			d82 := d81
			if d82.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d82)
			ctx.EmitStoreToStack(d82, 8)
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl19)
			d83 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d83)
			ctx.BindReg(r68, &d83)
			if r48 { ctx.UnprotectReg(r49) }
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d83)
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d83.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d84)
			}
			ctx.FreeDesc(&d83)
			var d85 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d85)
			}
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d85)
			var d86 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() + d85.Imm.Int())}
			} else if d85.Loc == scm.LocImm && d85.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r88, d84.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d86)
			} else if d84.Loc == scm.LocImm && d84.Imm.Int() == 0 {
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d85.Reg}
				ctx.BindReg(d85.Reg, &d86)
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d85.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d86)
			} else if d85.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d85.Imm.Int() >= -2147483648 && d85.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d85.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d85.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d86)
			} else {
				r89 := ctx.AllocRegExcept(d84.Reg, d85.Reg)
				ctx.W.EmitMovRegReg(r89, d84.Reg)
				ctx.W.EmitAddInt64(r89, d85.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d86)
			}
			if d86.Loc == scm.LocReg && d84.Loc == scm.LocReg && d86.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d84)
			ctx.FreeDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d86.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d87)
			}
			ctx.FreeDesc(&d86)
			var d88 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d88)
			}
			d89 := d88
			ctx.EnsureDesc(&d89)
			if d89.Loc != scm.LocImm && d89.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d89.Loc == scm.LocImm {
				if d89.Imm.Bool() {
					ctx.W.MarkLabel(lbl30)
					ctx.W.EmitJmp(lbl28)
				} else {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl29)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d89.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl30)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl30)
				ctx.W.EmitJmp(lbl28)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d88)
			ctx.W.MarkLabel(lbl16)
			ctx.EnsureDesc(&d43)
			d90 := d43
			_ = d90
			r92 := d43.Loc == scm.LocReg
			r93 := d43.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl33)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d90.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d90.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d91)
			}
			var d92 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d92)
			}
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d92)
			var d93 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d92.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d92.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d93)
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
				r97 := ctx.AllocRegExcept(d91.Reg, d93.Reg)
				ctx.W.EmitMovRegReg(r97, d91.Reg)
				ctx.W.EmitImulInt64(r97, d93.Reg)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d94)
			}
			if d94.Loc == scm.LocReg && d91.Loc == scm.LocReg && d94.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			ctx.FreeDesc(&d93)
			var d95 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d95)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d95)
			}
			ctx.BindReg(r98, &d95)
			ctx.EnsureDesc(&d94)
			var d96 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r99, d94.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d96)
			}
			if d96.Loc == scm.LocReg && d94.Loc == scm.LocReg && d96.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d96)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d95)
			if d96.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d96.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d96.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d95.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d95.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d97 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d97)
			ctx.FreeDesc(&d96)
			ctx.EnsureDesc(&d94)
			var d98 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r102, d94.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d98)
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
				r103 := ctx.AllocRegExcept(d97.Reg)
				ctx.W.EmitMovRegReg(r103, d97.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d98.Imm.Int()))
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d99)
			} else {
				{
					shiftSrc := d97.Reg
					r104 := ctx.AllocRegExcept(d97.Reg)
					ctx.W.EmitMovRegReg(r104, d97.Reg)
					shiftSrc = r104
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d100)
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
			ctx.EmitStoreToStack(d102, 16)
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
			ctx.EmitStoreToStack(d103, 16)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d100)
			ctx.W.MarkLabel(lbl35)
			d104 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d105)
			}
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d105)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d105.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d106)
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
				r108 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r108, d107.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d108)
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
				r109 := ctx.AllocRegExcept(d107.Reg, d106.Reg)
				ctx.W.EmitMovRegReg(r109, d107.Reg)
				ctx.W.EmitSubInt64(r109, d106.Reg)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d108)
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
				r110 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r110, d104.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d108.Imm.Int()))
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d109)
			} else {
				{
					shiftSrc := d104.Reg
					r111 := ctx.AllocRegExcept(d104.Reg)
					ctx.W.EmitMovRegReg(r111, d104.Reg)
					shiftSrc = r111
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
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			if d109.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d109)
			}
			ctx.W.EmitJmp(lbl32)
			ctx.W.MarkLabel(lbl34)
			d104 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d94)
			var d110 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r113, d94.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d110)
			}
			if d110.Loc == scm.LocReg && d94.Loc == scm.LocReg && d110.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			var d111 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d111)
			}
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d111.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d111.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d112)
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
				r116 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r116, d110.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d113)
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
				r117 := ctx.AllocRegExcept(d110.Reg, d112.Reg)
				ctx.W.EmitMovRegReg(r117, d110.Reg)
				ctx.W.EmitAddInt64(r117, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d113)
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
				r118 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitCmpRegImm32(d113.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d114)
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
			ctx.EmitStoreToStack(d116, 16)
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
			ctx.EmitStoreToStack(d117, 16)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d114)
			ctx.W.MarkLabel(lbl38)
			d104 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d94)
			var d118 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r119, d94.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d118)
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
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d95)
			if d119.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d119.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d119.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d95.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d95.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d120 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d120)
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d94)
			var d121 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d94.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r122, d94.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d121)
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
				r123 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r123, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d123)
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
				r124 := ctx.AllocRegExcept(d122.Reg, d121.Reg)
				ctx.W.EmitMovRegReg(r124, d122.Reg)
				ctx.W.EmitSubInt64(r124, d121.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d123)
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
				r125 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r125, d120.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d123.Imm.Int()))
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d124)
			} else {
				{
					shiftSrc := d120.Reg
					r126 := ctx.AllocRegExcept(d120.Reg)
					ctx.W.EmitMovRegReg(r126, d120.Reg)
					shiftSrc = r126
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
				r127 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r127, d99.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d125)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else if d124.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r128, d99.Reg)
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d125)
			} else {
				r129 := ctx.AllocRegExcept(d99.Reg, d124.Reg)
				ctx.W.EmitMovRegReg(r129, d99.Reg)
				ctx.W.EmitOrInt64(r129, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d125)
			}
			if d125.Loc == scm.LocReg && d99.Loc == scm.LocReg && d125.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			d126 := d125
			if d126.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d126)
			ctx.EmitStoreToStack(d126, 16)
			ctx.W.EmitJmp(lbl35)
			ctx.W.MarkLabel(lbl32)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d127)
			ctx.BindReg(r112, &d127)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d127.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d128)
			}
			ctx.FreeDesc(&d127)
			var d129 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d129)
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
				r132 := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegReg(r132, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d130)
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
				r133 := ctx.AllocRegExcept(d128.Reg, d129.Reg)
				ctx.W.EmitMovRegReg(r133, d128.Reg)
				ctx.W.EmitAddInt64(r133, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d130)
			}
			if d130.Loc == scm.LocReg && d128.Loc == scm.LocReg && d130.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.FreeDesc(&d129)
			ctx.EnsureDesc(&d43)
			d131 := d43
			_ = d131
			r134 := d43.Loc == scm.LocReg
			r135 := d43.Reg
			if r134 { ctx.ProtectReg(r135) }
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl42)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d131)
			var d132 scm.JITValueDesc
			if d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d131.Imm.Int()))))}
			} else {
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r136, d131.Reg)
				ctx.W.EmitShlRegImm8(r136, 32)
				ctx.W.EmitShrRegImm8(r136, 32)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d132)
			}
			var d133 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r137, thisptr.Reg, off)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
				ctx.BindReg(r137, &d133)
			}
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d133)
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d133.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d133.Reg)
				ctx.W.EmitShlRegImm8(r138, 56)
				ctx.W.EmitShrRegImm8(r138, 56)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d134)
			}
			ctx.FreeDesc(&d133)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() * d134.Imm.Int())}
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(scratch, d132.Reg)
				if d134.Imm.Int() >= -2147483648 && d134.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d134.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d134.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else {
				r139 := ctx.AllocRegExcept(d132.Reg, d134.Reg)
				ctx.W.EmitMovRegReg(r139, d132.Reg)
				ctx.W.EmitImulInt64(r139, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d135)
			}
			if d135.Loc == scm.LocReg && d132.Loc == scm.LocReg && d135.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.FreeDesc(&d134)
			var d136 scm.JITValueDesc
			r140 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r140, uint64(dataPtr))
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140, StackOff: int32(sliceLen)}
				ctx.BindReg(r140, &d136)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
				ctx.BindReg(r140, &d136)
			}
			ctx.BindReg(r140, &d136)
			ctx.EnsureDesc(&d135)
			var d137 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() / 64)}
			} else {
				r141 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r141, d135.Reg)
				ctx.W.EmitShrRegImm8(r141, 6)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d137)
			}
			if d137.Loc == scm.LocReg && d135.Loc == scm.LocReg && d137.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d137)
			r142 := ctx.AllocReg()
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d136)
			if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r142, uint64(d137.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r142, d137.Reg)
				ctx.W.EmitShlRegImm8(r142, 3)
			}
			if d136.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
				ctx.W.EmitAddInt64(r142, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r142, d136.Reg)
			}
			r143 := ctx.AllocRegExcept(r142)
			ctx.W.EmitMovRegMem(r143, r142, 0)
			ctx.FreeReg(r142)
			d138 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
			ctx.BindReg(r143, &d138)
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d135)
			var d139 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() % 64)}
			} else {
				r144 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r144, d135.Reg)
				ctx.W.EmitAndRegImm32(r144, 63)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d139)
			}
			if d139.Loc == scm.LocReg && d135.Loc == scm.LocReg && d139.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d138.Imm.Int()) << uint64(d139.Imm.Int())))}
			} else if d139.Loc == scm.LocImm {
				r145 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r145, d138.Reg)
				ctx.W.EmitShlRegImm8(r145, uint8(d139.Imm.Int()))
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d140)
			} else {
				{
					shiftSrc := d138.Reg
					r146 := ctx.AllocRegExcept(d138.Reg)
					ctx.W.EmitMovRegReg(r146, d138.Reg)
					shiftSrc = r146
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d139.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d139.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d139.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d140)
				}
			}
			if d140.Loc == scm.LocReg && d138.Loc == scm.LocReg && d140.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			ctx.FreeDesc(&d139)
			var d141 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d141)
			}
			d142 := d141
			ctx.EnsureDesc(&d142)
			if d142.Loc != scm.LocImm && d142.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			if d142.Loc == scm.LocImm {
				if d142.Imm.Bool() {
					ctx.W.MarkLabel(lbl45)
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.MarkLabel(lbl46)
			d143 := d140
			if d143.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d143)
			ctx.EmitStoreToStack(d143, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d142.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl46)
			d144 := d140
			if d144.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d144)
			ctx.EmitStoreToStack(d144, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d141)
			ctx.W.MarkLabel(lbl44)
			d145 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r148, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
				ctx.BindReg(r148, &d146)
			}
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d146)
			var d147 scm.JITValueDesc
			if d146.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d146.Imm.Int()))))}
			} else {
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r149, d146.Reg)
				ctx.W.EmitShlRegImm8(r149, 56)
				ctx.W.EmitShrRegImm8(r149, 56)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d147)
			}
			ctx.FreeDesc(&d146)
			d148 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d147)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() - d147.Imm.Int())}
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				r150 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r150, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d149)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d148.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(scratch, d148.Reg)
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d147.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else {
				r151 := ctx.AllocRegExcept(d148.Reg, d147.Reg)
				ctx.W.EmitMovRegReg(r151, d148.Reg)
				ctx.W.EmitSubInt64(r151, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d149)
			}
			if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d149)
			var d150 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d145.Imm.Int()) >> uint64(d149.Imm.Int())))}
			} else if d149.Loc == scm.LocImm {
				r152 := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegReg(r152, d145.Reg)
				ctx.W.EmitShrRegImm8(r152, uint8(d149.Imm.Int()))
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d150)
			} else {
				{
					shiftSrc := d145.Reg
					r153 := ctx.AllocRegExcept(d145.Reg)
					ctx.W.EmitMovRegReg(r153, d145.Reg)
					shiftSrc = r153
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d149.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d149.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d149.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d150)
				}
			}
			if d150.Loc == scm.LocReg && d145.Loc == scm.LocReg && d150.Reg == d145.Reg {
				ctx.TransferReg(d145.Reg)
				d145.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d145)
			ctx.FreeDesc(&d149)
			r154 := ctx.AllocReg()
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d150)
			if d150.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r154, d150)
			}
			ctx.W.EmitJmp(lbl41)
			ctx.W.MarkLabel(lbl43)
			d145 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d135)
			var d151 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() % 64)}
			} else {
				r155 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r155, d135.Reg)
				ctx.W.EmitAndRegImm32(r155, 63)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d151)
			}
			if d151.Loc == scm.LocReg && d135.Loc == scm.LocReg && d151.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			var d152 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r156, thisptr.Reg, off)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
				ctx.BindReg(r156, &d152)
			}
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d152)
			var d153 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d152.Imm.Int()))))}
			} else {
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r157, d152.Reg)
				ctx.W.EmitShlRegImm8(r157, 56)
				ctx.W.EmitShrRegImm8(r157, 56)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d153)
			}
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d153)
			var d154 scm.JITValueDesc
			if d151.Loc == scm.LocImm && d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d151.Imm.Int() + d153.Imm.Int())}
			} else if d153.Loc == scm.LocImm && d153.Imm.Int() == 0 {
				r158 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r158, d151.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d154)
			} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d153.Reg}
				ctx.BindReg(d153.Reg, &d154)
			} else if d151.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d151.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(scratch, d151.Reg)
				if d153.Imm.Int() >= -2147483648 && d153.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d153.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d153.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else {
				r159 := ctx.AllocRegExcept(d151.Reg, d153.Reg)
				ctx.W.EmitMovRegReg(r159, d151.Reg)
				ctx.W.EmitAddInt64(r159, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d154)
			}
			if d154.Loc == scm.LocReg && d151.Loc == scm.LocReg && d154.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			ctx.FreeDesc(&d153)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d154.Imm.Int()) > uint64(64))}
			} else {
				r160 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitCmpRegImm32(d154.Reg, 64)
				ctx.W.EmitSetcc(r160, scm.CcA)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
				ctx.BindReg(r160, &d155)
			}
			ctx.FreeDesc(&d154)
			d156 := d155
			ctx.EnsureDesc(&d156)
			if d156.Loc != scm.LocImm && d156.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			if d156.Loc == scm.LocImm {
				if d156.Imm.Bool() {
					ctx.W.MarkLabel(lbl48)
					ctx.W.EmitJmp(lbl47)
				} else {
					ctx.W.MarkLabel(lbl49)
			d157 := d140
			if d157.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d157)
			ctx.EmitStoreToStack(d157, 24)
					ctx.W.EmitJmp(lbl44)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d156.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl48)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl48)
				ctx.W.EmitJmp(lbl47)
				ctx.W.MarkLabel(lbl49)
			d158 := d140
			if d158.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d158)
			ctx.EmitStoreToStack(d158, 24)
				ctx.W.EmitJmp(lbl44)
			}
			ctx.FreeDesc(&d155)
			ctx.W.MarkLabel(lbl47)
			d145 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d135)
			var d159 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r161, d135.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d159)
			}
			if d159.Loc == scm.LocReg && d135.Loc == scm.LocReg && d159.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(scratch, d159.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			}
			if d160.Loc == scm.LocReg && d159.Loc == scm.LocReg && d160.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d159)
			ctx.EnsureDesc(&d160)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d136)
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d160.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d160.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d136.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d136.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d136.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d161 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d161)
			ctx.FreeDesc(&d160)
			ctx.EnsureDesc(&d135)
			var d162 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r164, d135.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d162)
			}
			if d162.Loc == scm.LocReg && d135.Loc == scm.LocReg && d162.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			d163 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d162)
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d162.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() - d162.Imm.Int())}
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r165, d163.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d164)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d163.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d162.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d164)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(scratch, d163.Reg)
				if d162.Imm.Int() >= -2147483648 && d162.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d162.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d162.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d164)
			} else {
				r166 := ctx.AllocRegExcept(d163.Reg, d162.Reg)
				ctx.W.EmitMovRegReg(r166, d163.Reg)
				ctx.W.EmitSubInt64(r166, d162.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d164)
			}
			if d164.Loc == scm.LocReg && d163.Loc == scm.LocReg && d164.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d161.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d161.Imm.Int()) >> uint64(d164.Imm.Int())))}
			} else if d164.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegReg(r167, d161.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d164.Imm.Int()))
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d165)
			} else {
				{
					shiftSrc := d161.Reg
					r168 := ctx.AllocRegExcept(d161.Reg)
					ctx.W.EmitMovRegReg(r168, d161.Reg)
					shiftSrc = r168
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d164.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d164.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d164.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d165)
				}
			}
			if d165.Loc == scm.LocReg && d161.Loc == scm.LocReg && d165.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d165)
			var d166 scm.JITValueDesc
			if d140.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() | d165.Imm.Int())}
			} else if d140.Loc == scm.LocImm && d140.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
				ctx.BindReg(d165.Reg, &d166)
			} else if d165.Loc == scm.LocImm && d165.Imm.Int() == 0 {
				r169 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r169, d140.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d166)
			} else if d140.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d140.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else if d165.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r170, d140.Reg)
				if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r170, int32(d165.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
					ctx.W.EmitOrInt64(r170, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d166)
			} else {
				r171 := ctx.AllocRegExcept(d140.Reg, d165.Reg)
				ctx.W.EmitMovRegReg(r171, d140.Reg)
				ctx.W.EmitOrInt64(r171, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d166)
			}
			if d166.Loc == scm.LocReg && d140.Loc == scm.LocReg && d166.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d165)
			d167 := d166
			if d167.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d167)
			ctx.EmitStoreToStack(d167, 24)
			ctx.W.EmitJmp(lbl44)
			ctx.W.MarkLabel(lbl41)
			d168 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d168)
			ctx.BindReg(r154, &d168)
			if r134 { ctx.UnprotectReg(r135) }
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d168)
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d168.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d169)
			}
			ctx.FreeDesc(&d168)
			var d170 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r173 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r173, thisptr.Reg, off)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
				ctx.BindReg(r173, &d170)
			}
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d169.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() + d170.Imm.Int())}
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r174 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r174, d169.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d171)
			} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
				ctx.BindReg(d170.Reg, &d171)
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d169.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(scratch, d169.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else {
				r175 := ctx.AllocRegExcept(d169.Reg, d170.Reg)
				ctx.W.EmitMovRegReg(r175, d169.Reg)
				ctx.W.EmitAddInt64(r175, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d171)
			}
			if d171.Loc == scm.LocReg && d169.Loc == scm.LocReg && d171.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			ctx.FreeDesc(&d170)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d171)
			var d173 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() + d171.Imm.Int())}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r176, d130.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d173)
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
				ctx.BindReg(d171.Reg, &d173)
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(scratch, d130.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d171.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else {
				r177 := ctx.AllocRegExcept(d130.Reg, d171.Reg)
				ctx.W.EmitMovRegReg(r177, d130.Reg)
				ctx.W.EmitAddInt64(r177, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d173)
			}
			if d173.Loc == scm.LocReg && d130.Loc == scm.LocReg && d173.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r178 := ctx.AllocReg()
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r178, fieldAddr)
				ctx.W.EmitMovRegMem64(r179, fieldAddr+8)
				d175 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r178, Reg2: r179}
				ctx.BindReg(r178, &d175)
				ctx.BindReg(r179, &d175)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r180, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r181, thisptr.Reg, off+8)
				d175 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d175)
				ctx.BindReg(r181, &d175)
			}
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d173)
			r182 := ctx.AllocReg()
			r183 := ctx.AllocRegExcept(r182)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d173)
			if d175.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d175.Imm.Int()))
			} else if d175.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r182, d175.Reg)
			} else {
				ctx.W.EmitMovRegReg(r182, d175.Reg)
			}
			if d130.Loc == scm.LocImm {
				if d130.Imm.Int() != 0 {
					if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r182, int32(d130.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
						ctx.W.EmitAddInt64(r182, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r182, d130.Reg)
			}
			if d173.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r183, uint64(d173.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r183, d173.Reg)
			}
			if d130.Loc == scm.LocImm {
				if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r183, int32(d130.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
					ctx.W.EmitSubInt64(r183, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r183, d130.Reg)
			}
			d176 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
			ctx.BindReg(r182, &d176)
			ctx.BindReg(r183, &d176)
			ctx.FreeDesc(&d130)
			ctx.FreeDesc(&d173)
			d177 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d177)
			ctx.BindReg(r1, &d177)
			d178 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d176}, 2)
			ctx.EmitMovPairToResult(&d178, &d177)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			var d179 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r184 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r184, thisptr.Reg, off)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r184}
				ctx.BindReg(r184, &d179)
			}
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d179)
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d179.Imm.Int()))))}
			} else {
				r185 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r185, d179.Reg)
				ctx.W.EmitShlRegImm8(r185, 32)
				ctx.W.EmitShrRegImm8(r185, 32)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d180)
			}
			ctx.FreeDesc(&d179)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d180)
			var d181 scm.JITValueDesc
			if d43.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d43.Imm.Int()) == uint64(d180.Imm.Int()))}
			} else if d180.Loc == scm.LocImm {
				r186 := ctx.AllocRegExcept(d43.Reg)
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d43.Reg, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitCmpInt64(d43.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r186, scm.CcE)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r186}
				ctx.BindReg(r186, &d181)
			} else if d43.Loc == scm.LocImm {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d180.Reg)
				ctx.W.EmitSetcc(r187, scm.CcE)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r187}
				ctx.BindReg(r187, &d181)
			} else {
				r188 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitCmpInt64(d43.Reg, d180.Reg)
				ctx.W.EmitSetcc(r188, scm.CcE)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r188}
				ctx.BindReg(r188, &d181)
			}
			ctx.FreeDesc(&d43)
			ctx.FreeDesc(&d180)
			d182 := d181
			ctx.EnsureDesc(&d182)
			if d182.Loc != scm.LocImm && d182.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			if d182.Loc == scm.LocImm {
				if d182.Imm.Bool() {
					ctx.W.MarkLabel(lbl51)
					ctx.W.EmitJmp(lbl50)
				} else {
					ctx.W.MarkLabel(lbl52)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d182.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl51)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl51)
				ctx.W.EmitJmp(lbl50)
				ctx.W.MarkLabel(lbl52)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.FreeDesc(&d181)
			ctx.W.MarkLabel(lbl29)
			ctx.EnsureDesc(&idxInt)
			d183 := idxInt
			_ = d183
			r189 := idxInt.Loc == scm.LocReg
			r190 := idxInt.Reg
			if r189 { ctx.ProtectReg(r190) }
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl54)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d183.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, d183.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d184)
			}
			var d185 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r192, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r192}
				ctx.BindReg(r192, &d185)
			}
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d185)
			var d186 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d185.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, d185.Reg)
				ctx.W.EmitShlRegImm8(r193, 56)
				ctx.W.EmitShrRegImm8(r193, 56)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d186)
			}
			ctx.FreeDesc(&d185)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d186)
			var d187 scm.JITValueDesc
			if d184.Loc == scm.LocImm && d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() * d186.Imm.Int())}
			} else if d184.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d184.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d187)
			} else if d186.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(scratch, d184.Reg)
				if d186.Imm.Int() >= -2147483648 && d186.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d186.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d187)
			} else {
				r194 := ctx.AllocRegExcept(d184.Reg, d186.Reg)
				ctx.W.EmitMovRegReg(r194, d184.Reg)
				ctx.W.EmitImulInt64(r194, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d187)
			}
			if d187.Loc == scm.LocReg && d184.Loc == scm.LocReg && d187.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d184)
			ctx.FreeDesc(&d186)
			var d188 scm.JITValueDesc
			r195 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r195, uint64(dataPtr))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195, StackOff: int32(sliceLen)}
				ctx.BindReg(r195, &d188)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r195, thisptr.Reg, off)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195}
				ctx.BindReg(r195, &d188)
			}
			ctx.BindReg(r195, &d188)
			ctx.EnsureDesc(&d187)
			var d189 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() / 64)}
			} else {
				r196 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r196, d187.Reg)
				ctx.W.EmitShrRegImm8(r196, 6)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d189)
			}
			if d189.Loc == scm.LocReg && d187.Loc == scm.LocReg && d189.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d189)
			r197 := ctx.AllocReg()
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d188)
			if d189.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r197, uint64(d189.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r197, d189.Reg)
				ctx.W.EmitShlRegImm8(r197, 3)
			}
			if d188.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d188.Imm.Int()))
				ctx.W.EmitAddInt64(r197, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r197, d188.Reg)
			}
			r198 := ctx.AllocRegExcept(r197)
			ctx.W.EmitMovRegMem(r198, r197, 0)
			ctx.FreeReg(r197)
			d190 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
			ctx.BindReg(r198, &d190)
			ctx.FreeDesc(&d189)
			ctx.EnsureDesc(&d187)
			var d191 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() % 64)}
			} else {
				r199 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r199, d187.Reg)
				ctx.W.EmitAndRegImm32(r199, 63)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d191)
			}
			if d191.Loc == scm.LocReg && d187.Loc == scm.LocReg && d191.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d191)
			var d192 scm.JITValueDesc
			if d190.Loc == scm.LocImm && d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d190.Imm.Int()) << uint64(d191.Imm.Int())))}
			} else if d191.Loc == scm.LocImm {
				r200 := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegReg(r200, d190.Reg)
				ctx.W.EmitShlRegImm8(r200, uint8(d191.Imm.Int()))
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d192)
			} else {
				{
					shiftSrc := d190.Reg
					r201 := ctx.AllocRegExcept(d190.Reg)
					ctx.W.EmitMovRegReg(r201, d190.Reg)
					shiftSrc = r201
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d191.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d191.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d191.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d192)
				}
			}
			if d192.Loc == scm.LocReg && d190.Loc == scm.LocReg && d192.Reg == d190.Reg {
				ctx.TransferReg(d190.Reg)
				d190.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d190)
			ctx.FreeDesc(&d191)
			var d193 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d193)
			}
			d194 := d193
			ctx.EnsureDesc(&d194)
			if d194.Loc != scm.LocImm && d194.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d194.Loc == scm.LocImm {
				if d194.Imm.Bool() {
					ctx.W.MarkLabel(lbl57)
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.W.MarkLabel(lbl58)
			d195 := d192
			if d195.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d195)
			ctx.EmitStoreToStack(d195, 32)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d194.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl58)
			d196 := d192
			if d196.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d196)
			ctx.EmitStoreToStack(d196, 32)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d193)
			ctx.W.MarkLabel(lbl56)
			d197 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d198 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r203, thisptr.Reg, off)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r203}
				ctx.BindReg(r203, &d198)
			}
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d198.Imm.Int()))))}
			} else {
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r204, d198.Reg)
				ctx.W.EmitShlRegImm8(r204, 56)
				ctx.W.EmitShrRegImm8(r204, 56)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d199)
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
				r205 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r205, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d201)
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
				r206 := ctx.AllocRegExcept(d200.Reg, d199.Reg)
				ctx.W.EmitMovRegReg(r206, d200.Reg)
				ctx.W.EmitSubInt64(r206, d199.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d201)
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
				r207 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r207, d197.Reg)
				ctx.W.EmitShrRegImm8(r207, uint8(d201.Imm.Int()))
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d202)
			} else {
				{
					shiftSrc := d197.Reg
					r208 := ctx.AllocRegExcept(d197.Reg)
					ctx.W.EmitMovRegReg(r208, d197.Reg)
					shiftSrc = r208
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
			r209 := ctx.AllocReg()
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d202)
			if d202.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r209, d202)
			}
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl55)
			d197 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d187)
			var d203 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() % 64)}
			} else {
				r210 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r210, d187.Reg)
				ctx.W.EmitAndRegImm32(r210, 63)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d203)
			}
			if d203.Loc == scm.LocReg && d187.Loc == scm.LocReg && d203.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			var d204 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r211, thisptr.Reg, off)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
				ctx.BindReg(r211, &d204)
			}
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d204)
			var d205 scm.JITValueDesc
			if d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d204.Imm.Int()))))}
			} else {
				r212 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r212, d204.Reg)
				ctx.W.EmitShlRegImm8(r212, 56)
				ctx.W.EmitShrRegImm8(r212, 56)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d205)
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
				r213 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegReg(r213, d203.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d206)
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
				r214 := ctx.AllocRegExcept(d203.Reg, d205.Reg)
				ctx.W.EmitMovRegReg(r214, d203.Reg)
				ctx.W.EmitAddInt64(r214, d205.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d206)
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
				r215 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitCmpRegImm32(d206.Reg, 64)
				ctx.W.EmitSetcc(r215, scm.CcA)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r215}
				ctx.BindReg(r215, &d207)
			}
			ctx.FreeDesc(&d206)
			d208 := d207
			ctx.EnsureDesc(&d208)
			if d208.Loc != scm.LocImm && d208.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d208.Loc == scm.LocImm {
				if d208.Imm.Bool() {
					ctx.W.MarkLabel(lbl60)
					ctx.W.EmitJmp(lbl59)
				} else {
					ctx.W.MarkLabel(lbl61)
			d209 := d192
			if d209.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d209)
			ctx.EmitStoreToStack(d209, 32)
					ctx.W.EmitJmp(lbl56)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d208.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl61)
			d210 := d192
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, 32)
				ctx.W.EmitJmp(lbl56)
			}
			ctx.FreeDesc(&d207)
			ctx.W.MarkLabel(lbl59)
			d197 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d187)
			var d211 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() / 64)}
			} else {
				r216 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r216, d187.Reg)
				ctx.W.EmitShrRegImm8(r216, 6)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d211)
			}
			if d211.Loc == scm.LocReg && d187.Loc == scm.LocReg && d211.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d211)
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d211.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegReg(scratch, d211.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			}
			if d212.Loc == scm.LocReg && d211.Loc == scm.LocReg && d212.Reg == d211.Reg {
				ctx.TransferReg(d211.Reg)
				d211.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d212)
			r217 := ctx.AllocReg()
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d188)
			if d212.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r217, uint64(d212.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r217, d212.Reg)
				ctx.W.EmitShlRegImm8(r217, 3)
			}
			if d188.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d188.Imm.Int()))
				ctx.W.EmitAddInt64(r217, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r217, d188.Reg)
			}
			r218 := ctx.AllocRegExcept(r217)
			ctx.W.EmitMovRegMem(r218, r217, 0)
			ctx.FreeReg(r217)
			d213 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
			ctx.BindReg(r218, &d213)
			ctx.FreeDesc(&d212)
			ctx.EnsureDesc(&d187)
			var d214 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() % 64)}
			} else {
				r219 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r219, d187.Reg)
				ctx.W.EmitAndRegImm32(r219, 63)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d214)
			}
			if d214.Loc == scm.LocReg && d187.Loc == scm.LocReg && d214.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d187)
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
				r220 := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(r220, d215.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d216)
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
				r221 := ctx.AllocRegExcept(d215.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r221, d215.Reg)
				ctx.W.EmitSubInt64(r221, d214.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d216)
			}
			if d216.Loc == scm.LocReg && d215.Loc == scm.LocReg && d216.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d213.Loc == scm.LocImm && d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d213.Imm.Int()) >> uint64(d216.Imm.Int())))}
			} else if d216.Loc == scm.LocImm {
				r222 := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitMovRegReg(r222, d213.Reg)
				ctx.W.EmitShrRegImm8(r222, uint8(d216.Imm.Int()))
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d217)
			} else {
				{
					shiftSrc := d213.Reg
					r223 := ctx.AllocRegExcept(d213.Reg)
					ctx.W.EmitMovRegReg(r223, d213.Reg)
					shiftSrc = r223
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
			if d217.Loc == scm.LocReg && d213.Loc == scm.LocReg && d217.Reg == d213.Reg {
				ctx.TransferReg(d213.Reg)
				d213.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d213)
			ctx.FreeDesc(&d216)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d192.Loc == scm.LocImm && d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d192.Imm.Int() | d217.Imm.Int())}
			} else if d192.Loc == scm.LocImm && d192.Imm.Int() == 0 {
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d217.Reg}
				ctx.BindReg(d217.Reg, &d218)
			} else if d217.Loc == scm.LocImm && d217.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(r224, d192.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d218)
			} else if d192.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d192.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d218)
			} else if d217.Loc == scm.LocImm {
				r225 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(r225, d192.Reg)
				if d217.Imm.Int() >= -2147483648 && d217.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r225, int32(d217.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d217.Imm.Int()))
					ctx.W.EmitOrInt64(r225, scm.RegR11)
				}
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d218)
			} else {
				r226 := ctx.AllocRegExcept(d192.Reg, d217.Reg)
				ctx.W.EmitMovRegReg(r226, d192.Reg)
				ctx.W.EmitOrInt64(r226, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d218)
			}
			if d218.Loc == scm.LocReg && d192.Loc == scm.LocReg && d218.Reg == d192.Reg {
				ctx.TransferReg(d192.Reg)
				d192.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d217)
			d219 := d218
			if d219.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d219)
			ctx.EmitStoreToStack(d219, 32)
			ctx.W.EmitJmp(lbl56)
			ctx.W.MarkLabel(lbl53)
			d220 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
			ctx.BindReg(r209, &d220)
			ctx.BindReg(r209, &d220)
			if r189 { ctx.UnprotectReg(r190) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d220)
			var d221 scm.JITValueDesc
			if d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d220.Imm.Int()))))}
			} else {
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r227, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d221)
			}
			ctx.FreeDesc(&d220)
			var d222 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r228 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r228, thisptr.Reg, off)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r228}
				ctx.BindReg(r228, &d222)
			}
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d222)
			var d223 scm.JITValueDesc
			if d221.Loc == scm.LocImm && d222.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d221.Imm.Int() + d222.Imm.Int())}
			} else if d222.Loc == scm.LocImm && d222.Imm.Int() == 0 {
				r229 := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(r229, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d223)
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d222.Reg}
				ctx.BindReg(d222.Reg, &d223)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d221.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d222.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else if d222.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegReg(scratch, d221.Reg)
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d222.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else {
				r230 := ctx.AllocRegExcept(d221.Reg, d222.Reg)
				ctx.W.EmitMovRegReg(r230, d221.Reg)
				ctx.W.EmitAddInt64(r230, d222.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d223)
			}
			if d223.Loc == scm.LocReg && d221.Loc == scm.LocReg && d223.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.FreeDesc(&d222)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d223.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d224)
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			var d225 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d87.Imm.Int()))))}
			} else {
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r232, d87.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d225)
			}
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d224)
			var d226 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d224.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d224.Imm.Int())}
			} else if d224.Loc == scm.LocImm && d224.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r233, d87.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d226)
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d224.Reg}
				ctx.BindReg(d224.Reg, &d226)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d224.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d226)
			} else if d224.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(scratch, d87.Reg)
				if d224.Imm.Int() >= -2147483648 && d224.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d224.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d224.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d226)
			} else {
				r234 := ctx.AllocRegExcept(d87.Reg, d224.Reg)
				ctx.W.EmitMovRegReg(r234, d87.Reg)
				ctx.W.EmitAddInt64(r234, d224.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d226)
			}
			if d226.Loc == scm.LocReg && d87.Loc == scm.LocReg && d226.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d224)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d226)
			var d227 scm.JITValueDesc
			if d226.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d226.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d226.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d227)
			}
			ctx.FreeDesc(&d226)
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d227)
			r236 := ctx.AllocReg()
			r237 := ctx.AllocRegExcept(r236)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d227)
			if d175.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d175.Imm.Int()))
			} else if d175.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r236, d175.Reg)
			} else {
				ctx.W.EmitMovRegReg(r236, d175.Reg)
			}
			if d225.Loc == scm.LocImm {
				if d225.Imm.Int() != 0 {
					if d225.Imm.Int() >= -2147483648 && d225.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r236, int32(d225.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d225.Imm.Int()))
						ctx.W.EmitAddInt64(r236, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r236, d225.Reg)
			}
			if d227.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r237, uint64(d227.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r237, d227.Reg)
			}
			if d225.Loc == scm.LocImm {
				if d225.Imm.Int() >= -2147483648 && d225.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r237, int32(d225.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d225.Imm.Int()))
					ctx.W.EmitSubInt64(r237, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r237, d225.Reg)
			}
			d228 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r236, Reg2: r237}
			ctx.BindReg(r236, &d228)
			ctx.BindReg(r237, &d228)
			ctx.FreeDesc(&d225)
			ctx.FreeDesc(&d227)
			d229 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d229)
			ctx.BindReg(r1, &d229)
			d230 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d228}, 2)
			ctx.EmitMovPairToResult(&d230, &d229)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl28)
			var d231 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r238, thisptr.Reg, off)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r238}
				ctx.BindReg(r238, &d231)
			}
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d231)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d231)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d231)
			var d232 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d87.Imm.Int()) == uint64(d231.Imm.Int()))}
			} else if d231.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d87.Reg)
				if d231.Imm.Int() >= -2147483648 && d231.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d87.Reg, int32(d231.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d231.Imm.Int()))
					ctx.W.EmitCmpInt64(d87.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d232)
			} else if d87.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d231.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d232)
			} else {
				r241 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitCmpInt64(d87.Reg, d231.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d232)
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d231)
			d233 := d232
			ctx.EnsureDesc(&d233)
			if d233.Loc != scm.LocImm && d233.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			lbl64 := ctx.W.ReserveLabel()
			if d233.Loc == scm.LocImm {
				if d233.Imm.Bool() {
					ctx.W.MarkLabel(lbl63)
					ctx.W.EmitJmp(lbl62)
				} else {
					ctx.W.MarkLabel(lbl64)
					ctx.W.EmitJmp(lbl29)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d233.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl63)
				ctx.W.EmitJmp(lbl64)
				ctx.W.MarkLabel(lbl63)
				ctx.W.EmitJmp(lbl62)
				ctx.W.MarkLabel(lbl64)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d232)
			ctx.W.MarkLabel(lbl50)
			d234 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d234)
			ctx.BindReg(r1, &d234)
			ctx.W.EmitMakeNil(d234)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl62)
			d235 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d235)
			ctx.BindReg(r1, &d235)
			ctx.W.EmitMakeNil(d235)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d236 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d236)
			ctx.BindReg(r1, &d236)
			ctx.EmitMovPairToResult(&d236, &result)
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
