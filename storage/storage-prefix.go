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

import "fmt"
import "strings"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StoragePrefix struct {
	// prefix compression
	prefixes         StorageInt
	prefixdictionary []string      // pref
	values           StorageString // only one depth (but can be cascaded!)
}

func (s *StoragePrefix) ComputeSize() uint {
	return s.prefixes.ComputeSize() + 24 + s.values.ComputeSize()
}

func (s *StoragePrefix) String() string {
	return fmt.Sprintf("prefix[%s]-%s", s.prefixdictionary[1], s.values.String())
}

func (s *StoragePrefix) GetCachedReader() ColumnReader { return s }

func (s *StoragePrefix) GetValue(i uint32) scm.Scmer {
	inner := s.values.GetValue(i)
	if inner.IsNil() {
		return scm.NewNil()
	}
	if !inner.IsString() {
		panic("invalid value in prefix storage")
	}
	idx := int64(s.prefixes.GetValueUInt(i)) + s.prefixes.offset
	if idx >= int64(len(s.prefixdictionary)) || idx < 0 {
		panic("prefix index out of range")
	}
	prefix := s.prefixdictionary[idx]
	return scm.NewString(prefix + inner.String())
}
func (s *StoragePrefix) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r0 := idxInt.Loc == scm.LocReg
			r1 := idxInt.Reg
			if r0 { ctx.ProtectReg(r1) }
			lbl1 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 232
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 232)
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r2, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d0)
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d0.Loc == scm.LocImm {
				if d0.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d0.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d0)
			ctx.W.MarkLabel(lbl3)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r3 := idxInt.Loc == scm.LocReg
			r4 := idxInt.Reg
			if r3 { ctx.ProtectReg(r4) }
			r5 := ctx.W.EmitSubRSP32Fixup()
			lbl5 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d1 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r6 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r6, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r6, 32)
				ctx.W.EmitShrRegImm8(r6, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r6}
				ctx.BindReg(r6, &d1)
			}
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r7, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
				ctx.BindReg(r7, &d2)
			}
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d2.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d2.Reg)
				ctx.W.EmitShlRegImm8(r8, 56)
				ctx.W.EmitShrRegImm8(r8, 56)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d3)
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
				r9 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegReg(r9, d1.Reg)
				ctx.W.EmitImulInt64(r9, d3.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d4)
			}
			if d4.Loc == scm.LocReg && d1.Loc == scm.LocReg && d4.Reg == d1.Reg {
				ctx.TransferReg(d1.Reg)
				d1.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d1)
			ctx.FreeDesc(&d3)
			var d5 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d5)
			}
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r11 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r11, d4.Reg)
				ctx.W.EmitShrRegImm8(r11, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d6)
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			r12 := ctx.AllocReg()
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r12, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r12, d6.Reg)
				ctx.W.EmitShlRegImm8(r12, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r12, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r12, d5.Reg)
			}
			r13 := ctx.AllocRegExcept(r12)
			ctx.W.EmitMovRegMem(r13, r12, 0)
			ctx.FreeReg(r12)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			ctx.BindReg(r13, &d7)
			ctx.FreeDesc(&d6)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r14 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r14, d4.Reg)
				ctx.W.EmitAndRegImm32(r14, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d8)
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
				r15 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r15, d7.Reg)
				ctx.W.EmitShlRegImm8(r15, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d9)
			} else {
				{
					shiftSrc := d7.Reg
					r16 := ctx.AllocRegExcept(d7.Reg)
					ctx.W.EmitMovRegReg(r16, d7.Reg)
					shiftSrc = r16
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25)
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r17, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
				ctx.BindReg(r17, &d10)
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d10.Loc == scm.LocImm {
				if d10.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			ctx.EmitStoreToStack(d9, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl7)
			r18 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r18, 0)
			ctx.ProtectReg(r18)
			d11 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r18}
			ctx.BindReg(r18, &d11)
			ctx.UnprotectReg(r18)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d12)
			}
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
			} else {
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r20, d12.Reg)
				ctx.W.EmitShlRegImm8(r20, 56)
				ctx.W.EmitShrRegImm8(r20, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d13)
			}
			ctx.FreeDesc(&d12)
			d14 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d13.Imm.Int())}
			} else if d13.Loc == scm.LocImm && d13.Imm.Int() == 0 {
				r21 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r21, d14.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d15)
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
				r22 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r22, d14.Reg)
				ctx.W.EmitSubInt64(r22, d13.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d15)
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
				r23 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r23, d11.Reg)
				ctx.W.EmitShrRegImm8(r23, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d16)
			} else {
				{
					shiftSrc := d11.Reg
					r24 := ctx.AllocRegExcept(d11.Reg)
					ctx.W.EmitMovRegReg(r24, d11.Reg)
					shiftSrc = r24
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
			r25 := ctx.AllocReg()
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			ctx.EmitMovToReg(r25, d16)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d17 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r26 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r26, d4.Reg)
				ctx.W.EmitAndRegImm32(r26, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d17)
			}
			if d17.Loc == scm.LocReg && d4.Loc == scm.LocReg && d17.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r27, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
				ctx.BindReg(r27, &d18)
			}
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r28, d18.Reg)
				ctx.W.EmitShlRegImm8(r28, 56)
				ctx.W.EmitShrRegImm8(r28, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d19)
			}
			ctx.FreeDesc(&d18)
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r29 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r29, d17.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d20)
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
				r30 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r30, d17.Reg)
				ctx.W.EmitAddInt64(r30, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d20)
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
				r31 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitCmpRegImm32(d20.Reg, 64)
				ctx.W.EmitSetcc(r31, scm.CcA)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
				ctx.BindReg(r31, &d21)
			}
			ctx.FreeDesc(&d20)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d21.Loc == scm.LocImm {
				if d21.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			ctx.EmitStoreToStack(d9, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			ctx.EmitStoreToStack(d9, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d21)
			ctx.W.MarkLabel(lbl9)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r32 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r32, d4.Reg)
				ctx.W.EmitShrRegImm8(r32, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r32}
				ctx.BindReg(r32, &d22)
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
			r33 := ctx.AllocReg()
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r33, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r33, d23.Reg)
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
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			ctx.BindReg(r34, &d24)
			ctx.FreeDesc(&d23)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d25 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r35 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r35, d4.Reg)
				ctx.W.EmitAndRegImm32(r35, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d25)
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
				r36 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r36, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d27)
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
				r37 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r37, d26.Reg)
				ctx.W.EmitSubInt64(r37, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d27)
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
				r38 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r38, d24.Reg)
				ctx.W.EmitShrRegImm8(r38, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d28)
			} else {
				{
					shiftSrc := d24.Reg
					r39 := ctx.AllocRegExcept(d24.Reg)
					ctx.W.EmitMovRegReg(r39, d24.Reg)
					shiftSrc = r39
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
				r40 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d29)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d29)
			} else if d28.Loc == scm.LocImm {
				r41 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r41, d9.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r41, int32(d28.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d28.Imm.Int()))
					ctx.W.EmitOrInt64(r41, scm.RegR11)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d29)
			} else {
				r42 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r42, d9.Reg)
				ctx.W.EmitOrInt64(r42, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d29)
			}
			if d29.Loc == scm.LocReg && d9.Loc == scm.LocReg && d29.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.EmitStoreToStack(d29, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.BindReg(r25, &d30)
			ctx.BindReg(r25, &d30)
			if r3 { ctx.UnprotectReg(r4) }
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d30.Imm.Int()))))}
			} else {
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r43, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d31)
			}
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r44, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r44}
				ctx.BindReg(r44, &d32)
			}
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d33 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r45 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r45, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d33)
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
				r46 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r46, d31.Reg)
				ctx.W.EmitAddInt64(r46, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d33)
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
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r47, d33.Reg)
				ctx.W.EmitShlRegImm8(r47, 32)
				ctx.W.EmitShrRegImm8(r47, 32)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d34)
			}
			ctx.FreeDesc(&d33)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r48, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d35)
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d35)
			ctx.W.MarkLabel(lbl2)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r49 := idxInt.Loc == scm.LocReg
			r50 := idxInt.Reg
			if r49 { ctx.ProtectReg(r50) }
			lbl14 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d36 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r51, 32)
				ctx.W.EmitShrRegImm8(r51, 32)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d36)
			}
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r52, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r52}
				ctx.BindReg(r52, &d37)
			}
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d37.Imm.Int()))))}
			} else {
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r53, d37.Reg)
				ctx.W.EmitShlRegImm8(r53, 56)
				ctx.W.EmitShrRegImm8(r53, 56)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d38)
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
				r54 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r54, d36.Reg)
				ctx.W.EmitImulInt64(r54, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d39)
			}
			if d39.Loc == scm.LocReg && d36.Loc == scm.LocReg && d39.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r55, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r55}
				ctx.BindReg(r55, &d40)
			}
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r56 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r56, d39.Reg)
				ctx.W.EmitShrRegImm8(r56, 6)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
				ctx.BindReg(r56, &d41)
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			r57 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r57, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r57, d41.Reg)
				ctx.W.EmitShlRegImm8(r57, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r57, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r57, d40.Reg)
			}
			r58 := ctx.AllocRegExcept(r57)
			ctx.W.EmitMovRegMem(r58, r57, 0)
			ctx.FreeReg(r57)
			d42 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
			ctx.BindReg(r58, &d42)
			ctx.FreeDesc(&d41)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r59 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r59, d39.Reg)
				ctx.W.EmitAndRegImm32(r59, 63)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d43)
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
				r60 := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegReg(r60, d42.Reg)
				ctx.W.EmitShlRegImm8(r60, uint8(d43.Imm.Int()))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d44)
			} else {
				{
					shiftSrc := d42.Reg
					r61 := ctx.AllocRegExcept(d42.Reg)
					ctx.W.EmitMovRegReg(r61, d42.Reg)
					shiftSrc = r61
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d45)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d45.Loc == scm.LocImm {
				if d45.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
			ctx.EmitStoreToStack(d44, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d45.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
			ctx.EmitStoreToStack(d44, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl16)
			r63 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r63, 8)
			ctx.ProtectReg(r63)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r63}
			ctx.BindReg(r63, &d46)
			ctx.UnprotectReg(r63)
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d47)
			}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d47.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d47.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d48)
			}
			ctx.FreeDesc(&d47)
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d48.Imm.Int())}
			} else if d48.Loc == scm.LocImm && d48.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r66, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d50)
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
				r67 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r67, d49.Reg)
				ctx.W.EmitSubInt64(r67, d48.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d50)
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
				r68 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(r68, d46.Reg)
				ctx.W.EmitShrRegImm8(r68, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d51)
			} else {
				{
					shiftSrc := d46.Reg
					r69 := ctx.AllocRegExcept(d46.Reg)
					ctx.W.EmitMovRegReg(r69, d46.Reg)
					shiftSrc = r69
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
			r70 := ctx.AllocReg()
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitMovToReg(r70, d51)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl15)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d52 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r71, d39.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d52)
			}
			if d52.Loc == scm.LocReg && d39.Loc == scm.LocReg && d52.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d53 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d53)
			}
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			var d54 scm.JITValueDesc
			if d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d53.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d53.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d54)
			}
			ctx.FreeDesc(&d53)
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d54.Imm.Int())}
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r74, d52.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d55)
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
				r75 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r75, d52.Reg)
				ctx.W.EmitAddInt64(r75, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d55)
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
				r76 := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitCmpRegImm32(d55.Reg, 64)
				ctx.W.EmitSetcc(r76, scm.CcA)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d56)
			}
			ctx.FreeDesc(&d55)
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d56.Loc == scm.LocImm {
				if d56.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
			ctx.EmitStoreToStack(d44, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d56.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
			ctx.EmitStoreToStack(d44, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.FreeDesc(&d56)
			ctx.W.MarkLabel(lbl18)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d57 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r77, d39.Reg)
				ctx.W.EmitShrRegImm8(r77, 6)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d57)
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
			r78 := ctx.AllocReg()
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r78, uint64(d58.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r78, d58.Reg)
				ctx.W.EmitShlRegImm8(r78, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r78, d40.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.W.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d59 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d59)
			ctx.FreeDesc(&d58)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d60 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r80, d39.Reg)
				ctx.W.EmitAndRegImm32(r80, 63)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d60)
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
				r81 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r81, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d62)
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
				r82 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r82, d61.Reg)
				ctx.W.EmitSubInt64(r82, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d62)
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
				r83 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r83, d59.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d63)
			} else {
				{
					shiftSrc := d59.Reg
					r84 := ctx.AllocRegExcept(d59.Reg)
					ctx.W.EmitMovRegReg(r84, d59.Reg)
					shiftSrc = r84
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
				r85 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r85, d44.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d64)
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d64)
			} else if d63.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r86, d44.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r86, int32(d63.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.W.EmitOrInt64(r86, scm.RegR11)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d64)
			} else {
				r87 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r87, d44.Reg)
				ctx.W.EmitOrInt64(r87, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d64)
			}
			if d64.Loc == scm.LocReg && d44.Loc == scm.LocReg && d64.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.EmitStoreToStack(d64, 8)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl14)
			ctx.W.ResolveFixups()
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d65)
			ctx.BindReg(r70, &d65)
			if r49 { ctx.UnprotectReg(r50) }
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d65.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d65.Reg)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d66)
			}
			ctx.FreeDesc(&d65)
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r89, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d67)
			}
			if d66.Loc == scm.LocStack || d66.Loc == scm.LocStackPair { ctx.EnsureDesc(&d66) }
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			var d68 scm.JITValueDesc
			if d66.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + d67.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r90, d66.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d68)
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
				r91 := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(r91, d66.Reg)
				ctx.W.EmitAddInt64(r91, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d68)
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
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d68.Reg)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d69)
			}
			ctx.FreeDesc(&d68)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d70)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d70.Loc == scm.LocImm {
				if d70.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d70.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d70)
			ctx.W.MarkLabel(lbl12)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			r94 := d34.Loc == scm.LocReg
			r95 := d34.Reg
			if r94 { ctx.ProtectReg(r95) }
			lbl23 := ctx.W.ReserveLabel()
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d71 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d34.Reg)
				ctx.W.EmitShlRegImm8(r96, 32)
				ctx.W.EmitShrRegImm8(r96, 32)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d71)
			}
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r97, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d72)
			}
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			var d73 scm.JITValueDesc
			if d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d72.Imm.Int()))))}
			} else {
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r98, d72.Reg)
				ctx.W.EmitShlRegImm8(r98, 56)
				ctx.W.EmitShrRegImm8(r98, 56)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d73)
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
				r99 := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegReg(r99, d71.Reg)
				ctx.W.EmitImulInt64(r99, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d74)
			}
			if d74.Loc == scm.LocReg && d71.Loc == scm.LocReg && d74.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.FreeDesc(&d73)
			var d75 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				r100 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r100, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d75)
			}
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d76 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r101 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r101, d74.Reg)
				ctx.W.EmitShrRegImm8(r101, 6)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d76)
			}
			if d76.Loc == scm.LocReg && d74.Loc == scm.LocReg && d76.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			r102 := ctx.AllocReg()
			if d76.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r102, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r102, d76.Reg)
				ctx.W.EmitShlRegImm8(r102, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r102, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r102, d75.Reg)
			}
			r103 := ctx.AllocRegExcept(r102)
			ctx.W.EmitMovRegMem(r103, r102, 0)
			ctx.FreeReg(r102)
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d77)
			ctx.FreeDesc(&d76)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d78 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r104 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r104, d74.Reg)
				ctx.W.EmitAndRegImm32(r104, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d78)
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
				r105 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(r105, d77.Reg)
				ctx.W.EmitShlRegImm8(r105, uint8(d78.Imm.Int()))
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d79)
			} else {
				{
					shiftSrc := d77.Reg
					r106 := ctx.AllocRegExcept(d77.Reg)
					ctx.W.EmitMovRegReg(r106, d77.Reg)
					shiftSrc = r106
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d80)
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
			ctx.EmitStoreToStack(d79, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d80.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
			ctx.EmitStoreToStack(d79, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d80)
			ctx.W.MarkLabel(lbl25)
			r108 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r108, 16)
			ctx.ProtectReg(r108)
			d81 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r108}
			ctx.BindReg(r108, &d81)
			ctx.UnprotectReg(r108)
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r109, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r109}
				ctx.BindReg(r109, &d82)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d82.Imm.Int()))))}
			} else {
				r110 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r110, d82.Reg)
				ctx.W.EmitShlRegImm8(r110, 56)
				ctx.W.EmitShrRegImm8(r110, 56)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d83)
			}
			ctx.FreeDesc(&d82)
			d84 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d83.Loc == scm.LocStack || d83.Loc == scm.LocStackPair { ctx.EnsureDesc(&d83) }
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() - d83.Imm.Int())}
			} else if d83.Loc == scm.LocImm && d83.Imm.Int() == 0 {
				r111 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r111, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d85)
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
				r112 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r112, d84.Reg)
				ctx.W.EmitSubInt64(r112, d83.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d85)
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
				r113 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r113, d81.Reg)
				ctx.W.EmitShrRegImm8(r113, uint8(d85.Imm.Int()))
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d86)
			} else {
				{
					shiftSrc := d81.Reg
					r114 := ctx.AllocRegExcept(d81.Reg)
					ctx.W.EmitMovRegReg(r114, d81.Reg)
					shiftSrc = r114
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
			r115 := ctx.AllocReg()
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			ctx.EmitMovToReg(r115, d86)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl24)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d87 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r116 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r116, d74.Reg)
				ctx.W.EmitAndRegImm32(r116, 63)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d87)
			}
			if d87.Loc == scm.LocReg && d74.Loc == scm.LocReg && d87.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d88 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r117 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r117, thisptr.Reg, off)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r117}
				ctx.BindReg(r117, &d88)
			}
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d88.Imm.Int()))))}
			} else {
				r118 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r118, d88.Reg)
				ctx.W.EmitShlRegImm8(r118, 56)
				ctx.W.EmitShrRegImm8(r118, 56)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d89)
			}
			ctx.FreeDesc(&d88)
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			var d90 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d89.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				r119 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r119, d87.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d90)
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
				r120 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r120, d87.Reg)
				ctx.W.EmitAddInt64(r120, d89.Reg)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
				ctx.BindReg(r120, &d90)
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
				r121 := ctx.AllocRegExcept(d90.Reg)
				ctx.W.EmitCmpRegImm32(d90.Reg, 64)
				ctx.W.EmitSetcc(r121, scm.CcA)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r121}
				ctx.BindReg(r121, &d91)
			}
			ctx.FreeDesc(&d90)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			if d91.Loc == scm.LocImm {
				if d91.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
			ctx.EmitStoreToStack(d79, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d91.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
			ctx.EmitStoreToStack(d79, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d91)
			ctx.W.MarkLabel(lbl27)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d92 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r122 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r122, d74.Reg)
				ctx.W.EmitShrRegImm8(r122, 6)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d92)
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
			r123 := ctx.AllocReg()
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r123, uint64(d93.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r123, d93.Reg)
				ctx.W.EmitShlRegImm8(r123, 3)
			}
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
				ctx.W.EmitAddInt64(r123, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r123, d75.Reg)
			}
			r124 := ctx.AllocRegExcept(r123)
			ctx.W.EmitMovRegMem(r124, r123, 0)
			ctx.FreeReg(r123)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r124}
			ctx.BindReg(r124, &d94)
			ctx.FreeDesc(&d93)
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d95 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r125 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r125, d74.Reg)
				ctx.W.EmitAndRegImm32(r125, 63)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d95)
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
				r126 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r126, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d97)
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
				r127 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r127, d96.Reg)
				ctx.W.EmitSubInt64(r127, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d97)
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
				r128 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r128, d94.Reg)
				ctx.W.EmitShrRegImm8(r128, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d98)
			} else {
				{
					shiftSrc := d94.Reg
					r129 := ctx.AllocRegExcept(d94.Reg)
					ctx.W.EmitMovRegReg(r129, d94.Reg)
					shiftSrc = r129
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
				r130 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r130, d79.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d99)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d99)
			} else if d98.Loc == scm.LocImm {
				r131 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r131, d79.Reg)
				if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r131, int32(d98.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d98.Imm.Int()))
					ctx.W.EmitOrInt64(r131, scm.RegR11)
				}
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d99)
			} else {
				r132 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r132, d79.Reg)
				ctx.W.EmitOrInt64(r132, d98.Reg)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d99)
			}
			if d99.Loc == scm.LocReg && d79.Loc == scm.LocReg && d99.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d98)
			ctx.EmitStoreToStack(d99, 16)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			d100 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
			ctx.BindReg(r115, &d100)
			ctx.BindReg(r115, &d100)
			if r94 { ctx.UnprotectReg(r95) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d100.Imm.Int()))))}
			} else {
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r133, d100.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d101)
			}
			ctx.FreeDesc(&d100)
			var d102 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r134 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r134, thisptr.Reg, off)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r134}
				ctx.BindReg(r134, &d102)
			}
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			var d103 scm.JITValueDesc
			if d101.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d101.Imm.Int() + d102.Imm.Int())}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				r135 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r135, d101.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d103)
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
				r136 := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegReg(r136, d101.Reg)
				ctx.W.EmitAddInt64(r136, d102.Reg)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d103)
			}
			if d103.Loc == scm.LocReg && d101.Loc == scm.LocReg && d103.Reg == d101.Reg {
				ctx.TransferReg(d101.Reg)
				d101.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d101)
			ctx.FreeDesc(&d102)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			r137 := d34.Loc == scm.LocReg
			r138 := d34.Reg
			if r137 { ctx.ProtectReg(r138) }
			lbl29 := ctx.W.ReserveLabel()
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d104 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d34.Imm.Int()))))}
			} else {
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r139, d34.Reg)
				ctx.W.EmitShlRegImm8(r139, 32)
				ctx.W.EmitShrRegImm8(r139, 32)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d104)
			}
			var d105 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r140, thisptr.Reg, off)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
				ctx.BindReg(r140, &d105)
			}
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d105.Imm.Int()))))}
			} else {
				r141 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r141, d105.Reg)
				ctx.W.EmitShlRegImm8(r141, 56)
				ctx.W.EmitShrRegImm8(r141, 56)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d106)
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
				r142 := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegReg(r142, d104.Reg)
				ctx.W.EmitImulInt64(r142, d106.Reg)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r142}
				ctx.BindReg(r142, &d107)
			}
			if d107.Loc == scm.LocReg && d104.Loc == scm.LocReg && d107.Reg == d104.Reg {
				ctx.TransferReg(d104.Reg)
				d104.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.FreeDesc(&d106)
			var d108 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				r143 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r143, thisptr.Reg, off)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
				ctx.BindReg(r143, &d108)
			}
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d109 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r144 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r144, d107.Reg)
				ctx.W.EmitShrRegImm8(r144, 6)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d109)
			}
			if d109.Loc == scm.LocReg && d107.Loc == scm.LocReg && d109.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			r145 := ctx.AllocReg()
			if d109.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r145, uint64(d109.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r145, d109.Reg)
				ctx.W.EmitShlRegImm8(r145, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r145, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r145, d108.Reg)
			}
			r146 := ctx.AllocRegExcept(r145)
			ctx.W.EmitMovRegMem(r146, r145, 0)
			ctx.FreeReg(r145)
			d110 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
			ctx.BindReg(r146, &d110)
			ctx.FreeDesc(&d109)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d111 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r147 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r147, d107.Reg)
				ctx.W.EmitAndRegImm32(r147, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d111)
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
				r148 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r148, d110.Reg)
				ctx.W.EmitShlRegImm8(r148, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d112)
			} else {
				{
					shiftSrc := d110.Reg
					r149 := ctx.AllocRegExcept(d110.Reg)
					ctx.W.EmitMovRegReg(r149, d110.Reg)
					shiftSrc = r149
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d113)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
			ctx.EmitStoreToStack(d112, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d113.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
			ctx.EmitStoreToStack(d112, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d113)
			ctx.W.MarkLabel(lbl31)
			r151 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r151, 24)
			ctx.ProtectReg(r151)
			d114 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r151}
			ctx.BindReg(r151, &d114)
			ctx.UnprotectReg(r151)
			var d115 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r152 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r152, thisptr.Reg, off)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r152}
				ctx.BindReg(r152, &d115)
			}
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d115.Imm.Int()))))}
			} else {
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r153, d115.Reg)
				ctx.W.EmitShlRegImm8(r153, 56)
				ctx.W.EmitShrRegImm8(r153, 56)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d116)
			}
			ctx.FreeDesc(&d115)
			d117 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair { ctx.EnsureDesc(&d116) }
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() - d116.Imm.Int())}
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				r154 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r154, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d118)
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
				r155 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r155, d117.Reg)
				ctx.W.EmitSubInt64(r155, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d118)
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
				r156 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r156, d114.Reg)
				ctx.W.EmitShrRegImm8(r156, uint8(d118.Imm.Int()))
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r156}
				ctx.BindReg(r156, &d119)
			} else {
				{
					shiftSrc := d114.Reg
					r157 := ctx.AllocRegExcept(d114.Reg)
					ctx.W.EmitMovRegReg(r157, d114.Reg)
					shiftSrc = r157
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
			r158 := ctx.AllocReg()
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			ctx.EmitMovToReg(r158, d119)
			ctx.W.EmitJmp(lbl29)
			ctx.W.MarkLabel(lbl30)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d120 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r159 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r159, d107.Reg)
				ctx.W.EmitAndRegImm32(r159, 63)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d120)
			}
			if d120.Loc == scm.LocReg && d107.Loc == scm.LocReg && d120.Reg == d107.Reg {
				ctx.TransferReg(d107.Reg)
				d107.Loc = scm.LocNone
			}
			var d121 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r160 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r160, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r160}
				ctx.BindReg(r160, &d121)
			}
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d121.Imm.Int()))))}
			} else {
				r161 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r161, d121.Reg)
				ctx.W.EmitShlRegImm8(r161, 56)
				ctx.W.EmitShrRegImm8(r161, 56)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d122)
			}
			ctx.FreeDesc(&d121)
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d123 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() + d122.Imm.Int())}
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r162 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r162, d120.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d123)
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
				r163 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r163, d120.Reg)
				ctx.W.EmitAddInt64(r163, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d123)
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
				r164 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitCmpRegImm32(d123.Reg, 64)
				ctx.W.EmitSetcc(r164, scm.CcA)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r164}
				ctx.BindReg(r164, &d124)
			}
			ctx.FreeDesc(&d123)
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d124.Loc == scm.LocImm {
				if d124.Imm.Bool() {
					ctx.W.EmitJmp(lbl33)
				} else {
			ctx.EmitStoreToStack(d112, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d124.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
			ctx.EmitStoreToStack(d112, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d124)
			ctx.W.MarkLabel(lbl33)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d125 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() / 64)}
			} else {
				r165 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r165, d107.Reg)
				ctx.W.EmitShrRegImm8(r165, 6)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d125)
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
			r166 := ctx.AllocReg()
			if d126.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r166, uint64(d126.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r166, d126.Reg)
				ctx.W.EmitShlRegImm8(r166, 3)
			}
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(r166, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r166, d108.Reg)
			}
			r167 := ctx.AllocRegExcept(r166)
			ctx.W.EmitMovRegMem(r167, r166, 0)
			ctx.FreeReg(r166)
			d127 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r167}
			ctx.BindReg(r167, &d127)
			ctx.FreeDesc(&d126)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			var d128 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d107.Imm.Int() % 64)}
			} else {
				r168 := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegReg(r168, d107.Reg)
				ctx.W.EmitAndRegImm32(r168, 63)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d128)
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
				r169 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r169, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d130)
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
				r170 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r170, d129.Reg)
				ctx.W.EmitSubInt64(r170, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d130)
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
				r171 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r171, d127.Reg)
				ctx.W.EmitShrRegImm8(r171, uint8(d130.Imm.Int()))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d131)
			} else {
				{
					shiftSrc := d127.Reg
					r172 := ctx.AllocRegExcept(d127.Reg)
					ctx.W.EmitMovRegReg(r172, d127.Reg)
					shiftSrc = r172
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
				r173 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r173, d112.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d132)
			} else if d112.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d112.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d132)
			} else if d131.Loc == scm.LocImm {
				r174 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r174, d112.Reg)
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r174, int32(d131.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
					ctx.W.EmitOrInt64(r174, scm.RegR11)
				}
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d132)
			} else {
				r175 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r175, d112.Reg)
				ctx.W.EmitOrInt64(r175, d131.Reg)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d132)
			}
			if d132.Loc == scm.LocReg && d112.Loc == scm.LocReg && d132.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.EmitStoreToStack(d132, 24)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d133 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
			ctx.BindReg(r158, &d133)
			ctx.BindReg(r158, &d133)
			if r137 { ctx.UnprotectReg(r138) }
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d134 scm.JITValueDesc
			if d133.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d133.Imm.Int()))))}
			} else {
				r176 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r176, d133.Reg)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d134)
			}
			ctx.FreeDesc(&d133)
			var d135 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r177 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r177, thisptr.Reg, off)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r177}
				ctx.BindReg(r177, &d135)
			}
			if d134.Loc == scm.LocStack || d134.Loc == scm.LocStackPair { ctx.EnsureDesc(&d134) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			var d136 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r178, d134.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d136)
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
				r179 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r179, d134.Reg)
				ctx.W.EmitAddInt64(r179, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d136)
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
				r180 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r180, d103.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d138)
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
				r181 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r181, d103.Reg)
				ctx.W.EmitAddInt64(r181, d136.Reg)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d138)
			}
			if d138.Loc == scm.LocReg && d103.Loc == scm.LocReg && d138.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			var d140 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r182 := ctx.AllocReg()
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r182, fieldAddr)
				ctx.W.EmitMovRegMem64(r183, fieldAddr+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
				ctx.BindReg(r182, &d140)
				ctx.BindReg(r183, &d140)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r184 := ctx.AllocReg()
				r185 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r184, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r185, thisptr.Reg, off+8)
				d140 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
				ctx.BindReg(r184, &d140)
				ctx.BindReg(r185, &d140)
			}
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			r186 := ctx.AllocReg()
			r187 := ctx.AllocRegExcept(r186)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r186, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r186, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r186, d140.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() != 0 {
					if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r186, int32(d103.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
						ctx.W.EmitAddInt64(r186, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r186, d103.Reg)
			}
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r187, uint64(d138.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r187, d138.Reg)
			}
			if d103.Loc == scm.LocImm {
				if d103.Imm.Int() >= -2147483648 && d103.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r187, int32(d103.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d103.Imm.Int()))
					ctx.W.EmitSubInt64(r187, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r187, d103.Reg)
			}
			d141 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d141)
			ctx.BindReg(r187, &d141)
			ctx.FreeDesc(&d103)
			ctx.FreeDesc(&d138)
			r188 := ctx.AllocReg()
			r189 := ctx.AllocRegExcept(r188)
			d142 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r188, Reg2: r189}
			ctx.BindReg(r188, &d142)
			ctx.BindReg(r189, &d142)
			d143 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d141}, 2)
			ctx.EmitMovPairToResult(&d143, &d142)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl11)
			var d144 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r190 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r190, thisptr.Reg, off)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r190}
				ctx.BindReg(r190, &d144)
			}
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			var d145 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d144.Imm.Int()))))}
			} else {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r191, d144.Reg)
				ctx.W.EmitShlRegImm8(r191, 32)
				ctx.W.EmitShrRegImm8(r191, 32)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r191}
				ctx.BindReg(r191, &d145)
			}
			ctx.FreeDesc(&d144)
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			var d146 scm.JITValueDesc
			if d34.Loc == scm.LocImm && d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d34.Imm.Int()) == uint64(d145.Imm.Int()))}
			} else if d145.Loc == scm.LocImm {
				r192 := ctx.AllocRegExcept(d34.Reg)
				if d145.Imm.Int() >= -2147483648 && d145.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d34.Reg, int32(d145.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d145.Imm.Int()))
					ctx.W.EmitCmpInt64(d34.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r192, scm.CcE)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
				ctx.BindReg(r192, &d146)
			} else if d34.Loc == scm.LocImm {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d145.Reg)
				ctx.W.EmitSetcc(r193, scm.CcE)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r193}
				ctx.BindReg(r193, &d146)
			} else {
				r194 := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitCmpInt64(d34.Reg, d145.Reg)
				ctx.W.EmitSetcc(r194, scm.CcE)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r194}
				ctx.BindReg(r194, &d146)
			}
			ctx.FreeDesc(&d34)
			ctx.FreeDesc(&d145)
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d146.Loc == scm.LocImm {
				if d146.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d146.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d146)
			ctx.W.MarkLabel(lbl21)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r195 := idxInt.Loc == scm.LocReg
			r196 := idxInt.Reg
			if r195 { ctx.ProtectReg(r196) }
			lbl37 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d147 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r197 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r197, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r197, 32)
				ctx.W.EmitShrRegImm8(r197, 32)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d147)
			}
			var d148 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r198 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r198, thisptr.Reg, off)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r198}
				ctx.BindReg(r198, &d148)
			}
			if d148.Loc == scm.LocStack || d148.Loc == scm.LocStackPair { ctx.EnsureDesc(&d148) }
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d148.Imm.Int()))))}
			} else {
				r199 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r199, d148.Reg)
				ctx.W.EmitShlRegImm8(r199, 56)
				ctx.W.EmitShrRegImm8(r199, 56)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d149)
			}
			ctx.FreeDesc(&d148)
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d150 scm.JITValueDesc
			if d147.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d147.Imm.Int() * d149.Imm.Int())}
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d147.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d149.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d150)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(scratch, d147.Reg)
				if d149.Imm.Int() >= -2147483648 && d149.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d149.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d149.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d150)
			} else {
				r200 := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegReg(r200, d147.Reg)
				ctx.W.EmitImulInt64(r200, d149.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d150)
			}
			if d150.Loc == scm.LocReg && d147.Loc == scm.LocReg && d150.Reg == d147.Reg {
				ctx.TransferReg(d147.Reg)
				d147.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			ctx.FreeDesc(&d149)
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				r201 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r201, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
				ctx.BindReg(r201, &d151)
			}
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d152 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() / 64)}
			} else {
				r202 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r202, d150.Reg)
				ctx.W.EmitShrRegImm8(r202, 6)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d152)
			}
			if d152.Loc == scm.LocReg && d150.Loc == scm.LocReg && d152.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			r203 := ctx.AllocReg()
			if d152.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r203, uint64(d152.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r203, d152.Reg)
				ctx.W.EmitShlRegImm8(r203, 3)
			}
			if d151.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
				ctx.W.EmitAddInt64(r203, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r203, d151.Reg)
			}
			r204 := ctx.AllocRegExcept(r203)
			ctx.W.EmitMovRegMem(r204, r203, 0)
			ctx.FreeReg(r203)
			d153 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
			ctx.BindReg(r204, &d153)
			ctx.FreeDesc(&d152)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d154 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() % 64)}
			} else {
				r205 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r205, d150.Reg)
				ctx.W.EmitAndRegImm32(r205, 63)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d154)
			}
			if d154.Loc == scm.LocReg && d150.Loc == scm.LocReg && d154.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			var d155 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d153.Imm.Int()) << uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r206 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r206, d153.Reg)
				ctx.W.EmitShlRegImm8(r206, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d155)
			} else {
				{
					shiftSrc := d153.Reg
					r207 := ctx.AllocRegExcept(d153.Reg)
					ctx.W.EmitMovRegReg(r207, d153.Reg)
					shiftSrc = r207
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d155)
				}
			}
			if d155.Loc == scm.LocReg && d153.Loc == scm.LocReg && d155.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d153)
			ctx.FreeDesc(&d154)
			var d156 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r208 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r208, thisptr.Reg, off)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r208}
				ctx.BindReg(r208, &d156)
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d156.Loc == scm.LocImm {
				if d156.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
			ctx.EmitStoreToStack(d155, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d156.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
			ctx.EmitStoreToStack(d155, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d156)
			ctx.W.MarkLabel(lbl39)
			r209 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r209, 32)
			ctx.ProtectReg(r209)
			d157 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r209}
			ctx.BindReg(r209, &d157)
			ctx.UnprotectReg(r209)
			var d158 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d158)
			}
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d159 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d158.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d158.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d159)
			}
			ctx.FreeDesc(&d158)
			d160 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			var d161 scm.JITValueDesc
			if d160.Loc == scm.LocImm && d159.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d160.Imm.Int() - d159.Imm.Int())}
			} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegReg(r212, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d161)
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d160.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d159.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegReg(scratch, d160.Reg)
				if d159.Imm.Int() >= -2147483648 && d159.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d159.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else {
				r213 := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegReg(r213, d160.Reg)
				ctx.W.EmitSubInt64(r213, d159.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d161)
			}
			if d161.Loc == scm.LocReg && d160.Loc == scm.LocReg && d161.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d159)
			if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair { ctx.EnsureDesc(&d157) }
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			var d162 scm.JITValueDesc
			if d157.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d157.Imm.Int()) >> uint64(d161.Imm.Int())))}
			} else if d161.Loc == scm.LocImm {
				r214 := ctx.AllocRegExcept(d157.Reg)
				ctx.W.EmitMovRegReg(r214, d157.Reg)
				ctx.W.EmitShrRegImm8(r214, uint8(d161.Imm.Int()))
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d162)
			} else {
				{
					shiftSrc := d157.Reg
					r215 := ctx.AllocRegExcept(d157.Reg)
					ctx.W.EmitMovRegReg(r215, d157.Reg)
					shiftSrc = r215
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d161.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d161.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d161.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d162)
				}
			}
			if d162.Loc == scm.LocReg && d157.Loc == scm.LocReg && d162.Reg == d157.Reg {
				ctx.TransferReg(d157.Reg)
				d157.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d157)
			ctx.FreeDesc(&d161)
			r216 := ctx.AllocReg()
			if d162.Loc == scm.LocStack || d162.Loc == scm.LocStackPair { ctx.EnsureDesc(&d162) }
			ctx.EmitMovToReg(r216, d162)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl38)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d163 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() % 64)}
			} else {
				r217 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r217, d150.Reg)
				ctx.W.EmitAndRegImm32(r217, 63)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d163)
			}
			if d163.Loc == scm.LocReg && d150.Loc == scm.LocReg && d163.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			var d164 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r218 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r218, thisptr.Reg, off)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r218}
				ctx.BindReg(r218, &d164)
			}
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d164.Imm.Int()))))}
			} else {
				r219 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r219, d164.Reg)
				ctx.W.EmitShlRegImm8(r219, 56)
				ctx.W.EmitShrRegImm8(r219, 56)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d165)
			}
			ctx.FreeDesc(&d164)
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			var d166 scm.JITValueDesc
			if d163.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d163.Imm.Int() + d165.Imm.Int())}
			} else if d165.Loc == scm.LocImm && d165.Imm.Int() == 0 {
				r220 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r220, d163.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d166)
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
				ctx.BindReg(d165.Reg, &d166)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d165.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d163.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else if d165.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(scratch, d163.Reg)
				if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d165.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else {
				r221 := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegReg(r221, d163.Reg)
				ctx.W.EmitAddInt64(r221, d165.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d166)
			}
			if d166.Loc == scm.LocReg && d163.Loc == scm.LocReg && d166.Reg == d163.Reg {
				ctx.TransferReg(d163.Reg)
				d163.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.FreeDesc(&d165)
			if d166.Loc == scm.LocStack || d166.Loc == scm.LocStackPair { ctx.EnsureDesc(&d166) }
			var d167 scm.JITValueDesc
			if d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d166.Imm.Int()) > uint64(64))}
			} else {
				r222 := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitCmpRegImm32(d166.Reg, 64)
				ctx.W.EmitSetcc(r222, scm.CcA)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r222}
				ctx.BindReg(r222, &d167)
			}
			ctx.FreeDesc(&d166)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d167.Loc == scm.LocImm {
				if d167.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
			ctx.EmitStoreToStack(d155, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d167.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
			ctx.EmitStoreToStack(d155, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d167)
			ctx.W.MarkLabel(lbl41)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d168 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() / 64)}
			} else {
				r223 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r223, d150.Reg)
				ctx.W.EmitShrRegImm8(r223, 6)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d168)
			}
			if d168.Loc == scm.LocReg && d150.Loc == scm.LocReg && d168.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(scratch, d168.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			}
			if d169.Loc == scm.LocReg && d168.Loc == scm.LocReg && d169.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d168)
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			r224 := ctx.AllocReg()
			if d169.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r224, uint64(d169.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r224, d169.Reg)
				ctx.W.EmitShlRegImm8(r224, 3)
			}
			if d151.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d151.Imm.Int()))
				ctx.W.EmitAddInt64(r224, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r224, d151.Reg)
			}
			r225 := ctx.AllocRegExcept(r224)
			ctx.W.EmitMovRegMem(r225, r224, 0)
			ctx.FreeReg(r224)
			d170 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r225}
			ctx.BindReg(r225, &d170)
			ctx.FreeDesc(&d169)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d171 scm.JITValueDesc
			if d150.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d150.Imm.Int() % 64)}
			} else {
				r226 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r226, d150.Reg)
				ctx.W.EmitAndRegImm32(r226, 63)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d171)
			}
			if d171.Loc == scm.LocReg && d150.Loc == scm.LocReg && d171.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			d172 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			var d173 scm.JITValueDesc
			if d172.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d172.Imm.Int() - d171.Imm.Int())}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r227 := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegReg(r227, d172.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d173)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d172.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegReg(scratch, d172.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d171.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d173)
			} else {
				r228 := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegReg(r228, d172.Reg)
				ctx.W.EmitSubInt64(r228, d171.Reg)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d173)
			}
			if d173.Loc == scm.LocReg && d172.Loc == scm.LocReg && d173.Reg == d172.Reg {
				ctx.TransferReg(d172.Reg)
				d172.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			var d174 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d170.Imm.Int()) >> uint64(d173.Imm.Int())))}
			} else if d173.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r229, d170.Reg)
				ctx.W.EmitShrRegImm8(r229, uint8(d173.Imm.Int()))
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d174)
			} else {
				{
					shiftSrc := d170.Reg
					r230 := ctx.AllocRegExcept(d170.Reg)
					ctx.W.EmitMovRegReg(r230, d170.Reg)
					shiftSrc = r230
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d173.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d173.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d173.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d174)
				}
			}
			if d174.Loc == scm.LocReg && d170.Loc == scm.LocReg && d174.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.FreeDesc(&d173)
			if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair { ctx.EnsureDesc(&d155) }
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			var d175 scm.JITValueDesc
			if d155.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d155.Imm.Int() | d174.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d174.Reg}
				ctx.BindReg(d174.Reg, &d175)
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r231, d155.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d175)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d155.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d175)
			} else if d174.Loc == scm.LocImm {
				r232 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r232, d155.Reg)
				if d174.Imm.Int() >= -2147483648 && d174.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r232, int32(d174.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d174.Imm.Int()))
					ctx.W.EmitOrInt64(r232, scm.RegR11)
				}
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d175)
			} else {
				r233 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegReg(r233, d155.Reg)
				ctx.W.EmitOrInt64(r233, d174.Reg)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d175)
			}
			if d175.Loc == scm.LocReg && d155.Loc == scm.LocReg && d175.Reg == d155.Reg {
				ctx.TransferReg(d155.Reg)
				d155.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.EmitStoreToStack(d175, 32)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d176 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r216}
			ctx.BindReg(r216, &d176)
			ctx.BindReg(r216, &d176)
			if r195 { ctx.UnprotectReg(r196) }
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d176.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d177)
			}
			ctx.FreeDesc(&d176)
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r235, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r235}
				ctx.BindReg(r235, &d178)
			}
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			var d179 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() + d178.Imm.Int())}
			} else if d178.Loc == scm.LocImm && d178.Imm.Int() == 0 {
				r236 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(r236, d177.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d179)
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d178.Reg}
				ctx.BindReg(d178.Reg, &d179)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d178.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d179)
			} else if d178.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(scratch, d177.Reg)
				if d178.Imm.Int() >= -2147483648 && d178.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d178.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d178.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d179)
			} else {
				r237 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(r237, d177.Reg)
				ctx.W.EmitAddInt64(r237, d178.Reg)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d179)
			}
			if d179.Loc == scm.LocReg && d177.Loc == scm.LocReg && d179.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d178)
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			var d180 scm.JITValueDesc
			if d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d179.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d180)
			}
			ctx.FreeDesc(&d179)
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			var d181 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d69.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r239, d69.Reg)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d181)
			}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			var d182 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + d180.Imm.Int())}
			} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
				r240 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r240, d69.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r240}
				ctx.BindReg(r240, &d182)
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d180.Reg}
				ctx.BindReg(d180.Reg, &d182)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d180.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d182)
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(scratch, d69.Reg)
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d182)
			} else {
				r241 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(r241, d69.Reg)
				ctx.W.EmitAddInt64(r241, d180.Reg)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r241}
				ctx.BindReg(r241, &d182)
			}
			if d182.Loc == scm.LocReg && d69.Loc == scm.LocReg && d182.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			if d182.Loc == scm.LocStack || d182.Loc == scm.LocStackPair { ctx.EnsureDesc(&d182) }
			var d183 scm.JITValueDesc
			if d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d182.Imm.Int()))))}
			} else {
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r242, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r242}
				ctx.BindReg(r242, &d183)
			}
			ctx.FreeDesc(&d182)
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			r243 := ctx.AllocReg()
			r244 := ctx.AllocRegExcept(r243)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			if d140.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r243, uint64(d140.Imm.Int()))
			} else if d140.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r243, d140.Reg)
			} else {
				ctx.W.EmitMovRegReg(r243, d140.Reg)
			}
			if d181.Loc == scm.LocImm {
				if d181.Imm.Int() != 0 {
					if d181.Imm.Int() >= -2147483648 && d181.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r243, int32(d181.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
						ctx.W.EmitAddInt64(r243, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r243, d181.Reg)
			}
			if d183.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r244, uint64(d183.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r244, d183.Reg)
			}
			if d181.Loc == scm.LocImm {
				if d181.Imm.Int() >= -2147483648 && d181.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r244, int32(d181.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
					ctx.W.EmitSubInt64(r244, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r244, d181.Reg)
			}
			d184 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r243, Reg2: r244}
			ctx.BindReg(r243, &d184)
			ctx.BindReg(r244, &d184)
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d183)
			d185 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r188, Reg2: r189}
			ctx.BindReg(r188, &d185)
			ctx.BindReg(r189, &d185)
			d186 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d184}, 2)
			ctx.EmitMovPairToResult(&d186, &d185)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl20)
			var d187 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r245, thisptr.Reg, off)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r245}
				ctx.BindReg(r245, &d187)
			}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d188 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d69.Imm.Int()) == uint64(d187.Imm.Int()))}
			} else if d187.Loc == scm.LocImm {
				r246 := ctx.AllocRegExcept(d69.Reg)
				if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d69.Reg, int32(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
					ctx.W.EmitCmpInt64(d69.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r246, scm.CcE)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r246}
				ctx.BindReg(r246, &d188)
			} else if d69.Loc == scm.LocImm {
				r247 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d187.Reg)
				ctx.W.EmitSetcc(r247, scm.CcE)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r247}
				ctx.BindReg(r247, &d188)
			} else {
				r248 := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitCmpInt64(d69.Reg, d187.Reg)
				ctx.W.EmitSetcc(r248, scm.CcE)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r248}
				ctx.BindReg(r248, &d188)
			}
			ctx.FreeDesc(&d69)
			ctx.FreeDesc(&d187)
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d188.Loc == scm.LocImm {
				if d188.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d188.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl44)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl44)
				ctx.W.EmitJmp(lbl43)
			}
			ctx.FreeDesc(&d188)
			ctx.W.MarkLabel(lbl35)
			d189 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r188, Reg2: r189}
			ctx.BindReg(r188, &d189)
			ctx.BindReg(r189, &d189)
			ctx.W.EmitMakeNil(d189)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl43)
			d190 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r188, Reg2: r189}
			ctx.BindReg(r188, &d190)
			ctx.BindReg(r189, &d190)
			ctx.W.EmitMakeNil(d190)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d191 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r188, Reg2: r189}
			ctx.BindReg(r188, &d191)
			ctx.BindReg(r189, &d191)
			ctx.BindReg(r188, &d191)
			ctx.BindReg(r189, &d191)
			if r0 { ctx.UnprotectReg(r1) }
			d192 := ctx.EmitTagEquals(&d191, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d192.Loc == scm.LocImm {
				if d192.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d192.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d192)
			ctx.W.MarkLabel(lbl46)
			d193 := ctx.EmitTagEquals(&d191, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			if d193.Loc == scm.LocImm {
				if d193.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d193.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d193)
			ctx.W.MarkLabel(lbl45)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl49)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl48)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			lbl51 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d194 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r249, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r249, 32)
				ctx.W.EmitShrRegImm8(r249, 32)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r249}
				ctx.BindReg(r249, &d194)
			}
			ctx.FreeDesc(&idxInt)
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r250, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r250}
				ctx.BindReg(r250, &d195)
			}
			if d195.Loc == scm.LocStack || d195.Loc == scm.LocStackPair { ctx.EnsureDesc(&d195) }
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r251 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r251, d195.Reg)
				ctx.W.EmitShlRegImm8(r251, 56)
				ctx.W.EmitShrRegImm8(r251, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d196)
			}
			ctx.FreeDesc(&d195)
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair { ctx.EnsureDesc(&d196) }
			var d197 scm.JITValueDesc
			if d194.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d194.Imm.Int() * d196.Imm.Int())}
			} else if d194.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d194.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(scratch, d194.Reg)
				if d196.Imm.Int() >= -2147483648 && d196.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d196.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d196.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d197)
			} else {
				r252 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(r252, d194.Reg)
				ctx.W.EmitImulInt64(r252, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r252}
				ctx.BindReg(r252, &d197)
			}
			if d197.Loc == scm.LocReg && d194.Loc == scm.LocReg && d197.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			ctx.FreeDesc(&d196)
			var d198 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				r253 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r253, thisptr.Reg, off)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r253}
				ctx.BindReg(r253, &d198)
			}
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d199 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() / 64)}
			} else {
				r254 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r254, d197.Reg)
				ctx.W.EmitShrRegImm8(r254, 6)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r254}
				ctx.BindReg(r254, &d199)
			}
			if d199.Loc == scm.LocReg && d197.Loc == scm.LocReg && d199.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			r255 := ctx.AllocReg()
			if d199.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r255, uint64(d199.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r255, d199.Reg)
				ctx.W.EmitShlRegImm8(r255, 3)
			}
			if d198.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d198.Imm.Int()))
				ctx.W.EmitAddInt64(r255, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r255, d198.Reg)
			}
			r256 := ctx.AllocRegExcept(r255)
			ctx.W.EmitMovRegMem(r256, r255, 0)
			ctx.FreeReg(r255)
			d200 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r256}
			ctx.BindReg(r256, &d200)
			ctx.FreeDesc(&d199)
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d201 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() % 64)}
			} else {
				r257 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r257, d197.Reg)
				ctx.W.EmitAndRegImm32(r257, 63)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d201)
			}
			if d201.Loc == scm.LocReg && d197.Loc == scm.LocReg && d201.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			var d202 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d200.Imm.Int()) << uint64(d201.Imm.Int())))}
			} else if d201.Loc == scm.LocImm {
				r258 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r258, d200.Reg)
				ctx.W.EmitShlRegImm8(r258, uint8(d201.Imm.Int()))
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r258}
				ctx.BindReg(r258, &d202)
			} else {
				{
					shiftSrc := d200.Reg
					r259 := ctx.AllocRegExcept(d200.Reg)
					ctx.W.EmitMovRegReg(r259, d200.Reg)
					shiftSrc = r259
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d201.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d201.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d201.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d202)
				}
			}
			if d202.Loc == scm.LocReg && d200.Loc == scm.LocReg && d202.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d201)
			var d203 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r260 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r260, thisptr.Reg, off)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d203)
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d203.Loc == scm.LocImm {
				if d203.Imm.Bool() {
					ctx.W.EmitJmp(lbl52)
				} else {
			ctx.EmitStoreToStack(d202, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d203.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl54)
			ctx.EmitStoreToStack(d202, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d203)
			ctx.W.MarkLabel(lbl53)
			r261 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r261, 40)
			ctx.ProtectReg(r261)
			d204 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r261}
			ctx.BindReg(r261, &d204)
			ctx.UnprotectReg(r261)
			var d205 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r262 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r262, thisptr.Reg, off)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r262}
				ctx.BindReg(r262, &d205)
			}
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			var d206 scm.JITValueDesc
			if d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d205.Imm.Int()))))}
			} else {
				r263 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r263, d205.Reg)
				ctx.W.EmitShlRegImm8(r263, 56)
				ctx.W.EmitShrRegImm8(r263, 56)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d206)
			}
			ctx.FreeDesc(&d205)
			d207 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm && d206.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d207.Imm.Int() - d206.Imm.Int())}
			} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
				r264 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r264, d207.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d208)
			} else if d207.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d207.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d206.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(scratch, d207.Reg)
				if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d206.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d208)
			} else {
				r265 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitMovRegReg(r265, d207.Reg)
				ctx.W.EmitSubInt64(r265, d206.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r265}
				ctx.BindReg(r265, &d208)
			}
			if d208.Loc == scm.LocReg && d207.Loc == scm.LocReg && d208.Reg == d207.Reg {
				ctx.TransferReg(d207.Reg)
				d207.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d206)
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			var d209 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d204.Imm.Int()) >> uint64(d208.Imm.Int())))}
			} else if d208.Loc == scm.LocImm {
				r266 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(r266, d204.Reg)
				ctx.W.EmitShrRegImm8(r266, uint8(d208.Imm.Int()))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r266}
				ctx.BindReg(r266, &d209)
			} else {
				{
					shiftSrc := d204.Reg
					r267 := ctx.AllocRegExcept(d204.Reg)
					ctx.W.EmitMovRegReg(r267, d204.Reg)
					shiftSrc = r267
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d208.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d208.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d208.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d209)
				}
			}
			if d209.Loc == scm.LocReg && d204.Loc == scm.LocReg && d209.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d208)
			r268 := ctx.AllocReg()
			if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair { ctx.EnsureDesc(&d209) }
			ctx.EmitMovToReg(r268, d209)
			ctx.W.EmitJmp(lbl51)
			ctx.W.MarkLabel(lbl52)
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d210 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() % 64)}
			} else {
				r269 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r269, d197.Reg)
				ctx.W.EmitAndRegImm32(r269, 63)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d210)
			}
			if d210.Loc == scm.LocReg && d197.Loc == scm.LocReg && d210.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			var d211 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r270 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r270, thisptr.Reg, off)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r270}
				ctx.BindReg(r270, &d211)
			}
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d211.Imm.Int()))))}
			} else {
				r271 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r271, d211.Reg)
				ctx.W.EmitShlRegImm8(r271, 56)
				ctx.W.EmitShrRegImm8(r271, 56)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d212)
			}
			ctx.FreeDesc(&d211)
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			var d213 scm.JITValueDesc
			if d210.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d210.Imm.Int() + d212.Imm.Int())}
			} else if d212.Loc == scm.LocImm && d212.Imm.Int() == 0 {
				r272 := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(r272, d210.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r272}
				ctx.BindReg(r272, &d213)
			} else if d210.Loc == scm.LocImm && d210.Imm.Int() == 0 {
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d212.Reg}
				ctx.BindReg(d212.Reg, &d213)
			} else if d210.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d210.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(scratch, d210.Reg)
				if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d212.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r273 := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(r273, d210.Reg)
				ctx.W.EmitAddInt64(r273, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d213)
			}
			if d213.Loc == scm.LocReg && d210.Loc == scm.LocReg && d213.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.FreeDesc(&d212)
			if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair { ctx.EnsureDesc(&d213) }
			var d214 scm.JITValueDesc
			if d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d213.Imm.Int()) > uint64(64))}
			} else {
				r274 := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitCmpRegImm32(d213.Reg, 64)
				ctx.W.EmitSetcc(r274, scm.CcA)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r274}
				ctx.BindReg(r274, &d214)
			}
			ctx.FreeDesc(&d213)
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d214.Loc == scm.LocImm {
				if d214.Imm.Bool() {
					ctx.W.EmitJmp(lbl55)
				} else {
			ctx.EmitStoreToStack(d202, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d214.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
			ctx.EmitStoreToStack(d202, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
			}
			ctx.FreeDesc(&d214)
			ctx.W.MarkLabel(lbl55)
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d215 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() / 64)}
			} else {
				r275 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r275, d197.Reg)
				ctx.W.EmitShrRegImm8(r275, 6)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r275}
				ctx.BindReg(r275, &d215)
			}
			if d215.Loc == scm.LocReg && d197.Loc == scm.LocReg && d215.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			if d215.Loc == scm.LocStack || d215.Loc == scm.LocStackPair { ctx.EnsureDesc(&d215) }
			var d216 scm.JITValueDesc
			if d215.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d215.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegReg(scratch, d215.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d216)
			}
			if d216.Loc == scm.LocReg && d215.Loc == scm.LocReg && d216.Reg == d215.Reg {
				ctx.TransferReg(d215.Reg)
				d215.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			if d216.Loc == scm.LocStack || d216.Loc == scm.LocStackPair { ctx.EnsureDesc(&d216) }
			r276 := ctx.AllocReg()
			if d216.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r276, uint64(d216.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r276, d216.Reg)
				ctx.W.EmitShlRegImm8(r276, 3)
			}
			if d198.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d198.Imm.Int()))
				ctx.W.EmitAddInt64(r276, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r276, d198.Reg)
			}
			r277 := ctx.AllocRegExcept(r276)
			ctx.W.EmitMovRegMem(r277, r276, 0)
			ctx.FreeReg(r276)
			d217 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r277}
			ctx.BindReg(r277, &d217)
			ctx.FreeDesc(&d216)
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d218 scm.JITValueDesc
			if d197.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() % 64)}
			} else {
				r278 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r278, d197.Reg)
				ctx.W.EmitAndRegImm32(r278, 63)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d218)
			}
			if d218.Loc == scm.LocReg && d197.Loc == scm.LocReg && d218.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d197)
			d219 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d218.Loc == scm.LocStack || d218.Loc == scm.LocStackPair { ctx.EnsureDesc(&d218) }
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm && d218.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d219.Imm.Int() - d218.Imm.Int())}
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				r279 := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegReg(r279, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d220)
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d219.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegReg(scratch, d219.Reg)
				if d218.Imm.Int() >= -2147483648 && d218.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d218.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d218.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else {
				r280 := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegReg(r280, d219.Reg)
				ctx.W.EmitSubInt64(r280, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r280}
				ctx.BindReg(r280, &d220)
			}
			if d220.Loc == scm.LocReg && d219.Loc == scm.LocReg && d220.Reg == d219.Reg {
				ctx.TransferReg(d219.Reg)
				d219.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			if d217.Loc == scm.LocStack || d217.Loc == scm.LocStackPair { ctx.EnsureDesc(&d217) }
			if d220.Loc == scm.LocStack || d220.Loc == scm.LocStackPair { ctx.EnsureDesc(&d220) }
			var d221 scm.JITValueDesc
			if d217.Loc == scm.LocImm && d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d217.Imm.Int()) >> uint64(d220.Imm.Int())))}
			} else if d220.Loc == scm.LocImm {
				r281 := ctx.AllocRegExcept(d217.Reg)
				ctx.W.EmitMovRegReg(r281, d217.Reg)
				ctx.W.EmitShrRegImm8(r281, uint8(d220.Imm.Int()))
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d221)
			} else {
				{
					shiftSrc := d217.Reg
					r282 := ctx.AllocRegExcept(d217.Reg)
					ctx.W.EmitMovRegReg(r282, d217.Reg)
					shiftSrc = r282
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d220.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d220.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d220.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d221)
				}
			}
			if d221.Loc == scm.LocReg && d217.Loc == scm.LocReg && d221.Reg == d217.Reg {
				ctx.TransferReg(d217.Reg)
				d217.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d217)
			ctx.FreeDesc(&d220)
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d222 scm.JITValueDesc
			if d202.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() | d221.Imm.Int())}
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
				ctx.BindReg(d221.Reg, &d222)
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				r283 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r283, d202.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d222)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else if d221.Loc == scm.LocImm {
				r284 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r284, d202.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r284, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitOrInt64(r284, scm.RegR11)
				}
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d222)
			} else {
				r285 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r285, d202.Reg)
				ctx.W.EmitOrInt64(r285, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r285}
				ctx.BindReg(r285, &d222)
			}
			if d222.Loc == scm.LocReg && d202.Loc == scm.LocReg && d222.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.EmitStoreToStack(d222, 40)
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl51)
			ctx.W.ResolveFixups()
			d223 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
			ctx.BindReg(r268, &d223)
			ctx.BindReg(r268, &d223)
			ctx.FreeDesc(&idxInt)
			if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair { ctx.EnsureDesc(&d223) }
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d223.Imm.Int()))))}
			} else {
				r286 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r286, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d224)
			}
			ctx.FreeDesc(&d223)
			var d225 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r287 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r287, thisptr.Reg, off)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r287}
				ctx.BindReg(r287, &d225)
			}
			if d224.Loc == scm.LocStack || d224.Loc == scm.LocStackPair { ctx.EnsureDesc(&d224) }
			if d225.Loc == scm.LocStack || d225.Loc == scm.LocStackPair { ctx.EnsureDesc(&d225) }
			var d226 scm.JITValueDesc
			if d224.Loc == scm.LocImm && d225.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d224.Imm.Int() + d225.Imm.Int())}
			} else if d225.Loc == scm.LocImm && d225.Imm.Int() == 0 {
				r288 := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(r288, d224.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r288}
				ctx.BindReg(r288, &d226)
			} else if d224.Loc == scm.LocImm && d224.Imm.Int() == 0 {
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d225.Reg}
				ctx.BindReg(d225.Reg, &d226)
			} else if d224.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d224.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d225.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d226)
			} else if d225.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(scratch, d224.Reg)
				if d225.Imm.Int() >= -2147483648 && d225.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d225.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d225.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d226)
			} else {
				r289 := ctx.AllocRegExcept(d224.Reg)
				ctx.W.EmitMovRegReg(r289, d224.Reg)
				ctx.W.EmitAddInt64(r289, d225.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r289}
				ctx.BindReg(r289, &d226)
			}
			if d226.Loc == scm.LocReg && d224.Loc == scm.LocReg && d226.Reg == d224.Reg {
				ctx.TransferReg(d224.Reg)
				d224.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d224)
			ctx.FreeDesc(&d225)
			var d227 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r290, fieldAddr)
				ctx.W.EmitMovRegMem64(r291, fieldAddr+8)
				d227 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d227)
				ctx.BindReg(r291, &d227)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r292 := ctx.AllocReg()
				r293 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r292, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r293, thisptr.Reg, off+8)
				d227 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r292, Reg2: r293}
				ctx.BindReg(r292, &d227)
				ctx.BindReg(r293, &d227)
			}
			var d228 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d227.StackOff))}
			} else {
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d227.Reg2}
				ctx.BindReg(d227.Reg2, &d228)
			}
			if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair { ctx.EnsureDesc(&d228) }
			if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair { ctx.EnsureDesc(&d226) }
			if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair { ctx.EnsureDesc(&d228) }
			var d230 scm.JITValueDesc
			if d226.Loc == scm.LocImm && d228.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d226.Imm.Int() >= d228.Imm.Int())}
			} else if d228.Loc == scm.LocImm {
				r294 := ctx.AllocRegExcept(d226.Reg)
				if d228.Imm.Int() >= -2147483648 && d228.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d226.Reg, int32(d228.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d228.Imm.Int()))
					ctx.W.EmitCmpInt64(d226.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r294, scm.CcGE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d230)
			} else if d226.Loc == scm.LocImm {
				r295 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d228.Reg)
				ctx.W.EmitSetcc(r295, scm.CcGE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r295}
				ctx.BindReg(r295, &d230)
			} else {
				r296 := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitCmpInt64(d226.Reg, d228.Reg)
				ctx.W.EmitSetcc(r296, scm.CcGE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r296}
				ctx.BindReg(r296, &d230)
			}
			ctx.FreeDesc(&d228)
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			if d230.Loc == scm.LocImm {
				if d230.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d230.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl59)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d230)
			ctx.W.MarkLabel(lbl58)
			if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair { ctx.EnsureDesc(&d226) }
			var d231 scm.JITValueDesc
			if d226.Loc == scm.LocImm {
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d226.Imm.Int() < 0)}
			} else {
				r297 := ctx.AllocRegExcept(d226.Reg)
				ctx.W.EmitCmpRegImm32(d226.Reg, 0)
				ctx.W.EmitSetcc(r297, scm.CcL)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r297}
				ctx.BindReg(r297, &d231)
			}
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d231.Loc == scm.LocImm {
				if d231.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl60)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d231.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl61)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d231)
			ctx.W.MarkLabel(lbl57)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl60)
			if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair { ctx.EnsureDesc(&d226) }
			r298 := ctx.AllocReg()
			if d226.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r298, uint64(d226.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r298, d226.Reg)
				ctx.W.EmitShlRegImm8(r298, 4)
			}
			if d227.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d227.Imm.Int()))
				ctx.W.EmitAddInt64(r298, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r298, d227.Reg)
			}
			r299 := ctx.AllocRegExcept(r298)
			r300 := ctx.AllocRegExcept(r298, r299)
			ctx.W.EmitMovRegMem(r299, r298, 0)
			ctx.W.EmitMovRegMem(r300, r298, 8)
			ctx.FreeReg(r298)
			d232 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r299, Reg2: r300}
			ctx.BindReg(r299, &d232)
			ctx.BindReg(r300, &d232)
			ctx.FreeDesc(&d226)
			d233 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d191}, 2)
			ctx.FreeDesc(&d191)
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			if d233.Loc == scm.LocStack || d233.Loc == scm.LocStackPair { ctx.EnsureDesc(&d233) }
			d234 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d232, d233}, 2)
			ctx.FreeDesc(&d232)
			d235 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d234}, 2)
			ctx.EmitMovPairToResult(&d235, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r5, int32(48))
			ctx.W.EmitAddRSP32(int32(48))
			return result
}

func (s *StoragePrefix) prepare() {
	// set up scan
	s.prefixes.prepare()
	s.values.prepare()
}
func (s *StoragePrefix) scan(i uint32, value scm.Scmer) {
	if value.IsNil() {
		s.values.scan(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.scan(i, scm.NewInt(int64(pfid)))
			s.values.scan(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) init(i uint32) {
	s.prefixes.init(i)
	s.values.init(i)
}
func (s *StoragePrefix) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values.build(i, scm.NewNil())
		return
	}
	v := scm.String(value)

	for pfid := len(s.prefixdictionary) - 1; pfid >= 0; pfid-- {
		if strings.HasPrefix(v, s.prefixdictionary[pfid]) {
			// learn the string stripped from its prefix
			s.prefixes.build(i, scm.NewInt(int64(pfid)))
			s.values.build(i, scm.NewString(v[len(s.prefixdictionary[pfid]):]))
			return
		}
	}
}
func (s *StoragePrefix) finish() {
	s.prefixes.finish()
	s.values.finish()
}
func (s *StoragePrefix) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	// TODO: if s.values proposes a StoragePrefix, build it into our cascade??
	return nil
}
