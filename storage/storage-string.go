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
			var bbs [9]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			bbpos_0_8 := int32(-1)
			_ = bbpos_0_8
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			bbs[0].RenderCount++
			bbpos_0_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d1.Loc == scm.LocImm {
				if d1.Imm.Bool() {
					ctx.W.MarkLabel(lbl7)
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.W.MarkLabel(lbl8)
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl7)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d0)
			bbs[2].RenderCount++
			bbpos_0_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl6)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&idxInt)
			d2 := idxInt
			_ = d2
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 := ctx.W.EmitSubRSP32Fixup()
			lbl9 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d13.Loc == scm.LocImm {
				if d13.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.MarkLabel(lbl13)
			d14 := d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, 0)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d13.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
			d15 := d11
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d12)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl11)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl9)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl10)
			ctx.W.ResolveFixups()
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
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.MarkLabel(lbl16)
			d28 := d11
			if d28.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 0)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl16)
			d29 := d11
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d26)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl14)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl9)
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
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d44)
			bbs[7].RenderCount++
			bbpos_0_7 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d43)
			d46 := d43
			_ = d46
			r48 := d43.Loc == scm.LocReg
			r49 := d43.Reg
			if r48 { ctx.ProtectReg(r49) }
			lbl20 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl20)
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl21)
			ctx.W.ResolveFixups()
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
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl25)
			ctx.W.ResolveFixups()
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
			ctx.W.MarkLabel(lbl20)
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
			ctx.EnsureDesc(&d43)
			d87 := d43
			_ = d87
			r90 := d43.Loc == scm.LocReg
			r91 := d43.Reg
			if r90 { ctx.ProtectReg(r91) }
			lbl28 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d87.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d87.Reg)
				ctx.W.EmitShlRegImm8(r92, 32)
				ctx.W.EmitShrRegImm8(r92, 32)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d88)
			}
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d89)
			}
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d89)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d89.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d89.Reg)
				ctx.W.EmitShlRegImm8(r94, 56)
				ctx.W.EmitShrRegImm8(r94, 56)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d90)
			}
			ctx.FreeDesc(&d89)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d88.Loc == scm.LocImm && d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() * d90.Imm.Int())}
			} else if d88.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d88.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d90.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d91)
			} else if d90.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(scratch, d88.Reg)
				if d90.Imm.Int() >= -2147483648 && d90.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d90.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d90.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d91)
			} else {
				r95 := ctx.AllocRegExcept(d88.Reg, d90.Reg)
				ctx.W.EmitMovRegReg(r95, d88.Reg)
				ctx.W.EmitImulInt64(r95, d90.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r95}
				ctx.BindReg(r95, &d91)
			}
			if d91.Loc == scm.LocReg && d88.Loc == scm.LocReg && d91.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			ctx.FreeDesc(&d90)
			var d92 scm.JITValueDesc
			r96 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r96, uint64(dataPtr))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96, StackOff: int32(sliceLen)}
				ctx.BindReg(r96, &d92)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r96, thisptr.Reg, off)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r96}
				ctx.BindReg(r96, &d92)
			}
			ctx.BindReg(r96, &d92)
			ctx.EnsureDesc(&d91)
			var d93 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() / 64)}
			} else {
				r97 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r97, d91.Reg)
				ctx.W.EmitShrRegImm8(r97, 6)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d93)
			}
			if d93.Loc == scm.LocReg && d91.Loc == scm.LocReg && d93.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d93)
			r98 := ctx.AllocReg()
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d92)
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r98, uint64(d93.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r98, d93.Reg)
				ctx.W.EmitShlRegImm8(r98, 3)
			}
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
				ctx.W.EmitAddInt64(r98, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r98, d92.Reg)
			}
			r99 := ctx.AllocRegExcept(r98)
			ctx.W.EmitMovRegMem(r99, r98, 0)
			ctx.FreeReg(r98)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			ctx.BindReg(r99, &d94)
			ctx.FreeDesc(&d93)
			ctx.EnsureDesc(&d91)
			var d95 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r100 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r100, d91.Reg)
				ctx.W.EmitAndRegImm32(r100, 63)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
				ctx.BindReg(r100, &d95)
			}
			if d95.Loc == scm.LocReg && d91.Loc == scm.LocReg && d95.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d94)
			ctx.EnsureDesc(&d95)
			var d96 scm.JITValueDesc
			if d94.Loc == scm.LocImm && d95.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) << uint64(d95.Imm.Int())))}
			} else if d95.Loc == scm.LocImm {
				r101 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r101, d94.Reg)
				ctx.W.EmitShlRegImm8(r101, uint8(d95.Imm.Int()))
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d96)
			} else {
				{
					shiftSrc := d94.Reg
					r102 := ctx.AllocRegExcept(d94.Reg)
					ctx.W.EmitMovRegReg(r102, d94.Reg)
					shiftSrc = r102
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d95.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d95.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d95.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d96)
				}
			}
			if d96.Loc == scm.LocReg && d94.Loc == scm.LocReg && d96.Reg == d94.Reg {
				ctx.TransferReg(d94.Reg)
				d94.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			ctx.FreeDesc(&d95)
			var d97 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r103, thisptr.Reg, off)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
				ctx.BindReg(r103, &d97)
			}
			d98 := d97
			ctx.EnsureDesc(&d98)
			if d98.Loc != scm.LocImm && d98.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d98.Loc == scm.LocImm {
				if d98.Imm.Bool() {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl29)
				} else {
					ctx.W.MarkLabel(lbl32)
			d99 := d96
			if d99.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d99)
			ctx.EmitStoreToStack(d99, 16)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d98.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl32)
			d100 := d96
			if d100.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d100)
			ctx.EmitStoreToStack(d100, 16)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d97)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl30)
			ctx.W.ResolveFixups()
			d101 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r104 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r104, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r104}
				ctx.BindReg(r104, &d102)
			}
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d102)
			var d103 scm.JITValueDesc
			if d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d102.Imm.Int()))))}
			} else {
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r105, d102.Reg)
				ctx.W.EmitShlRegImm8(r105, 56)
				ctx.W.EmitShrRegImm8(r105, 56)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d103)
			}
			ctx.FreeDesc(&d102)
			d104 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d103)
			var d105 scm.JITValueDesc
			if d104.Loc == scm.LocImm && d103.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d104.Imm.Int() - d103.Imm.Int())}
			} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
				r106 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r106, d104.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r106}
				ctx.BindReg(r106, &d105)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d104.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d103.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d105)
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(scratch, d104.Reg)
				if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d103.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d105)
			} else {
				r107 := ctx.AllocRegExcept(d104.Reg, d103.Reg)
				ctx.W.EmitMovRegReg(r107, d104.Reg)
				ctx.W.EmitSubInt64(r107, d103.Reg)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d105)
			}
			if d105.Loc == scm.LocReg && d104.Loc == scm.LocReg && d105.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d103)
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d105)
			var d106 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d101.Imm.Int()) >> uint64(d105.Imm.Int())))}
			} else if d105.Loc == scm.LocImm {
				r108 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r108, d101.Reg)
				ctx.W.EmitShrRegImm8(r108, uint8(d105.Imm.Int()))
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d106)
			} else {
				{
					shiftSrc := d101.Reg
					r109 := ctx.AllocRegExcept(d101.Reg)
					ctx.W.EmitMovRegReg(r109, d101.Reg)
					shiftSrc = r109
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d105.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d105.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d105.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d106)
				}
			}
			if d106.Loc == scm.LocReg && d101.Loc == scm.LocReg && d106.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d105)
			r110 := ctx.AllocReg()
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d106)
			if d106.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r110, d106)
			}
			ctx.W.EmitJmp(lbl28)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d91)
			var d107 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r111 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r111, d91.Reg)
				ctx.W.EmitAndRegImm32(r111, 63)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d107)
			}
			if d107.Loc == scm.LocReg && d91.Loc == scm.LocReg && d107.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			var d108 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r112 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r112, thisptr.Reg, off)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
				ctx.BindReg(r112, &d108)
			}
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d108)
			var d109 scm.JITValueDesc
			if d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d108.Imm.Int()))))}
			} else {
				r113 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r113, d108.Reg)
				ctx.W.EmitShlRegImm8(r113, 56)
				ctx.W.EmitShrRegImm8(r113, 56)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d109)
			}
			ctx.FreeDesc(&d108)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d107.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() + d109.Imm.Int())}
			} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
				r114 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r114, d107.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r114}
				ctx.BindReg(r114, &d110)
			} else if d107.Loc == scm.LocImm && d107.Imm.Int() == 0 {
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d109.Reg}
				ctx.BindReg(d109.Reg, &d110)
			} else if d107.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d109.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			} else if d109.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(scratch, d107.Reg)
				if d109.Imm.Int() >= -2147483648 && d109.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d109.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d109.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d110)
			} else {
				r115 := ctx.AllocRegExcept(d107.Reg, d109.Reg)
				ctx.W.EmitMovRegReg(r115, d107.Reg)
				ctx.W.EmitAddInt64(r115, d109.Reg)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d110)
			}
			if d110.Loc == scm.LocReg && d107.Loc == scm.LocReg && d110.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d110)
			var d111 scm.JITValueDesc
			if d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d110.Imm.Int()) > uint64(64))}
			} else {
				r116 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitCmpRegImm32(d110.Reg, 64)
				ctx.W.EmitSetcc(r116, scm.CcA)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r116}
				ctx.BindReg(r116, &d111)
			}
			ctx.FreeDesc(&d110)
			d112 := d111
			ctx.EnsureDesc(&d112)
			if d112.Loc != scm.LocImm && d112.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d112.Loc == scm.LocImm {
				if d112.Imm.Bool() {
					ctx.W.MarkLabel(lbl34)
					ctx.W.EmitJmp(lbl33)
				} else {
					ctx.W.MarkLabel(lbl35)
			d113 := d96
			if d113.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d113)
			ctx.EmitStoreToStack(d113, 16)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d112.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl35)
			d114 := d96
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, 16)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d111)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl33)
			ctx.W.ResolveFixups()
			d101 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d91)
			var d115 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() / 64)}
			} else {
				r117 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r117, d91.Reg)
				ctx.W.EmitShrRegImm8(r117, 6)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d115)
			}
			if d115.Loc == scm.LocReg && d91.Loc == scm.LocReg && d115.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d115)
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d115.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(scratch, d115.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d116)
			}
			if d116.Loc == scm.LocReg && d115.Loc == scm.LocReg && d116.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			ctx.EnsureDesc(&d116)
			r118 := ctx.AllocReg()
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d92)
			if d116.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r118, uint64(d116.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r118, d116.Reg)
				ctx.W.EmitShlRegImm8(r118, 3)
			}
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
				ctx.W.EmitAddInt64(r118, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r118, d92.Reg)
			}
			r119 := ctx.AllocRegExcept(r118)
			ctx.W.EmitMovRegMem(r119, r118, 0)
			ctx.FreeReg(r118)
			d117 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r119}
			ctx.BindReg(r119, &d117)
			ctx.FreeDesc(&d116)
			ctx.EnsureDesc(&d91)
			var d118 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r120 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r120, d91.Reg)
				ctx.W.EmitAndRegImm32(r120, 63)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d118)
			}
			if d118.Loc == scm.LocReg && d91.Loc == scm.LocReg && d118.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			d119 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d118)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() - d118.Imm.Int())}
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				r121 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r121, d119.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d120)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d118.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d118.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			} else {
				r122 := ctx.AllocRegExcept(d119.Reg, d118.Reg)
				ctx.W.EmitMovRegReg(r122, d119.Reg)
				ctx.W.EmitSubInt64(r122, d118.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d120)
			}
			if d120.Loc == scm.LocReg && d119.Loc == scm.LocReg && d120.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d120)
			var d121 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d117.Imm.Int()) >> uint64(d120.Imm.Int())))}
			} else if d120.Loc == scm.LocImm {
				r123 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r123, d117.Reg)
				ctx.W.EmitShrRegImm8(r123, uint8(d120.Imm.Int()))
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d121)
			} else {
				{
					shiftSrc := d117.Reg
					r124 := ctx.AllocRegExcept(d117.Reg)
					ctx.W.EmitMovRegReg(r124, d117.Reg)
					shiftSrc = r124
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d120.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d120.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d120.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d121)
				}
			}
			if d121.Loc == scm.LocReg && d117.Loc == scm.LocReg && d121.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d117)
			ctx.FreeDesc(&d120)
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() | d121.Imm.Int())}
			} else if d96.Loc == scm.LocImm && d96.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d121.Reg}
				ctx.BindReg(d121.Reg, &d122)
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				r125 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r125, d96.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d122)
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == scm.LocImm {
				r126 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r126, d96.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r126, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitOrInt64(r126, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d122)
			} else {
				r127 := ctx.AllocRegExcept(d96.Reg, d121.Reg)
				ctx.W.EmitMovRegReg(r127, d96.Reg)
				ctx.W.EmitOrInt64(r127, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d122)
			}
			if d122.Loc == scm.LocReg && d96.Loc == scm.LocReg && d122.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d121)
			d123 := d122
			if d123.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d123)
			ctx.EmitStoreToStack(d123, 16)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			d124 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r110}
			ctx.BindReg(r110, &d124)
			ctx.BindReg(r110, &d124)
			if r90 { ctx.UnprotectReg(r91) }
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d124.Imm.Int()))))}
			} else {
				r128 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r128, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d125)
			}
			ctx.FreeDesc(&d124)
			var d126 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r129 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r129, thisptr.Reg, off)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
				ctx.BindReg(r129, &d126)
			}
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d126)
			var d127 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d126.Imm.Int())}
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				r130 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r130, d125.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d127)
			} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
				ctx.BindReg(d126.Reg, &d127)
			} else if d125.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d125.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(scratch, d125.Reg)
				if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d126.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else {
				r131 := ctx.AllocRegExcept(d125.Reg, d126.Reg)
				ctx.W.EmitMovRegReg(r131, d125.Reg)
				ctx.W.EmitAddInt64(r131, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d127)
			}
			if d127.Loc == scm.LocReg && d125.Loc == scm.LocReg && d127.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d126)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d127)
			var d129 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegReg(r132, d86.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d129)
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d129)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegReg(scratch, d86.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d129)
			} else {
				r133 := ctx.AllocRegExcept(d86.Reg, d127.Reg)
				ctx.W.EmitMovRegReg(r133, d86.Reg)
				ctx.W.EmitAddInt64(r133, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d129)
			}
			if d129.Loc == scm.LocReg && d86.Loc == scm.LocReg && d129.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d129)
			var d131 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).dictionary)
				r134 := ctx.AllocReg()
				r135 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r134, fieldAddr)
				ctx.W.EmitMovRegMem64(r135, fieldAddr+8)
				d131 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r134, Reg2: r135}
				ctx.BindReg(r134, &d131)
				ctx.BindReg(r135, &d131)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).dictionary))
				r136 := ctx.AllocReg()
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r136, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r137, thisptr.Reg, off+8)
				d131 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r136, Reg2: r137}
				ctx.BindReg(r136, &d131)
				ctx.BindReg(r137, &d131)
			}
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d129)
			r138 := ctx.AllocReg()
			r139 := ctx.AllocRegExcept(r138)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d129)
			if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r138, uint64(d131.Imm.Int()))
			} else if d131.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r138, d131.Reg)
			} else {
				ctx.W.EmitMovRegReg(r138, d131.Reg)
			}
			if d86.Loc == scm.LocImm {
				if d86.Imm.Int() != 0 {
					if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r138, int32(d86.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
						ctx.W.EmitAddInt64(r138, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r138, d86.Reg)
			}
			if d129.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r139, uint64(d129.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r139, d129.Reg)
			}
			if d86.Loc == scm.LocImm {
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r139, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitSubInt64(r139, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r139, d86.Reg)
			}
			d132 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r138, Reg2: r139}
			ctx.BindReg(r138, &d132)
			ctx.BindReg(r139, &d132)
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d129)
			d133 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d133)
			ctx.BindReg(r1, &d133)
			d134 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d132}, 2)
			ctx.EmitMovPairToResult(&d134, &d133)
			ctx.W.EmitJmp(lbl0)
			bbs[1].RenderCount++
			bbpos_0_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&idxInt)
			d135 := idxInt
			_ = d135
			r140 := idxInt.Loc == scm.LocReg
			r141 := idxInt.Reg
			if r140 { ctx.ProtectReg(r141) }
			lbl36 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d135)
			var d136 scm.JITValueDesc
			if d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d135.Imm.Int()))))}
			} else {
				r142 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r142, d135.Reg)
				ctx.W.EmitShlRegImm8(r142, 32)
				ctx.W.EmitShrRegImm8(r142, 32)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d136)
			}
			var d137 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r143 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r143, thisptr.Reg, off)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
				ctx.BindReg(r143, &d137)
			}
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d137)
			var d138 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d137.Imm.Int()))))}
			} else {
				r144 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r144, d137.Reg)
				ctx.W.EmitShlRegImm8(r144, 56)
				ctx.W.EmitShrRegImm8(r144, 56)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d138)
			}
			ctx.FreeDesc(&d137)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d138)
			var d139 scm.JITValueDesc
			if d136.Loc == scm.LocImm && d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() * d138.Imm.Int())}
			} else if d136.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d136.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(scratch, d136.Reg)
				if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d138.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d139)
			} else {
				r145 := ctx.AllocRegExcept(d136.Reg, d138.Reg)
				ctx.W.EmitMovRegReg(r145, d136.Reg)
				ctx.W.EmitImulInt64(r145, d138.Reg)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d139)
			}
			if d139.Loc == scm.LocReg && d136.Loc == scm.LocReg && d139.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			ctx.FreeDesc(&d138)
			var d140 scm.JITValueDesc
			r146 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r146, uint64(dataPtr))
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146, StackOff: int32(sliceLen)}
				ctx.BindReg(r146, &d140)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 0)
				ctx.W.EmitMovRegMem(r146, thisptr.Reg, off)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
				ctx.BindReg(r146, &d140)
			}
			ctx.BindReg(r146, &d140)
			ctx.EnsureDesc(&d139)
			var d141 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() / 64)}
			} else {
				r147 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r147, d139.Reg)
				ctx.W.EmitShrRegImm8(r147, 6)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d141)
			}
			if d141.Loc == scm.LocReg && d139.Loc == scm.LocReg && d141.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d141)
			r148 := ctx.AllocReg()
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d140)
			if d141.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r148, uint64(d141.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r148, d141.Reg)
				ctx.W.EmitShlRegImm8(r148, 3)
			}
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
				ctx.W.EmitAddInt64(r148, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r148, d140.Reg)
			}
			r149 := ctx.AllocRegExcept(r148)
			ctx.W.EmitMovRegMem(r149, r148, 0)
			ctx.FreeReg(r148)
			d142 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
			ctx.BindReg(r149, &d142)
			ctx.FreeDesc(&d141)
			ctx.EnsureDesc(&d139)
			var d143 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() % 64)}
			} else {
				r150 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r150, d139.Reg)
				ctx.W.EmitAndRegImm32(r150, 63)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d143)
			}
			if d143.Loc == scm.LocReg && d139.Loc == scm.LocReg && d143.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d142.Imm.Int()) << uint64(d143.Imm.Int())))}
			} else if d143.Loc == scm.LocImm {
				r151 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r151, d142.Reg)
				ctx.W.EmitShlRegImm8(r151, uint8(d143.Imm.Int()))
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d144)
			} else {
				{
					shiftSrc := d142.Reg
					r152 := ctx.AllocRegExcept(d142.Reg)
					ctx.W.EmitMovRegReg(r152, d142.Reg)
					shiftSrc = r152
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d143.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d143.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d143.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d144)
				}
			}
			if d144.Loc == scm.LocReg && d142.Loc == scm.LocReg && d144.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			ctx.FreeDesc(&d143)
			var d145 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 25)
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r153, thisptr.Reg, off)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
				ctx.BindReg(r153, &d145)
			}
			d146 := d145
			ctx.EnsureDesc(&d146)
			if d146.Loc != scm.LocImm && d146.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d146.Loc == scm.LocImm {
				if d146.Imm.Bool() {
					ctx.W.MarkLabel(lbl39)
					ctx.W.EmitJmp(lbl37)
				} else {
					ctx.W.MarkLabel(lbl40)
			d147 := d144
			if d147.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d147)
			ctx.EmitStoreToStack(d147, 24)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d146.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.W.EmitJmp(lbl40)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl40)
			d148 := d144
			if d148.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d148)
			ctx.EmitStoreToStack(d148, 24)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d145)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl38)
			ctx.W.ResolveFixups()
			d149 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d150 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r154 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r154, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
				ctx.BindReg(r154, &d150)
			}
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d150)
			var d151 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d150.Imm.Int()))))}
			} else {
				r155 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r155, d150.Reg)
				ctx.W.EmitShlRegImm8(r155, 56)
				ctx.W.EmitShrRegImm8(r155, 56)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d151)
			}
			ctx.FreeDesc(&d150)
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
				r156 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r156, d152.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d153)
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
				r157 := ctx.AllocRegExcept(d152.Reg, d151.Reg)
				ctx.W.EmitMovRegReg(r157, d152.Reg)
				ctx.W.EmitSubInt64(r157, d151.Reg)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d153)
			}
			if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d153)
			var d154 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d149.Imm.Int()) >> uint64(d153.Imm.Int())))}
			} else if d153.Loc == scm.LocImm {
				r158 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r158, d149.Reg)
				ctx.W.EmitShrRegImm8(r158, uint8(d153.Imm.Int()))
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d154)
			} else {
				{
					shiftSrc := d149.Reg
					r159 := ctx.AllocRegExcept(d149.Reg)
					ctx.W.EmitMovRegReg(r159, d149.Reg)
					shiftSrc = r159
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
			if d154.Loc == scm.LocReg && d149.Loc == scm.LocReg && d154.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			ctx.FreeDesc(&d153)
			r160 := ctx.AllocReg()
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d154)
			if d154.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r160, d154)
			}
			ctx.W.EmitJmp(lbl36)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d149 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d139)
			var d155 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() % 64)}
			} else {
				r161 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r161, d139.Reg)
				ctx.W.EmitAndRegImm32(r161, 63)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d155)
			}
			if d155.Loc == scm.LocReg && d139.Loc == scm.LocReg && d155.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			var d156 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 24)
				r162 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r162, thisptr.Reg, off)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r162}
				ctx.BindReg(r162, &d156)
			}
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d156)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d156.Imm.Int()))))}
			} else {
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r163, d156.Reg)
				ctx.W.EmitShlRegImm8(r163, 56)
				ctx.W.EmitShrRegImm8(r163, 56)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d157)
			}
			ctx.FreeDesc(&d156)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() + d157.Imm.Int())}
			} else if d157.Loc == scm.LocImm && d157.Imm.Int() == 0 {
				r164 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r164, d155.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d158)
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d157.Reg}
				ctx.BindReg(d157.Reg, &d158)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d157.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else if d157.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(scratch, d155.Reg)
				if d157.Imm.Int() >= -2147483648 && d157.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d157.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d157.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else {
				r165 := ctx.AllocRegExcept(d155.Reg, d157.Reg)
				ctx.W.EmitMovRegReg(r165, d155.Reg)
				ctx.W.EmitAddInt64(r165, d157.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d158)
			}
			if d158.Loc == scm.LocReg && d155.Loc == scm.LocReg && d158.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d158.Imm.Int()) > uint64(64))}
			} else {
				r166 := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitCmpRegImm32(d158.Reg, 64)
				ctx.W.EmitSetcc(r166, scm.CcA)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r166}
				ctx.BindReg(r166, &d159)
			}
			ctx.FreeDesc(&d158)
			d160 := d159
			ctx.EnsureDesc(&d160)
			if d160.Loc != scm.LocImm && d160.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d160.Loc == scm.LocImm {
				if d160.Imm.Bool() {
					ctx.W.MarkLabel(lbl42)
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.MarkLabel(lbl43)
			d161 := d144
			if d161.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d161)
			ctx.EmitStoreToStack(d161, 24)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d160.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl43)
			d162 := d144
			if d162.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d162)
			ctx.EmitStoreToStack(d162, 24)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d159)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl41)
			ctx.W.ResolveFixups()
			d149 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d139)
			var d163 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() / 64)}
			} else {
				r167 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r167, d139.Reg)
				ctx.W.EmitShrRegImm8(r167, 6)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d163)
			}
			if d163.Loc == scm.LocReg && d139.Loc == scm.LocReg && d163.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d163)
			var d164 scm.JITValueDesc
			if d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(scratch, d163.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d164)
			}
			if d164.Loc == scm.LocReg && d163.Loc == scm.LocReg && d164.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d164)
			r168 := ctx.AllocReg()
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d140)
			if d164.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r168, uint64(d164.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r168, d164.Reg)
				ctx.W.EmitShlRegImm8(r168, 3)
			}
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d140.Imm.Int()))
				ctx.W.EmitAddInt64(r168, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r168, d140.Reg)
			}
			r169 := ctx.AllocRegExcept(r168)
			ctx.W.EmitMovRegMem(r169, r168, 0)
			ctx.FreeReg(r168)
			d165 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r169}
			ctx.BindReg(r169, &d165)
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d139)
			var d166 scm.JITValueDesc
			if d139.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() % 64)}
			} else {
				r170 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r170, d139.Reg)
				ctx.W.EmitAndRegImm32(r170, 63)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d166)
			}
			if d166.Loc == scm.LocReg && d139.Loc == scm.LocReg && d166.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			d167 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d166)
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() - d166.Imm.Int())}
			} else if d166.Loc == scm.LocImm && d166.Imm.Int() == 0 {
				r171 := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(r171, d167.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d168)
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d167.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d166.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			} else if d166.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(scratch, d167.Reg)
				if d166.Imm.Int() >= -2147483648 && d166.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d166.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d166.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d168)
			} else {
				r172 := ctx.AllocRegExcept(d167.Reg, d166.Reg)
				ctx.W.EmitMovRegReg(r172, d167.Reg)
				ctx.W.EmitSubInt64(r172, d166.Reg)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d168)
			}
			if d168.Loc == scm.LocReg && d167.Loc == scm.LocReg && d168.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d168)
			var d169 scm.JITValueDesc
			if d165.Loc == scm.LocImm && d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d165.Imm.Int()) >> uint64(d168.Imm.Int())))}
			} else if d168.Loc == scm.LocImm {
				r173 := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegReg(r173, d165.Reg)
				ctx.W.EmitShrRegImm8(r173, uint8(d168.Imm.Int()))
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d169)
			} else {
				{
					shiftSrc := d165.Reg
					r174 := ctx.AllocRegExcept(d165.Reg)
					ctx.W.EmitMovRegReg(r174, d165.Reg)
					shiftSrc = r174
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d168.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d168.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d168.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d169)
				}
			}
			if d169.Loc == scm.LocReg && d165.Loc == scm.LocReg && d169.Reg == d165.Reg {
				ctx.TransferReg(d165.Reg)
				d165.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d165)
			ctx.FreeDesc(&d168)
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d144.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() | d169.Imm.Int())}
			} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
				ctx.BindReg(d169.Reg, &d170)
			} else if d169.Loc == scm.LocImm && d169.Imm.Int() == 0 {
				r175 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r175, d144.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d170)
			} else if d144.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d144.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else if d169.Loc == scm.LocImm {
				r176 := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(r176, d144.Reg)
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r176, int32(d169.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
					ctx.W.EmitOrInt64(r176, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d170)
			} else {
				r177 := ctx.AllocRegExcept(d144.Reg, d169.Reg)
				ctx.W.EmitMovRegReg(r177, d144.Reg)
				ctx.W.EmitOrInt64(r177, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d170)
			}
			if d170.Loc == scm.LocReg && d144.Loc == scm.LocReg && d170.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			d171 := d170
			if d171.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d171)
			ctx.EmitStoreToStack(d171, 24)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			d172 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
			ctx.BindReg(r160, &d172)
			ctx.BindReg(r160, &d172)
			if r140 { ctx.UnprotectReg(r141) }
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d172.Imm.Int()))))}
			} else {
				r178 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r178, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d173)
			}
			ctx.FreeDesc(&d172)
			var d174 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 32)
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r179, thisptr.Reg, off)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r179}
				ctx.BindReg(r179, &d174)
			}
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d174)
			var d175 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() + d174.Imm.Int())}
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				r180 := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(r180, d173.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d175)
			} else if d173.Loc == scm.LocImm && d173.Imm.Int() == 0 {
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d174.Reg}
				ctx.BindReg(d174.Reg, &d175)
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d175)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(scratch, d173.Reg)
				if d174.Imm.Int() >= -2147483648 && d174.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d174.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d174.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d175)
			} else {
				r181 := ctx.AllocRegExcept(d173.Reg, d174.Reg)
				ctx.W.EmitMovRegReg(r181, d173.Reg)
				ctx.W.EmitAddInt64(r181, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d175)
			}
			if d175.Loc == scm.LocReg && d173.Loc == scm.LocReg && d175.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			ctx.FreeDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d175.Imm.Int()))))}
			} else {
				r182 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r182, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d176)
			}
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).starts) + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).starts) + 56)
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r183, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
				ctx.BindReg(r183, &d177)
			}
			d178 := d177
			ctx.EnsureDesc(&d178)
			if d178.Loc != scm.LocImm && d178.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			if d178.Loc == scm.LocImm {
				if d178.Imm.Bool() {
					ctx.W.MarkLabel(lbl45)
					ctx.W.EmitJmp(lbl44)
				} else {
					ctx.W.MarkLabel(lbl46)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d178.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl45)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl45)
				ctx.W.EmitJmp(lbl44)
				ctx.W.MarkLabel(lbl46)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d177)
			bbs[4].RenderCount++
			bbpos_0_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&idxInt)
			d179 := idxInt
			_ = d179
			r184 := idxInt.Loc == scm.LocReg
			r185 := idxInt.Reg
			if r184 { ctx.ProtectReg(r185) }
			lbl47 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d179)
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d179.Imm.Int()))))}
			} else {
				r186 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r186, d179.Reg)
				ctx.W.EmitShlRegImm8(r186, 32)
				ctx.W.EmitShrRegImm8(r186, 32)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d180)
			}
			var d181 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r187, thisptr.Reg, off)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r187}
				ctx.BindReg(r187, &d181)
			}
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d181)
			var d182 scm.JITValueDesc
			if d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d181.Imm.Int()))))}
			} else {
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r188, d181.Reg)
				ctx.W.EmitShlRegImm8(r188, 56)
				ctx.W.EmitShrRegImm8(r188, 56)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r188}
				ctx.BindReg(r188, &d182)
			}
			ctx.FreeDesc(&d181)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if d180.Loc == scm.LocImm && d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() * d182.Imm.Int())}
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d180.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(scratch, d180.Reg)
				if d182.Imm.Int() >= -2147483648 && d182.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d182.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d182.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else {
				r189 := ctx.AllocRegExcept(d180.Reg, d182.Reg)
				ctx.W.EmitMovRegReg(r189, d180.Reg)
				ctx.W.EmitImulInt64(r189, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d183)
			}
			if d183.Loc == scm.LocReg && d180.Loc == scm.LocReg && d183.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			ctx.FreeDesc(&d182)
			var d184 scm.JITValueDesc
			r190 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r190, uint64(dataPtr))
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190, StackOff: int32(sliceLen)}
				ctx.BindReg(r190, &d184)
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 0)
				ctx.W.EmitMovRegMem(r190, thisptr.Reg, off)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190}
				ctx.BindReg(r190, &d184)
			}
			ctx.BindReg(r190, &d184)
			ctx.EnsureDesc(&d183)
			var d185 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() / 64)}
			} else {
				r191 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r191, d183.Reg)
				ctx.W.EmitShrRegImm8(r191, 6)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d185)
			}
			if d185.Loc == scm.LocReg && d183.Loc == scm.LocReg && d185.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d185)
			r192 := ctx.AllocReg()
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d184)
			if d185.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r192, uint64(d185.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r192, d185.Reg)
				ctx.W.EmitShlRegImm8(r192, 3)
			}
			if d184.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(r192, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r192, d184.Reg)
			}
			r193 := ctx.AllocRegExcept(r192)
			ctx.W.EmitMovRegMem(r193, r192, 0)
			ctx.FreeReg(r192)
			d186 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r193}
			ctx.BindReg(r193, &d186)
			ctx.FreeDesc(&d185)
			ctx.EnsureDesc(&d183)
			var d187 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r194 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r194, d183.Reg)
				ctx.W.EmitAndRegImm32(r194, 63)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r194}
				ctx.BindReg(r194, &d187)
			}
			if d187.Loc == scm.LocReg && d183.Loc == scm.LocReg && d187.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d187)
			var d188 scm.JITValueDesc
			if d186.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d186.Imm.Int()) << uint64(d187.Imm.Int())))}
			} else if d187.Loc == scm.LocImm {
				r195 := ctx.AllocRegExcept(d186.Reg)
				ctx.W.EmitMovRegReg(r195, d186.Reg)
				ctx.W.EmitShlRegImm8(r195, uint8(d187.Imm.Int()))
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d188)
			} else {
				{
					shiftSrc := d186.Reg
					r196 := ctx.AllocRegExcept(d186.Reg)
					ctx.W.EmitMovRegReg(r196, d186.Reg)
					shiftSrc = r196
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d187.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d187.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d187.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d188)
				}
			}
			if d188.Loc == scm.LocReg && d186.Loc == scm.LocReg && d188.Reg == d186.Reg {
				ctx.TransferReg(d186.Reg)
				d186.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d186)
			ctx.FreeDesc(&d187)
			var d189 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 25)
				r197 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r197, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d189)
			}
			d190 := d189
			ctx.EnsureDesc(&d190)
			if d190.Loc != scm.LocImm && d190.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			if d190.Loc == scm.LocImm {
				if d190.Imm.Bool() {
					ctx.W.MarkLabel(lbl50)
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.MarkLabel(lbl51)
			d191 := d188
			if d191.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d191)
			ctx.EmitStoreToStack(d191, 32)
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d190.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl51)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl51)
			d192 := d188
			if d192.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d192)
			ctx.EmitStoreToStack(d192, 32)
				ctx.W.EmitJmp(lbl49)
			}
			ctx.FreeDesc(&d189)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl49)
			ctx.W.ResolveFixups()
			d193 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d194 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r198, thisptr.Reg, off)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
				ctx.BindReg(r198, &d194)
			}
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d194)
			var d195 scm.JITValueDesc
			if d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d194.Imm.Int()))))}
			} else {
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r199, d194.Reg)
				ctx.W.EmitShlRegImm8(r199, 56)
				ctx.W.EmitShrRegImm8(r199, 56)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d195)
			}
			ctx.FreeDesc(&d194)
			d196 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d195)
			var d197 scm.JITValueDesc
			if d196.Loc == scm.LocImm && d195.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d196.Imm.Int() - d195.Imm.Int())}
			} else if d195.Loc == scm.LocImm && d195.Imm.Int() == 0 {
				r200 := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(r200, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d197)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d196.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d195.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else if d195.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(scratch, d196.Reg)
				if d195.Imm.Int() >= -2147483648 && d195.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d195.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d195.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else {
				r201 := ctx.AllocRegExcept(d196.Reg, d195.Reg)
				ctx.W.EmitMovRegReg(r201, d196.Reg)
				ctx.W.EmitSubInt64(r201, d195.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d197)
			}
			if d197.Loc == scm.LocReg && d196.Loc == scm.LocReg && d197.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d195)
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d197)
			var d198 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d193.Imm.Int()) >> uint64(d197.Imm.Int())))}
			} else if d197.Loc == scm.LocImm {
				r202 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r202, d193.Reg)
				ctx.W.EmitShrRegImm8(r202, uint8(d197.Imm.Int()))
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d198)
			} else {
				{
					shiftSrc := d193.Reg
					r203 := ctx.AllocRegExcept(d193.Reg)
					ctx.W.EmitMovRegReg(r203, d193.Reg)
					shiftSrc = r203
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d197.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d197.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d197.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d198)
				}
			}
			if d198.Loc == scm.LocReg && d193.Loc == scm.LocReg && d198.Reg == d193.Reg {
				ctx.TransferReg(d193.Reg)
				d193.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d193)
			ctx.FreeDesc(&d197)
			r204 := ctx.AllocReg()
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d198)
			if d198.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r204, d198)
			}
			ctx.W.EmitJmp(lbl47)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl48)
			ctx.W.ResolveFixups()
			d193 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d183)
			var d199 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r205 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r205, d183.Reg)
				ctx.W.EmitAndRegImm32(r205, 63)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d199)
			}
			if d199.Loc == scm.LocReg && d183.Loc == scm.LocReg && d199.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			var d200 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 24)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d200)
			}
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d200)
			var d201 scm.JITValueDesc
			if d200.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d200.Imm.Int()))))}
			} else {
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r207, d200.Reg)
				ctx.W.EmitShlRegImm8(r207, 56)
				ctx.W.EmitShrRegImm8(r207, 56)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d201)
			}
			ctx.FreeDesc(&d200)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d199.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d199.Imm.Int() + d201.Imm.Int())}
			} else if d201.Loc == scm.LocImm && d201.Imm.Int() == 0 {
				r208 := ctx.AllocRegExcept(d199.Reg)
				ctx.W.EmitMovRegReg(r208, d199.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d202)
			} else if d199.Loc == scm.LocImm && d199.Imm.Int() == 0 {
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d201.Reg}
				ctx.BindReg(d201.Reg, &d202)
			} else if d199.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d199.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d199.Reg)
				ctx.W.EmitMovRegReg(scratch, d199.Reg)
				if d201.Imm.Int() >= -2147483648 && d201.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d201.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d201.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d202)
			} else {
				r209 := ctx.AllocRegExcept(d199.Reg, d201.Reg)
				ctx.W.EmitMovRegReg(r209, d199.Reg)
				ctx.W.EmitAddInt64(r209, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d202)
			}
			if d202.Loc == scm.LocReg && d199.Loc == scm.LocReg && d202.Reg == d199.Reg {
				ctx.TransferReg(d199.Reg)
				d199.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d199)
			ctx.FreeDesc(&d201)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d202.Imm.Int()) > uint64(64))}
			} else {
				r210 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitCmpRegImm32(d202.Reg, 64)
				ctx.W.EmitSetcc(r210, scm.CcA)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r210}
				ctx.BindReg(r210, &d203)
			}
			ctx.FreeDesc(&d202)
			d204 := d203
			ctx.EnsureDesc(&d204)
			if d204.Loc != scm.LocImm && d204.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d204.Loc == scm.LocImm {
				if d204.Imm.Bool() {
					ctx.W.MarkLabel(lbl53)
					ctx.W.EmitJmp(lbl52)
				} else {
					ctx.W.MarkLabel(lbl54)
			d205 := d188
			if d205.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d205)
			ctx.EmitStoreToStack(d205, 32)
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d204.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl54)
			d206 := d188
			if d206.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d206)
			ctx.EmitStoreToStack(d206, 32)
				ctx.W.EmitJmp(lbl49)
			}
			ctx.FreeDesc(&d203)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl52)
			ctx.W.ResolveFixups()
			d193 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d183)
			var d207 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() / 64)}
			} else {
				r211 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r211, d183.Reg)
				ctx.W.EmitShrRegImm8(r211, 6)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d207)
			}
			if d207.Loc == scm.LocReg && d183.Loc == scm.LocReg && d207.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d207)
			ctx.EnsureDesc(&d207)
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(scratch, d207.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			}
			if d208.Loc == scm.LocReg && d207.Loc == scm.LocReg && d208.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d207)
			ctx.EnsureDesc(&d208)
			r212 := ctx.AllocReg()
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d184)
			if d208.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r212, uint64(d208.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r212, d208.Reg)
				ctx.W.EmitShlRegImm8(r212, 3)
			}
			if d184.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d184.Imm.Int()))
				ctx.W.EmitAddInt64(r212, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r212, d184.Reg)
			}
			r213 := ctx.AllocRegExcept(r212)
			ctx.W.EmitMovRegMem(r213, r212, 0)
			ctx.FreeReg(r212)
			d209 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
			ctx.BindReg(r213, &d209)
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d183)
			var d210 scm.JITValueDesc
			if d183.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() % 64)}
			} else {
				r214 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r214, d183.Reg)
				ctx.W.EmitAndRegImm32(r214, 63)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d210)
			}
			if d210.Loc == scm.LocReg && d183.Loc == scm.LocReg && d210.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d183)
			d211 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d210)
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm && d210.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d211.Imm.Int() - d210.Imm.Int())}
			} else if d210.Loc == scm.LocImm && d210.Imm.Int() == 0 {
				r215 := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegReg(r215, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d212)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d211.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d210.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else if d210.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegReg(scratch, d211.Reg)
				if d210.Imm.Int() >= -2147483648 && d210.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d210.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d210.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d212)
			} else {
				r216 := ctx.AllocRegExcept(d211.Reg, d210.Reg)
				ctx.W.EmitMovRegReg(r216, d211.Reg)
				ctx.W.EmitSubInt64(r216, d210.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d212)
			}
			if d212.Loc == scm.LocReg && d211.Loc == scm.LocReg && d212.Reg == d211.Reg {
				ctx.TransferReg(d211.Reg)
				d211.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d212)
			var d213 scm.JITValueDesc
			if d209.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d209.Imm.Int()) >> uint64(d212.Imm.Int())))}
			} else if d212.Loc == scm.LocImm {
				r217 := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegReg(r217, d209.Reg)
				ctx.W.EmitShrRegImm8(r217, uint8(d212.Imm.Int()))
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d213)
			} else {
				{
					shiftSrc := d209.Reg
					r218 := ctx.AllocRegExcept(d209.Reg)
					ctx.W.EmitMovRegReg(r218, d209.Reg)
					shiftSrc = r218
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d212.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d212.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d212.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d213)
				}
			}
			if d213.Loc == scm.LocReg && d209.Loc == scm.LocReg && d213.Reg == d209.Reg {
				ctx.TransferReg(d209.Reg)
				d209.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d209)
			ctx.FreeDesc(&d212)
			ctx.EnsureDesc(&d188)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() | d213.Imm.Int())}
			} else if d188.Loc == scm.LocImm && d188.Imm.Int() == 0 {
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d213.Reg}
				ctx.BindReg(d213.Reg, &d214)
			} else if d213.Loc == scm.LocImm && d213.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r219, d188.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d214)
			} else if d188.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d188.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else if d213.Loc == scm.LocImm {
				r220 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r220, d188.Reg)
				if d213.Imm.Int() >= -2147483648 && d213.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r220, int32(d213.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
					ctx.W.EmitOrInt64(r220, scm.RegR11)
				}
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d214)
			} else {
				r221 := ctx.AllocRegExcept(d188.Reg, d213.Reg)
				ctx.W.EmitMovRegReg(r221, d188.Reg)
				ctx.W.EmitOrInt64(r221, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d214)
			}
			if d214.Loc == scm.LocReg && d188.Loc == scm.LocReg && d214.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d213)
			d215 := d214
			if d215.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d215)
			ctx.EmitStoreToStack(d215, 32)
			ctx.W.EmitJmp(lbl49)
			ctx.W.MarkLabel(lbl47)
			d216 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
			ctx.BindReg(r204, &d216)
			ctx.BindReg(r204, &d216)
			if r184 { ctx.UnprotectReg(r185) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d216)
			var d217 scm.JITValueDesc
			if d216.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d216.Imm.Int()))))}
			} else {
				r222 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r222, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d217)
			}
			ctx.FreeDesc(&d216)
			var d218 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).lens) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).lens) + 32)
				r223 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r223, thisptr.Reg, off)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r223}
				ctx.BindReg(r223, &d218)
			}
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d218)
			var d219 scm.JITValueDesc
			if d217.Loc == scm.LocImm && d218.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d217.Imm.Int() + d218.Imm.Int())}
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegReg(r224, d217.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d219)
			} else if d217.Loc == scm.LocImm && d217.Imm.Int() == 0 {
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d218.Reg}
				ctx.BindReg(d218.Reg, &d219)
			} else if d217.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d217.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d218.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d219)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegReg(scratch, d217.Reg)
				if d218.Imm.Int() >= -2147483648 && d218.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d218.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d218.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d219)
			} else {
				r225 := ctx.AllocRegExcept(d217.Reg, d218.Reg)
				ctx.W.EmitMovRegReg(r225, d217.Reg)
				ctx.W.EmitAddInt64(r225, d218.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d219)
			}
			if d219.Loc == scm.LocReg && d217.Loc == scm.LocReg && d219.Reg == d217.Reg {
				ctx.TransferReg(d217.Reg)
				d217.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d217)
			ctx.FreeDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d219.Imm.Int()))))}
			} else {
				r226 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r226, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d220)
			}
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d221 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d176.Imm.Int()))))}
			} else {
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r227, d176.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d221)
			}
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d220)
			var d222 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d220.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() + d220.Imm.Int())}
			} else if d220.Loc == scm.LocImm && d220.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r228, d176.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d222)
			} else if d176.Loc == scm.LocImm && d176.Imm.Int() == 0 {
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d220.Reg}
				ctx.BindReg(d220.Reg, &d222)
			} else if d176.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d220.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d176.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d220.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else if d220.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(scratch, d176.Reg)
				if d220.Imm.Int() >= -2147483648 && d220.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d220.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d220.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else {
				r229 := ctx.AllocRegExcept(d176.Reg, d220.Reg)
				ctx.W.EmitMovRegReg(r229, d176.Reg)
				ctx.W.EmitAddInt64(r229, d220.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d222)
			}
			if d222.Loc == scm.LocReg && d176.Loc == scm.LocReg && d222.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d222)
			var d223 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d222.Imm.Int()))))}
			} else {
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r230, d222.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d223)
			}
			ctx.FreeDesc(&d222)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d223)
			r231 := ctx.AllocReg()
			r232 := ctx.AllocRegExcept(r231)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d223)
			if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r231, uint64(d131.Imm.Int()))
			} else if d131.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r231, d131.Reg)
			} else {
				ctx.W.EmitMovRegReg(r231, d131.Reg)
			}
			if d221.Loc == scm.LocImm {
				if d221.Imm.Int() != 0 {
					if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r231, int32(d221.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
						ctx.W.EmitAddInt64(r231, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r231, d221.Reg)
			}
			if d223.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r232, uint64(d223.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r232, d223.Reg)
			}
			if d221.Loc == scm.LocImm {
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r232, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitSubInt64(r232, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r232, d221.Reg)
			}
			d224 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r231, Reg2: r232}
			ctx.BindReg(r231, &d224)
			ctx.BindReg(r232, &d224)
			ctx.FreeDesc(&d221)
			ctx.FreeDesc(&d223)
			d225 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d225)
			ctx.BindReg(r1, &d225)
			d226 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d224}, 2)
			ctx.EmitMovPairToResult(&d226, &d225)
			ctx.W.EmitJmp(lbl0)
			bbs[8].RenderCount++
			bbpos_0_8 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl17)
			ctx.W.ResolveFixups()
			var d227 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageString)(nil).values) + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageString)(nil).values) + 64)
				r233 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r233, thisptr.Reg, off)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r233}
				ctx.BindReg(r233, &d227)
			}
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d227)
			var d228 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d227.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d227.Reg)
				ctx.W.EmitShlRegImm8(r234, 32)
				ctx.W.EmitShrRegImm8(r234, 32)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d228)
			}
			ctx.FreeDesc(&d227)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d228)
			var d229 scm.JITValueDesc
			if d43.Loc == scm.LocImm && d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d43.Imm.Int()) == uint64(d228.Imm.Int()))}
			} else if d228.Loc == scm.LocImm {
				r235 := ctx.AllocRegExcept(d43.Reg)
				if d228.Imm.Int() >= -2147483648 && d228.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d43.Reg, int32(d228.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d228.Imm.Int()))
					ctx.W.EmitCmpInt64(d43.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r235, scm.CcE)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r235}
				ctx.BindReg(r235, &d229)
			} else if d43.Loc == scm.LocImm {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d228.Reg)
				ctx.W.EmitSetcc(r236, scm.CcE)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r236}
				ctx.BindReg(r236, &d229)
			} else {
				r237 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitCmpInt64(d43.Reg, d228.Reg)
				ctx.W.EmitSetcc(r237, scm.CcE)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r237}
				ctx.BindReg(r237, &d229)
			}
			ctx.FreeDesc(&d43)
			ctx.FreeDesc(&d228)
			d230 := d229
			ctx.EnsureDesc(&d230)
			if d230.Loc != scm.LocImm && d230.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d230.Loc == scm.LocImm {
				if d230.Imm.Bool() {
					ctx.W.MarkLabel(lbl55)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl56)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d230.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl55)
				ctx.W.EmitJmp(lbl56)
				ctx.W.MarkLabel(lbl55)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d229)
			bbs[5].RenderCount++
			bbpos_0_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl44)
			ctx.W.ResolveFixups()
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
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d231)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d231)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d231)
			var d232 scm.JITValueDesc
			if d176.Loc == scm.LocImm && d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d176.Imm.Int()) == uint64(d231.Imm.Int()))}
			} else if d231.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d176.Reg)
				if d231.Imm.Int() >= -2147483648 && d231.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d176.Reg, int32(d231.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d231.Imm.Int()))
					ctx.W.EmitCmpInt64(d176.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d232)
			} else if d176.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d176.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d231.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d232)
			} else {
				r241 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitCmpInt64(d176.Reg, d231.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d232)
			}
			ctx.FreeDesc(&d176)
			ctx.FreeDesc(&d231)
			d233 := d232
			ctx.EnsureDesc(&d233)
			if d233.Loc != scm.LocImm && d233.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d233.Loc == scm.LocImm {
				if d233.Imm.Bool() {
					ctx.W.MarkLabel(lbl57)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.MarkLabel(lbl58)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d233.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl58)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d232)
			bbs[6].RenderCount++
			bbpos_0_6 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl3)
			ctx.W.ResolveFixups()
			d234 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d234)
			ctx.BindReg(r1, &d234)
			ctx.W.EmitMakeNil(d234)
			ctx.W.EmitJmp(lbl0)
			bbs[3].RenderCount++
			bbpos_0_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
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
