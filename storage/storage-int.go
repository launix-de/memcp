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
	chunk      []uint64 `jit:"immutable-after-finish"`
	bitsize    uint8    `jit:"immutable-after-finish"`
	crossWord  bool     `jit:"immutable-after-finish"` // true when 64 % bitsize != 0 (cross-word spanning possible)
	offset     int64    `jit:"immutable-after-finish"`
	max        int64    // only of statistic use
	count      uint64   // only stored for serialization purposes
	hasNull    bool     `jit:"immutable-after-finish"`
	null       uint64   `jit:"immutable-after-finish"`
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
	s.crossWord = s.bitsize > 0 && 64%uint(s.bitsize) != 0
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
			if thisptr.Loc == scm.LocStack || thisptr.Loc == scm.LocStackPair { ctx.EnsureDesc(&thisptr) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r0 := thisptr.Loc == scm.LocReg
			r1 := thisptr.Reg
			if r0 { ctx.ProtectReg(r1) }
			r2 := ctx.W.EmitSubRSP32Fixup()
			lbl1 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d0 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r3, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r3, 32)
				ctx.W.EmitShrRegImm8(r3, 32)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
				ctx.BindReg(r3, &d0)
			}
			ctx.FreeDesc(&idxInt)
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r4, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d1.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r5, d1.Reg)
				ctx.W.EmitShlRegImm8(r5, 56)
				ctx.W.EmitShrRegImm8(r5, 56)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d2)
			}
			ctx.FreeDesc(&d1)
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() * d2.Imm.Int())}
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d3)
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d3)
			} else {
				r6 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r6, d0.Reg)
				ctx.W.EmitImulInt64(r6, d2.Reg)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d3)
			}
			if d3.Loc == scm.LocReg && d0.Loc == scm.LocReg && d3.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d0)
			ctx.FreeDesc(&d2)
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r7, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d4)
			}
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r8 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r8, d3.Reg)
				ctx.W.EmitShrRegImm8(r8, 6)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			if d5.Loc == scm.LocReg && d3.Loc == scm.LocReg && d5.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			r9 := ctx.AllocReg()
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r9, uint64(d5.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r9, d5.Reg)
				ctx.W.EmitShlRegImm8(r9, 3)
			}
			if d4.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(r9, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r9, d4.Reg)
			}
			r10 := ctx.AllocRegExcept(r9)
			ctx.W.EmitMovRegMem(r10, r9, 0)
			ctx.FreeReg(r9)
			d6 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			ctx.BindReg(r10, &d6)
			ctx.FreeDesc(&d5)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d7 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r11 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r11, d3.Reg)
				ctx.W.EmitAndRegImm32(r11, 63)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d7)
			}
			if d7.Loc == scm.LocReg && d3.Loc == scm.LocReg && d7.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) << uint64(d7.Imm.Int())))}
			} else if d7.Loc == scm.LocImm {
				r12 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r12, d6.Reg)
				ctx.W.EmitShlRegImm8(r12, uint8(d7.Imm.Int()))
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d8)
			} else {
				{
					shiftSrc := d6.Reg
					r13 := ctx.AllocRegExcept(d6.Reg)
					ctx.W.EmitMovRegReg(r13, d6.Reg)
					shiftSrc = r13
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d7.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d7.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d7.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d8)
				}
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			ctx.FreeDesc(&d7)
			var d9 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).crossWord)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).crossWord))
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r14, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
				ctx.BindReg(r14, &d9)
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d9.Loc == scm.LocImm {
				if d9.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d9.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
			ctx.EmitStoreToStack(d8, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d9)
			ctx.W.MarkLabel(lbl3)
			r15 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r15, 0)
			ctx.ProtectReg(r15)
			d10 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r15}
			ctx.BindReg(r15, &d10)
			ctx.UnprotectReg(r15)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r16, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d11)
			}
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d12 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d11.Imm.Int()))))}
			} else {
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r17, d11.Reg)
				ctx.W.EmitShlRegImm8(r17, 56)
				ctx.W.EmitShrRegImm8(r17, 56)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d12)
			}
			ctx.FreeDesc(&d11)
			d13 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d14 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() - d12.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				r18 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r18, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
				ctx.BindReg(r18, &d14)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d13.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(scratch, d13.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d12.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r19 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r19, d13.Reg)
				ctx.W.EmitSubInt64(r19, d12.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d14)
			}
			if d14.Loc == scm.LocReg && d13.Loc == scm.LocReg && d14.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			var d15 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) >> uint64(d14.Imm.Int())))}
			} else if d14.Loc == scm.LocImm {
				r20 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r20, d10.Reg)
				ctx.W.EmitShrRegImm8(r20, uint8(d14.Imm.Int()))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d15)
			} else {
				{
					shiftSrc := d10.Reg
					r21 := ctx.AllocRegExcept(d10.Reg)
					ctx.W.EmitMovRegReg(r21, d10.Reg)
					shiftSrc = r21
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d14.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d14.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d14.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d15)
				}
			}
			if d15.Loc == scm.LocReg && d10.Loc == scm.LocReg && d15.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d14)
			r22 := ctx.AllocReg()
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			ctx.EmitMovToReg(r22, d15)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl2)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d16 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r23 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r23, d3.Reg)
				ctx.W.EmitAndRegImm32(r23, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d16)
			}
			if d16.Loc == scm.LocReg && d3.Loc == scm.LocReg && d16.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d17)
			}
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d17.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d17.Reg)
				ctx.W.EmitShlRegImm8(r25, 56)
				ctx.W.EmitShrRegImm8(r25, 56)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d18)
			}
			ctx.FreeDesc(&d17)
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			var d19 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() + d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r26, d16.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d19)
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
				ctx.BindReg(d18.Reg, &d19)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d18.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else {
				r27 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r27, d16.Reg)
				ctx.W.EmitAddInt64(r27, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d19)
			}
			if d19.Loc == scm.LocReg && d16.Loc == scm.LocReg && d19.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.FreeDesc(&d18)
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d19.Imm.Int()) > uint64(64))}
			} else {
				r28 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitCmpRegImm32(d19.Reg, 64)
				ctx.W.EmitSetcc(r28, scm.CcA)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r28}
				ctx.BindReg(r28, &d20)
			}
			ctx.FreeDesc(&d19)
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d20.Loc == scm.LocImm {
				if d20.Imm.Bool() {
					ctx.W.EmitJmp(lbl5)
				} else {
			ctx.EmitStoreToStack(d8, 0)
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d20.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
			ctx.EmitStoreToStack(d8, 0)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl5)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d21 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() / 64)}
			} else {
				r29 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r29, d3.Reg)
				ctx.W.EmitShrRegImm8(r29, 6)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d21)
			}
			if d21.Loc == scm.LocReg && d3.Loc == scm.LocReg && d21.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			}
			if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			r30 := ctx.AllocReg()
			if d22.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r30, uint64(d22.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r30, d22.Reg)
				ctx.W.EmitShlRegImm8(r30, 3)
			}
			if d4.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(r30, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r30, d4.Reg)
			}
			r31 := ctx.AllocRegExcept(r30)
			ctx.W.EmitMovRegMem(r31, r30, 0)
			ctx.FreeReg(r30)
			d23 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r31}
			ctx.BindReg(r31, &d23)
			ctx.FreeDesc(&d22)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			var d24 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() % 64)}
			} else {
				r32 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r32, d3.Reg)
				ctx.W.EmitAndRegImm32(r32, 63)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d24)
			}
			if d24.Loc == scm.LocReg && d3.Loc == scm.LocReg && d24.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			d25 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				r33 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r33, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(scratch, d25.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r34 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r34, d25.Reg)
				ctx.W.EmitSubInt64(r34, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d26)
			}
			if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			var d27 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d26.Imm.Int())))}
			} else if d26.Loc == scm.LocImm {
				r35 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r35, d23.Reg)
				ctx.W.EmitShrRegImm8(r35, uint8(d26.Imm.Int()))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d27)
			} else {
				{
					shiftSrc := d23.Reg
					r36 := ctx.AllocRegExcept(d23.Reg)
					ctx.W.EmitMovRegReg(r36, d23.Reg)
					shiftSrc = r36
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d26.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d26.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d26.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d27)
				}
			}
			if d27.Loc == scm.LocReg && d23.Loc == scm.LocReg && d27.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d26)
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			var d28 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() | d27.Imm.Int())}
			} else if d8.Loc == scm.LocImm && d8.Imm.Int() == 0 {
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d27.Reg}
				ctx.BindReg(d27.Reg, &d28)
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r37, d8.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d28)
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d27.Loc == scm.LocImm {
				r38 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r38, d8.Reg)
				if d27.Imm.Int() >= -2147483648 && d27.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r38, int32(d27.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
					ctx.W.EmitOrInt64(r38, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d28)
			} else {
				r39 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r39, d8.Reg)
				ctx.W.EmitOrInt64(r39, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d28)
			}
			if d28.Loc == scm.LocReg && d8.Loc == scm.LocReg && d28.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.EmitStoreToStack(d28, 0)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d29 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
			ctx.BindReg(r22, &d29)
			ctx.BindReg(r22, &d29)
			if r0 { ctx.UnprotectReg(r1) }
			ctx.FreeDesc(&idxInt)
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
				r40 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r40, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r40}
				ctx.BindReg(r40, &d30)
			}
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			if d30.Loc == scm.LocImm {
				if d30.Imm.Bool() {
					ctx.W.EmitJmp(lbl7)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d30.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d30)
			ctx.W.MarkLabel(lbl8)
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			var d31 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d29.Imm.Int()))))}
			} else {
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r41, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d31)
			}
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
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
			ctx.W.EmitMakeInt(result, d33)
			if d33.Loc == scm.LocReg { ctx.FreeReg(d33.Reg) }
			result.Type = scm.TagInt
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl7)
			var d34 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d34)
			}
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d35 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d29.Imm.Int()) == uint64(d34.Imm.Int()))}
			} else if d34.Loc == scm.LocImm {
				r46 := ctx.AllocRegExcept(d29.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d29.Reg, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitCmpInt64(d29.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r46, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
				ctx.BindReg(r46, &d35)
			} else if d29.Loc == scm.LocImm {
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d34.Reg)
				ctx.W.EmitSetcc(r47, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r47}
				ctx.BindReg(r47, &d35)
			} else {
				r48 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitCmpInt64(d29.Reg, d34.Reg)
				ctx.W.EmitSetcc(r48, scm.CcE)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r48}
				ctx.BindReg(r48, &d35)
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d34)
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.EmitJmp(lbl8)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r2, int32(8))
			ctx.W.EmitAddRSP32(int32(8))
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
	if s.crossWord && bitpos%64+uint(s.bitsize) > 64 {
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
	s.crossWord = 64%uint(s.bitsize) != 0
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
