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
			ctx.EnsureDesc(&idxInt)
			d0 := idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl3)
			var d1 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 232
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 232)
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r4, thisptr.Reg, off)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d1)
			}
			d2 := d1
			ctx.EnsureDesc(&d2)
			if d2.Loc != scm.LocImm && d2.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			if d2.Loc == scm.LocImm {
				if d2.Imm.Bool() {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				} else {
					ctx.W.MarkLabel(lbl7)
					ctx.W.EmitJmp(lbl5)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
				ctx.W.MarkLabel(lbl7)
				ctx.W.EmitJmp(lbl5)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl5)
			ctx.EnsureDesc(&d0)
			d3 := d0
			_ = d3
			r5 := d0.Loc == scm.LocReg
			r6 := d0.Reg
			if r5 { ctx.ProtectReg(r6) }
			r7 := ctx.W.EmitSubRSP32Fixup()
			lbl8 := ctx.W.ReserveLabel()
			lbl9 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl9)
			ctx.EnsureDesc(&d3)
			ctx.EnsureDesc(&d3)
			var d4 scm.JITValueDesc
			if d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d3.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d3.Reg)
				ctx.W.EmitShlRegImm8(r8, 32)
				ctx.W.EmitShrRegImm8(r8, 32)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d4)
			}
			var d5 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r9, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d5)
			}
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d5)
			var d6 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d5.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r10, d5.Reg)
				ctx.W.EmitShlRegImm8(r10, 56)
				ctx.W.EmitShrRegImm8(r10, 56)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d6)
			}
			ctx.FreeDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d6)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d6)
			var d7 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d6.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() * d6.Imm.Int())}
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d6.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d7)
			} else if d6.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d6.Imm.Int() >= -2147483648 && d6.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d6.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d6.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d7)
			} else {
				r11 := ctx.AllocRegExcept(d4.Reg, d6.Reg)
				ctx.W.EmitMovRegReg(r11, d4.Reg)
				ctx.W.EmitImulInt64(r11, d6.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d7)
			}
			if d7.Loc == scm.LocReg && d4.Loc == scm.LocReg && d7.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			ctx.FreeDesc(&d6)
			var d8 scm.JITValueDesc
			r12 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r12, uint64(dataPtr))
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12, StackOff: int32(sliceLen)}
				ctx.BindReg(r12, &d8)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				ctx.W.EmitMovRegMem(r12, thisptr.Reg, off)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d8)
			}
			ctx.BindReg(r12, &d8)
			ctx.EnsureDesc(&d7)
			var d9 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r13, d7.Reg)
				ctx.W.EmitShrRegImm8(r13, 6)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d9)
			}
			if d9.Loc == scm.LocReg && d7.Loc == scm.LocReg && d9.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d9)
			r14 := ctx.AllocReg()
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d8)
			if d9.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r14, uint64(d9.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r14, d9.Reg)
				ctx.W.EmitShlRegImm8(r14, 3)
			}
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
				ctx.W.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r14, d8.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.W.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d10 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.BindReg(r15, &d10)
			ctx.FreeDesc(&d9)
			ctx.EnsureDesc(&d7)
			var d11 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r16, d7.Reg)
				ctx.W.EmitAndRegImm32(r16, 63)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d11)
			}
			if d11.Loc == scm.LocReg && d7.Loc == scm.LocReg && d11.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d11)
			var d12 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d10.Imm.Int()) << uint64(d11.Imm.Int())))}
			} else if d11.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r17, d10.Reg)
				ctx.W.EmitShlRegImm8(r17, uint8(d11.Imm.Int()))
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d12)
			} else {
				{
					shiftSrc := d10.Reg
					r18 := ctx.AllocRegExcept(d10.Reg)
					ctx.W.EmitMovRegReg(r18, d10.Reg)
					shiftSrc = r18
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d11.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d11.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d11.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d12)
				}
			}
			if d12.Loc == scm.LocReg && d10.Loc == scm.LocReg && d12.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d11)
			var d13 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25)
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d13)
			}
			d14 := d13
			ctx.EnsureDesc(&d14)
			if d14.Loc != scm.LocImm && d14.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d14.Loc == scm.LocImm {
				if d14.Imm.Bool() {
					ctx.W.MarkLabel(lbl12)
					ctx.W.EmitJmp(lbl10)
				} else {
					ctx.W.MarkLabel(lbl13)
			d15 := d12
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d14.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl12)
				ctx.W.EmitJmp(lbl10)
				ctx.W.MarkLabel(lbl13)
			d16 := d12
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 0)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d13)
			ctx.W.MarkLabel(lbl11)
			d17 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r20, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
				ctx.BindReg(r20, &d18)
			}
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d18.Imm.Int()))))}
			} else {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r21, d18.Reg)
				ctx.W.EmitShlRegImm8(r21, 56)
				ctx.W.EmitShrRegImm8(r21, 56)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d19)
			}
			ctx.FreeDesc(&d18)
			d20 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d19)
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d19.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				r22 := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(r22, d20.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d21)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				if d19.Imm.Int() >= -2147483648 && d19.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d19.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d19.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d21)
			} else {
				r23 := ctx.AllocRegExcept(d20.Reg, d19.Reg)
				ctx.W.EmitMovRegReg(r23, d20.Reg)
				ctx.W.EmitSubInt64(r23, d19.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d21)
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d21)
			var d22 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				r24 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r24, d17.Reg)
				ctx.W.EmitShrRegImm8(r24, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d22)
			} else {
				{
					shiftSrc := d17.Reg
					r25 := ctx.AllocRegExcept(d17.Reg)
					ctx.W.EmitMovRegReg(r25, d17.Reg)
					shiftSrc = r25
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d21.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d21.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d21.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d22)
				}
			}
			if d22.Loc == scm.LocReg && d17.Loc == scm.LocReg && d22.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d21)
			r26 := ctx.AllocReg()
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d22)
			if d22.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r26, d22)
			}
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl10)
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d7)
			var d23 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() % 64)}
			} else {
				r27 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r27, d7.Reg)
				ctx.W.EmitAndRegImm32(r27, 63)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d23)
			}
			if d23.Loc == scm.LocReg && d7.Loc == scm.LocReg && d23.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r28, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
				ctx.BindReg(r28, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r29, d24.Reg)
				ctx.W.EmitShlRegImm8(r29, 56)
				ctx.W.EmitShrRegImm8(r29, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d25)
			}
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r30, d23.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d26)
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d25.Reg}
				ctx.BindReg(d25.Reg, &d26)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(scratch, d23.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r31 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
				ctx.W.EmitMovRegReg(r31, d23.Reg)
				ctx.W.EmitAddInt64(r31, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d26)
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d26)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d26.Imm.Int()) > uint64(64))}
			} else {
				r32 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitCmpRegImm32(d26.Reg, 64)
				ctx.W.EmitSetcc(r32, scm.CcA)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d27)
			}
			ctx.FreeDesc(&d26)
			d28 := d27
			ctx.EnsureDesc(&d28)
			if d28.Loc != scm.LocImm && d28.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d28.Loc == scm.LocImm {
				if d28.Imm.Bool() {
					ctx.W.MarkLabel(lbl15)
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.MarkLabel(lbl16)
			d29 := d12
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
					ctx.W.EmitJmp(lbl11)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d28.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl15)
				ctx.W.EmitJmp(lbl14)
				ctx.W.MarkLabel(lbl16)
			d30 := d12
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, 0)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d27)
			ctx.W.MarkLabel(lbl14)
			d17 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d7)
			var d31 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r33, d7.Reg)
				ctx.W.EmitShrRegImm8(r33, 6)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d31)
			}
			if d31.Loc == scm.LocReg && d7.Loc == scm.LocReg && d31.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(scratch, d31.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			}
			if d32.Loc == scm.LocReg && d31.Loc == scm.LocReg && d32.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d8)
			if d32.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r34, uint64(d32.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r34, d32.Reg)
				ctx.W.EmitShlRegImm8(r34, 3)
			}
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d8.Imm.Int()))
				ctx.W.EmitAddInt64(r34, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r34, d8.Reg)
			}
			r35 := ctx.AllocRegExcept(r34)
			ctx.W.EmitMovRegMem(r35, r34, 0)
			ctx.FreeReg(r34)
			d33 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d33)
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d7)
			var d34 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r36, d7.Reg)
				ctx.W.EmitAndRegImm32(r36, 63)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d34)
			}
			if d34.Loc == scm.LocReg && d7.Loc == scm.LocReg && d34.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d7)
			d35 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			ctx.EnsureDesc(&d35)
			ctx.EnsureDesc(&d34)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() - d34.Imm.Int())}
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(r37, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d36)
			} else if d35.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(scratch, d35.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else {
				r38 := ctx.AllocRegExcept(d35.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r38, d35.Reg)
				ctx.W.EmitSubInt64(r38, d34.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d33.Imm.Int()) >> uint64(d36.Imm.Int())))}
			} else if d36.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r39, d33.Reg)
				ctx.W.EmitShrRegImm8(r39, uint8(d36.Imm.Int()))
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d37)
			} else {
				{
					shiftSrc := d33.Reg
					r40 := ctx.AllocRegExcept(d33.Reg)
					ctx.W.EmitMovRegReg(r40, d33.Reg)
					shiftSrc = r40
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d36.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d36.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d36.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d37)
				}
			}
			if d37.Loc == scm.LocReg && d33.Loc == scm.LocReg && d37.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d33)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d37)
			var d38 scm.JITValueDesc
			if d12.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() | d37.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r41, d12.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d38)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r42, d12.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r42, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitOrInt64(r42, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			} else {
				r43 := ctx.AllocRegExcept(d12.Reg, d37.Reg)
				ctx.W.EmitMovRegReg(r43, d12.Reg)
				ctx.W.EmitOrInt64(r43, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d38)
			}
			if d38.Loc == scm.LocReg && d12.Loc == scm.LocReg && d38.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			d39 := d38
			if d39.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d39)
			ctx.EmitStoreToStack(d39, 0)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			d40 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d40)
			ctx.BindReg(r26, &d40)
			if r5 { ctx.UnprotectReg(r6) }
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d40.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d41)
			}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d42)
			}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + d42.Imm.Int())}
			} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r46, d41.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d43)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r47 := ctx.AllocRegExcept(d41.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r47, d41.Reg)
				ctx.W.EmitAddInt64(r47, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d43)
			}
			if d43.Loc == scm.LocReg && d41.Loc == scm.LocReg && d43.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d43.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d43.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d45)
			}
			d46 := d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				} else {
					ctx.W.MarkLabel(lbl20)
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
				ctx.W.MarkLabel(lbl20)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl4)
			ctx.EnsureDesc(&d0)
			d47 := d0
			_ = d47
			r50 := d0.Loc == scm.LocReg
			r51 := d0.Reg
			if r50 { ctx.ProtectReg(r51) }
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl22)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d47.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d47.Reg)
				ctx.W.EmitShlRegImm8(r52, 32)
				ctx.W.EmitShrRegImm8(r52, 32)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d48)
			}
			var d49 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r53, thisptr.Reg, off)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d49)
			}
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d49.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, d49.Reg)
				ctx.W.EmitShlRegImm8(r54, 56)
				ctx.W.EmitShrRegImm8(r54, 56)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d50)
			}
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d50)
			var d51 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d50.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d48.Imm.Int() * d50.Imm.Int())}
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d48.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d50.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else if d50.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegReg(scratch, d48.Reg)
				if d50.Imm.Int() >= -2147483648 && d50.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d50.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d51)
			} else {
				r55 := ctx.AllocRegExcept(d48.Reg, d50.Reg)
				ctx.W.EmitMovRegReg(r55, d48.Reg)
				ctx.W.EmitImulInt64(r55, d50.Reg)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d51)
			}
			if d51.Loc == scm.LocReg && d48.Loc == scm.LocReg && d51.Reg == d48.Reg {
				ctx.TransferReg(d48.Reg)
				d48.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d50)
			var d52 scm.JITValueDesc
			r56 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r56, uint64(dataPtr))
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56, StackOff: int32(sliceLen)}
				ctx.BindReg(r56, &d52)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r56, thisptr.Reg, off)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
				ctx.BindReg(r56, &d52)
			}
			ctx.BindReg(r56, &d52)
			ctx.EnsureDesc(&d51)
			var d53 scm.JITValueDesc
			if d51.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() / 64)}
			} else {
				r57 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r57, d51.Reg)
				ctx.W.EmitShrRegImm8(r57, 6)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d53)
			}
			if d53.Loc == scm.LocReg && d51.Loc == scm.LocReg && d53.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d53)
			r58 := ctx.AllocReg()
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d52)
			if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r58, uint64(d53.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r58, d53.Reg)
				ctx.W.EmitShlRegImm8(r58, 3)
			}
			if d52.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(r58, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r58, d52.Reg)
			}
			r59 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r59, r58, 0)
			ctx.FreeReg(r58)
			d54 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d54)
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d51)
			var d55 scm.JITValueDesc
			if d51.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() % 64)}
			} else {
				r60 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r60, d51.Reg)
				ctx.W.EmitAndRegImm32(r60, 63)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d55)
			}
			if d55.Loc == scm.LocReg && d51.Loc == scm.LocReg && d55.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d54.Imm.Int()) << uint64(d55.Imm.Int())))}
			} else if d55.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegReg(r61, d54.Reg)
				ctx.W.EmitShlRegImm8(r61, uint8(d55.Imm.Int()))
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d56)
			} else {
				{
					shiftSrc := d54.Reg
					r62 := ctx.AllocRegExcept(d54.Reg)
					ctx.W.EmitMovRegReg(r62, d54.Reg)
					shiftSrc = r62
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d55.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d55.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d55.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d56)
				}
			}
			if d56.Loc == scm.LocReg && d54.Loc == scm.LocReg && d56.Reg == d54.Reg {
				ctx.TransferReg(d54.Reg)
				d54.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d54)
			ctx.FreeDesc(&d55)
			var d57 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r63, thisptr.Reg, off)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
				ctx.BindReg(r63, &d57)
			}
			d58 := d57
			ctx.EnsureDesc(&d58)
			if d58.Loc != scm.LocImm && d58.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d58.Loc == scm.LocImm {
				if d58.Imm.Bool() {
					ctx.W.MarkLabel(lbl25)
					ctx.W.EmitJmp(lbl23)
				} else {
					ctx.W.MarkLabel(lbl26)
			d59 := d56
			if d59.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			ctx.EmitStoreToStack(d59, 8)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d58.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl26)
				ctx.W.MarkLabel(lbl25)
				ctx.W.EmitJmp(lbl23)
				ctx.W.MarkLabel(lbl26)
			d60 := d56
			if d60.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 8)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d57)
			ctx.W.MarkLabel(lbl24)
			d61 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d62 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d62)
			}
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d62.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d62.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d63)
			}
			ctx.FreeDesc(&d62)
			d64 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d63)
			var d65 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d63.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() - d63.Imm.Int())}
			} else if d63.Loc == scm.LocImm && d63.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(r66, d64.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d65)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d63.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else if d63.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(scratch, d64.Reg)
				if d63.Imm.Int() >= -2147483648 && d63.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d63.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d63.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else {
				r67 := ctx.AllocRegExcept(d64.Reg, d63.Reg)
				ctx.W.EmitMovRegReg(r67, d64.Reg)
				ctx.W.EmitSubInt64(r67, d63.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d65)
			}
			if d65.Loc == scm.LocReg && d64.Loc == scm.LocReg && d65.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d63)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) >> uint64(d65.Imm.Int())))}
			} else if d65.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r68, d61.Reg)
				ctx.W.EmitShrRegImm8(r68, uint8(d65.Imm.Int()))
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d66)
			} else {
				{
					shiftSrc := d61.Reg
					r69 := ctx.AllocRegExcept(d61.Reg)
					ctx.W.EmitMovRegReg(r69, d61.Reg)
					shiftSrc = r69
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d65.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d65.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d65.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d66)
				}
			}
			if d66.Loc == scm.LocReg && d61.Loc == scm.LocReg && d66.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d61)
			ctx.FreeDesc(&d65)
			r70 := ctx.AllocReg()
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d66)
			if d66.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r70, d66)
			}
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl23)
			d61 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d51)
			var d67 scm.JITValueDesc
			if d51.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r71, d51.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d67)
			}
			if d67.Loc == scm.LocReg && d51.Loc == scm.LocReg && d67.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			var d68 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d68)
			}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d68)
			var d69 scm.JITValueDesc
			if d68.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d68.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d68.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d69)
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d69)
			var d70 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d67.Imm.Int() + d69.Imm.Int())}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(r74, d67.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d70)
			} else if d67.Loc == scm.LocImm && d67.Imm.Int() == 0 {
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d69.Reg}
				ctx.BindReg(d69.Reg, &d70)
			} else if d67.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d67.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitMovRegReg(scratch, d67.Reg)
				if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d69.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else {
				r75 := ctx.AllocRegExcept(d67.Reg, d69.Reg)
				ctx.W.EmitMovRegReg(r75, d67.Reg)
				ctx.W.EmitAddInt64(r75, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d70)
			}
			if d70.Loc == scm.LocReg && d67.Loc == scm.LocReg && d70.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d67)
			ctx.FreeDesc(&d69)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d70.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitCmpRegImm32(d70.Reg, 64)
				ctx.W.EmitSetcc(r76, scm.CcA)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d71)
			}
			ctx.FreeDesc(&d70)
			d72 := d71
			ctx.EnsureDesc(&d72)
			if d72.Loc != scm.LocImm && d72.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d72.Loc == scm.LocImm {
				if d72.Imm.Bool() {
					ctx.W.MarkLabel(lbl28)
					ctx.W.EmitJmp(lbl27)
				} else {
					ctx.W.MarkLabel(lbl29)
			d73 := d56
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 8)
					ctx.W.EmitJmp(lbl24)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d72.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl29)
			d74 := d56
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d74)
			ctx.EmitStoreToStack(d74, 8)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d71)
			ctx.W.MarkLabel(lbl27)
			d61 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d51)
			var d75 scm.JITValueDesc
			if d51.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r77, d51.Reg)
				ctx.W.EmitShrRegImm8(r77, 6)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d75)
			}
			if d75.Loc == scm.LocReg && d51.Loc == scm.LocReg && d75.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(scratch, d75.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			}
			if d76.Loc == scm.LocReg && d75.Loc == scm.LocReg && d76.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d76)
			r78 := ctx.AllocReg()
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d52)
			if d76.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r78, uint64(d76.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r78, d76.Reg)
				ctx.W.EmitShlRegImm8(r78, 3)
			}
			if d52.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
				ctx.W.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r78, d52.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.W.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d77 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d77)
			ctx.FreeDesc(&d76)
			ctx.EnsureDesc(&d51)
			var d78 scm.JITValueDesc
			if d51.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d51.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r80, d51.Reg)
				ctx.W.EmitAndRegImm32(r80, 63)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d78)
			}
			if d78.Loc == scm.LocReg && d51.Loc == scm.LocReg && d78.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			d79 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d79)
			ctx.EnsureDesc(&d78)
			var d80 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() - d78.Imm.Int())}
			} else if d78.Loc == scm.LocImm && d78.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r81, d79.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d80)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d78.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else if d78.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d78.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d80)
			} else {
				r82 := ctx.AllocRegExcept(d79.Reg, d78.Reg)
				ctx.W.EmitMovRegReg(r82, d79.Reg)
				ctx.W.EmitSubInt64(r82, d78.Reg)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d80)
			}
			if d80.Loc == scm.LocReg && d79.Loc == scm.LocReg && d80.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d77.Imm.Int()) >> uint64(d80.Imm.Int())))}
			} else if d80.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegReg(r83, d77.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d80.Imm.Int()))
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d81)
			} else {
				{
					shiftSrc := d77.Reg
					r84 := ctx.AllocRegExcept(d77.Reg)
					ctx.W.EmitMovRegReg(r84, d77.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d80.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d80.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d80.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d81)
				}
			}
			if d81.Loc == scm.LocReg && d77.Loc == scm.LocReg && d81.Reg == d77.Reg {
				ctx.TransferReg(d77.Reg)
				d77.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d80)
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d81)
			var d82 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() | d81.Imm.Int())}
			} else if d56.Loc == scm.LocImm && d56.Imm.Int() == 0 {
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d81.Reg}
				ctx.BindReg(d81.Reg, &d82)
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				r85 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r85, d56.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d82)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d82)
			} else if d81.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r86, d56.Reg)
				if d81.Imm.Int() >= -2147483648 && d81.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r86, int32(d81.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d81.Imm.Int()))
					ctx.W.EmitOrInt64(r86, scm.RegR11)
				}
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d82)
			} else {
				r87 := ctx.AllocRegExcept(d56.Reg, d81.Reg)
				ctx.W.EmitMovRegReg(r87, d56.Reg)
				ctx.W.EmitOrInt64(r87, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d82)
			}
			if d82.Loc == scm.LocReg && d56.Loc == scm.LocReg && d82.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			d83 := d82
			if d83.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d83)
			ctx.EmitStoreToStack(d83, 8)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl21)
			d84 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d84)
			ctx.BindReg(r70, &d84)
			if r50 { ctx.UnprotectReg(r51) }
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d84)
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d84.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d84.Reg)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d85)
			}
			ctx.FreeDesc(&d84)
			var d86 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r89, thisptr.Reg, off)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d86)
			}
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			ctx.EnsureDesc(&d85)
			ctx.EnsureDesc(&d86)
			var d87 scm.JITValueDesc
			if d85.Loc == scm.LocImm && d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() + d86.Imm.Int())}
			} else if d86.Loc == scm.LocImm && d86.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(r90, d85.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d87)
			} else if d85.Loc == scm.LocImm && d85.Imm.Int() == 0 {
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d86.Reg}
				ctx.BindReg(d86.Reg, &d87)
			} else if d85.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d86.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else if d86.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(scratch, d85.Reg)
				if d86.Imm.Int() >= -2147483648 && d86.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d86.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d86.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d87)
			} else {
				r91 := ctx.AllocRegExcept(d85.Reg, d86.Reg)
				ctx.W.EmitMovRegReg(r91, d85.Reg)
				ctx.W.EmitAddInt64(r91, d86.Reg)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d87)
			}
			if d87.Loc == scm.LocReg && d85.Loc == scm.LocReg && d87.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.FreeDesc(&d86)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d87.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d88)
			}
			ctx.FreeDesc(&d87)
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d89)
			}
			d90 := d89
			ctx.EnsureDesc(&d90)
			if d90.Loc != scm.LocImm && d90.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d90.Loc == scm.LocImm {
				if d90.Imm.Bool() {
					ctx.W.MarkLabel(lbl32)
					ctx.W.EmitJmp(lbl30)
				} else {
					ctx.W.MarkLabel(lbl33)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d90.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
				ctx.W.MarkLabel(lbl33)
				ctx.W.EmitJmp(lbl31)
			}
			ctx.FreeDesc(&d89)
			ctx.W.MarkLabel(lbl18)
			ctx.EnsureDesc(&d44)
			d91 := d44
			_ = d91
			r94 := d44.Loc == scm.LocReg
			r95 := d44.Reg
			if r94 { ctx.ProtectReg(r95) }
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl35)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d91.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d91.Reg)
				ctx.W.EmitShlRegImm8(r96, 32)
				ctx.W.EmitShrRegImm8(r96, 32)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d92)
			}
			var d93 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r97, thisptr.Reg, off)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d93)
			}
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d93)
			var d94 scm.JITValueDesc
			if d93.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d93.Imm.Int()))))}
			} else {
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r98, d93.Reg)
				ctx.W.EmitShlRegImm8(r98, 56)
				ctx.W.EmitShrRegImm8(r98, 56)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d94)
			}
			ctx.FreeDesc(&d93)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d94)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d94)
			ctx.EnsureDesc(&d92)
			ctx.EnsureDesc(&d94)
			var d95 scm.JITValueDesc
			if d92.Loc == scm.LocImm && d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() * d94.Imm.Int())}
			} else if d92.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d92.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d94.Reg)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d95)
			} else if d94.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(scratch, d92.Reg)
				if d94.Imm.Int() >= -2147483648 && d94.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d94.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d94.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d95)
			} else {
				r99 := ctx.AllocRegExcept(d92.Reg, d94.Reg)
				ctx.W.EmitMovRegReg(r99, d92.Reg)
				ctx.W.EmitImulInt64(r99, d94.Reg)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d95)
			}
			if d95.Loc == scm.LocReg && d92.Loc == scm.LocReg && d95.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d92)
			ctx.FreeDesc(&d94)
			var d96 scm.JITValueDesc
			r100 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r100, uint64(dataPtr))
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100, StackOff: int32(sliceLen)}
				ctx.BindReg(r100, &d96)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r100, thisptr.Reg, off)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d96)
			}
			ctx.BindReg(r100, &d96)
			ctx.EnsureDesc(&d95)
			var d97 scm.JITValueDesc
			if d95.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() / 64)}
			} else {
				r101 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r101, d95.Reg)
				ctx.W.EmitShrRegImm8(r101, 6)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d97)
			}
			if d97.Loc == scm.LocReg && d95.Loc == scm.LocReg && d97.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d97)
			r102 := ctx.AllocReg()
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d96)
			if d97.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r102, uint64(d97.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r102, d97.Reg)
				ctx.W.EmitShlRegImm8(r102, 3)
			}
			if d96.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d96.Imm.Int()))
				ctx.W.EmitAddInt64(r102, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r102, d96.Reg)
			}
			r103 := ctx.AllocRegExcept(r102)
			ctx.W.EmitMovRegMem(r103, r102, 0)
			ctx.FreeReg(r102)
			d98 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d98)
			ctx.FreeDesc(&d97)
			ctx.EnsureDesc(&d95)
			var d99 scm.JITValueDesc
			if d95.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() % 64)}
			} else {
				r104 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r104, d95.Reg)
				ctx.W.EmitAndRegImm32(r104, 63)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d99)
			}
			if d99.Loc == scm.LocReg && d95.Loc == scm.LocReg && d99.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d98)
			ctx.EnsureDesc(&d99)
			var d100 scm.JITValueDesc
			if d98.Loc == scm.LocImm && d99.Loc == scm.LocImm {
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d98.Imm.Int()) << uint64(d99.Imm.Int())))}
			} else if d99.Loc == scm.LocImm {
				r105 := ctx.AllocRegExcept(d98.Reg)
				ctx.W.EmitMovRegReg(r105, d98.Reg)
				ctx.W.EmitShlRegImm8(r105, uint8(d99.Imm.Int()))
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d100)
			} else {
				{
					shiftSrc := d98.Reg
					r106 := ctx.AllocRegExcept(d98.Reg)
					ctx.W.EmitMovRegReg(r106, d98.Reg)
					shiftSrc = r106
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d99.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d99.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d99.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d100 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d100)
				}
			}
			if d100.Loc == scm.LocReg && d98.Loc == scm.LocReg && d100.Reg == d98.Reg {
				ctx.TransferReg(d98.Reg)
				d98.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d98)
			ctx.FreeDesc(&d99)
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d101)
			}
			d102 := d101
			ctx.EnsureDesc(&d102)
			if d102.Loc != scm.LocImm && d102.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d102.Loc == scm.LocImm {
				if d102.Imm.Bool() {
					ctx.W.MarkLabel(lbl38)
					ctx.W.EmitJmp(lbl36)
				} else {
					ctx.W.MarkLabel(lbl39)
			d103 := d100
			if d103.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d103)
			ctx.EmitStoreToStack(d103, 16)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d102.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl38)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl38)
				ctx.W.EmitJmp(lbl36)
				ctx.W.MarkLabel(lbl39)
			d104 := d100
			if d104.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d104)
			ctx.EmitStoreToStack(d104, 16)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d101)
			ctx.W.MarkLabel(lbl37)
			d105 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d106 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r108, thisptr.Reg, off)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
				ctx.BindReg(r108, &d106)
			}
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d106.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r109, d106.Reg)
				ctx.W.EmitShlRegImm8(r109, 56)
				ctx.W.EmitShrRegImm8(r109, 56)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d107)
			}
			ctx.FreeDesc(&d106)
			d108 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d107)
			var d109 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d107.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() - d107.Imm.Int())}
			} else if d107.Loc == scm.LocImm && d107.Imm.Int() == 0 {
				r110 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r110, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d109)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d107.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d107.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else if d107.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(scratch, d108.Reg)
				if d107.Imm.Int() >= -2147483648 && d107.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d107.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d107.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else {
				r111 := ctx.AllocRegExcept(d108.Reg, d107.Reg)
				ctx.W.EmitMovRegReg(r111, d108.Reg)
				ctx.W.EmitSubInt64(r111, d107.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d109)
			}
			if d109.Loc == scm.LocReg && d108.Loc == scm.LocReg && d109.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d105.Imm.Int()) >> uint64(d109.Imm.Int())))}
			} else if d109.Loc == scm.LocImm {
				r112 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(r112, d105.Reg)
				ctx.W.EmitShrRegImm8(r112, uint8(d109.Imm.Int()))
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d110)
			} else {
				{
					shiftSrc := d105.Reg
					r113 := ctx.AllocRegExcept(d105.Reg)
					ctx.W.EmitMovRegReg(r113, d105.Reg)
					shiftSrc = r113
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d109.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d109.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d109.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d110)
				}
			}
			if d110.Loc == scm.LocReg && d105.Loc == scm.LocReg && d110.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d105)
			ctx.FreeDesc(&d109)
			r114 := ctx.AllocReg()
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d110)
			if d110.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r114, d110)
			}
			ctx.W.EmitJmp(lbl34)
			ctx.W.MarkLabel(lbl36)
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d95)
			var d111 scm.JITValueDesc
			if d95.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() % 64)}
			} else {
				r115 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r115, d95.Reg)
				ctx.W.EmitAndRegImm32(r115, 63)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d111)
			}
			if d111.Loc == scm.LocReg && d95.Loc == scm.LocReg && d111.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			var d112 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r116, thisptr.Reg, off)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
				ctx.BindReg(r116, &d112)
			}
			ctx.EnsureDesc(&d112)
			ctx.EnsureDesc(&d112)
			var d113 scm.JITValueDesc
			if d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d112.Imm.Int()))))}
			} else {
				r117 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r117, d112.Reg)
				ctx.W.EmitShlRegImm8(r117, 56)
				ctx.W.EmitShrRegImm8(r117, 56)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d113)
			}
			ctx.FreeDesc(&d112)
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d111)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d111.Loc == scm.LocImm && d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d111.Imm.Int() + d113.Imm.Int())}
			} else if d113.Loc == scm.LocImm && d113.Imm.Int() == 0 {
				r118 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(r118, d111.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d114)
			} else if d111.Loc == scm.LocImm && d111.Imm.Int() == 0 {
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d113.Reg}
				ctx.BindReg(d113.Reg, &d114)
			} else if d111.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d111.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d113.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d114)
			} else if d113.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitMovRegReg(scratch, d111.Reg)
				if d113.Imm.Int() >= -2147483648 && d113.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d113.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d113.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d114)
			} else {
				r119 := ctx.AllocRegExcept(d111.Reg, d113.Reg)
				ctx.W.EmitMovRegReg(r119, d111.Reg)
				ctx.W.EmitAddInt64(r119, d113.Reg)
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d114)
			}
			if d114.Loc == scm.LocReg && d111.Loc == scm.LocReg && d114.Reg == d111.Reg {
				ctx.TransferReg(d111.Reg)
				d111.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d111)
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d114)
			var d115 scm.JITValueDesc
			if d114.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d114.Imm.Int()) > uint64(64))}
			} else {
				r120 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitCmpRegImm32(d114.Reg, 64)
				ctx.W.EmitSetcc(r120, scm.CcA)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r120}
				ctx.BindReg(r120, &d115)
			}
			ctx.FreeDesc(&d114)
			d116 := d115
			ctx.EnsureDesc(&d116)
			if d116.Loc != scm.LocImm && d116.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d116.Loc == scm.LocImm {
				if d116.Imm.Bool() {
					ctx.W.MarkLabel(lbl41)
					ctx.W.EmitJmp(lbl40)
				} else {
					ctx.W.MarkLabel(lbl42)
			d117 := d100
			if d117.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d117)
			ctx.EmitStoreToStack(d117, 16)
					ctx.W.EmitJmp(lbl37)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d116.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
				ctx.W.EmitJmp(lbl42)
				ctx.W.MarkLabel(lbl41)
				ctx.W.EmitJmp(lbl40)
				ctx.W.MarkLabel(lbl42)
			d118 := d100
			if d118.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d118)
			ctx.EmitStoreToStack(d118, 16)
				ctx.W.EmitJmp(lbl37)
			}
			ctx.FreeDesc(&d115)
			ctx.W.MarkLabel(lbl40)
			d105 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d95)
			var d119 scm.JITValueDesc
			if d95.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() / 64)}
			} else {
				r121 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r121, d95.Reg)
				ctx.W.EmitShrRegImm8(r121, 6)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d119)
			}
			if d119.Loc == scm.LocReg && d95.Loc == scm.LocReg && d119.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			}
			if d120.Loc == scm.LocReg && d119.Loc == scm.LocReg && d120.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d120)
			r122 := ctx.AllocReg()
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d96)
			if d120.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r122, uint64(d120.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r122, d120.Reg)
				ctx.W.EmitShlRegImm8(r122, 3)
			}
			if d96.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d96.Imm.Int()))
				ctx.W.EmitAddInt64(r122, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r122, d96.Reg)
			}
			r123 := ctx.AllocRegExcept(r122)
			ctx.W.EmitMovRegMem(r123, r122, 0)
			ctx.FreeReg(r122)
			d121 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
			ctx.BindReg(r123, &d121)
			ctx.FreeDesc(&d120)
			ctx.EnsureDesc(&d95)
			var d122 scm.JITValueDesc
			if d95.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d95.Imm.Int() % 64)}
			} else {
				r124 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r124, d95.Reg)
				ctx.W.EmitAndRegImm32(r124, 63)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d122)
			}
			if d122.Loc == scm.LocReg && d95.Loc == scm.LocReg && d122.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			d123 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d122)
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() - d122.Imm.Int())}
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r125 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r125, d123.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d124)
			} else if d123.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d123.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d122.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d124)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(scratch, d123.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d122.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d124)
			} else {
				r126 := ctx.AllocRegExcept(d123.Reg, d122.Reg)
				ctx.W.EmitMovRegReg(r126, d123.Reg)
				ctx.W.EmitSubInt64(r126, d122.Reg)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d124)
			}
			if d124.Loc == scm.LocReg && d123.Loc == scm.LocReg && d124.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			ctx.EnsureDesc(&d121)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d121.Loc == scm.LocImm && d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d121.Imm.Int()) >> uint64(d124.Imm.Int())))}
			} else if d124.Loc == scm.LocImm {
				r127 := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegReg(r127, d121.Reg)
				ctx.W.EmitShrRegImm8(r127, uint8(d124.Imm.Int()))
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d125)
			} else {
				{
					shiftSrc := d121.Reg
					r128 := ctx.AllocRegExcept(d121.Reg)
					ctx.W.EmitMovRegReg(r128, d121.Reg)
					shiftSrc = r128
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d124.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d124.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d124.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d125)
				}
			}
			if d125.Loc == scm.LocReg && d121.Loc == scm.LocReg && d125.Reg == d121.Reg {
				ctx.TransferReg(d121.Reg)
				d121.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d121)
			ctx.FreeDesc(&d124)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d100.Loc == scm.LocImm && d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d100.Imm.Int() | d125.Imm.Int())}
			} else if d100.Loc == scm.LocImm && d100.Imm.Int() == 0 {
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d125.Reg}
				ctx.BindReg(d125.Reg, &d126)
			} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
				r129 := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(r129, d100.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d126)
			} else if d100.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d100.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d126)
			} else if d125.Loc == scm.LocImm {
				r130 := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(r130, d100.Reg)
				if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r130, int32(d125.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d125.Imm.Int()))
					ctx.W.EmitOrInt64(r130, scm.RegR11)
				}
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d126)
			} else {
				r131 := ctx.AllocRegExcept(d100.Reg, d125.Reg)
				ctx.W.EmitMovRegReg(r131, d100.Reg)
				ctx.W.EmitOrInt64(r131, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d126)
			}
			if d126.Loc == scm.LocReg && d100.Loc == scm.LocReg && d126.Reg == d100.Reg {
				ctx.TransferReg(d100.Reg)
				d100.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			d127 := d126
			if d127.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d127)
			ctx.EmitStoreToStack(d127, 16)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl34)
			d128 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
			ctx.BindReg(r114, &d128)
			ctx.BindReg(r114, &d128)
			if r94 { ctx.UnprotectReg(r95) }
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d128)
			var d129 scm.JITValueDesc
			if d128.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d128.Imm.Int()))))}
			} else {
				r132 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r132, d128.Reg)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d129)
			}
			ctx.FreeDesc(&d128)
			var d130 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r133, thisptr.Reg, off)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
				ctx.BindReg(r133, &d130)
			}
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d130)
			var d131 scm.JITValueDesc
			if d129.Loc == scm.LocImm && d130.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d129.Imm.Int() + d130.Imm.Int())}
			} else if d130.Loc == scm.LocImm && d130.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(r134, d129.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d131)
			} else if d129.Loc == scm.LocImm && d129.Imm.Int() == 0 {
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d130.Reg}
				ctx.BindReg(d130.Reg, &d131)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d129.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else if d130.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegReg(scratch, d129.Reg)
				if d130.Imm.Int() >= -2147483648 && d130.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d130.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d130.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d131)
			} else {
				r135 := ctx.AllocRegExcept(d129.Reg, d130.Reg)
				ctx.W.EmitMovRegReg(r135, d129.Reg)
				ctx.W.EmitAddInt64(r135, d130.Reg)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d131)
			}
			if d131.Loc == scm.LocReg && d129.Loc == scm.LocReg && d131.Reg == d129.Reg {
				ctx.TransferReg(d129.Reg)
				d129.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d129)
			ctx.FreeDesc(&d130)
			ctx.EnsureDesc(&d44)
			d132 := d44
			_ = d132
			r136 := d44.Loc == scm.LocReg
			r137 := d44.Reg
			if r136 { ctx.ProtectReg(r137) }
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl44)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d132)
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d132.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d132.Reg)
				ctx.W.EmitShlRegImm8(r138, 32)
				ctx.W.EmitShrRegImm8(r138, 32)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d133)
			}
			var d134 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r139, thisptr.Reg, off)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r139}
				ctx.BindReg(r139, &d134)
			}
			ctx.EnsureDesc(&d134)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d134.Imm.Int()))))}
			} else {
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r140, d134.Reg)
				ctx.W.EmitShlRegImm8(r140, 56)
				ctx.W.EmitShrRegImm8(r140, 56)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d135)
			}
			ctx.FreeDesc(&d134)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d135)
			var d136 scm.JITValueDesc
			if d133.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d133.Imm.Int() * d135.Imm.Int())}
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d133.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d136)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegReg(scratch, d133.Reg)
				if d135.Imm.Int() >= -2147483648 && d135.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d135.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d135.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d136)
			} else {
				r141 := ctx.AllocRegExcept(d133.Reg, d135.Reg)
				ctx.W.EmitMovRegReg(r141, d133.Reg)
				ctx.W.EmitImulInt64(r141, d135.Reg)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d136)
			}
			if d136.Loc == scm.LocReg && d133.Loc == scm.LocReg && d136.Reg == d133.Reg {
				ctx.TransferReg(d133.Reg)
				d133.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d133)
			ctx.FreeDesc(&d135)
			var d137 scm.JITValueDesc
			r142 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r142, uint64(dataPtr))
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142, StackOff: int32(sliceLen)}
				ctx.BindReg(r142, &d137)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r142, thisptr.Reg, off)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142}
				ctx.BindReg(r142, &d137)
			}
			ctx.BindReg(r142, &d137)
			ctx.EnsureDesc(&d136)
			var d138 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() / 64)}
			} else {
				r143 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r143, d136.Reg)
				ctx.W.EmitShrRegImm8(r143, 6)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d138)
			}
			if d138.Loc == scm.LocReg && d136.Loc == scm.LocReg && d138.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d138)
			r144 := ctx.AllocReg()
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d137)
			if d138.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r144, uint64(d138.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r144, d138.Reg)
				ctx.W.EmitShlRegImm8(r144, 3)
			}
			if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(r144, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r144, d137.Reg)
			}
			r145 := ctx.AllocRegExcept(r144)
			ctx.W.EmitMovRegMem(r145, r144, 0)
			ctx.FreeReg(r144)
			d139 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			ctx.BindReg(r145, &d139)
			ctx.FreeDesc(&d138)
			ctx.EnsureDesc(&d136)
			var d140 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
			} else {
				r146 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r146, d136.Reg)
				ctx.W.EmitAndRegImm32(r146, 63)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d140)
			}
			if d140.Loc == scm.LocReg && d136.Loc == scm.LocReg && d140.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d139.Imm.Int()) << uint64(d140.Imm.Int())))}
			} else if d140.Loc == scm.LocImm {
				r147 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r147, d139.Reg)
				ctx.W.EmitShlRegImm8(r147, uint8(d140.Imm.Int()))
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d141)
			} else {
				{
					shiftSrc := d139.Reg
					r148 := ctx.AllocRegExcept(d139.Reg)
					ctx.W.EmitMovRegReg(r148, d139.Reg)
					shiftSrc = r148
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d140.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d140.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d140.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d141)
				}
			}
			if d141.Loc == scm.LocReg && d139.Loc == scm.LocReg && d141.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			ctx.FreeDesc(&d140)
			var d142 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r149, thisptr.Reg, off)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
				ctx.BindReg(r149, &d142)
			}
			d143 := d142
			ctx.EnsureDesc(&d143)
			if d143.Loc != scm.LocImm && d143.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			if d143.Loc == scm.LocImm {
				if d143.Imm.Bool() {
					ctx.W.MarkLabel(lbl47)
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.MarkLabel(lbl48)
			d144 := d141
			if d144.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d144)
			ctx.EmitStoreToStack(d144, 24)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d143.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl48)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
				ctx.W.MarkLabel(lbl48)
			d145 := d141
			if d145.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d145)
			ctx.EmitStoreToStack(d145, 24)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d142)
			ctx.W.MarkLabel(lbl46)
			d146 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d147)
			}
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d147)
			var d148 scm.JITValueDesc
			if d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d147.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d147.Reg)
				ctx.W.EmitShlRegImm8(r151, 56)
				ctx.W.EmitShrRegImm8(r151, 56)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d148)
			}
			ctx.FreeDesc(&d147)
			d149 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d148)
			ctx.EnsureDesc(&d149)
			ctx.EnsureDesc(&d148)
			var d150 scm.JITValueDesc
			if d149.Loc == scm.LocImm && d148.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() - d148.Imm.Int())}
			} else if d148.Loc == scm.LocImm && d148.Imm.Int() == 0 {
				r152 := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(r152, d149.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d150)
			} else if d149.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d149.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d148.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d150)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(scratch, d149.Reg)
				if d148.Imm.Int() >= -2147483648 && d148.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d148.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d148.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d150)
			} else {
				r153 := ctx.AllocRegExcept(d149.Reg, d148.Reg)
				ctx.W.EmitMovRegReg(r153, d149.Reg)
				ctx.W.EmitSubInt64(r153, d148.Reg)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d150)
			}
			if d150.Loc == scm.LocReg && d149.Loc == scm.LocReg && d150.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d148)
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d150)
			var d151 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d146.Imm.Int()) >> uint64(d150.Imm.Int())))}
			} else if d150.Loc == scm.LocImm {
				r154 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r154, d146.Reg)
				ctx.W.EmitShrRegImm8(r154, uint8(d150.Imm.Int()))
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d151)
			} else {
				{
					shiftSrc := d146.Reg
					r155 := ctx.AllocRegExcept(d146.Reg)
					ctx.W.EmitMovRegReg(r155, d146.Reg)
					shiftSrc = r155
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d150.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d150.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d150.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d151)
				}
			}
			if d151.Loc == scm.LocReg && d146.Loc == scm.LocReg && d151.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d150)
			r156 := ctx.AllocReg()
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			if d151.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r156, d151)
			}
			ctx.W.EmitJmp(lbl43)
			ctx.W.MarkLabel(lbl45)
			d146 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d136)
			var d152 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
			} else {
				r157 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r157, d136.Reg)
				ctx.W.EmitAndRegImm32(r157, 63)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d152)
			}
			if d152.Loc == scm.LocReg && d136.Loc == scm.LocReg && d152.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			var d153 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d153)
			}
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d153)
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d153.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d153.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d154)
			}
			ctx.FreeDesc(&d153)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d152.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() + d154.Imm.Int())}
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(r160, d152.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d155)
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d154.Reg}
				ctx.BindReg(d154.Reg, &d155)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d152.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d154.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d155)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(scratch, d152.Reg)
				if d154.Imm.Int() >= -2147483648 && d154.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d154.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d154.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d155)
			} else {
				r161 := ctx.AllocRegExcept(d152.Reg, d154.Reg)
				ctx.W.EmitMovRegReg(r161, d152.Reg)
				ctx.W.EmitAddInt64(r161, d154.Reg)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d155)
			}
			if d155.Loc == scm.LocReg && d152.Loc == scm.LocReg && d155.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.FreeDesc(&d154)
			ctx.EnsureDesc(&d155)
			var d156 scm.JITValueDesc
			if d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d155.Imm.Int()) > uint64(64))}
			} else {
				r162 := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitCmpRegImm32(d155.Reg, 64)
				ctx.W.EmitSetcc(r162, scm.CcA)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
				ctx.BindReg(r162, &d156)
			}
			ctx.FreeDesc(&d155)
			d157 := d156
			ctx.EnsureDesc(&d157)
			if d157.Loc != scm.LocImm && d157.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			if d157.Loc == scm.LocImm {
				if d157.Imm.Bool() {
					ctx.W.MarkLabel(lbl50)
					ctx.W.EmitJmp(lbl49)
				} else {
					ctx.W.MarkLabel(lbl51)
			d158 := d141
			if d158.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d158)
			ctx.EmitStoreToStack(d158, 24)
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d157.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl51)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl51)
			d159 := d141
			if d159.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d159)
			ctx.EmitStoreToStack(d159, 24)
				ctx.W.EmitJmp(lbl46)
			}
			ctx.FreeDesc(&d156)
			ctx.W.MarkLabel(lbl49)
			d146 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d136)
			var d160 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() / 64)}
			} else {
				r163 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r163, d136.Reg)
				ctx.W.EmitShrRegImm8(r163, 6)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d160)
			}
			if d160.Loc == scm.LocReg && d136.Loc == scm.LocReg && d160.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d160)
			ctx.EnsureDesc(&d160)
			var d161 scm.JITValueDesc
			if d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d160.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegReg(scratch, d160.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			}
			if d161.Loc == scm.LocReg && d160.Loc == scm.LocReg && d161.Reg == d160.Reg {
				ctx.TransferReg(d160.Reg)
				d160.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d160)
			ctx.EnsureDesc(&d161)
			r164 := ctx.AllocReg()
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d137)
			if d161.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r164, uint64(d161.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r164, d161.Reg)
				ctx.W.EmitShlRegImm8(r164, 3)
			}
			if d137.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(r164, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r164, d137.Reg)
			}
			r165 := ctx.AllocRegExcept(r164)
			ctx.W.EmitMovRegMem(r165, r164, 0)
			ctx.FreeReg(r164)
			d162 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
			ctx.BindReg(r165, &d162)
			ctx.FreeDesc(&d161)
			ctx.EnsureDesc(&d136)
			var d163 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d136.Imm.Int() % 64)}
			} else {
				r166 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r166, d136.Reg)
				ctx.W.EmitAndRegImm32(r166, 63)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d163)
			}
			if d163.Loc == scm.LocReg && d136.Loc == scm.LocReg && d163.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			d164 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d163)
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm && d163.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() - d163.Imm.Int())}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				r167 := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegReg(r167, d164.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d165)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d164.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d163.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegReg(scratch, d164.Reg)
				if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d163.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			} else {
				r168 := ctx.AllocRegExcept(d164.Reg, d163.Reg)
				ctx.W.EmitMovRegReg(r168, d164.Reg)
				ctx.W.EmitSubInt64(r168, d163.Reg)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d165)
			}
			if d165.Loc == scm.LocReg && d164.Loc == scm.LocReg && d165.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d165)
			var d166 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d162.Imm.Int()) >> uint64(d165.Imm.Int())))}
			} else if d165.Loc == scm.LocImm {
				r169 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(r169, d162.Reg)
				ctx.W.EmitShrRegImm8(r169, uint8(d165.Imm.Int()))
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d166)
			} else {
				{
					shiftSrc := d162.Reg
					r170 := ctx.AllocRegExcept(d162.Reg)
					ctx.W.EmitMovRegReg(r170, d162.Reg)
					shiftSrc = r170
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d165.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d165.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d165.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d166)
				}
			}
			if d166.Loc == scm.LocReg && d162.Loc == scm.LocReg && d166.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d165)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d166)
			var d167 scm.JITValueDesc
			if d141.Loc == scm.LocImm && d166.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d141.Imm.Int() | d166.Imm.Int())}
			} else if d141.Loc == scm.LocImm && d141.Imm.Int() == 0 {
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d166.Reg}
				ctx.BindReg(d166.Reg, &d167)
			} else if d166.Loc == scm.LocImm && d166.Imm.Int() == 0 {
				r171 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r171, d141.Reg)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d167)
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d141.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d166.Reg)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d167)
			} else if d166.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegReg(r172, d141.Reg)
				if d166.Imm.Int() >= -2147483648 && d166.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r172, int32(d166.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d166.Imm.Int()))
					ctx.W.EmitOrInt64(r172, scm.RegR11)
				}
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d167)
			} else {
				r173 := ctx.AllocRegExcept(d141.Reg, d166.Reg)
				ctx.W.EmitMovRegReg(r173, d141.Reg)
				ctx.W.EmitOrInt64(r173, d166.Reg)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d167)
			}
			if d167.Loc == scm.LocReg && d141.Loc == scm.LocReg && d167.Reg == d141.Reg {
				ctx.TransferReg(d141.Reg)
				d141.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			d168 := d167
			if d168.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d168)
			ctx.EmitStoreToStack(d168, 24)
			ctx.W.EmitJmp(lbl46)
			ctx.W.MarkLabel(lbl43)
			d169 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
			ctx.BindReg(r156, &d169)
			ctx.BindReg(r156, &d169)
			if r136 { ctx.UnprotectReg(r137) }
			ctx.EnsureDesc(&d169)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d169.Imm.Int()))))}
			} else {
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r174, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d170)
			}
			ctx.FreeDesc(&d169)
			var d171 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r175, thisptr.Reg, off)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r175}
				ctx.BindReg(r175, &d171)
			}
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d171)
			ctx.EnsureDesc(&d170)
			ctx.EnsureDesc(&d171)
			var d172 scm.JITValueDesc
			if d170.Loc == scm.LocImm && d171.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() + d171.Imm.Int())}
			} else if d171.Loc == scm.LocImm && d171.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r176, d170.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d172)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d171.Reg}
				ctx.BindReg(d171.Reg, &d172)
			} else if d170.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d171.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d170.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else if d171.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(scratch, d170.Reg)
				if d171.Imm.Int() >= -2147483648 && d171.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d171.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d172)
			} else {
				r177 := ctx.AllocRegExcept(d170.Reg, d171.Reg)
				ctx.W.EmitMovRegReg(r177, d170.Reg)
				ctx.W.EmitAddInt64(r177, d171.Reg)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d172)
			}
			if d172.Loc == scm.LocReg && d170.Loc == scm.LocReg && d172.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			ctx.FreeDesc(&d171)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d172)
			var d174 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d172.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d131.Imm.Int() + d172.Imm.Int())}
			} else if d172.Loc == scm.LocImm && d172.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r178, d131.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d174)
			} else if d131.Loc == scm.LocImm && d131.Imm.Int() == 0 {
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d172.Reg}
				ctx.BindReg(d172.Reg, &d174)
			} else if d131.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d172.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d131.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d174)
			} else if d172.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(scratch, d131.Reg)
				if d172.Imm.Int() >= -2147483648 && d172.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d172.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d172.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d174)
			} else {
				r179 := ctx.AllocRegExcept(d131.Reg, d172.Reg)
				ctx.W.EmitMovRegReg(r179, d131.Reg)
				ctx.W.EmitAddInt64(r179, d172.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d174)
			}
			if d174.Loc == scm.LocReg && d131.Loc == scm.LocReg && d174.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d172)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d174)
			var d176 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r180, fieldAddr)
				ctx.W.EmitMovRegMem64(r181, fieldAddr+8)
				d176 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d176)
				ctx.BindReg(r181, &d176)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r182 := ctx.AllocReg()
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r182, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off+8)
				d176 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
				ctx.BindReg(r182, &d176)
				ctx.BindReg(r183, &d176)
			}
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d174)
			r184 := ctx.AllocReg()
			r185 := ctx.AllocRegExcept(r184)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d131)
			ctx.EnsureDesc(&d174)
			if d176.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r184, uint64(d176.Imm.Int()))
			} else if d176.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r184, d176.Reg)
			} else {
				ctx.W.EmitMovRegReg(r184, d176.Reg)
			}
			if d131.Loc == scm.LocImm {
				if d131.Imm.Int() != 0 {
					if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r184, int32(d131.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
						ctx.W.EmitAddInt64(r184, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r184, d131.Reg)
			}
			if d174.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r185, uint64(d174.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r185, d174.Reg)
			}
			if d131.Loc == scm.LocImm {
				if d131.Imm.Int() >= -2147483648 && d131.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r185, int32(d131.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
					ctx.W.EmitSubInt64(r185, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r185, d131.Reg)
			}
			d177 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d177)
			ctx.BindReg(r185, &d177)
			ctx.FreeDesc(&d131)
			ctx.FreeDesc(&d174)
			r186 := ctx.AllocReg()
			r187 := ctx.AllocRegExcept(r186)
			d178 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d178)
			ctx.BindReg(r187, &d178)
			d179 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d177}, 2)
			ctx.EmitMovPairToResult(&d179, &d178)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl17)
			var d180 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r188, thisptr.Reg, off)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d180)
			}
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d180)
			var d181 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d180.Imm.Int()))))}
			} else {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r189, d180.Reg)
				ctx.W.EmitShlRegImm8(r189, 32)
				ctx.W.EmitShrRegImm8(r189, 32)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d181)
			}
			ctx.FreeDesc(&d180)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d181)
			var d182 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d181.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) == uint64(d181.Imm.Int()))}
			} else if d181.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d44.Reg)
				if d181.Imm.Int() >= -2147483648 && d181.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d44.Reg, int32(d181.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
					ctx.W.EmitCmpInt64(d44.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r190, scm.CcE)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r190}
				ctx.BindReg(r190, &d182)
			} else if d44.Loc == scm.LocImm {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d181.Reg)
				ctx.W.EmitSetcc(r191, scm.CcE)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r191}
				ctx.BindReg(r191, &d182)
			} else {
				r192 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitCmpInt64(d44.Reg, d181.Reg)
				ctx.W.EmitSetcc(r192, scm.CcE)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
				ctx.BindReg(r192, &d182)
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d181)
			d183 := d182
			ctx.EnsureDesc(&d183)
			if d183.Loc != scm.LocImm && d183.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d183.Loc == scm.LocImm {
				if d183.Imm.Bool() {
					ctx.W.MarkLabel(lbl53)
					ctx.W.EmitJmp(lbl52)
				} else {
					ctx.W.MarkLabel(lbl54)
					ctx.W.EmitJmp(lbl18)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d183.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
				ctx.W.MarkLabel(lbl53)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.FreeDesc(&d182)
			ctx.W.MarkLabel(lbl31)
			ctx.EnsureDesc(&d0)
			d184 := d0
			_ = d184
			r193 := d0.Loc == scm.LocReg
			r194 := d0.Reg
			if r193 { ctx.ProtectReg(r194) }
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl56)
			ctx.EnsureDesc(&d184)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d184.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r195, d184.Reg)
				ctx.W.EmitShlRegImm8(r195, 32)
				ctx.W.EmitShrRegImm8(r195, 32)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d185)
			}
			var d186 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r196 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r196, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r196}
				ctx.BindReg(r196, &d186)
			}
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d186)
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d186.Imm.Int()))))}
			} else {
				r197 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r197, d186.Reg)
				ctx.W.EmitShlRegImm8(r197, 56)
				ctx.W.EmitShrRegImm8(r197, 56)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d187)
			}
			ctx.FreeDesc(&d186)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d187)
			var d188 scm.JITValueDesc
			if d185.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() * d187.Imm.Int())}
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d185.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(scratch, d185.Reg)
				if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else {
				r198 := ctx.AllocRegExcept(d185.Reg, d187.Reg)
				ctx.W.EmitMovRegReg(r198, d185.Reg)
				ctx.W.EmitImulInt64(r198, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d188)
			}
			if d188.Loc == scm.LocReg && d185.Loc == scm.LocReg && d188.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			ctx.FreeDesc(&d187)
			var d189 scm.JITValueDesc
			r199 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r199, uint64(dataPtr))
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199, StackOff: int32(sliceLen)}
				ctx.BindReg(r199, &d189)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r199, thisptr.Reg, off)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d189)
			}
			ctx.BindReg(r199, &d189)
			ctx.EnsureDesc(&d188)
			var d190 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() / 64)}
			} else {
				r200 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r200, d188.Reg)
				ctx.W.EmitShrRegImm8(r200, 6)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d190)
			}
			if d190.Loc == scm.LocReg && d188.Loc == scm.LocReg && d190.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d190)
			r201 := ctx.AllocReg()
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d189)
			if d190.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r201, uint64(d190.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r201, d190.Reg)
				ctx.W.EmitShlRegImm8(r201, 3)
			}
			if d189.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d189.Imm.Int()))
				ctx.W.EmitAddInt64(r201, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r201, d189.Reg)
			}
			r202 := ctx.AllocRegExcept(r201)
			ctx.W.EmitMovRegMem(r202, r201, 0)
			ctx.FreeReg(r201)
			d191 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
			ctx.BindReg(r202, &d191)
			ctx.FreeDesc(&d190)
			ctx.EnsureDesc(&d188)
			var d192 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r203 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r203, d188.Reg)
				ctx.W.EmitAndRegImm32(r203, 63)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d192)
			}
			if d192.Loc == scm.LocReg && d188.Loc == scm.LocReg && d192.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d192)
			var d193 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d191.Imm.Int()) << uint64(d192.Imm.Int())))}
			} else if d192.Loc == scm.LocImm {
				r204 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r204, d191.Reg)
				ctx.W.EmitShlRegImm8(r204, uint8(d192.Imm.Int()))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d193)
			} else {
				{
					shiftSrc := d191.Reg
					r205 := ctx.AllocRegExcept(d191.Reg)
					ctx.W.EmitMovRegReg(r205, d191.Reg)
					shiftSrc = r205
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d192.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d192.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d192.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d193)
				}
			}
			if d193.Loc == scm.LocReg && d191.Loc == scm.LocReg && d193.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d191)
			ctx.FreeDesc(&d192)
			var d194 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d194)
			}
			d195 := d194
			ctx.EnsureDesc(&d195)
			if d195.Loc != scm.LocImm && d195.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			if d195.Loc == scm.LocImm {
				if d195.Imm.Bool() {
					ctx.W.MarkLabel(lbl59)
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.MarkLabel(lbl60)
			d196 := d193
			if d196.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d196)
			ctx.EmitStoreToStack(d196, 32)
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d195.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl59)
				ctx.W.EmitJmp(lbl57)
				ctx.W.MarkLabel(lbl60)
			d197 := d193
			if d197.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d197)
			ctx.EmitStoreToStack(d197, 32)
				ctx.W.EmitJmp(lbl58)
			}
			ctx.FreeDesc(&d194)
			ctx.W.MarkLabel(lbl58)
			d198 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d199 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r207, thisptr.Reg, off)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
				ctx.BindReg(r207, &d199)
			}
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d199)
			var d200 scm.JITValueDesc
			if d199.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d199.Imm.Int()))))}
			} else {
				r208 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r208, d199.Reg)
				ctx.W.EmitShlRegImm8(r208, 56)
				ctx.W.EmitShrRegImm8(r208, 56)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d200)
			}
			ctx.FreeDesc(&d199)
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
				r209 := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegReg(r209, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d202)
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
				r210 := ctx.AllocRegExcept(d201.Reg, d200.Reg)
				ctx.W.EmitMovRegReg(r210, d201.Reg)
				ctx.W.EmitSubInt64(r210, d200.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d202)
			}
			if d202.Loc == scm.LocReg && d201.Loc == scm.LocReg && d202.Reg == d201.Reg {
				ctx.TransferReg(d201.Reg)
				d201.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.EnsureDesc(&d198)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d198.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d198.Imm.Int()) >> uint64(d202.Imm.Int())))}
			} else if d202.Loc == scm.LocImm {
				r211 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(r211, d198.Reg)
				ctx.W.EmitShrRegImm8(r211, uint8(d202.Imm.Int()))
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d203)
			} else {
				{
					shiftSrc := d198.Reg
					r212 := ctx.AllocRegExcept(d198.Reg)
					ctx.W.EmitMovRegReg(r212, d198.Reg)
					shiftSrc = r212
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
			if d203.Loc == scm.LocReg && d198.Loc == scm.LocReg && d203.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d198)
			ctx.FreeDesc(&d202)
			r213 := ctx.AllocReg()
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d203)
			if d203.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r213, d203)
			}
			ctx.W.EmitJmp(lbl55)
			ctx.W.MarkLabel(lbl57)
			d198 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d188)
			var d204 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r214 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r214, d188.Reg)
				ctx.W.EmitAndRegImm32(r214, 63)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d204)
			}
			if d204.Loc == scm.LocReg && d188.Loc == scm.LocReg && d204.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			var d205 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r215 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r215, thisptr.Reg, off)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r215}
				ctx.BindReg(r215, &d205)
			}
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d205)
			var d206 scm.JITValueDesc
			if d205.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d205.Imm.Int()))))}
			} else {
				r216 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r216, d205.Reg)
				ctx.W.EmitShlRegImm8(r216, 56)
				ctx.W.EmitShrRegImm8(r216, 56)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d206)
			}
			ctx.FreeDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d206)
			var d207 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d206.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d204.Imm.Int() + d206.Imm.Int())}
			} else if d206.Loc == scm.LocImm && d206.Imm.Int() == 0 {
				r217 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(r217, d204.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d207)
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d206.Reg}
				ctx.BindReg(d206.Reg, &d207)
			} else if d204.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d204.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(scratch, d204.Reg)
				if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d206.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else {
				r218 := ctx.AllocRegExcept(d204.Reg, d206.Reg)
				ctx.W.EmitMovRegReg(r218, d204.Reg)
				ctx.W.EmitAddInt64(r218, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d207)
			}
			if d207.Loc == scm.LocReg && d204.Loc == scm.LocReg && d207.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d206)
			ctx.EnsureDesc(&d207)
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d207.Imm.Int()) > uint64(64))}
			} else {
				r219 := ctx.AllocRegExcept(d207.Reg)
				ctx.W.EmitCmpRegImm32(d207.Reg, 64)
				ctx.W.EmitSetcc(r219, scm.CcA)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r219}
				ctx.BindReg(r219, &d208)
			}
			ctx.FreeDesc(&d207)
			d209 := d208
			ctx.EnsureDesc(&d209)
			if d209.Loc != scm.LocImm && d209.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl61 := ctx.W.ReserveLabel()
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			if d209.Loc == scm.LocImm {
				if d209.Imm.Bool() {
					ctx.W.MarkLabel(lbl62)
					ctx.W.EmitJmp(lbl61)
				} else {
					ctx.W.MarkLabel(lbl63)
			d210 := d193
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, 32)
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d209.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl62)
				ctx.W.EmitJmp(lbl63)
				ctx.W.MarkLabel(lbl62)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl63)
			d211 := d193
			if d211.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d211)
			ctx.EmitStoreToStack(d211, 32)
				ctx.W.EmitJmp(lbl58)
			}
			ctx.FreeDesc(&d208)
			ctx.W.MarkLabel(lbl61)
			d198 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d188)
			var d212 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() / 64)}
			} else {
				r220 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r220, d188.Reg)
				ctx.W.EmitShrRegImm8(r220, 6)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d212)
			}
			if d212.Loc == scm.LocReg && d188.Loc == scm.LocReg && d212.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d212)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			ctx.EnsureDesc(&d213)
			r221 := ctx.AllocReg()
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d189)
			if d213.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r221, uint64(d213.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r221, d213.Reg)
				ctx.W.EmitShlRegImm8(r221, 3)
			}
			if d189.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d189.Imm.Int()))
				ctx.W.EmitAddInt64(r221, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r221, d189.Reg)
			}
			r222 := ctx.AllocRegExcept(r221)
			ctx.W.EmitMovRegMem(r222, r221, 0)
			ctx.FreeReg(r221)
			d214 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r222}
			ctx.BindReg(r222, &d214)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d188)
			var d215 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() % 64)}
			} else {
				r223 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r223, d188.Reg)
				ctx.W.EmitAndRegImm32(r223, 63)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d215)
			}
			if d215.Loc == scm.LocReg && d188.Loc == scm.LocReg && d215.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d188)
			d216 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d215)
			var d217 scm.JITValueDesc
			if d216.Loc == scm.LocImm && d215.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d216.Imm.Int() - d215.Imm.Int())}
			} else if d215.Loc == scm.LocImm && d215.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d216.Reg)
				ctx.W.EmitMovRegReg(r224, d216.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d217)
			} else if d216.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d216.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else if d215.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d216.Reg)
				ctx.W.EmitMovRegReg(scratch, d216.Reg)
				if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d215.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else {
				r225 := ctx.AllocRegExcept(d216.Reg, d215.Reg)
				ctx.W.EmitMovRegReg(r225, d216.Reg)
				ctx.W.EmitSubInt64(r225, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d217)
			}
			if d217.Loc == scm.LocReg && d216.Loc == scm.LocReg && d217.Reg == d216.Reg {
				ctx.TransferReg(d216.Reg)
				d216.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d214.Loc == scm.LocImm && d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d214.Imm.Int()) >> uint64(d217.Imm.Int())))}
			} else if d217.Loc == scm.LocImm {
				r226 := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegReg(r226, d214.Reg)
				ctx.W.EmitShrRegImm8(r226, uint8(d217.Imm.Int()))
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d218)
			} else {
				{
					shiftSrc := d214.Reg
					r227 := ctx.AllocRegExcept(d214.Reg)
					ctx.W.EmitMovRegReg(r227, d214.Reg)
					shiftSrc = r227
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d217.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d217.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d217.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d218)
				}
			}
			if d218.Loc == scm.LocReg && d214.Loc == scm.LocReg && d218.Reg == d214.Reg {
				ctx.TransferReg(d214.Reg)
				d214.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			ctx.FreeDesc(&d217)
			ctx.EnsureDesc(&d193)
			ctx.EnsureDesc(&d218)
			var d219 scm.JITValueDesc
			if d193.Loc == scm.LocImm && d218.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d193.Imm.Int() | d218.Imm.Int())}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d218.Reg}
				ctx.BindReg(d218.Reg, &d219)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r228, d193.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d219)
			} else if d193.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d193.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d218.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d219)
			} else if d218.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegReg(r229, d193.Reg)
				if d218.Imm.Int() >= -2147483648 && d218.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r229, int32(d218.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d218.Imm.Int()))
					ctx.W.EmitOrInt64(r229, scm.RegR11)
				}
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d219)
			} else {
				r230 := ctx.AllocRegExcept(d193.Reg, d218.Reg)
				ctx.W.EmitMovRegReg(r230, d193.Reg)
				ctx.W.EmitOrInt64(r230, d218.Reg)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d219)
			}
			if d219.Loc == scm.LocReg && d193.Loc == scm.LocReg && d219.Reg == d193.Reg {
				ctx.TransferReg(d193.Reg)
				d193.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			d220 := d219
			if d220.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d220)
			ctx.EmitStoreToStack(d220, 32)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl55)
			d221 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
			ctx.BindReg(r213, &d221)
			ctx.BindReg(r213, &d221)
			if r193 { ctx.UnprotectReg(r194) }
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d221)
			var d222 scm.JITValueDesc
			if d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d221.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d222)
			}
			ctx.FreeDesc(&d221)
			var d223 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r232, thisptr.Reg, off)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r232}
				ctx.BindReg(r232, &d223)
			}
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d222.Loc == scm.LocImm && d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() + d223.Imm.Int())}
			} else if d223.Loc == scm.LocImm && d223.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r233, d222.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d224)
			} else if d222.Loc == scm.LocImm && d222.Imm.Int() == 0 {
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d223.Reg}
				ctx.BindReg(d223.Reg, &d224)
			} else if d222.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d223.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d222.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d224)
			} else if d223.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(scratch, d222.Reg)
				if d223.Imm.Int() >= -2147483648 && d223.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d223.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d223.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d224)
			} else {
				r234 := ctx.AllocRegExcept(d222.Reg, d223.Reg)
				ctx.W.EmitMovRegReg(r234, d222.Reg)
				ctx.W.EmitAddInt64(r234, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d224)
			}
			if d224.Loc == scm.LocReg && d222.Loc == scm.LocReg && d224.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d222)
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d224)
			var d225 scm.JITValueDesc
			if d224.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d224.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d224.Reg)
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d225)
			}
			ctx.FreeDesc(&d224)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d226 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d88.Imm.Int()))))}
			} else {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r236, d88.Reg)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d226)
			}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d225)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d225)
			var d227 scm.JITValueDesc
			if d88.Loc == scm.LocImm && d225.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() + d225.Imm.Int())}
			} else if d225.Loc == scm.LocImm && d225.Imm.Int() == 0 {
				r237 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r237, d88.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d227)
			} else if d88.Loc == scm.LocImm && d88.Imm.Int() == 0 {
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d225.Reg}
				ctx.BindReg(d225.Reg, &d227)
			} else if d88.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d88.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d225.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			} else if d225.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(scratch, d88.Reg)
				if d225.Imm.Int() >= -2147483648 && d225.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d225.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d225.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d227)
			} else {
				r238 := ctx.AllocRegExcept(d88.Reg, d225.Reg)
				ctx.W.EmitMovRegReg(r238, d88.Reg)
				ctx.W.EmitAddInt64(r238, d225.Reg)
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d227)
			}
			if d227.Loc == scm.LocReg && d88.Loc == scm.LocReg && d227.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d225)
			ctx.EnsureDesc(&d227)
			ctx.EnsureDesc(&d227)
			var d228 scm.JITValueDesc
			if d227.Loc == scm.LocImm {
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d227.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r239, d227.Reg)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d228)
			}
			ctx.FreeDesc(&d227)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d228)
			r240 := ctx.AllocReg()
			r241 := ctx.AllocRegExcept(r240)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d226)
			ctx.EnsureDesc(&d228)
			if d176.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r240, uint64(d176.Imm.Int()))
			} else if d176.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r240, d176.Reg)
			} else {
				ctx.W.EmitMovRegReg(r240, d176.Reg)
			}
			if d226.Loc == scm.LocImm {
				if d226.Imm.Int() != 0 {
					if d226.Imm.Int() >= -2147483648 && d226.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r240, int32(d226.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
						ctx.W.EmitAddInt64(r240, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r240, d226.Reg)
			}
			if d228.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r241, uint64(d228.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r241, d228.Reg)
			}
			if d226.Loc == scm.LocImm {
				if d226.Imm.Int() >= -2147483648 && d226.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r241, int32(d226.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d226.Imm.Int()))
					ctx.W.EmitSubInt64(r241, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r241, d226.Reg)
			}
			d229 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r240, Reg2: r241}
			ctx.BindReg(r240, &d229)
			ctx.BindReg(r241, &d229)
			ctx.FreeDesc(&d226)
			ctx.FreeDesc(&d228)
			d230 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d230)
			ctx.BindReg(r187, &d230)
			d231 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d229}, 2)
			ctx.EmitMovPairToResult(&d231, &d230)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl30)
			var d232 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r242, thisptr.Reg, off)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d232)
			}
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d88.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d88.Imm.Int()) == uint64(d232.Imm.Int()))}
			} else if d232.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d88.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d88.Reg, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitCmpInt64(d88.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r243, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d233)
			} else if d88.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d88.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d232.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d233)
			} else {
				r245 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitCmpInt64(d88.Reg, d232.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d233)
			}
			ctx.FreeDesc(&d88)
			ctx.FreeDesc(&d232)
			d234 := d233
			ctx.EnsureDesc(&d234)
			if d234.Loc != scm.LocImm && d234.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			if d234.Loc == scm.LocImm {
				if d234.Imm.Bool() {
					ctx.W.MarkLabel(lbl65)
					ctx.W.EmitJmp(lbl64)
				} else {
					ctx.W.MarkLabel(lbl66)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d234.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl65)
				ctx.W.EmitJmp(lbl66)
				ctx.W.MarkLabel(lbl65)
				ctx.W.EmitJmp(lbl64)
				ctx.W.MarkLabel(lbl66)
				ctx.W.EmitJmp(lbl31)
			}
			ctx.FreeDesc(&d233)
			ctx.W.MarkLabel(lbl52)
			d235 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d235)
			ctx.BindReg(r187, &d235)
			ctx.W.EmitMakeNil(d235)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl64)
			d236 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d236)
			ctx.BindReg(r187, &d236)
			ctx.W.EmitMakeNil(d236)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl2)
			d237 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d237)
			ctx.BindReg(r187, &d237)
			ctx.BindReg(r186, &d237)
			ctx.BindReg(r187, &d237)
			if r2 { ctx.UnprotectReg(r3) }
			d239 := d237
			d239.ID = 0
			d238 := ctx.EmitTagEquals(&d239, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			d240 := d238
			ctx.EnsureDesc(&d240)
			if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl67 := ctx.W.ReserveLabel()
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
					ctx.W.MarkLabel(lbl69)
					ctx.W.EmitJmp(lbl67)
				} else {
					ctx.W.MarkLabel(lbl70)
					ctx.W.EmitJmp(lbl68)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d240.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl69)
				ctx.W.EmitJmp(lbl70)
				ctx.W.MarkLabel(lbl69)
				ctx.W.EmitJmp(lbl67)
				ctx.W.MarkLabel(lbl70)
				ctx.W.EmitJmp(lbl68)
			}
			ctx.FreeDesc(&d238)
			ctx.W.MarkLabel(lbl68)
			d242 := d237
			d242.ID = 0
			d241 := ctx.EmitTagEquals(&d242, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			d243 := d241
			ctx.EnsureDesc(&d243)
			if d243.Loc != scm.LocImm && d243.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			if d243.Loc == scm.LocImm {
				if d243.Imm.Bool() {
					ctx.W.MarkLabel(lbl73)
					ctx.W.EmitJmp(lbl71)
				} else {
					ctx.W.MarkLabel(lbl74)
					ctx.W.EmitJmp(lbl72)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d243.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl73)
				ctx.W.EmitJmp(lbl74)
				ctx.W.MarkLabel(lbl73)
				ctx.W.EmitJmp(lbl71)
				ctx.W.MarkLabel(lbl74)
				ctx.W.EmitJmp(lbl72)
			}
			ctx.FreeDesc(&d241)
			ctx.W.MarkLabel(lbl67)
			d244 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d244)
			ctx.BindReg(r1, &d244)
			ctx.W.EmitMakeNil(d244)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl72)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl71)
			ctx.EnsureDesc(&idxInt)
			d245 := idxInt
			_ = d245
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl76)
			ctx.EnsureDesc(&d245)
			ctx.EnsureDesc(&d245)
			var d246 scm.JITValueDesc
			if d245.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d245.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r248, d245.Reg)
				ctx.W.EmitShlRegImm8(r248, 32)
				ctx.W.EmitShrRegImm8(r248, 32)
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d246)
			}
			var d247 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r249, thisptr.Reg, off)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d247)
			}
			ctx.EnsureDesc(&d247)
			ctx.EnsureDesc(&d247)
			var d248 scm.JITValueDesc
			if d247.Loc == scm.LocImm {
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d247.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r250, d247.Reg)
				ctx.W.EmitShlRegImm8(r250, 56)
				ctx.W.EmitShrRegImm8(r250, 56)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d248)
			}
			ctx.FreeDesc(&d247)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d246)
			ctx.EnsureDesc(&d248)
			var d249 scm.JITValueDesc
			if d246.Loc == scm.LocImm && d248.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d246.Imm.Int() * d248.Imm.Int())}
			} else if d246.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d246.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d248.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else if d248.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(scratch, d246.Reg)
				if d248.Imm.Int() >= -2147483648 && d248.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d248.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d248.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else {
				r251 := ctx.AllocRegExcept(d246.Reg, d248.Reg)
				ctx.W.EmitMovRegReg(r251, d246.Reg)
				ctx.W.EmitImulInt64(r251, d248.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d249)
			}
			if d249.Loc == scm.LocReg && d246.Loc == scm.LocReg && d249.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			ctx.FreeDesc(&d248)
			var d250 scm.JITValueDesc
			r252 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r252, uint64(dataPtr))
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252, StackOff: int32(sliceLen)}
				ctx.BindReg(r252, &d250)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.W.EmitMovRegMem(r252, thisptr.Reg, off)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
				ctx.BindReg(r252, &d250)
			}
			ctx.BindReg(r252, &d250)
			ctx.EnsureDesc(&d249)
			var d251 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() / 64)}
			} else {
				r253 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r253, d249.Reg)
				ctx.W.EmitShrRegImm8(r253, 6)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d251)
			}
			if d251.Loc == scm.LocReg && d249.Loc == scm.LocReg && d251.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d251)
			r254 := ctx.AllocReg()
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			if d251.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r254, uint64(d251.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r254, d251.Reg)
				ctx.W.EmitShlRegImm8(r254, 3)
			}
			if d250.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
				ctx.W.EmitAddInt64(r254, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r254, d250.Reg)
			}
			r255 := ctx.AllocRegExcept(r254)
			ctx.W.EmitMovRegMem(r255, r254, 0)
			ctx.FreeReg(r254)
			d252 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
			ctx.BindReg(r255, &d252)
			ctx.FreeDesc(&d251)
			ctx.EnsureDesc(&d249)
			var d253 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r256 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r256, d249.Reg)
				ctx.W.EmitAndRegImm32(r256, 63)
				d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d253)
			}
			if d253.Loc == scm.LocReg && d249.Loc == scm.LocReg && d253.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d252)
			ctx.EnsureDesc(&d253)
			var d254 scm.JITValueDesc
			if d252.Loc == scm.LocImm && d253.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d252.Imm.Int()) << uint64(d253.Imm.Int())))}
			} else if d253.Loc == scm.LocImm {
				r257 := ctx.AllocRegExcept(d252.Reg)
				ctx.W.EmitMovRegReg(r257, d252.Reg)
				ctx.W.EmitShlRegImm8(r257, uint8(d253.Imm.Int()))
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d254)
			} else {
				{
					shiftSrc := d252.Reg
					r258 := ctx.AllocRegExcept(d252.Reg)
					ctx.W.EmitMovRegReg(r258, d252.Reg)
					shiftSrc = r258
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d253.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d253.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d253.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d254)
				}
			}
			if d254.Loc == scm.LocReg && d252.Loc == scm.LocReg && d254.Reg == d252.Reg {
				ctx.TransferReg(d252.Reg)
				d252.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d252)
			ctx.FreeDesc(&d253)
			var d255 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r259 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r259, thisptr.Reg, off)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r259}
				ctx.BindReg(r259, &d255)
			}
			d256 := d255
			ctx.EnsureDesc(&d256)
			if d256.Loc != scm.LocImm && d256.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			lbl80 := ctx.W.ReserveLabel()
			if d256.Loc == scm.LocImm {
				if d256.Imm.Bool() {
					ctx.W.MarkLabel(lbl79)
					ctx.W.EmitJmp(lbl77)
				} else {
					ctx.W.MarkLabel(lbl80)
			d257 := d254
			if d257.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d257)
			ctx.EmitStoreToStack(d257, 40)
					ctx.W.EmitJmp(lbl78)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d256.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl79)
				ctx.W.EmitJmp(lbl80)
				ctx.W.MarkLabel(lbl79)
				ctx.W.EmitJmp(lbl77)
				ctx.W.MarkLabel(lbl80)
			d258 := d254
			if d258.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d258)
			ctx.EmitStoreToStack(d258, 40)
				ctx.W.EmitJmp(lbl78)
			}
			ctx.FreeDesc(&d255)
			ctx.W.MarkLabel(lbl78)
			d259 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			var d260 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r260 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r260, thisptr.Reg, off)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d260)
			}
			ctx.EnsureDesc(&d260)
			ctx.EnsureDesc(&d260)
			var d261 scm.JITValueDesc
			if d260.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d260.Imm.Int()))))}
			} else {
				r261 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r261, d260.Reg)
				ctx.W.EmitShlRegImm8(r261, 56)
				ctx.W.EmitShrRegImm8(r261, 56)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
				ctx.BindReg(r261, &d261)
			}
			ctx.FreeDesc(&d260)
			d262 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d261)
			var d263 scm.JITValueDesc
			if d262.Loc == scm.LocImm && d261.Loc == scm.LocImm {
				d263 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d262.Imm.Int() - d261.Imm.Int())}
			} else if d261.Loc == scm.LocImm && d261.Imm.Int() == 0 {
				r262 := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(r262, d262.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
				ctx.BindReg(r262, &d263)
			} else if d262.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d262.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d261.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else if d261.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d262.Reg)
				ctx.W.EmitMovRegReg(scratch, d262.Reg)
				if d261.Imm.Int() >= -2147483648 && d261.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d261.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d261.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d263)
			} else {
				r263 := ctx.AllocRegExcept(d262.Reg, d261.Reg)
				ctx.W.EmitMovRegReg(r263, d262.Reg)
				ctx.W.EmitSubInt64(r263, d261.Reg)
				d263 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d263)
			}
			if d263.Loc == scm.LocReg && d262.Loc == scm.LocReg && d263.Reg == d262.Reg {
				ctx.TransferReg(d262.Reg)
				d262.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d261)
			ctx.EnsureDesc(&d259)
			ctx.EnsureDesc(&d263)
			var d264 scm.JITValueDesc
			if d259.Loc == scm.LocImm && d263.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d259.Imm.Int()) >> uint64(d263.Imm.Int())))}
			} else if d263.Loc == scm.LocImm {
				r264 := ctx.AllocRegExcept(d259.Reg)
				ctx.W.EmitMovRegReg(r264, d259.Reg)
				ctx.W.EmitShrRegImm8(r264, uint8(d263.Imm.Int()))
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d264)
			} else {
				{
					shiftSrc := d259.Reg
					r265 := ctx.AllocRegExcept(d259.Reg)
					ctx.W.EmitMovRegReg(r265, d259.Reg)
					shiftSrc = r265
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d263.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d263.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d263.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d264)
				}
			}
			if d264.Loc == scm.LocReg && d259.Loc == scm.LocReg && d264.Reg == d259.Reg {
				ctx.TransferReg(d259.Reg)
				d259.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d259)
			ctx.FreeDesc(&d263)
			r266 := ctx.AllocReg()
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d264)
			if d264.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r266, d264)
			}
			ctx.W.EmitJmp(lbl75)
			ctx.W.MarkLabel(lbl77)
			d259 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d249)
			var d265 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r267 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r267, d249.Reg)
				ctx.W.EmitAndRegImm32(r267, 63)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d265)
			}
			if d265.Loc == scm.LocReg && d249.Loc == scm.LocReg && d265.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			var d266 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r268 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r268, thisptr.Reg, off)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
				ctx.BindReg(r268, &d266)
			}
			ctx.EnsureDesc(&d266)
			ctx.EnsureDesc(&d266)
			var d267 scm.JITValueDesc
			if d266.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d266.Imm.Int()))))}
			} else {
				r269 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r269, d266.Reg)
				ctx.W.EmitShlRegImm8(r269, 56)
				ctx.W.EmitShrRegImm8(r269, 56)
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d267)
			}
			ctx.FreeDesc(&d266)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d267)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d265.Loc == scm.LocImm && d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d265.Imm.Int() + d267.Imm.Int())}
			} else if d267.Loc == scm.LocImm && d267.Imm.Int() == 0 {
				r270 := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(r270, d265.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
				ctx.BindReg(r270, &d268)
			} else if d265.Loc == scm.LocImm && d265.Imm.Int() == 0 {
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d267.Reg}
				ctx.BindReg(d267.Reg, &d268)
			} else if d265.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d267.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d265.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			} else if d267.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(scratch, d265.Reg)
				if d267.Imm.Int() >= -2147483648 && d267.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d267.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			} else {
				r271 := ctx.AllocRegExcept(d265.Reg, d267.Reg)
				ctx.W.EmitMovRegReg(r271, d265.Reg)
				ctx.W.EmitAddInt64(r271, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d268)
			}
			if d268.Loc == scm.LocReg && d265.Loc == scm.LocReg && d268.Reg == d265.Reg {
				ctx.TransferReg(d265.Reg)
				d265.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d265)
			ctx.FreeDesc(&d267)
			ctx.EnsureDesc(&d268)
			var d269 scm.JITValueDesc
			if d268.Loc == scm.LocImm {
				d269 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d268.Imm.Int()) > uint64(64))}
			} else {
				r272 := ctx.AllocRegExcept(d268.Reg)
				ctx.W.EmitCmpRegImm32(d268.Reg, 64)
				ctx.W.EmitSetcc(r272, scm.CcA)
				d269 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r272}
				ctx.BindReg(r272, &d269)
			}
			ctx.FreeDesc(&d268)
			d270 := d269
			ctx.EnsureDesc(&d270)
			if d270.Loc != scm.LocImm && d270.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			lbl83 := ctx.W.ReserveLabel()
			if d270.Loc == scm.LocImm {
				if d270.Imm.Bool() {
					ctx.W.MarkLabel(lbl82)
					ctx.W.EmitJmp(lbl81)
				} else {
					ctx.W.MarkLabel(lbl83)
			d271 := d254
			if d271.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d271)
			ctx.EmitStoreToStack(d271, 40)
					ctx.W.EmitJmp(lbl78)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d270.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl82)
				ctx.W.EmitJmp(lbl83)
				ctx.W.MarkLabel(lbl82)
				ctx.W.EmitJmp(lbl81)
				ctx.W.MarkLabel(lbl83)
			d272 := d254
			if d272.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d272)
			ctx.EmitStoreToStack(d272, 40)
				ctx.W.EmitJmp(lbl78)
			}
			ctx.FreeDesc(&d269)
			ctx.W.MarkLabel(lbl81)
			d259 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d249)
			var d273 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() / 64)}
			} else {
				r273 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r273, d249.Reg)
				ctx.W.EmitShrRegImm8(r273, 6)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d273)
			}
			if d273.Loc == scm.LocReg && d249.Loc == scm.LocReg && d273.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d273)
			var d274 scm.JITValueDesc
			if d273.Loc == scm.LocImm {
				d274 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d273.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d273.Reg)
				ctx.W.EmitMovRegReg(scratch, d273.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d274 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d274)
			}
			if d274.Loc == scm.LocReg && d273.Loc == scm.LocReg && d274.Reg == d273.Reg {
				ctx.TransferReg(d273.Reg)
				d273.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d273)
			ctx.EnsureDesc(&d274)
			r274 := ctx.AllocReg()
			ctx.EnsureDesc(&d274)
			ctx.EnsureDesc(&d250)
			if d274.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r274, uint64(d274.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r274, d274.Reg)
				ctx.W.EmitShlRegImm8(r274, 3)
			}
			if d250.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
				ctx.W.EmitAddInt64(r274, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r274, d250.Reg)
			}
			r275 := ctx.AllocRegExcept(r274)
			ctx.W.EmitMovRegMem(r275, r274, 0)
			ctx.FreeReg(r274)
			d275 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r275}
			ctx.BindReg(r275, &d275)
			ctx.FreeDesc(&d274)
			ctx.EnsureDesc(&d249)
			var d276 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d276 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d249.Imm.Int() % 64)}
			} else {
				r276 := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegReg(r276, d249.Reg)
				ctx.W.EmitAndRegImm32(r276, 63)
				d276 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
				ctx.BindReg(r276, &d276)
			}
			if d276.Loc == scm.LocReg && d249.Loc == scm.LocReg && d276.Reg == d249.Reg {
				ctx.TransferReg(d249.Reg)
				d249.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d249)
			d277 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d276)
			ctx.EnsureDesc(&d277)
			ctx.EnsureDesc(&d276)
			var d278 scm.JITValueDesc
			if d277.Loc == scm.LocImm && d276.Loc == scm.LocImm {
				d278 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d277.Imm.Int() - d276.Imm.Int())}
			} else if d276.Loc == scm.LocImm && d276.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d277.Reg)
				ctx.W.EmitMovRegReg(r277, d277.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d278)
			} else if d277.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d276.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d277.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d276.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d278)
			} else if d276.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d277.Reg)
				ctx.W.EmitMovRegReg(scratch, d277.Reg)
				if d276.Imm.Int() >= -2147483648 && d276.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d276.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d276.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d278)
			} else {
				r278 := ctx.AllocRegExcept(d277.Reg, d276.Reg)
				ctx.W.EmitMovRegReg(r278, d277.Reg)
				ctx.W.EmitSubInt64(r278, d276.Reg)
				d278 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d278)
			}
			if d278.Loc == scm.LocReg && d277.Loc == scm.LocReg && d278.Reg == d277.Reg {
				ctx.TransferReg(d277.Reg)
				d277.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d276)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d278)
			var d279 scm.JITValueDesc
			if d275.Loc == scm.LocImm && d278.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d275.Imm.Int()) >> uint64(d278.Imm.Int())))}
			} else if d278.Loc == scm.LocImm {
				r279 := ctx.AllocRegExcept(d275.Reg)
				ctx.W.EmitMovRegReg(r279, d275.Reg)
				ctx.W.EmitShrRegImm8(r279, uint8(d278.Imm.Int()))
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d279)
			} else {
				{
					shiftSrc := d275.Reg
					r280 := ctx.AllocRegExcept(d275.Reg)
					ctx.W.EmitMovRegReg(r280, d275.Reg)
					shiftSrc = r280
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d278.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d278.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d278.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d279)
				}
			}
			if d279.Loc == scm.LocReg && d275.Loc == scm.LocReg && d279.Reg == d275.Reg {
				ctx.TransferReg(d275.Reg)
				d275.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d275)
			ctx.FreeDesc(&d278)
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d279)
			var d280 scm.JITValueDesc
			if d254.Loc == scm.LocImm && d279.Loc == scm.LocImm {
				d280 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d254.Imm.Int() | d279.Imm.Int())}
			} else if d254.Loc == scm.LocImm && d254.Imm.Int() == 0 {
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d279.Reg}
				ctx.BindReg(d279.Reg, &d280)
			} else if d279.Loc == scm.LocImm && d279.Imm.Int() == 0 {
				r281 := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(r281, d254.Reg)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d280)
			} else if d254.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d279.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d254.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d279.Reg)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d280)
			} else if d279.Loc == scm.LocImm {
				r282 := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(r282, d254.Reg)
				if d279.Imm.Int() >= -2147483648 && d279.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r282, int32(d279.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d279.Imm.Int()))
					ctx.W.EmitOrInt64(r282, scm.RegR11)
				}
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d280)
			} else {
				r283 := ctx.AllocRegExcept(d254.Reg, d279.Reg)
				ctx.W.EmitMovRegReg(r283, d254.Reg)
				ctx.W.EmitOrInt64(r283, d279.Reg)
				d280 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d280)
			}
			if d280.Loc == scm.LocReg && d254.Loc == scm.LocReg && d280.Reg == d254.Reg {
				ctx.TransferReg(d254.Reg)
				d254.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d279)
			d281 := d280
			if d281.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d281)
			ctx.EmitStoreToStack(d281, 40)
			ctx.W.EmitJmp(lbl78)
			ctx.W.MarkLabel(lbl75)
			d282 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r266}
			ctx.BindReg(r266, &d282)
			ctx.BindReg(r266, &d282)
			if r246 { ctx.UnprotectReg(r247) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d282)
			ctx.EnsureDesc(&d282)
			var d283 scm.JITValueDesc
			if d282.Loc == scm.LocImm {
				d283 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d282.Imm.Int()))))}
			} else {
				r284 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r284, d282.Reg)
				d283 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d283)
			}
			ctx.FreeDesc(&d282)
			var d284 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d284 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r285 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r285, thisptr.Reg, off)
				d284 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
				ctx.BindReg(r285, &d284)
			}
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d284)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d284)
			ctx.EnsureDesc(&d283)
			ctx.EnsureDesc(&d284)
			var d285 scm.JITValueDesc
			if d283.Loc == scm.LocImm && d284.Loc == scm.LocImm {
				d285 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d283.Imm.Int() + d284.Imm.Int())}
			} else if d284.Loc == scm.LocImm && d284.Imm.Int() == 0 {
				r286 := ctx.AllocRegExcept(d283.Reg)
				ctx.W.EmitMovRegReg(r286, d283.Reg)
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d285)
			} else if d283.Loc == scm.LocImm && d283.Imm.Int() == 0 {
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d284.Reg}
				ctx.BindReg(d284.Reg, &d285)
			} else if d283.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d284.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d283.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d284.Reg)
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d285)
			} else if d284.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d283.Reg)
				ctx.W.EmitMovRegReg(scratch, d283.Reg)
				if d284.Imm.Int() >= -2147483648 && d284.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d284.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d284.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d285)
			} else {
				r287 := ctx.AllocRegExcept(d283.Reg, d284.Reg)
				ctx.W.EmitMovRegReg(r287, d283.Reg)
				ctx.W.EmitAddInt64(r287, d284.Reg)
				d285 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
				ctx.BindReg(r287, &d285)
			}
			if d285.Loc == scm.LocReg && d283.Loc == scm.LocReg && d285.Reg == d283.Reg {
				ctx.TransferReg(d283.Reg)
				d283.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d283)
			ctx.FreeDesc(&d284)
			var d286 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r288 := ctx.AllocReg()
				r289 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r288, fieldAddr)
				ctx.W.EmitMovRegMem64(r289, fieldAddr+8)
				d286 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r288, Reg2: r289}
				ctx.BindReg(r288, &d286)
				ctx.BindReg(r289, &d286)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r290, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r291, thisptr.Reg, off+8)
				d286 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d286)
				ctx.BindReg(r291, &d286)
			}
			var d287 scm.JITValueDesc
			if d286.Loc == scm.LocImm {
				d287 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d286.StackOff))}
			} else {
				ctx.EnsureDesc(&d286)
				if d286.Loc == scm.LocRegPair {
					d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d286.Reg2}
					ctx.BindReg(d286.Reg2, &d287)
					ctx.BindReg(d286.Reg2, &d287)
				} else if d286.Loc == scm.LocReg {
					d287 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d286.Reg}
					ctx.BindReg(d286.Reg, &d287)
					ctx.BindReg(d286.Reg, &d287)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d287)
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d287)
			var d289 scm.JITValueDesc
			if d285.Loc == scm.LocImm && d287.Loc == scm.LocImm {
				d289 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d285.Imm.Int() >= d287.Imm.Int())}
			} else if d287.Loc == scm.LocImm {
				r292 := ctx.AllocRegExcept(d285.Reg)
				if d287.Imm.Int() >= -2147483648 && d287.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d285.Reg, int32(d287.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d287.Imm.Int()))
					ctx.W.EmitCmpInt64(d285.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r292, scm.CcGE)
				d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r292}
				ctx.BindReg(r292, &d289)
			} else if d285.Loc == scm.LocImm {
				r293 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d285.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d287.Reg)
				ctx.W.EmitSetcc(r293, scm.CcGE)
				d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r293}
				ctx.BindReg(r293, &d289)
			} else {
				r294 := ctx.AllocRegExcept(d285.Reg)
				ctx.W.EmitCmpInt64(d285.Reg, d287.Reg)
				ctx.W.EmitSetcc(r294, scm.CcGE)
				d289 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d289)
			}
			ctx.FreeDesc(&d287)
			d290 := d289
			ctx.EnsureDesc(&d290)
			if d290.Loc != scm.LocImm && d290.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl84 := ctx.W.ReserveLabel()
			lbl85 := ctx.W.ReserveLabel()
			lbl86 := ctx.W.ReserveLabel()
			lbl87 := ctx.W.ReserveLabel()
			if d290.Loc == scm.LocImm {
				if d290.Imm.Bool() {
					ctx.W.MarkLabel(lbl86)
					ctx.W.EmitJmp(lbl84)
				} else {
					ctx.W.MarkLabel(lbl87)
					ctx.W.EmitJmp(lbl85)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d290.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl86)
				ctx.W.EmitJmp(lbl87)
				ctx.W.MarkLabel(lbl86)
				ctx.W.EmitJmp(lbl84)
				ctx.W.MarkLabel(lbl87)
				ctx.W.EmitJmp(lbl85)
			}
			ctx.FreeDesc(&d289)
			ctx.W.MarkLabel(lbl85)
			ctx.EnsureDesc(&d285)
			var d291 scm.JITValueDesc
			if d285.Loc == scm.LocImm {
				d291 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d285.Imm.Int() < 0)}
			} else {
				r295 := ctx.AllocRegExcept(d285.Reg)
				ctx.W.EmitCmpRegImm32(d285.Reg, 0)
				ctx.W.EmitSetcc(r295, scm.CcL)
				d291 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r295}
				ctx.BindReg(r295, &d291)
			}
			d292 := d291
			ctx.EnsureDesc(&d292)
			if d292.Loc != scm.LocImm && d292.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl88 := ctx.W.ReserveLabel()
			lbl89 := ctx.W.ReserveLabel()
			lbl90 := ctx.W.ReserveLabel()
			if d292.Loc == scm.LocImm {
				if d292.Imm.Bool() {
					ctx.W.MarkLabel(lbl89)
					ctx.W.EmitJmp(lbl84)
				} else {
					ctx.W.MarkLabel(lbl90)
					ctx.W.EmitJmp(lbl88)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d292.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl89)
				ctx.W.EmitJmp(lbl90)
				ctx.W.MarkLabel(lbl89)
				ctx.W.EmitJmp(lbl84)
				ctx.W.MarkLabel(lbl90)
				ctx.W.EmitJmp(lbl88)
			}
			ctx.FreeDesc(&d291)
			ctx.W.MarkLabel(lbl84)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl88)
			ctx.EnsureDesc(&d285)
			r296 := ctx.AllocReg()
			ctx.EnsureDesc(&d285)
			ctx.EnsureDesc(&d286)
			if d285.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r296, uint64(d285.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r296, d285.Reg)
				ctx.W.EmitShlRegImm8(r296, 4)
			}
			if d286.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d286.Imm.Int()))
				ctx.W.EmitAddInt64(r296, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r296, d286.Reg)
			}
			r297 := ctx.AllocRegExcept(r296)
			r298 := ctx.AllocRegExcept(r296, r297)
			ctx.W.EmitMovRegMem(r297, r296, 0)
			ctx.W.EmitMovRegMem(r298, r296, 8)
			ctx.FreeReg(r296)
			d293 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r297, Reg2: r298}
			ctx.BindReg(r297, &d293)
			ctx.BindReg(r298, &d293)
			ctx.FreeDesc(&d285)
			if d237.Loc != scm.LocImm && d237.Type == scm.JITTypeUnknown {
				panic("jit: scm.Scmer.String on unknown dynamic type")
			}
			d294 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d237}, 2)
			ctx.FreeDesc(&d237)
			ctx.EnsureDesc(&d293)
			ctx.EnsureDesc(&d294)
			d295 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d293, d294}, 2)
			ctx.FreeDesc(&d293)
			d296 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d296)
			ctx.BindReg(r1, &d296)
			d297 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d295}, 2)
			ctx.EmitMovPairToResult(&d297, &d296)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d298 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d298)
			ctx.BindReg(r1, &d298)
			ctx.EmitMovPairToResult(&d298, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r7, int32(48))
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
