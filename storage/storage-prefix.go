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
import "unsafe"
import "strings"
import "github.com/launix-de/memcp/scm"

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
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
			} else {
				idxInt = idx
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			r0 := idxInt.Loc == scm.LocReg
			r1 := idxInt.Reg
			if r0 { ctx.ProtectReg(r1) }
			r2 := ctx.AllocReg()
			r3 := ctx.AllocReg()
			lbl1 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 232
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMem8(r4, fieldAddr)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 232)
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r5, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
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
			ctx.W.MarkLabel(lbl3)
			r6 := idxInt.Loc == scm.LocReg
			r7 := idxInt.Reg
			if r6 { ctx.ProtectReg(r7) }
			r8 := ctx.W.EmitSubRSP32Fixup()
			r9 := ctx.AllocReg()
			lbl5 := ctx.W.ReserveLabel()
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r10, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			}
			var d4 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d2.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d2.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d2.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(scratch, idxInt.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r11 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(r11, idxInt.Reg)
				ctx.W.EmitImulInt64(r11, d2.Reg)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
			}
			if d4.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d4.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d2)
			var d5 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r12, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			}
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r13, d4.Reg)
				ctx.W.EmitShrRegImm8(r13, 6)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
			}
			if d6.Loc == scm.LocReg && d4.Loc == scm.LocReg && d6.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			r14 := ctx.AllocReg()
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r14, uint64(d6.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r14, d6.Reg)
				ctx.W.EmitShlRegImm8(r14, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r14, d5.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.W.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d7 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.FreeDesc(&d6)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r16, d4.Reg)
				ctx.W.EmitAndRegImm32(r16, 63)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) << uint64(d8.Imm.Int())))}
			} else if d8.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d7.Reg, uint8(d8.Imm.Int()))
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d7.Reg}
			} else {
				{
					shiftSrc := d7.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
			}
			d14 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() - d12.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else if d14.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d12.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d12.Loc == scm.LocImm {
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d14.Reg, int32(d12.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitSubInt64(d14.Reg, scm.RegR11)
				}
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else {
				ctx.W.EmitSubInt64(d14.Reg, d12.Reg)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			}
			if d15.Loc == scm.LocReg && d14.Loc == scm.LocReg && d15.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			var d16 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d11.Imm.Int()) >> uint64(d15.Imm.Int())))}
			} else if d15.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d11.Reg, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d11.Reg}
			} else {
				{
					shiftSrc := d11.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d16.Loc == scm.LocReg && d11.Loc == scm.LocReg && d16.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d15)
			ctx.EmitMovToReg(r9, d16)
			ctx.W.EmitJmp(lbl5)
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl6)
			var d17 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r20 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r20, d4.Reg)
				ctx.W.EmitAndRegImm32(r20, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
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
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r21, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			}
			var d20 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() + d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d18.Loc == scm.LocImm {
				if d18.Imm.Int() >= -2147483648 && d18.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d17.Reg, int32(d18.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.W.EmitAddInt64(d17.Reg, scm.RegR11)
				}
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else {
				ctx.W.EmitAddInt64(d17.Reg, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			}
			if d20.Loc == scm.LocReg && d17.Loc == scm.LocReg && d20.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d18)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d20.Imm.Int() > 64)}
			} else {
				r22 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d20.Reg, 64)
				ctx.W.EmitSetcc(r22, scm.CcG)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r22}
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
			var d22 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r23 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r23, d4.Reg)
				ctx.W.EmitShrRegImm8(r23, 6)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			}
			if d22.Loc == scm.LocReg && d4.Loc == scm.LocReg && d22.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d22.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d22.Reg, int32(1))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d22.Reg}
			}
			if d23.Loc == scm.LocReg && d22.Loc == scm.LocReg && d23.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			r24 := ctx.AllocReg()
			if d23.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r24, uint64(d23.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r24, d23.Reg)
				ctx.W.EmitShlRegImm8(r24, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r24, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r24, d5.Reg)
			}
			r25 := ctx.AllocRegExcept(r24)
			ctx.W.EmitMovRegMem(r25, r24, 0)
			ctx.FreeReg(r24)
			d24 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
			ctx.FreeDesc(&d23)
			var d25 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d4.Reg, 63)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d4.Reg}
			}
			if d25.Loc == scm.LocReg && d4.Loc == scm.LocReg && d25.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d26 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d26.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d25.Loc == scm.LocImm {
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d26.Reg, int32(d25.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(d26.Reg, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			} else {
				ctx.W.EmitSubInt64(d26.Reg, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			var d28 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d24.Reg, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else {
				{
					shiftSrc := d24.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d28.Loc == scm.LocReg && d24.Loc == scm.LocReg && d28.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d27)
			var d29 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d28.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r26, d9.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d28.Loc == scm.LocImm {
				r27 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r27, d9.Reg)
				if d28.Imm.Int() >= -2147483648 && d28.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r27, int32(d28.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
					ctx.W.EmitOrInt64(r27, scratch)
					ctx.FreeReg(scratch)
				}
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
			} else {
				r28 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r28, d9.Reg)
				ctx.W.EmitOrInt64(r28, d28.Reg)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
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
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			if r6 { ctx.UnprotectReg(r7) }
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r29, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			}
			var d33 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d30.Reg, int32(d32.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.W.EmitAddInt64(d30.Reg, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else {
				ctx.W.EmitAddInt64(d30.Reg, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			}
			if d33.Loc == scm.LocReg && d30.Loc == scm.LocReg && d33.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d32)
			var d35 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r30 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r30, thisptr.Reg, off)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
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
			r31 := idxInt.Loc == scm.LocReg
			r32 := idxInt.Reg
			if r31 { ctx.ProtectReg(r32) }
			r33 := ctx.AllocReg()
			lbl14 := ctx.W.ReserveLabel()
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
			}
			var d39 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d37.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(scratch, idxInt.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r35 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(r35, idxInt.Reg)
				ctx.W.EmitImulInt64(r35, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
			}
			if d39.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d39.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				r36 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r36, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			}
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r37 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r37, d39.Reg)
				ctx.W.EmitShrRegImm8(r37, 6)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			r38 := ctx.AllocReg()
			if d41.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r38, uint64(d41.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r38, d41.Reg)
				ctx.W.EmitShlRegImm8(r38, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r38, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r38, d40.Reg)
			}
			r39 := ctx.AllocRegExcept(r38)
			ctx.W.EmitMovRegMem(r39, r38, 0)
			ctx.FreeReg(r38)
			d42 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r40 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r40, d39.Reg)
				ctx.W.EmitAndRegImm32(r40, 63)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
			}
			if d43.Loc == scm.LocReg && d39.Loc == scm.LocReg && d43.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d44 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d42.Imm.Int()) << uint64(d43.Imm.Int())))}
			} else if d43.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d42.Reg, uint8(d43.Imm.Int()))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
			} else {
				{
					shiftSrc := d42.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				r41 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r41, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
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
			r42 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r42, 8)
			ctx.ProtectReg(r42)
			d46 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r42}
			ctx.UnprotectReg(r42)
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r43, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
			}
			d49 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() - d47.Imm.Int())}
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d47.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d47.Loc == scm.LocImm {
				if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d49.Reg, int32(d47.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitSubInt64(d49.Reg, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			} else {
				ctx.W.EmitSubInt64(d49.Reg, d47.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			var d51 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d46.Imm.Int()) >> uint64(d50.Imm.Int())))}
			} else if d50.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d46.Reg, uint8(d50.Imm.Int()))
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
			} else {
				{
					shiftSrc := d46.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d51.Loc == scm.LocReg && d46.Loc == scm.LocReg && d51.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d50)
			ctx.EmitMovToReg(r33, d51)
			ctx.W.EmitJmp(lbl14)
			ctx.FreeDesc(&d51)
			ctx.W.MarkLabel(lbl15)
			var d52 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				r44 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r44, d39.Reg)
				ctx.W.EmitAndRegImm32(r44, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
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
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r45, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
			}
			var d55 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() + d53.Imm.Int())}
			} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d53.Reg}
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d53.Loc == scm.LocImm {
				if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d52.Reg, int32(d53.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(d52.Reg, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			} else {
				ctx.W.EmitAddInt64(d52.Reg, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d52.Reg}
			}
			if d55.Loc == scm.LocReg && d52.Loc == scm.LocReg && d55.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.FreeDesc(&d53)
			var d56 scm.JITValueDesc
			if d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d55.Imm.Int() > 64)}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d55.Reg, 64)
				ctx.W.EmitSetcc(r46, scm.CcG)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r46}
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
			var d57 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() / 64)}
			} else {
				r47 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r47, d39.Reg)
				ctx.W.EmitShrRegImm8(r47, 6)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
			}
			if d57.Loc == scm.LocReg && d39.Loc == scm.LocReg && d57.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d57.Reg, int32(1))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d57.Reg}
			}
			if d58.Loc == scm.LocReg && d57.Loc == scm.LocReg && d58.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			r48 := ctx.AllocReg()
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r48, uint64(d58.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r48, d58.Reg)
				ctx.W.EmitShlRegImm8(r48, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r48, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r48, d40.Reg)
			}
			r49 := ctx.AllocRegExcept(r48)
			ctx.W.EmitMovRegMem(r49, r48, 0)
			ctx.FreeReg(r48)
			d59 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
			ctx.FreeDesc(&d58)
			var d60 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d39.Reg, 63)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			}
			if d60.Loc == scm.LocReg && d39.Loc == scm.LocReg && d60.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			d61 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() - d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d61.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d60.Loc == scm.LocImm {
				if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d61.Reg, int32(d60.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
				ctx.W.EmitSubInt64(d61.Reg, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			} else {
				ctx.W.EmitSubInt64(d61.Reg, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			var d63 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) >> uint64(d62.Imm.Int())))}
			} else if d62.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d59.Reg, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
			} else {
				{
					shiftSrc := d59.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d63.Loc == scm.LocReg && d59.Loc == scm.LocReg && d63.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d62)
			var d64 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() | d63.Imm.Int())}
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d63.Reg}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r50 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r50, d44.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d63.Loc == scm.LocImm {
				r51 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r51, d44.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r51, int32(d63.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d63.Imm.Int()))
					ctx.W.EmitOrInt64(r51, scratch)
					ctx.FreeReg(scratch)
				}
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
			} else {
				r52 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r52, d44.Reg)
				ctx.W.EmitOrInt64(r52, d63.Reg)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
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
			d65 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			if r31 { ctx.UnprotectReg(r32) }
			var d67 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r53, thisptr.Reg, off)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
			}
			var d68 scm.JITValueDesc
			if d65.Loc == scm.LocImm && d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d65.Imm.Int() + d67.Imm.Int())}
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			} else if d65.Loc == scm.LocImm && d65.Imm.Int() == 0 {
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d67.Reg}
			} else if d65.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d65.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d67.Loc == scm.LocImm {
				if d67.Imm.Int() >= -2147483648 && d67.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d65.Reg, int32(d67.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d67.Imm.Int()))
				ctx.W.EmitAddInt64(d65.Reg, scm.RegR11)
				}
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			} else {
				ctx.W.EmitAddInt64(d65.Reg, d67.Reg)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d65.Reg}
			}
			if d68.Loc == scm.LocReg && d65.Loc == scm.LocReg && d68.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.FreeDesc(&d67)
			var d70 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r54, thisptr.Reg, off)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
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
			r55 := d33.Loc == scm.LocReg
			r56 := d33.Reg
			if r55 { ctx.ProtectReg(r56) }
			r57 := ctx.AllocReg()
			lbl23 := ctx.W.ReserveLabel()
			var d72 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r58, thisptr.Reg, off)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r58}
			}
			var d74 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() * d72.Imm.Int())}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(scratch, d33.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d72.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r59 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r59, d33.Reg)
				ctx.W.EmitImulInt64(r59, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
			}
			if d74.Loc == scm.LocReg && d33.Loc == scm.LocReg && d74.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			var d75 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r60 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r60, d74.Reg)
				ctx.W.EmitShrRegImm8(r60, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
			}
			if d75.Loc == scm.LocReg && d74.Loc == scm.LocReg && d75.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			r61 := ctx.AllocReg()
			if d75.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r61, uint64(d75.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r61, d75.Reg)
				ctx.W.EmitShlRegImm8(r61, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r61, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r61, d40.Reg)
			}
			r62 := ctx.AllocRegExcept(r61)
			ctx.W.EmitMovRegMem(r62, r61, 0)
			ctx.FreeReg(r61)
			d76 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r63 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r63, d74.Reg)
				ctx.W.EmitAndRegImm32(r63, 63)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
			}
			if d77.Loc == scm.LocReg && d74.Loc == scm.LocReg && d77.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d78 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d76.Imm.Int()) << uint64(d77.Imm.Int())))}
			} else if d77.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d76.Reg, uint8(d77.Imm.Int()))
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
			} else {
				{
					shiftSrc := d76.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d77.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d77.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d77.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d78.Loc == scm.LocReg && d76.Loc == scm.LocReg && d78.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			ctx.FreeDesc(&d77)
			var d79 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d79.Loc == scm.LocImm {
				if d79.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
					ctx.EmitStoreToStack(d78, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d79.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.EmitStoreToStack(d78, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d79)
			ctx.W.MarkLabel(lbl25)
			r65 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r65, 16)
			ctx.ProtectReg(r65)
			d80 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r65}
			ctx.UnprotectReg(r65)
			var d81 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r66, thisptr.Reg, off)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r66}
			}
			d83 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d83.Imm.Int() - d81.Imm.Int())}
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d83.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d81.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d81.Loc == scm.LocImm {
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d83.Reg, int32(d81.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
				ctx.W.EmitSubInt64(d83.Reg, scm.RegR11)
				}
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			} else {
				ctx.W.EmitSubInt64(d83.Reg, d81.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d83.Reg}
			}
			if d84.Loc == scm.LocReg && d83.Loc == scm.LocReg && d84.Reg == d83.Reg {
				ctx.TransferReg(d83.Reg)
				d83.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			var d85 scm.JITValueDesc
			if d80.Loc == scm.LocImm && d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d80.Imm.Int()) >> uint64(d84.Imm.Int())))}
			} else if d84.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d80.Reg, uint8(d84.Imm.Int()))
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d80.Reg}
			} else {
				{
					shiftSrc := d80.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d84.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d84.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d84.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d85.Loc == scm.LocReg && d80.Loc == scm.LocReg && d85.Reg == d80.Reg {
				ctx.TransferReg(d80.Reg)
				d80.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d80)
			ctx.FreeDesc(&d84)
			ctx.EmitMovToReg(r57, d85)
			ctx.W.EmitJmp(lbl23)
			ctx.FreeDesc(&d85)
			ctx.W.MarkLabel(lbl24)
			var d86 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				r67 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r67, d74.Reg)
				ctx.W.EmitAndRegImm32(r67, 63)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
			}
			if d86.Loc == scm.LocReg && d74.Loc == scm.LocReg && d86.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d87 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r68 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r68, thisptr.Reg, off)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			}
			var d89 scm.JITValueDesc
			if d86.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d86.Imm.Int() + d87.Imm.Int())}
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d87.Reg}
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d86.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d87.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d87.Loc == scm.LocImm {
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d86.Reg, int32(d87.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(d86.Reg, scm.RegR11)
				}
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			} else {
				ctx.W.EmitAddInt64(d86.Reg, d87.Reg)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
			}
			if d89.Loc == scm.LocReg && d86.Loc == scm.LocReg && d89.Reg == d86.Reg {
				ctx.TransferReg(d86.Reg)
				d86.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d86)
			ctx.FreeDesc(&d87)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d89.Imm.Int() > 64)}
			} else {
				r69 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d89.Reg, 64)
				ctx.W.EmitSetcc(r69, scm.CcG)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r69}
			}
			ctx.FreeDesc(&d89)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			if d90.Loc == scm.LocImm {
				if d90.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.EmitStoreToStack(d78, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d90.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
				ctx.EmitStoreToStack(d78, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d90)
			ctx.W.MarkLabel(lbl27)
			var d91 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() / 64)}
			} else {
				r70 := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegReg(r70, d74.Reg)
				ctx.W.EmitShrRegImm8(r70, 6)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r70}
			}
			if d91.Loc == scm.LocReg && d74.Loc == scm.LocReg && d91.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d91.Reg, int32(1))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d91.Reg}
			}
			if d92.Loc == scm.LocReg && d91.Loc == scm.LocReg && d92.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			r71 := ctx.AllocReg()
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r71, uint64(d92.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r71, d92.Reg)
				ctx.W.EmitShlRegImm8(r71, 3)
			}
			if d40.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitAddInt64(r71, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r71, d40.Reg)
			}
			r72 := ctx.AllocRegExcept(r71)
			ctx.W.EmitMovRegMem(r72, r71, 0)
			ctx.FreeReg(r71)
			d93 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			if d74.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d74.Reg, 63)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d74.Reg}
			}
			if d94.Loc == scm.LocReg && d74.Loc == scm.LocReg && d94.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			d95 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d96 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d94.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() - d94.Imm.Int())}
			} else if d94.Loc == scm.LocImm && d94.Imm.Int() == 0 {
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else if d95.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d95.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d94.Reg)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d94.Loc == scm.LocImm {
				if d94.Imm.Int() >= -2147483648 && d94.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d95.Reg, int32(d94.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
				ctx.W.EmitSubInt64(d95.Reg, scm.RegR11)
				}
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			} else {
				ctx.W.EmitSubInt64(d95.Reg, d94.Reg)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d95.Reg}
			}
			if d96.Loc == scm.LocReg && d95.Loc == scm.LocReg && d96.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d94)
			var d97 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d96.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) >> uint64(d96.Imm.Int())))}
			} else if d96.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d93.Reg, uint8(d96.Imm.Int()))
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d93.Reg}
			} else {
				{
					shiftSrc := d93.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d96.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d96.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d96.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d97.Loc == scm.LocReg && d93.Loc == scm.LocReg && d97.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			ctx.FreeDesc(&d96)
			var d98 scm.JITValueDesc
			if d78.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d78.Imm.Int() | d97.Imm.Int())}
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d97.Reg}
			} else if d97.Loc == scm.LocImm && d97.Imm.Int() == 0 {
				r73 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r73, d78.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d78.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d97.Loc == scm.LocImm {
				r74 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r74, d78.Reg)
				if d97.Imm.Int() >= -2147483648 && d97.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r74, int32(d97.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d97.Imm.Int()))
					ctx.W.EmitOrInt64(r74, scratch)
					ctx.FreeReg(scratch)
				}
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
			} else {
				r75 := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegReg(r75, d78.Reg)
				ctx.W.EmitOrInt64(r75, d97.Reg)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
			}
			if d98.Loc == scm.LocReg && d78.Loc == scm.LocReg && d98.Reg == d78.Reg {
				ctx.TransferReg(d78.Reg)
				d78.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d97)
			ctx.EmitStoreToStack(d98, 16)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			d99 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			if r55 { ctx.UnprotectReg(r56) }
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r76 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r76, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r76}
			}
			var d102 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() + d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d101.Loc == scm.LocImm {
				if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d99.Reg, int32(d101.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
				ctx.W.EmitAddInt64(d99.Reg, scm.RegR11)
				}
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			} else {
				ctx.W.EmitAddInt64(d99.Reg, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d99.Reg}
			}
			if d102.Loc == scm.LocReg && d99.Loc == scm.LocReg && d102.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			ctx.FreeDesc(&d101)
			r77 := d33.Loc == scm.LocReg
			r78 := d33.Reg
			if r77 { ctx.ProtectReg(r78) }
			r79 := ctx.AllocReg()
			lbl29 := ctx.W.ReserveLabel()
			var d104 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r80 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r80, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r80}
			}
			var d106 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() * d104.Imm.Int())}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(scratch, d33.Reg)
				if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d104.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r81 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r81, d33.Reg)
				ctx.W.EmitImulInt64(r81, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
			}
			if d106.Loc == scm.LocReg && d33.Loc == scm.LocReg && d106.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				r82 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r82, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r82}
			}
			var d108 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() / 64)}
			} else {
				r83 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r83, d106.Reg)
				ctx.W.EmitShrRegImm8(r83, 6)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
			}
			if d108.Loc == scm.LocReg && d106.Loc == scm.LocReg && d108.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			r84 := ctx.AllocReg()
			if d108.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r84, uint64(d108.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r84, d108.Reg)
				ctx.W.EmitShlRegImm8(r84, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r84, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r84, d107.Reg)
			}
			r85 := ctx.AllocRegExcept(r84)
			ctx.W.EmitMovRegMem(r85, r84, 0)
			ctx.FreeReg(r84)
			d109 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r85}
			ctx.FreeDesc(&d108)
			var d110 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r86 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r86, d106.Reg)
				ctx.W.EmitAndRegImm32(r86, 63)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
			}
			if d110.Loc == scm.LocReg && d106.Loc == scm.LocReg && d110.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d111 scm.JITValueDesc
			if d109.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d109.Imm.Int()) << uint64(d110.Imm.Int())))}
			} else if d110.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d109.Reg, uint8(d110.Imm.Int()))
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d109.Reg}
			} else {
				{
					shiftSrc := d109.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d110.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d110.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d110.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d111.Loc == scm.LocReg && d109.Loc == scm.LocReg && d111.Reg == d109.Reg {
				ctx.TransferReg(d109.Reg)
				d109.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			ctx.FreeDesc(&d110)
			var d112 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r87, thisptr.Reg, off)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d112.Loc == scm.LocImm {
				if d112.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.EmitStoreToStack(d111, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d112.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.EmitStoreToStack(d111, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d112)
			ctx.W.MarkLabel(lbl31)
			r88 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r88, 24)
			ctx.ProtectReg(r88)
			d113 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r88}
			ctx.UnprotectReg(r88)
			var d114 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r89, thisptr.Reg, off)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
			}
			d116 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d114.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() - d114.Imm.Int())}
			} else if d114.Loc == scm.LocImm && d114.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d114.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d114.Loc == scm.LocImm {
				if d114.Imm.Int() >= -2147483648 && d114.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d116.Reg, int32(d114.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d114.Imm.Int()))
				ctx.W.EmitSubInt64(d116.Reg, scm.RegR11)
				}
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			} else {
				ctx.W.EmitSubInt64(d116.Reg, d114.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d114)
			var d118 scm.JITValueDesc
			if d113.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d113.Imm.Int()) >> uint64(d117.Imm.Int())))}
			} else if d117.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d113.Reg, uint8(d117.Imm.Int()))
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d113.Reg}
			} else {
				{
					shiftSrc := d113.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d118.Loc == scm.LocReg && d113.Loc == scm.LocReg && d118.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			ctx.FreeDesc(&d117)
			ctx.EmitMovToReg(r79, d118)
			ctx.W.EmitJmp(lbl29)
			ctx.FreeDesc(&d118)
			ctx.W.MarkLabel(lbl30)
			var d119 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				r90 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r90, d106.Reg)
				ctx.W.EmitAndRegImm32(r90, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
			}
			if d119.Loc == scm.LocReg && d106.Loc == scm.LocReg && d119.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
			}
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d120.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + d120.Imm.Int())}
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d120.Reg}
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d120.Loc == scm.LocImm {
				if d120.Imm.Int() >= -2147483648 && d120.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d119.Reg, int32(d120.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d120.Imm.Int()))
				ctx.W.EmitAddInt64(d119.Reg, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			} else {
				ctx.W.EmitAddInt64(d119.Reg, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d120)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d122.Imm.Int() > 64)}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d122.Reg, 64)
				ctx.W.EmitSetcc(r92, scm.CcG)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r92}
			}
			ctx.FreeDesc(&d122)
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d123.Loc == scm.LocImm {
				if d123.Imm.Bool() {
					ctx.W.EmitJmp(lbl33)
				} else {
					ctx.EmitStoreToStack(d111, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d123.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.EmitStoreToStack(d111, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d123)
			ctx.W.MarkLabel(lbl33)
			var d124 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() / 64)}
			} else {
				r93 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r93, d106.Reg)
				ctx.W.EmitShrRegImm8(r93, 6)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r93}
			}
			if d124.Loc == scm.LocReg && d106.Loc == scm.LocReg && d124.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			var d125 scm.JITValueDesc
			if d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d124.Reg, int32(1))
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
			}
			if d125.Loc == scm.LocReg && d124.Loc == scm.LocReg && d125.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			r94 := ctx.AllocReg()
			if d125.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r94, uint64(d125.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r94, d125.Reg)
				ctx.W.EmitShlRegImm8(r94, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r94, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r94, d107.Reg)
			}
			r95 := ctx.AllocRegExcept(r94)
			ctx.W.EmitMovRegMem(r95, r94, 0)
			ctx.FreeReg(r94)
			d126 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d106.Reg, 63)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d106.Reg}
			}
			if d127.Loc == scm.LocReg && d106.Loc == scm.LocReg && d127.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			d128 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d129 scm.JITValueDesc
			if d128.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d128.Imm.Int() - d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d128.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d127.Loc == scm.LocImm {
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d128.Reg, int32(d127.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
				ctx.W.EmitSubInt64(d128.Reg, scm.RegR11)
				}
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			} else {
				ctx.W.EmitSubInt64(d128.Reg, d127.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
			}
			if d129.Loc == scm.LocReg && d128.Loc == scm.LocReg && d129.Reg == d128.Reg {
				ctx.TransferReg(d128.Reg)
				d128.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			var d130 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d126.Imm.Int()) >> uint64(d129.Imm.Int())))}
			} else if d129.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d126.Reg, uint8(d129.Imm.Int()))
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d126.Reg}
			} else {
				{
					shiftSrc := d126.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d129.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d129.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d129.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d130.Loc == scm.LocReg && d126.Loc == scm.LocReg && d130.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d129)
			var d131 scm.JITValueDesc
			if d111.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d111.Imm.Int() | d130.Imm.Int())}
			} else if d111.Loc == scm.LocImm && d111.Imm.Int() == 0 {
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d130.Reg}
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				r96 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r96, d111.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
			} else if d111.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d111.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d130.Loc == scm.LocImm {
				r97 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r97, d111.Reg)
				if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r97, int32(d130.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d130.Imm.Int()))
					ctx.W.EmitOrInt64(r97, scratch)
					ctx.FreeReg(scratch)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
			} else {
				r98 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r98, d111.Reg)
				ctx.W.EmitOrInt64(r98, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
			}
			if d131.Loc == scm.LocReg && d111.Loc == scm.LocReg && d131.Reg == d111.Reg {
				ctx.TransferReg(d111.Reg)
				d111.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d130)
			ctx.EmitStoreToStack(d131, 24)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d132 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			if r77 { ctx.UnprotectReg(r78) }
			var d134 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r99 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r99, thisptr.Reg, off)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r99}
			}
			var d135 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() + d134.Imm.Int())}
			} else if d134.Loc == scm.LocImm && d134.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d134.Reg}
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d134.Loc == scm.LocImm {
				if d134.Imm.Int() >= -2147483648 && d134.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d132.Reg, int32(d134.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d134.Imm.Int()))
				ctx.W.EmitAddInt64(d132.Reg, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			} else {
				ctx.W.EmitAddInt64(d132.Reg, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d132.Reg}
			}
			if d135.Loc == scm.LocReg && d132.Loc == scm.LocReg && d135.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d132)
			ctx.FreeDesc(&d134)
			var d137 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d102.Imm.Int() + d135.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				r100 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r100, d102.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r100}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d135.Reg}
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(scratch, d102.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r101 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r101, d102.Reg)
				ctx.W.EmitAddInt64(r101, d135.Reg)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
			}
			if d137.Loc == scm.LocReg && d102.Loc == scm.LocReg && d137.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d135)
			var d139 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r102 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r102, fieldAddr)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r102}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r103 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r103, thisptr.Reg, off)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			}
			var d141 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() - d102.Imm.Int())}
			} else {
				r104 := ctx.AllocReg()
				if d137.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r104, uint64(d137.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r104, d137.Reg)
				}
				if d102.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
					ctx.W.EmitSubInt64(r104, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitSubInt64(r104, d102.Reg)
				}
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
			}
			var d142 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + d102.Imm.Int())}
			} else {
				r105 := ctx.AllocReg()
				if d139.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r105, uint64(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r105, d139.Reg)
				}
				if d102.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d102.Imm.Int()))
					ctx.W.EmitAddInt64(r105, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitAddInt64(r105, d102.Reg)
				}
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
			}
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewInt(d142.Imm.Int())}
				_ = d141
			} else {
				r106 := ctx.AllocReg()
				r107 := ctx.AllocReg()
				if d142.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r106, uint64(d142.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r106, d142.Reg)
					ctx.FreeReg(d142.Reg)
				}
				if d141.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r107, uint64(d141.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r107, d141.Reg)
					ctx.FreeReg(d141.Reg)
				}
				d143 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r106, Reg2: r107}
			}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d137)
			d144 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
			d145 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d143}, 2)
			ctx.EmitMovPairToResult(&d145, &d144)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl11)
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r108, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
			}
			var d148 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d146.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d33.Imm.Int() == d146.Imm.Int())}
			} else if d146.Loc == scm.LocImm {
				r109 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d33.Reg, int32(d146.Imm.Int()))
				ctx.W.EmitSetcc(r109, scm.CcE)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r109}
			} else if d33.Loc == scm.LocImm {
				r110 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d146.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r110, scm.CcE)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r110}
			} else {
				r111 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d33.Reg, d146.Reg)
				ctx.W.EmitSetcc(r111, scm.CcE)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r111}
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d146)
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d148.Loc == scm.LocImm {
				if d148.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d148.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d148)
			ctx.W.MarkLabel(lbl21)
			r112 := idxInt.Loc == scm.LocReg
			r113 := idxInt.Reg
			if r112 { ctx.ProtectReg(r113) }
			r114 := ctx.AllocReg()
			lbl37 := ctx.W.ReserveLabel()
			var d150 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r115, thisptr.Reg, off)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r115}
			}
			var d152 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d150.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d150.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d150.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(scratch, idxInt.Reg)
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d150.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r116 := ctx.AllocRegExcept(idxInt.Reg)
				ctx.W.EmitMovRegReg(r116, idxInt.Reg)
				ctx.W.EmitImulInt64(r116, d150.Reg)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
			}
			if d152.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d152.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			var d153 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() / 64)}
			} else {
				r117 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r117, d152.Reg)
				ctx.W.EmitShrRegImm8(r117, 6)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
			}
			if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			r118 := ctx.AllocReg()
			if d153.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r118, uint64(d153.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r118, d153.Reg)
				ctx.W.EmitShlRegImm8(r118, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r118, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r118, d107.Reg)
			}
			r119 := ctx.AllocRegExcept(r118)
			ctx.W.EmitMovRegMem(r119, r118, 0)
			ctx.FreeReg(r118)
			d154 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r119}
			ctx.FreeDesc(&d153)
			var d155 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() % 64)}
			} else {
				r120 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r120, d152.Reg)
				ctx.W.EmitAndRegImm32(r120, 63)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r120}
			}
			if d155.Loc == scm.LocReg && d152.Loc == scm.LocReg && d155.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			var d156 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d154.Imm.Int()) << uint64(d155.Imm.Int())))}
			} else if d155.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d154.Reg, uint8(d155.Imm.Int()))
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d154.Reg}
			} else {
				{
					shiftSrc := d154.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d155.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d155.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d155.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d156.Loc == scm.LocReg && d154.Loc == scm.LocReg && d156.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.FreeDesc(&d155)
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r121 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r121, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d157.Loc == scm.LocImm {
				if d157.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
					ctx.EmitStoreToStack(d156, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d157.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
				ctx.EmitStoreToStack(d156, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d157)
			ctx.W.MarkLabel(lbl39)
			r122 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r122, 32)
			ctx.ProtectReg(r122)
			d158 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r122}
			ctx.UnprotectReg(r122)
			var d159 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r123 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r123, thisptr.Reg, off)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
			}
			d161 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm && d159.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d161.Imm.Int() - d159.Imm.Int())}
			} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
			} else if d161.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d161.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d159.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d159.Loc == scm.LocImm {
				if d159.Imm.Int() >= -2147483648 && d159.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d161.Reg, int32(d159.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d159.Imm.Int()))
				ctx.W.EmitSubInt64(d161.Reg, scm.RegR11)
				}
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
			} else {
				ctx.W.EmitSubInt64(d161.Reg, d159.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
			}
			if d162.Loc == scm.LocReg && d161.Loc == scm.LocReg && d162.Reg == d161.Reg {
				ctx.TransferReg(d161.Reg)
				d161.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d159)
			var d163 scm.JITValueDesc
			if d158.Loc == scm.LocImm && d162.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d158.Imm.Int()) >> uint64(d162.Imm.Int())))}
			} else if d162.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d158.Reg, uint8(d162.Imm.Int()))
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
			} else {
				{
					shiftSrc := d158.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d162.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d162.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d162.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d163.Loc == scm.LocReg && d158.Loc == scm.LocReg && d163.Reg == d158.Reg {
				ctx.TransferReg(d158.Reg)
				d158.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			ctx.FreeDesc(&d162)
			ctx.EmitMovToReg(r114, d163)
			ctx.W.EmitJmp(lbl37)
			ctx.FreeDesc(&d163)
			ctx.W.MarkLabel(lbl38)
			var d164 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() % 64)}
			} else {
				r124 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r124, d152.Reg)
				ctx.W.EmitAndRegImm32(r124, 63)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
			}
			if d164.Loc == scm.LocReg && d152.Loc == scm.LocReg && d164.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			var d165 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r125 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r125, thisptr.Reg, off)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r125}
			}
			var d167 scm.JITValueDesc
			if d164.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + d165.Imm.Int())}
			} else if d165.Loc == scm.LocImm && d165.Imm.Int() == 0 {
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d165.Reg}
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d164.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d165.Reg)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d165.Loc == scm.LocImm {
				if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d164.Reg, int32(d165.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
				ctx.W.EmitAddInt64(d164.Reg, scm.RegR11)
				}
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			} else {
				ctx.W.EmitAddInt64(d164.Reg, d165.Reg)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
			}
			if d167.Loc == scm.LocReg && d164.Loc == scm.LocReg && d167.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.FreeDesc(&d165)
			var d168 scm.JITValueDesc
			if d167.Loc == scm.LocImm {
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d167.Imm.Int() > 64)}
			} else {
				r126 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d167.Reg, 64)
				ctx.W.EmitSetcc(r126, scm.CcG)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r126}
			}
			ctx.FreeDesc(&d167)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d168.Loc == scm.LocImm {
				if d168.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.EmitStoreToStack(d156, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d168.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
				ctx.EmitStoreToStack(d156, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d168)
			ctx.W.MarkLabel(lbl41)
			var d169 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() / 64)}
			} else {
				r127 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r127, d152.Reg)
				ctx.W.EmitShrRegImm8(r127, 6)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
			}
			if d169.Loc == scm.LocReg && d152.Loc == scm.LocReg && d169.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d169.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d169.Reg, int32(1))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d169.Reg}
			}
			if d170.Loc == scm.LocReg && d169.Loc == scm.LocReg && d170.Reg == d169.Reg {
				ctx.TransferReg(d169.Reg)
				d169.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d169)
			r128 := ctx.AllocReg()
			if d170.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r128, uint64(d170.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r128, d170.Reg)
				ctx.W.EmitShlRegImm8(r128, 3)
			}
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
				ctx.W.EmitAddInt64(r128, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r128, d107.Reg)
			}
			r129 := ctx.AllocRegExcept(r128)
			ctx.W.EmitMovRegMem(r129, r128, 0)
			ctx.FreeReg(r128)
			d171 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r129}
			ctx.FreeDesc(&d170)
			var d172 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d152.Reg, 63)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d152.Reg}
			}
			if d172.Loc == scm.LocReg && d152.Loc == scm.LocReg && d172.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			d173 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() - d172.Imm.Int())}
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d172.Loc == scm.LocImm {
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d173.Reg, int32(d172.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
				ctx.W.EmitSubInt64(d173.Reg, scm.RegR11)
				}
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
			} else {
				ctx.W.EmitSubInt64(d173.Reg, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d173.Reg}
			}
			if d174.Loc == scm.LocReg && d173.Loc == scm.LocReg && d174.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			var d175 scm.JITValueDesc
			if d171.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d171.Imm.Int()) >> uint64(d174.Imm.Int())))}
			} else if d174.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d171.Reg, uint8(d174.Imm.Int()))
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
			} else {
				{
					shiftSrc := d171.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d174.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d174.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d174.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d175.Loc == scm.LocReg && d171.Loc == scm.LocReg && d175.Reg == d171.Reg {
				ctx.TransferReg(d171.Reg)
				d171.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d171)
			ctx.FreeDesc(&d174)
			var d176 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() | d175.Imm.Int())}
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				r130 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r130, d156.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d175.Loc == scm.LocImm {
				r131 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r131, d156.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r131, int32(d175.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d175.Imm.Int()))
					ctx.W.EmitOrInt64(r131, scratch)
					ctx.FreeReg(scratch)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
			} else {
				r132 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r132, d156.Reg)
				ctx.W.EmitOrInt64(r132, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
			}
			if d176.Loc == scm.LocReg && d156.Loc == scm.LocReg && d176.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d175)
			ctx.EmitStoreToStack(d176, 32)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d177 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
			if r112 { ctx.UnprotectReg(r113) }
			var d179 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r133, thisptr.Reg, off)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
			}
			var d180 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() + d179.Imm.Int())}
			} else if d179.Loc == scm.LocImm && d179.Imm.Int() == 0 {
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d179.Reg}
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d179.Loc == scm.LocImm {
				if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d177.Reg, int32(d179.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
				ctx.W.EmitAddInt64(d177.Reg, scm.RegR11)
				}
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			} else {
				ctx.W.EmitAddInt64(d177.Reg, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d177.Reg}
			}
			if d180.Loc == scm.LocReg && d177.Loc == scm.LocReg && d180.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d179)
			var d183 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d68.Imm.Int() + d180.Imm.Int())}
			} else if d180.Loc == scm.LocImm && d180.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r134, d68.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d180.Reg}
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d180.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d180.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(scratch, d68.Reg)
				if d180.Imm.Int() >= -2147483648 && d180.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d180.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d180.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r135 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r135, d68.Reg)
				ctx.W.EmitAddInt64(r135, d180.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
			}
			if d183.Loc == scm.LocReg && d68.Loc == scm.LocReg && d183.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			var d186 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d183.Imm.Int() - d68.Imm.Int())}
			} else {
				r136 := ctx.AllocReg()
				if d183.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r136, uint64(d183.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r136, d183.Reg)
				}
				if d68.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
					ctx.W.EmitSubInt64(r136, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitSubInt64(r136, d68.Reg)
				}
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
			}
			var d187 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() + d68.Imm.Int())}
			} else {
				r137 := ctx.AllocReg()
				if d139.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r137, uint64(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r137, d139.Reg)
				}
				if d68.Loc == scm.LocImm {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
					ctx.W.EmitAddInt64(r137, scratch)
					ctx.FreeReg(scratch)
				} else {
					ctx.W.EmitAddInt64(r137, d68.Reg)
				}
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r137}
			}
			var d188 scm.JITValueDesc
			if d187.Loc == scm.LocImm && d186.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagString, Imm: scm.NewInt(d187.Imm.Int())}
				_ = d186
			} else {
				r138 := ctx.AllocReg()
				r139 := ctx.AllocReg()
				if d187.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r138, uint64(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r138, d187.Reg)
					ctx.FreeReg(d187.Reg)
				}
				if d186.Loc == scm.LocImm {
					ctx.W.EmitMovRegImm64(r139, uint64(d186.Imm.Int()))
				} else {
					ctx.W.EmitMovRegReg(r139, d186.Reg)
					ctx.FreeReg(d186.Reg)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r138, Reg2: r139}
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d183)
			d189 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
			d190 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d188}, 2)
			ctx.EmitMovPairToResult(&d190, &d189)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl20)
			var d191 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
			}
			var d192 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d191.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d68.Imm.Int() == d191.Imm.Int())}
			} else if d191.Loc == scm.LocImm {
				r141 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d68.Reg, int32(d191.Imm.Int()))
				ctx.W.EmitSetcc(r141, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r141}
			} else if d68.Loc == scm.LocImm {
				r142 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d68.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d191.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r142, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r142}
			} else {
				r143 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d68.Reg, d191.Reg)
				ctx.W.EmitSetcc(r143, scm.CcE)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r143}
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d191)
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d192.Loc == scm.LocImm {
				if d192.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d192.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl44)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl44)
				ctx.W.EmitJmp(lbl43)
			}
			ctx.FreeDesc(&d192)
			ctx.W.MarkLabel(lbl35)
			d193 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
			ctx.W.EmitMakeNil(d193)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl43)
			d194 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
			ctx.W.EmitMakeNil(d194)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d195 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r2, Reg2: r3}
			if r0 { ctx.UnprotectReg(r1) }
			d196 := ctx.EmitTagEquals(&d195, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d196.Loc == scm.LocImm {
				if d196.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d196.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d196)
			ctx.W.MarkLabel(lbl46)
			d197 := ctx.EmitTagEquals(&d195, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			if d197.Loc == scm.LocImm {
				if d197.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d197.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d197)
			ctx.W.MarkLabel(lbl45)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl49)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl48)
			r144 := ctx.AllocReg()
			lbl51 := ctx.W.ReserveLabel()
			var d199 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r145 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r145, thisptr.Reg, off)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			}
			var d201 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm && d199.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idxInt.Imm.Int() * d199.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(idxInt.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d199.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d199.Loc == scm.LocImm {
				if d199.Imm.Int() >= -2147483648 && d199.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(idxInt.Reg, int32(d199.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d199.Imm.Int()))
				ctx.W.EmitImulInt64(idxInt.Reg, scm.RegR11)
				}
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			} else {
				ctx.W.EmitImulInt64(idxInt.Reg, d199.Reg)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idxInt.Reg}
			}
			if d201.Loc == scm.LocReg && idxInt.Loc == scm.LocReg && d201.Reg == idxInt.Reg {
				ctx.TransferReg(idxInt.Reg)
				idxInt.Loc = scm.LocNone
			}
			ctx.FreeDesc(&idxInt)
			ctx.FreeDesc(&d199)
			var d202 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r146, thisptr.Reg, off)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r146}
			}
			var d203 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() / 64)}
			} else {
				r147 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r147, d201.Reg)
				ctx.W.EmitShrRegImm8(r147, 6)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
			}
			if d203.Loc == scm.LocReg && d201.Loc == scm.LocReg && d203.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			r148 := ctx.AllocReg()
			if d203.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r148, uint64(d203.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r148, d203.Reg)
				ctx.W.EmitShlRegImm8(r148, 3)
			}
			if d202.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
				ctx.W.EmitAddInt64(r148, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r148, d202.Reg)
			}
			r149 := ctx.AllocRegExcept(r148)
			ctx.W.EmitMovRegMem(r149, r148, 0)
			ctx.FreeReg(r148)
			d204 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
			ctx.FreeDesc(&d203)
			var d205 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() % 64)}
			} else {
				r150 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r150, d201.Reg)
				ctx.W.EmitAndRegImm32(r150, 63)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
			}
			if d205.Loc == scm.LocReg && d201.Loc == scm.LocReg && d205.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			var d206 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d204.Imm.Int()) << uint64(d205.Imm.Int())))}
			} else if d205.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d204.Reg, uint8(d205.Imm.Int()))
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d204.Reg}
			} else {
				{
					shiftSrc := d204.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d205.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d205.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d205.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d206.Loc == scm.LocReg && d204.Loc == scm.LocReg && d206.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d205)
			var d207 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r151, thisptr.Reg, off)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r151}
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d207.Loc == scm.LocImm {
				if d207.Imm.Bool() {
					ctx.W.EmitJmp(lbl52)
				} else {
					ctx.EmitStoreToStack(d206, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d207.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl54)
				ctx.EmitStoreToStack(d206, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d207)
			ctx.W.MarkLabel(lbl53)
			r152 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r152, 40)
			ctx.ProtectReg(r152)
			d208 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r152}
			ctx.UnprotectReg(r152)
			var d209 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r153 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r153, thisptr.Reg, off)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
			}
			d211 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm && d209.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d211.Imm.Int() - d209.Imm.Int())}
			} else if d209.Loc == scm.LocImm && d209.Imm.Int() == 0 {
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d211.Reg}
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d211.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d209.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d209.Loc == scm.LocImm {
				if d209.Imm.Int() >= -2147483648 && d209.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d211.Reg, int32(d209.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d209.Imm.Int()))
				ctx.W.EmitSubInt64(d211.Reg, scm.RegR11)
				}
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d211.Reg}
			} else {
				ctx.W.EmitSubInt64(d211.Reg, d209.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d211.Reg}
			}
			if d212.Loc == scm.LocReg && d211.Loc == scm.LocReg && d212.Reg == d211.Reg {
				ctx.TransferReg(d211.Reg)
				d211.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d209)
			var d213 scm.JITValueDesc
			if d208.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d208.Imm.Int()) >> uint64(d212.Imm.Int())))}
			} else if d212.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d208.Reg, uint8(d212.Imm.Int()))
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
			} else {
				{
					shiftSrc := d208.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d213.Loc == scm.LocReg && d208.Loc == scm.LocReg && d213.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.FreeDesc(&d212)
			ctx.EmitMovToReg(r144, d213)
			ctx.W.EmitJmp(lbl51)
			ctx.FreeDesc(&d213)
			ctx.W.MarkLabel(lbl52)
			var d214 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() % 64)}
			} else {
				r154 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r154, d201.Reg)
				ctx.W.EmitAndRegImm32(r154, 63)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
			}
			if d214.Loc == scm.LocReg && d201.Loc == scm.LocReg && d214.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			var d215 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r155 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r155, thisptr.Reg, off)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r155}
			}
			var d217 scm.JITValueDesc
			if d214.Loc == scm.LocImm && d215.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d214.Imm.Int() + d215.Imm.Int())}
			} else if d215.Loc == scm.LocImm && d215.Imm.Int() == 0 {
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d215.Reg}
			} else if d214.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d214.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d215.Loc == scm.LocImm {
				if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d214.Reg, int32(d215.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
				ctx.W.EmitAddInt64(d214.Reg, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
			} else {
				ctx.W.EmitAddInt64(d214.Reg, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
			}
			if d217.Loc == scm.LocReg && d214.Loc == scm.LocReg && d217.Reg == d214.Reg {
				ctx.TransferReg(d214.Reg)
				d214.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.FreeDesc(&d215)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d217.Imm.Int() > 64)}
			} else {
				r156 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d217.Reg, 64)
				ctx.W.EmitSetcc(r156, scm.CcG)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r156}
			}
			ctx.FreeDesc(&d217)
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d218.Loc == scm.LocImm {
				if d218.Imm.Bool() {
					ctx.W.EmitJmp(lbl55)
				} else {
					ctx.EmitStoreToStack(d206, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d218.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
				ctx.EmitStoreToStack(d206, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
			}
			ctx.FreeDesc(&d218)
			ctx.W.MarkLabel(lbl55)
			var d219 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() / 64)}
			} else {
				r157 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r157, d201.Reg)
				ctx.W.EmitShrRegImm8(r157, 6)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
			}
			if d219.Loc == scm.LocReg && d201.Loc == scm.LocReg && d219.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			var d220 scm.JITValueDesc
			if d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d219.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d219.Reg, int32(1))
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d219.Reg}
			}
			if d220.Loc == scm.LocReg && d219.Loc == scm.LocReg && d220.Reg == d219.Reg {
				ctx.TransferReg(d219.Reg)
				d219.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d219)
			r158 := ctx.AllocReg()
			if d220.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r158, uint64(d220.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r158, d220.Reg)
				ctx.W.EmitShlRegImm8(r158, 3)
			}
			if d202.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
				ctx.W.EmitAddInt64(r158, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r158, d202.Reg)
			}
			r159 := ctx.AllocRegExcept(r158)
			ctx.W.EmitMovRegMem(r159, r158, 0)
			ctx.FreeReg(r158)
			d221 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r159}
			ctx.FreeDesc(&d220)
			var d222 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d201.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d201.Reg, 63)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d201.Reg}
			}
			if d222.Loc == scm.LocReg && d201.Loc == scm.LocReg && d222.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			d223 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm && d222.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d223.Imm.Int() - d222.Imm.Int())}
			} else if d222.Loc == scm.LocImm && d222.Imm.Int() == 0 {
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d223.Reg}
			} else if d223.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d223.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d222.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d222.Loc == scm.LocImm {
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d223.Reg, int32(d222.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
				ctx.W.EmitSubInt64(d223.Reg, scm.RegR11)
				}
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d223.Reg}
			} else {
				ctx.W.EmitSubInt64(d223.Reg, d222.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d223.Reg}
			}
			if d224.Loc == scm.LocReg && d223.Loc == scm.LocReg && d224.Reg == d223.Reg {
				ctx.TransferReg(d223.Reg)
				d223.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d222)
			var d225 scm.JITValueDesc
			if d221.Loc == scm.LocImm && d224.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d221.Imm.Int()) >> uint64(d224.Imm.Int())))}
			} else if d224.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d221.Reg, uint8(d224.Imm.Int()))
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
			} else {
				{
					shiftSrc := d221.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d224.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d224.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d224.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d225.Loc == scm.LocReg && d221.Loc == scm.LocReg && d225.Reg == d221.Reg {
				ctx.TransferReg(d221.Reg)
				d221.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.FreeDesc(&d224)
			var d226 scm.JITValueDesc
			if d206.Loc == scm.LocImm && d225.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() | d225.Imm.Int())}
			} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d225.Reg}
			} else if d225.Loc == scm.LocImm && d225.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r160, d206.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d206.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d225.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d225.Loc == scm.LocImm {
				r161 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r161, d206.Reg)
				if d225.Imm.Int() >= -2147483648 && d225.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r161, int32(d225.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d225.Imm.Int()))
					ctx.W.EmitOrInt64(r161, scratch)
					ctx.FreeReg(scratch)
				}
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
			} else {
				r162 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r162, d206.Reg)
				ctx.W.EmitOrInt64(r162, d225.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
			}
			if d226.Loc == scm.LocReg && d206.Loc == scm.LocReg && d226.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d225)
			ctx.EmitStoreToStack(d226, 40)
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl51)
			ctx.W.ResolveFixups()
			d227 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r144}
			ctx.FreeDesc(&idxInt)
			var d229 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r163 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r163, thisptr.Reg, off)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			}
			var d230 scm.JITValueDesc
			if d227.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() + d229.Imm.Int())}
			} else if d229.Loc == scm.LocImm && d229.Imm.Int() == 0 {
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d227.Reg}
			} else if d227.Loc == scm.LocImm && d227.Imm.Int() == 0 {
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d229.Reg}
			} else if d227.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d227.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d229.Reg)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d229.Loc == scm.LocImm {
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d227.Reg, int32(d229.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
				ctx.W.EmitAddInt64(d227.Reg, scm.RegR11)
				}
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d227.Reg}
			} else {
				ctx.W.EmitAddInt64(d227.Reg, d229.Reg)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d227.Reg}
			}
			if d230.Loc == scm.LocReg && d227.Loc == scm.LocReg && d230.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d227)
			ctx.FreeDesc(&d229)
			var d231 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r164 := ctx.AllocReg()
				r165 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r164, fieldAddr)
				ctx.W.EmitMovRegMem64(r165, fieldAddr+8)
				d231 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r164, Reg2: r165}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r166 := ctx.AllocReg()
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r166, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r167, thisptr.Reg, off+8)
				d231 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r166, Reg2: r167}
			}
			var d232 scm.JITValueDesc
			if d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d231.StackOff))}
			} else {
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d231.Reg2}
			}
			var d234 scm.JITValueDesc
			if d230.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d230.Imm.Int() >= d232.Imm.Int())}
			} else if d232.Loc == scm.LocImm {
				r168 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitCmpRegImm32(d230.Reg, int32(d232.Imm.Int()))
				ctx.W.EmitSetcc(r168, scm.CcGE)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r168}
			} else if d230.Loc == scm.LocImm {
				r169 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d230.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d232.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r169, scm.CcGE)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r169}
			} else {
				r170 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitCmpInt64(d230.Reg, d232.Reg)
				ctx.W.EmitSetcc(r170, scm.CcGE)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r170}
			}
			ctx.FreeDesc(&d232)
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			if d234.Loc == scm.LocImm {
				if d234.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d234.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl59)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d234)
			ctx.W.MarkLabel(lbl58)
			var d235 scm.JITValueDesc
			if d230.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d230.Imm.Int() < 0)}
			} else {
				r171 := ctx.AllocRegExcept(d230.Reg)
				ctx.W.EmitCmpRegImm32(d230.Reg, 0)
				ctx.W.EmitSetcc(r171, scm.CcL)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r171}
			}
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d235.Loc == scm.LocImm {
				if d235.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl60)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d235.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl61)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d235)
			ctx.W.MarkLabel(lbl57)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl60)
			r172 := ctx.AllocReg()
			if d230.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r172, uint64(d230.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r172, d230.Reg)
				ctx.W.EmitShlRegImm8(r172, 3)
			}
			if d231.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d231.Imm.Int()))
				ctx.W.EmitAddInt64(r172, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r172, d231.Reg)
			}
			r173 := ctx.AllocRegExcept(r172)
			ctx.W.EmitMovRegMem(r173, r172, 0)
			ctx.FreeReg(r172)
			d236 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
			ctx.FreeDesc(&d230)
			d237 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d195}, 2)
			ctx.FreeDesc(&d195)
			d238 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d236, d237}, 2)
			ctx.FreeDesc(&d236)
			d239 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d238}, 2)
			ctx.EmitMovPairToResult(&d239, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			ctx.W.PatchInt32(r8, int32(48))
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
