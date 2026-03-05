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
			var d0 scm.JITValueDesc
			_ = d0
			var r6 unsafe.Pointer
			_ = r6
			var d1 scm.JITValueDesc
			_ = d1
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
			var d6 scm.JITValueDesc
			_ = d6
			var d7 scm.JITValueDesc
			_ = d7
			var d8 scm.JITValueDesc
			_ = d8
			var d9 scm.JITValueDesc
			_ = d9
			var d10 scm.JITValueDesc
			_ = d10
			var d11 scm.JITValueDesc
			_ = d11
			var d12 scm.JITValueDesc
			_ = d12
			var d13 scm.JITValueDesc
			_ = d13
			var d14 scm.JITValueDesc
			_ = d14
			var d15 scm.JITValueDesc
			_ = d15
			var d16 scm.JITValueDesc
			_ = d16
			var d17 scm.JITValueDesc
			_ = d17
			var d18 scm.JITValueDesc
			_ = d18
			var d19 scm.JITValueDesc
			_ = d19
			var d20 scm.JITValueDesc
			_ = d20
			var d21 scm.JITValueDesc
			_ = d21
			var d22 scm.JITValueDesc
			_ = d22
			var d23 scm.JITValueDesc
			_ = d23
			var d24 scm.JITValueDesc
			_ = d24
			var d25 scm.JITValueDesc
			_ = d25
			var d26 scm.JITValueDesc
			_ = d26
			var d27 scm.JITValueDesc
			_ = d27
			var d28 scm.JITValueDesc
			_ = d28
			var d29 scm.JITValueDesc
			_ = d29
			var d30 scm.JITValueDesc
			_ = d30
			var d31 scm.JITValueDesc
			_ = d31
			var d32 scm.JITValueDesc
			_ = d32
			var d33 scm.JITValueDesc
			_ = d33
			var d34 scm.JITValueDesc
			_ = d34
			var d35 scm.JITValueDesc
			_ = d35
			var d36 scm.JITValueDesc
			_ = d36
			var d37 scm.JITValueDesc
			_ = d37
			var d38 scm.JITValueDesc
			_ = d38
			var d39 scm.JITValueDesc
			_ = d39
			var d85 scm.JITValueDesc
			_ = d85
			var d86 scm.JITValueDesc
			_ = d86
			var d87 scm.JITValueDesc
			_ = d87
			var d88 scm.JITValueDesc
			_ = d88
			var d89 scm.JITValueDesc
			_ = d89
			var d90 scm.JITValueDesc
			_ = d90
			var d91 scm.JITValueDesc
			_ = d91
			var d92 scm.JITValueDesc
			_ = d92
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
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
			var bbs [4]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&thisptr)
			ctx.EnsureDesc(&idxInt)
			d0 = idxInt
			_ = d0
			r2 := thisptr.Loc == scm.LocReg
			r3 := thisptr.Reg
			if r2 { ctx.ProtectReg(r3) }
			r4 := idxInt.Loc == scm.LocReg
			r5 := idxInt.Reg
			if r4 { ctx.ProtectReg(r5) }
			r6 = ctx.EmitSubRSP32Fixup()
			_ = r6
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			lbl5 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d2 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d0.Imm.Int()))))}
			} else {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegReg(r7, d0.Reg)
				ctx.EmitShlRegImm8(r7, 32)
				ctx.EmitShrRegImm8(r7, 32)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d2)
			}
			var d3 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r8 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r8, thisptr.Reg, off)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r8}
				ctx.BindReg(r8, &d3)
			}
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d3.Imm.Int()))))}
			} else {
				r9 := ctx.AllocReg()
				ctx.EmitMovRegReg(r9, d3.Reg)
				ctx.EmitShlRegImm8(r9, 56)
				ctx.EmitShrRegImm8(r9, 56)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d4)
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
				ctx.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.EmitImulInt64(scratch, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.EmitMovRegReg(scratch, d2.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d4.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d5)
			} else {
				r10 := ctx.AllocRegExcept(d2.Reg, d4.Reg)
				ctx.EmitMovRegReg(r10, d2.Reg)
				ctx.EmitImulInt64(r10, d4.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d5)
			}
			if d5.Loc == scm.LocReg && d2.Loc == scm.LocReg && d5.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			ctx.FreeDesc(&d4)
			var d6 scm.JITValueDesc
			r11 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).chunk)
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r11, uint64(dataPtr))
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11, StackOff: int32(sliceLen)}
				ctx.BindReg(r11, &d6)
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).chunk))
				ctx.EmitMovRegMem(r11, thisptr.Reg, off)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r11}
				ctx.BindReg(r11, &d6)
			}
			ctx.BindReg(r11, &d6)
			ctx.EnsureDesc(&d5)
			var d7 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r12 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r12, d5.Reg)
				ctx.EmitShrRegImm8(r12, 6)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d7)
			}
			if d7.Loc == scm.LocReg && d5.Loc == scm.LocReg && d7.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			r13 := ctx.AllocReg()
			ctx.EnsureDesc(&d7)
			ctx.EnsureDesc(&d6)
			if d7.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r13, uint64(d7.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r13, d7.Reg)
				ctx.EmitShlRegImm8(r13, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r13, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r13, d6.Reg)
			}
			r14 := ctx.AllocRegExcept(r13)
			ctx.EmitMovRegMem(r14, r13, 0)
			ctx.FreeReg(r13)
			d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			ctx.BindReg(r14, &d8)
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d5)
			var d9 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r15 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r15, d5.Reg)
				ctx.EmitAndRegImm32(r15, 63)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d9)
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
				r16 := ctx.AllocRegExcept(d8.Reg)
				ctx.EmitMovRegReg(r16, d8.Reg)
				ctx.EmitShlRegImm8(r16, uint8(d9.Imm.Int()))
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d10)
			} else {
				{
					shiftSrc := d8.Reg
					r17 := ctx.AllocRegExcept(d8.Reg)
					ctx.EmitMovRegReg(r17, d8.Reg)
					shiftSrc = r17
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d9.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d9.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d9.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).crossWord)
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).crossWord))
				r18 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r18, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != scm.LocImm && d12.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl6 := ctx.ReserveLabel()
			lbl7 := ctx.ReserveLabel()
			lbl8 := ctx.ReserveLabel()
			lbl9 := ctx.ReserveLabel()
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl6)
				} else {
					ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d13 = d10
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d13)
			ctx.EmitStoreToStack(d13, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
					ctx.EmitJmp(lbl7)
				}
			} else {
				ctx.EmitCmpRegImm32(d12.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl8)
				ctx.EmitJmp(lbl9)
				ctx.MarkLabel(lbl8)
				ctx.EmitJmp(lbl6)
				ctx.MarkLabel(lbl9)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d14 = d10
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d14)
			ctx.EmitStoreToStack(d14, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
				ctx.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d11)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl7)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			var d15 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r19 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r19, thisptr.Reg, off)
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
				ctx.EmitMovRegReg(r20, d15.Reg)
				ctx.EmitShlRegImm8(r20, 56)
				ctx.EmitShrRegImm8(r20, 56)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d16)
			}
			ctx.FreeDesc(&d15)
			d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
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
				ctx.EmitMovRegReg(r21, d17.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d18)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.EmitSubInt64(scratch, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.EmitMovRegReg(scratch, d17.Reg)
				if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d16.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d16.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d18)
			} else {
				r22 := ctx.AllocRegExcept(d17.Reg, d16.Reg)
				ctx.EmitMovRegReg(r22, d17.Reg)
				ctx.EmitSubInt64(r22, d16.Reg)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d18)
			}
			if d18.Loc == scm.LocReg && d17.Loc == scm.LocReg && d18.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d1.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d1.Imm.Int()) >> uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegReg(r23, d1.Reg)
				ctx.EmitShrRegImm8(r23, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d19)
			} else {
				{
					shiftSrc := d1.Reg
					r24 := ctx.AllocRegExcept(d1.Reg)
					ctx.EmitMovRegReg(r24, d1.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d1.Loc == scm.LocReg && d19.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d18)
			r25 := ctx.AllocReg()
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d19)
			if d19.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r25, d19)
			}
			ctx.EmitJmp(lbl5)
			bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl6)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d20 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r26 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r26, d5.Reg)
				ctx.EmitAndRegImm32(r26, 63)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d20)
			}
			if d20.Loc == scm.LocReg && d5.Loc == scm.LocReg && d20.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			var d21 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).bitsize)
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).bitsize))
				r27 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r27, thisptr.Reg, off)
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
				ctx.EmitMovRegReg(r28, d21.Reg)
				ctx.EmitShlRegImm8(r28, 56)
				ctx.EmitShrRegImm8(r28, 56)
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
				ctx.EmitMovRegReg(r29, d20.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d23)
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
				ctx.BindReg(d22.Reg, &d23)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d22.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.EmitAddInt64(scratch, d22.Reg)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else if d22.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.EmitMovRegReg(scratch, d20.Reg)
				if d22.Imm.Int() >= -2147483648 && d22.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d22.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d22.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d23)
			} else {
				r30 := ctx.AllocRegExcept(d20.Reg, d22.Reg)
				ctx.EmitMovRegReg(r30, d20.Reg)
				ctx.EmitAddInt64(r30, d22.Reg)
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
				ctx.EmitCmpRegImm32(d23.Reg, 64)
				ctx.EmitSetcc(r31, scm.CcA)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
				ctx.BindReg(r31, &d24)
			}
			ctx.FreeDesc(&d23)
			d25 = d24
			ctx.EnsureDesc(&d25)
			if d25.Loc != scm.LocImm && d25.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl10 := ctx.ReserveLabel()
			lbl11 := ctx.ReserveLabel()
			lbl12 := ctx.ReserveLabel()
			if d25.Loc == scm.LocImm {
				if d25.Imm.Bool() {
					ctx.MarkLabel(lbl11)
					ctx.EmitJmp(lbl10)
				} else {
					ctx.MarkLabel(lbl12)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d26 = d10
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d26)
			ctx.EmitStoreToStack(d26, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
					ctx.EmitJmp(lbl7)
				}
			} else {
				ctx.EmitCmpRegImm32(d25.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl11)
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl11)
				ctx.EmitJmp(lbl10)
				ctx.MarkLabel(lbl12)
			ctx.EnsureDesc(&d10)
			if d10.Loc == scm.LocReg {
				ctx.ProtectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.ProtectReg(d10.Reg)
				ctx.ProtectReg(d10.Reg2)
			}
			d27 = d10
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d27)
			ctx.EmitStoreToStack(d27, int32(bbs[2].PhiBase)+int32(0))
			if d10.Loc == scm.LocReg {
				ctx.UnprotectReg(d10.Reg)
			} else if d10.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d10.Reg)
				ctx.UnprotectReg(d10.Reg2)
			}
				ctx.EmitJmp(lbl7)
			}
			ctx.FreeDesc(&d24)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl10)
			ctx.ResolveFixups()
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			ctx.EnsureDesc(&d5)
			var d28 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 64)}
			} else {
				r32 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r32, d5.Reg)
				ctx.EmitShrRegImm8(r32, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d28)
			}
			if d28.Loc == scm.LocReg && d5.Loc == scm.LocReg && d28.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.EmitMovRegReg(scratch, d28.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
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
			ctx.EnsureDesc(&d6)
			if d29.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r33, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r33, d29.Reg)
				ctx.EmitShlRegImm8(r33, 3)
			}
			if d6.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
				ctx.EmitAddInt64(r33, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r33, d6.Reg)
			}
			r34 := ctx.AllocRegExcept(r33)
			ctx.EmitMovRegMem(r34, r33, 0)
			ctx.FreeReg(r33)
			d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d30)
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d5)
			var d31 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() % 64)}
			} else {
				r35 := ctx.AllocRegExcept(d5.Reg)
				ctx.EmitMovRegReg(r35, d5.Reg)
				ctx.EmitAndRegImm32(r35, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d31)
			}
			if d31.Loc == scm.LocReg && d5.Loc == scm.LocReg && d31.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
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
				ctx.EmitMovRegReg(r36, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.EmitMovRegReg(scratch, d32.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r37 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.EmitMovRegReg(r37, d32.Reg)
				ctx.EmitSubInt64(r37, d31.Reg)
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
				ctx.EmitMovRegReg(r38, d30.Reg)
				ctx.EmitShrRegImm8(r38, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r39 := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegReg(r39, d30.Reg)
					shiftSrc = r39
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d33.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d33.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d33.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
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
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() | d34.Imm.Int())}
			} else if d10.Loc == scm.LocImm && d10.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r40 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r40, d10.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d35)
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
				ctx.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r41 := ctx.AllocRegExcept(d10.Reg)
				ctx.EmitMovRegReg(r41, d10.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r41, int32(d34.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.EmitOrInt64(r41, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d35)
			} else {
				r42 := ctx.AllocRegExcept(d10.Reg, d34.Reg)
				ctx.EmitMovRegReg(r42, d10.Reg)
				ctx.EmitOrInt64(r42, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d35)
			}
			if d35.Loc == scm.LocReg && d10.Loc == scm.LocReg && d35.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d35)
			if d35.Loc == scm.LocReg {
				ctx.ProtectReg(d35.Reg)
			} else if d35.Loc == scm.LocRegPair {
				ctx.ProtectReg(d35.Reg)
				ctx.ProtectReg(d35.Reg2)
			}
			d36 = d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, int32(bbs[2].PhiBase)+int32(0))
			if d35.Loc == scm.LocReg {
				ctx.UnprotectReg(d35.Reg)
			} else if d35.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d35.Reg)
				ctx.UnprotectReg(d35.Reg2)
			}
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl5)
			d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
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
				ctx.EmitMovRegMemB(r43, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			d39 = d38
			ctx.EnsureDesc(&d39)
			if d39.Loc != scm.LocImm && d39.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d39.Loc == scm.LocImm {
				if d39.Imm.Bool() {
			ps40 := scm.PhiState{General: ps.General}
			ps40.OverlayValues = make([]scm.JITValueDesc, 40)
			ps40.OverlayValues[0] = d0
			ps40.OverlayValues[1] = d1
			ps40.OverlayValues[2] = d2
			ps40.OverlayValues[3] = d3
			ps40.OverlayValues[4] = d4
			ps40.OverlayValues[5] = d5
			ps40.OverlayValues[6] = d6
			ps40.OverlayValues[7] = d7
			ps40.OverlayValues[8] = d8
			ps40.OverlayValues[9] = d9
			ps40.OverlayValues[10] = d10
			ps40.OverlayValues[11] = d11
			ps40.OverlayValues[12] = d12
			ps40.OverlayValues[13] = d13
			ps40.OverlayValues[14] = d14
			ps40.OverlayValues[15] = d15
			ps40.OverlayValues[16] = d16
			ps40.OverlayValues[17] = d17
			ps40.OverlayValues[18] = d18
			ps40.OverlayValues[19] = d19
			ps40.OverlayValues[20] = d20
			ps40.OverlayValues[21] = d21
			ps40.OverlayValues[22] = d22
			ps40.OverlayValues[23] = d23
			ps40.OverlayValues[24] = d24
			ps40.OverlayValues[25] = d25
			ps40.OverlayValues[26] = d26
			ps40.OverlayValues[27] = d27
			ps40.OverlayValues[28] = d28
			ps40.OverlayValues[29] = d29
			ps40.OverlayValues[30] = d30
			ps40.OverlayValues[31] = d31
			ps40.OverlayValues[32] = d32
			ps40.OverlayValues[33] = d33
			ps40.OverlayValues[34] = d34
			ps40.OverlayValues[35] = d35
			ps40.OverlayValues[36] = d36
			ps40.OverlayValues[37] = d37
			ps40.OverlayValues[38] = d38
			ps40.OverlayValues[39] = d39
					return bbs[3].RenderPS(ps40)
				}
			ps41 := scm.PhiState{General: ps.General}
			ps41.OverlayValues = make([]scm.JITValueDesc, 40)
			ps41.OverlayValues[0] = d0
			ps41.OverlayValues[1] = d1
			ps41.OverlayValues[2] = d2
			ps41.OverlayValues[3] = d3
			ps41.OverlayValues[4] = d4
			ps41.OverlayValues[5] = d5
			ps41.OverlayValues[6] = d6
			ps41.OverlayValues[7] = d7
			ps41.OverlayValues[8] = d8
			ps41.OverlayValues[9] = d9
			ps41.OverlayValues[10] = d10
			ps41.OverlayValues[11] = d11
			ps41.OverlayValues[12] = d12
			ps41.OverlayValues[13] = d13
			ps41.OverlayValues[14] = d14
			ps41.OverlayValues[15] = d15
			ps41.OverlayValues[16] = d16
			ps41.OverlayValues[17] = d17
			ps41.OverlayValues[18] = d18
			ps41.OverlayValues[19] = d19
			ps41.OverlayValues[20] = d20
			ps41.OverlayValues[21] = d21
			ps41.OverlayValues[22] = d22
			ps41.OverlayValues[23] = d23
			ps41.OverlayValues[24] = d24
			ps41.OverlayValues[25] = d25
			ps41.OverlayValues[26] = d26
			ps41.OverlayValues[27] = d27
			ps41.OverlayValues[28] = d28
			ps41.OverlayValues[29] = d29
			ps41.OverlayValues[30] = d30
			ps41.OverlayValues[31] = d31
			ps41.OverlayValues[32] = d32
			ps41.OverlayValues[33] = d33
			ps41.OverlayValues[34] = d34
			ps41.OverlayValues[35] = d35
			ps41.OverlayValues[36] = d36
			ps41.OverlayValues[37] = d37
			ps41.OverlayValues[38] = d38
			ps41.OverlayValues[39] = d39
				return bbs[2].RenderPS(ps41)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d39.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl13)
			ctx.EmitJmp(lbl14)
			ctx.MarkLabel(lbl13)
			ctx.EmitJmp(lbl4)
			ctx.MarkLabel(lbl14)
			ctx.EmitJmp(lbl3)
			ps42 := scm.PhiState{General: true}
			ps42.OverlayValues = make([]scm.JITValueDesc, 40)
			ps42.OverlayValues[0] = d0
			ps42.OverlayValues[1] = d1
			ps42.OverlayValues[2] = d2
			ps42.OverlayValues[3] = d3
			ps42.OverlayValues[4] = d4
			ps42.OverlayValues[5] = d5
			ps42.OverlayValues[6] = d6
			ps42.OverlayValues[7] = d7
			ps42.OverlayValues[8] = d8
			ps42.OverlayValues[9] = d9
			ps42.OverlayValues[10] = d10
			ps42.OverlayValues[11] = d11
			ps42.OverlayValues[12] = d12
			ps42.OverlayValues[13] = d13
			ps42.OverlayValues[14] = d14
			ps42.OverlayValues[15] = d15
			ps42.OverlayValues[16] = d16
			ps42.OverlayValues[17] = d17
			ps42.OverlayValues[18] = d18
			ps42.OverlayValues[19] = d19
			ps42.OverlayValues[20] = d20
			ps42.OverlayValues[21] = d21
			ps42.OverlayValues[22] = d22
			ps42.OverlayValues[23] = d23
			ps42.OverlayValues[24] = d24
			ps42.OverlayValues[25] = d25
			ps42.OverlayValues[26] = d26
			ps42.OverlayValues[27] = d27
			ps42.OverlayValues[28] = d28
			ps42.OverlayValues[29] = d29
			ps42.OverlayValues[30] = d30
			ps42.OverlayValues[31] = d31
			ps42.OverlayValues[32] = d32
			ps42.OverlayValues[33] = d33
			ps42.OverlayValues[34] = d34
			ps42.OverlayValues[35] = d35
			ps42.OverlayValues[36] = d36
			ps42.OverlayValues[37] = d37
			ps42.OverlayValues[38] = d38
			ps42.OverlayValues[39] = d39
			ps43 := scm.PhiState{General: true}
			ps43.OverlayValues = make([]scm.JITValueDesc, 40)
			ps43.OverlayValues[0] = d0
			ps43.OverlayValues[1] = d1
			ps43.OverlayValues[2] = d2
			ps43.OverlayValues[3] = d3
			ps43.OverlayValues[4] = d4
			ps43.OverlayValues[5] = d5
			ps43.OverlayValues[6] = d6
			ps43.OverlayValues[7] = d7
			ps43.OverlayValues[8] = d8
			ps43.OverlayValues[9] = d9
			ps43.OverlayValues[10] = d10
			ps43.OverlayValues[11] = d11
			ps43.OverlayValues[12] = d12
			ps43.OverlayValues[13] = d13
			ps43.OverlayValues[14] = d14
			ps43.OverlayValues[15] = d15
			ps43.OverlayValues[16] = d16
			ps43.OverlayValues[17] = d17
			ps43.OverlayValues[18] = d18
			ps43.OverlayValues[19] = d19
			ps43.OverlayValues[20] = d20
			ps43.OverlayValues[21] = d21
			ps43.OverlayValues[22] = d22
			ps43.OverlayValues[23] = d23
			ps43.OverlayValues[24] = d24
			ps43.OverlayValues[25] = d25
			ps43.OverlayValues[26] = d26
			ps43.OverlayValues[27] = d27
			ps43.OverlayValues[28] = d28
			ps43.OverlayValues[29] = d29
			ps43.OverlayValues[30] = d30
			ps43.OverlayValues[31] = d31
			ps43.OverlayValues[32] = d32
			ps43.OverlayValues[33] = d33
			ps43.OverlayValues[34] = d34
			ps43.OverlayValues[35] = d35
			ps43.OverlayValues[36] = d36
			ps43.OverlayValues[37] = d37
			ps43.OverlayValues[38] = d38
			ps43.OverlayValues[39] = d39
			snap44 := d0
			snap45 := d1
			snap46 := d2
			snap47 := d3
			snap48 := d4
			snap49 := d5
			snap50 := d6
			snap51 := d7
			snap52 := d8
			snap53 := d9
			snap54 := d10
			snap55 := d11
			snap56 := d12
			snap57 := d13
			snap58 := d14
			snap59 := d15
			snap60 := d16
			snap61 := d17
			snap62 := d18
			snap63 := d19
			snap64 := d20
			snap65 := d21
			snap66 := d22
			snap67 := d23
			snap68 := d24
			snap69 := d25
			snap70 := d26
			snap71 := d27
			snap72 := d28
			snap73 := d29
			snap74 := d30
			snap75 := d31
			snap76 := d32
			snap77 := d33
			snap78 := d34
			snap79 := d35
			snap80 := d36
			snap81 := d37
			snap82 := d38
			snap83 := d39
			alloc84 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps43)
			}
			ctx.RestoreAllocState(alloc84)
			d0 = snap44
			d1 = snap45
			d2 = snap46
			d3 = snap47
			d4 = snap48
			d5 = snap49
			d6 = snap50
			d7 = snap51
			d8 = snap52
			d9 = snap53
			d10 = snap54
			d11 = snap55
			d12 = snap56
			d13 = snap57
			d14 = snap58
			d15 = snap59
			d16 = snap60
			d17 = snap61
			d18 = snap62
			d19 = snap63
			d20 = snap64
			d21 = snap65
			d22 = snap66
			d23 = snap67
			d24 = snap68
			d25 = snap69
			d26 = snap70
			d27 = snap71
			d28 = snap72
			d29 = snap73
			d30 = snap74
			d31 = snap75
			d32 = snap76
			d33 = snap77
			d34 = snap78
			d35 = snap79
			d36 = snap80
			d37 = snap81
			d38 = snap82
			d39 = snap83
			if !bbs[3].Rendered {
				return bbs[3].RenderPS(ps42)
			}
			return result
			ctx.FreeDesc(&d38)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			ctx.ReclaimUntrackedRegs()
			d85 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d85)
			ctx.BindReg(r1, &d85)
			ctx.EmitMakeNil(d85)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d86 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.EmitMovRegReg(r44, d37.Reg)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d86)
			}
			var d87 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).offset)
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).offset))
				r45 := ctx.AllocReg()
				ctx.EmitMovRegMem(r45, thisptr.Reg, off)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d87)
			}
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d87.Imm.Int())}
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegReg(r46, d86.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d88)
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
				ctx.BindReg(d87.Reg, &d88)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.EmitAddInt64(scratch, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.EmitMovRegReg(scratch, d86.Reg)
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d87.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else {
				r47 := ctx.AllocRegExcept(d86.Reg, d87.Reg)
				ctx.EmitMovRegReg(r47, d86.Reg)
				ctx.EmitAddInt64(r47, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d88)
			}
			if d88.Loc == scm.LocReg && d86.Loc == scm.LocReg && d88.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d87)
			ctx.EnsureDesc(&d88)
			d89 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d89)
			ctx.BindReg(r1, &d89)
			ctx.EnsureDesc(&d88)
			ctx.EmitMakeInt(d89, d88)
			if d88.Loc == scm.LocReg { ctx.FreeReg(d88.Reg) }
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
					ps.General = true
					return bbs[3].RenderPS(ps)
				}
			}
			bbs[3].VisitCount++
			if ps.General {
				if bbs[3].Rendered {
					ctx.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != scm.LocNone {
				d6 = ps.OverlayValues[6]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			if len(ps.OverlayValues) > 14 && ps.OverlayValues[14].Loc != scm.LocNone {
				d14 = ps.OverlayValues[14]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != scm.LocNone {
				d17 = ps.OverlayValues[17]
			}
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != scm.LocNone {
				d88 = ps.OverlayValues[88]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			ctx.ReclaimUntrackedRegs()
			var d90 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageInt)(nil).null)
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageInt)(nil).null))
				r48 := ctx.AllocReg()
				ctx.EmitMovRegMem(r48, thisptr.Reg, off)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d90)
			}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d37.Imm.Int()) == uint64(d90.Imm.Int()))}
			} else if d90.Loc == scm.LocImm {
				r49 := ctx.AllocRegExcept(d37.Reg)
				if d90.Imm.Int() >= -2147483648 && d90.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d37.Reg, int32(d90.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d90.Imm.Int()))
					ctx.EmitCmpInt64(d37.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r49, scm.CcE)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r49}
				ctx.BindReg(r49, &d91)
			} else if d37.Loc == scm.LocImm {
				r50 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d90.Reg)
				ctx.EmitSetcc(r50, scm.CcE)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r50}
				ctx.BindReg(r50, &d91)
			} else {
				r51 := ctx.AllocRegExcept(d37.Reg)
				ctx.EmitCmpInt64(d37.Reg, d90.Reg)
				ctx.EmitSetcc(r51, scm.CcE)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r51}
				ctx.BindReg(r51, &d91)
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d90)
			d92 = d91
			ctx.EnsureDesc(&d92)
			if d92.Loc != scm.LocImm && d92.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d92.Loc == scm.LocImm {
				if d92.Imm.Bool() {
			ps93 := scm.PhiState{General: ps.General}
			ps93.OverlayValues = make([]scm.JITValueDesc, 93)
			ps93.OverlayValues[0] = d0
			ps93.OverlayValues[1] = d1
			ps93.OverlayValues[2] = d2
			ps93.OverlayValues[3] = d3
			ps93.OverlayValues[4] = d4
			ps93.OverlayValues[5] = d5
			ps93.OverlayValues[6] = d6
			ps93.OverlayValues[7] = d7
			ps93.OverlayValues[8] = d8
			ps93.OverlayValues[9] = d9
			ps93.OverlayValues[10] = d10
			ps93.OverlayValues[11] = d11
			ps93.OverlayValues[12] = d12
			ps93.OverlayValues[13] = d13
			ps93.OverlayValues[14] = d14
			ps93.OverlayValues[15] = d15
			ps93.OverlayValues[16] = d16
			ps93.OverlayValues[17] = d17
			ps93.OverlayValues[18] = d18
			ps93.OverlayValues[19] = d19
			ps93.OverlayValues[20] = d20
			ps93.OverlayValues[21] = d21
			ps93.OverlayValues[22] = d22
			ps93.OverlayValues[23] = d23
			ps93.OverlayValues[24] = d24
			ps93.OverlayValues[25] = d25
			ps93.OverlayValues[26] = d26
			ps93.OverlayValues[27] = d27
			ps93.OverlayValues[28] = d28
			ps93.OverlayValues[29] = d29
			ps93.OverlayValues[30] = d30
			ps93.OverlayValues[31] = d31
			ps93.OverlayValues[32] = d32
			ps93.OverlayValues[33] = d33
			ps93.OverlayValues[34] = d34
			ps93.OverlayValues[35] = d35
			ps93.OverlayValues[36] = d36
			ps93.OverlayValues[37] = d37
			ps93.OverlayValues[38] = d38
			ps93.OverlayValues[39] = d39
			ps93.OverlayValues[85] = d85
			ps93.OverlayValues[86] = d86
			ps93.OverlayValues[87] = d87
			ps93.OverlayValues[88] = d88
			ps93.OverlayValues[89] = d89
			ps93.OverlayValues[90] = d90
			ps93.OverlayValues[91] = d91
			ps93.OverlayValues[92] = d92
					return bbs[1].RenderPS(ps93)
				}
			ps94 := scm.PhiState{General: ps.General}
			ps94.OverlayValues = make([]scm.JITValueDesc, 93)
			ps94.OverlayValues[0] = d0
			ps94.OverlayValues[1] = d1
			ps94.OverlayValues[2] = d2
			ps94.OverlayValues[3] = d3
			ps94.OverlayValues[4] = d4
			ps94.OverlayValues[5] = d5
			ps94.OverlayValues[6] = d6
			ps94.OverlayValues[7] = d7
			ps94.OverlayValues[8] = d8
			ps94.OverlayValues[9] = d9
			ps94.OverlayValues[10] = d10
			ps94.OverlayValues[11] = d11
			ps94.OverlayValues[12] = d12
			ps94.OverlayValues[13] = d13
			ps94.OverlayValues[14] = d14
			ps94.OverlayValues[15] = d15
			ps94.OverlayValues[16] = d16
			ps94.OverlayValues[17] = d17
			ps94.OverlayValues[18] = d18
			ps94.OverlayValues[19] = d19
			ps94.OverlayValues[20] = d20
			ps94.OverlayValues[21] = d21
			ps94.OverlayValues[22] = d22
			ps94.OverlayValues[23] = d23
			ps94.OverlayValues[24] = d24
			ps94.OverlayValues[25] = d25
			ps94.OverlayValues[26] = d26
			ps94.OverlayValues[27] = d27
			ps94.OverlayValues[28] = d28
			ps94.OverlayValues[29] = d29
			ps94.OverlayValues[30] = d30
			ps94.OverlayValues[31] = d31
			ps94.OverlayValues[32] = d32
			ps94.OverlayValues[33] = d33
			ps94.OverlayValues[34] = d34
			ps94.OverlayValues[35] = d35
			ps94.OverlayValues[36] = d36
			ps94.OverlayValues[37] = d37
			ps94.OverlayValues[38] = d38
			ps94.OverlayValues[39] = d39
			ps94.OverlayValues[85] = d85
			ps94.OverlayValues[86] = d86
			ps94.OverlayValues[87] = d87
			ps94.OverlayValues[88] = d88
			ps94.OverlayValues[89] = d89
			ps94.OverlayValues[90] = d90
			ps94.OverlayValues[91] = d91
			ps94.OverlayValues[92] = d92
				return bbs[2].RenderPS(ps94)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl15 := ctx.ReserveLabel()
			lbl16 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d92.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl15)
			ctx.EmitJmp(lbl16)
			ctx.MarkLabel(lbl15)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl16)
			ctx.EmitJmp(lbl3)
			ps95 := scm.PhiState{General: true}
			ps95.OverlayValues = make([]scm.JITValueDesc, 93)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[6] = d6
			ps95.OverlayValues[7] = d7
			ps95.OverlayValues[8] = d8
			ps95.OverlayValues[9] = d9
			ps95.OverlayValues[10] = d10
			ps95.OverlayValues[11] = d11
			ps95.OverlayValues[12] = d12
			ps95.OverlayValues[13] = d13
			ps95.OverlayValues[14] = d14
			ps95.OverlayValues[15] = d15
			ps95.OverlayValues[16] = d16
			ps95.OverlayValues[17] = d17
			ps95.OverlayValues[18] = d18
			ps95.OverlayValues[19] = d19
			ps95.OverlayValues[20] = d20
			ps95.OverlayValues[21] = d21
			ps95.OverlayValues[22] = d22
			ps95.OverlayValues[23] = d23
			ps95.OverlayValues[24] = d24
			ps95.OverlayValues[25] = d25
			ps95.OverlayValues[26] = d26
			ps95.OverlayValues[27] = d27
			ps95.OverlayValues[28] = d28
			ps95.OverlayValues[29] = d29
			ps95.OverlayValues[30] = d30
			ps95.OverlayValues[31] = d31
			ps95.OverlayValues[32] = d32
			ps95.OverlayValues[33] = d33
			ps95.OverlayValues[34] = d34
			ps95.OverlayValues[35] = d35
			ps95.OverlayValues[36] = d36
			ps95.OverlayValues[37] = d37
			ps95.OverlayValues[38] = d38
			ps95.OverlayValues[39] = d39
			ps95.OverlayValues[85] = d85
			ps95.OverlayValues[86] = d86
			ps95.OverlayValues[87] = d87
			ps95.OverlayValues[88] = d88
			ps95.OverlayValues[89] = d89
			ps95.OverlayValues[90] = d90
			ps95.OverlayValues[91] = d91
			ps95.OverlayValues[92] = d92
			ps96 := scm.PhiState{General: true}
			ps96.OverlayValues = make([]scm.JITValueDesc, 93)
			ps96.OverlayValues[0] = d0
			ps96.OverlayValues[1] = d1
			ps96.OverlayValues[2] = d2
			ps96.OverlayValues[3] = d3
			ps96.OverlayValues[4] = d4
			ps96.OverlayValues[5] = d5
			ps96.OverlayValues[6] = d6
			ps96.OverlayValues[7] = d7
			ps96.OverlayValues[8] = d8
			ps96.OverlayValues[9] = d9
			ps96.OverlayValues[10] = d10
			ps96.OverlayValues[11] = d11
			ps96.OverlayValues[12] = d12
			ps96.OverlayValues[13] = d13
			ps96.OverlayValues[14] = d14
			ps96.OverlayValues[15] = d15
			ps96.OverlayValues[16] = d16
			ps96.OverlayValues[17] = d17
			ps96.OverlayValues[18] = d18
			ps96.OverlayValues[19] = d19
			ps96.OverlayValues[20] = d20
			ps96.OverlayValues[21] = d21
			ps96.OverlayValues[22] = d22
			ps96.OverlayValues[23] = d23
			ps96.OverlayValues[24] = d24
			ps96.OverlayValues[25] = d25
			ps96.OverlayValues[26] = d26
			ps96.OverlayValues[27] = d27
			ps96.OverlayValues[28] = d28
			ps96.OverlayValues[29] = d29
			ps96.OverlayValues[30] = d30
			ps96.OverlayValues[31] = d31
			ps96.OverlayValues[32] = d32
			ps96.OverlayValues[33] = d33
			ps96.OverlayValues[34] = d34
			ps96.OverlayValues[35] = d35
			ps96.OverlayValues[36] = d36
			ps96.OverlayValues[37] = d37
			ps96.OverlayValues[38] = d38
			ps96.OverlayValues[39] = d39
			ps96.OverlayValues[85] = d85
			ps96.OverlayValues[86] = d86
			ps96.OverlayValues[87] = d87
			ps96.OverlayValues[88] = d88
			ps96.OverlayValues[89] = d89
			ps96.OverlayValues[90] = d90
			ps96.OverlayValues[91] = d91
			ps96.OverlayValues[92] = d92
			snap97 := d0
			snap98 := d1
			snap99 := d2
			snap100 := d3
			snap101 := d4
			snap102 := d5
			snap103 := d6
			snap104 := d7
			snap105 := d8
			snap106 := d9
			snap107 := d10
			snap108 := d11
			snap109 := d12
			snap110 := d13
			snap111 := d14
			snap112 := d15
			snap113 := d16
			snap114 := d17
			snap115 := d18
			snap116 := d19
			snap117 := d20
			snap118 := d21
			snap119 := d22
			snap120 := d23
			snap121 := d24
			snap122 := d25
			snap123 := d26
			snap124 := d27
			snap125 := d28
			snap126 := d29
			snap127 := d30
			snap128 := d31
			snap129 := d32
			snap130 := d33
			snap131 := d34
			snap132 := d35
			snap133 := d36
			snap134 := d37
			snap135 := d38
			snap136 := d39
			snap137 := d85
			snap138 := d86
			snap139 := d87
			snap140 := d88
			snap141 := d89
			snap142 := d90
			snap143 := d91
			snap144 := d92
			alloc145 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps96)
			}
			ctx.RestoreAllocState(alloc145)
			d0 = snap97
			d1 = snap98
			d2 = snap99
			d3 = snap100
			d4 = snap101
			d5 = snap102
			d6 = snap103
			d7 = snap104
			d8 = snap105
			d9 = snap106
			d10 = snap107
			d11 = snap108
			d12 = snap109
			d13 = snap110
			d14 = snap111
			d15 = snap112
			d16 = snap113
			d17 = snap114
			d18 = snap115
			d19 = snap116
			d20 = snap117
			d21 = snap118
			d22 = snap119
			d23 = snap120
			d24 = snap121
			d25 = snap122
			d26 = snap123
			d27 = snap124
			d28 = snap125
			d29 = snap126
			d30 = snap127
			d31 = snap128
			d32 = snap129
			d33 = snap130
			d34 = snap131
			d35 = snap132
			d36 = snap133
			d37 = snap134
			d38 = snap135
			d39 = snap136
			d85 = snap137
			d86 = snap138
			d87 = snap139
			d88 = snap140
			d89 = snap141
			d90 = snap142
			d91 = snap143
			d92 = snap144
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps95)
			}
			return result
			ctx.FreeDesc(&d91)
			return result
			}
			ps146 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps146)
			ctx.MarkLabel(lbl0)
			d147 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d147)
			ctx.BindReg(r1, &d147)
			ctx.EmitMovPairToResult(&d147, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r6, int32(16))
			ctx.EmitAddRSP32(int32(16))
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
