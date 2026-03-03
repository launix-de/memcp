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
				if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
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
				ctx.BindReg(r0, &d0)
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r1 := idxInt.Loc == scm.LocReg
			r2 := idxInt.Reg
			if r1 { ctx.ProtectReg(r2) }
			r3 := ctx.W.EmitSubRSP32Fixup()
			lbl4 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d1 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r4, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r4, 32)
				ctx.W.EmitShrRegImm8(r4, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r5, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
				ctx.BindReg(r5, &d2)
			}
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r6, d2.Reg)
				ctx.W.EmitShlRegImm8(r6, 56)
				ctx.W.EmitShrRegImm8(r6, 56)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d3)
			}
			ctx.FreeDesc(&d2)
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d4 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d1.Imm.Int() * d3.Imm.Int())}
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d1.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d4)
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(scratch, d1.Reg)
				if d3.Imm.Int() >= -2147483648 && d3.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d3.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d4)
			} else {
				r7 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(r7, d1.Reg)
				ctx.W.EmitImulInt64(r7, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d4)
			}
			if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d3)
			var d5 scm.JITValueDesc
			r8 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r8, uint64(dataPtr))
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8, StackOff: int32(sliceLen)}
				ctx.BindReg(r8, &d5)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				ctx.W.EmitMovRegMem(r8, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			ctx.BindReg(r8, &d5)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				ctx.W.EmitShrRegImm8(r9, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d6)
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			r10 := ctx.AllocReg()
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
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
			ctx.BindReg(r11, &d7)
			ctx.FreeDesc(&d6)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r12 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r12, d4.Reg)
				ctx.W.EmitAndRegImm32(r12, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d8)
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) << uint64(d8.Imm.Int())))}
			} else if d8.Loc == scm.LocImm {
				r13 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r13, d7.Reg)
				ctx.W.EmitShlRegImm8(r13, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d9)
			} else {
				{
					shiftSrc := d7.Reg
					r14 := ctx.AllocRegExcept(d7.Reg)
					ctx.W.EmitMovRegReg(r14, d7.Reg)
					shiftSrc = r14
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
					ctx.BindReg(shiftSrc, &d9)
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
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d10)
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d10.Loc == scm.LocImm {
				if d10.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
			d11 := d9
			if d11.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			ctx.EmitStoreToStack(d11, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
			d12 := d9
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			ctx.EmitStoreToStack(d12, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl6)
			d13 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d14 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r16, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d14)
			}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d14.Imm.Int()))))}
			} else {
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r17, d14.Reg)
				ctx.W.EmitShlRegImm8(r17, 56)
				ctx.W.EmitShrRegImm8(r17, 56)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d17 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - d15.Imm.Int())}
			} else if d15.Loc == scm.LocImm && d15.Imm.Int() == 0 {
				r18 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r18, d16.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d17)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d15.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else {
				r19 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r19, d16.Reg)
				ctx.W.EmitSubInt64(r19, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d17)
			}
			if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			var d18 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) >> uint64(d17.Imm.Int())))}
			} else if d17.Loc == scm.LocImm {
				r20 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r20, d13.Reg)
				ctx.W.EmitShrRegImm8(r20, uint8(d17.Imm.Int()))
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d18)
			} else {
				{
					shiftSrc := d13.Reg
					r21 := ctx.AllocRegExcept(d13.Reg)
					ctx.W.EmitMovRegReg(r21, d13.Reg)
					shiftSrc = r21
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d17.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d17.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d17.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d18)
				}
			}
			if d18.Loc == scm.LocReg && d13.Loc == scm.LocReg && d18.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d17)
			r22 := ctx.AllocReg()
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d18.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r22, d18.Reg2)
			} else {
				ctx.EmitMovToReg(r22, d18)
			}
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl5)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d19 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r23 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r23, d4.Reg)
				ctx.W.EmitAndRegImm32(r23, 63)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d19)
			}
			if d19.Loc == scm.LocReg && d4.Loc == scm.LocReg && d19.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d20)
			}
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d20.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d20.Reg)
				ctx.W.EmitShlRegImm8(r25, 56)
				ctx.W.EmitShrRegImm8(r25, 56)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d21)
			}
			ctx.FreeDesc(&d20)
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d22 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() + d21.Imm.Int())}
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r26, d19.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d22)
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
				ctx.BindReg(d21.Reg, &d22)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(scratch, d19.Reg)
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else {
				r27 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r27, d19.Reg)
				ctx.W.EmitAddInt64(r27, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d22)
			}
			if d22.Loc == scm.LocReg && d19.Loc == scm.LocReg && d22.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.FreeDesc(&d21)
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d22.Imm.Int()) > uint64(64))}
			} else {
				r28 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitCmpRegImm32(d22.Reg, 64)
				ctx.W.EmitSetcc(r28, scm.CcA)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r28}
				ctx.BindReg(r28, &d23)
			}
			ctx.FreeDesc(&d22)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d23.Loc == scm.LocImm {
				if d23.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
			d24 := d9
			if d24.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			ctx.EmitStoreToStack(d24, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d23.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
			d25 := d9
			if d25.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			ctx.EmitStoreToStack(d25, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d23)
			ctx.W.MarkLabel(lbl8)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d26 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r29 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r29, d4.Reg)
				ctx.W.EmitShrRegImm8(r29, 6)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d26)
			}
			if d26.Loc == scm.LocReg && d4.Loc == scm.LocReg && d26.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			r30 := ctx.AllocReg()
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r30, uint64(d27.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r30, d27.Reg)
				ctx.W.EmitShlRegImm8(r30, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r30, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r30, d5.Reg)
			}
			r31 := ctx.AllocRegExcept(r30)
			ctx.W.EmitMovRegMem(r31, r30, 0)
			ctx.FreeReg(r30)
			d28 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d28)
			ctx.FreeDesc(&d27)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d29 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r32, d4.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d29)
			}
			if d29.Loc == scm.LocReg && d4.Loc == scm.LocReg && d29.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d30 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() - d29.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r33 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r33, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r34 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r34, d30.Reg)
				ctx.W.EmitSubInt64(r34, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d31)
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			var d32 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) >> uint64(d31.Imm.Int())))}
			} else if d31.Loc == scm.LocImm {
				r35 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r35, d28.Reg)
				ctx.W.EmitShrRegImm8(r35, uint8(d31.Imm.Int()))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d32)
			} else {
				{
					shiftSrc := d28.Reg
					r36 := ctx.AllocRegExcept(d28.Reg)
					ctx.W.EmitMovRegReg(r36, d28.Reg)
					shiftSrc = r36
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d31.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d31.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d31.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d32)
				}
			}
			if d32.Loc == scm.LocReg && d28.Loc == scm.LocReg && d32.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d31)
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d33 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d32.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d33)
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r37, d9.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d33)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == scm.LocImm {
				r38 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r38, d9.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r38, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitOrInt64(r38, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d33)
			} else {
				r39 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r39, d9.Reg)
				ctx.W.EmitOrInt64(r39, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d33)
			}
			if d33.Loc == scm.LocReg && d9.Loc == scm.LocReg && d33.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			d34 := d33
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			ctx.EmitStoreToStack(d34, 0)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			d35 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
			ctx.BindReg(r22, &d35)
			ctx.BindReg(r22, &d35)
			if r1 { ctx.UnprotectReg(r2) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d35.Imm.Int()))))}
			} else {
				r40 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r40, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d36)
			}
			ctx.FreeDesc(&d35)
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r41, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
				ctx.BindReg(r41, &d37)
			}
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r42 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r42, d36.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else {
				r43 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r43, d36.Reg)
				ctx.W.EmitAddInt64(r43, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			if d38.Loc == scm.LocReg && d36.Loc == scm.LocReg && d38.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d37)
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d38.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d38.Reg)
				ctx.W.EmitShlRegImm8(r44, 32)
				ctx.W.EmitShrRegImm8(r44, 32)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d39)
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r45, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d40)
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d40.Loc == scm.LocImm {
				if d40.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d40.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d40)
			ctx.W.MarkLabel(lbl1)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r46 := idxInt.Loc == scm.LocReg
			r47 := idxInt.Reg
			if r46 { ctx.ProtectReg(r47) }
			lbl13 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d41 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d41)
			}
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d42)
			}
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d42.Reg)
				ctx.W.EmitShlRegImm8(r50, 56)
				ctx.W.EmitShrRegImm8(r50, 56)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d43)
			}
			ctx.FreeDesc(&d42)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() * d43.Imm.Int())}
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r51 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r51, d41.Reg)
				ctx.W.EmitImulInt64(r51, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d44)
			}
			if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			r52 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r52, uint64(dataPtr))
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52, StackOff: int32(sliceLen)}
				ctx.BindReg(r52, &d45)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r52, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
				ctx.BindReg(r52, &d45)
			}
			ctx.BindReg(r52, &d45)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d46 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r53 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r53, d44.Reg)
				ctx.W.EmitShrRegImm8(r53, 6)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d46)
			}
			if d46.Loc == scm.LocReg && d44.Loc == scm.LocReg && d46.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			r54 := ctx.AllocReg()
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d46.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r54, uint64(d46.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r54, d46.Reg)
				ctx.W.EmitShlRegImm8(r54, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r54, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r54, d45.Reg)
			}
			r55 := ctx.AllocRegExcept(r54)
			ctx.W.EmitMovRegMem(r55, r54, 0)
			ctx.FreeReg(r54)
			d47 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r55}
			ctx.BindReg(r55, &d47)
			ctx.FreeDesc(&d46)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d48 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r56 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r56, d44.Reg)
				ctx.W.EmitAndRegImm32(r56, 63)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
				ctx.BindReg(r56, &d48)
			}
			if d48.Loc == scm.LocReg && d44.Loc == scm.LocReg && d48.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) << uint64(d48.Imm.Int())))}
			} else if d48.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r57, d47.Reg)
				ctx.W.EmitShlRegImm8(r57, uint8(d48.Imm.Int()))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d49)
			} else {
				{
					shiftSrc := d47.Reg
					r58 := ctx.AllocRegExcept(d47.Reg)
					ctx.W.EmitMovRegReg(r58, d47.Reg)
					shiftSrc = r58
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d48.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d48.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d48.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d49)
				}
			}
			if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d48)
			var d50 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r59, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
				ctx.BindReg(r59, &d50)
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d50.Loc == scm.LocImm {
				if d50.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
			d51 := d49
			if d51.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitStoreToStack(d51, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d50.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
			d52 := d49
			if d52.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			ctx.EmitStoreToStack(d52, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d50)
			ctx.W.MarkLabel(lbl15)
			d53 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d54 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r60, thisptr.Reg, off)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
				ctx.BindReg(r60, &d54)
			}
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			var d55 scm.JITValueDesc
			if d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d54.Imm.Int()))))}
			} else {
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r61, d54.Reg)
				ctx.W.EmitShlRegImm8(r61, 56)
				ctx.W.EmitShrRegImm8(r61, 56)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d55)
			}
			ctx.FreeDesc(&d54)
			d56 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			var d57 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() - d55.Imm.Int())}
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				r62 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r62, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r62}
				ctx.BindReg(r62, &d57)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(scratch, d56.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else {
				r63 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r63, d56.Reg)
				ctx.W.EmitSubInt64(r63, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d57)
			}
			if d57.Loc == scm.LocReg && d56.Loc == scm.LocReg && d57.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			if d57.Loc == scm.LocStack || d57.Loc == scm.LocStackPair { ctx.EnsureDesc(&d57) }
			var d58 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d57.Imm.Int())))}
			} else if d57.Loc == scm.LocImm {
				r64 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r64, d53.Reg)
				ctx.W.EmitShrRegImm8(r64, uint8(d57.Imm.Int()))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d58)
			} else {
				{
					shiftSrc := d53.Reg
					r65 := ctx.AllocRegExcept(d53.Reg)
					ctx.W.EmitMovRegReg(r65, d53.Reg)
					shiftSrc = r65
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d57.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d57.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d57.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d58)
				}
			}
			if d58.Loc == scm.LocReg && d53.Loc == scm.LocReg && d58.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.FreeDesc(&d57)
			r66 := ctx.AllocReg()
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d58.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r66, d58.Reg2)
			} else {
				ctx.EmitMovToReg(r66, d58)
			}
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl14)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d59 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r67 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r67, d44.Reg)
				ctx.W.EmitAndRegImm32(r67, 63)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d59)
			}
			if d59.Loc == scm.LocReg && d44.Loc == scm.LocReg && d59.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
				ctx.BindReg(r68, &d60)
			}
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d60.Imm.Int()))))}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r69, d60.Reg)
				ctx.W.EmitShlRegImm8(r69, 56)
				ctx.W.EmitShrRegImm8(r69, 56)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d61)
			}
			ctx.FreeDesc(&d60)
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			var d62 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + d61.Imm.Int())}
			} else if d61.Loc == scm.LocImm && d61.Imm.Int() == 0 {
				r70 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r70, d59.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
				ctx.BindReg(r70, &d62)
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
				ctx.BindReg(d61.Reg, &d62)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else {
				r71 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r71, d59.Reg)
				ctx.W.EmitAddInt64(r71, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d62)
			}
			if d62.Loc == scm.LocReg && d59.Loc == scm.LocReg && d62.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d61)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d62.Imm.Int()) > uint64(64))}
			} else {
				r72 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitCmpRegImm32(d62.Reg, 64)
				ctx.W.EmitSetcc(r72, scm.CcA)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r72}
				ctx.BindReg(r72, &d63)
			}
			ctx.FreeDesc(&d62)
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d63.Loc == scm.LocImm {
				if d63.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
			d64 := d49
			if d64.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			ctx.EmitStoreToStack(d64, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d63.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
			d65 := d49
			if d65.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			ctx.EmitStoreToStack(d65, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d63)
			ctx.W.MarkLabel(lbl17)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d66 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r73 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r73, d44.Reg)
				ctx.W.EmitShrRegImm8(r73, 6)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d66)
			}
			if d66.Loc == scm.LocReg && d44.Loc == scm.LocReg && d66.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d66.Loc == scm.LocStack || d66.Loc == scm.LocStackPair { ctx.EnsureDesc(&d66) }
			var d67 scm.JITValueDesc
			if d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(scratch, d66.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d67)
			}
			if d67.Loc == scm.LocReg && d66.Loc == scm.LocReg && d67.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d66)
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			r74 := ctx.AllocReg()
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d67.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r74, uint64(d67.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r74, d67.Reg)
				ctx.W.EmitShlRegImm8(r74, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r74, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r74, d45.Reg)
			}
			r75 := ctx.AllocRegExcept(r74)
			ctx.W.EmitMovRegMem(r75, r74, 0)
			ctx.FreeReg(r74)
			d68 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r75}
			ctx.BindReg(r75, &d68)
			ctx.FreeDesc(&d67)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d69 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r76 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r76, d44.Reg)
				ctx.W.EmitAndRegImm32(r76, 63)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r76}
				ctx.BindReg(r76, &d69)
			}
			if d69.Loc == scm.LocReg && d44.Loc == scm.LocReg && d69.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			d70 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm && d69.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d70.Imm.Int() - d69.Imm.Int())}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				r77 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r77, d70.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d71)
			} else if d70.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d70.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d69.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(scratch, d70.Reg)
				if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d69.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else {
				r78 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r78, d70.Reg)
				ctx.W.EmitSubInt64(r78, d69.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d71)
			}
			if d71.Loc == scm.LocReg && d70.Loc == scm.LocReg && d71.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d69)
			if d68.Loc == scm.LocStack || d68.Loc == scm.LocStackPair { ctx.EnsureDesc(&d68) }
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			var d72 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d71.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) >> uint64(d71.Imm.Int())))}
			} else if d71.Loc == scm.LocImm {
				r79 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r79, d68.Reg)
				ctx.W.EmitShrRegImm8(r79, uint8(d71.Imm.Int()))
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d72)
			} else {
				{
					shiftSrc := d68.Reg
					r80 := ctx.AllocRegExcept(d68.Reg)
					ctx.W.EmitMovRegReg(r80, d68.Reg)
					shiftSrc = r80
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d71.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d71.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d71.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d72)
				}
			}
			if d72.Loc == scm.LocReg && d68.Loc == scm.LocReg && d72.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d71)
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			var d73 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() | d72.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d72.Reg}
				ctx.BindReg(d72.Reg, &d73)
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r81, d49.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d73)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d72.Loc == scm.LocImm {
				r82 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r82, d49.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r82, int32(d72.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.W.EmitOrInt64(r82, scm.RegR11)
				}
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d73)
			} else {
				r83 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r83, d49.Reg)
				ctx.W.EmitOrInt64(r83, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d73)
			}
			if d73.Loc == scm.LocReg && d49.Loc == scm.LocReg && d73.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			d74 := d73
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			ctx.EmitStoreToStack(d74, 8)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl13)
			ctx.W.ResolveFixups()
			d75 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			ctx.BindReg(r66, &d75)
			ctx.BindReg(r66, &d75)
			if r46 { ctx.UnprotectReg(r47) }
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d75.Imm.Int()))))}
			} else {
				r84 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r84, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d76)
			}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r85 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r85, thisptr.Reg, off)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
				ctx.BindReg(r85, &d77)
			}
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			var d78 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() + d77.Imm.Int())}
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				r86 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(r86, d76.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d78)
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
				ctx.BindReg(d77.Reg, &d78)
			} else if d76.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d76.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(scratch, d76.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else {
				r87 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(r87, d76.Reg)
				ctx.W.EmitAddInt64(r87, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d78)
			}
			if d78.Loc == scm.LocReg && d76.Loc == scm.LocReg && d78.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			ctx.FreeDesc(&d77)
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d78.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d78.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d79)
			}
			ctx.FreeDesc(&d78)
			var d80 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r89, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d80)
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d80.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl21)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl21)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d80)
			ctx.W.MarkLabel(lbl11)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			r90 := d39.Loc == scm.LocReg
			r91 := d39.Reg
			if r90 { ctx.ProtectReg(r91) }
			lbl22 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d81 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d39.Reg)
				ctx.W.EmitShlRegImm8(r92, 32)
				ctx.W.EmitShrRegImm8(r92, 32)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d81)
			}
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d82)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d82.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d82.Reg)
				ctx.W.EmitShlRegImm8(r94, 56)
				ctx.W.EmitShrRegImm8(r94, 56)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d83)
			}
			ctx.FreeDesc(&d82)
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			if d83.Loc == scm.LocStack || d83.Loc == scm.LocStackPair { ctx.EnsureDesc(&d83) }
			var d84 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d81.Imm.Int() * d83.Imm.Int())}
			} else if d81.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d81.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(scratch, d81.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else {
				r95 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r95, d81.Reg)
				ctx.W.EmitImulInt64(r95, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d84)
			}
			if d84.Loc == scm.LocReg && d81.Loc == scm.LocReg && d84.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d83)
			var d85 scm.JITValueDesc
			r96 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r96, uint64(dataPtr))
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96, StackOff: int32(sliceLen)}
				ctx.BindReg(r96, &d85)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r96, thisptr.Reg, off)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
				ctx.BindReg(r96, &d85)
			}
			ctx.BindReg(r96, &d85)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d86 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() / 64)}
			} else {
				r97 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r97, d84.Reg)
				ctx.W.EmitShrRegImm8(r97, 6)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d86)
			}
			if d86.Loc == scm.LocReg && d84.Loc == scm.LocReg && d86.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			r98 := ctx.AllocReg()
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d86.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r98, uint64(d86.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r98, d86.Reg)
				ctx.W.EmitShlRegImm8(r98, 3)
			}
			if d85.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(r98, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r98, d85.Reg)
			}
			r99 := ctx.AllocRegExcept(r98)
			ctx.W.EmitMovRegMem(r99, r98, 0)
			ctx.FreeReg(r98)
			d87 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			ctx.BindReg(r99, &d87)
			ctx.FreeDesc(&d86)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d88 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r100 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r100, d84.Reg)
				ctx.W.EmitAndRegImm32(r100, 63)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d88)
			}
			if d88.Loc == scm.LocReg && d84.Loc == scm.LocReg && d88.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d89 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d87.Imm.Int()) << uint64(d88.Imm.Int())))}
			} else if d88.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r101, d87.Reg)
				ctx.W.EmitShlRegImm8(r101, uint8(d88.Imm.Int()))
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d89)
			} else {
				{
					shiftSrc := d87.Reg
					r102 := ctx.AllocRegExcept(d87.Reg)
					ctx.W.EmitMovRegReg(r102, d87.Reg)
					shiftSrc = r102
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d88.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d88.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d88.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d89)
				}
			}
			if d89.Loc == scm.LocReg && d87.Loc == scm.LocReg && d89.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d88)
			var d90 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r103, thisptr.Reg, off)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
				ctx.BindReg(r103, &d90)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d90.Loc == scm.LocImm {
				if d90.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
			d91 := d89
			if d91.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d91.Loc == scm.LocStack || d91.Loc == scm.LocStackPair { ctx.EnsureDesc(&d91) }
			ctx.EmitStoreToStack(d91, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d90.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
			d92 := d89
			if d92.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair { ctx.EnsureDesc(&d92) }
			ctx.EmitStoreToStack(d92, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d90)
			ctx.W.MarkLabel(lbl24)
			d93 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d94 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r104, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
				ctx.BindReg(r104, &d94)
			}
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d95 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d94.Imm.Int()))))}
			} else {
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r105, d94.Reg)
				ctx.W.EmitShlRegImm8(r105, 56)
				ctx.W.EmitShrRegImm8(r105, 56)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d95)
			}
			ctx.FreeDesc(&d94)
			d96 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			var d97 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d95.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() - d95.Imm.Int())}
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				r106 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r106, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d97)
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d97)
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(scratch, d96.Reg)
				if d95.Imm.Int() >= -2147483648 && d95.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d95.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d95.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d97)
			} else {
				r107 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r107, d96.Reg)
				ctx.W.EmitSubInt64(r107, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d97)
			}
			if d97.Loc == scm.LocReg && d96.Loc == scm.LocReg && d97.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair { ctx.EnsureDesc(&d93) }
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			var d98 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) >> uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				r108 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r108, d93.Reg)
				ctx.W.EmitShrRegImm8(r108, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d98)
			} else {
				{
					shiftSrc := d93.Reg
					r109 := ctx.AllocRegExcept(d93.Reg)
					ctx.W.EmitMovRegReg(r109, d93.Reg)
					shiftSrc = r109
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
					ctx.BindReg(shiftSrc, &d98)
				}
			}
			if d98.Loc == scm.LocReg && d93.Loc == scm.LocReg && d98.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			ctx.FreeDesc(&d97)
			r110 := ctx.AllocReg()
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			if d98.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r110, d98.Reg2)
			} else {
				ctx.EmitMovToReg(r110, d98)
			}
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl23)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d99 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r111 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r111, d84.Reg)
				ctx.W.EmitAndRegImm32(r111, 63)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d99)
			}
			if d99.Loc == scm.LocReg && d84.Loc == scm.LocReg && d99.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			var d100 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r112, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
				ctx.BindReg(r112, &d100)
			}
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d100.Imm.Int()))))}
			} else {
				r113 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r113, d100.Reg)
				ctx.W.EmitShlRegImm8(r113, 56)
				ctx.W.EmitShrRegImm8(r113, 56)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d101)
			}
			ctx.FreeDesc(&d100)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			var d102 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() + d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				r114 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r114, d99.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
				ctx.BindReg(r114, &d102)
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
				ctx.BindReg(d101.Reg, &d102)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d102)
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(scratch, d99.Reg)
				if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d101.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d102)
			} else {
				r115 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r115, d99.Reg)
				ctx.W.EmitAddInt64(r115, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d102)
			}
			if d102.Loc == scm.LocReg && d99.Loc == scm.LocReg && d102.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			ctx.FreeDesc(&d101)
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			var d103 scm.JITValueDesc
			if d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d102.Imm.Int()) > uint64(64))}
			} else {
				r116 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitCmpRegImm32(d102.Reg, 64)
				ctx.W.EmitSetcc(r116, scm.CcA)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
				ctx.BindReg(r116, &d103)
			}
			ctx.FreeDesc(&d102)
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d103.Loc == scm.LocImm {
				if d103.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
			d104 := d89
			if d104.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			ctx.EmitStoreToStack(d104, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d103.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
			d105 := d89
			if d105.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			ctx.EmitStoreToStack(d105, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl26)
			}
			ctx.FreeDesc(&d103)
			ctx.W.MarkLabel(lbl26)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d106 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() / 64)}
			} else {
				r117 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r117, d84.Reg)
				ctx.W.EmitShrRegImm8(r117, 6)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d106)
			}
			if d106.Loc == scm.LocReg && d84.Loc == scm.LocReg && d106.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(scratch, d106.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			}
			if d107.Loc == scm.LocReg && d106.Loc == scm.LocReg && d107.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			r118 := ctx.AllocReg()
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r118, uint64(d107.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r118, d107.Reg)
				ctx.W.EmitShlRegImm8(r118, 3)
			}
			if d85.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(r118, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r118, d85.Reg)
			}
			r119 := ctx.AllocRegExcept(r118)
			ctx.W.EmitMovRegMem(r119, r118, 0)
			ctx.FreeReg(r118)
			d108 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r119}
			ctx.BindReg(r119, &d108)
			ctx.FreeDesc(&d107)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d109 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r120 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r120, d84.Reg)
				ctx.W.EmitAndRegImm32(r120, 63)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d109)
			}
			if d109.Loc == scm.LocReg && d84.Loc == scm.LocReg && d109.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d84)
			d110 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			var d111 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() - d109.Imm.Int())}
			} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
				r121 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r121, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d111)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d110.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else if d109.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(scratch, d110.Reg)
				if d109.Imm.Int() >= -2147483648 && d109.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d109.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d109.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else {
				r122 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r122, d110.Reg)
				ctx.W.EmitSubInt64(r122, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d111)
			}
			if d111.Loc == scm.LocReg && d110.Loc == scm.LocReg && d111.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair { ctx.EnsureDesc(&d108) }
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			var d112 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d108.Imm.Int()) >> uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				r123 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r123, d108.Reg)
				ctx.W.EmitShrRegImm8(r123, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d112)
			} else {
				{
					shiftSrc := d108.Reg
					r124 := ctx.AllocRegExcept(d108.Reg)
					ctx.W.EmitMovRegReg(r124, d108.Reg)
					shiftSrc = r124
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d111.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d111.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d111.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d112)
				}
			}
			if d112.Loc == scm.LocReg && d108.Loc == scm.LocReg && d112.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d111)
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			var d113 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() | d112.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d112.Reg}
				ctx.BindReg(d112.Reg, &d113)
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				r125 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r125, d89.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d113)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else if d112.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r126, d89.Reg)
				if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r126, int32(d112.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
					ctx.W.EmitOrInt64(r126, scm.RegR11)
				}
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d113)
			} else {
				r127 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r127, d89.Reg)
				ctx.W.EmitOrInt64(r127, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d113)
			}
			if d113.Loc == scm.LocReg && d89.Loc == scm.LocReg && d113.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			d114 := d113
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			ctx.EmitStoreToStack(d114, 16)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
			d115 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
			ctx.BindReg(r110, &d115)
			ctx.BindReg(r110, &d115)
			if r90 { ctx.UnprotectReg(r91) }
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d115.Imm.Int()))))}
			} else {
				r128 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r128, d115.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d116)
			}
			ctx.FreeDesc(&d115)
			var d117 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r129, thisptr.Reg, off)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
				ctx.BindReg(r129, &d117)
			}
			if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair { ctx.EnsureDesc(&d116) }
			if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair { ctx.EnsureDesc(&d117) }
			var d118 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + d117.Imm.Int())}
			} else if d117.Loc == scm.LocImm && d117.Imm.Int() == 0 {
				r130 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r130, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d118)
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d117.Reg}
				ctx.BindReg(d117.Reg, &d118)
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else if d117.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(scratch, d116.Reg)
				if d117.Imm.Int() >= -2147483648 && d117.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d117.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d117.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else {
				r131 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r131, d116.Reg)
				ctx.W.EmitAddInt64(r131, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d118)
			}
			if d118.Loc == scm.LocReg && d116.Loc == scm.LocReg && d118.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.FreeDesc(&d117)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			r132 := d39.Loc == scm.LocReg
			r133 := d39.Reg
			if r132 { ctx.ProtectReg(r133) }
			lbl28 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d119 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
			} else {
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r134, d39.Reg)
				ctx.W.EmitShlRegImm8(r134, 32)
				ctx.W.EmitShrRegImm8(r134, 32)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d119)
			}
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r135 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r135, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r135}
				ctx.BindReg(r135, &d120)
			}
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d120.Imm.Int()))))}
			} else {
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r136, d120.Reg)
				ctx.W.EmitShlRegImm8(r136, 56)
				ctx.W.EmitShrRegImm8(r136, 56)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d121)
			}
			ctx.FreeDesc(&d120)
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() * d121.Imm.Int())}
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r137 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r137, d119.Reg)
				ctx.W.EmitImulInt64(r137, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d122)
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			r138 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r138, uint64(dataPtr))
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r138, StackOff: int32(sliceLen)}
				ctx.BindReg(r138, &d123)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r138, thisptr.Reg, off)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r138}
				ctx.BindReg(r138, &d123)
			}
			ctx.BindReg(r138, &d123)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d124 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() / 64)}
			} else {
				r139 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r139, d122.Reg)
				ctx.W.EmitShrRegImm8(r139, 6)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d124)
			}
			if d124.Loc == scm.LocReg && d122.Loc == scm.LocReg && d124.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			r140 := ctx.AllocReg()
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			if d124.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r140, uint64(d124.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r140, d124.Reg)
				ctx.W.EmitShlRegImm8(r140, 3)
			}
			if d123.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
				ctx.W.EmitAddInt64(r140, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r140, d123.Reg)
			}
			r141 := ctx.AllocRegExcept(r140)
			ctx.W.EmitMovRegMem(r141, r140, 0)
			ctx.FreeReg(r140)
			d125 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r141}
			ctx.BindReg(r141, &d125)
			ctx.FreeDesc(&d124)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d126 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r142 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r142, d122.Reg)
				ctx.W.EmitAndRegImm32(r142, 63)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d126)
			}
			if d126.Loc == scm.LocReg && d122.Loc == scm.LocReg && d126.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			var d127 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d125.Imm.Int()) << uint64(d126.Imm.Int())))}
			} else if d126.Loc == scm.LocImm {
				r143 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r143, d125.Reg)
				ctx.W.EmitShlRegImm8(r143, uint8(d126.Imm.Int()))
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d127)
			} else {
				{
					shiftSrc := d125.Reg
					r144 := ctx.AllocRegExcept(d125.Reg)
					ctx.W.EmitMovRegReg(r144, d125.Reg)
					shiftSrc = r144
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d126.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d126.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d126.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d127)
				}
			}
			if d127.Loc == scm.LocReg && d125.Loc == scm.LocReg && d127.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d126)
			var d128 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r145, thisptr.Reg, off)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
				ctx.BindReg(r145, &d128)
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d128.Loc == scm.LocImm {
				if d128.Imm.Bool() {
					ctx.W.EmitJmp(lbl29)
				} else {
			d129 := d127
			if d129.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair { ctx.EnsureDesc(&d129) }
			ctx.EmitStoreToStack(d129, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d128.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
			d130 := d127
			if d130.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d130.Loc == scm.LocStack || d130.Loc == scm.LocStackPair { ctx.EnsureDesc(&d130) }
			ctx.EmitStoreToStack(d130, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d128)
			ctx.W.MarkLabel(lbl30)
			d131 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r146, thisptr.Reg, off)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
				ctx.BindReg(r146, &d132)
			}
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d132.Imm.Int()))))}
			} else {
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r147, d132.Reg)
				ctx.W.EmitShlRegImm8(r147, 56)
				ctx.W.EmitShrRegImm8(r147, 56)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d133)
			}
			ctx.FreeDesc(&d132)
			d134 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d135 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() - d133.Imm.Int())}
			} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
				r148 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r148, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d135)
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d133.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(scratch, d134.Reg)
				if d133.Imm.Int() >= -2147483648 && d133.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d133.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else {
				r149 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r149, d134.Reg)
				ctx.W.EmitSubInt64(r149, d133.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d135)
			}
			if d135.Loc == scm.LocReg && d134.Loc == scm.LocReg && d135.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d133)
			if d131.Loc == scm.LocStack || d131.Loc == scm.LocStackPair { ctx.EnsureDesc(&d131) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			var d136 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d131.Imm.Int()) >> uint64(d135.Imm.Int())))}
			} else if d135.Loc == scm.LocImm {
				r150 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r150, d131.Reg)
				ctx.W.EmitShrRegImm8(r150, uint8(d135.Imm.Int()))
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d136)
			} else {
				{
					shiftSrc := d131.Reg
					r151 := ctx.AllocRegExcept(d131.Reg)
					ctx.W.EmitMovRegReg(r151, d131.Reg)
					shiftSrc = r151
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d135.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d135.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d135.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d136)
				}
			}
			if d136.Loc == scm.LocReg && d131.Loc == scm.LocReg && d136.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.FreeDesc(&d135)
			r152 := ctx.AllocReg()
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d136.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r152, d136.Reg2)
			} else {
				ctx.EmitMovToReg(r152, d136)
			}
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl29)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d137 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r153 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r153, d122.Reg)
				ctx.W.EmitAndRegImm32(r153, 63)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d137)
			}
			if d137.Loc == scm.LocReg && d122.Loc == scm.LocReg && d137.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			var d138 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r154 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r154, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
				ctx.BindReg(r154, &d138)
			}
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			var d139 scm.JITValueDesc
			if d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
			} else {
				r155 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r155, d138.Reg)
				ctx.W.EmitShlRegImm8(r155, 56)
				ctx.W.EmitShrRegImm8(r155, 56)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d139)
			}
			ctx.FreeDesc(&d138)
			if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair { ctx.EnsureDesc(&d137) }
			if d139.Loc == scm.LocStack || d139.Loc == scm.LocStackPair { ctx.EnsureDesc(&d139) }
			var d140 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() + d139.Imm.Int())}
			} else if d139.Loc == scm.LocImm && d139.Imm.Int() == 0 {
				r156 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r156, d137.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d140)
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d139.Reg}
				ctx.BindReg(d139.Reg, &d140)
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(scratch, d137.Reg)
				if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else {
				r157 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r157, d137.Reg)
				ctx.W.EmitAddInt64(r157, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d140)
			}
			if d140.Loc == scm.LocReg && d137.Loc == scm.LocReg && d140.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d139)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			var d141 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d140.Imm.Int()) > uint64(64))}
			} else {
				r158 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitCmpRegImm32(d140.Reg, 64)
				ctx.W.EmitSetcc(r158, scm.CcA)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r158}
				ctx.BindReg(r158, &d141)
			}
			ctx.FreeDesc(&d140)
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d141.Loc == scm.LocImm {
				if d141.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
			d142 := d127
			if d142.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			ctx.EmitStoreToStack(d142, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d141.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl33)
			d143 := d127
			if d143.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			ctx.EmitStoreToStack(d143, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d141)
			ctx.W.MarkLabel(lbl32)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d144 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() / 64)}
			} else {
				r159 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r159, d122.Reg)
				ctx.W.EmitShrRegImm8(r159, 6)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d144)
			}
			if d144.Loc == scm.LocReg && d122.Loc == scm.LocReg && d144.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			var d145 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(scratch, d144.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d145)
			}
			if d145.Loc == scm.LocReg && d144.Loc == scm.LocReg && d145.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			r160 := ctx.AllocReg()
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r160, uint64(d145.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r160, d145.Reg)
				ctx.W.EmitShlRegImm8(r160, 3)
			}
			if d123.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
				ctx.W.EmitAddInt64(r160, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r160, d123.Reg)
			}
			r161 := ctx.AllocRegExcept(r160)
			ctx.W.EmitMovRegMem(r161, r160, 0)
			ctx.FreeReg(r160)
			d146 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r161}
			ctx.BindReg(r161, &d146)
			ctx.FreeDesc(&d145)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d147 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r162 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r162, d122.Reg)
				ctx.W.EmitAndRegImm32(r162, 63)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d147)
			}
			if d147.Loc == scm.LocReg && d122.Loc == scm.LocReg && d147.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			d148 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() - d147.Imm.Int())}
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				r163 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r163, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d149)
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
				r164 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r164, d148.Reg)
				ctx.W.EmitSubInt64(r164, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d149)
			}
			if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			if d146.Loc == scm.LocStack || d146.Loc == scm.LocStackPair { ctx.EnsureDesc(&d146) }
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d150 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d146.Imm.Int()) >> uint64(d149.Imm.Int())))}
			} else if d149.Loc == scm.LocImm {
				r165 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r165, d146.Reg)
				ctx.W.EmitShrRegImm8(r165, uint8(d149.Imm.Int()))
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d150)
			} else {
				{
					shiftSrc := d146.Reg
					r166 := ctx.AllocRegExcept(d146.Reg)
					ctx.W.EmitMovRegReg(r166, d146.Reg)
					shiftSrc = r166
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
			if d150.Loc == scm.LocReg && d146.Loc == scm.LocReg && d150.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d149)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d151 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() | d150.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
				ctx.BindReg(d150.Reg, &d151)
			} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
				r167 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r167, d127.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d151)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else if d150.Loc == scm.LocImm {
				r168 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r168, d127.Reg)
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r168, int32(d150.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.W.EmitOrInt64(r168, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d151)
			} else {
				r169 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r169, d127.Reg)
				ctx.W.EmitOrInt64(r169, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d151)
			}
			if d151.Loc == scm.LocReg && d127.Loc == scm.LocReg && d151.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			d152 := d151
			if d152.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			ctx.EmitStoreToStack(d152, 24)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			ctx.W.ResolveFixups()
			d153 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
			ctx.BindReg(r152, &d153)
			ctx.BindReg(r152, &d153)
			if r132 { ctx.UnprotectReg(r133) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d153.Imm.Int()))))}
			} else {
				r170 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r170, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d154)
			}
			ctx.FreeDesc(&d153)
			var d155 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r171 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r171, thisptr.Reg, off)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r171}
				ctx.BindReg(r171, &d155)
			}
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair { ctx.EnsureDesc(&d155) }
			var d156 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d154.Imm.Int() + d155.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r172 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r172, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d156)
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
				ctx.BindReg(d155.Reg, &d156)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d154.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(scratch, d154.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else {
				r173 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r173, d154.Reg)
				ctx.W.EmitAddInt64(r173, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d156)
			}
			if d156.Loc == scm.LocReg && d154.Loc == scm.LocReg && d156.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.FreeDesc(&d155)
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d156.Loc == scm.LocStack || d156.Loc == scm.LocStackPair { ctx.EnsureDesc(&d156) }
			var d158 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() + d156.Imm.Int())}
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				r174 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r174, d118.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d158)
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d156.Reg}
				ctx.BindReg(d156.Reg, &d158)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d118.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(scratch, d118.Reg)
				if d156.Imm.Int() >= -2147483648 && d156.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d156.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d156.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else {
				r175 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r175, d118.Reg)
				ctx.W.EmitAddInt64(r175, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d158)
			}
			if d158.Loc == scm.LocReg && d118.Loc == scm.LocReg && d158.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d160 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r176 := ctx.AllocReg()
				r177 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r176, fieldAddr)
				ctx.W.EmitMovRegMem64(r177, fieldAddr+8)
				d160 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r176, Reg2: r177}
				ctx.BindReg(r176, &d160)
				ctx.BindReg(r177, &d160)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r178 := ctx.AllocReg()
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r178, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r179, thisptr.Reg, off+8)
				d160 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r178, Reg2: r179}
				ctx.BindReg(r178, &d160)
				ctx.BindReg(r179, &d160)
			}
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			r180 := ctx.AllocReg()
			r181 := ctx.AllocRegExcept(r180)
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r180, uint64(d160.Imm.Int()))
			} else if d160.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r180, d160.Reg)
			} else {
				ctx.W.EmitMovRegReg(r180, d160.Reg)
			}
			if d118.Loc == scm.LocImm {
				if d118.Imm.Int() != 0 {
					if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r180, int32(d118.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
						ctx.W.EmitAddInt64(r180, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r180, d118.Reg)
			}
			if d158.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r181, uint64(d158.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r181, d158.Reg)
			}
			if d118.Loc == scm.LocImm {
				if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r181, int32(d118.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
					ctx.W.EmitSubInt64(r181, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r181, d118.Reg)
			}
			d161 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
			ctx.BindReg(r180, &d161)
			ctx.BindReg(r181, &d161)
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d158)
			d162 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d161}, 2)
			ctx.EmitMovPairToResult(&d162, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			var d163 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r182 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r182, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r182}
				ctx.BindReg(r182, &d163)
			}
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d163.Imm.Int()))))}
			} else {
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r183, d163.Reg)
				ctx.W.EmitShlRegImm8(r183, 32)
				ctx.W.EmitShrRegImm8(r183, 32)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r183}
				ctx.BindReg(r183, &d164)
			}
			ctx.FreeDesc(&d163)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			var d165 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d39.Imm.Int()) == uint64(d164.Imm.Int()))}
			} else if d164.Loc == scm.LocImm {
				r184 := ctx.AllocRegExcept(d39.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d39.Reg, int32(d164.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.W.EmitCmpInt64(d39.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r184, scm.CcE)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r184}
				ctx.BindReg(r184, &d165)
			} else if d39.Loc == scm.LocImm {
				r185 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d164.Reg)
				ctx.W.EmitSetcc(r185, scm.CcE)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r185}
				ctx.BindReg(r185, &d165)
			} else {
				r186 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpInt64(d39.Reg, d164.Reg)
				ctx.W.EmitSetcc(r186, scm.CcE)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r186}
				ctx.BindReg(r186, &d165)
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d164)
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d165.Loc == scm.LocImm {
				if d165.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d165.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d165)
			ctx.W.MarkLabel(lbl20)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			lbl36 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d166 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r187, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r187, 32)
				ctx.W.EmitShrRegImm8(r187, 32)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d166)
			}
			ctx.FreeDesc(&idxInt)
			var d167 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r188, thisptr.Reg, off)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d167)
			}
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d167.Imm.Int()))))}
			} else {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r189, d167.Reg)
				ctx.W.EmitShlRegImm8(r189, 56)
				ctx.W.EmitShrRegImm8(r189, 56)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d168)
			}
			ctx.FreeDesc(&d167)
			if d166.Loc == scm.LocStack || d166.Loc == scm.LocStackPair { ctx.EnsureDesc(&d166) }
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			var d169 scm.JITValueDesc
			if d166.Loc == scm.LocImm && d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d166.Imm.Int() * d168.Imm.Int())}
			} else if d166.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d166.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegReg(scratch, d166.Reg)
				if d168.Imm.Int() >= -2147483648 && d168.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d168.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d168.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else {
				r190 := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegReg(r190, d166.Reg)
				ctx.W.EmitImulInt64(r190, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d169)
			}
			if d169.Loc == scm.LocReg && d166.Loc == scm.LocReg && d169.Reg == d166.Reg {
				ctx.TransferReg(d166.Reg)
				d166.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.FreeDesc(&d168)
			var d170 scm.JITValueDesc
			r191 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r191, uint64(dataPtr))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191, StackOff: int32(sliceLen)}
				ctx.BindReg(r191, &d170)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r191, thisptr.Reg, off)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
				ctx.BindReg(r191, &d170)
			}
			ctx.BindReg(r191, &d170)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d171 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() / 64)}
			} else {
				r192 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r192, d169.Reg)
				ctx.W.EmitShrRegImm8(r192, 6)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d171)
			}
			if d171.Loc == scm.LocReg && d169.Loc == scm.LocReg && d171.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			r193 := ctx.AllocReg()
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d171.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r193, uint64(d171.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r193, d171.Reg)
				ctx.W.EmitShlRegImm8(r193, 3)
			}
			if d170.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
				ctx.W.EmitAddInt64(r193, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r193, d170.Reg)
			}
			r194 := ctx.AllocRegExcept(r193)
			ctx.W.EmitMovRegMem(r194, r193, 0)
			ctx.FreeReg(r193)
			d172 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
			ctx.BindReg(r194, &d172)
			ctx.FreeDesc(&d171)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d173 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() % 64)}
			} else {
				r195 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r195, d169.Reg)
				ctx.W.EmitAndRegImm32(r195, 63)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d173)
			}
			if d173.Loc == scm.LocReg && d169.Loc == scm.LocReg && d173.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			var d174 scm.JITValueDesc
			if d172.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d172.Imm.Int()) << uint64(d173.Imm.Int())))}
			} else if d173.Loc == scm.LocImm {
				r196 := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegReg(r196, d172.Reg)
				ctx.W.EmitShlRegImm8(r196, uint8(d173.Imm.Int()))
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d174)
			} else {
				{
					shiftSrc := d172.Reg
					r197 := ctx.AllocRegExcept(d172.Reg)
					ctx.W.EmitMovRegReg(r197, d172.Reg)
					shiftSrc = r197
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d173.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d173.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d173.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d174)
				}
			}
			if d174.Loc == scm.LocReg && d172.Loc == scm.LocReg && d174.Reg == d172.Reg {
				ctx.TransferReg(d172.Reg)
				d172.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r198, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
				ctx.BindReg(r198, &d175)
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d175.Loc == scm.LocImm {
				if d175.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
			d176 := d174
			if d176.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			ctx.EmitStoreToStack(d176, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d175.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			d177 := d174
			if d177.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			ctx.EmitStoreToStack(d177, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d175)
			ctx.W.MarkLabel(lbl38)
			d178 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d179 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r199, thisptr.Reg, off)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d179)
			}
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d179.Imm.Int()))))}
			} else {
				r200 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r200, d179.Reg)
				ctx.W.EmitShlRegImm8(r200, 56)
				ctx.W.EmitShrRegImm8(r200, 56)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d180)
			}
			ctx.FreeDesc(&d179)
			d181 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() - d180.Imm.Int())}
			} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
				r201 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r201, d181.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d182)
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d181.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d180.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d182)
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(scratch, d181.Reg)
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d182)
			} else {
				r202 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r202, d181.Reg)
				ctx.W.EmitSubInt64(r202, d180.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d182)
			}
			if d182.Loc == scm.LocReg && d181.Loc == scm.LocReg && d182.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			var d183 scm.JITValueDesc
			if d178.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d178.Imm.Int()) >> uint64(d182.Imm.Int())))}
			} else if d182.Loc == scm.LocImm {
				r203 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r203, d178.Reg)
				ctx.W.EmitShrRegImm8(r203, uint8(d182.Imm.Int()))
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d183)
			} else {
				{
					shiftSrc := d178.Reg
					r204 := ctx.AllocRegExcept(d178.Reg)
					ctx.W.EmitMovRegReg(r204, d178.Reg)
					shiftSrc = r204
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d182.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d182.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d182.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d183)
				}
			}
			if d183.Loc == scm.LocReg && d178.Loc == scm.LocReg && d183.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d178)
			ctx.FreeDesc(&d182)
			r205 := ctx.AllocReg()
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			if d183.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r205, d183.Reg2)
			} else {
				ctx.EmitMovToReg(r205, d183)
			}
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl37)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d184 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() % 64)}
			} else {
				r206 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r206, d169.Reg)
				ctx.W.EmitAndRegImm32(r206, 63)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d184)
			}
			if d184.Loc == scm.LocReg && d169.Loc == scm.LocReg && d184.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			var d185 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r207, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
				ctx.BindReg(r207, &d185)
			}
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			var d186 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d185.Imm.Int()))))}
			} else {
				r208 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r208, d185.Reg)
				ctx.W.EmitShlRegImm8(r208, 56)
				ctx.W.EmitShrRegImm8(r208, 56)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d186)
			}
			ctx.FreeDesc(&d185)
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			var d187 scm.JITValueDesc
			if d184.Loc == scm.LocImm && d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() + d186.Imm.Int())}
			} else if d186.Loc == scm.LocImm && d186.Imm.Int() == 0 {
				r209 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r209, d184.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d187)
			} else if d184.Loc == scm.LocImm && d184.Imm.Int() == 0 {
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d186.Reg}
				ctx.BindReg(d186.Reg, &d187)
			} else if d184.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d187)
			} else if d186.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(scratch, d184.Reg)
				if d186.Imm.Int() >= -2147483648 && d186.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d186.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d187)
			} else {
				r210 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r210, d184.Reg)
				ctx.W.EmitAddInt64(r210, d186.Reg)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d187)
			}
			if d187.Loc == scm.LocReg && d184.Loc == scm.LocReg && d187.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d184)
			ctx.FreeDesc(&d186)
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d188 scm.JITValueDesc
			if d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d187.Imm.Int()) > uint64(64))}
			} else {
				r211 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitCmpRegImm32(d187.Reg, 64)
				ctx.W.EmitSetcc(r211, scm.CcA)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r211}
				ctx.BindReg(r211, &d188)
			}
			ctx.FreeDesc(&d187)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d188.Loc == scm.LocImm {
				if d188.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
			d189 := d174
			if d189.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d189.Loc == scm.LocStack || d189.Loc == scm.LocStackPair { ctx.EnsureDesc(&d189) }
			ctx.EmitStoreToStack(d189, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d188.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
			d190 := d174
			if d190.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			ctx.EmitStoreToStack(d190, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d188)
			ctx.W.MarkLabel(lbl40)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d191 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() / 64)}
			} else {
				r212 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r212, d169.Reg)
				ctx.W.EmitShrRegImm8(r212, 6)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d191)
			}
			if d191.Loc == scm.LocReg && d169.Loc == scm.LocReg && d191.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			var d192 scm.JITValueDesc
			if d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(scratch, d191.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d192)
			}
			if d192.Loc == scm.LocReg && d191.Loc == scm.LocReg && d192.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d191)
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			r213 := ctx.AllocReg()
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d192.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r213, uint64(d192.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r213, d192.Reg)
				ctx.W.EmitShlRegImm8(r213, 3)
			}
			if d170.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
				ctx.W.EmitAddInt64(r213, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r213, d170.Reg)
			}
			r214 := ctx.AllocRegExcept(r213)
			ctx.W.EmitMovRegMem(r214, r213, 0)
			ctx.FreeReg(r213)
			d193 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r214}
			ctx.BindReg(r214, &d193)
			ctx.FreeDesc(&d192)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d194 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() % 64)}
			} else {
				r215 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r215, d169.Reg)
				ctx.W.EmitAndRegImm32(r215, 63)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d194)
			}
			if d194.Loc == scm.LocReg && d169.Loc == scm.LocReg && d194.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			d195 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm && d194.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d195.Imm.Int() - d194.Imm.Int())}
			} else if d194.Loc == scm.LocImm && d194.Imm.Int() == 0 {
				r216 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r216, d195.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d196)
			} else if d195.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d195.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d196)
			} else if d194.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(scratch, d195.Reg)
				if d194.Imm.Int() >= -2147483648 && d194.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d194.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d194.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d196)
			} else {
				r217 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r217, d195.Reg)
				ctx.W.EmitSubInt64(r217, d194.Reg)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d196)
			}
			if d196.Loc == scm.LocReg && d195.Loc == scm.LocReg && d196.Reg == d195.Reg {
				ctx.TransferReg(d195.Reg)
				d195.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair { ctx.EnsureDesc(&d196) }
			var d197 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d193.Imm.Int()) >> uint64(d196.Imm.Int())))}
			} else if d196.Loc == scm.LocImm {
				r218 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r218, d193.Reg)
				ctx.W.EmitShrRegImm8(r218, uint8(d196.Imm.Int()))
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d197)
			} else {
				{
					shiftSrc := d193.Reg
					r219 := ctx.AllocRegExcept(d193.Reg)
					ctx.W.EmitMovRegReg(r219, d193.Reg)
					shiftSrc = r219
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d196.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d196.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d196.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d197)
				}
			}
			if d197.Loc == scm.LocReg && d193.Loc == scm.LocReg && d197.Reg == d193.Reg {
				ctx.TransferReg(d193.Reg)
				d193.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d193)
			ctx.FreeDesc(&d196)
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d198 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() | d197.Imm.Int())}
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d197.Reg}
				ctx.BindReg(d197.Reg, &d198)
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r220, d174.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d198)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else if d197.Loc == scm.LocImm {
				r221 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r221, d174.Reg)
				if d197.Imm.Int() >= -2147483648 && d197.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r221, int32(d197.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d197.Imm.Int()))
					ctx.W.EmitOrInt64(r221, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d198)
			} else {
				r222 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r222, d174.Reg)
				ctx.W.EmitOrInt64(r222, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d198)
			}
			if d198.Loc == scm.LocReg && d174.Loc == scm.LocReg && d198.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			d199 := d198
			if d199.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			ctx.EmitStoreToStack(d199, 32)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			ctx.W.ResolveFixups()
			d200 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
			ctx.BindReg(r205, &d200)
			ctx.BindReg(r205, &d200)
			ctx.FreeDesc(&idxInt)
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			var d201 scm.JITValueDesc
			if d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d200.Imm.Int()))))}
			} else {
				r223 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r223, d200.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d201)
			}
			ctx.FreeDesc(&d200)
			var d202 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r224 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r224, thisptr.Reg, off)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r224}
				ctx.BindReg(r224, &d202)
			}
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			var d203 scm.JITValueDesc
			if d201.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() + d202.Imm.Int())}
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				r225 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r225, d201.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d203)
			} else if d201.Loc == scm.LocImm && d201.Imm.Int() == 0 {
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d202.Reg}
				ctx.BindReg(d202.Reg, &d203)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d201.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(scratch, d201.Reg)
				if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d202.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r226 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r226, d201.Reg)
				ctx.W.EmitAddInt64(r226, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d203)
			}
			if d203.Loc == scm.LocReg && d201.Loc == scm.LocReg && d203.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			ctx.FreeDesc(&d202)
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			var d204 scm.JITValueDesc
			if d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d203.Imm.Int()))))}
			} else {
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r227, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d204)
			}
			ctx.FreeDesc(&d203)
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			var d205 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d79.Imm.Int()))))}
			} else {
				r228 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r228, d79.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d205)
			}
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			var d206 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d204.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() + d204.Imm.Int())}
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				r229 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r229, d79.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d206)
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d204.Reg}
				ctx.BindReg(d204.Reg, &d206)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d204.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else if d204.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				if d204.Imm.Int() >= -2147483648 && d204.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d204.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d204.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d206)
			} else {
				r230 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r230, d79.Reg)
				ctx.W.EmitAddInt64(r230, d204.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d206)
			}
			if d206.Loc == scm.LocReg && d79.Loc == scm.LocReg && d206.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d206.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d207)
			}
			ctx.FreeDesc(&d206)
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			r232 := ctx.AllocReg()
			r233 := ctx.AllocRegExcept(r232)
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r232, uint64(d160.Imm.Int()))
			} else if d160.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r232, d160.Reg)
			} else {
				ctx.W.EmitMovRegReg(r232, d160.Reg)
			}
			if d205.Loc == scm.LocImm {
				if d205.Imm.Int() != 0 {
					if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r232, int32(d205.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
						ctx.W.EmitAddInt64(r232, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r232, d205.Reg)
			}
			if d207.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r233, uint64(d207.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r233, d207.Reg)
			}
			if d205.Loc == scm.LocImm {
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r233, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitSubInt64(r233, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r233, d205.Reg)
			}
			d208 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r232, Reg2: r233}
			ctx.BindReg(r232, &d208)
			ctx.BindReg(r233, &d208)
			ctx.FreeDesc(&d205)
			ctx.FreeDesc(&d207)
			d209 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d208}, 2)
			ctx.EmitMovPairToResult(&d209, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			var d210 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r234, thisptr.Reg, off)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r234}
				ctx.BindReg(r234, &d210)
			}
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			var d211 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) == uint64(d210.Imm.Int()))}
			} else if d210.Loc == scm.LocImm {
				r235 := ctx.AllocRegExcept(d79.Reg)
				if d210.Imm.Int() >= -2147483648 && d210.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d79.Reg, int32(d210.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d210.Imm.Int()))
					ctx.W.EmitCmpInt64(d79.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r235, scm.CcE)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r235}
				ctx.BindReg(r235, &d211)
			} else if d79.Loc == scm.LocImm {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d210.Reg)
				ctx.W.EmitSetcc(r236, scm.CcE)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r236}
				ctx.BindReg(r236, &d211)
			} else {
				r237 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitCmpInt64(d79.Reg, d210.Reg)
				ctx.W.EmitSetcc(r237, scm.CcE)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r237}
				ctx.BindReg(r237, &d211)
			}
			ctx.FreeDesc(&d79)
			ctx.FreeDesc(&d210)
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d211.Loc == scm.LocImm {
				if d211.Imm.Bool() {
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d211.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d211)
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
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
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
