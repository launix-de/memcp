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
	chunk     []uint64 `jit:"immutable-after-finish"`
	bitsize   uint8    `jit:"immutable-after-finish"`
	crossWord bool     `jit:"immutable-after-finish"` // true when 64 % bitsize != 0 (cross-word spanning possible)
	offset    int64    `jit:"immutable-after-finish"`
	max       int64    // only of statistic use
	count     uint64   // only stored for serialization purposes
	hasNull   bool     `jit:"immutable-after-finish"`
	null      uint64   `jit:"immutable-after-finish"`
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
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&thisptr)
			ctx.EnsureDesc(&idxInt)
			d0 := idxInt
			_ = d0
			r2 := thisptr.Loc == scm.LocReg
			r3 := thisptr.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 := idxInt.Loc == scm.LocReg
			r5 := idxInt.Reg
			if r4 { ctx.ProtectReg(r5) }
			r6 := ctx.W.EmitSubRSP32Fixup()
			lbl3 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r7, d0.Reg)
				ctx.W.EmitShlRegImm8(r7, 32)
				ctx.W.EmitShrRegImm8(r7, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d1)
			}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r8, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
				ctx.BindReg(r8, &d2)
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
			} else {
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r9, d2.Reg)
				ctx.W.EmitShlRegImm8(r9, 56)
				ctx.W.EmitShrRegImm8(r9, 56)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d3)
			}
			ctx.FreeDesc(&d2)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d3)
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
				r10 := ctx.AllocRegExcept(d1.Reg, d3.Reg)
				ctx.W.EmitMovRegReg(r10, d1.Reg)
				ctx.W.EmitImulInt64(r10, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d4)
			}
			if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d3)
			var d5 scm.JITValueDesc
			r11 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r11, uint64(dataPtr))
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11, StackOff: int32(sliceLen)}
				ctx.BindReg(r11, &d5)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				ctx.W.EmitMovRegMem(r11, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
				ctx.BindReg(r11, &d5)
			}
			ctx.BindReg(r11, &d5)
			ctx.EnsureDesc(&d4)
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r12 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r12, d4.Reg)
				ctx.W.EmitShrRegImm8(r12, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d6)
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d6)
			r13 := ctx.AllocReg()
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d5)
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r13, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r13, d6.Reg)
				ctx.W.EmitShlRegImm8(r13, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r13, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r13, d5.Reg)
			}
			r14 := ctx.AllocRegExcept(r13)
			ctx.W.EmitMovRegMem(r14, r13, 0)
			ctx.FreeReg(r13)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			ctx.BindReg(r14, &d7)
			ctx.FreeDesc(&d6)
			ctx.EnsureDesc(&d4)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r15 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r15, d4.Reg)
				ctx.W.EmitAndRegImm32(r15, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d8)
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d8)
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) << uint64(d8.Imm.Int())))}
			} else if d8.Loc == scm.LocImm {
				r16 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r16, d7.Reg)
				ctx.W.EmitShlRegImm8(r16, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d9)
			} else {
				{
					shiftSrc := d7.Reg
					r17 := ctx.AllocRegExcept(d7.Reg)
					ctx.W.EmitMovRegReg(r17, d7.Reg)
					shiftSrc = r17
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).crossWord)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).crossWord))
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d10)
			}
			d11 := d10
			ctx.EnsureDesc(&d11)
			if d11.Loc != scm.LocImm && d11.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d11.Loc == scm.LocImm {
				if d11.Imm.Bool() {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.MarkLabel(lbl7)
			d12 := d9
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d12)
			ctx.EmitStoreToStack(d12, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d11.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl7)
			d13 := d9
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, 0)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d10)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d14 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d15 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d15)
			}
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d15)
			var d16 scm.JITValueDesc
			if d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d15.Imm.Int()))))}
			} else {
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r20, d15.Reg)
				ctx.W.EmitShlRegImm8(r20, 56)
				ctx.W.EmitShrRegImm8(r20, 56)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d16)
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
				r21 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r21, d17.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d18)
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
				r22 := ctx.AllocRegExcept(d17.Reg, d16.Reg)
				ctx.W.EmitMovRegReg(r22, d17.Reg)
				ctx.W.EmitSubInt64(r22, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d18)
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
				r23 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r23, d14.Reg)
				ctx.W.EmitShrRegImm8(r23, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d19)
			} else {
				{
					shiftSrc := d14.Reg
					r24 := ctx.AllocRegExcept(d14.Reg)
					ctx.W.EmitMovRegReg(r24, d14.Reg)
					shiftSrc = r24
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
			r25 := ctx.AllocReg()
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r25, d19)
			}
			ctx.W.EmitJmp(lbl3)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d20 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r26 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r26, d4.Reg)
				ctx.W.EmitAndRegImm32(r26, 63)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d20)
			}
			if d20.Loc == scm.LocReg && d4.Loc == scm.LocReg && d20.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r27, thisptr.Reg, off)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
				ctx.BindReg(r27, &d21)
			}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d21.Imm.Int()))))}
			} else {
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r28, d21.Reg)
				ctx.W.EmitShlRegImm8(r28, 56)
				ctx.W.EmitShrRegImm8(r28, 56)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d22)
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
				r29 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(r29, d20.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d23)
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
				r30 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.W.EmitMovRegReg(r30, d20.Reg)
				ctx.W.EmitAddInt64(r30, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d23)
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
				r31 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitCmpRegImm32(d23.Reg, 64)
				ctx.W.EmitSetcc(r31, scm.CcA)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
				ctx.BindReg(r31, &d24)
			}
			ctx.FreeDesc(&d23)
			d25 := d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != scm.LocImm && d25.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d25.Loc == scm.LocImm {
				if d25.Imm.Bool() {
					ctx.W.MarkLabel(lbl9)
					ctx.W.EmitJmp(lbl8)
				} else {
					ctx.W.MarkLabel(lbl10)
			d26 := d9
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, 0)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl9)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl9)
				ctx.W.EmitJmp(lbl8)
				ctx.W.MarkLabel(lbl10)
			d27 := d9
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, 0)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d24)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl8)
			ctx.W.ResolveFixups()
			d14 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			var d28 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r32 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r32, d4.Reg)
				ctx.W.EmitShrRegImm8(r32, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d28)
			}
			if d28.Loc == scm.LocReg && d4.Loc == scm.LocReg && d28.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
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
			r33 := ctx.AllocReg()
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d5)
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r33, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r33, d29.Reg)
				ctx.W.EmitShlRegImm8(r33, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r33, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r33, d5.Reg)
			}
			r34 := ctx.AllocRegExcept(r33)
			ctx.W.EmitMovRegMem(r34, r33, 0)
			ctx.FreeReg(r33)
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d30)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d4)
			var d31 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r35 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r35, d4.Reg)
				ctx.W.EmitAndRegImm32(r35, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d31)
			}
			if d31.Loc == scm.LocReg && d4.Loc == scm.LocReg && d31.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
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
				r36 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r36, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d33)
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
				r37 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.W.EmitMovRegReg(r37, d32.Reg)
				ctx.W.EmitSubInt64(r37, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d33)
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
				r38 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r38, d30.Reg)
				ctx.W.EmitShrRegImm8(r38, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r39 := ctx.AllocRegExcept(d30.Reg)
					ctx.W.EmitMovRegReg(r39, d30.Reg)
					shiftSrc = r39
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
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d34.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r41 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r41, d9.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r41, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitOrInt64(r41, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d35)
			} else {
				r42 := ctx.AllocRegExcept(d9.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r42, d9.Reg)
				ctx.W.EmitOrInt64(r42, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d35)
			}
			if d35.Loc == scm.LocReg && d9.Loc == scm.LocReg && d35.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			d36 := d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 0)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl3)
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d37)
			ctx.BindReg(r25, &d37)
			if r2 { ctx.UnprotectReg(r3) }
			if r4 { ctx.UnprotectReg(r5) }
			ctx.FreeDesc(&idxInt)
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).hasNull)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).hasNull))
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r43, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			d39 := d38
			ctx.EnsureDesc(&d39)
			if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocImm {
				if d39.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.MarkLabel(lbl13)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d39.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl11)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d38)
			bbpos_0_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d40 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d37.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d40)
			}
			var d41 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d41)
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
				r46 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r46, d40.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d42)
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
				r47 := ctx.AllocRegExcept(d40.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r47, d40.Reg)
				ctx.W.EmitAddInt64(r47, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d42)
			}
			if d42.Loc == scm.LocReg && d40.Loc == scm.LocReg && d42.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d42)
			d43 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d43)
			ctx.BindReg(r1, &d43)
			ctx.EnsureDesc(&d42)
			ctx.W.EmitMakeInt(d43, d42)
			if d42.Loc == scm.LocReg { ctx.FreeReg(d42.Reg) }
			ctx.W.EmitJmp(lbl0)
			bbpos_0_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl11)
			ctx.W.ResolveFixups()
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r48, thisptr.Reg, off)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) == uint64(d44.Imm.Int()))}
			} else if d44.Loc == scm.LocImm {
				r49 := ctx.AllocRegExcept(d37.Reg)
				if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d37.Reg, int32(d44.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
					ctx.W.EmitCmpInt64(d37.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r49, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
				ctx.BindReg(r49, &d45)
			} else if d37.Loc == scm.LocImm {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d44.Reg)
				ctx.W.EmitSetcc(r50, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
				ctx.BindReg(r50, &d45)
			} else {
				r51 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitCmpInt64(d37.Reg, d44.Reg)
				ctx.W.EmitSetcc(r51, scm.CcE)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
				ctx.BindReg(r51, &d45)
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d44)
			d46 := d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d45)
			bbpos_0_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d47 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d47)
			ctx.BindReg(r1, &d47)
			ctx.W.EmitMakeNil(d47)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d48 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d48)
			ctx.BindReg(r1, &d48)
			ctx.EmitMovPairToResult(&d48, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r6, int32(8))
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
