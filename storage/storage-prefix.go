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
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl6)
				ctx.W.EmitJmp(lbl7)
			}
			ctx.W.MarkLabel(lbl6)
			ctx.W.EmitJmp(lbl4)
			ctx.W.MarkLabel(lbl7)
			ctx.W.EmitJmp(lbl5)
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
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d14.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl12)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.W.MarkLabel(lbl12)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl13)
			d15 := d12
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
			ctx.W.EmitJmp(lbl11)
			ctx.FreeDesc(&d13)
			ctx.W.MarkLabel(lbl11)
			d16 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r20, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
				ctx.BindReg(r20, &d17)
			}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d17)
			var d18 scm.JITValueDesc
			if d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d17.Imm.Int()))))}
			} else {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r21, d17.Reg)
				ctx.W.EmitShlRegImm8(r21, 56)
				ctx.W.EmitShrRegImm8(r21, 56)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d18)
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
				r22 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r22, d19.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d20)
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
				r23 := ctx.AllocRegExcept(d19.Reg, d18.Reg)
				ctx.W.EmitMovRegReg(r23, d19.Reg)
				ctx.W.EmitSubInt64(r23, d18.Reg)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d20)
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
				r24 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r24, d16.Reg)
				ctx.W.EmitShrRegImm8(r24, uint8(d20.Imm.Int()))
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d21)
			} else {
				{
					shiftSrc := d16.Reg
					r25 := ctx.AllocRegExcept(d16.Reg)
					ctx.W.EmitMovRegReg(r25, d16.Reg)
					shiftSrc = r25
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
			r26 := ctx.AllocReg()
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			if d21.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r26, d21)
			}
			ctx.W.EmitJmp(lbl8)
			ctx.W.MarkLabel(lbl10)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d7)
			var d22 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() % 64)}
			} else {
				r27 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r27, d7.Reg)
				ctx.W.EmitAndRegImm32(r27, 63)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d22)
			}
			if d22.Loc == scm.LocReg && d7.Loc == scm.LocReg && d22.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			var d23 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r28, thisptr.Reg, off)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
				ctx.BindReg(r28, &d23)
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d23.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r29, d23.Reg)
				ctx.W.EmitShlRegImm8(r29, 56)
				ctx.W.EmitShrRegImm8(r29, 56)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d24)
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
				r30 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r30, d22.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d25)
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
				r31 := ctx.AllocRegExcept(d22.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r31, d22.Reg)
				ctx.W.EmitAddInt64(r31, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d25)
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
				r32 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitCmpRegImm32(d25.Reg, 64)
				ctx.W.EmitSetcc(r32, scm.CcA)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d26)
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
					ctx.W.EmitJmp(lbl15)
				} else {
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl15)
				ctx.W.EmitJmp(lbl16)
			}
			ctx.W.MarkLabel(lbl15)
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl16)
			d28 := d12
			if d28.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d28)
			ctx.EmitStoreToStack(d28, 0)
			ctx.W.EmitJmp(lbl11)
			ctx.FreeDesc(&d26)
			ctx.W.MarkLabel(lbl14)
			d16 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d7)
			var d29 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r33, d7.Reg)
				ctx.W.EmitShrRegImm8(r33, 6)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d29)
			}
			if d29.Loc == scm.LocReg && d7.Loc == scm.LocReg && d29.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(scratch, d29.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			}
			if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d30)
			r34 := ctx.AllocReg()
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d8)
			if d30.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r34, uint64(d30.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r34, d30.Reg)
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
			d31 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d31)
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d7)
			var d32 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r36, d7.Reg)
				ctx.W.EmitAndRegImm32(r36, 63)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d32)
			}
			if d32.Loc == scm.LocReg && d7.Loc == scm.LocReg && d32.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d7)
			d33 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d33)
			ctx.EnsureDesc(&d32)
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() - d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r37, d33.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d34)
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(scratch, d33.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d34)
			} else {
				r38 := ctx.AllocRegExcept(d33.Reg, d32.Reg)
				ctx.W.EmitMovRegReg(r38, d33.Reg)
				ctx.W.EmitSubInt64(r38, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d34)
			}
			if d34.Loc == scm.LocReg && d33.Loc == scm.LocReg && d34.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d34)
			var d35 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d31.Imm.Int()) >> uint64(d34.Imm.Int())))}
			} else if d34.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r39, d31.Reg)
				ctx.W.EmitShrRegImm8(r39, uint8(d34.Imm.Int()))
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d35)
			} else {
				{
					shiftSrc := d31.Reg
					r40 := ctx.AllocRegExcept(d31.Reg)
					ctx.W.EmitMovRegReg(r40, d31.Reg)
					shiftSrc = r40
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d34.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d34.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d34.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d35)
				}
			}
			if d35.Loc == scm.LocReg && d31.Loc == scm.LocReg && d35.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.FreeDesc(&d34)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d12.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() | d35.Imm.Int())}
			} else if d12.Loc == scm.LocImm && d12.Imm.Int() == 0 {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d35.Reg}
				ctx.BindReg(d35.Reg, &d36)
			} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r41, d12.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d36)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d12.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d35.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r42, d12.Reg)
				if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r42, int32(d35.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d35.Imm.Int()))
					ctx.W.EmitOrInt64(r42, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d36)
			} else {
				r43 := ctx.AllocRegExcept(d12.Reg, d35.Reg)
				ctx.W.EmitMovRegReg(r43, d12.Reg)
				ctx.W.EmitOrInt64(r43, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d36)
			}
			if d36.Loc == scm.LocReg && d12.Loc == scm.LocReg && d36.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			d37 := d36
			if d37.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			ctx.EmitStoreToStack(d37, 0)
			ctx.W.EmitJmp(lbl11)
			ctx.W.MarkLabel(lbl8)
			d38 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d38)
			ctx.BindReg(r26, &d38)
			if r5 { ctx.UnprotectReg(r6) }
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d38)
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d38.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d39)
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d40)
			}
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() + d40.Imm.Int())}
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r46, d39.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d41)
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
				ctx.BindReg(d40.Reg, &d41)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(scratch, d39.Reg)
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d40.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else {
				r47 := ctx.AllocRegExcept(d39.Reg, d40.Reg)
				ctx.W.EmitMovRegReg(r47, d39.Reg)
				ctx.W.EmitAddInt64(r47, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d41)
			}
			if d41.Loc == scm.LocReg && d39.Loc == scm.LocReg && d41.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d41.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d41.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d42)
			}
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d43)
			}
			d44 := d43
			ctx.EnsureDesc(&d44)
			if d44.Loc != scm.LocImm && d44.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			if d44.Loc == scm.LocImm {
				if d44.Imm.Bool() {
					ctx.W.EmitJmp(lbl19)
				} else {
					ctx.W.EmitJmp(lbl20)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d44.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.W.MarkLabel(lbl19)
			ctx.W.EmitJmp(lbl17)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl18)
			ctx.FreeDesc(&d43)
			ctx.W.MarkLabel(lbl4)
			ctx.EnsureDesc(&d0)
			d45 := d0
			_ = d45
			r50 := d0.Loc == scm.LocReg
			r51 := d0.Reg
			if r50 { ctx.ProtectReg(r51) }
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl22)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d45)
			var d46 scm.JITValueDesc
			if d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d45.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d45.Reg)
				ctx.W.EmitShlRegImm8(r52, 32)
				ctx.W.EmitShrRegImm8(r52, 32)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d46)
			}
			var d47 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r53, thisptr.Reg, off)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d47)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d47)
			var d48 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d47.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, d47.Reg)
				ctx.W.EmitShlRegImm8(r54, 56)
				ctx.W.EmitShrRegImm8(r54, 56)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d48)
			}
			ctx.FreeDesc(&d47)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d46.Imm.Int() * d48.Imm.Int())}
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d46.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else if d48.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegReg(scratch, d46.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			} else {
				r55 := ctx.AllocRegExcept(d46.Reg, d48.Reg)
				ctx.W.EmitMovRegReg(r55, d46.Reg)
				ctx.W.EmitImulInt64(r55, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d49)
			}
			if d49.Loc == scm.LocReg && d46.Loc == scm.LocReg && d49.Reg == d46.Reg {
				ctx.TransferReg(d46.Reg)
				d46.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d48)
			var d50 scm.JITValueDesc
			r56 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r56, uint64(dataPtr))
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56, StackOff: int32(sliceLen)}
				ctx.BindReg(r56, &d50)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r56, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
				ctx.BindReg(r56, &d50)
			}
			ctx.BindReg(r56, &d50)
			ctx.EnsureDesc(&d49)
			var d51 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() / 64)}
			} else {
				r57 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r57, d49.Reg)
				ctx.W.EmitShrRegImm8(r57, 6)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d51)
			}
			if d51.Loc == scm.LocReg && d49.Loc == scm.LocReg && d51.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d51)
			r58 := ctx.AllocReg()
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d50)
			if d51.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r58, uint64(d51.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r58, d51.Reg)
				ctx.W.EmitShlRegImm8(r58, 3)
			}
			if d50.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.W.EmitAddInt64(r58, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r58, d50.Reg)
			}
			r59 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r59, r58, 0)
			ctx.FreeReg(r58)
			d52 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d52)
			ctx.FreeDesc(&d51)
			ctx.EnsureDesc(&d49)
			var d53 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() % 64)}
			} else {
				r60 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r60, d49.Reg)
				ctx.W.EmitAndRegImm32(r60, 63)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d53)
			}
			if d53.Loc == scm.LocReg && d49.Loc == scm.LocReg && d53.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d53)
			var d54 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d52.Imm.Int()) << uint64(d53.Imm.Int())))}
			} else if d53.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r61, d52.Reg)
				ctx.W.EmitShlRegImm8(r61, uint8(d53.Imm.Int()))
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d54)
			} else {
				{
					shiftSrc := d52.Reg
					r62 := ctx.AllocRegExcept(d52.Reg)
					ctx.W.EmitMovRegReg(r62, d52.Reg)
					shiftSrc = r62
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d53.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d53.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d53.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d54)
				}
			}
			if d54.Loc == scm.LocReg && d52.Loc == scm.LocReg && d54.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.FreeDesc(&d53)
			var d55 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r63, thisptr.Reg, off)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
				ctx.BindReg(r63, &d55)
			}
			d56 := d55
			ctx.EnsureDesc(&d56)
			if d56.Loc != scm.LocImm && d56.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d56.Loc == scm.LocImm {
				if d56.Imm.Bool() {
					ctx.W.EmitJmp(lbl25)
				} else {
					ctx.W.EmitJmp(lbl26)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d56.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl25)
				ctx.W.EmitJmp(lbl26)
			}
			ctx.W.MarkLabel(lbl25)
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl26)
			d57 := d54
			if d57.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 8)
			ctx.W.EmitJmp(lbl24)
			ctx.FreeDesc(&d55)
			ctx.W.MarkLabel(lbl24)
			d58 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d59 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d59)
			}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d59.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d59.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d60)
			}
			ctx.FreeDesc(&d59)
			d61 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d60)
			var d62 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d61.Imm.Int() - d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegReg(r66, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d62)
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
				r67 := ctx.AllocRegExcept(d61.Reg, d60.Reg)
				ctx.W.EmitMovRegReg(r67, d61.Reg)
				ctx.W.EmitSubInt64(r67, d60.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d62)
			}
			if d62.Loc == scm.LocReg && d61.Loc == scm.LocReg && d62.Reg == d61.Reg {
				ctx.TransferReg(d61.Reg)
				d61.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d58.Imm.Int()) >> uint64(d62.Imm.Int())))}
			} else if d62.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(r68, d58.Reg)
				ctx.W.EmitShrRegImm8(r68, uint8(d62.Imm.Int()))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d63)
			} else {
				{
					shiftSrc := d58.Reg
					r69 := ctx.AllocRegExcept(d58.Reg)
					ctx.W.EmitMovRegReg(r69, d58.Reg)
					shiftSrc = r69
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
			if d63.Loc == scm.LocReg && d58.Loc == scm.LocReg && d63.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.FreeDesc(&d62)
			r70 := ctx.AllocReg()
			ctx.EnsureDesc(&d63)
			ctx.EnsureDesc(&d63)
			if d63.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r70, d63)
			}
			ctx.W.EmitJmp(lbl21)
			ctx.W.MarkLabel(lbl23)
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d49)
			var d64 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r71, d49.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d64)
			}
			if d64.Loc == scm.LocReg && d49.Loc == scm.LocReg && d64.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			var d65 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d65)
			}
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d65.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d65.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d66)
			}
			ctx.FreeDesc(&d65)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d64)
			ctx.EnsureDesc(&d66)
			var d67 scm.JITValueDesc
			if d64.Loc == scm.LocImm && d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d64.Imm.Int() + d66.Imm.Int())}
			} else if d66.Loc == scm.LocImm && d66.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(r74, d64.Reg)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d67)
			} else if d64.Loc == scm.LocImm && d64.Imm.Int() == 0 {
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d66.Reg}
				ctx.BindReg(d66.Reg, &d67)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d64.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d66.Reg)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d67)
			} else if d66.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegReg(scratch, d64.Reg)
				if d66.Imm.Int() >= -2147483648 && d66.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d66.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d66.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d67)
			} else {
				r75 := ctx.AllocRegExcept(d64.Reg, d66.Reg)
				ctx.W.EmitMovRegReg(r75, d64.Reg)
				ctx.W.EmitAddInt64(r75, d66.Reg)
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d67)
			}
			if d67.Loc == scm.LocReg && d64.Loc == scm.LocReg && d67.Reg == d64.Reg {
				ctx.TransferReg(d64.Reg)
				d64.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d64)
			ctx.FreeDesc(&d66)
			ctx.EnsureDesc(&d67)
			var d68 scm.JITValueDesc
			if d67.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d67.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d67.Reg)
				ctx.W.EmitCmpRegImm32(d67.Reg, 64)
				ctx.W.EmitSetcc(r76, scm.CcA)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d68)
			}
			ctx.FreeDesc(&d67)
			d69 := d68
			ctx.EnsureDesc(&d69)
			if d69.Loc != scm.LocImm && d69.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			lbl29 := ctx.W.ReserveLabel()
			if d69.Loc == scm.LocImm {
				if d69.Imm.Bool() {
					ctx.W.EmitJmp(lbl28)
				} else {
					ctx.W.EmitJmp(lbl29)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d69.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
				ctx.W.EmitJmp(lbl29)
			}
			ctx.W.MarkLabel(lbl28)
			ctx.W.EmitJmp(lbl27)
			ctx.W.MarkLabel(lbl29)
			d70 := d54
			if d70.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d70)
			ctx.EmitStoreToStack(d70, 8)
			ctx.W.EmitJmp(lbl24)
			ctx.FreeDesc(&d68)
			ctx.W.MarkLabel(lbl27)
			d58 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d49)
			var d71 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r77, d49.Reg)
				ctx.W.EmitShrRegImm8(r77, 6)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d71)
			}
			if d71.Loc == scm.LocReg && d49.Loc == scm.LocReg && d71.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d71)
			ctx.EnsureDesc(&d71)
			var d72 scm.JITValueDesc
			if d71.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d71.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegReg(scratch, d71.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d72)
			}
			if d72.Loc == scm.LocReg && d71.Loc == scm.LocReg && d72.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d72)
			r78 := ctx.AllocReg()
			ctx.EnsureDesc(&d72)
			ctx.EnsureDesc(&d50)
			if d72.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r78, uint64(d72.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r78, d72.Reg)
				ctx.W.EmitShlRegImm8(r78, 3)
			}
			if d50.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.W.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r78, d50.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.W.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d73 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d73)
			ctx.FreeDesc(&d72)
			ctx.EnsureDesc(&d49)
			var d74 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r80, d49.Reg)
				ctx.W.EmitAndRegImm32(r80, 63)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d74)
			}
			if d74.Loc == scm.LocReg && d49.Loc == scm.LocReg && d74.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			d75 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d74)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d75.Imm.Int() - d74.Imm.Int())}
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(r81, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d76)
			} else if d75.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d75.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d74.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegReg(scratch, d75.Reg)
				if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d74.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else {
				r82 := ctx.AllocRegExcept(d75.Reg, d74.Reg)
				ctx.W.EmitMovRegReg(r82, d75.Reg)
				ctx.W.EmitSubInt64(r82, d74.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d76)
			}
			if d76.Loc == scm.LocReg && d75.Loc == scm.LocReg && d76.Reg == d75.Reg {
				ctx.TransferReg(d75.Reg)
				d75.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			ctx.EnsureDesc(&d73)
			ctx.EnsureDesc(&d76)
			var d77 scm.JITValueDesc
			if d73.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d73.Imm.Int()) >> uint64(d76.Imm.Int())))}
			} else if d76.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegReg(r83, d73.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d76.Imm.Int()))
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d77)
			} else {
				{
					shiftSrc := d73.Reg
					r84 := ctx.AllocRegExcept(d73.Reg)
					ctx.W.EmitMovRegReg(r84, d73.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d76.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d76.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d76.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d77)
				}
			}
			if d77.Loc == scm.LocReg && d73.Loc == scm.LocReg && d77.Reg == d73.Reg {
				ctx.TransferReg(d73.Reg)
				d73.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d73)
			ctx.FreeDesc(&d76)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d77)
			var d78 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() | d77.Imm.Int())}
			} else if d54.Loc == scm.LocImm && d54.Imm.Int() == 0 {
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
				ctx.BindReg(d77.Reg, &d78)
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				r85 := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegReg(r85, d54.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d78)
			} else if d54.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else if d77.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d54.Reg)
				ctx.W.EmitMovRegReg(r86, d54.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r86, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitOrInt64(r86, scm.RegR11)
				}
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d78)
			} else {
				r87 := ctx.AllocRegExcept(d54.Reg, d77.Reg)
				ctx.W.EmitMovRegReg(r87, d54.Reg)
				ctx.W.EmitOrInt64(r87, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d78)
			}
			if d78.Loc == scm.LocReg && d54.Loc == scm.LocReg && d78.Reg == d54.Reg {
				ctx.TransferReg(d54.Reg)
				d54.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d77)
			d79 := d78
			if d79.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d79)
			ctx.EmitStoreToStack(d79, 8)
			ctx.W.EmitJmp(lbl24)
			ctx.W.MarkLabel(lbl21)
			d80 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d80)
			ctx.BindReg(r70, &d80)
			if r50 { ctx.UnprotectReg(r51) }
			ctx.EnsureDesc(&d80)
			ctx.EnsureDesc(&d80)
			var d81 scm.JITValueDesc
			if d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d80.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d81)
			}
			ctx.FreeDesc(&d80)
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r89, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d82)
			}
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d82)
			ctx.EnsureDesc(&d81)
			ctx.EnsureDesc(&d82)
			var d83 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d81.Imm.Int() + d82.Imm.Int())}
			} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r90, d81.Reg)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d83)
			} else if d81.Loc == scm.LocImm && d81.Imm.Int() == 0 {
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d82.Reg}
				ctx.BindReg(d82.Reg, &d83)
			} else if d81.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d81.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d82.Reg)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d83)
			} else if d82.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(scratch, d81.Reg)
				if d82.Imm.Int() >= -2147483648 && d82.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d82.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d82.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d83)
			} else {
				r91 := ctx.AllocRegExcept(d81.Reg, d82.Reg)
				ctx.W.EmitMovRegReg(r91, d81.Reg)
				ctx.W.EmitAddInt64(r91, d82.Reg)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d83)
			}
			if d83.Loc == scm.LocReg && d81.Loc == scm.LocReg && d83.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d82)
			ctx.EnsureDesc(&d83)
			ctx.EnsureDesc(&d83)
			var d84 scm.JITValueDesc
			if d83.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d83.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d84)
			}
			ctx.FreeDesc(&d83)
			var d85 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d85)
			}
			d86 := d85
			ctx.EnsureDesc(&d86)
			if d86.Loc != scm.LocImm && d86.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			lbl33 := ctx.W.ReserveLabel()
			if d86.Loc == scm.LocImm {
				if d86.Imm.Bool() {
					ctx.W.EmitJmp(lbl32)
				} else {
					ctx.W.EmitJmp(lbl33)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d86.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.W.MarkLabel(lbl32)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl33)
			ctx.W.EmitJmp(lbl31)
			ctx.FreeDesc(&d85)
			ctx.W.MarkLabel(lbl18)
			ctx.EnsureDesc(&d42)
			d87 := d42
			_ = d87
			r94 := d42.Loc == scm.LocReg
			r95 := d42.Reg
			if r94 { ctx.ProtectReg(r95) }
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl35)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			var d88 scm.JITValueDesc
			if d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d87.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d87.Reg)
				ctx.W.EmitShlRegImm8(r96, 32)
				ctx.W.EmitShrRegImm8(r96, 32)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d88)
			}
			var d89 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r97, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d89)
			}
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d89)
			var d90 scm.JITValueDesc
			if d89.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d89.Imm.Int()))))}
			} else {
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r98, d89.Reg)
				ctx.W.EmitShlRegImm8(r98, 56)
				ctx.W.EmitShrRegImm8(r98, 56)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d90)
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
				r99 := ctx.AllocRegExcept(d88.Reg, d90.Reg)
				ctx.W.EmitMovRegReg(r99, d88.Reg)
				ctx.W.EmitImulInt64(r99, d90.Reg)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d91)
			}
			if d91.Loc == scm.LocReg && d88.Loc == scm.LocReg && d91.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			ctx.FreeDesc(&d90)
			var d92 scm.JITValueDesc
			r100 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r100, uint64(dataPtr))
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100, StackOff: int32(sliceLen)}
				ctx.BindReg(r100, &d92)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r100, thisptr.Reg, off)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d92)
			}
			ctx.BindReg(r100, &d92)
			ctx.EnsureDesc(&d91)
			var d93 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() / 64)}
			} else {
				r101 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r101, d91.Reg)
				ctx.W.EmitShrRegImm8(r101, 6)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d93)
			}
			if d93.Loc == scm.LocReg && d91.Loc == scm.LocReg && d93.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d93)
			r102 := ctx.AllocReg()
			ctx.EnsureDesc(&d93)
			ctx.EnsureDesc(&d92)
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r102, uint64(d93.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r102, d93.Reg)
				ctx.W.EmitShlRegImm8(r102, 3)
			}
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
				ctx.W.EmitAddInt64(r102, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r102, d92.Reg)
			}
			r103 := ctx.AllocRegExcept(r102)
			ctx.W.EmitMovRegMem(r103, r102, 0)
			ctx.FreeReg(r102)
			d94 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d94)
			ctx.FreeDesc(&d93)
			ctx.EnsureDesc(&d91)
			var d95 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r104 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r104, d91.Reg)
				ctx.W.EmitAndRegImm32(r104, 63)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d95)
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
				r105 := ctx.AllocRegExcept(d94.Reg)
				ctx.W.EmitMovRegReg(r105, d94.Reg)
				ctx.W.EmitShlRegImm8(r105, uint8(d95.Imm.Int()))
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d96)
			} else {
				{
					shiftSrc := d94.Reg
					r106 := ctx.AllocRegExcept(d94.Reg)
					ctx.W.EmitMovRegReg(r106, d94.Reg)
					shiftSrc = r106
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d97)
			}
			d98 := d97
			ctx.EnsureDesc(&d98)
			if d98.Loc != scm.LocImm && d98.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl36 := ctx.W.ReserveLabel()
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			if d98.Loc == scm.LocImm {
				if d98.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d98.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl38)
				ctx.W.EmitJmp(lbl39)
			}
			ctx.W.MarkLabel(lbl38)
			ctx.W.EmitJmp(lbl36)
			ctx.W.MarkLabel(lbl39)
			d99 := d96
			if d99.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d99)
			ctx.EmitStoreToStack(d99, 16)
			ctx.W.EmitJmp(lbl37)
			ctx.FreeDesc(&d97)
			ctx.W.MarkLabel(lbl37)
			d100 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d101 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r108, thisptr.Reg, off)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
				ctx.BindReg(r108, &d101)
			}
			ctx.EnsureDesc(&d101)
			ctx.EnsureDesc(&d101)
			var d102 scm.JITValueDesc
			if d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d101.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r109, d101.Reg)
				ctx.W.EmitShlRegImm8(r109, 56)
				ctx.W.EmitShrRegImm8(r109, 56)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d102)
			}
			ctx.FreeDesc(&d101)
			d103 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d102)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d102.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() - d102.Imm.Int())}
			} else if d102.Loc == scm.LocImm && d102.Imm.Int() == 0 {
				r110 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r110, d103.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d104)
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d102.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d104)
			} else if d102.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(scratch, d103.Reg)
				if d102.Imm.Int() >= -2147483648 && d102.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d102.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d102.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d104)
			} else {
				r111 := ctx.AllocRegExcept(d103.Reg, d102.Reg)
				ctx.W.EmitMovRegReg(r111, d103.Reg)
				ctx.W.EmitSubInt64(r111, d102.Reg)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d104)
			}
			if d104.Loc == scm.LocReg && d103.Loc == scm.LocReg && d104.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d102)
			ctx.EnsureDesc(&d100)
			ctx.EnsureDesc(&d104)
			var d105 scm.JITValueDesc
			if d100.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d100.Imm.Int()) >> uint64(d104.Imm.Int())))}
			} else if d104.Loc == scm.LocImm {
				r112 := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(r112, d100.Reg)
				ctx.W.EmitShrRegImm8(r112, uint8(d104.Imm.Int()))
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d105)
			} else {
				{
					shiftSrc := d100.Reg
					r113 := ctx.AllocRegExcept(d100.Reg)
					ctx.W.EmitMovRegReg(r113, d100.Reg)
					shiftSrc = r113
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d104.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d104.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d104.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d105)
				}
			}
			if d105.Loc == scm.LocReg && d100.Loc == scm.LocReg && d105.Reg == d100.Reg {
				ctx.TransferReg(d100.Reg)
				d100.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d100)
			ctx.FreeDesc(&d104)
			r114 := ctx.AllocReg()
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d105)
			if d105.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r114, d105)
			}
			ctx.W.EmitJmp(lbl34)
			ctx.W.MarkLabel(lbl36)
			d100 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d91)
			var d106 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r115 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r115, d91.Reg)
				ctx.W.EmitAndRegImm32(r115, 63)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d106)
			}
			if d106.Loc == scm.LocReg && d91.Loc == scm.LocReg && d106.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			var d107 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r116, thisptr.Reg, off)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
				ctx.BindReg(r116, &d107)
			}
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			var d108 scm.JITValueDesc
			if d107.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d107.Imm.Int()))))}
			} else {
				r117 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r117, d107.Reg)
				ctx.W.EmitShlRegImm8(r117, 56)
				ctx.W.EmitShrRegImm8(r117, 56)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d108)
			}
			ctx.FreeDesc(&d107)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d106)
			ctx.EnsureDesc(&d108)
			var d109 scm.JITValueDesc
			if d106.Loc == scm.LocImm && d108.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() + d108.Imm.Int())}
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				r118 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(r118, d106.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d109)
			} else if d106.Loc == scm.LocImm && d106.Imm.Int() == 0 {
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d108.Reg}
				ctx.BindReg(d108.Reg, &d109)
			} else if d106.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d106.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(scratch, d106.Reg)
				if d108.Imm.Int() >= -2147483648 && d108.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d108.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d108.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d109)
			} else {
				r119 := ctx.AllocRegExcept(d106.Reg, d108.Reg)
				ctx.W.EmitMovRegReg(r119, d106.Reg)
				ctx.W.EmitAddInt64(r119, d108.Reg)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d109)
			}
			if d109.Loc == scm.LocReg && d106.Loc == scm.LocReg && d109.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			ctx.FreeDesc(&d108)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d109.Imm.Int()) > uint64(64))}
			} else {
				r120 := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitCmpRegImm32(d109.Reg, 64)
				ctx.W.EmitSetcc(r120, scm.CcA)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r120}
				ctx.BindReg(r120, &d110)
			}
			ctx.FreeDesc(&d109)
			d111 := d110
			ctx.EnsureDesc(&d111)
			if d111.Loc != scm.LocImm && d111.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl40 := ctx.W.ReserveLabel()
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d111.Loc == scm.LocImm {
				if d111.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.EmitJmp(lbl42)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d111.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl41)
				ctx.W.EmitJmp(lbl42)
			}
			ctx.W.MarkLabel(lbl41)
			ctx.W.EmitJmp(lbl40)
			ctx.W.MarkLabel(lbl42)
			d112 := d96
			if d112.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d112)
			ctx.EmitStoreToStack(d112, 16)
			ctx.W.EmitJmp(lbl37)
			ctx.FreeDesc(&d110)
			ctx.W.MarkLabel(lbl40)
			d100 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d91)
			var d113 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() / 64)}
			} else {
				r121 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r121, d91.Reg)
				ctx.W.EmitShrRegImm8(r121, 6)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d113)
			}
			if d113.Loc == scm.LocReg && d91.Loc == scm.LocReg && d113.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d113)
			ctx.EnsureDesc(&d113)
			var d114 scm.JITValueDesc
			if d113.Loc == scm.LocImm {
				d114 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d113.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegReg(scratch, d113.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d114 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d114)
			}
			if d114.Loc == scm.LocReg && d113.Loc == scm.LocReg && d114.Reg == d113.Reg {
				ctx.TransferReg(d113.Reg)
				d113.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			ctx.EnsureDesc(&d114)
			r122 := ctx.AllocReg()
			ctx.EnsureDesc(&d114)
			ctx.EnsureDesc(&d92)
			if d114.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r122, uint64(d114.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r122, d114.Reg)
				ctx.W.EmitShlRegImm8(r122, 3)
			}
			if d92.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d92.Imm.Int()))
				ctx.W.EmitAddInt64(r122, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r122, d92.Reg)
			}
			r123 := ctx.AllocRegExcept(r122)
			ctx.W.EmitMovRegMem(r123, r122, 0)
			ctx.FreeReg(r122)
			d115 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
			ctx.BindReg(r123, &d115)
			ctx.FreeDesc(&d114)
			ctx.EnsureDesc(&d91)
			var d116 scm.JITValueDesc
			if d91.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d91.Imm.Int() % 64)}
			} else {
				r124 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r124, d91.Reg)
				ctx.W.EmitAndRegImm32(r124, 63)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d116)
			}
			if d116.Loc == scm.LocReg && d91.Loc == scm.LocReg && d116.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			d117 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d116)
			var d118 scm.JITValueDesc
			if d117.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d117.Imm.Int() - d116.Imm.Int())}
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				r125 := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegReg(r125, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d118)
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
				r126 := ctx.AllocRegExcept(d117.Reg, d116.Reg)
				ctx.W.EmitMovRegReg(r126, d117.Reg)
				ctx.W.EmitSubInt64(r126, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d118)
			}
			if d118.Loc == scm.LocReg && d117.Loc == scm.LocReg && d118.Reg == d117.Reg {
				ctx.TransferReg(d117.Reg)
				d117.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.EnsureDesc(&d115)
			ctx.EnsureDesc(&d118)
			var d119 scm.JITValueDesc
			if d115.Loc == scm.LocImm && d118.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d115.Imm.Int()) >> uint64(d118.Imm.Int())))}
			} else if d118.Loc == scm.LocImm {
				r127 := ctx.AllocRegExcept(d115.Reg)
				ctx.W.EmitMovRegReg(r127, d115.Reg)
				ctx.W.EmitShrRegImm8(r127, uint8(d118.Imm.Int()))
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d119)
			} else {
				{
					shiftSrc := d115.Reg
					r128 := ctx.AllocRegExcept(d115.Reg)
					ctx.W.EmitMovRegReg(r128, d115.Reg)
					shiftSrc = r128
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
			if d119.Loc == scm.LocReg && d115.Loc == scm.LocReg && d119.Reg == d115.Reg {
				ctx.TransferReg(d115.Reg)
				d115.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d115)
			ctx.FreeDesc(&d118)
			ctx.EnsureDesc(&d96)
			ctx.EnsureDesc(&d119)
			var d120 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() | d119.Imm.Int())}
			} else if d96.Loc == scm.LocImm && d96.Imm.Int() == 0 {
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d119.Reg}
				ctx.BindReg(d119.Reg, &d120)
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				r129 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r129, d96.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d120)
			} else if d96.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d96.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d119.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d120)
			} else if d119.Loc == scm.LocImm {
				r130 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r130, d96.Reg)
				if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r130, int32(d119.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
					ctx.W.EmitOrInt64(r130, scm.RegR11)
				}
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d120)
			} else {
				r131 := ctx.AllocRegExcept(d96.Reg, d119.Reg)
				ctx.W.EmitMovRegReg(r131, d96.Reg)
				ctx.W.EmitOrInt64(r131, d119.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d120)
			}
			if d120.Loc == scm.LocReg && d96.Loc == scm.LocReg && d120.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			d121 := d120
			if d121.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d121)
			ctx.EmitStoreToStack(d121, 16)
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl34)
			d122 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
			ctx.BindReg(r114, &d122)
			ctx.BindReg(r114, &d122)
			if r94 { ctx.UnprotectReg(r95) }
			ctx.EnsureDesc(&d122)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d122.Imm.Int()))))}
			} else {
				r132 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r132, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d123)
			}
			ctx.FreeDesc(&d122)
			var d124 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r133, thisptr.Reg, off)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
				ctx.BindReg(r133, &d124)
			}
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d124)
			ctx.EnsureDesc(&d123)
			ctx.EnsureDesc(&d124)
			var d125 scm.JITValueDesc
			if d123.Loc == scm.LocImm && d124.Loc == scm.LocImm {
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d123.Imm.Int() + d124.Imm.Int())}
			} else if d124.Loc == scm.LocImm && d124.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(r134, d123.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d125)
			} else if d123.Loc == scm.LocImm && d123.Imm.Int() == 0 {
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d124.Reg}
				ctx.BindReg(d124.Reg, &d125)
			} else if d123.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d123.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d123.Reg)
				ctx.W.EmitMovRegReg(scratch, d123.Reg)
				if d124.Imm.Int() >= -2147483648 && d124.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d124.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d124.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d125)
			} else {
				r135 := ctx.AllocRegExcept(d123.Reg, d124.Reg)
				ctx.W.EmitMovRegReg(r135, d123.Reg)
				ctx.W.EmitAddInt64(r135, d124.Reg)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d125)
			}
			if d125.Loc == scm.LocReg && d123.Loc == scm.LocReg && d125.Reg == d123.Reg {
				ctx.TransferReg(d123.Reg)
				d123.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d123)
			ctx.FreeDesc(&d124)
			ctx.EnsureDesc(&d42)
			d126 := d42
			_ = d126
			r136 := d42.Loc == scm.LocReg
			r137 := d42.Reg
			if r136 { ctx.ProtectReg(r137) }
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl44)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d126)
			var d127 scm.JITValueDesc
			if d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d126.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d126.Reg)
				ctx.W.EmitShlRegImm8(r138, 32)
				ctx.W.EmitShrRegImm8(r138, 32)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d127)
			}
			var d128 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r139, thisptr.Reg, off)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r139}
				ctx.BindReg(r139, &d128)
			}
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d128)
			var d129 scm.JITValueDesc
			if d128.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d128.Imm.Int()))))}
			} else {
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r140, d128.Reg)
				ctx.W.EmitShlRegImm8(r140, 56)
				ctx.W.EmitShrRegImm8(r140, 56)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d129)
			}
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d129)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d129)
			var d130 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d129.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() * d129.Imm.Int())}
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d129.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d129.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(scratch, d127.Reg)
				if d129.Imm.Int() >= -2147483648 && d129.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d129.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d129.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r141 := ctx.AllocRegExcept(d127.Reg, d129.Reg)
				ctx.W.EmitMovRegReg(r141, d127.Reg)
				ctx.W.EmitImulInt64(r141, d129.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d130)
			}
			if d130.Loc == scm.LocReg && d127.Loc == scm.LocReg && d130.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			ctx.FreeDesc(&d129)
			var d131 scm.JITValueDesc
			r142 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r142, uint64(dataPtr))
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142, StackOff: int32(sliceLen)}
				ctx.BindReg(r142, &d131)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r142, thisptr.Reg, off)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142}
				ctx.BindReg(r142, &d131)
			}
			ctx.BindReg(r142, &d131)
			ctx.EnsureDesc(&d130)
			var d132 scm.JITValueDesc
			if d130.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() / 64)}
			} else {
				r143 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r143, d130.Reg)
				ctx.W.EmitShrRegImm8(r143, 6)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d132)
			}
			if d132.Loc == scm.LocReg && d130.Loc == scm.LocReg && d132.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d132)
			r144 := ctx.AllocReg()
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d131)
			if d132.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r144, uint64(d132.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r144, d132.Reg)
				ctx.W.EmitShlRegImm8(r144, 3)
			}
			if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
				ctx.W.EmitAddInt64(r144, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r144, d131.Reg)
			}
			r145 := ctx.AllocRegExcept(r144)
			ctx.W.EmitMovRegMem(r145, r144, 0)
			ctx.FreeReg(r144)
			d133 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			ctx.BindReg(r145, &d133)
			ctx.FreeDesc(&d132)
			ctx.EnsureDesc(&d130)
			var d134 scm.JITValueDesc
			if d130.Loc == scm.LocImm {
				d134 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() % 64)}
			} else {
				r146 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r146, d130.Reg)
				ctx.W.EmitAndRegImm32(r146, 63)
				d134 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d134)
			}
			if d134.Loc == scm.LocReg && d130.Loc == scm.LocReg && d134.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d133)
			ctx.EnsureDesc(&d134)
			var d135 scm.JITValueDesc
			if d133.Loc == scm.LocImm && d134.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d133.Imm.Int()) << uint64(d134.Imm.Int())))}
			} else if d134.Loc == scm.LocImm {
				r147 := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegReg(r147, d133.Reg)
				ctx.W.EmitShlRegImm8(r147, uint8(d134.Imm.Int()))
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d135)
			} else {
				{
					shiftSrc := d133.Reg
					r148 := ctx.AllocRegExcept(d133.Reg)
					ctx.W.EmitMovRegReg(r148, d133.Reg)
					shiftSrc = r148
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d134.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d134.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d134.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d135)
				}
			}
			if d135.Loc == scm.LocReg && d133.Loc == scm.LocReg && d135.Reg == d133.Reg {
				ctx.TransferReg(d133.Reg)
				d133.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d133)
			ctx.FreeDesc(&d134)
			var d136 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r149, thisptr.Reg, off)
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
				ctx.BindReg(r149, &d136)
			}
			d137 := d136
			ctx.EnsureDesc(&d137)
			if d137.Loc != scm.LocImm && d137.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			lbl48 := ctx.W.ReserveLabel()
			if d137.Loc == scm.LocImm {
				if d137.Imm.Bool() {
					ctx.W.EmitJmp(lbl47)
				} else {
					ctx.W.EmitJmp(lbl48)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d137.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.W.MarkLabel(lbl47)
			ctx.W.EmitJmp(lbl45)
			ctx.W.MarkLabel(lbl48)
			d138 := d135
			if d138.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d138)
			ctx.EmitStoreToStack(d138, 24)
			ctx.W.EmitJmp(lbl46)
			ctx.FreeDesc(&d136)
			ctx.W.MarkLabel(lbl46)
			d139 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d140 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d140)
			}
			ctx.EnsureDesc(&d140)
			ctx.EnsureDesc(&d140)
			var d141 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d140.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d140.Reg)
				ctx.W.EmitShlRegImm8(r151, 56)
				ctx.W.EmitShrRegImm8(r151, 56)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d141)
			}
			ctx.FreeDesc(&d140)
			d142 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d141)
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d141)
			var d143 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d141.Loc == scm.LocImm {
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() - d141.Imm.Int())}
			} else if d141.Loc == scm.LocImm && d141.Imm.Int() == 0 {
				r152 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r152, d142.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d143)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d141.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d141.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d143)
			} else if d141.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(scratch, d142.Reg)
				if d141.Imm.Int() >= -2147483648 && d141.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d141.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d143)
			} else {
				r153 := ctx.AllocRegExcept(d142.Reg, d141.Reg)
				ctx.W.EmitMovRegReg(r153, d142.Reg)
				ctx.W.EmitSubInt64(r153, d141.Reg)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d143)
			}
			if d143.Loc == scm.LocReg && d142.Loc == scm.LocReg && d143.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d141)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d143)
			var d144 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d139.Imm.Int()) >> uint64(d143.Imm.Int())))}
			} else if d143.Loc == scm.LocImm {
				r154 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r154, d139.Reg)
				ctx.W.EmitShrRegImm8(r154, uint8(d143.Imm.Int()))
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d144)
			} else {
				{
					shiftSrc := d139.Reg
					r155 := ctx.AllocRegExcept(d139.Reg)
					ctx.W.EmitMovRegReg(r155, d139.Reg)
					shiftSrc = r155
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d143.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d143.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d143.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d144)
				}
			}
			if d144.Loc == scm.LocReg && d139.Loc == scm.LocReg && d144.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d139)
			ctx.FreeDesc(&d143)
			r156 := ctx.AllocReg()
			ctx.EnsureDesc(&d144)
			ctx.EnsureDesc(&d144)
			if d144.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r156, d144)
			}
			ctx.W.EmitJmp(lbl43)
			ctx.W.MarkLabel(lbl45)
			d139 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d130)
			var d145 scm.JITValueDesc
			if d130.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() % 64)}
			} else {
				r157 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r157, d130.Reg)
				ctx.W.EmitAndRegImm32(r157, 63)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d145)
			}
			if d145.Loc == scm.LocReg && d130.Loc == scm.LocReg && d145.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d146)
			}
			ctx.EnsureDesc(&d146)
			ctx.EnsureDesc(&d146)
			var d147 scm.JITValueDesc
			if d146.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d146.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d146.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d147)
			}
			ctx.FreeDesc(&d146)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d147)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d147)
			var d148 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d148 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() + d147.Imm.Int())}
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegReg(r160, d145.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d148)
			} else if d145.Loc == scm.LocImm && d145.Imm.Int() == 0 {
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d147.Reg}
				ctx.BindReg(d147.Reg, &d148)
			} else if d145.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d148)
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegReg(scratch, d145.Reg)
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d147.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d148)
			} else {
				r161 := ctx.AllocRegExcept(d145.Reg, d147.Reg)
				ctx.W.EmitMovRegReg(r161, d145.Reg)
				ctx.W.EmitAddInt64(r161, d147.Reg)
				d148 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d148)
			}
			if d148.Loc == scm.LocReg && d145.Loc == scm.LocReg && d148.Reg == d145.Reg {
				ctx.TransferReg(d145.Reg)
				d145.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d145)
			ctx.FreeDesc(&d147)
			ctx.EnsureDesc(&d148)
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d148.Imm.Int()) > uint64(64))}
			} else {
				r162 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitCmpRegImm32(d148.Reg, 64)
				ctx.W.EmitSetcc(r162, scm.CcA)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
				ctx.BindReg(r162, &d149)
			}
			ctx.FreeDesc(&d148)
			d150 := d149
			ctx.EnsureDesc(&d150)
			if d150.Loc != scm.LocImm && d150.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			if d150.Loc == scm.LocImm {
				if d150.Imm.Bool() {
					ctx.W.EmitJmp(lbl50)
				} else {
					ctx.W.EmitJmp(lbl51)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d150.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl51)
			}
			ctx.W.MarkLabel(lbl50)
			ctx.W.EmitJmp(lbl49)
			ctx.W.MarkLabel(lbl51)
			d151 := d135
			if d151.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d151)
			ctx.EmitStoreToStack(d151, 24)
			ctx.W.EmitJmp(lbl46)
			ctx.FreeDesc(&d149)
			ctx.W.MarkLabel(lbl49)
			d139 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d130)
			var d152 scm.JITValueDesc
			if d130.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() / 64)}
			} else {
				r163 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r163, d130.Reg)
				ctx.W.EmitShrRegImm8(r163, 6)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d152)
			}
			if d152.Loc == scm.LocReg && d130.Loc == scm.LocReg && d152.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d152)
			var d153 scm.JITValueDesc
			if d152.Loc == scm.LocImm {
				d153 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d152.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegReg(scratch, d152.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d153 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d153)
			}
			if d153.Loc == scm.LocReg && d152.Loc == scm.LocReg && d153.Reg == d152.Reg {
				ctx.TransferReg(d152.Reg)
				d152.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d153)
			r164 := ctx.AllocReg()
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d131)
			if d153.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r164, uint64(d153.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r164, d153.Reg)
				ctx.W.EmitShlRegImm8(r164, 3)
			}
			if d131.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d131.Imm.Int()))
				ctx.W.EmitAddInt64(r164, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r164, d131.Reg)
			}
			r165 := ctx.AllocRegExcept(r164)
			ctx.W.EmitMovRegMem(r165, r164, 0)
			ctx.FreeReg(r164)
			d154 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
			ctx.BindReg(r165, &d154)
			ctx.FreeDesc(&d153)
			ctx.EnsureDesc(&d130)
			var d155 scm.JITValueDesc
			if d130.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d130.Imm.Int() % 64)}
			} else {
				r166 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r166, d130.Reg)
				ctx.W.EmitAndRegImm32(r166, 63)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d155)
			}
			if d155.Loc == scm.LocReg && d130.Loc == scm.LocReg && d155.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d130)
			d156 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d155)
			var d157 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() - d155.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r167 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r167, d156.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d157)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d155.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d157)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(scratch, d156.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d157)
			} else {
				r168 := ctx.AllocRegExcept(d156.Reg, d155.Reg)
				ctx.W.EmitMovRegReg(r168, d156.Reg)
				ctx.W.EmitSubInt64(r168, d155.Reg)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d157)
			}
			if d157.Loc == scm.LocReg && d156.Loc == scm.LocReg && d157.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			ctx.EnsureDesc(&d154)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d154.Imm.Int()) >> uint64(d157.Imm.Int())))}
			} else if d157.Loc == scm.LocImm {
				r169 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r169, d154.Reg)
				ctx.W.EmitShrRegImm8(r169, uint8(d157.Imm.Int()))
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d158)
			} else {
				{
					shiftSrc := d154.Reg
					r170 := ctx.AllocRegExcept(d154.Reg)
					ctx.W.EmitMovRegReg(r170, d154.Reg)
					shiftSrc = r170
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d157.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d157.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d157.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d158)
				}
			}
			if d158.Loc == scm.LocReg && d154.Loc == scm.LocReg && d158.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d135)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d135.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d135.Imm.Int() | d158.Imm.Int())}
			} else if d135.Loc == scm.LocImm && d135.Imm.Int() == 0 {
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
				ctx.BindReg(d158.Reg, &d159)
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r171 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r171, d135.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d159)
			} else if d135.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d135.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d158.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d135.Reg)
				ctx.W.EmitMovRegReg(r172, d135.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r172, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitOrInt64(r172, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d159)
			} else {
				r173 := ctx.AllocRegExcept(d135.Reg, d158.Reg)
				ctx.W.EmitMovRegReg(r173, d135.Reg)
				ctx.W.EmitOrInt64(r173, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d159)
			}
			if d159.Loc == scm.LocReg && d135.Loc == scm.LocReg && d159.Reg == d135.Reg {
				ctx.TransferReg(d135.Reg)
				d135.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d158)
			d160 := d159
			if d160.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d160)
			ctx.EmitStoreToStack(d160, 24)
			ctx.W.EmitJmp(lbl46)
			ctx.W.MarkLabel(lbl43)
			d161 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
			ctx.BindReg(r156, &d161)
			ctx.BindReg(r156, &d161)
			if r136 { ctx.UnprotectReg(r137) }
			ctx.EnsureDesc(&d161)
			ctx.EnsureDesc(&d161)
			var d162 scm.JITValueDesc
			if d161.Loc == scm.LocImm {
				d162 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d161.Imm.Int()))))}
			} else {
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r174, d161.Reg)
				d162 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d162)
			}
			ctx.FreeDesc(&d161)
			var d163 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r175, thisptr.Reg, off)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r175}
				ctx.BindReg(r175, &d163)
			}
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d163)
			ctx.EnsureDesc(&d162)
			ctx.EnsureDesc(&d163)
			var d164 scm.JITValueDesc
			if d162.Loc == scm.LocImm && d163.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d162.Imm.Int() + d163.Imm.Int())}
			} else if d163.Loc == scm.LocImm && d163.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(r176, d162.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d164)
			} else if d162.Loc == scm.LocImm && d162.Imm.Int() == 0 {
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d163.Reg}
				ctx.BindReg(d163.Reg, &d164)
			} else if d162.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d163.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d162.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d163.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d164)
			} else if d163.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d162.Reg)
				ctx.W.EmitMovRegReg(scratch, d162.Reg)
				if d163.Imm.Int() >= -2147483648 && d163.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d163.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d163.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d164)
			} else {
				r177 := ctx.AllocRegExcept(d162.Reg, d163.Reg)
				ctx.W.EmitMovRegReg(r177, d162.Reg)
				ctx.W.EmitAddInt64(r177, d163.Reg)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d164)
			}
			if d164.Loc == scm.LocReg && d162.Loc == scm.LocReg && d164.Reg == d162.Reg {
				ctx.TransferReg(d162.Reg)
				d162.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d162)
			ctx.FreeDesc(&d163)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d164)
			var d166 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d164.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d125.Imm.Int() + d164.Imm.Int())}
			} else if d164.Loc == scm.LocImm && d164.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r178, d125.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d166)
			} else if d125.Loc == scm.LocImm && d125.Imm.Int() == 0 {
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d164.Reg}
				ctx.BindReg(d164.Reg, &d166)
			} else if d125.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d125.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else if d164.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(scratch, d125.Reg)
				if d164.Imm.Int() >= -2147483648 && d164.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d164.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d164.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d166)
			} else {
				r179 := ctx.AllocRegExcept(d125.Reg, d164.Reg)
				ctx.W.EmitMovRegReg(r179, d125.Reg)
				ctx.W.EmitAddInt64(r179, d164.Reg)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d166)
			}
			if d166.Loc == scm.LocReg && d125.Loc == scm.LocReg && d166.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d166)
			var d168 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r180, fieldAddr)
				ctx.W.EmitMovRegMem64(r181, fieldAddr+8)
				d168 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d168)
				ctx.BindReg(r181, &d168)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r182 := ctx.AllocReg()
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r182, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off+8)
				d168 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
				ctx.BindReg(r182, &d168)
				ctx.BindReg(r183, &d168)
			}
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d166)
			r184 := ctx.AllocReg()
			r185 := ctx.AllocRegExcept(r184)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d166)
			if d168.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r184, uint64(d168.Imm.Int()))
			} else if d168.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r184, d168.Reg)
			} else {
				ctx.W.EmitMovRegReg(r184, d168.Reg)
			}
			if d125.Loc == scm.LocImm {
				if d125.Imm.Int() != 0 {
					if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r184, int32(d125.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d125.Imm.Int()))
						ctx.W.EmitAddInt64(r184, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r184, d125.Reg)
			}
			if d166.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r185, uint64(d166.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r185, d166.Reg)
			}
			if d125.Loc == scm.LocImm {
				if d125.Imm.Int() >= -2147483648 && d125.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r185, int32(d125.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d125.Imm.Int()))
					ctx.W.EmitSubInt64(r185, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r185, d125.Reg)
			}
			d169 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d169)
			ctx.BindReg(r185, &d169)
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d166)
			r186 := ctx.AllocReg()
			r187 := ctx.AllocRegExcept(r186)
			d170 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d170)
			ctx.BindReg(r187, &d170)
			d171 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d169}, 2)
			ctx.EmitMovPairToResult(&d171, &d170)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl17)
			var d172 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r188, thisptr.Reg, off)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d172)
			}
			ctx.EnsureDesc(&d172)
			ctx.EnsureDesc(&d172)
			var d173 scm.JITValueDesc
			if d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d172.Imm.Int()))))}
			} else {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r189, d172.Reg)
				ctx.W.EmitShlRegImm8(r189, 32)
				ctx.W.EmitShrRegImm8(r189, 32)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d173)
			}
			ctx.FreeDesc(&d172)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d173)
			var d174 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d42.Imm.Int()) == uint64(d173.Imm.Int()))}
			} else if d173.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d42.Reg)
				if d173.Imm.Int() >= -2147483648 && d173.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d42.Reg, int32(d173.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d173.Imm.Int()))
					ctx.W.EmitCmpInt64(d42.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r190, scm.CcE)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r190}
				ctx.BindReg(r190, &d174)
			} else if d42.Loc == scm.LocImm {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d173.Reg)
				ctx.W.EmitSetcc(r191, scm.CcE)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r191}
				ctx.BindReg(r191, &d174)
			} else {
				r192 := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitCmpInt64(d42.Reg, d173.Reg)
				ctx.W.EmitSetcc(r192, scm.CcE)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
				ctx.BindReg(r192, &d174)
			}
			ctx.FreeDesc(&d42)
			ctx.FreeDesc(&d173)
			d175 := d174
			ctx.EnsureDesc(&d175)
			if d175.Loc != scm.LocImm && d175.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d175.Loc == scm.LocImm {
				if d175.Imm.Bool() {
					ctx.W.EmitJmp(lbl53)
				} else {
					ctx.W.EmitJmp(lbl54)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d175.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl53)
				ctx.W.EmitJmp(lbl54)
			}
			ctx.W.MarkLabel(lbl53)
			ctx.W.EmitJmp(lbl52)
			ctx.W.MarkLabel(lbl54)
			ctx.W.EmitJmp(lbl18)
			ctx.FreeDesc(&d174)
			ctx.W.MarkLabel(lbl31)
			ctx.EnsureDesc(&d0)
			d176 := d0
			_ = d176
			r193 := d0.Loc == scm.LocReg
			r194 := d0.Reg
			if r193 { ctx.ProtectReg(r194) }
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl56)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d176.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r195, d176.Reg)
				ctx.W.EmitShlRegImm8(r195, 32)
				ctx.W.EmitShrRegImm8(r195, 32)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d177)
			}
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r196 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r196, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r196}
				ctx.BindReg(r196, &d178)
			}
			ctx.EnsureDesc(&d178)
			ctx.EnsureDesc(&d178)
			var d179 scm.JITValueDesc
			if d178.Loc == scm.LocImm {
				d179 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d178.Imm.Int()))))}
			} else {
				r197 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r197, d178.Reg)
				ctx.W.EmitShlRegImm8(r197, 56)
				ctx.W.EmitShrRegImm8(r197, 56)
				d179 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d179)
			}
			ctx.FreeDesc(&d178)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d179)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d179)
			var d180 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d179.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() * d179.Imm.Int())}
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d180)
			} else if d179.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(scratch, d177.Reg)
				if d179.Imm.Int() >= -2147483648 && d179.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d179.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d179.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d180)
			} else {
				r198 := ctx.AllocRegExcept(d177.Reg, d179.Reg)
				ctx.W.EmitMovRegReg(r198, d177.Reg)
				ctx.W.EmitImulInt64(r198, d179.Reg)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d180)
			}
			if d180.Loc == scm.LocReg && d177.Loc == scm.LocReg && d180.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d179)
			var d181 scm.JITValueDesc
			r199 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r199, uint64(dataPtr))
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199, StackOff: int32(sliceLen)}
				ctx.BindReg(r199, &d181)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r199, thisptr.Reg, off)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d181)
			}
			ctx.BindReg(r199, &d181)
			ctx.EnsureDesc(&d180)
			var d182 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() / 64)}
			} else {
				r200 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r200, d180.Reg)
				ctx.W.EmitShrRegImm8(r200, 6)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d182)
			}
			if d182.Loc == scm.LocReg && d180.Loc == scm.LocReg && d182.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d182)
			r201 := ctx.AllocReg()
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d181)
			if d182.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r201, uint64(d182.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r201, d182.Reg)
				ctx.W.EmitShlRegImm8(r201, 3)
			}
			if d181.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
				ctx.W.EmitAddInt64(r201, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r201, d181.Reg)
			}
			r202 := ctx.AllocRegExcept(r201)
			ctx.W.EmitMovRegMem(r202, r201, 0)
			ctx.FreeReg(r201)
			d183 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
			ctx.BindReg(r202, &d183)
			ctx.FreeDesc(&d182)
			ctx.EnsureDesc(&d180)
			var d184 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() % 64)}
			} else {
				r203 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r203, d180.Reg)
				ctx.W.EmitAndRegImm32(r203, 63)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d184)
			}
			if d184.Loc == scm.LocReg && d180.Loc == scm.LocReg && d184.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d184)
			var d185 scm.JITValueDesc
			if d183.Loc == scm.LocImm && d184.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d183.Imm.Int()) << uint64(d184.Imm.Int())))}
			} else if d184.Loc == scm.LocImm {
				r204 := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegReg(r204, d183.Reg)
				ctx.W.EmitShlRegImm8(r204, uint8(d184.Imm.Int()))
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d185)
			} else {
				{
					shiftSrc := d183.Reg
					r205 := ctx.AllocRegExcept(d183.Reg)
					ctx.W.EmitMovRegReg(r205, d183.Reg)
					shiftSrc = r205
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d184.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d184.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d184.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d185)
				}
			}
			if d185.Loc == scm.LocReg && d183.Loc == scm.LocReg && d185.Reg == d183.Reg {
				ctx.TransferReg(d183.Reg)
				d183.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d183)
			ctx.FreeDesc(&d184)
			var d186 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d186)
			}
			d187 := d186
			ctx.EnsureDesc(&d187)
			if d187.Loc != scm.LocImm && d187.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			if d187.Loc == scm.LocImm {
				if d187.Imm.Bool() {
					ctx.W.EmitJmp(lbl59)
				} else {
					ctx.W.EmitJmp(lbl60)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d187.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl60)
			}
			ctx.W.MarkLabel(lbl59)
			ctx.W.EmitJmp(lbl57)
			ctx.W.MarkLabel(lbl60)
			d188 := d185
			if d188.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d188)
			ctx.EmitStoreToStack(d188, 32)
			ctx.W.EmitJmp(lbl58)
			ctx.FreeDesc(&d186)
			ctx.W.MarkLabel(lbl58)
			d189 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d190 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r207, thisptr.Reg, off)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
				ctx.BindReg(r207, &d190)
			}
			ctx.EnsureDesc(&d190)
			ctx.EnsureDesc(&d190)
			var d191 scm.JITValueDesc
			if d190.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d190.Imm.Int()))))}
			} else {
				r208 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r208, d190.Reg)
				ctx.W.EmitShlRegImm8(r208, 56)
				ctx.W.EmitShrRegImm8(r208, 56)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d191)
			}
			ctx.FreeDesc(&d190)
			d192 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d191)
			ctx.EnsureDesc(&d192)
			ctx.EnsureDesc(&d191)
			var d193 scm.JITValueDesc
			if d192.Loc == scm.LocImm && d191.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d192.Imm.Int() - d191.Imm.Int())}
			} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
				r209 := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(r209, d192.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d193)
			} else if d192.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d192.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d191.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d193)
			} else if d191.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(scratch, d192.Reg)
				if d191.Imm.Int() >= -2147483648 && d191.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d191.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d191.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d193)
			} else {
				r210 := ctx.AllocRegExcept(d192.Reg, d191.Reg)
				ctx.W.EmitMovRegReg(r210, d192.Reg)
				ctx.W.EmitSubInt64(r210, d191.Reg)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d193)
			}
			if d193.Loc == scm.LocReg && d192.Loc == scm.LocReg && d193.Reg == d192.Reg {
				ctx.TransferReg(d192.Reg)
				d192.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d191)
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d193)
			var d194 scm.JITValueDesc
			if d189.Loc == scm.LocImm && d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d189.Imm.Int()) >> uint64(d193.Imm.Int())))}
			} else if d193.Loc == scm.LocImm {
				r211 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r211, d189.Reg)
				ctx.W.EmitShrRegImm8(r211, uint8(d193.Imm.Int()))
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d194)
			} else {
				{
					shiftSrc := d189.Reg
					r212 := ctx.AllocRegExcept(d189.Reg)
					ctx.W.EmitMovRegReg(r212, d189.Reg)
					shiftSrc = r212
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d193.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d193.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d193.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d194)
				}
			}
			if d194.Loc == scm.LocReg && d189.Loc == scm.LocReg && d194.Reg == d189.Reg {
				ctx.TransferReg(d189.Reg)
				d189.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d189)
			ctx.FreeDesc(&d193)
			r213 := ctx.AllocReg()
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d194)
			if d194.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r213, d194)
			}
			ctx.W.EmitJmp(lbl55)
			ctx.W.MarkLabel(lbl57)
			d189 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d180)
			var d195 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() % 64)}
			} else {
				r214 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r214, d180.Reg)
				ctx.W.EmitAndRegImm32(r214, 63)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d195)
			}
			if d195.Loc == scm.LocReg && d180.Loc == scm.LocReg && d195.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			var d196 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r215 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r215, thisptr.Reg, off)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r215}
				ctx.BindReg(r215, &d196)
			}
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d196)
			var d197 scm.JITValueDesc
			if d196.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d196.Imm.Int()))))}
			} else {
				r216 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r216, d196.Reg)
				ctx.W.EmitShlRegImm8(r216, 56)
				ctx.W.EmitShrRegImm8(r216, 56)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d197)
			}
			ctx.FreeDesc(&d196)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d197)
			var d198 scm.JITValueDesc
			if d195.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d195.Imm.Int() + d197.Imm.Int())}
			} else if d197.Loc == scm.LocImm && d197.Imm.Int() == 0 {
				r217 := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(r217, d195.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d198)
			} else if d195.Loc == scm.LocImm && d195.Imm.Int() == 0 {
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d197.Reg}
				ctx.BindReg(d197.Reg, &d198)
			} else if d195.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d195.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d195.Reg)
				ctx.W.EmitMovRegReg(scratch, d195.Reg)
				if d197.Imm.Int() >= -2147483648 && d197.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d197.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d197.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else {
				r218 := ctx.AllocRegExcept(d195.Reg, d197.Reg)
				ctx.W.EmitMovRegReg(r218, d195.Reg)
				ctx.W.EmitAddInt64(r218, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d198)
			}
			if d198.Loc == scm.LocReg && d195.Loc == scm.LocReg && d198.Reg == d195.Reg {
				ctx.TransferReg(d195.Reg)
				d195.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d195)
			ctx.FreeDesc(&d197)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d198.Imm.Int()) > uint64(64))}
			} else {
				r219 := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitCmpRegImm32(d198.Reg, 64)
				ctx.W.EmitSetcc(r219, scm.CcA)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r219}
				ctx.BindReg(r219, &d199)
			}
			ctx.FreeDesc(&d198)
			d200 := d199
			ctx.EnsureDesc(&d200)
			if d200.Loc != scm.LocImm && d200.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl61 := ctx.W.ReserveLabel()
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			if d200.Loc == scm.LocImm {
				if d200.Imm.Bool() {
					ctx.W.EmitJmp(lbl62)
				} else {
					ctx.W.EmitJmp(lbl63)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d200.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl62)
				ctx.W.EmitJmp(lbl63)
			}
			ctx.W.MarkLabel(lbl62)
			ctx.W.EmitJmp(lbl61)
			ctx.W.MarkLabel(lbl63)
			d201 := d185
			if d201.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d201)
			ctx.EmitStoreToStack(d201, 32)
			ctx.W.EmitJmp(lbl58)
			ctx.FreeDesc(&d199)
			ctx.W.MarkLabel(lbl61)
			d189 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d180)
			var d202 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() / 64)}
			} else {
				r220 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r220, d180.Reg)
				ctx.W.EmitShrRegImm8(r220, 6)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d202)
			}
			if d202.Loc == scm.LocReg && d180.Loc == scm.LocReg && d202.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(scratch, d202.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			}
			if d203.Loc == scm.LocReg && d202.Loc == scm.LocReg && d203.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			r221 := ctx.AllocReg()
			ctx.EnsureDesc(&d203)
			ctx.EnsureDesc(&d181)
			if d203.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r221, uint64(d203.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r221, d203.Reg)
				ctx.W.EmitShlRegImm8(r221, 3)
			}
			if d181.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
				ctx.W.EmitAddInt64(r221, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r221, d181.Reg)
			}
			r222 := ctx.AllocRegExcept(r221)
			ctx.W.EmitMovRegMem(r222, r221, 0)
			ctx.FreeReg(r221)
			d204 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r222}
			ctx.BindReg(r222, &d204)
			ctx.FreeDesc(&d203)
			ctx.EnsureDesc(&d180)
			var d205 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d180.Imm.Int() % 64)}
			} else {
				r223 := ctx.AllocRegExcept(d180.Reg)
				ctx.W.EmitMovRegReg(r223, d180.Reg)
				ctx.W.EmitAndRegImm32(r223, 63)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d205)
			}
			if d205.Loc == scm.LocReg && d180.Loc == scm.LocReg && d205.Reg == d180.Reg {
				ctx.TransferReg(d180.Reg)
				d180.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d180)
			d206 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d205)
			ctx.EnsureDesc(&d206)
			ctx.EnsureDesc(&d205)
			var d207 scm.JITValueDesc
			if d206.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d206.Imm.Int() - d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(r224, d206.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d207)
			} else if d206.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d206.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d206.Reg)
				ctx.W.EmitMovRegReg(scratch, d206.Reg)
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else {
				r225 := ctx.AllocRegExcept(d206.Reg, d205.Reg)
				ctx.W.EmitMovRegReg(r225, d206.Reg)
				ctx.W.EmitSubInt64(r225, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d207)
			}
			if d207.Loc == scm.LocReg && d206.Loc == scm.LocReg && d207.Reg == d206.Reg {
				ctx.TransferReg(d206.Reg)
				d206.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			ctx.EnsureDesc(&d204)
			ctx.EnsureDesc(&d207)
			var d208 scm.JITValueDesc
			if d204.Loc == scm.LocImm && d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d204.Imm.Int()) >> uint64(d207.Imm.Int())))}
			} else if d207.Loc == scm.LocImm {
				r226 := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegReg(r226, d204.Reg)
				ctx.W.EmitShrRegImm8(r226, uint8(d207.Imm.Int()))
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d208)
			} else {
				{
					shiftSrc := d204.Reg
					r227 := ctx.AllocRegExcept(d204.Reg)
					ctx.W.EmitMovRegReg(r227, d204.Reg)
					shiftSrc = r227
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d207.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d207.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d207.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d208)
				}
			}
			if d208.Loc == scm.LocReg && d204.Loc == scm.LocReg && d208.Reg == d204.Reg {
				ctx.TransferReg(d204.Reg)
				d204.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			ctx.FreeDesc(&d207)
			ctx.EnsureDesc(&d185)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d185.Loc == scm.LocImm && d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() | d208.Imm.Int())}
			} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d208.Reg}
				ctx.BindReg(d208.Reg, &d209)
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r228, d185.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d209)
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d185.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			} else if d208.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r229, d185.Reg)
				if d208.Imm.Int() >= -2147483648 && d208.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r229, int32(d208.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d208.Imm.Int()))
					ctx.W.EmitOrInt64(r229, scm.RegR11)
				}
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d209)
			} else {
				r230 := ctx.AllocRegExcept(d185.Reg, d208.Reg)
				ctx.W.EmitMovRegReg(r230, d185.Reg)
				ctx.W.EmitOrInt64(r230, d208.Reg)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d209)
			}
			if d209.Loc == scm.LocReg && d185.Loc == scm.LocReg && d209.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			d210 := d209
			if d210.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d210)
			ctx.EmitStoreToStack(d210, 32)
			ctx.W.EmitJmp(lbl58)
			ctx.W.MarkLabel(lbl55)
			d211 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
			ctx.BindReg(r213, &d211)
			ctx.BindReg(r213, &d211)
			if r193 { ctx.UnprotectReg(r194) }
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d211)
			var d212 scm.JITValueDesc
			if d211.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d211.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d211.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d212)
			}
			ctx.FreeDesc(&d211)
			var d213 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r232, thisptr.Reg, off)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r232}
				ctx.BindReg(r232, &d213)
			}
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() + d213.Imm.Int())}
			} else if d213.Loc == scm.LocImm && d213.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r233, d212.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d214)
			} else if d212.Loc == scm.LocImm && d212.Imm.Int() == 0 {
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d213.Reg}
				ctx.BindReg(d213.Reg, &d214)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d213.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else if d213.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				if d213.Imm.Int() >= -2147483648 && d213.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d213.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d213.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d214)
			} else {
				r234 := ctx.AllocRegExcept(d212.Reg, d213.Reg)
				ctx.W.EmitMovRegReg(r234, d212.Reg)
				ctx.W.EmitAddInt64(r234, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d214)
			}
			if d214.Loc == scm.LocReg && d212.Loc == scm.LocReg && d214.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d212)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d214)
			ctx.EnsureDesc(&d214)
			var d215 scm.JITValueDesc
			if d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d214.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d215)
			}
			ctx.FreeDesc(&d214)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d84)
			var d216 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d216 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d84.Imm.Int()))))}
			} else {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r236, d84.Reg)
				d216 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d216)
			}
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d215)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d215)
			var d217 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d215.Loc == scm.LocImm {
				d217 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() + d215.Imm.Int())}
			} else if d215.Loc == scm.LocImm && d215.Imm.Int() == 0 {
				r237 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r237, d84.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d217)
			} else if d84.Loc == scm.LocImm && d84.Imm.Int() == 0 {
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d215.Reg}
				ctx.BindReg(d215.Reg, &d217)
			} else if d84.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d215.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d84.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else if d215.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(scratch, d84.Reg)
				if d215.Imm.Int() >= -2147483648 && d215.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d215.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d215.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d217)
			} else {
				r238 := ctx.AllocRegExcept(d84.Reg, d215.Reg)
				ctx.W.EmitMovRegReg(r238, d84.Reg)
				ctx.W.EmitAddInt64(r238, d215.Reg)
				d217 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d217)
			}
			if d217.Loc == scm.LocReg && d84.Loc == scm.LocReg && d217.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d215)
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d217.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r239, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d218)
			}
			ctx.FreeDesc(&d217)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d218)
			r240 := ctx.AllocReg()
			r241 := ctx.AllocRegExcept(r240)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d216)
			ctx.EnsureDesc(&d218)
			if d168.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r240, uint64(d168.Imm.Int()))
			} else if d168.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r240, d168.Reg)
			} else {
				ctx.W.EmitMovRegReg(r240, d168.Reg)
			}
			if d216.Loc == scm.LocImm {
				if d216.Imm.Int() != 0 {
					if d216.Imm.Int() >= -2147483648 && d216.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r240, int32(d216.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d216.Imm.Int()))
						ctx.W.EmitAddInt64(r240, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r240, d216.Reg)
			}
			if d218.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r241, uint64(d218.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r241, d218.Reg)
			}
			if d216.Loc == scm.LocImm {
				if d216.Imm.Int() >= -2147483648 && d216.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r241, int32(d216.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d216.Imm.Int()))
					ctx.W.EmitSubInt64(r241, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r241, d216.Reg)
			}
			d219 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r240, Reg2: r241}
			ctx.BindReg(r240, &d219)
			ctx.BindReg(r241, &d219)
			ctx.FreeDesc(&d216)
			ctx.FreeDesc(&d218)
			d220 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d220)
			ctx.BindReg(r187, &d220)
			d221 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d219}, 2)
			ctx.EmitMovPairToResult(&d221, &d220)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl30)
			var d222 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r242, thisptr.Reg, off)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d222)
			}
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d84)
			ctx.EnsureDesc(&d222)
			var d223 scm.JITValueDesc
			if d84.Loc == scm.LocImm && d222.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d84.Imm.Int()) == uint64(d222.Imm.Int()))}
			} else if d222.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d84.Reg)
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d84.Reg, int32(d222.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.W.EmitCmpInt64(d84.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r243, scm.CcE)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d223)
			} else if d84.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d84.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d222.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d223)
			} else {
				r245 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitCmpInt64(d84.Reg, d222.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d223)
			}
			ctx.FreeDesc(&d84)
			ctx.FreeDesc(&d222)
			d224 := d223
			ctx.EnsureDesc(&d224)
			if d224.Loc != scm.LocImm && d224.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			lbl66 := ctx.W.ReserveLabel()
			if d224.Loc == scm.LocImm {
				if d224.Imm.Bool() {
					ctx.W.EmitJmp(lbl65)
				} else {
					ctx.W.EmitJmp(lbl66)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d224.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl65)
				ctx.W.EmitJmp(lbl66)
			}
			ctx.W.MarkLabel(lbl65)
			ctx.W.EmitJmp(lbl64)
			ctx.W.MarkLabel(lbl66)
			ctx.W.EmitJmp(lbl31)
			ctx.FreeDesc(&d223)
			ctx.W.MarkLabel(lbl52)
			d225 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d225)
			ctx.BindReg(r187, &d225)
			ctx.W.EmitMakeNil(d225)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl64)
			d226 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d226)
			ctx.BindReg(r187, &d226)
			ctx.W.EmitMakeNil(d226)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl2)
			d227 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d227)
			ctx.BindReg(r187, &d227)
			ctx.BindReg(r186, &d227)
			ctx.BindReg(r187, &d227)
			if r2 { ctx.UnprotectReg(r3) }
			d229 := d227
			d229.ID = 0
			d228 := ctx.EmitTagEquals(&d229, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			d230 := d228
			ctx.EnsureDesc(&d230)
			if d230.Loc != scm.LocImm && d230.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl67 := ctx.W.ReserveLabel()
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			lbl70 := ctx.W.ReserveLabel()
			if d230.Loc == scm.LocImm {
				if d230.Imm.Bool() {
					ctx.W.EmitJmp(lbl69)
				} else {
					ctx.W.EmitJmp(lbl70)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d230.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl69)
				ctx.W.EmitJmp(lbl70)
			}
			ctx.W.MarkLabel(lbl69)
			ctx.W.EmitJmp(lbl67)
			ctx.W.MarkLabel(lbl70)
			ctx.W.EmitJmp(lbl68)
			ctx.FreeDesc(&d228)
			ctx.W.MarkLabel(lbl68)
			d232 := d227
			d232.ID = 0
			d231 := ctx.EmitTagEquals(&d232, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			d233 := d231
			ctx.EnsureDesc(&d233)
			if d233.Loc != scm.LocImm && d233.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			if d233.Loc == scm.LocImm {
				if d233.Imm.Bool() {
					ctx.W.EmitJmp(lbl73)
				} else {
					ctx.W.EmitJmp(lbl74)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d233.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl73)
				ctx.W.EmitJmp(lbl74)
			}
			ctx.W.MarkLabel(lbl73)
			ctx.W.EmitJmp(lbl71)
			ctx.W.MarkLabel(lbl74)
			ctx.W.EmitJmp(lbl72)
			ctx.FreeDesc(&d231)
			ctx.W.MarkLabel(lbl67)
			d234 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d234)
			ctx.BindReg(r1, &d234)
			ctx.W.EmitMakeNil(d234)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl72)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl71)
			ctx.EnsureDesc(&idxInt)
			d235 := idxInt
			_ = d235
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl76)
			ctx.EnsureDesc(&d235)
			ctx.EnsureDesc(&d235)
			var d236 scm.JITValueDesc
			if d235.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d235.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r248, d235.Reg)
				ctx.W.EmitShlRegImm8(r248, 32)
				ctx.W.EmitShrRegImm8(r248, 32)
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d236)
			}
			var d237 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r249, thisptr.Reg, off)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d237)
			}
			ctx.EnsureDesc(&d237)
			ctx.EnsureDesc(&d237)
			var d238 scm.JITValueDesc
			if d237.Loc == scm.LocImm {
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d237.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r250, d237.Reg)
				ctx.W.EmitShlRegImm8(r250, 56)
				ctx.W.EmitShrRegImm8(r250, 56)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d238)
			}
			ctx.FreeDesc(&d237)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d238)
			ctx.EnsureDesc(&d236)
			ctx.EnsureDesc(&d238)
			var d239 scm.JITValueDesc
			if d236.Loc == scm.LocImm && d238.Loc == scm.LocImm {
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d236.Imm.Int() * d238.Imm.Int())}
			} else if d236.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d238.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d236.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d238.Reg)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d239)
			} else if d238.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d236.Reg)
				ctx.W.EmitMovRegReg(scratch, d236.Reg)
				if d238.Imm.Int() >= -2147483648 && d238.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d238.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d238.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d239)
			} else {
				r251 := ctx.AllocRegExcept(d236.Reg, d238.Reg)
				ctx.W.EmitMovRegReg(r251, d236.Reg)
				ctx.W.EmitImulInt64(r251, d238.Reg)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d239)
			}
			if d239.Loc == scm.LocReg && d236.Loc == scm.LocReg && d239.Reg == d236.Reg {
				ctx.TransferReg(d236.Reg)
				d236.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d236)
			ctx.FreeDesc(&d238)
			var d240 scm.JITValueDesc
			r252 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r252, uint64(dataPtr))
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252, StackOff: int32(sliceLen)}
				ctx.BindReg(r252, &d240)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.W.EmitMovRegMem(r252, thisptr.Reg, off)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
				ctx.BindReg(r252, &d240)
			}
			ctx.BindReg(r252, &d240)
			ctx.EnsureDesc(&d239)
			var d241 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() / 64)}
			} else {
				r253 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r253, d239.Reg)
				ctx.W.EmitShrRegImm8(r253, 6)
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d241)
			}
			if d241.Loc == scm.LocReg && d239.Loc == scm.LocReg && d241.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d241)
			r254 := ctx.AllocReg()
			ctx.EnsureDesc(&d241)
			ctx.EnsureDesc(&d240)
			if d241.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r254, uint64(d241.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r254, d241.Reg)
				ctx.W.EmitShlRegImm8(r254, 3)
			}
			if d240.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d240.Imm.Int()))
				ctx.W.EmitAddInt64(r254, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r254, d240.Reg)
			}
			r255 := ctx.AllocRegExcept(r254)
			ctx.W.EmitMovRegMem(r255, r254, 0)
			ctx.FreeReg(r254)
			d242 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
			ctx.BindReg(r255, &d242)
			ctx.FreeDesc(&d241)
			ctx.EnsureDesc(&d239)
			var d243 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() % 64)}
			} else {
				r256 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r256, d239.Reg)
				ctx.W.EmitAndRegImm32(r256, 63)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d243)
			}
			if d243.Loc == scm.LocReg && d239.Loc == scm.LocReg && d243.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d242)
			ctx.EnsureDesc(&d243)
			var d244 scm.JITValueDesc
			if d242.Loc == scm.LocImm && d243.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d242.Imm.Int()) << uint64(d243.Imm.Int())))}
			} else if d243.Loc == scm.LocImm {
				r257 := ctx.AllocRegExcept(d242.Reg)
				ctx.W.EmitMovRegReg(r257, d242.Reg)
				ctx.W.EmitShlRegImm8(r257, uint8(d243.Imm.Int()))
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d244)
			} else {
				{
					shiftSrc := d242.Reg
					r258 := ctx.AllocRegExcept(d242.Reg)
					ctx.W.EmitMovRegReg(r258, d242.Reg)
					shiftSrc = r258
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d243.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d243.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d243.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d244)
				}
			}
			if d244.Loc == scm.LocReg && d242.Loc == scm.LocReg && d244.Reg == d242.Reg {
				ctx.TransferReg(d242.Reg)
				d242.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d242)
			ctx.FreeDesc(&d243)
			var d245 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r259 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r259, thisptr.Reg, off)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r259}
				ctx.BindReg(r259, &d245)
			}
			d246 := d245
			ctx.EnsureDesc(&d246)
			if d246.Loc != scm.LocImm && d246.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl77 := ctx.W.ReserveLabel()
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			lbl80 := ctx.W.ReserveLabel()
			if d246.Loc == scm.LocImm {
				if d246.Imm.Bool() {
					ctx.W.EmitJmp(lbl79)
				} else {
					ctx.W.EmitJmp(lbl80)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d246.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl79)
				ctx.W.EmitJmp(lbl80)
			}
			ctx.W.MarkLabel(lbl79)
			ctx.W.EmitJmp(lbl77)
			ctx.W.MarkLabel(lbl80)
			d247 := d244
			if d247.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d247)
			ctx.EmitStoreToStack(d247, 40)
			ctx.W.EmitJmp(lbl78)
			ctx.FreeDesc(&d245)
			ctx.W.MarkLabel(lbl78)
			d248 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			var d249 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r260 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r260, thisptr.Reg, off)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d249)
			}
			ctx.EnsureDesc(&d249)
			ctx.EnsureDesc(&d249)
			var d250 scm.JITValueDesc
			if d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d249.Imm.Int()))))}
			} else {
				r261 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r261, d249.Reg)
				ctx.W.EmitShlRegImm8(r261, 56)
				ctx.W.EmitShrRegImm8(r261, 56)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
				ctx.BindReg(r261, &d250)
			}
			ctx.FreeDesc(&d249)
			d251 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			ctx.EnsureDesc(&d251)
			ctx.EnsureDesc(&d250)
			var d252 scm.JITValueDesc
			if d251.Loc == scm.LocImm && d250.Loc == scm.LocImm {
				d252 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d251.Imm.Int() - d250.Imm.Int())}
			} else if d250.Loc == scm.LocImm && d250.Imm.Int() == 0 {
				r262 := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(r262, d251.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
				ctx.BindReg(r262, &d252)
			} else if d251.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d250.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d251.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d250.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d252)
			} else if d250.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d251.Reg)
				ctx.W.EmitMovRegReg(scratch, d251.Reg)
				if d250.Imm.Int() >= -2147483648 && d250.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d250.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d252)
			} else {
				r263 := ctx.AllocRegExcept(d251.Reg, d250.Reg)
				ctx.W.EmitMovRegReg(r263, d251.Reg)
				ctx.W.EmitSubInt64(r263, d250.Reg)
				d252 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d252)
			}
			if d252.Loc == scm.LocReg && d251.Loc == scm.LocReg && d252.Reg == d251.Reg {
				ctx.TransferReg(d251.Reg)
				d251.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d250)
			ctx.EnsureDesc(&d248)
			ctx.EnsureDesc(&d252)
			var d253 scm.JITValueDesc
			if d248.Loc == scm.LocImm && d252.Loc == scm.LocImm {
				d253 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d248.Imm.Int()) >> uint64(d252.Imm.Int())))}
			} else if d252.Loc == scm.LocImm {
				r264 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r264, d248.Reg)
				ctx.W.EmitShrRegImm8(r264, uint8(d252.Imm.Int()))
				d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d253)
			} else {
				{
					shiftSrc := d248.Reg
					r265 := ctx.AllocRegExcept(d248.Reg)
					ctx.W.EmitMovRegReg(r265, d248.Reg)
					shiftSrc = r265
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d252.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d252.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d252.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d253 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d253)
				}
			}
			if d253.Loc == scm.LocReg && d248.Loc == scm.LocReg && d253.Reg == d248.Reg {
				ctx.TransferReg(d248.Reg)
				d248.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d248)
			ctx.FreeDesc(&d252)
			r266 := ctx.AllocReg()
			ctx.EnsureDesc(&d253)
			ctx.EnsureDesc(&d253)
			if d253.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r266, d253)
			}
			ctx.W.EmitJmp(lbl75)
			ctx.W.MarkLabel(lbl77)
			d248 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d239)
			var d254 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() % 64)}
			} else {
				r267 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r267, d239.Reg)
				ctx.W.EmitAndRegImm32(r267, 63)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d254)
			}
			if d254.Loc == scm.LocReg && d239.Loc == scm.LocReg && d254.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			var d255 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r268 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r268, thisptr.Reg, off)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
				ctx.BindReg(r268, &d255)
			}
			ctx.EnsureDesc(&d255)
			ctx.EnsureDesc(&d255)
			var d256 scm.JITValueDesc
			if d255.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d255.Imm.Int()))))}
			} else {
				r269 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r269, d255.Reg)
				ctx.W.EmitShlRegImm8(r269, 56)
				ctx.W.EmitShrRegImm8(r269, 56)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d256)
			}
			ctx.FreeDesc(&d255)
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d256)
			ctx.EnsureDesc(&d254)
			ctx.EnsureDesc(&d256)
			var d257 scm.JITValueDesc
			if d254.Loc == scm.LocImm && d256.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d254.Imm.Int() + d256.Imm.Int())}
			} else if d256.Loc == scm.LocImm && d256.Imm.Int() == 0 {
				r270 := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(r270, d254.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
				ctx.BindReg(r270, &d257)
			} else if d254.Loc == scm.LocImm && d254.Imm.Int() == 0 {
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d256.Reg}
				ctx.BindReg(d256.Reg, &d257)
			} else if d254.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d254.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d256.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d257)
			} else if d256.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(scratch, d254.Reg)
				if d256.Imm.Int() >= -2147483648 && d256.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d256.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d256.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d257)
			} else {
				r271 := ctx.AllocRegExcept(d254.Reg, d256.Reg)
				ctx.W.EmitMovRegReg(r271, d254.Reg)
				ctx.W.EmitAddInt64(r271, d256.Reg)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d257)
			}
			if d257.Loc == scm.LocReg && d254.Loc == scm.LocReg && d257.Reg == d254.Reg {
				ctx.TransferReg(d254.Reg)
				d254.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d254)
			ctx.FreeDesc(&d256)
			ctx.EnsureDesc(&d257)
			var d258 scm.JITValueDesc
			if d257.Loc == scm.LocImm {
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d257.Imm.Int()) > uint64(64))}
			} else {
				r272 := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitCmpRegImm32(d257.Reg, 64)
				ctx.W.EmitSetcc(r272, scm.CcA)
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r272}
				ctx.BindReg(r272, &d258)
			}
			ctx.FreeDesc(&d257)
			d259 := d258
			ctx.EnsureDesc(&d259)
			if d259.Loc != scm.LocImm && d259.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			lbl83 := ctx.W.ReserveLabel()
			if d259.Loc == scm.LocImm {
				if d259.Imm.Bool() {
					ctx.W.EmitJmp(lbl82)
				} else {
					ctx.W.EmitJmp(lbl83)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d259.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl82)
				ctx.W.EmitJmp(lbl83)
			}
			ctx.W.MarkLabel(lbl82)
			ctx.W.EmitJmp(lbl81)
			ctx.W.MarkLabel(lbl83)
			d260 := d244
			if d260.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d260)
			ctx.EmitStoreToStack(d260, 40)
			ctx.W.EmitJmp(lbl78)
			ctx.FreeDesc(&d258)
			ctx.W.MarkLabel(lbl81)
			d248 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			ctx.EnsureDesc(&d239)
			var d261 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() / 64)}
			} else {
				r273 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r273, d239.Reg)
				ctx.W.EmitShrRegImm8(r273, 6)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d261)
			}
			if d261.Loc == scm.LocReg && d239.Loc == scm.LocReg && d261.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d261)
			ctx.EnsureDesc(&d261)
			var d262 scm.JITValueDesc
			if d261.Loc == scm.LocImm {
				d262 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d261.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d261.Reg)
				ctx.W.EmitMovRegReg(scratch, d261.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d262 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d262)
			}
			if d262.Loc == scm.LocReg && d261.Loc == scm.LocReg && d262.Reg == d261.Reg {
				ctx.TransferReg(d261.Reg)
				d261.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d261)
			ctx.EnsureDesc(&d262)
			r274 := ctx.AllocReg()
			ctx.EnsureDesc(&d262)
			ctx.EnsureDesc(&d240)
			if d262.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r274, uint64(d262.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r274, d262.Reg)
				ctx.W.EmitShlRegImm8(r274, 3)
			}
			if d240.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d240.Imm.Int()))
				ctx.W.EmitAddInt64(r274, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r274, d240.Reg)
			}
			r275 := ctx.AllocRegExcept(r274)
			ctx.W.EmitMovRegMem(r275, r274, 0)
			ctx.FreeReg(r274)
			d263 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r275}
			ctx.BindReg(r275, &d263)
			ctx.FreeDesc(&d262)
			ctx.EnsureDesc(&d239)
			var d264 scm.JITValueDesc
			if d239.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d239.Imm.Int() % 64)}
			} else {
				r276 := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegReg(r276, d239.Reg)
				ctx.W.EmitAndRegImm32(r276, 63)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
				ctx.BindReg(r276, &d264)
			}
			if d264.Loc == scm.LocReg && d239.Loc == scm.LocReg && d264.Reg == d239.Reg {
				ctx.TransferReg(d239.Reg)
				d239.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d239)
			d265 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d264)
			ctx.EnsureDesc(&d265)
			ctx.EnsureDesc(&d264)
			var d266 scm.JITValueDesc
			if d265.Loc == scm.LocImm && d264.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d265.Imm.Int() - d264.Imm.Int())}
			} else if d264.Loc == scm.LocImm && d264.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(r277, d265.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d266)
			} else if d265.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d264.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d265.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d264.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d266)
			} else if d264.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegReg(scratch, d265.Reg)
				if d264.Imm.Int() >= -2147483648 && d264.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d264.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d264.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d266)
			} else {
				r278 := ctx.AllocRegExcept(d265.Reg, d264.Reg)
				ctx.W.EmitMovRegReg(r278, d265.Reg)
				ctx.W.EmitSubInt64(r278, d264.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d266)
			}
			if d266.Loc == scm.LocReg && d265.Loc == scm.LocReg && d266.Reg == d265.Reg {
				ctx.TransferReg(d265.Reg)
				d265.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d264)
			ctx.EnsureDesc(&d263)
			ctx.EnsureDesc(&d266)
			var d267 scm.JITValueDesc
			if d263.Loc == scm.LocImm && d266.Loc == scm.LocImm {
				d267 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d263.Imm.Int()) >> uint64(d266.Imm.Int())))}
			} else if d266.Loc == scm.LocImm {
				r279 := ctx.AllocRegExcept(d263.Reg)
				ctx.W.EmitMovRegReg(r279, d263.Reg)
				ctx.W.EmitShrRegImm8(r279, uint8(d266.Imm.Int()))
				d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d267)
			} else {
				{
					shiftSrc := d263.Reg
					r280 := ctx.AllocRegExcept(d263.Reg)
					ctx.W.EmitMovRegReg(r280, d263.Reg)
					shiftSrc = r280
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d266.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d266.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d266.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d267 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d267)
				}
			}
			if d267.Loc == scm.LocReg && d263.Loc == scm.LocReg && d267.Reg == d263.Reg {
				ctx.TransferReg(d263.Reg)
				d263.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d263)
			ctx.FreeDesc(&d266)
			ctx.EnsureDesc(&d244)
			ctx.EnsureDesc(&d267)
			var d268 scm.JITValueDesc
			if d244.Loc == scm.LocImm && d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d244.Imm.Int() | d267.Imm.Int())}
			} else if d244.Loc == scm.LocImm && d244.Imm.Int() == 0 {
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d267.Reg}
				ctx.BindReg(d267.Reg, &d268)
			} else if d267.Loc == scm.LocImm && d267.Imm.Int() == 0 {
				r281 := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegReg(r281, d244.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d268)
			} else if d244.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d267.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d244.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d268)
			} else if d267.Loc == scm.LocImm {
				r282 := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegReg(r282, d244.Reg)
				if d267.Imm.Int() >= -2147483648 && d267.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r282, int32(d267.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
					ctx.W.EmitOrInt64(r282, scm.RegR11)
				}
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d268)
			} else {
				r283 := ctx.AllocRegExcept(d244.Reg, d267.Reg)
				ctx.W.EmitMovRegReg(r283, d244.Reg)
				ctx.W.EmitOrInt64(r283, d267.Reg)
				d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d268)
			}
			if d268.Loc == scm.LocReg && d244.Loc == scm.LocReg && d268.Reg == d244.Reg {
				ctx.TransferReg(d244.Reg)
				d244.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d267)
			d269 := d268
			if d269.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d269)
			ctx.EmitStoreToStack(d269, 40)
			ctx.W.EmitJmp(lbl78)
			ctx.W.MarkLabel(lbl75)
			d270 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r266}
			ctx.BindReg(r266, &d270)
			ctx.BindReg(r266, &d270)
			if r246 { ctx.UnprotectReg(r247) }
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d270)
			ctx.EnsureDesc(&d270)
			var d271 scm.JITValueDesc
			if d270.Loc == scm.LocImm {
				d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d270.Imm.Int()))))}
			} else {
				r284 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r284, d270.Reg)
				d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d271)
			}
			ctx.FreeDesc(&d270)
			var d272 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d272 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r285 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r285, thisptr.Reg, off)
				d272 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
				ctx.BindReg(r285, &d272)
			}
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d272)
			ctx.EnsureDesc(&d271)
			ctx.EnsureDesc(&d272)
			var d273 scm.JITValueDesc
			if d271.Loc == scm.LocImm && d272.Loc == scm.LocImm {
				d273 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d271.Imm.Int() + d272.Imm.Int())}
			} else if d272.Loc == scm.LocImm && d272.Imm.Int() == 0 {
				r286 := ctx.AllocRegExcept(d271.Reg)
				ctx.W.EmitMovRegReg(r286, d271.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d273)
			} else if d271.Loc == scm.LocImm && d271.Imm.Int() == 0 {
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d272.Reg}
				ctx.BindReg(d272.Reg, &d273)
			} else if d271.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d272.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d271.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d272.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d273)
			} else if d272.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d271.Reg)
				ctx.W.EmitMovRegReg(scratch, d271.Reg)
				if d272.Imm.Int() >= -2147483648 && d272.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d272.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d272.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d273)
			} else {
				r287 := ctx.AllocRegExcept(d271.Reg, d272.Reg)
				ctx.W.EmitMovRegReg(r287, d271.Reg)
				ctx.W.EmitAddInt64(r287, d272.Reg)
				d273 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
				ctx.BindReg(r287, &d273)
			}
			if d273.Loc == scm.LocReg && d271.Loc == scm.LocReg && d273.Reg == d271.Reg {
				ctx.TransferReg(d271.Reg)
				d271.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d271)
			ctx.FreeDesc(&d272)
			var d274 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r288 := ctx.AllocReg()
				r289 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r288, fieldAddr)
				ctx.W.EmitMovRegMem64(r289, fieldAddr+8)
				d274 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r288, Reg2: r289}
				ctx.BindReg(r288, &d274)
				ctx.BindReg(r289, &d274)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r290, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r291, thisptr.Reg, off+8)
				d274 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d274)
				ctx.BindReg(r291, &d274)
			}
			var d275 scm.JITValueDesc
			if d274.Loc == scm.LocImm {
				d275 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d274.StackOff))}
			} else {
				ctx.EnsureDesc(&d274)
				if d274.Loc == scm.LocRegPair {
					d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d274.Reg2}
					ctx.BindReg(d274.Reg2, &d275)
					ctx.BindReg(d274.Reg2, &d275)
				} else if d274.Loc == scm.LocReg {
					d275 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d274.Reg}
					ctx.BindReg(d274.Reg, &d275)
					ctx.BindReg(d274.Reg, &d275)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d275)
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d275)
			var d277 scm.JITValueDesc
			if d273.Loc == scm.LocImm && d275.Loc == scm.LocImm {
				d277 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d273.Imm.Int() >= d275.Imm.Int())}
			} else if d275.Loc == scm.LocImm {
				r292 := ctx.AllocRegExcept(d273.Reg)
				if d275.Imm.Int() >= -2147483648 && d275.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d273.Reg, int32(d275.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d275.Imm.Int()))
					ctx.W.EmitCmpInt64(d273.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r292, scm.CcGE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r292}
				ctx.BindReg(r292, &d277)
			} else if d273.Loc == scm.LocImm {
				r293 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d273.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d275.Reg)
				ctx.W.EmitSetcc(r293, scm.CcGE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r293}
				ctx.BindReg(r293, &d277)
			} else {
				r294 := ctx.AllocRegExcept(d273.Reg)
				ctx.W.EmitCmpInt64(d273.Reg, d275.Reg)
				ctx.W.EmitSetcc(r294, scm.CcGE)
				d277 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d277)
			}
			ctx.FreeDesc(&d275)
			d278 := d277
			ctx.EnsureDesc(&d278)
			if d278.Loc != scm.LocImm && d278.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl84 := ctx.W.ReserveLabel()
			lbl85 := ctx.W.ReserveLabel()
			lbl86 := ctx.W.ReserveLabel()
			lbl87 := ctx.W.ReserveLabel()
			if d278.Loc == scm.LocImm {
				if d278.Imm.Bool() {
					ctx.W.EmitJmp(lbl86)
				} else {
					ctx.W.EmitJmp(lbl87)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d278.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl86)
				ctx.W.EmitJmp(lbl87)
			}
			ctx.W.MarkLabel(lbl86)
			ctx.W.EmitJmp(lbl84)
			ctx.W.MarkLabel(lbl87)
			ctx.W.EmitJmp(lbl85)
			ctx.FreeDesc(&d277)
			ctx.W.MarkLabel(lbl85)
			ctx.EnsureDesc(&d273)
			var d279 scm.JITValueDesc
			if d273.Loc == scm.LocImm {
				d279 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d273.Imm.Int() < 0)}
			} else {
				r295 := ctx.AllocRegExcept(d273.Reg)
				ctx.W.EmitCmpRegImm32(d273.Reg, 0)
				ctx.W.EmitSetcc(r295, scm.CcL)
				d279 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r295}
				ctx.BindReg(r295, &d279)
			}
			d280 := d279
			ctx.EnsureDesc(&d280)
			if d280.Loc != scm.LocImm && d280.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl88 := ctx.W.ReserveLabel()
			lbl89 := ctx.W.ReserveLabel()
			lbl90 := ctx.W.ReserveLabel()
			if d280.Loc == scm.LocImm {
				if d280.Imm.Bool() {
					ctx.W.EmitJmp(lbl89)
				} else {
					ctx.W.EmitJmp(lbl90)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d280.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl89)
				ctx.W.EmitJmp(lbl90)
			}
			ctx.W.MarkLabel(lbl89)
			ctx.W.EmitJmp(lbl84)
			ctx.W.MarkLabel(lbl90)
			ctx.W.EmitJmp(lbl88)
			ctx.FreeDesc(&d279)
			ctx.W.MarkLabel(lbl84)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl88)
			ctx.EnsureDesc(&d273)
			r296 := ctx.AllocReg()
			ctx.EnsureDesc(&d273)
			ctx.EnsureDesc(&d274)
			if d273.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r296, uint64(d273.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r296, d273.Reg)
				ctx.W.EmitShlRegImm8(r296, 4)
			}
			if d274.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d274.Imm.Int()))
				ctx.W.EmitAddInt64(r296, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r296, d274.Reg)
			}
			r297 := ctx.AllocRegExcept(r296)
			r298 := ctx.AllocRegExcept(r296, r297)
			ctx.W.EmitMovRegMem(r297, r296, 0)
			ctx.W.EmitMovRegMem(r298, r296, 8)
			ctx.FreeReg(r296)
			d281 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r297, Reg2: r298}
			ctx.BindReg(r297, &d281)
			ctx.BindReg(r298, &d281)
			ctx.FreeDesc(&d273)
			if d227.Loc != scm.LocImm && d227.Type == scm.JITTypeUnknown {
				panic("jit: scm.Scmer.String on unknown dynamic type")
			}
			d282 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d227}, 2)
			ctx.FreeDesc(&d227)
			ctx.EnsureDesc(&d281)
			ctx.EnsureDesc(&d282)
			d283 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d281, d282}, 2)
			ctx.FreeDesc(&d281)
			d284 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d284)
			ctx.BindReg(r1, &d284)
			d285 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d283}, 2)
			ctx.EmitMovPairToResult(&d285, &d284)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d286 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d286)
			ctx.BindReg(r1, &d286)
			ctx.EmitMovPairToResult(&d286, &result)
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
