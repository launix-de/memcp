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
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
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
			ctx.EnsureDesc(&idxInt)
			d1 := idxInt
			_ = d1
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 := ctx.W.EmitSubRSP32Fixup()
			lbl4 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d1)
			var d2 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d1.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r6, d1.Reg)
				ctx.W.EmitShlRegImm8(r6, 32)
				ctx.W.EmitShrRegImm8(r6, 32)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d2)
			}
			var d3 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r7, thisptr.Reg, off)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d3)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d3.Reg)
				ctx.W.EmitShlRegImm8(r8, 56)
				ctx.W.EmitShrRegImm8(r8, 56)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d4)
			}
			ctx.FreeDesc(&d3)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d4)
			var d5 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() * d4.Imm.Int())}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(scratch, d2.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else {
				r9 := ctx.AllocRegExcept(d2.Reg, d4.Reg)
				ctx.W.EmitMovRegReg(r9, d2.Reg)
				ctx.W.EmitImulInt64(r9, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d5)
			}
			if d5.Loc == scm.LocReg && d2.Loc == scm.LocReg && d5.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d4)
			var d6 scm.JITValueDesc
			r10 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r10, uint64(dataPtr))
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10, StackOff: int32(sliceLen)}
				ctx.BindReg(r10, &d6)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 0)
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d6)
			}
			ctx.BindReg(r10, &d6)
			ctx.EnsureDesc(&d5)
			var d7 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r11, d5.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d7)
			}
			if d7.Loc == scm.LocReg && d5.Loc == scm.LocReg && d7.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			r12 := ctx.AllocReg()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d7.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d7.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r12, d6.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d8 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d8)
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d5)
			var d9 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r14, d5.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d9)
			}
			if d9.Loc == scm.LocReg && d5.Loc == scm.LocReg && d9.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d9)
			var d10 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) << uint64(d9.Imm.Int())))}
			} else if d9.Loc == scm.LocImm {
				r15 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r15, d8.Reg)
				ctx.W.EmitShlRegImm8(r15, uint8(d9.Imm.Int()))
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d10)
			} else {
				{
					shiftSrc := d8.Reg
					r16 := ctx.AllocRegExcept(d8.Reg)
					ctx.W.EmitMovRegReg(r16, d8.Reg)
					shiftSrc = r16
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d9.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d9.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d9.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d10)
				}
			}
			if d10.Loc == scm.LocReg && d8.Loc == scm.LocReg && d10.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d9)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 25)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d11)
			}
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d11.Loc == scm.LocImm {
				if d11.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
			d12 := d10
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			ctx.EmitStoreToStack(d12, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
			d13 := d10
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d11)
			ctx.W.MarkLabel(lbl6)
			d14 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d15 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d15)
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			var d16 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d15.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d15.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d16)
			}
			ctx.FreeDesc(&d15)
			d17 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d16)
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() - d16.Imm.Int())}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r20, d17.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d18)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else {
				r21 := ctx.AllocRegExcept(d17.Reg, d16.Reg)
				ctx.W.EmitMovRegReg(r21, d17.Reg)
				ctx.W.EmitSubInt64(r21, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d18)
			}
			if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d14)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) >> uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r22, d14.Reg)
				ctx.W.EmitShrRegImm8(r22, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d19)
			} else {
				{
					shiftSrc := d14.Reg
					r23 := ctx.AllocRegExcept(d14.Reg)
					ctx.W.EmitMovRegReg(r23, d14.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d14.Loc == scm.LocReg && d19.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			ctx.FreeDesc(&d18)
			r24 := ctx.AllocReg()
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r24, d19)
			}
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl5)
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d20 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r25, d5.Reg)
				ctx.W.EmitAndRegImm32(r25, 63)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d20)
			}
			if d20.Loc == scm.LocReg && d5.Loc == scm.LocReg && d20.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d21)
			}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d21.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d22)
			}
			ctx.FreeDesc(&d21)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d22)
			var d23 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() + d22.Imm.Int())}
			} else if d22.Loc == scm.LocImm && d22.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(r28, d20.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d23)
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r29 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.W.EmitMovRegReg(r29, d20.Reg)
				ctx.W.EmitAddInt64(r29, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d23)
			}
			if d23.Loc == scm.LocReg && d20.Loc == scm.LocReg && d23.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			ctx.FreeDesc(&d22)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d23.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitCmpRegImm32(d23.Reg, 64)
				ctx.W.EmitSetcc(r30, scm.CcA)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d24)
			}
			ctx.FreeDesc(&d23)
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d24.Loc == scm.LocImm {
				if d24.Imm.Bool() {
					ctx.W.EmitJmp(lbl8)
				} else {
			d25 := d10
			if d25.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d25)
			ctx.EmitStoreToStack(d25, 0)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d24.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
			d26 := d10
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 0)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
			}
			ctx.FreeDesc(&d24)
			ctx.W.MarkLabel(lbl8)
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d27 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r31, d5.Reg)
				ctx.W.EmitShrRegImm8(r31, 6)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d27)
			}
			if d27.Loc == scm.LocReg && d5.Loc == scm.LocReg && d27.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			}
			if d28.Loc == scm.LocReg && d27.Loc == scm.LocReg && d28.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.EnsureDesc(&d28)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d6)
			if d28.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r32, uint64(d28.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r32, d28.Reg)
				ctx.W.EmitShlRegImm8(r32, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.W.EmitAddInt64(r32, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r32, d6.Reg)
			}
			r33 := ctx.AllocRegExcept(r32)
			ctx.W.EmitMovRegMem(r33, r32, 0)
			ctx.FreeReg(r32)
			d29 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d29)
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d5)
			var d30 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegReg(r34, d5.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d30)
			}
			if d30.Loc == scm.LocReg && d5.Loc == scm.LocReg && d30.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			d31 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d30)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d30.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() - d30.Imm.Int())}
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r35, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d32)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d30.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(scratch, d31.Reg)
				if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d30.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else {
				r36 := ctx.AllocRegExcept(d31.Reg, d30.Reg)
				ctx.W.EmitMovRegReg(r36, d31.Reg)
				ctx.W.EmitSubInt64(r36, d30.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d32)
			}
			if d32.Loc == scm.LocReg && d31.Loc == scm.LocReg && d32.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d32)
			var d33 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d29.Imm.Int()) >> uint64(d32.Imm.Int())))}
			} else if d32.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r37, d29.Reg)
				ctx.W.EmitShrRegImm8(r37, uint8(d32.Imm.Int()))
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d33)
			} else {
				{
					shiftSrc := d29.Reg
					r38 := ctx.AllocRegExcept(d29.Reg)
					ctx.W.EmitMovRegReg(r38, d29.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d32.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d32.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d32.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d33)
				}
			}
			if d33.Loc == scm.LocReg && d29.Loc == scm.LocReg && d33.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d33.Imm.Int())}
			} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
				ctx.BindReg(d33.Reg, &d34)
			} else if d33.Loc == scm.LocImm && d33.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r39, d10.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d34)
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d33.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			} else if d33.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r40, d10.Reg)
				if d33.Imm.Int() >= -2147483648 && d33.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r40, int32(d33.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d33.Imm.Int()))
					ctx.W.EmitOrInt64(r40, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d34)
			} else {
				r41 := ctx.AllocRegExcept(d10.Reg, d33.Reg)
				ctx.W.EmitMovRegReg(r41, d10.Reg)
				ctx.W.EmitOrInt64(r41, d33.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d34)
			}
			if d34.Loc == scm.LocReg && d10.Loc == scm.LocReg && d34.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			d35 := d34
			if d35.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			ctx.EmitStoreToStack(d35, 0)
			ctx.W.EmitJmp(lbl6)
			ctx.W.MarkLabel(lbl4)
			d36 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d36)
			ctx.BindReg(r24, &d36)
			if r3 { ctx.UnprotectReg(r4) }
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d36.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d36.Reg)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d37)
			}
			ctx.FreeDesc(&d36)
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			var d39 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r44, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d39)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d38.Reg}
				ctx.BindReg(d38.Reg, &d39)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(scratch, d37.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r45 := ctx.AllocRegExcept(d37.Reg, d38.Reg)
				ctx.W.EmitMovRegReg(r45, d37.Reg)
				ctx.W.EmitAddInt64(r45, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d39)
			}
			if d39.Loc == scm.LocReg && d37.Loc == scm.LocReg && d39.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d39.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r46, d39.Reg)
				ctx.W.EmitShlRegImm8(r46, 32)
				ctx.W.EmitShrRegImm8(r46, 32)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d40)
			}
			ctx.FreeDesc(&d39)
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d41)
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d41.Loc == scm.LocImm {
				if d41.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d41.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d41)
			ctx.W.MarkLabel(lbl1)
			ctx.EnsureDesc(&idxInt)
			d42 := idxInt
			_ = d42
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			lbl13 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d42.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, d42.Reg)
				ctx.W.EmitShlRegImm8(r50, 32)
				ctx.W.EmitShrRegImm8(r50, 32)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d43)
			}
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r51, thisptr.Reg, off)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d44)
			}
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d44.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d44.Reg)
				ctx.W.EmitShlRegImm8(r52, 56)
				ctx.W.EmitShrRegImm8(r52, 56)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d45)
			}
			ctx.FreeDesc(&d44)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d45)
			var d46 scm.JITValueDesc
			if d43.Loc == scm.LocImm && d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() * d45.Imm.Int())}
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(scratch, d43.Reg)
				if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d45.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else {
				r53 := ctx.AllocRegExcept(d43.Reg, d45.Reg)
				ctx.W.EmitMovRegReg(r53, d43.Reg)
				ctx.W.EmitImulInt64(r53, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d46)
			}
			if d46.Loc == scm.LocReg && d43.Loc == scm.LocReg && d46.Reg == d43.Reg {
				ctx.TransferReg(d43.Reg)
				d43.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d43)
			ctx.FreeDesc(&d45)
			var d47 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r54, uint64(dataPtr))
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d47)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d47)
			}
			ctx.BindReg(r54, &d47)
			ctx.EnsureDesc(&d46)
			var d48 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r55, d46.Reg)
				ctx.W.EmitShrRegImm8(r55, 6)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d48)
			}
			if d48.Loc == scm.LocReg && d46.Loc == scm.LocReg && d48.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d48)
			r56 := ctx.AllocReg()
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			if d48.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r56, uint64(d48.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r56, d48.Reg)
				ctx.W.EmitShlRegImm8(r56, 3)
			}
			if d47.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r56, d47.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.W.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d49 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d49)
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d46)
			var d50 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r58, d46.Reg)
				ctx.W.EmitAndRegImm32(r58, 63)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d50)
			}
			if d50.Loc == scm.LocReg && d46.Loc == scm.LocReg && d50.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d49.Imm.Int()) << uint64(d50.Imm.Int())))}
			} else if d50.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r59, d49.Reg)
				ctx.W.EmitShlRegImm8(r59, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d51)
			} else {
				{
					shiftSrc := d49.Reg
					r60 := ctx.AllocRegExcept(d49.Reg)
					ctx.W.EmitMovRegReg(r60, d49.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d50.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d50.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d50.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d51)
				}
			}
			if d51.Loc == scm.LocReg && d49.Loc == scm.LocReg && d51.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.FreeDesc(&d50)
			var d52 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d52)
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d52.Loc == scm.LocImm {
				if d52.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
			d53 := d51
			if d53.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d53)
			ctx.EmitStoreToStack(d53, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d52.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
			d54 := d51
			if d54.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d54)
			ctx.EmitStoreToStack(d54, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d52)
			ctx.W.MarkLabel(lbl15)
			d55 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d56 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d56)
			}
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d56)
			var d57 scm.JITValueDesc
			if d56.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d56.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d56.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d57)
			}
			ctx.FreeDesc(&d56)
			d58 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d57)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() - d57.Imm.Int())}
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(r64, d58.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d59)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d57.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d59)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(scratch, d58.Reg)
				if d57.Imm.Int() >= -2147483648 && d57.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d57.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d57.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d59)
			} else {
				r65 := ctx.AllocRegExcept(d58.Reg, d57.Reg)
				ctx.W.EmitMovRegReg(r65, d58.Reg)
				ctx.W.EmitSubInt64(r65, d57.Reg)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d59)
			}
			if d59.Loc == scm.LocReg && d58.Loc == scm.LocReg && d59.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d55.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d55.Imm.Int()) >> uint64(d59.Imm.Int())))}
			} else if d59.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegReg(r66, d55.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d59.Imm.Int()))
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d60)
			} else {
				{
					shiftSrc := d55.Reg
					r67 := ctx.AllocRegExcept(d55.Reg)
					ctx.W.EmitMovRegReg(r67, d55.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d59.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d59.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d59.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d60)
				}
			}
			if d60.Loc == scm.LocReg && d55.Loc == scm.LocReg && d60.Reg == d55.Reg {
				ctx.TransferReg(d55.Reg)
				d55.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			ctx.FreeDesc(&d59)
			r68 := ctx.AllocReg()
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d60)
			if d60.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r68, d60)
			}
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl14)
			d55 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d46)
			var d61 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r69, d46.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d61)
			}
			if d61.Loc == scm.LocReg && d46.Loc == scm.LocReg && d61.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			var d62 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d62)
			}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d62.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d62.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d63)
			}
			ctx.FreeDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d63)
			var d64 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() + d63.Imm.Int())}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r72, d61.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d64)
			} else if d61.Loc == scm.LocImm && d61.Imm.Int() == 0 {
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d63.Reg}
				ctx.BindReg(d63.Reg, &d64)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d61.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(scratch, d61.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d63.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else {
				r73 := ctx.AllocRegExcept(d61.Reg, d63.Reg)
				ctx.W.EmitMovRegReg(r73, d61.Reg)
				ctx.W.EmitAddInt64(r73, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d64)
			}
			if d64.Loc == scm.LocReg && d61.Loc == scm.LocReg && d64.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d61)
			ctx.FreeDesc(&d63)
			ctx.EnsureDesc(&d64)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d64.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitCmpRegImm32(d64.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d65)
			}
			ctx.FreeDesc(&d64)
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d65.Loc == scm.LocImm {
				if d65.Imm.Bool() {
					ctx.W.EmitJmp(lbl17)
				} else {
			d66 := d51
			if d66.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			ctx.EmitStoreToStack(d66, 8)
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d65.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
			d67 := d51
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d67)
			ctx.EmitStoreToStack(d67, 8)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d65)
			ctx.W.MarkLabel(lbl17)
			d55 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d46)
			var d68 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r75, d46.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d68)
			}
			if d68.Loc == scm.LocReg && d46.Loc == scm.LocReg && d68.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d68)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(scratch, d68.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d69)
			}
			if d69.Loc == scm.LocReg && d68.Loc == scm.LocReg && d69.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d69)
			r76 := ctx.AllocReg()
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d47)
			if d69.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d69.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d69.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d47.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d47.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d70 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d70)
			ctx.FreeDesc(&d69)
			ctx.EnsureDesc(&d46)
			var d71 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r78, d46.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d71)
			}
			if d71.Loc == scm.LocReg && d46.Loc == scm.LocReg && d71.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			d72 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d71)
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm && d71.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d72.Imm.Int() - d71.Imm.Int())}
			} else if d71.Loc == scm.LocImm && d71.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(r79, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d73)
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d72.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d71.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d71.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegReg(scratch, d72.Reg)
				if d71.Imm.Int() >= -2147483648 && d71.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d71.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else {
				r80 := ctx.AllocRegExcept(d72.Reg, d71.Reg)
				ctx.W.EmitMovRegReg(r80, d72.Reg)
				ctx.W.EmitSubInt64(r80, d71.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d73)
			}
			if d73.Loc == scm.LocReg && d72.Loc == scm.LocReg && d73.Reg == d72.Reg {
				ctx.TransferReg(d72.Reg)
				d72.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d70)
			ctx.EnsureDesc(&d73)
			var d74 scm.JITValueDesc
			if d70.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d70.Imm.Int()) >> uint64(d73.Imm.Int())))}
			} else if d73.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r81, d70.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d73.Imm.Int()))
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d74)
			} else {
				{
					shiftSrc := d70.Reg
					r82 := ctx.AllocRegExcept(d70.Reg)
					ctx.W.EmitMovRegReg(r82, d70.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d73.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d73.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d73.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d74)
				}
			}
			if d74.Loc == scm.LocReg && d70.Loc == scm.LocReg && d74.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d70)
			ctx.FreeDesc(&d73)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d74)
			var d75 scm.JITValueDesc
			if d51.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() | d74.Imm.Int())}
			} else if d51.Loc == scm.LocImm && d51.Imm.Int() == 0 {
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
				ctx.BindReg(d74.Reg, &d75)
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r83, d51.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d75)
			} else if d51.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d51.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d75)
			} else if d74.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r84, d51.Reg)
				if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d74.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d75)
			} else {
				r85 := ctx.AllocRegExcept(d51.Reg, d74.Reg)
				ctx.W.EmitMovRegReg(r85, d51.Reg)
				ctx.W.EmitOrInt64(r85, d74.Reg)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d75)
			}
			if d75.Loc == scm.LocReg && d51.Loc == scm.LocReg && d75.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			d76 := d75
			if d76.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d76)
			ctx.EmitStoreToStack(d76, 8)
			ctx.W.EmitJmp(lbl15)
			ctx.W.MarkLabel(lbl13)
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d77)
			ctx.BindReg(r68, &d77)
			if r48 { ctx.UnprotectReg(r49) }
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d77)
			var d78 scm.JITValueDesc
			if d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d77.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d78)
			}
			ctx.FreeDesc(&d77)
			var d79 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d79)
			}
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			var d80 scm.JITValueDesc
			if d78.Loc == scm.LocImm && d79.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() + d79.Imm.Int())}
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r88, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d80)
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d79.Reg}
				ctx.BindReg(d79.Reg, &d80)
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(scratch, d78.Reg)
				if d79.Imm.Int() >= -2147483648 && d79.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d79.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r89 := ctx.AllocRegExcept(d78.Reg, d79.Reg)
				ctx.W.EmitMovRegReg(r89, d78.Reg)
				ctx.W.EmitAddInt64(r89, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d80)
			}
			if d80.Loc == scm.LocReg && d78.Loc == scm.LocReg && d80.Reg == d78.Reg {
				ctx.TransferReg(d78.Reg)
				d78.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d78)
			ctx.FreeDesc(&d79)
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d80.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d81)
			}
			ctx.FreeDesc(&d80)
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d82)
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			if d82.Loc == scm.LocImm {
				if d82.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d82.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl21)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl21)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.FreeDesc(&d82)
			ctx.W.MarkLabel(lbl11)
			ctx.EnsureDesc(&d40)
			d83 := d40
			_ = d83
			r92 := d40.Loc == scm.LocReg
			r93 := d40.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl22 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d83)
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d83.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d83.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d84)
			}
			var d85 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d85)
			}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d85)
			var d86 scm.JITValueDesc
			if d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d85.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d85.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d86)
			}
			ctx.FreeDesc(&d85)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() * d86.Imm.Int())}
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else {
				r97 := ctx.AllocRegExcept(d84.Reg, d86.Reg)
				ctx.W.EmitMovRegReg(r97, d84.Reg)
				ctx.W.EmitImulInt64(r97, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d87)
			}
			if d87.Loc == scm.LocReg && d84.Loc == scm.LocReg && d87.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d84)
			ctx.FreeDesc(&d86)
			var d88 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d88)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d88)
			}
			ctx.BindReg(r98, &d88)
			ctx.EnsureDesc(&d87)
			var d89 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r99, d87.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d89)
			}
			if d89.Loc == scm.LocReg && d87.Loc == scm.LocReg && d89.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d89)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d88)
			if d89.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d89.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d89.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d88.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d88.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d90 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d90)
			ctx.FreeDesc(&d89)
			ctx.EnsureDesc(&d87)
			var d91 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r102, d87.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d91)
			}
			if d91.Loc == scm.LocReg && d87.Loc == scm.LocReg && d91.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d90.Loc == scm.LocImm && d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d90.Imm.Int()) << uint64(d91.Imm.Int())))}
			} else if d91.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegReg(r103, d90.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d91.Imm.Int()))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d92)
			} else {
				{
					shiftSrc := d90.Reg
					r104 := ctx.AllocRegExcept(d90.Reg)
					ctx.W.EmitMovRegReg(r104, d90.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d91.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d91.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d91.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d92)
				}
			}
			if d92.Loc == scm.LocReg && d90.Loc == scm.LocReg && d92.Reg == d90.Reg {
				ctx.TransferReg(d90.Reg)
				d90.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d90)
			ctx.FreeDesc(&d91)
			var d93 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d93)
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			if d93.Loc == scm.LocImm {
				if d93.Imm.Bool() {
					ctx.W.EmitJmp(lbl23)
				} else {
			d94 := d92
			if d94.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d94)
			ctx.EmitStoreToStack(d94, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d93.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
			d95 := d92
			if d95.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d95)
			ctx.EmitStoreToStack(d95, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.FreeDesc(&d93)
			ctx.W.MarkLabel(lbl24)
			d96 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d97 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d97)
			}
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d97)
			var d98 scm.JITValueDesc
			if d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d97.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d97.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d98)
			}
			ctx.FreeDesc(&d97)
			d99 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d99)
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d99)
			ctx.EnsureDesc(&d98)
			var d100 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d98.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() - d98.Imm.Int())}
			} else if d98.Loc == scm.LocImm && d98.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r108, d99.Reg)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d100)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d98.Reg)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d100)
			} else if d98.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(scratch, d99.Reg)
				if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d98.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d100)
			} else {
				r109 := ctx.AllocRegExcept(d99.Reg, d98.Reg)
				ctx.W.EmitMovRegReg(r109, d99.Reg)
				ctx.W.EmitSubInt64(r109, d98.Reg)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d100)
			}
			if d100.Loc == scm.LocReg && d99.Loc == scm.LocReg && d100.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d98)
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d100)
			var d101 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d96.Imm.Int()) >> uint64(d100.Imm.Int())))}
			} else if d100.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r110, d96.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d100.Imm.Int()))
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d101)
			} else {
				{
					shiftSrc := d96.Reg
					r111 := ctx.AllocRegExcept(d96.Reg)
					ctx.W.EmitMovRegReg(r111, d96.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d100.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d100.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d100.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d101)
				}
			}
			if d101.Loc == scm.LocReg && d96.Loc == scm.LocReg && d101.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d96)
			ctx.FreeDesc(&d100)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d101)
			if d101.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d101)
			}
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl23)
			d96 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d87)
			var d102 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r113, d87.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d102)
			}
			if d102.Loc == scm.LocReg && d87.Loc == scm.LocReg && d102.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			var d103 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d103)
			}
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d103.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d103.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d104)
			}
			ctx.FreeDesc(&d103)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d104)
			var d105 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() + d104.Imm.Int())}
			} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r116, d102.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d105)
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d104.Reg}
				ctx.BindReg(d104.Reg, &d105)
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d104.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d105)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(scratch, d102.Reg)
				if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d104.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d105)
			} else {
				r117 := ctx.AllocRegExcept(d102.Reg, d104.Reg)
				ctx.W.EmitMovRegReg(r117, d102.Reg)
				ctx.W.EmitAddInt64(r117, d104.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d105)
			}
			if d105.Loc == scm.LocReg && d102.Loc == scm.LocReg && d105.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d104)
			ctx.EnsureDesc(&d105)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d105.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitCmpRegImm32(d105.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d106)
			}
			ctx.FreeDesc(&d105)
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d106.Loc == scm.LocImm {
				if d106.Imm.Bool() {
					ctx.W.EmitJmp(lbl26)
				} else {
			d107 := d92
			if d107.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d107)
			ctx.EmitStoreToStack(d107, 16)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d106.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl27)
			d108 := d92
			if d108.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d108)
			ctx.EmitStoreToStack(d108, 16)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl27)
				ctx.W.EmitJmp(lbl26)
			}
			ctx.FreeDesc(&d106)
			ctx.W.MarkLabel(lbl26)
			d96 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d87)
			var d109 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r119, d87.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d109)
			}
			if d109.Loc == scm.LocReg && d87.Loc == scm.LocReg && d109.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d109.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegReg(scratch, d109.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			}
			if d110.Loc == scm.LocReg && d109.Loc == scm.LocReg && d110.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d110)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d88)
			if d110.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d110.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d110.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d88.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d88.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d111 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d111)
			ctx.FreeDesc(&d110)
			ctx.EnsureDesc(&d87)
			var d112 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r122, d87.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d112)
			}
			if d112.Loc == scm.LocReg && d87.Loc == scm.LocReg && d112.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			d113 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d112)
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() - d112.Imm.Int())}
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(r123, d113.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d114)
			} else if d113.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d113.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d112.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d114)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(scratch, d113.Reg)
				if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d112.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d114)
			} else {
				r124 := ctx.AllocRegExcept(d113.Reg, d112.Reg)
				ctx.W.EmitMovRegReg(r124, d113.Reg)
				ctx.W.EmitSubInt64(r124, d112.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d114)
			}
			if d114.Loc == scm.LocReg && d113.Loc == scm.LocReg && d114.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d114)
			var d115 scm.JITValueDesc
			if d111.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d111.Imm.Int()) >> uint64(d114.Imm.Int())))}
			} else if d114.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r125, d111.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d114.Imm.Int()))
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d115)
			} else {
				{
					shiftSrc := d111.Reg
					r126 := ctx.AllocRegExcept(d111.Reg)
					ctx.W.EmitMovRegReg(r126, d111.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d114.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d114.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d114.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d115)
				}
			}
			if d115.Loc == scm.LocReg && d111.Loc == scm.LocReg && d115.Reg == d111.Reg {
				ctx.TransferReg(d111.Reg)
				d111.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d111)
			ctx.FreeDesc(&d114)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d115)
			var d116 scm.JITValueDesc
			if d92.Loc == scm.LocImm && d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() | d115.Imm.Int())}
			} else if d92.Loc == scm.LocImm && d92.Imm.Int() == 0 {
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d115.Reg}
				ctx.BindReg(d115.Reg, &d116)
			} else if d115.Loc == scm.LocImm && d115.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r127, d92.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d116)
			} else if d92.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d92.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d115.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			} else if d115.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r128, d92.Reg)
				if d115.Imm.Int() >= -2147483648 && d115.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d115.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d115.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d116)
			} else {
				r129 := ctx.AllocRegExcept(d92.Reg, d115.Reg)
				ctx.W.EmitMovRegReg(r129, d92.Reg)
				ctx.W.EmitOrInt64(r129, d115.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d116)
			}
			if d116.Loc == scm.LocReg && d92.Loc == scm.LocReg && d116.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			d117 := d116
			if d117.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d117)
			ctx.EmitStoreToStack(d117, 16)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl22)
			d118 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d118)
			ctx.BindReg(r112, &d118)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d118)
			var d119 scm.JITValueDesc
			if d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d118.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d118.Reg)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d119)
			}
			ctx.FreeDesc(&d118)
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d120)
			}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			var d121 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r132, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d121)
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
				ctx.BindReg(d120.Reg, &d121)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d120.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else {
				r133 := ctx.AllocRegExcept(d119.Reg, d120.Reg)
				ctx.W.EmitMovRegReg(r133, d119.Reg)
				ctx.W.EmitAddInt64(r133, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d121)
			}
			if d121.Loc == scm.LocReg && d119.Loc == scm.LocReg && d121.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d120)
			ctx.EnsureDesc(&d40)
			d122 := d40
			_ = d122
			r134 := d40.Loc == scm.LocReg
			r135 := d40.Reg
			if r134 { ctx.ProtectReg(r135) }
			lbl28 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d122.Imm.Int()))))}
			} else {
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r136, d122.Reg)
				ctx.W.EmitShlRegImm8(r136, 32)
				ctx.W.EmitShrRegImm8(r136, 32)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d123)
			}
			var d124 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r137, thisptr.Reg, off)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
				ctx.BindReg(r137, &d124)
			}
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d124.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d124.Reg)
				ctx.W.EmitShlRegImm8(r138, 56)
				ctx.W.EmitShrRegImm8(r138, 56)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d125)
			}
			ctx.FreeDesc(&d124)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d123.Loc == scm.LocImm && d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() * d125.Imm.Int())}
			} else if d123.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d123.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			} else if d125.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(scratch, d123.Reg)
				if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d125.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d125.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			} else {
				r139 := ctx.AllocRegExcept(d123.Reg, d125.Reg)
				ctx.W.EmitMovRegReg(r139, d123.Reg)
				ctx.W.EmitImulInt64(r139, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d126)
			}
			if d126.Loc == scm.LocReg && d123.Loc == scm.LocReg && d126.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d123)
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			r140 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r140, uint64(dataPtr))
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140, StackOff: int32(sliceLen)}
				ctx.BindReg(r140, &d127)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
				ctx.BindReg(r140, &d127)
			}
			ctx.BindReg(r140, &d127)
			ctx.EnsureDesc(&d126)
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() / 64)}
			} else {
				r141 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r141, d126.Reg)
				ctx.W.EmitShrRegImm8(r141, 6)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d128)
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d128)
			r142 := ctx.AllocReg()
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d127)
			if d128.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r142, uint64(d128.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r142, d128.Reg)
				ctx.W.EmitShlRegImm8(r142, 3)
			}
			if d127.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.W.EmitAddInt64(r142, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r142, d127.Reg)
			}
			r143 := ctx.AllocRegExcept(r142)
			ctx.W.EmitMovRegMem(r143, r142, 0)
			ctx.FreeReg(r142)
			d129 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
			ctx.BindReg(r143, &d129)
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d126)
			var d130 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() % 64)}
			} else {
				r144 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r144, d126.Reg)
				ctx.W.EmitAndRegImm32(r144, 63)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d130)
			}
			if d130.Loc == scm.LocReg && d126.Loc == scm.LocReg && d130.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d130)
			var d131 scm.JITValueDesc
			if d129.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d129.Imm.Int()) << uint64(d130.Imm.Int())))}
			} else if d130.Loc == scm.LocImm {
				r145 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r145, d129.Reg)
				ctx.W.EmitShlRegImm8(r145, uint8(d130.Imm.Int()))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d131)
			} else {
				{
					shiftSrc := d129.Reg
					r146 := ctx.AllocRegExcept(d129.Reg)
					ctx.W.EmitMovRegReg(r146, d129.Reg)
					shiftSrc = r146
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d130.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d130.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d130.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d131)
				}
			}
			if d131.Loc == scm.LocReg && d129.Loc == scm.LocReg && d131.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d129)
			ctx.FreeDesc(&d130)
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d132)
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			if d132.Loc == scm.LocImm {
				if d132.Imm.Bool() {
					ctx.W.EmitJmp(lbl29)
				} else {
			d133 := d131
			if d133.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d133)
			ctx.EmitStoreToStack(d133, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d132.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
			d134 := d131
			if d134.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d134)
			ctx.EmitStoreToStack(d134, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.FreeDesc(&d132)
			ctx.W.MarkLabel(lbl30)
			d135 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d136 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r148, thisptr.Reg, off)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
				ctx.BindReg(r148, &d136)
			}
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d137 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d136.Imm.Int()))))}
			} else {
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r149, d136.Reg)
				ctx.W.EmitShlRegImm8(r149, 56)
				ctx.W.EmitShrRegImm8(r149, 56)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d137)
			}
			ctx.FreeDesc(&d136)
			d138 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d137)
			var d139 scm.JITValueDesc
			if d138.Loc == scm.LocImm && d137.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d138.Imm.Int() - d137.Imm.Int())}
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				r150 := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(r150, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d139)
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d138.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d137.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegReg(scratch, d138.Reg)
				if d137.Imm.Int() >= -2147483648 && d137.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d137.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else {
				r151 := ctx.AllocRegExcept(d138.Reg, d137.Reg)
				ctx.W.EmitMovRegReg(r151, d138.Reg)
				ctx.W.EmitSubInt64(r151, d137.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d139)
			}
			if d139.Loc == scm.LocReg && d138.Loc == scm.LocReg && d139.Reg == d138.Reg {
				ctx.TransferReg(d138.Reg)
				d138.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d135.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d135.Imm.Int()) >> uint64(d139.Imm.Int())))}
			} else if d139.Loc == scm.LocImm {
				r152 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r152, d135.Reg)
				ctx.W.EmitShrRegImm8(r152, uint8(d139.Imm.Int()))
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d140)
			} else {
				{
					shiftSrc := d135.Reg
					r153 := ctx.AllocRegExcept(d135.Reg)
					ctx.W.EmitMovRegReg(r153, d135.Reg)
					shiftSrc = r153
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d139.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d139.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d139.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d140)
				}
			}
			if d140.Loc == scm.LocReg && d135.Loc == scm.LocReg && d140.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			ctx.FreeDesc(&d139)
			r154 := ctx.AllocReg()
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d140)
			if d140.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r154, d140)
			}
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl29)
			d135 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d126)
			var d141 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() % 64)}
			} else {
				r155 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r155, d126.Reg)
				ctx.W.EmitAndRegImm32(r155, 63)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d141)
			}
			if d141.Loc == scm.LocReg && d126.Loc == scm.LocReg && d141.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r156, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
				ctx.BindReg(r156, &d142)
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d142)
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d142.Imm.Int()))))}
			} else {
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r157, d142.Reg)
				ctx.W.EmitShlRegImm8(r157, 56)
				ctx.W.EmitShrRegImm8(r157, 56)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d143)
			}
			ctx.FreeDesc(&d142)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() + d143.Imm.Int())}
			} else if d143.Loc == scm.LocImm && d143.Imm.Int() == 0 {
				r158 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r158, d141.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d144)
			} else if d141.Loc == scm.LocImm && d141.Imm.Int() == 0 {
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d143.Reg}
				ctx.BindReg(d143.Reg, &d144)
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else if d143.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(scratch, d141.Reg)
				if d143.Imm.Int() >= -2147483648 && d143.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d143.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d143.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d144)
			} else {
				r159 := ctx.AllocRegExcept(d141.Reg, d143.Reg)
				ctx.W.EmitMovRegReg(r159, d141.Reg)
				ctx.W.EmitAddInt64(r159, d143.Reg)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d144)
			}
			if d144.Loc == scm.LocReg && d141.Loc == scm.LocReg && d144.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			ctx.FreeDesc(&d143)
			ctx.EnsureDesc(&d144)
			var d145 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d144.Imm.Int()) > uint64(64))}
			} else {
				r160 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitCmpRegImm32(d144.Reg, 64)
				ctx.W.EmitSetcc(r160, scm.CcA)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
				ctx.BindReg(r160, &d145)
			}
			ctx.FreeDesc(&d144)
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d145.Loc == scm.LocImm {
				if d145.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
			d146 := d131
			if d146.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d146)
			ctx.EmitStoreToStack(d146, 24)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d145.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl33)
			d147 := d131
			if d147.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d147)
			ctx.EmitStoreToStack(d147, 24)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl32)
			}
			ctx.FreeDesc(&d145)
			ctx.W.MarkLabel(lbl32)
			d135 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d126)
			var d148 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r161, d126.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d148)
			}
			if d148.Loc == scm.LocReg && d126.Loc == scm.LocReg && d148.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(scratch, d148.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			}
			if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d148)
			ctx.EnsureDesc(&d149)
			r162 := ctx.AllocReg()
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d127)
			if d149.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d149.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d149.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d127.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d127.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d150 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d150)
			ctx.FreeDesc(&d149)
			ctx.EnsureDesc(&d126)
			var d151 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r164, d126.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d151)
			}
			if d151.Loc == scm.LocReg && d126.Loc == scm.LocReg && d151.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			d152 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d151)
			var d153 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d151.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() - d151.Imm.Int())}
			} else if d151.Loc == scm.LocImm && d151.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r165, d152.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d153)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d152.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d151.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d153)
			} else if d151.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(scratch, d152.Reg)
				if d151.Imm.Int() >= -2147483648 && d151.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d151.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d153)
			} else {
				r166 := ctx.AllocRegExcept(d152.Reg, d151.Reg)
				ctx.W.EmitMovRegReg(r166, d152.Reg)
				ctx.W.EmitSubInt64(r166, d151.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d153)
			}
			if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d153)
			var d154 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d150.Imm.Int()) >> uint64(d153.Imm.Int())))}
			} else if d153.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r167, d150.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d153.Imm.Int()))
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d154)
			} else {
				{
					shiftSrc := d150.Reg
					r168 := ctx.AllocRegExcept(d150.Reg)
					ctx.W.EmitMovRegReg(r168, d150.Reg)
					shiftSrc = r168
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d153.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d153.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d153.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d154)
				}
			}
			if d154.Loc == scm.LocReg && d150.Loc == scm.LocReg && d154.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d153)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d131.Imm.Int() | d154.Imm.Int())}
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d154.Reg}
				ctx.BindReg(d154.Reg, &d155)
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				r169 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r169, d131.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d155)
			} else if d131.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d154.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d155)
			} else if d154.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r170, d131.Reg)
				if d154.Imm.Int() >= -2147483648 && d154.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r170, int32(d154.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d154.Imm.Int()))
					ctx.W.EmitOrInt64(r170, scm.RegR11)
				}
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d155)
			} else {
				r171 := ctx.AllocRegExcept(d131.Reg, d154.Reg)
				ctx.W.EmitMovRegReg(r171, d131.Reg)
				ctx.W.EmitOrInt64(r171, d154.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d155)
			}
			if d155.Loc == scm.LocReg && d131.Loc == scm.LocReg && d155.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			d156 := d155
			if d156.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d156)
			ctx.EmitStoreToStack(d156, 24)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			d157 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d157)
			ctx.BindReg(r154, &d157)
			if r134 { ctx.UnprotectReg(r135) }
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d157.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d158)
			}
			ctx.FreeDesc(&d157)
			var d159 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r173 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r173, thisptr.Reg, off)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
				ctx.BindReg(r173, &d159)
			}
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d159)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d158.Loc == scm.LocImm && d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d158.Imm.Int() + d159.Imm.Int())}
			} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
				r174 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(r174, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d160)
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d159.Reg}
				ctx.BindReg(d159.Reg, &d160)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d158.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d159.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegReg(scratch, d158.Reg)
				if d159.Imm.Int() >= -2147483648 && d159.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d159.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else {
				r175 := ctx.AllocRegExcept(d158.Reg, d159.Reg)
				ctx.W.EmitMovRegReg(r175, d158.Reg)
				ctx.W.EmitAddInt64(r175, d159.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d160)
			}
			if d160.Loc == scm.LocReg && d158.Loc == scm.LocReg && d160.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.FreeDesc(&d159)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d160)
			var d162 scm.JITValueDesc
			if d121.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d121.Imm.Int() + d160.Imm.Int())}
			} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(r176, d121.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d162)
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
				ctx.BindReg(d160.Reg, &d162)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d121.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d160.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(scratch, d121.Reg)
				if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d160.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d162)
			} else {
				r177 := ctx.AllocRegExcept(d121.Reg, d160.Reg)
				ctx.W.EmitMovRegReg(r177, d121.Reg)
				ctx.W.EmitAddInt64(r177, d160.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d162)
			}
			if d162.Loc == scm.LocReg && d121.Loc == scm.LocReg && d162.Reg == d121.Reg {
				ctx.TransferReg(d121.Reg)
				d121.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d160)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d162)
			var d164 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r178 := ctx.AllocReg()
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r178, fieldAddr)
				ctx.W.EmitMovRegMem64(r179, fieldAddr+8)
				d164 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r178, Reg2: r179}
				ctx.BindReg(r178, &d164)
				ctx.BindReg(r179, &d164)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r180, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r181, thisptr.Reg, off+8)
				d164 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d164)
				ctx.BindReg(r181, &d164)
			}
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d162)
			r182 := ctx.AllocReg()
			r183 := ctx.AllocRegExcept(r182)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d162)
			if d164.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d164.Imm.Int()))
			} else if d164.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r182, d164.Reg)
			} else {
				ctx.W.EmitMovRegReg(r182, d164.Reg)
			}
			if d121.Loc == scm.LocImm {
				if d121.Imm.Int() != 0 {
					if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r182, int32(d121.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
						ctx.W.EmitAddInt64(r182, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r182, d121.Reg)
			}
			if d162.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r183, uint64(d162.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r183, d162.Reg)
			}
			if d121.Loc == scm.LocImm {
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r183, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitSubInt64(r183, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r183, d121.Reg)
			}
			d165 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
			ctx.BindReg(r182, &d165)
			ctx.BindReg(r183, &d165)
			ctx.FreeDesc(&d121)
			ctx.FreeDesc(&d162)
			d166 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d166)
			ctx.BindReg(r1, &d166)
			d167 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d165}, 2)
			ctx.EmitMovPairToResult(&d167, &d166)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl10)
			var d168 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r184 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r184, thisptr.Reg, off)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r184}
				ctx.BindReg(r184, &d168)
			}
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d168)
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d168.Imm.Int()))))}
			} else {
				r185 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r185, d168.Reg)
				ctx.W.EmitShlRegImm8(r185, 32)
				ctx.W.EmitShrRegImm8(r185, 32)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d169)
			}
			ctx.FreeDesc(&d168)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d40.Imm.Int()) == uint64(d169.Imm.Int()))}
			} else if d169.Loc == scm.LocImm {
				r186 := ctx.AllocRegExcept(d40.Reg)
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d169.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r186, scm.CcE)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r186}
				ctx.BindReg(r186, &d170)
			} else if d40.Loc == scm.LocImm {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d169.Reg)
				ctx.W.EmitSetcc(r187, scm.CcE)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r187}
				ctx.BindReg(r187, &d170)
			} else {
				r188 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitCmpInt64(d40.Reg, d169.Reg)
				ctx.W.EmitSetcc(r188, scm.CcE)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r188}
				ctx.BindReg(r188, &d170)
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d169)
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d170.Loc == scm.LocImm {
				if d170.Imm.Bool() {
					ctx.W.EmitJmp(lbl34)
				} else {
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d170.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl35)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl35)
				ctx.W.EmitJmp(lbl34)
			}
			ctx.FreeDesc(&d170)
			ctx.W.MarkLabel(lbl20)
			ctx.EnsureDesc(&idxInt)
			d171 := idxInt
			_ = d171
			r189 := idxInt.Loc == scm.LocReg
			r190 := idxInt.Reg
			if r189 { ctx.ProtectReg(r190) }
			lbl36 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d171)
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d171.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, d171.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d172)
			}
			var d173 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r192, thisptr.Reg, off)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r192}
				ctx.BindReg(r192, &d173)
			}
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d173.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, d173.Reg)
				ctx.W.EmitShlRegImm8(r193, 56)
				ctx.W.EmitShrRegImm8(r193, 56)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d174)
			}
			ctx.FreeDesc(&d173)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d174)
			var d175 scm.JITValueDesc
			if d172.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d172.Imm.Int() * d174.Imm.Int())}
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d172.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d175)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegReg(scratch, d172.Reg)
				if d174.Imm.Int() >= -2147483648 && d174.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d174.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d174.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d175)
			} else {
				r194 := ctx.AllocRegExcept(d172.Reg, d174.Reg)
				ctx.W.EmitMovRegReg(r194, d172.Reg)
				ctx.W.EmitImulInt64(r194, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d175)
			}
			if d175.Loc == scm.LocReg && d172.Loc == scm.LocReg && d175.Reg == d172.Reg {
				ctx.TransferReg(d172.Reg)
				d172.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			ctx.FreeDesc(&d174)
			var d176 scm.JITValueDesc
			r195 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r195, uint64(dataPtr))
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195, StackOff: int32(sliceLen)}
				ctx.BindReg(r195, &d176)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r195, thisptr.Reg, off)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195}
				ctx.BindReg(r195, &d176)
			}
			ctx.BindReg(r195, &d176)
			ctx.EnsureDesc(&d175)
			var d177 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() / 64)}
			} else {
				r196 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r196, d175.Reg)
				ctx.W.EmitShrRegImm8(r196, 6)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d177)
			}
			if d177.Loc == scm.LocReg && d175.Loc == scm.LocReg && d177.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d177)
			r197 := ctx.AllocReg()
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d176)
			if d177.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r197, uint64(d177.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r197, d177.Reg)
				ctx.W.EmitShlRegImm8(r197, 3)
			}
			if d176.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(r197, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r197, d176.Reg)
			}
			r198 := ctx.AllocRegExcept(r197)
			ctx.W.EmitMovRegMem(r198, r197, 0)
			ctx.FreeReg(r197)
			d178 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
			ctx.BindReg(r198, &d178)
			ctx.FreeDesc(&d177)
			ctx.EnsureDesc(&d175)
			var d179 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() % 64)}
			} else {
				r199 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r199, d175.Reg)
				ctx.W.EmitAndRegImm32(r199, 63)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d179)
			}
			if d179.Loc == scm.LocReg && d175.Loc == scm.LocReg && d179.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d179)
			var d180 scm.JITValueDesc
			if d178.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d178.Imm.Int()) << uint64(d179.Imm.Int())))}
			} else if d179.Loc == scm.LocImm {
				r200 := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegReg(r200, d178.Reg)
				ctx.W.EmitShlRegImm8(r200, uint8(d179.Imm.Int()))
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d180)
			} else {
				{
					shiftSrc := d178.Reg
					r201 := ctx.AllocRegExcept(d178.Reg)
					ctx.W.EmitMovRegReg(r201, d178.Reg)
					shiftSrc = r201
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d179.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d179.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d179.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d180)
				}
			}
			if d180.Loc == scm.LocReg && d178.Loc == scm.LocReg && d180.Reg == d178.Reg {
				ctx.TransferReg(d178.Reg)
				d178.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d178)
			ctx.FreeDesc(&d179)
			var d181 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d181)
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d181.Loc == scm.LocImm {
				if d181.Imm.Bool() {
					ctx.W.EmitJmp(lbl37)
				} else {
			d182 := d180
			if d182.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d182)
			ctx.EmitStoreToStack(d182, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d181.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
			d183 := d180
			if d183.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d183)
			ctx.EmitStoreToStack(d183, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d181)
			ctx.W.MarkLabel(lbl38)
			d184 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d185 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r203, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r203}
				ctx.BindReg(r203, &d185)
			}
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d185)
			var d186 scm.JITValueDesc
			if d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d185.Imm.Int()))))}
			} else {
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r204, d185.Reg)
				ctx.W.EmitShlRegImm8(r204, 56)
				ctx.W.EmitShrRegImm8(r204, 56)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d186)
			}
			ctx.FreeDesc(&d185)
			d187 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d186)
			var d188 scm.JITValueDesc
			if d187.Loc == scm.LocImm && d186.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d187.Imm.Int() - d186.Imm.Int())}
			} else if d186.Loc == scm.LocImm && d186.Imm.Int() == 0 {
				r205 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r205, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d188)
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d187.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d186.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else if d186.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(scratch, d187.Reg)
				if d186.Imm.Int() >= -2147483648 && d186.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d186.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d186.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else {
				r206 := ctx.AllocRegExcept(d187.Reg, d186.Reg)
				ctx.W.EmitMovRegReg(r206, d187.Reg)
				ctx.W.EmitSubInt64(r206, d186.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d188)
			}
			if d188.Loc == scm.LocReg && d187.Loc == scm.LocReg && d188.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d188)
			var d189 scm.JITValueDesc
			if d184.Loc == scm.LocImm && d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d184.Imm.Int()) >> uint64(d188.Imm.Int())))}
			} else if d188.Loc == scm.LocImm {
				r207 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r207, d184.Reg)
				ctx.W.EmitShrRegImm8(r207, uint8(d188.Imm.Int()))
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d189)
			} else {
				{
					shiftSrc := d184.Reg
					r208 := ctx.AllocRegExcept(d184.Reg)
					ctx.W.EmitMovRegReg(r208, d184.Reg)
					shiftSrc = r208
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d188.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d188.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d188.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d189)
				}
			}
			if d189.Loc == scm.LocReg && d184.Loc == scm.LocReg && d189.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d184)
			ctx.FreeDesc(&d188)
			r209 := ctx.AllocReg()
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d189)
			if d189.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r209, d189)
			}
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl37)
			d184 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d175)
			var d190 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() % 64)}
			} else {
				r210 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r210, d175.Reg)
				ctx.W.EmitAndRegImm32(r210, 63)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d190)
			}
			if d190.Loc == scm.LocReg && d175.Loc == scm.LocReg && d190.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			var d191 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r211, thisptr.Reg, off)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
				ctx.BindReg(r211, &d191)
			}
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d191)
			var d192 scm.JITValueDesc
			if d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d191.Imm.Int()))))}
			} else {
				r212 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r212, d191.Reg)
				ctx.W.EmitShlRegImm8(r212, 56)
				ctx.W.EmitShrRegImm8(r212, 56)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d192)
			}
			ctx.FreeDesc(&d191)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d192)
			var d193 scm.JITValueDesc
			if d190.Loc == scm.LocImm && d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d190.Imm.Int() + d192.Imm.Int())}
			} else if d192.Loc == scm.LocImm && d192.Imm.Int() == 0 {
				r213 := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegReg(r213, d190.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d193)
			} else if d190.Loc == scm.LocImm && d190.Imm.Int() == 0 {
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d192.Reg}
				ctx.BindReg(d192.Reg, &d193)
			} else if d190.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d190.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d192.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d193)
			} else if d192.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d190.Reg)
				ctx.W.EmitMovRegReg(scratch, d190.Reg)
				if d192.Imm.Int() >= -2147483648 && d192.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d192.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d192.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d193)
			} else {
				r214 := ctx.AllocRegExcept(d190.Reg, d192.Reg)
				ctx.W.EmitMovRegReg(r214, d190.Reg)
				ctx.W.EmitAddInt64(r214, d192.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d193)
			}
			if d193.Loc == scm.LocReg && d190.Loc == scm.LocReg && d193.Reg == d190.Reg {
				ctx.TransferReg(d190.Reg)
				d190.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d190)
			ctx.FreeDesc(&d192)
			ctx.EnsureDesc(&d193)
			var d194 scm.JITValueDesc
			if d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d193.Imm.Int()) > uint64(64))}
			} else {
				r215 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitCmpRegImm32(d193.Reg, 64)
				ctx.W.EmitSetcc(r215, scm.CcA)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r215}
				ctx.BindReg(r215, &d194)
			}
			ctx.FreeDesc(&d193)
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			if d194.Loc == scm.LocImm {
				if d194.Imm.Bool() {
					ctx.W.EmitJmp(lbl40)
				} else {
			d195 := d180
			if d195.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d195)
			ctx.EmitStoreToStack(d195, 32)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d194.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
			d196 := d180
			if d196.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d196)
			ctx.EmitStoreToStack(d196, 32)
				ctx.W.EmitJmp(lbl38)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
			}
			ctx.FreeDesc(&d194)
			ctx.W.MarkLabel(lbl40)
			d184 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d175)
			var d197 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() / 64)}
			} else {
				r216 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r216, d175.Reg)
				ctx.W.EmitShrRegImm8(r216, 6)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d197)
			}
			if d197.Loc == scm.LocReg && d175.Loc == scm.LocReg && d197.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d197)
			var d198 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(scratch, d197.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			}
			if d198.Loc == scm.LocReg && d197.Loc == scm.LocReg && d198.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			ctx.EnsureDesc(&d198)
			r217 := ctx.AllocReg()
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d176)
			if d198.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r217, uint64(d198.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r217, d198.Reg)
				ctx.W.EmitShlRegImm8(r217, 3)
			}
			if d176.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(r217, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r217, d176.Reg)
			}
			r218 := ctx.AllocRegExcept(r217)
			ctx.W.EmitMovRegMem(r218, r217, 0)
			ctx.FreeReg(r217)
			d199 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
			ctx.BindReg(r218, &d199)
			ctx.FreeDesc(&d198)
			ctx.EnsureDesc(&d175)
			var d200 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() % 64)}
			} else {
				r219 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r219, d175.Reg)
				ctx.W.EmitAndRegImm32(r219, 63)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d200)
			}
			if d200.Loc == scm.LocReg && d175.Loc == scm.LocReg && d200.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			d201 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d200)
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm && d200.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() - d200.Imm.Int())}
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r220, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d202)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d201.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d200.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(scratch, d201.Reg)
				if d200.Imm.Int() >= -2147483648 && d200.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d200.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d200.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else {
				r221 := ctx.AllocRegExcept(d201.Reg, d200.Reg)
				ctx.W.EmitMovRegReg(r221, d201.Reg)
				ctx.W.EmitSubInt64(r221, d200.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d202)
			}
			if d202.Loc == scm.LocReg && d201.Loc == scm.LocReg && d202.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d199.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d199.Imm.Int()) >> uint64(d202.Imm.Int())))}
			} else if d202.Loc == scm.LocImm {
				r222 := ctx.AllocRegExcept(d199.Reg)
				ctx.W.EmitMovRegReg(r222, d199.Reg)
				ctx.W.EmitShrRegImm8(r222, uint8(d202.Imm.Int()))
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d203)
			} else {
				{
					shiftSrc := d199.Reg
					r223 := ctx.AllocRegExcept(d199.Reg)
					ctx.W.EmitMovRegReg(r223, d199.Reg)
					shiftSrc = r223
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d202.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d202.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d202.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d203)
				}
			}
			if d203.Loc == scm.LocReg && d199.Loc == scm.LocReg && d203.Reg == d199.Reg {
				ctx.TransferReg(d199.Reg)
				d199.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d199)
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d180.Loc == scm.LocImm && d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() | d203.Imm.Int())}
			} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d203.Reg}
				ctx.BindReg(d203.Reg, &d204)
			} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r224, d180.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d204)
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			} else if d203.Loc == scm.LocImm {
				r225 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r225, d180.Reg)
				if d203.Imm.Int() >= -2147483648 && d203.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r225, int32(d203.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
					ctx.W.EmitOrInt64(r225, scm.RegR11)
				}
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d204)
			} else {
				r226 := ctx.AllocRegExcept(d180.Reg, d203.Reg)
				ctx.W.EmitMovRegReg(r226, d180.Reg)
				ctx.W.EmitOrInt64(r226, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d204)
			}
			if d204.Loc == scm.LocReg && d180.Loc == scm.LocReg && d204.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d203)
			d205 := d204
			if d205.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d205)
			ctx.EmitStoreToStack(d205, 32)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			d206 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r209}
			ctx.BindReg(r209, &d206)
			ctx.BindReg(r209, &d206)
			if r189 { ctx.UnprotectReg(r190) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d206.Imm.Int()))))}
			} else {
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r227, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d207)
			}
			ctx.FreeDesc(&d206)
			var d208 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r228 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r228, thisptr.Reg, off)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r228}
				ctx.BindReg(r228, &d208)
			}
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() + d208.Imm.Int())}
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r229 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r229, d207.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d209)
			} else if d207.Loc == scm.LocImm && d207.Imm.Int() == 0 {
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
				ctx.BindReg(d208.Reg, &d209)
			} else if d207.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(scratch, d207.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else {
				r230 := ctx.AllocRegExcept(d207.Reg, d208.Reg)
				ctx.W.EmitMovRegReg(r230, d207.Reg)
				ctx.W.EmitAddInt64(r230, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d209)
			}
			if d209.Loc == scm.LocReg && d207.Loc == scm.LocReg && d209.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d207)
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d209)
			var d210 scm.JITValueDesc
			if d209.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d209.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d209.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d210)
			}
			ctx.FreeDesc(&d209)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d81)
			var d211 scm.JITValueDesc
			if d81.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d81.Imm.Int()))))}
			} else {
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r232, d81.Reg)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d211)
			}
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d210)
			var d212 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d210.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d81.Imm.Int() + d210.Imm.Int())}
			} else if d210.Loc == scm.LocImm && d210.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r233, d81.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d212)
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d210.Reg}
				ctx.BindReg(d210.Reg, &d212)
			} else if d81.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d81.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d210.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else if d210.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(scratch, d81.Reg)
				if d210.Imm.Int() >= -2147483648 && d210.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d210.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d210.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else {
				r234 := ctx.AllocRegExcept(d81.Reg, d210.Reg)
				ctx.W.EmitMovRegReg(r234, d81.Reg)
				ctx.W.EmitAddInt64(r234, d210.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d212)
			}
			if d212.Loc == scm.LocReg && d81.Loc == scm.LocReg && d212.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d212)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d212.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d213)
			}
			ctx.FreeDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d213)
			r236 := ctx.AllocReg()
			r237 := ctx.AllocRegExcept(r236)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d213)
			if d164.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d164.Imm.Int()))
			} else if d164.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r236, d164.Reg)
			} else {
				ctx.W.EmitMovRegReg(r236, d164.Reg)
			}
			if d211.Loc == scm.LocImm {
				if d211.Imm.Int() != 0 {
					if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r236, int32(d211.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
						ctx.W.EmitAddInt64(r236, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r236, d211.Reg)
			}
			if d213.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r237, uint64(d213.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r237, d213.Reg)
			}
			if d211.Loc == scm.LocImm {
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r237, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitSubInt64(r237, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r237, d211.Reg)
			}
			d214 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r236, Reg2: r237}
			ctx.BindReg(r236, &d214)
			ctx.BindReg(r237, &d214)
			ctx.FreeDesc(&d211)
			ctx.FreeDesc(&d213)
			d215 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d215)
			ctx.BindReg(r1, &d215)
			d216 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d214}, 2)
			ctx.EmitMovPairToResult(&d216, &d215)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl19)
			var d217 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 64)
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r238, thisptr.Reg, off)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r238}
				ctx.BindReg(r238, &d217)
			}
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d81.Imm.Int()) == uint64(d217.Imm.Int()))}
			} else if d217.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d81.Reg)
				if d217.Imm.Int() >= -2147483648 && d217.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d81.Reg, int32(d217.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d217.Imm.Int()))
					ctx.W.EmitCmpInt64(d81.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d218)
			} else if d81.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d217.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d218)
			} else {
				r241 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitCmpInt64(d81.Reg, d217.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d218)
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d217)
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d218.Loc == scm.LocImm {
				if d218.Imm.Bool() {
					ctx.W.EmitJmp(lbl42)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d218.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl43)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl43)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.FreeDesc(&d218)
			ctx.W.MarkLabel(lbl34)
			d219 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d219)
			ctx.BindReg(r1, &d219)
			ctx.W.EmitMakeNil(d219)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl42)
			d220 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d220)
			ctx.BindReg(r1, &d220)
			ctx.W.EmitMakeNil(d220)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d221 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d221)
			ctx.BindReg(r1, &d221)
			ctx.EmitMovPairToResult(&d221, &result)
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
