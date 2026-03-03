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
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
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
				ctx.BindReg(r8, &d5)
			}
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
			r16 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r16, 0)
			ctx.ProtectReg(r16)
			d11 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r16}
			ctx.BindReg(r16, &d11)
			ctx.UnprotectReg(r16)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 24)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d12)
			}
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
			} else {
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r18, d12.Reg)
				ctx.W.EmitShlRegImm8(r18, 56)
				ctx.W.EmitShrRegImm8(r18, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d13)
			}
			ctx.FreeDesc(&d12)
			d14 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d13.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				r19 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r19, d14.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d15)
			} else if d14.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d13.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d15)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(scratch, d14.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d15)
			} else {
				r20 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r20, d14.Reg)
				ctx.W.EmitSubInt64(r20, d13.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d15)
			}
			if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d16 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) >> uint64(d15.Imm.Int())))}
			} else if d15.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r21, d11.Reg)
				ctx.W.EmitShrRegImm8(r21, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d16)
			} else {
				{
					shiftSrc := d11.Reg
					r22 := ctx.AllocRegExcept(d11.Reg)
					ctx.W.EmitMovRegReg(r22, d11.Reg)
					shiftSrc = r22
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
					ctx.BindReg(shiftSrc, &d16)
				}
			}
			if d16.Loc == scm.LocReg && d11.Loc == scm.LocReg && d16.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d15)
			r23 := ctx.AllocReg()
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			ctx.EmitMovToReg(r23, d16)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl5)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d17 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r24 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r24, d4.Reg)
				ctx.W.EmitAndRegImm32(r24, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d17)
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
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d18)
			}
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r26, d18.Reg)
				ctx.W.EmitShlRegImm8(r26, 56)
				ctx.W.EmitShrRegImm8(r26, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d19)
			}
			ctx.FreeDesc(&d18)
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r27 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r27, d17.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d20)
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d19.Reg}
				ctx.BindReg(d19.Reg, &d20)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d20)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(scratch, d17.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d20)
			} else {
				r28 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r28, d17.Reg)
				ctx.W.EmitAddInt64(r28, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d20)
			}
			if d20.Loc == scm.LocReg && d17.Loc == scm.LocReg && d20.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d19)
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d20.Imm.Int()) > uint64(64))}
			} else {
				r29 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitCmpRegImm32(d20.Reg, 64)
				ctx.W.EmitSetcc(r29, scm.CcA)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
				ctx.BindReg(r29, &d21)
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
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r30, d4.Reg)
				ctx.W.EmitShrRegImm8(r30, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d22)
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(scratch, d22.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			r31 := ctx.AllocReg()
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r31, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r31, d23.Reg)
				ctx.W.EmitShlRegImm8(r31, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r31, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r31, d5.Reg)
			}
			r32 := ctx.AllocRegExcept(r31)
			ctx.W.EmitMovRegMem(r32, r31, 0)
			ctx.FreeReg(r31)
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d24)
			ctx.FreeDesc(&d23)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d25 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r33, d4.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d25)
			}
			if d25.Loc == scm.LocReg && d4.Loc == scm.LocReg && d25.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d26 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r34, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d27)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else {
				r35 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r35, d26.Reg)
				ctx.W.EmitSubInt64(r35, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			var d28 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				r36 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r36, d24.Reg)
				ctx.W.EmitShrRegImm8(r36, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d28)
			} else {
				{
					shiftSrc := d24.Reg
					r37 := ctx.AllocRegExcept(d24.Reg)
					ctx.W.EmitMovRegReg(r37, d24.Reg)
					shiftSrc = r37
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
					ctx.BindReg(shiftSrc, &d28)
				}
			}
			if d28.Loc == scm.LocReg && d24.Loc == scm.LocReg && d28.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d27)
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			var d29 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d28.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
				ctx.BindReg(d28.Reg, &d29)
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				r38 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r38, d9.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d29)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else if d28.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r39, d9.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r39, int32(d28.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
					ctx.W.EmitOrInt64(r39, scm.RegR11)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d29)
			} else {
				r40 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				ctx.W.EmitOrInt64(r40, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d29)
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
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
			ctx.BindReg(r23, &d30)
			ctx.BindReg(r23, &d30)
			if r1 { ctx.UnprotectReg(r2) }
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d30.Imm.Int()))))}
			} else {
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r41, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d31)
			}
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 32)
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r42, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
				ctx.BindReg(r42, &d32)
			}
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d33 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r43, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d33)
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d33)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(scratch, d31.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r44 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r44, d31.Reg)
				ctx.W.EmitAddInt64(r44, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d33)
			}
			if d33.Loc == scm.LocReg && d31.Loc == scm.LocReg && d33.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.FreeDesc(&d32)
			if d33.Loc == scm.LocStack || d33.Loc == scm.LocStackPair { ctx.EnsureDesc(&d33) }
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d33.Imm.Int()))))}
			} else {
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r45, d33.Reg)
				ctx.W.EmitShlRegImm8(r45, 32)
				ctx.W.EmitShrRegImm8(r45, 32)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d34)
			}
			ctx.FreeDesc(&d33)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 56)
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r46, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r46}
				ctx.BindReg(r46, &d35)
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r47 := idxInt.Loc == scm.LocReg
			r48 := idxInt.Reg
			if r47 { ctx.ProtectReg(r48) }
			lbl13 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d36 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r49, 32)
				ctx.W.EmitShrRegImm8(r49, 32)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d36)
			}
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r50, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d37)
			}
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d37.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d37.Reg)
				ctx.W.EmitShlRegImm8(r51, 56)
				ctx.W.EmitShrRegImm8(r51, 56)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d38)
			}
			ctx.FreeDesc(&d37)
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() * d38.Imm.Int())}
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r52 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r52, d36.Reg)
				ctx.W.EmitImulInt64(r52, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d39)
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
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r53, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d40)
			}
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r54 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r54, d39.Reg)
				ctx.W.EmitShrRegImm8(r54, 6)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d41)
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			r55 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r55, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r55, d41.Reg)
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
			d42 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
			ctx.BindReg(r56, &d42)
			ctx.FreeDesc(&d41)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r57 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r57, d39.Reg)
				ctx.W.EmitAndRegImm32(r57, 63)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d43)
			}
			if d43.Loc == scm.LocReg && d39.Loc == scm.LocReg && d43.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) << uint64(d43.Imm.Int())))}
			} else if d43.Loc == scm.LocImm {
				r58 := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegReg(r58, d42.Reg)
				ctx.W.EmitShlRegImm8(r58, uint8(d43.Imm.Int()))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d44)
			} else {
				{
					shiftSrc := d42.Reg
					r59 := ctx.AllocRegExcept(d42.Reg)
					ctx.W.EmitMovRegReg(r59, d42.Reg)
					shiftSrc = r59
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
					ctx.BindReg(shiftSrc, &d44)
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
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r60, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r60}
				ctx.BindReg(r60, &d45)
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
			r61 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r61, 8)
			ctx.ProtectReg(r61)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r61}
			ctx.BindReg(r61, &d46)
			ctx.UnprotectReg(r61)
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d47)
			}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d47.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d47.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d48)
			}
			ctx.FreeDesc(&d47)
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r64, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r65 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r65, d49.Reg)
				ctx.W.EmitSubInt64(r65, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d50)
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d50.Loc == scm.LocStack || d50.Loc == scm.LocStackPair { ctx.EnsureDesc(&d50) }
			var d51 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d50.Imm.Int())))}
			} else if d50.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r66, d46.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d51)
			} else {
				{
					shiftSrc := d46.Reg
					r67 := ctx.AllocRegExcept(d46.Reg)
					ctx.W.EmitMovRegReg(r67, d46.Reg)
					shiftSrc = r67
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
					ctx.BindReg(shiftSrc, &d51)
				}
			}
			if d51.Loc == scm.LocReg && d46.Loc == scm.LocReg && d51.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d50)
			r68 := ctx.AllocReg()
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitMovToReg(r68, d51)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl14)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d52 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r69, d39.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d52)
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
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d53)
			}
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			var d54 scm.JITValueDesc
			if d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d53.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d53.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d54)
			}
			ctx.FreeDesc(&d53)
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d54.Imm.Int())}
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r72, d52.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d55)
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d54.Reg}
				ctx.BindReg(d54.Reg, &d55)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else if d54.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(scratch, d52.Reg)
				if d54.Imm.Int() >= -2147483648 && d54.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d54.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d54.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else {
				r73 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r73, d52.Reg)
				ctx.W.EmitAddInt64(r73, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d55)
			}
			if d55.Loc == scm.LocReg && d52.Loc == scm.LocReg && d55.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.FreeDesc(&d54)
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d55.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitCmpRegImm32(d55.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d56)
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
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d57 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r75, d39.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d57)
			}
			if d57.Loc == scm.LocReg && d39.Loc == scm.LocReg && d57.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			if d57.Loc == scm.LocStack || d57.Loc == scm.LocStackPair { ctx.EnsureDesc(&d57) }
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.W.EmitMovRegReg(scratch, d57.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d58)
			}
			if d58.Loc == scm.LocReg && d57.Loc == scm.LocReg && d58.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			r76 := ctx.AllocReg()
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d58.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d58.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d40.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d59 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d59)
			ctx.FreeDesc(&d58)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d60 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r78, d39.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d60)
			}
			if d60.Loc == scm.LocReg && d39.Loc == scm.LocReg && d60.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			d61 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() - d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r79, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d62)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d61.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else if d60.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(scratch, d61.Reg)
				if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d60.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else {
				r80 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r80, d61.Reg)
				ctx.W.EmitSubInt64(r80, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d62)
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d63 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) >> uint64(d62.Imm.Int())))}
			} else if d62.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r81, d59.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d63)
			} else {
				{
					shiftSrc := d59.Reg
					r82 := ctx.AllocRegExcept(d59.Reg)
					ctx.W.EmitMovRegReg(r82, d59.Reg)
					shiftSrc = r82
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
					ctx.BindReg(shiftSrc, &d63)
				}
			}
			if d63.Loc == scm.LocReg && d59.Loc == scm.LocReg && d63.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d62)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d63.Loc == scm.LocStack || d63.Loc == scm.LocStackPair { ctx.EnsureDesc(&d63) }
			var d64 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() | d63.Imm.Int())}
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d63.Reg}
				ctx.BindReg(d63.Reg, &d64)
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r83, d44.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d64)
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else if d63.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r84, d44.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d63.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d64)
			} else {
				r85 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r85, d44.Reg)
				ctx.W.EmitOrInt64(r85, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d64)
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
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d65)
			ctx.BindReg(r68, &d65)
			if r47 { ctx.UnprotectReg(r48) }
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d65.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d66)
			}
			ctx.FreeDesc(&d65)
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d67)
			}
			if d66.Loc == scm.LocStack || d66.Loc == scm.LocStackPair { ctx.EnsureDesc(&d66) }
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			var d68 scm.JITValueDesc
			if d66.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + d67.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r88, d66.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d68)
			} else if d66.Loc == scm.LocImm && d66.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
				ctx.BindReg(d67.Reg, &d68)
			} else if d66.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d66.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d68)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(scratch, d66.Reg)
				if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d67.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d68)
			} else {
				r89 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r89, d66.Reg)
				ctx.W.EmitAddInt64(r89, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d68)
			}
			if d68.Loc == scm.LocReg && d66.Loc == scm.LocReg && d68.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d66)
			ctx.FreeDesc(&d67)
			if d68.Loc == scm.LocStack || d68.Loc == scm.LocStackPair { ctx.EnsureDesc(&d68) }
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d68.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d68.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d69)
			}
			ctx.FreeDesc(&d68)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d70)
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
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			r92 := d34.Loc == scm.LocReg
			r93 := d34.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl22 := ctx.W.ReserveLabel()
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d71 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d34.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d71)
			}
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d72)
			}
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d72.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d73)
			}
			ctx.FreeDesc(&d72)
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			if d73.Loc == scm.LocStack || d73.Loc == scm.LocStackPair { ctx.EnsureDesc(&d73) }
			var d74 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d73.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d71.Imm.Int() * d73.Imm.Int())}
			} else if d71.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d71.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d74)
			} else if d73.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegReg(scratch, d71.Reg)
				if d73.Imm.Int() >= -2147483648 && d73.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d73.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d73.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d74)
			} else {
				r97 := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegReg(r97, d71.Reg)
				ctx.W.EmitImulInt64(r97, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d74)
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
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d75)
			}
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d76 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r99, d74.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d76)
			}
			if d76.Loc == scm.LocReg && d74.Loc == scm.LocReg && d76.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			r100 := ctx.AllocReg()
			if d76.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d76.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d75.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d77)
			ctx.FreeDesc(&d76)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d78 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r102, d74.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d78)
			}
			if d78.Loc == scm.LocReg && d74.Loc == scm.LocReg && d78.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) << uint64(d78.Imm.Int())))}
			} else if d78.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(r103, d77.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d78.Imm.Int()))
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d79)
			} else {
				{
					shiftSrc := d77.Reg
					r104 := ctx.AllocRegExcept(d77.Reg)
					ctx.W.EmitMovRegReg(r104, d77.Reg)
					shiftSrc = r104
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
					ctx.BindReg(shiftSrc, &d79)
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
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d80)
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
			r106 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r106, 16)
			ctx.ProtectReg(r106)
			d81 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r106}
			ctx.BindReg(r106, &d81)
			ctx.UnprotectReg(r106)
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d82)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d82.Imm.Int()))))}
			} else {
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r108, d82.Reg)
				ctx.W.EmitShlRegImm8(r108, 56)
				ctx.W.EmitShrRegImm8(r108, 56)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d83)
			}
			ctx.FreeDesc(&d82)
			d84 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d83.Loc == scm.LocStack || d83.Loc == scm.LocStackPair { ctx.EnsureDesc(&d83) }
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() - d83.Imm.Int())}
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				r109 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r109, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d85)
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d85)
			} else {
				r110 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r110, d84.Reg)
				ctx.W.EmitSubInt64(r110, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d85)
			}
			if d85.Loc == scm.LocReg && d84.Loc == scm.LocReg && d85.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d83)
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			var d86 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d85.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d81.Imm.Int()) >> uint64(d85.Imm.Int())))}
			} else if d85.Loc == scm.LocImm {
				r111 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r111, d81.Reg)
				ctx.W.EmitShrRegImm8(r111, uint8(d85.Imm.Int()))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d86)
			} else {
				{
					shiftSrc := d81.Reg
					r112 := ctx.AllocRegExcept(d81.Reg)
					ctx.W.EmitMovRegReg(r112, d81.Reg)
					shiftSrc = r112
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
					ctx.BindReg(shiftSrc, &d86)
				}
			}
			if d86.Loc == scm.LocReg && d81.Loc == scm.LocReg && d86.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d85)
			r113 := ctx.AllocReg()
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			ctx.EmitMovToReg(r113, d86)
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl23)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d87 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r114 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r114, d74.Reg)
				ctx.W.EmitAndRegImm32(r114, 63)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
				ctx.BindReg(r114, &d87)
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
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r115, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
				ctx.BindReg(r115, &d88)
			}
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d88.Imm.Int()))))}
			} else {
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r116, d88.Reg)
				ctx.W.EmitShlRegImm8(r116, 56)
				ctx.W.EmitShrRegImm8(r116, 56)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d89)
			}
			ctx.FreeDesc(&d88)
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			var d90 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d89.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				r117 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r117, d87.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d90)
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d89.Reg}
				ctx.BindReg(d89.Reg, &d90)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(scratch, d87.Reg)
				if d89.Imm.Int() >= -2147483648 && d89.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d89.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d90)
			} else {
				r118 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r118, d87.Reg)
				ctx.W.EmitAddInt64(r118, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d90)
			}
			if d90.Loc == scm.LocReg && d87.Loc == scm.LocReg && d90.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d89)
			if d90.Loc == scm.LocStack || d90.Loc == scm.LocStackPair { ctx.EnsureDesc(&d90) }
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d90.Imm.Int()) > uint64(64))}
			} else {
				r119 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitCmpRegImm32(d90.Reg, 64)
				ctx.W.EmitSetcc(r119, scm.CcA)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r119}
				ctx.BindReg(r119, &d91)
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
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d92 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r120 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r120, d74.Reg)
				ctx.W.EmitShrRegImm8(r120, 6)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d92)
			}
			if d92.Loc == scm.LocReg && d74.Loc == scm.LocReg && d92.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair { ctx.EnsureDesc(&d92) }
			var d93 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(scratch, d92.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d93)
			}
			if d93.Loc == scm.LocReg && d92.Loc == scm.LocReg && d93.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d92)
			if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair { ctx.EnsureDesc(&d93) }
			r121 := ctx.AllocReg()
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r121, uint64(d93.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r121, d93.Reg)
				ctx.W.EmitShlRegImm8(r121, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r121, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r121, d75.Reg)
			}
			r122 := ctx.AllocRegExcept(r121)
			ctx.W.EmitMovRegMem(r122, r121, 0)
			ctx.FreeReg(r121)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r122}
			ctx.BindReg(r122, &d94)
			ctx.FreeDesc(&d93)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d95 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r123 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r123, d74.Reg)
				ctx.W.EmitAndRegImm32(r123, 63)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d95)
			}
			if d95.Loc == scm.LocReg && d74.Loc == scm.LocReg && d95.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			d96 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			var d97 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d95.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() - d95.Imm.Int())}
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				r124 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r124, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d97)
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
				r125 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r125, d96.Reg)
				ctx.W.EmitSubInt64(r125, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d97)
			}
			if d97.Loc == scm.LocReg && d96.Loc == scm.LocReg && d97.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			var d98 scm.JITValueDesc
			if d94.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) >> uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r126, d94.Reg)
				ctx.W.EmitShrRegImm8(r126, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d98)
			} else {
				{
					shiftSrc := d94.Reg
					r127 := ctx.AllocRegExcept(d94.Reg)
					ctx.W.EmitMovRegReg(r127, d94.Reg)
					shiftSrc = r127
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
			if d98.Loc == scm.LocReg && d94.Loc == scm.LocReg && d98.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			ctx.FreeDesc(&d97)
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			var d99 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d98.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() | d98.Imm.Int())}
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d98.Reg}
				ctx.BindReg(d98.Reg, &d99)
			} else if d98.Loc == scm.LocImm && d98.Imm.Int() == 0 {
				r128 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r128, d79.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d99)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d99)
			} else if d98.Loc == scm.LocImm {
				r129 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r129, d79.Reg)
				if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r129, int32(d98.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitOrInt64(r129, scm.RegR11)
				}
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d99)
			} else {
				r130 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r130, d79.Reg)
				ctx.W.EmitOrInt64(r130, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d99)
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
			d100 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r113}
			ctx.BindReg(r113, &d100)
			ctx.BindReg(r113, &d100)
			if r92 { ctx.UnprotectReg(r93) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d100.Imm.Int()))))}
			} else {
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r131, d100.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d101)
			}
			ctx.FreeDesc(&d100)
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r132 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r132, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r132}
				ctx.BindReg(r132, &d102)
			}
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			var d103 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() + d102.Imm.Int())}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				r133 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r133, d101.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d103)
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d102.Reg}
				ctx.BindReg(d102.Reg, &d103)
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(scratch, d101.Reg)
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d102.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d103)
			} else {
				r134 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r134, d101.Reg)
				ctx.W.EmitAddInt64(r134, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d103)
			}
			if d103.Loc == scm.LocReg && d101.Loc == scm.LocReg && d103.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d102)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			r135 := d34.Loc == scm.LocReg
			r136 := d34.Reg
			if r135 { ctx.ProtectReg(r136) }
			lbl28 := ctx.W.ReserveLabel()
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d104 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r137, d34.Reg)
				ctx.W.EmitShlRegImm8(r137, 32)
				ctx.W.EmitShrRegImm8(r137, 32)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
				ctx.BindReg(r137, &d104)
			}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r138, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r138}
				ctx.BindReg(r138, &d105)
			}
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r139, d105.Reg)
				ctx.W.EmitShlRegImm8(r139, 56)
				ctx.W.EmitShrRegImm8(r139, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d106)
			}
			ctx.FreeDesc(&d105)
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			var d107 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() * d106.Imm.Int())}
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			} else if d106.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(scratch, d104.Reg)
				if d106.Imm.Int() >= -2147483648 && d106.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d106.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d106.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			} else {
				r140 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r140, d104.Reg)
				ctx.W.EmitImulInt64(r140, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d107)
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
				r141 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r141, thisptr.Reg, off)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r141}
				ctx.BindReg(r141, &d108)
			}
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d109 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r142 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r142, d107.Reg)
				ctx.W.EmitShrRegImm8(r142, 6)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d109)
			}
			if d109.Loc == scm.LocReg && d107.Loc == scm.LocReg && d109.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			r143 := ctx.AllocReg()
			if d109.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r143, uint64(d109.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r143, d109.Reg)
				ctx.W.EmitShlRegImm8(r143, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r143, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r143, d108.Reg)
			}
			r144 := ctx.AllocRegExcept(r143)
			ctx.W.EmitMovRegMem(r144, r143, 0)
			ctx.FreeReg(r143)
			d110 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r144}
			ctx.BindReg(r144, &d110)
			ctx.FreeDesc(&d109)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d111 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r145 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r145, d107.Reg)
				ctx.W.EmitAndRegImm32(r145, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d111)
			}
			if d111.Loc == scm.LocReg && d107.Loc == scm.LocReg && d111.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			if d110.Loc == scm.LocStack || d110.Loc == scm.LocStackPair { ctx.EnsureDesc(&d110) }
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			var d112 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d110.Imm.Int()) << uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				r146 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r146, d110.Reg)
				ctx.W.EmitShlRegImm8(r146, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d112)
			} else {
				{
					shiftSrc := d110.Reg
					r147 := ctx.AllocRegExcept(d110.Reg)
					ctx.W.EmitMovRegReg(r147, d110.Reg)
					shiftSrc = r147
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
					ctx.BindReg(shiftSrc, &d112)
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
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r148, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
				ctx.BindReg(r148, &d113)
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
			r149 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r149, 24)
			ctx.ProtectReg(r149)
			d114 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r149}
			ctx.BindReg(r149, &d114)
			ctx.UnprotectReg(r149)
			var d115 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d115)
			}
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d115.Reg)
				ctx.W.EmitShlRegImm8(r151, 56)
				ctx.W.EmitShrRegImm8(r151, 56)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d116)
			}
			ctx.FreeDesc(&d115)
			d117 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair { ctx.EnsureDesc(&d116) }
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() - d116.Imm.Int())}
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				r152 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r152, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d118)
			} else if d117.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d117.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(scratch, d117.Reg)
				if d116.Imm.Int() >= -2147483648 && d116.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d116.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else {
				r153 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r153, d117.Reg)
				ctx.W.EmitSubInt64(r153, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d118)
			}
			if d118.Loc == scm.LocReg && d117.Loc == scm.LocReg && d118.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			var d119 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d114.Imm.Int()) >> uint64(d118.Imm.Int())))}
			} else if d118.Loc == scm.LocImm {
				r154 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r154, d114.Reg)
				ctx.W.EmitShrRegImm8(r154, uint8(d118.Imm.Int()))
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d119)
			} else {
				{
					shiftSrc := d114.Reg
					r155 := ctx.AllocRegExcept(d114.Reg)
					ctx.W.EmitMovRegReg(r155, d114.Reg)
					shiftSrc = r155
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
					ctx.BindReg(shiftSrc, &d119)
				}
			}
			if d119.Loc == scm.LocReg && d114.Loc == scm.LocReg && d119.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			ctx.FreeDesc(&d118)
			r156 := ctx.AllocReg()
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			ctx.EmitMovToReg(r156, d119)
			ctx.W.EmitJmp(lbl28)
			ctx.W.MarkLabel(lbl29)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d120 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r157 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r157, d107.Reg)
				ctx.W.EmitAndRegImm32(r157, 63)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d120)
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
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d121)
			}
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d121.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d121.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d122)
			}
			ctx.FreeDesc(&d121)
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d123 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() + d122.Imm.Int())}
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r160, d120.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d123)
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
				ctx.BindReg(d122.Reg, &d123)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(scratch, d120.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d122.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else {
				r161 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r161, d120.Reg)
				ctx.W.EmitAddInt64(r161, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d123)
			}
			if d123.Loc == scm.LocReg && d120.Loc == scm.LocReg && d123.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			ctx.FreeDesc(&d122)
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d123.Imm.Int()) > uint64(64))}
			} else {
				r162 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitCmpRegImm32(d123.Reg, 64)
				ctx.W.EmitSetcc(r162, scm.CcA)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
				ctx.BindReg(r162, &d124)
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
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d125 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r163 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r163, d107.Reg)
				ctx.W.EmitShrRegImm8(r163, 6)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d125)
			}
			if d125.Loc == scm.LocReg && d107.Loc == scm.LocReg && d125.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(scratch, d125.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			}
			if d126.Loc == scm.LocReg && d125.Loc == scm.LocReg && d126.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			r164 := ctx.AllocReg()
			if d126.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r164, uint64(d126.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r164, d126.Reg)
				ctx.W.EmitShlRegImm8(r164, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r164, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r164, d108.Reg)
			}
			r165 := ctx.AllocRegExcept(r164)
			ctx.W.EmitMovRegMem(r165, r164, 0)
			ctx.FreeReg(r164)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
			ctx.BindReg(r165, &d127)
			ctx.FreeDesc(&d126)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d128 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r166 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r166, d107.Reg)
				ctx.W.EmitAndRegImm32(r166, 63)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d128)
			}
			if d128.Loc == scm.LocReg && d107.Loc == scm.LocReg && d128.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			d129 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			var d130 scm.JITValueDesc
			if d129.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() - d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				r167 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r167, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d130)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d129.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(scratch, d129.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r168 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r168, d129.Reg)
				ctx.W.EmitSubInt64(r168, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d130)
			}
			if d130.Loc == scm.LocReg && d129.Loc == scm.LocReg && d130.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			if d130.Loc == scm.LocStack || d130.Loc == scm.LocStackPair { ctx.EnsureDesc(&d130) }
			var d131 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d127.Imm.Int()) >> uint64(d130.Imm.Int())))}
			} else if d130.Loc == scm.LocImm {
				r169 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r169, d127.Reg)
				ctx.W.EmitShrRegImm8(r169, uint8(d130.Imm.Int()))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d131)
			} else {
				{
					shiftSrc := d127.Reg
					r170 := ctx.AllocRegExcept(d127.Reg)
					ctx.W.EmitMovRegReg(r170, d127.Reg)
					shiftSrc = r170
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
					ctx.BindReg(shiftSrc, &d131)
				}
			}
			if d131.Loc == scm.LocReg && d127.Loc == scm.LocReg && d131.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.FreeDesc(&d130)
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			if d131.Loc == scm.LocStack || d131.Loc == scm.LocStackPair { ctx.EnsureDesc(&d131) }
			var d132 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d112.Imm.Int() | d131.Imm.Int())}
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d131.Reg}
				ctx.BindReg(d131.Reg, &d132)
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				r171 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r171, d112.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d132)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d132)
			} else if d131.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r172, d112.Reg)
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r172, int32(d131.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
					ctx.W.EmitOrInt64(r172, scm.RegR11)
				}
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d132)
			} else {
				r173 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r173, d112.Reg)
				ctx.W.EmitOrInt64(r173, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d132)
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
			d133 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
			ctx.BindReg(r156, &d133)
			ctx.BindReg(r156, &d133)
			if r135 { ctx.UnprotectReg(r136) }
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d133.Imm.Int()))))}
			} else {
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r174, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d134)
			}
			ctx.FreeDesc(&d133)
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r175, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r175}
				ctx.BindReg(r175, &d135)
			}
			if d134.Loc == scm.LocStack || d134.Loc == scm.LocStackPair { ctx.EnsureDesc(&d134) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			var d136 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r176, d134.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d136)
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
				ctx.BindReg(d135.Reg, &d136)
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d136)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(scratch, d134.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d136)
			} else {
				r177 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r177, d134.Reg)
				ctx.W.EmitAddInt64(r177, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d136)
			}
			if d136.Loc == scm.LocReg && d134.Loc == scm.LocReg && d136.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d134)
			ctx.FreeDesc(&d135)
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			var d138 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() + d136.Imm.Int())}
			} else if d136.Loc == scm.LocImm && d136.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r178, d103.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d138)
			} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d136.Reg}
				ctx.BindReg(d136.Reg, &d138)
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d138)
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
				ctx.BindReg(scratch, &d138)
			} else {
				r179 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r179, d103.Reg)
				ctx.W.EmitAddInt64(r179, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d138)
			}
			if d138.Loc == scm.LocReg && d103.Loc == scm.LocReg && d138.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			var d140 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r180, fieldAddr)
				ctx.W.EmitMovRegMem64(r181, fieldAddr+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d140)
				ctx.BindReg(r181, &d140)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r182 := ctx.AllocReg()
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r182, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
				ctx.BindReg(r182, &d140)
				ctx.BindReg(r183, &d140)
			}
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			r184 := ctx.AllocReg()
			r185 := ctx.AllocRegExcept(r184)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r184, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r184, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r184, d140.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() != 0 {
					if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r184, int32(d103.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
						ctx.W.EmitAddInt64(r184, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r184, d103.Reg)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r185, uint64(d138.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r185, d138.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r185, int32(d103.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitSubInt64(r185, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r185, d103.Reg)
			}
			d141 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d141)
			ctx.BindReg(r185, &d141)
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
				r186 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r186, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r186}
				ctx.BindReg(r186, &d143)
			}
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d144 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d143.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r187, d143.Reg)
				ctx.W.EmitShlRegImm8(r187, 32)
				ctx.W.EmitShrRegImm8(r187, 32)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d144)
			}
			ctx.FreeDesc(&d143)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			var d145 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d34.Imm.Int()) == uint64(d144.Imm.Int()))}
			} else if d144.Loc == scm.LocImm {
				r188 := ctx.AllocRegExcept(d34.Reg)
				if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d34.Reg, int32(d144.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
					ctx.W.EmitCmpInt64(d34.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r188, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r188}
				ctx.BindReg(r188, &d145)
			} else if d34.Loc == scm.LocImm {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d144.Reg)
				ctx.W.EmitSetcc(r189, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r189}
				ctx.BindReg(r189, &d145)
			} else {
				r190 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitCmpInt64(d34.Reg, d144.Reg)
				ctx.W.EmitSetcc(r190, scm.CcE)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r190}
				ctx.BindReg(r190, &d145)
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			lbl36 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d146 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d146)
			}
			ctx.FreeDesc(&idxInt)
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r192, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r192}
				ctx.BindReg(r192, &d147)
			}
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			var d148 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d147.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, d147.Reg)
				ctx.W.EmitShlRegImm8(r193, 56)
				ctx.W.EmitShrRegImm8(r193, 56)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d148)
			}
			ctx.FreeDesc(&d147)
			if d146.Loc == scm.LocStack || d146.Loc == scm.LocStackPair { ctx.EnsureDesc(&d146) }
			if d148.Loc == scm.LocStack || d148.Loc == scm.LocStackPair { ctx.EnsureDesc(&d148) }
			var d149 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d146.Imm.Int() * d148.Imm.Int())}
			} else if d146.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d146.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(scratch, d146.Reg)
				if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d148.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else {
				r194 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r194, d146.Reg)
				ctx.W.EmitImulInt64(r194, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d149)
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
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r195, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r195}
				ctx.BindReg(r195, &d150)
			}
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d151 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() / 64)}
			} else {
				r196 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r196, d149.Reg)
				ctx.W.EmitShrRegImm8(r196, 6)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d151)
			}
			if d151.Loc == scm.LocReg && d149.Loc == scm.LocReg && d151.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			if d151.Loc == scm.LocStack || d151.Loc == scm.LocStackPair { ctx.EnsureDesc(&d151) }
			r197 := ctx.AllocReg()
			if d151.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r197, uint64(d151.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r197, d151.Reg)
				ctx.W.EmitShlRegImm8(r197, 3)
			}
			if d150.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
				ctx.W.EmitAddInt64(r197, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r197, d150.Reg)
			}
			r198 := ctx.AllocRegExcept(r197)
			ctx.W.EmitMovRegMem(r198, r197, 0)
			ctx.FreeReg(r197)
			d152 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
			ctx.BindReg(r198, &d152)
			ctx.FreeDesc(&d151)
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d153 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				r199 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r199, d149.Reg)
				ctx.W.EmitAndRegImm32(r199, 63)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d153)
			}
			if d153.Loc == scm.LocReg && d149.Loc == scm.LocReg && d153.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			var d154 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d152.Imm.Int()) << uint64(d153.Imm.Int())))}
			} else if d153.Loc == scm.LocImm {
				r200 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r200, d152.Reg)
				ctx.W.EmitShlRegImm8(r200, uint8(d153.Imm.Int()))
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d154)
			} else {
				{
					shiftSrc := d152.Reg
					r201 := ctx.AllocRegExcept(d152.Reg)
					ctx.W.EmitMovRegReg(r201, d152.Reg)
					shiftSrc = r201
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
					ctx.BindReg(shiftSrc, &d154)
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
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d155)
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
			r203 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r203, 32)
			ctx.ProtectReg(r203)
			d156 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r203}
			ctx.BindReg(r203, &d156)
			ctx.UnprotectReg(r203)
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r204, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d157)
			}
			if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair { ctx.EnsureDesc(&d157) }
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r205 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r205, d157.Reg)
				ctx.W.EmitShlRegImm8(r205, 56)
				ctx.W.EmitShrRegImm8(r205, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d158)
			}
			ctx.FreeDesc(&d157)
			d159 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() - d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r206 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(r206, d159.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d160)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(scratch, d159.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d160)
			} else {
				r207 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(r207, d159.Reg)
				ctx.W.EmitSubInt64(r207, d158.Reg)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d160)
			}
			if d160.Loc == scm.LocReg && d159.Loc == scm.LocReg && d160.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			if d156.Loc == scm.LocStack || d156.Loc == scm.LocStackPair { ctx.EnsureDesc(&d156) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			var d161 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d156.Imm.Int()) >> uint64(d160.Imm.Int())))}
			} else if d160.Loc == scm.LocImm {
				r208 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r208, d156.Reg)
				ctx.W.EmitShrRegImm8(r208, uint8(d160.Imm.Int()))
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d161)
			} else {
				{
					shiftSrc := d156.Reg
					r209 := ctx.AllocRegExcept(d156.Reg)
					ctx.W.EmitMovRegReg(r209, d156.Reg)
					shiftSrc = r209
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
					ctx.BindReg(shiftSrc, &d161)
				}
			}
			if d161.Loc == scm.LocReg && d156.Loc == scm.LocReg && d161.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d160)
			r210 := ctx.AllocReg()
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			ctx.EmitMovToReg(r210, d161)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl37)
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d162 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				r211 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r211, d149.Reg)
				ctx.W.EmitAndRegImm32(r211, 63)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d162)
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
				r212 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r212, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r212}
				ctx.BindReg(r212, &d163)
			}
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d163.Imm.Int()))))}
			} else {
				r213 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r213, d163.Reg)
				ctx.W.EmitShlRegImm8(r213, 56)
				ctx.W.EmitShrRegImm8(r213, 56)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d164)
			}
			ctx.FreeDesc(&d163)
			if d162.Loc == scm.LocStack || d162.Loc == scm.LocStackPair { ctx.EnsureDesc(&d162) }
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			var d165 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r214 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(r214, d162.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d165)
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
				ctx.BindReg(d164.Reg, &d165)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(scratch, d162.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d164.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else {
				r215 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(r215, d162.Reg)
				ctx.W.EmitAddInt64(r215, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d165)
			}
			if d165.Loc == scm.LocReg && d162.Loc == scm.LocReg && d165.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d164)
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			var d166 scm.JITValueDesc
			if d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d165.Imm.Int()) > uint64(64))}
			} else {
				r216 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitCmpRegImm32(d165.Reg, 64)
				ctx.W.EmitSetcc(r216, scm.CcA)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r216}
				ctx.BindReg(r216, &d166)
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
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d167 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() / 64)}
			} else {
				r217 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r217, d149.Reg)
				ctx.W.EmitShrRegImm8(r217, 6)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d167)
			}
			if d167.Loc == scm.LocReg && d149.Loc == scm.LocReg && d167.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(scratch, d167.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			}
			if d168.Loc == scm.LocReg && d167.Loc == scm.LocReg && d168.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			r218 := ctx.AllocReg()
			if d168.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r218, uint64(d168.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r218, d168.Reg)
				ctx.W.EmitShlRegImm8(r218, 3)
			}
			if d150.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
				ctx.W.EmitAddInt64(r218, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r218, d150.Reg)
			}
			r219 := ctx.AllocRegExcept(r218)
			ctx.W.EmitMovRegMem(r219, r218, 0)
			ctx.FreeReg(r218)
			d169 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r219}
			ctx.BindReg(r219, &d169)
			ctx.FreeDesc(&d168)
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d170 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() % 64)}
			} else {
				r220 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r220, d149.Reg)
				ctx.W.EmitAndRegImm32(r220, 63)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d170)
			}
			if d170.Loc == scm.LocReg && d149.Loc == scm.LocReg && d170.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			d171 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d172 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d171.Imm.Int() - d170.Imm.Int())}
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r221 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r221, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d172)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d171.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(scratch, d171.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else {
				r222 := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegReg(r222, d171.Reg)
				ctx.W.EmitSubInt64(r222, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d172)
			}
			if d172.Loc == scm.LocReg && d171.Loc == scm.LocReg && d172.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			var d173 scm.JITValueDesc
			if d169.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d169.Imm.Int()) >> uint64(d172.Imm.Int())))}
			} else if d172.Loc == scm.LocImm {
				r223 := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegReg(r223, d169.Reg)
				ctx.W.EmitShrRegImm8(r223, uint8(d172.Imm.Int()))
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d173)
			} else {
				{
					shiftSrc := d169.Reg
					r224 := ctx.AllocRegExcept(d169.Reg)
					ctx.W.EmitMovRegReg(r224, d169.Reg)
					shiftSrc = r224
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
					ctx.BindReg(shiftSrc, &d173)
				}
			}
			if d173.Loc == scm.LocReg && d169.Loc == scm.LocReg && d173.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			ctx.FreeDesc(&d172)
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			var d174 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d154.Imm.Int() | d173.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
				ctx.BindReg(d173.Reg, &d174)
			} else if d173.Loc == scm.LocImm && d173.Imm.Int() == 0 {
				r225 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r225, d154.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d174)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d154.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d174)
			} else if d173.Loc == scm.LocImm {
				r226 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r226, d154.Reg)
				if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r226, int32(d173.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
					ctx.W.EmitOrInt64(r226, scm.RegR11)
				}
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d174)
			} else {
				r227 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r227, d154.Reg)
				ctx.W.EmitOrInt64(r227, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d174)
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
			d175 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
			ctx.BindReg(r210, &d175)
			ctx.BindReg(r210, &d175)
			ctx.FreeDesc(&idxInt)
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d175.Imm.Int()))))}
			} else {
				r228 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r228, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d176)
			}
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r229 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r229, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r229}
				ctx.BindReg(r229, &d177)
			}
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			var d178 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d177.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() + d177.Imm.Int())}
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				r230 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r230, d176.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d178)
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
				ctx.BindReg(d177.Reg, &d178)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(scratch, d176.Reg)
				if d177.Imm.Int() >= -2147483648 && d177.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d177.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d178)
			} else {
				r231 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r231, d176.Reg)
				ctx.W.EmitAddInt64(r231, d177.Reg)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d178)
			}
			if d178.Loc == scm.LocReg && d176.Loc == scm.LocReg && d178.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d177)
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			var d179 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d178.Imm.Int()))))}
			} else {
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r232, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d179)
			}
			ctx.FreeDesc(&d178)
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			var d180 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d69.Imm.Int()))))}
			} else {
				r233 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r233, d69.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d180)
			}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			var d181 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + d179.Imm.Int())}
			} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
				r234 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r234, d69.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d181)
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
				ctx.BindReg(d179.Reg, &d181)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d179.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d181)
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
				ctx.BindReg(scratch, &d181)
			} else {
				r235 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r235, d69.Reg)
				ctx.W.EmitAddInt64(r235, d179.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d181)
			}
			if d181.Loc == scm.LocReg && d69.Loc == scm.LocReg && d181.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d181.Imm.Int()))))}
			} else {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r236, d181.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d182)
			}
			ctx.FreeDesc(&d181)
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			r237 := ctx.AllocReg()
			r238 := ctx.AllocRegExcept(r237)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r237, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r237, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r237, d140.Reg)
			}
			if d180.Loc == scm.LocImm {
				if d180.Imm.Int() != 0 {
					if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r237, int32(d180.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
						ctx.W.EmitAddInt64(r237, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r237, d180.Reg)
			}
			if d182.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r238, uint64(d182.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r238, d182.Reg)
			}
			if d180.Loc == scm.LocImm {
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r238, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitSubInt64(r238, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r238, d180.Reg)
			}
			d183 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r237, Reg2: r238}
			ctx.BindReg(r237, &d183)
			ctx.BindReg(r238, &d183)
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
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r239, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r239}
				ctx.BindReg(r239, &d185)
			}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			var d186 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d185.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d69.Imm.Int()) == uint64(d185.Imm.Int()))}
			} else if d185.Loc == scm.LocImm {
				r240 := ctx.AllocRegExcept(d69.Reg)
				if d185.Imm.Int() >= -2147483648 && d185.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d69.Reg, int32(d185.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
					ctx.W.EmitCmpInt64(d69.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r240, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d186)
			} else if d69.Loc == scm.LocImm {
				r241 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d185.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d186)
			} else {
				r242 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitCmpInt64(d69.Reg, d185.Reg)
				ctx.W.EmitSetcc(r242, scm.CcE)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r242}
				ctx.BindReg(r242, &d186)
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
