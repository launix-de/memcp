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
				if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			d0 := idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			lbl1 := ctx.W.ReserveLabel()
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
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d1.Loc == scm.LocImm {
				if d1.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d1.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d1)
			ctx.W.MarkLabel(lbl3)
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			d2 := d0
			_ = d2
			r5 := d0.Loc == scm.LocReg
			r6 := d0.Reg
			if r5 { ctx.ProtectReg(r6) }
			r7 := ctx.W.EmitSubRSP32Fixup()
			lbl5 := ctx.W.ReserveLabel()
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d2.Imm.Int()))))}
			} else {
				r8 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r8, d2.Reg)
				ctx.W.EmitShlRegImm8(r8, 32)
				ctx.W.EmitShrRegImm8(r8, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d3)
			}
			var d4 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r9 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r9, thisptr.Reg, off)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
				ctx.BindReg(r9, &d4)
			}
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d5 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d4.Imm.Int()))))}
			} else {
				r10 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitShlRegImm8(r10, 56)
				ctx.W.EmitShrRegImm8(r10, 56)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d5)
			}
			ctx.FreeDesc(&d4)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
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
				r11 := ctx.AllocRegExcept(d3.Reg, d5.Reg)
				ctx.W.EmitMovRegReg(r11, d3.Reg)
				ctx.W.EmitImulInt64(r11, d5.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d6)
			}
			if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d3)
			ctx.FreeDesc(&d5)
			var d7 scm.JITValueDesc
			r12 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r12, uint64(dataPtr))
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12, StackOff: int32(sliceLen)}
				ctx.BindReg(r12, &d7)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				ctx.W.EmitMovRegMem(r12, thisptr.Reg, off)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
				ctx.BindReg(r12, &d7)
			}
			ctx.BindReg(r12, &d7)
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d8 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r13 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r13, d6.Reg)
				ctx.W.EmitShrRegImm8(r13, 6)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r13}
				ctx.BindReg(r13, &d8)
			}
			if d8.Loc == scm.LocReg && d6.Loc == scm.LocReg && d8.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			r14 := ctx.AllocReg()
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r14, uint64(d8.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r14, d8.Reg)
				ctx.W.EmitShlRegImm8(r14, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.W.EmitAddInt64(r14, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r14, d7.Reg)
			}
			r15 := ctx.AllocRegExcept(r14)
			ctx.W.EmitMovRegMem(r15, r14, 0)
			ctx.FreeReg(r14)
			d9 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			ctx.BindReg(r15, &d9)
			ctx.FreeDesc(&d8)
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d10 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r16 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r16, d6.Reg)
				ctx.W.EmitAndRegImm32(r16, 63)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d10)
			}
			if d10.Loc == scm.LocReg && d6.Loc == scm.LocReg && d10.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			var d11 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) << uint64(d10.Imm.Int())))}
			} else if d10.Loc == scm.LocImm {
				r17 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r17, d9.Reg)
				ctx.W.EmitShlRegImm8(r17, uint8(d10.Imm.Int()))
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d11)
			} else {
				{
					shiftSrc := d9.Reg
					r18 := ctx.AllocRegExcept(d9.Reg)
					ctx.W.EmitMovRegReg(r18, d9.Reg)
					shiftSrc = r18
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
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 25)
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r19, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
				ctx.BindReg(r19, &d12)
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			d13 := d11
			if d13.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			ctx.EmitStoreToStack(d13, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d12.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			d14 := d11
			if d14.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			ctx.EmitStoreToStack(d14, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d12)
			ctx.W.MarkLabel(lbl7)
			d15 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d16 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r20, thisptr.Reg, off)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
				ctx.BindReg(r20, &d16)
			}
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 scm.JITValueDesc
			if d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d16.Imm.Int()))))}
			} else {
				r21 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r21, d16.Reg)
				ctx.W.EmitShlRegImm8(r21, 56)
				ctx.W.EmitShrRegImm8(r21, 56)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d17)
			}
			ctx.FreeDesc(&d16)
			d18 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			var d19 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() - d17.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				r22 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r22, d18.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d19)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(scratch, d18.Reg)
				if d17.Imm.Int() >= -2147483648 && d17.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d17.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d17.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else {
				r23 := ctx.AllocRegExcept(d18.Reg, d17.Reg)
				ctx.W.EmitMovRegReg(r23, d18.Reg)
				ctx.W.EmitSubInt64(r23, d17.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d19)
			}
			if d19.Loc == scm.LocReg && d18.Loc == scm.LocReg && d19.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) >> uint64(d19.Imm.Int())))}
			} else if d19.Loc == scm.LocImm {
				r24 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r24, d15.Reg)
				ctx.W.EmitShrRegImm8(r24, uint8(d19.Imm.Int()))
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r24}
				ctx.BindReg(r24, &d20)
			} else {
				{
					shiftSrc := d15.Reg
					r25 := ctx.AllocRegExcept(d15.Reg)
					ctx.W.EmitMovRegReg(r25, d15.Reg)
					shiftSrc = r25
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d19.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d19.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d19.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d20)
				}
			}
			if d20.Loc == scm.LocReg && d15.Loc == scm.LocReg && d20.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.FreeDesc(&d19)
			r26 := ctx.AllocReg()
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r26, d20)
			}
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			d15 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d21 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r27 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r27, d6.Reg)
				ctx.W.EmitAndRegImm32(r27, 63)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d21)
			}
			if d21.Loc == scm.LocReg && d6.Loc == scm.LocReg && d21.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			var d22 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r28, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
				ctx.BindReg(r28, &d22)
			}
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d22.Imm.Int()))))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r29, d22.Reg)
				ctx.W.EmitShlRegImm8(r29, 56)
				ctx.W.EmitShrRegImm8(r29, 56)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d23)
			}
			ctx.FreeDesc(&d22)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			var d24 scm.JITValueDesc
			if d21.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() + d23.Imm.Int())}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				r30 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r30, d21.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d24)
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
				ctx.BindReg(d23.Reg, &d24)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d24)
			} else {
				r31 := ctx.AllocRegExcept(d21.Reg, d23.Reg)
				ctx.W.EmitMovRegReg(r31, d21.Reg)
				ctx.W.EmitAddInt64(r31, d23.Reg)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d24)
			}
			if d24.Loc == scm.LocReg && d21.Loc == scm.LocReg && d24.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d23)
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d24.Imm.Int()) > uint64(64))}
			} else {
				r32 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitCmpRegImm32(d24.Reg, 64)
				ctx.W.EmitSetcc(r32, scm.CcA)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
				ctx.BindReg(r32, &d25)
			}
			ctx.FreeDesc(&d24)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d25.Loc == scm.LocImm {
				if d25.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d26 := d11
			if d26.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			ctx.EmitStoreToStack(d26, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d25.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			d27 := d11
			if d27.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			ctx.EmitStoreToStack(d27, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d25)
			ctx.W.MarkLabel(lbl9)
			d15 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d28 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 64)}
			} else {
				r33 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r33, d6.Reg)
				ctx.W.EmitShrRegImm8(r33, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d28)
			}
			if d28.Loc == scm.LocReg && d6.Loc == scm.LocReg && d28.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
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
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			r34 := ctx.AllocReg()
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r34, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r34, d29.Reg)
				ctx.W.EmitShlRegImm8(r34, 3)
			}
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d7.Imm.Int()))
				ctx.W.EmitAddInt64(r34, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r34, d7.Reg)
			}
			r35 := ctx.AllocRegExcept(r34)
			ctx.W.EmitMovRegMem(r35, r34, 0)
			ctx.FreeReg(r34)
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
			ctx.BindReg(r35, &d30)
			ctx.FreeDesc(&d29)
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d31 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() % 64)}
			} else {
				r36 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r36, d6.Reg)
				ctx.W.EmitAndRegImm32(r36, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d31)
			}
			if d31.Loc == scm.LocReg && d6.Loc == scm.LocReg && d31.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			d32 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegReg(r37, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d33)
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
				r38 := ctx.AllocRegExcept(d32.Reg, d31.Reg)
				ctx.W.EmitMovRegReg(r38, d32.Reg)
				ctx.W.EmitSubInt64(r38, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d33)
			}
			if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			if d33.Loc == scm.LocStack || d33.Loc == scm.LocStackPair { ctx.EnsureDesc(&d33) }
			var d34 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r39, d30.Reg)
				ctx.W.EmitShrRegImm8(r39, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d34)
			} else {
				{
					shiftSrc := d30.Reg
					r40 := ctx.AllocRegExcept(d30.Reg)
					ctx.W.EmitMovRegReg(r40, d30.Reg)
					shiftSrc = r40
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
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d35 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() | d34.Imm.Int())}
			} else if d11.Loc == scm.LocImm && d11.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d34.Reg}
				ctx.BindReg(d34.Reg, &d35)
			} else if d34.Loc == scm.LocImm && d34.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r41, d11.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d35)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d34.Loc == scm.LocImm {
				r42 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r42, d11.Reg)
				if d34.Imm.Int() >= -2147483648 && d34.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r42, int32(d34.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d34.Imm.Int()))
					ctx.W.EmitOrInt64(r42, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d35)
			} else {
				r43 := ctx.AllocRegExcept(d11.Reg, d34.Reg)
				ctx.W.EmitMovRegReg(r43, d11.Reg)
				ctx.W.EmitOrInt64(r43, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d35)
			}
			if d35.Loc == scm.LocReg && d11.Loc == scm.LocReg && d35.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			d36 := d35
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			ctx.EmitStoreToStack(d36, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
			ctx.BindReg(r26, &d37)
			ctx.BindReg(r26, &d37)
			if r5 { ctx.UnprotectReg(r6) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d37.Imm.Int()))))}
			} else {
				r44 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r44, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d38)
			}
			ctx.FreeDesc(&d37)
			var d39 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r45 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r45, thisptr.Reg, off)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r45}
				ctx.BindReg(r45, &d39)
			}
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				r46 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r46, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d40)
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
				ctx.BindReg(d39.Reg, &d40)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r47 := ctx.AllocRegExcept(d38.Reg, d39.Reg)
				ctx.W.EmitMovRegReg(r47, d38.Reg)
				ctx.W.EmitAddInt64(r47, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d40)
			}
			if d40.Loc == scm.LocReg && d38.Loc == scm.LocReg && d40.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d39)
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d40.Imm.Int()))))}
			} else {
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r48, d40.Reg)
				ctx.W.EmitShlRegImm8(r48, 32)
				ctx.W.EmitShrRegImm8(r48, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d41)
			}
			ctx.FreeDesc(&d40)
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r49, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r49}
				ctx.BindReg(r49, &d42)
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d42.Loc == scm.LocImm {
				if d42.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d42.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d42)
			ctx.W.MarkLabel(lbl2)
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			d43 := d0
			_ = d43
			r50 := d0.Loc == scm.LocReg
			r51 := d0.Reg
			if r50 { ctx.ProtectReg(r51) }
			lbl14 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d43.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d43.Reg)
				ctx.W.EmitShlRegImm8(r52, 32)
				ctx.W.EmitShrRegImm8(r52, 32)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d44)
			}
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r53 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r53, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r53}
				ctx.BindReg(r53, &d45)
			}
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			var d46 scm.JITValueDesc
			if d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d45.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, d45.Reg)
				ctx.W.EmitShlRegImm8(r54, 56)
				ctx.W.EmitShrRegImm8(r54, 56)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d46)
			}
			ctx.FreeDesc(&d45)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			var d47 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() * d46.Imm.Int())}
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(scratch, d44.Reg)
				if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d46.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else {
				r55 := ctx.AllocRegExcept(d44.Reg, d46.Reg)
				ctx.W.EmitMovRegReg(r55, d44.Reg)
				ctx.W.EmitImulInt64(r55, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d47)
			}
			if d47.Loc == scm.LocReg && d44.Loc == scm.LocReg && d47.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d46)
			var d48 scm.JITValueDesc
			r56 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r56, uint64(dataPtr))
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56, StackOff: int32(sliceLen)}
				ctx.BindReg(r56, &d48)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r56, thisptr.Reg, off)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r56}
				ctx.BindReg(r56, &d48)
			}
			ctx.BindReg(r56, &d48)
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() / 64)}
			} else {
				r57 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r57, d47.Reg)
				ctx.W.EmitShrRegImm8(r57, 6)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r57}
				ctx.BindReg(r57, &d49)
			}
			if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			r58 := ctx.AllocReg()
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			if d49.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r58, uint64(d49.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r58, d49.Reg)
				ctx.W.EmitShlRegImm8(r58, 3)
			}
			if d48.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitAddInt64(r58, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r58, d48.Reg)
			}
			r59 := ctx.AllocRegExcept(r58)
			ctx.W.EmitMovRegMem(r59, r58, 0)
			ctx.FreeReg(r58)
			d50 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r59}
			ctx.BindReg(r59, &d50)
			ctx.FreeDesc(&d49)
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d51 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() % 64)}
			} else {
				r60 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r60, d47.Reg)
				ctx.W.EmitAndRegImm32(r60, 63)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r60}
				ctx.BindReg(r60, &d51)
			}
			if d51.Loc == scm.LocReg && d47.Loc == scm.LocReg && d51.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			if d50.Loc == scm.LocStack || d50.Loc == scm.LocStackPair { ctx.EnsureDesc(&d50) }
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			var d52 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) << uint64(d51.Imm.Int())))}
			} else if d51.Loc == scm.LocImm {
				r61 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitMovRegReg(r61, d50.Reg)
				ctx.W.EmitShlRegImm8(r61, uint8(d51.Imm.Int()))
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r61}
				ctx.BindReg(r61, &d52)
			} else {
				{
					shiftSrc := d50.Reg
					r62 := ctx.AllocRegExcept(d50.Reg)
					ctx.W.EmitMovRegReg(r62, d50.Reg)
					shiftSrc = r62
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d51.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d51.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d51.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d52)
				}
			}
			if d52.Loc == scm.LocReg && d50.Loc == scm.LocReg && d52.Reg == d50.Reg {
				ctx.TransferReg(d50.Reg)
				d50.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d51)
			var d53 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r63, thisptr.Reg, off)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r63}
				ctx.BindReg(r63, &d53)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d53.Loc == scm.LocImm {
				if d53.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
			d54 := d52
			if d54.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			ctx.EmitStoreToStack(d54, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d53.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
			d55 := d52
			if d55.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			ctx.EmitStoreToStack(d55, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.FreeDesc(&d53)
			ctx.W.MarkLabel(lbl16)
			d56 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d57 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r64, thisptr.Reg, off)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r64}
				ctx.BindReg(r64, &d57)
			}
			if d57.Loc == scm.LocStack || d57.Loc == scm.LocStackPair { ctx.EnsureDesc(&d57) }
			if d57.Loc == scm.LocStack || d57.Loc == scm.LocStackPair { ctx.EnsureDesc(&d57) }
			var d58 scm.JITValueDesc
			if d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d57.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, d57.Reg)
				ctx.W.EmitShlRegImm8(r65, 56)
				ctx.W.EmitShrRegImm8(r65, 56)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d58)
			}
			ctx.FreeDesc(&d57)
			d59 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			var d60 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d58.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() - d58.Imm.Int())}
			} else if d58.Loc == scm.LocImm && d58.Imm.Int() == 0 {
				r66 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r66, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d60)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d58.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d58.Imm.Int() >= -2147483648 && d58.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d58.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else {
				r67 := ctx.AllocRegExcept(d59.Reg, d58.Reg)
				ctx.W.EmitMovRegReg(r67, d59.Reg)
				ctx.W.EmitSubInt64(r67, d58.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r67}
				ctx.BindReg(r67, &d60)
			}
			if d60.Loc == scm.LocReg && d59.Loc == scm.LocReg && d60.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			if d56.Loc == scm.LocStack || d56.Loc == scm.LocStackPair { ctx.EnsureDesc(&d56) }
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			var d61 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d56.Imm.Int()) >> uint64(d60.Imm.Int())))}
			} else if d60.Loc == scm.LocImm {
				r68 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r68, d56.Reg)
				ctx.W.EmitShrRegImm8(r68, uint8(d60.Imm.Int()))
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r68}
				ctx.BindReg(r68, &d61)
			} else {
				{
					shiftSrc := d56.Reg
					r69 := ctx.AllocRegExcept(d56.Reg)
					ctx.W.EmitMovRegReg(r69, d56.Reg)
					shiftSrc = r69
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d60.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d60.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d60.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d61)
				}
			}
			if d61.Loc == scm.LocReg && d56.Loc == scm.LocReg && d61.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d56)
			ctx.FreeDesc(&d60)
			r70 := ctx.AllocReg()
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			if d61.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r70, d61)
			}
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl15)
			d56 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d62 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() % 64)}
			} else {
				r71 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r71, d47.Reg)
				ctx.W.EmitAndRegImm32(r71, 63)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d62)
			}
			if d62.Loc == scm.LocReg && d47.Loc == scm.LocReg && d62.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			var d63 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r72 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r72, thisptr.Reg, off)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r72}
				ctx.BindReg(r72, &d63)
			}
			if d63.Loc == scm.LocStack || d63.Loc == scm.LocStackPair { ctx.EnsureDesc(&d63) }
			if d63.Loc == scm.LocStack || d63.Loc == scm.LocStackPair { ctx.EnsureDesc(&d63) }
			var d64 scm.JITValueDesc
			if d63.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d63.Imm.Int()))))}
			} else {
				r73 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r73, d63.Reg)
				ctx.W.EmitShlRegImm8(r73, 56)
				ctx.W.EmitShrRegImm8(r73, 56)
				d64 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d64)
			}
			ctx.FreeDesc(&d63)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			var d65 scm.JITValueDesc
			if d62.Loc == scm.LocImm && d64.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d62.Imm.Int() + d64.Imm.Int())}
			} else if d64.Loc == scm.LocImm && d64.Imm.Int() == 0 {
				r74 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(r74, d62.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r74}
				ctx.BindReg(r74, &d65)
			} else if d62.Loc == scm.LocImm && d62.Imm.Int() == 0 {
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d64.Reg}
				ctx.BindReg(d64.Reg, &d65)
			} else if d62.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d64.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d62.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d64.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else if d64.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitMovRegReg(scratch, d62.Reg)
				if d64.Imm.Int() >= -2147483648 && d64.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d64.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d64.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			} else {
				r75 := ctx.AllocRegExcept(d62.Reg, d64.Reg)
				ctx.W.EmitMovRegReg(r75, d62.Reg)
				ctx.W.EmitAddInt64(r75, d64.Reg)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d65)
			}
			if d65.Loc == scm.LocReg && d62.Loc == scm.LocReg && d65.Reg == d62.Reg {
				ctx.TransferReg(d62.Reg)
				d62.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d62)
			ctx.FreeDesc(&d64)
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d65.Imm.Int()) > uint64(64))}
			} else {
				r76 := ctx.AllocRegExcept(d65.Reg)
				ctx.W.EmitCmpRegImm32(d65.Reg, 64)
				ctx.W.EmitSetcc(r76, scm.CcA)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r76}
				ctx.BindReg(r76, &d66)
			}
			ctx.FreeDesc(&d65)
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d66.Loc == scm.LocImm {
				if d66.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
			d67 := d52
			if d67.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			ctx.EmitStoreToStack(d67, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d66.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
			d68 := d52
			if d68.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d68.Loc == scm.LocStack || d68.Loc == scm.LocStackPair { ctx.EnsureDesc(&d68) }
			ctx.EmitStoreToStack(d68, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.FreeDesc(&d66)
			ctx.W.MarkLabel(lbl18)
			d56 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d69 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() / 64)}
			} else {
				r77 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r77, d47.Reg)
				ctx.W.EmitShrRegImm8(r77, 6)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r77}
				ctx.BindReg(r77, &d69)
			}
			if d69.Loc == scm.LocReg && d47.Loc == scm.LocReg && d69.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegReg(scratch, d69.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			}
			if d70.Loc == scm.LocReg && d69.Loc == scm.LocReg && d70.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d69)
			if d70.Loc == scm.LocStack || d70.Loc == scm.LocStackPair { ctx.EnsureDesc(&d70) }
			r78 := ctx.AllocReg()
			if d70.Loc == scm.LocStack || d70.Loc == scm.LocStackPair { ctx.EnsureDesc(&d70) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			if d70.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r78, uint64(d70.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r78, d70.Reg)
				ctx.W.EmitShlRegImm8(r78, 3)
			}
			if d48.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitAddInt64(r78, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r78, d48.Reg)
			}
			r79 := ctx.AllocRegExcept(r78)
			ctx.W.EmitMovRegMem(r79, r78, 0)
			ctx.FreeReg(r78)
			d71 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r79}
			ctx.BindReg(r79, &d71)
			ctx.FreeDesc(&d70)
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d72 scm.JITValueDesc
			if d47.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() % 64)}
			} else {
				r80 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r80, d47.Reg)
				ctx.W.EmitAndRegImm32(r80, 63)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d72)
			}
			if d72.Loc == scm.LocReg && d47.Loc == scm.LocReg && d72.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			d73 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			if d73.Loc == scm.LocStack || d73.Loc == scm.LocStackPair { ctx.EnsureDesc(&d73) }
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			if d73.Loc == scm.LocStack || d73.Loc == scm.LocStackPair { ctx.EnsureDesc(&d73) }
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			var d74 scm.JITValueDesc
			if d73.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d73.Imm.Int() - d72.Imm.Int())}
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				r81 := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegReg(r81, d73.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d74)
			} else if d73.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d73.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d74)
			} else if d72.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d73.Reg)
				ctx.W.EmitMovRegReg(scratch, d73.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d72.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d74)
			} else {
				r82 := ctx.AllocRegExcept(d73.Reg, d72.Reg)
				ctx.W.EmitMovRegReg(r82, d73.Reg)
				ctx.W.EmitSubInt64(r82, d72.Reg)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r82}
				ctx.BindReg(r82, &d74)
			}
			if d74.Loc == scm.LocReg && d73.Loc == scm.LocReg && d74.Reg == d73.Reg {
				ctx.TransferReg(d73.Reg)
				d73.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			var d75 scm.JITValueDesc
			if d71.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d71.Imm.Int()) >> uint64(d74.Imm.Int())))}
			} else if d74.Loc == scm.LocImm {
				r83 := ctx.AllocRegExcept(d71.Reg)
				ctx.W.EmitMovRegReg(r83, d71.Reg)
				ctx.W.EmitShrRegImm8(r83, uint8(d74.Imm.Int()))
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d75)
			} else {
				{
					shiftSrc := d71.Reg
					r84 := ctx.AllocRegExcept(d71.Reg)
					ctx.W.EmitMovRegReg(r84, d71.Reg)
					shiftSrc = r84
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d74.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d74.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d74.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d75)
				}
			}
			if d75.Loc == scm.LocReg && d71.Loc == scm.LocReg && d75.Reg == d71.Reg {
				ctx.TransferReg(d71.Reg)
				d71.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.FreeDesc(&d74)
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			var d76 scm.JITValueDesc
			if d52.Loc == scm.LocImm && d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d52.Imm.Int() | d75.Imm.Int())}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d75.Reg}
				ctx.BindReg(d75.Reg, &d76)
			} else if d75.Loc == scm.LocImm && d75.Imm.Int() == 0 {
				r85 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r85, d52.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d76)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d75.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d52.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d76)
			} else if d75.Loc == scm.LocImm {
				r86 := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegReg(r86, d52.Reg)
				if d75.Imm.Int() >= -2147483648 && d75.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r86, int32(d75.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d75.Imm.Int()))
					ctx.W.EmitOrInt64(r86, scm.RegR11)
				}
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d76)
			} else {
				r87 := ctx.AllocRegExcept(d52.Reg, d75.Reg)
				ctx.W.EmitMovRegReg(r87, d52.Reg)
				ctx.W.EmitOrInt64(r87, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r87}
				ctx.BindReg(r87, &d76)
			}
			if d76.Loc == scm.LocReg && d52.Loc == scm.LocReg && d76.Reg == d52.Reg {
				ctx.TransferReg(d52.Reg)
				d52.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d75)
			d77 := d76
			if d77.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			ctx.EmitStoreToStack(d77, 8)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl14)
			d78 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
			ctx.BindReg(r70, &d78)
			ctx.BindReg(r70, &d78)
			if r50 { ctx.UnprotectReg(r51) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d78.Imm.Int()))))}
			} else {
				r88 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r88, d78.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d79)
			}
			ctx.FreeDesc(&d78)
			var d80 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r89 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r89, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r89}
				ctx.BindReg(r89, &d80)
			}
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair { ctx.EnsureDesc(&d80) }
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair { ctx.EnsureDesc(&d80) }
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d80.Loc == scm.LocStack || d80.Loc == scm.LocStackPair { ctx.EnsureDesc(&d80) }
			var d81 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d80.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() + d80.Imm.Int())}
			} else if d80.Loc == scm.LocImm && d80.Imm.Int() == 0 {
				r90 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r90, d79.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d81)
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d80.Reg}
				ctx.BindReg(d80.Reg, &d81)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d80.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			} else if d80.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				if d80.Imm.Int() >= -2147483648 && d80.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d80.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d80.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d81)
			} else {
				r91 := ctx.AllocRegExcept(d79.Reg, d80.Reg)
				ctx.W.EmitMovRegReg(r91, d79.Reg)
				ctx.W.EmitAddInt64(r91, d80.Reg)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r91}
				ctx.BindReg(r91, &d81)
			}
			if d81.Loc == scm.LocReg && d79.Loc == scm.LocReg && d81.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d79)
			ctx.FreeDesc(&d80)
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			var d82 scm.JITValueDesc
			if d81.Loc == scm.LocImm {
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d81.Imm.Int()))))}
			} else {
				r92 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r92, d81.Reg)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r92}
				ctx.BindReg(r92, &d82)
			}
			ctx.FreeDesc(&d81)
			var d83 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r93 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r93, thisptr.Reg, off)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r93}
				ctx.BindReg(r93, &d83)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d83.Loc == scm.LocImm {
				if d83.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d83.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d83)
			ctx.W.MarkLabel(lbl12)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			d84 := d41
			_ = d84
			r94 := d41.Loc == scm.LocReg
			r95 := d41.Reg
			if r94 { ctx.ProtectReg(r95) }
			lbl23 := ctx.W.ReserveLabel()
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d85 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d84.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d84.Reg)
				ctx.W.EmitShlRegImm8(r96, 32)
				ctx.W.EmitShrRegImm8(r96, 32)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d85)
			}
			var d86 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r97 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r97, thisptr.Reg, off)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r97}
				ctx.BindReg(r97, &d86)
			}
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			var d87 scm.JITValueDesc
			if d86.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d86.Imm.Int()))))}
			} else {
				r98 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r98, d86.Reg)
				ctx.W.EmitShlRegImm8(r98, 56)
				ctx.W.EmitShrRegImm8(r98, 56)
				d87 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r98}
				ctx.BindReg(r98, &d87)
			}
			ctx.FreeDesc(&d86)
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			var d88 scm.JITValueDesc
			if d85.Loc == scm.LocImm && d87.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d85.Imm.Int() * d87.Imm.Int())}
			} else if d85.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d85.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d85.Reg)
				ctx.W.EmitMovRegReg(scratch, d85.Reg)
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d87.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d88)
			} else {
				r99 := ctx.AllocRegExcept(d85.Reg, d87.Reg)
				ctx.W.EmitMovRegReg(r99, d85.Reg)
				ctx.W.EmitImulInt64(r99, d87.Reg)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d88)
			}
			if d88.Loc == scm.LocReg && d85.Loc == scm.LocReg && d88.Reg == d85.Reg {
				ctx.TransferReg(d85.Reg)
				d85.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d85)
			ctx.FreeDesc(&d87)
			var d89 scm.JITValueDesc
			r100 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r100, uint64(dataPtr))
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100, StackOff: int32(sliceLen)}
				ctx.BindReg(r100, &d89)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r100, thisptr.Reg, off)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r100}
				ctx.BindReg(r100, &d89)
			}
			ctx.BindReg(r100, &d89)
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d90 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() / 64)}
			} else {
				r101 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r101, d88.Reg)
				ctx.W.EmitShrRegImm8(r101, 6)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r101}
				ctx.BindReg(r101, &d90)
			}
			if d90.Loc == scm.LocReg && d88.Loc == scm.LocReg && d90.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			if d90.Loc == scm.LocStack || d90.Loc == scm.LocStackPair { ctx.EnsureDesc(&d90) }
			r102 := ctx.AllocReg()
			if d90.Loc == scm.LocStack || d90.Loc == scm.LocStackPair { ctx.EnsureDesc(&d90) }
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			if d90.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r102, uint64(d90.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r102, d90.Reg)
				ctx.W.EmitShlRegImm8(r102, 3)
			}
			if d89.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(r102, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r102, d89.Reg)
			}
			r103 := ctx.AllocRegExcept(r102)
			ctx.W.EmitMovRegMem(r103, r102, 0)
			ctx.FreeReg(r102)
			d91 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r103}
			ctx.BindReg(r103, &d91)
			ctx.FreeDesc(&d90)
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d92 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				r104 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r104, d88.Reg)
				ctx.W.EmitAndRegImm32(r104, 63)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r104}
				ctx.BindReg(r104, &d92)
			}
			if d92.Loc == scm.LocReg && d88.Loc == scm.LocReg && d92.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			if d91.Loc == scm.LocStack || d91.Loc == scm.LocStackPair { ctx.EnsureDesc(&d91) }
			if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair { ctx.EnsureDesc(&d92) }
			var d93 scm.JITValueDesc
			if d91.Loc == scm.LocImm && d92.Loc == scm.LocImm {
				d93 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d91.Imm.Int()) << uint64(d92.Imm.Int())))}
			} else if d92.Loc == scm.LocImm {
				r105 := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegReg(r105, d91.Reg)
				ctx.W.EmitShlRegImm8(r105, uint8(d92.Imm.Int()))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r105}
				ctx.BindReg(r105, &d93)
			} else {
				{
					shiftSrc := d91.Reg
					r106 := ctx.AllocRegExcept(d91.Reg)
					ctx.W.EmitMovRegReg(r106, d91.Reg)
					shiftSrc = r106
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d92.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d92.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d92.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d93 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d93)
				}
			}
			if d93.Loc == scm.LocReg && d91.Loc == scm.LocReg && d93.Reg == d91.Reg {
				ctx.TransferReg(d91.Reg)
				d91.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d91)
			ctx.FreeDesc(&d92)
			var d94 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r107, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r107}
				ctx.BindReg(r107, &d94)
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d94.Loc == scm.LocImm {
				if d94.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
			d95 := d93
			if d95.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			ctx.EmitStoreToStack(d95, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d94.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
			d96 := d93
			if d96.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d96.Loc == scm.LocStack || d96.Loc == scm.LocStackPair { ctx.EnsureDesc(&d96) }
			ctx.EmitStoreToStack(d96, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d94)
			ctx.W.MarkLabel(lbl25)
			d97 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d98 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r108 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r108, thisptr.Reg, off)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r108}
				ctx.BindReg(r108, &d98)
			}
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			var d99 scm.JITValueDesc
			if d98.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d98.Imm.Int()))))}
			} else {
				r109 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r109, d98.Reg)
				ctx.W.EmitShlRegImm8(r109, 56)
				ctx.W.EmitShrRegImm8(r109, 56)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d99)
			}
			ctx.FreeDesc(&d98)
			d100 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm && d99.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d100.Imm.Int() - d99.Imm.Int())}
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				r110 := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(r110, d100.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d101)
			} else if d100.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d100.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d99.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d101)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d100.Reg)
				ctx.W.EmitMovRegReg(scratch, d100.Reg)
				if d99.Imm.Int() >= -2147483648 && d99.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d99.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d99.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d101)
			} else {
				r111 := ctx.AllocRegExcept(d100.Reg, d99.Reg)
				ctx.W.EmitMovRegReg(r111, d100.Reg)
				ctx.W.EmitSubInt64(r111, d99.Reg)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r111}
				ctx.BindReg(r111, &d101)
			}
			if d101.Loc == scm.LocReg && d100.Loc == scm.LocReg && d101.Reg == d100.Reg {
				ctx.TransferReg(d100.Reg)
				d100.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			var d102 scm.JITValueDesc
			if d97.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d97.Imm.Int()) >> uint64(d101.Imm.Int())))}
			} else if d101.Loc == scm.LocImm {
				r112 := ctx.AllocRegExcept(d97.Reg)
				ctx.W.EmitMovRegReg(r112, d97.Reg)
				ctx.W.EmitShrRegImm8(r112, uint8(d101.Imm.Int()))
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r112}
				ctx.BindReg(r112, &d102)
			} else {
				{
					shiftSrc := d97.Reg
					r113 := ctx.AllocRegExcept(d97.Reg)
					ctx.W.EmitMovRegReg(r113, d97.Reg)
					shiftSrc = r113
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d101.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d101.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d101.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d102)
				}
			}
			if d102.Loc == scm.LocReg && d97.Loc == scm.LocReg && d102.Reg == d97.Reg {
				ctx.TransferReg(d97.Reg)
				d97.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d97)
			ctx.FreeDesc(&d101)
			r114 := ctx.AllocReg()
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			if d102.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r114, d102)
			}
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl24)
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d103 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				r115 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r115, d88.Reg)
				ctx.W.EmitAndRegImm32(r115, 63)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d103)
			}
			if d103.Loc == scm.LocReg && d88.Loc == scm.LocReg && d103.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			var d104 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r116 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r116, thisptr.Reg, off)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r116}
				ctx.BindReg(r116, &d104)
			}
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			var d105 scm.JITValueDesc
			if d104.Loc == scm.LocImm {
				d105 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d104.Imm.Int()))))}
			} else {
				r117 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r117, d104.Reg)
				ctx.W.EmitShlRegImm8(r117, 56)
				ctx.W.EmitShrRegImm8(r117, 56)
				d105 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d105)
			}
			ctx.FreeDesc(&d104)
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			if d103.Loc == scm.LocStack || d103.Loc == scm.LocStackPair { ctx.EnsureDesc(&d103) }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			var d106 scm.JITValueDesc
			if d103.Loc == scm.LocImm && d105.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d103.Imm.Int() + d105.Imm.Int())}
			} else if d105.Loc == scm.LocImm && d105.Imm.Int() == 0 {
				r118 := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(r118, d103.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r118}
				ctx.BindReg(r118, &d106)
			} else if d103.Loc == scm.LocImm && d103.Imm.Int() == 0 {
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d105.Reg}
				ctx.BindReg(d105.Reg, &d106)
			} else if d103.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d103.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else if d105.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d103.Reg)
				ctx.W.EmitMovRegReg(scratch, d103.Reg)
				if d105.Imm.Int() >= -2147483648 && d105.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d105.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d105.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else {
				r119 := ctx.AllocRegExcept(d103.Reg, d105.Reg)
				ctx.W.EmitMovRegReg(r119, d103.Reg)
				ctx.W.EmitAddInt64(r119, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d106)
			}
			if d106.Loc == scm.LocReg && d103.Loc == scm.LocReg && d106.Reg == d103.Reg {
				ctx.TransferReg(d103.Reg)
				d103.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d103)
			ctx.FreeDesc(&d105)
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d106.Imm.Int()) > uint64(64))}
			} else {
				r120 := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitCmpRegImm32(d106.Reg, 64)
				ctx.W.EmitSetcc(r120, scm.CcA)
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r120}
				ctx.BindReg(r120, &d107)
			}
			ctx.FreeDesc(&d106)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			if d107.Loc == scm.LocImm {
				if d107.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
			d108 := d93
			if d108.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair { ctx.EnsureDesc(&d108) }
			ctx.EmitStoreToStack(d108, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d107.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
			d109 := d93
			if d109.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			ctx.EmitStoreToStack(d109, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d107)
			ctx.W.MarkLabel(lbl27)
			d97 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d110 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() / 64)}
			} else {
				r121 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r121, d88.Reg)
				ctx.W.EmitShrRegImm8(r121, 6)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r121}
				ctx.BindReg(r121, &d110)
			}
			if d110.Loc == scm.LocReg && d88.Loc == scm.LocReg && d110.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			if d110.Loc == scm.LocStack || d110.Loc == scm.LocStackPair { ctx.EnsureDesc(&d110) }
			var d111 scm.JITValueDesc
			if d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(scratch, d110.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			}
			if d111.Loc == scm.LocReg && d110.Loc == scm.LocReg && d111.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d110)
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			r122 := ctx.AllocReg()
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			if d111.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r122, uint64(d111.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r122, d111.Reg)
				ctx.W.EmitShlRegImm8(r122, 3)
			}
			if d89.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d89.Imm.Int()))
				ctx.W.EmitAddInt64(r122, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r122, d89.Reg)
			}
			r123 := ctx.AllocRegExcept(r122)
			ctx.W.EmitMovRegMem(r123, r122, 0)
			ctx.FreeReg(r122)
			d112 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r123}
			ctx.BindReg(r123, &d112)
			ctx.FreeDesc(&d111)
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d113 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d88.Imm.Int() % 64)}
			} else {
				r124 := ctx.AllocRegExcept(d88.Reg)
				ctx.W.EmitMovRegReg(r124, d88.Reg)
				ctx.W.EmitAndRegImm32(r124, 63)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d113)
			}
			if d113.Loc == scm.LocReg && d88.Loc == scm.LocReg && d113.Reg == d88.Reg {
				ctx.TransferReg(d88.Reg)
				d88.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d88)
			d114 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d113.Loc == scm.LocStack || d113.Loc == scm.LocStackPair { ctx.EnsureDesc(&d113) }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			if d113.Loc == scm.LocStack || d113.Loc == scm.LocStackPair { ctx.EnsureDesc(&d113) }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			if d113.Loc == scm.LocStack || d113.Loc == scm.LocStackPair { ctx.EnsureDesc(&d113) }
			var d115 scm.JITValueDesc
			if d114.Loc == scm.LocImm && d113.Loc == scm.LocImm {
				d115 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d114.Imm.Int() - d113.Imm.Int())}
			} else if d113.Loc == scm.LocImm && d113.Imm.Int() == 0 {
				r125 := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(r125, d114.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d115)
			} else if d114.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d113.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d114.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d113.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else if d113.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d114.Reg)
				ctx.W.EmitMovRegReg(scratch, d114.Reg)
				if d113.Imm.Int() >= -2147483648 && d113.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d113.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d113.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d115)
			} else {
				r126 := ctx.AllocRegExcept(d114.Reg, d113.Reg)
				ctx.W.EmitMovRegReg(r126, d114.Reg)
				ctx.W.EmitSubInt64(r126, d113.Reg)
				d115 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r126}
				ctx.BindReg(r126, &d115)
			}
			if d115.Loc == scm.LocReg && d114.Loc == scm.LocReg && d115.Reg == d114.Reg {
				ctx.TransferReg(d114.Reg)
				d114.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d113)
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d112.Loc == scm.LocImm && d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d112.Imm.Int()) >> uint64(d115.Imm.Int())))}
			} else if d115.Loc == scm.LocImm {
				r127 := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegReg(r127, d112.Reg)
				ctx.W.EmitShrRegImm8(r127, uint8(d115.Imm.Int()))
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d116)
			} else {
				{
					shiftSrc := d112.Reg
					r128 := ctx.AllocRegExcept(d112.Reg)
					ctx.W.EmitMovRegReg(r128, d112.Reg)
					shiftSrc = r128
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d115.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d115.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d115.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d116)
				}
			}
			if d116.Loc == scm.LocReg && d112.Loc == scm.LocReg && d116.Reg == d112.Reg {
				ctx.TransferReg(d112.Reg)
				d112.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			ctx.FreeDesc(&d115)
			if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair { ctx.EnsureDesc(&d93) }
			if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair { ctx.EnsureDesc(&d116) }
			var d117 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d93.Imm.Int() | d116.Imm.Int())}
			} else if d93.Loc == scm.LocImm && d93.Imm.Int() == 0 {
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d116.Reg}
				ctx.BindReg(d116.Reg, &d117)
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				r129 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r129, d93.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d117)
			} else if d93.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d93.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			} else if d116.Loc == scm.LocImm {
				r130 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r130, d93.Reg)
				if d116.Imm.Int() >= -2147483648 && d116.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r130, int32(d116.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d116.Imm.Int()))
					ctx.W.EmitOrInt64(r130, scm.RegR11)
				}
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d117)
			} else {
				r131 := ctx.AllocRegExcept(d93.Reg, d116.Reg)
				ctx.W.EmitMovRegReg(r131, d93.Reg)
				ctx.W.EmitOrInt64(r131, d116.Reg)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r131}
				ctx.BindReg(r131, &d117)
			}
			if d117.Loc == scm.LocReg && d93.Loc == scm.LocReg && d117.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			d118 := d117
			if d118.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			ctx.EmitStoreToStack(d118, 16)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl23)
			d119 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
			ctx.BindReg(r114, &d119)
			ctx.BindReg(r114, &d119)
			if r94 { ctx.UnprotectReg(r95) }
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			var d120 scm.JITValueDesc
			if d119.Loc == scm.LocImm {
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d119.Imm.Int()))))}
			} else {
				r132 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r132, d119.Reg)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d120)
			}
			ctx.FreeDesc(&d119)
			var d121 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r133 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r133, thisptr.Reg, off)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r133}
				ctx.BindReg(r133, &d121)
			}
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() + d121.Imm.Int())}
			} else if d121.Loc == scm.LocImm && d121.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r134, d120.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d122)
			} else if d120.Loc == scm.LocImm && d120.Imm.Int() == 0 {
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d121.Reg}
				ctx.BindReg(d121.Reg, &d122)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(scratch, d120.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r135 := ctx.AllocRegExcept(d120.Reg, d121.Reg)
				ctx.W.EmitMovRegReg(r135, d120.Reg)
				ctx.W.EmitAddInt64(r135, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d122)
			}
			if d122.Loc == scm.LocReg && d120.Loc == scm.LocReg && d122.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d120)
			ctx.FreeDesc(&d121)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			d123 := d41
			_ = d123
			r136 := d41.Loc == scm.LocReg
			r137 := d41.Reg
			if r136 { ctx.ProtectReg(r137) }
			lbl29 := ctx.W.ReserveLabel()
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			var d124 scm.JITValueDesc
			if d123.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d123.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d123.Reg)
				ctx.W.EmitShlRegImm8(r138, 32)
				ctx.W.EmitShrRegImm8(r138, 32)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d124)
			}
			var d125 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d125 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r139, thisptr.Reg, off)
				d125 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r139}
				ctx.BindReg(r139, &d125)
			}
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d125.Imm.Int()))))}
			} else {
				r140 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r140, d125.Reg)
				ctx.W.EmitShlRegImm8(r140, 56)
				ctx.W.EmitShrRegImm8(r140, 56)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r140}
				ctx.BindReg(r140, &d126)
			}
			ctx.FreeDesc(&d125)
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			var d127 scm.JITValueDesc
			if d124.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d124.Imm.Int() * d126.Imm.Int())}
			} else if d124.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d124.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d124.Reg)
				ctx.W.EmitMovRegReg(scratch, d124.Reg)
				if d126.Imm.Int() >= -2147483648 && d126.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d126.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d126.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d127)
			} else {
				r141 := ctx.AllocRegExcept(d124.Reg, d126.Reg)
				ctx.W.EmitMovRegReg(r141, d124.Reg)
				ctx.W.EmitImulInt64(r141, d126.Reg)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d127)
			}
			if d127.Loc == scm.LocReg && d124.Loc == scm.LocReg && d127.Reg == d124.Reg {
				ctx.TransferReg(d124.Reg)
				d124.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d124)
			ctx.FreeDesc(&d126)
			var d128 scm.JITValueDesc
			r142 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r142, uint64(dataPtr))
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142, StackOff: int32(sliceLen)}
				ctx.BindReg(r142, &d128)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r142, thisptr.Reg, off)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r142}
				ctx.BindReg(r142, &d128)
			}
			ctx.BindReg(r142, &d128)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d129 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d129 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() / 64)}
			} else {
				r143 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r143, d127.Reg)
				ctx.W.EmitShrRegImm8(r143, 6)
				d129 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r143}
				ctx.BindReg(r143, &d129)
			}
			if d129.Loc == scm.LocReg && d127.Loc == scm.LocReg && d129.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair { ctx.EnsureDesc(&d129) }
			r144 := ctx.AllocReg()
			if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair { ctx.EnsureDesc(&d129) }
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			if d129.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r144, uint64(d129.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r144, d129.Reg)
				ctx.W.EmitShlRegImm8(r144, 3)
			}
			if d128.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
				ctx.W.EmitAddInt64(r144, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r144, d128.Reg)
			}
			r145 := ctx.AllocRegExcept(r144)
			ctx.W.EmitMovRegMem(r145, r144, 0)
			ctx.FreeReg(r144)
			d130 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r145}
			ctx.BindReg(r145, &d130)
			ctx.FreeDesc(&d129)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d131 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d131 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() % 64)}
			} else {
				r146 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r146, d127.Reg)
				ctx.W.EmitAndRegImm32(r146, 63)
				d131 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d131)
			}
			if d131.Loc == scm.LocReg && d127.Loc == scm.LocReg && d131.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			if d130.Loc == scm.LocStack || d130.Loc == scm.LocStackPair { ctx.EnsureDesc(&d130) }
			if d131.Loc == scm.LocStack || d131.Loc == scm.LocStackPair { ctx.EnsureDesc(&d131) }
			var d132 scm.JITValueDesc
			if d130.Loc == scm.LocImm && d131.Loc == scm.LocImm {
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d130.Imm.Int()) << uint64(d131.Imm.Int())))}
			} else if d131.Loc == scm.LocImm {
				r147 := ctx.AllocRegExcept(d130.Reg)
				ctx.W.EmitMovRegReg(r147, d130.Reg)
				ctx.W.EmitShlRegImm8(r147, uint8(d131.Imm.Int()))
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r147}
				ctx.BindReg(r147, &d132)
			} else {
				{
					shiftSrc := d130.Reg
					r148 := ctx.AllocRegExcept(d130.Reg)
					ctx.W.EmitMovRegReg(r148, d130.Reg)
					shiftSrc = r148
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d131.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d131.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d131.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d132 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d132)
				}
			}
			if d132.Loc == scm.LocReg && d130.Loc == scm.LocReg && d132.Reg == d130.Reg {
				ctx.TransferReg(d130.Reg)
				d130.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d130)
			ctx.FreeDesc(&d131)
			var d133 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r149, thisptr.Reg, off)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r149}
				ctx.BindReg(r149, &d133)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d133.Loc == scm.LocImm {
				if d133.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
			d134 := d132
			if d134.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d134.Loc == scm.LocStack || d134.Loc == scm.LocStackPair { ctx.EnsureDesc(&d134) }
			ctx.EmitStoreToStack(d134, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d133.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
			d135 := d132
			if d135.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			ctx.EmitStoreToStack(d135, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d133)
			ctx.W.MarkLabel(lbl31)
			d136 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d137 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r150 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r150, thisptr.Reg, off)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d137)
			}
			if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair { ctx.EnsureDesc(&d137) }
			if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair { ctx.EnsureDesc(&d137) }
			var d138 scm.JITValueDesc
			if d137.Loc == scm.LocImm {
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d137.Imm.Int()))))}
			} else {
				r151 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r151, d137.Reg)
				ctx.W.EmitShlRegImm8(r151, 56)
				ctx.W.EmitShrRegImm8(r151, 56)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d138)
			}
			ctx.FreeDesc(&d137)
			d139 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d139.Loc == scm.LocStack || d139.Loc == scm.LocStackPair { ctx.EnsureDesc(&d139) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d139.Loc == scm.LocStack || d139.Loc == scm.LocStackPair { ctx.EnsureDesc(&d139) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			var d140 scm.JITValueDesc
			if d139.Loc == scm.LocImm && d138.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d139.Imm.Int() - d138.Imm.Int())}
			} else if d138.Loc == scm.LocImm && d138.Imm.Int() == 0 {
				r152 := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(r152, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d140)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d138.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d139.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d138.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else if d138.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegReg(scratch, d139.Reg)
				if d138.Imm.Int() >= -2147483648 && d138.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d138.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d138.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else {
				r153 := ctx.AllocRegExcept(d139.Reg, d138.Reg)
				ctx.W.EmitMovRegReg(r153, d139.Reg)
				ctx.W.EmitSubInt64(r153, d138.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r153}
				ctx.BindReg(r153, &d140)
			}
			if d140.Loc == scm.LocReg && d139.Loc == scm.LocReg && d140.Reg == d139.Reg {
				ctx.TransferReg(d139.Reg)
				d139.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d138)
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			var d141 scm.JITValueDesc
			if d136.Loc == scm.LocImm && d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d136.Imm.Int()) >> uint64(d140.Imm.Int())))}
			} else if d140.Loc == scm.LocImm {
				r154 := ctx.AllocRegExcept(d136.Reg)
				ctx.W.EmitMovRegReg(r154, d136.Reg)
				ctx.W.EmitShrRegImm8(r154, uint8(d140.Imm.Int()))
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d141)
			} else {
				{
					shiftSrc := d136.Reg
					r155 := ctx.AllocRegExcept(d136.Reg)
					ctx.W.EmitMovRegReg(r155, d136.Reg)
					shiftSrc = r155
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d140.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d140.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d140.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d141)
				}
			}
			if d141.Loc == scm.LocReg && d136.Loc == scm.LocReg && d141.Reg == d136.Reg {
				ctx.TransferReg(d136.Reg)
				d136.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d136)
			ctx.FreeDesc(&d140)
			r156 := ctx.AllocReg()
			if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair { ctx.EnsureDesc(&d141) }
			if d141.Loc == scm.LocStack || d141.Loc == scm.LocStackPair { ctx.EnsureDesc(&d141) }
			if d141.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r156, d141)
			}
			ctx.W.EmitJmp(lbl29)
			ctx.W.MarkLabel(lbl30)
			d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d142 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() % 64)}
			} else {
				r157 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r157, d127.Reg)
				ctx.W.EmitAndRegImm32(r157, 63)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d142)
			}
			if d142.Loc == scm.LocReg && d127.Loc == scm.LocReg && d142.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			var d143 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d143 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d143 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d143)
			}
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			var d144 scm.JITValueDesc
			if d143.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d143.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d143.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d144)
			}
			ctx.FreeDesc(&d143)
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			var d145 scm.JITValueDesc
			if d142.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d142.Imm.Int() + d144.Imm.Int())}
			} else if d144.Loc == scm.LocImm && d144.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(r160, d142.Reg)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d145)
			} else if d142.Loc == scm.LocImm && d142.Imm.Int() == 0 {
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d144.Reg}
				ctx.BindReg(d144.Reg, &d145)
			} else if d142.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d142.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d144.Reg)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d145)
			} else if d144.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d142.Reg)
				ctx.W.EmitMovRegReg(scratch, d142.Reg)
				if d144.Imm.Int() >= -2147483648 && d144.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d144.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d144.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d145)
			} else {
				r161 := ctx.AllocRegExcept(d142.Reg, d144.Reg)
				ctx.W.EmitMovRegReg(r161, d142.Reg)
				ctx.W.EmitAddInt64(r161, d144.Reg)
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d145)
			}
			if d145.Loc == scm.LocReg && d142.Loc == scm.LocReg && d145.Reg == d142.Reg {
				ctx.TransferReg(d142.Reg)
				d142.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d142)
			ctx.FreeDesc(&d144)
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			var d146 scm.JITValueDesc
			if d145.Loc == scm.LocImm {
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d145.Imm.Int()) > uint64(64))}
			} else {
				r162 := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitCmpRegImm32(d145.Reg, 64)
				ctx.W.EmitSetcc(r162, scm.CcA)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r162}
				ctx.BindReg(r162, &d146)
			}
			ctx.FreeDesc(&d145)
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d146.Loc == scm.LocImm {
				if d146.Imm.Bool() {
					ctx.W.EmitJmp(lbl33)
				} else {
			d147 := d132
			if d147.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			ctx.EmitStoreToStack(d147, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d146.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
			d148 := d132
			if d148.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d148.Loc == scm.LocStack || d148.Loc == scm.LocStackPair { ctx.EnsureDesc(&d148) }
			ctx.EmitStoreToStack(d148, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d146)
			ctx.W.MarkLabel(lbl33)
			d136 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d149 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() / 64)}
			} else {
				r163 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r163, d127.Reg)
				ctx.W.EmitShrRegImm8(r163, 6)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r163}
				ctx.BindReg(r163, &d149)
			}
			if d149.Loc == scm.LocReg && d127.Loc == scm.LocReg && d149.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d150 scm.JITValueDesc
			if d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d149.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d149.Reg)
				ctx.W.EmitMovRegReg(scratch, d149.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d150)
			}
			if d150.Loc == scm.LocReg && d149.Loc == scm.LocReg && d150.Reg == d149.Reg {
				ctx.TransferReg(d149.Reg)
				d149.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d149)
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			r164 := ctx.AllocReg()
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			if d128.Loc == scm.LocStack || d128.Loc == scm.LocStackPair { ctx.EnsureDesc(&d128) }
			if d150.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r164, uint64(d150.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r164, d150.Reg)
				ctx.W.EmitShlRegImm8(r164, 3)
			}
			if d128.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
				ctx.W.EmitAddInt64(r164, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r164, d128.Reg)
			}
			r165 := ctx.AllocRegExcept(r164)
			ctx.W.EmitMovRegMem(r165, r164, 0)
			ctx.FreeReg(r164)
			d151 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r165}
			ctx.BindReg(r165, &d151)
			ctx.FreeDesc(&d150)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			var d152 scm.JITValueDesc
			if d127.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() % 64)}
			} else {
				r166 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r166, d127.Reg)
				ctx.W.EmitAndRegImm32(r166, 63)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d152)
			}
			if d152.Loc == scm.LocReg && d127.Loc == scm.LocReg && d152.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d127)
			d153 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() - d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				r167 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r167, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d154)
			} else if d153.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d152.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d153.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else if d152.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(scratch, d153.Reg)
				if d152.Imm.Int() >= -2147483648 && d152.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d152.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d152.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d154)
			} else {
				r168 := ctx.AllocRegExcept(d153.Reg, d152.Reg)
				ctx.W.EmitMovRegReg(r168, d153.Reg)
				ctx.W.EmitSubInt64(r168, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d154)
			}
			if d154.Loc == scm.LocReg && d153.Loc == scm.LocReg && d154.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			if d151.Loc == scm.LocStack || d151.Loc == scm.LocStackPair { ctx.EnsureDesc(&d151) }
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			var d155 scm.JITValueDesc
			if d151.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d151.Imm.Int()) >> uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r169 := ctx.AllocRegExcept(d151.Reg)
				ctx.W.EmitMovRegReg(r169, d151.Reg)
				ctx.W.EmitShrRegImm8(r169, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d155)
			} else {
				{
					shiftSrc := d151.Reg
					r170 := ctx.AllocRegExcept(d151.Reg)
					ctx.W.EmitMovRegReg(r170, d151.Reg)
					shiftSrc = r170
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d154.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d154.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d154.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d155)
				}
			}
			if d155.Loc == scm.LocReg && d151.Loc == scm.LocReg && d155.Reg == d151.Reg {
				ctx.TransferReg(d151.Reg)
				d151.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d151)
			ctx.FreeDesc(&d154)
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair { ctx.EnsureDesc(&d155) }
			var d156 scm.JITValueDesc
			if d132.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d132.Imm.Int() | d155.Imm.Int())}
			} else if d132.Loc == scm.LocImm && d132.Imm.Int() == 0 {
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
				ctx.BindReg(d155.Reg, &d156)
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r171 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(r171, d132.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d156)
			} else if d132.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d132.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d155.Loc == scm.LocImm {
				r172 := ctx.AllocRegExcept(d132.Reg)
				ctx.W.EmitMovRegReg(r172, d132.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r172, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitOrInt64(r172, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d156)
			} else {
				r173 := ctx.AllocRegExcept(d132.Reg, d155.Reg)
				ctx.W.EmitMovRegReg(r173, d132.Reg)
				ctx.W.EmitOrInt64(r173, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r173}
				ctx.BindReg(r173, &d156)
			}
			if d156.Loc == scm.LocReg && d132.Loc == scm.LocReg && d156.Reg == d132.Reg {
				ctx.TransferReg(d132.Reg)
				d132.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d155)
			d157 := d156
			if d157.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d157.Loc == scm.LocStack || d157.Loc == scm.LocStackPair { ctx.EnsureDesc(&d157) }
			ctx.EmitStoreToStack(d157, 24)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl29)
			d158 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
			ctx.BindReg(r156, &d158)
			ctx.BindReg(r156, &d158)
			if r136 { ctx.UnprotectReg(r137) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d159 scm.JITValueDesc
			if d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d158.Imm.Int()))))}
			} else {
				r174 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r174, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d159)
			}
			ctx.FreeDesc(&d158)
			var d160 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r175 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r175, thisptr.Reg, off)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r175}
				ctx.BindReg(r175, &d160)
			}
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d159.Loc == scm.LocStack || d159.Loc == scm.LocStackPair { ctx.EnsureDesc(&d159) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			var d161 scm.JITValueDesc
			if d159.Loc == scm.LocImm && d160.Loc == scm.LocImm {
				d161 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d159.Imm.Int() + d160.Imm.Int())}
			} else if d160.Loc == scm.LocImm && d160.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(r176, d159.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d161)
			} else if d159.Loc == scm.LocImm && d159.Imm.Int() == 0 {
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d160.Reg}
				ctx.BindReg(d160.Reg, &d161)
			} else if d159.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d160.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d159.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else if d160.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitMovRegReg(scratch, d159.Reg)
				if d160.Imm.Int() >= -2147483648 && d160.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d160.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d160.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d161)
			} else {
				r177 := ctx.AllocRegExcept(d159.Reg, d160.Reg)
				ctx.W.EmitMovRegReg(r177, d159.Reg)
				ctx.W.EmitAddInt64(r177, d160.Reg)
				d161 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d161)
			}
			if d161.Loc == scm.LocReg && d159.Loc == scm.LocReg && d161.Reg == d159.Reg {
				ctx.TransferReg(d159.Reg)
				d159.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d159)
			ctx.FreeDesc(&d160)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d161.Loc == scm.LocStack || d161.Loc == scm.LocStackPair { ctx.EnsureDesc(&d161) }
			var d163 scm.JITValueDesc
			if d122.Loc == scm.LocImm && d161.Loc == scm.LocImm {
				d163 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() + d161.Imm.Int())}
			} else if d161.Loc == scm.LocImm && d161.Imm.Int() == 0 {
				r178 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r178, d122.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r178}
				ctx.BindReg(r178, &d163)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d161.Reg}
				ctx.BindReg(d161.Reg, &d163)
			} else if d122.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d161.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d122.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d161.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else if d161.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(scratch, d122.Reg)
				if d161.Imm.Int() >= -2147483648 && d161.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d161.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d161.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d163)
			} else {
				r179 := ctx.AllocRegExcept(d122.Reg, d161.Reg)
				ctx.W.EmitMovRegReg(r179, d122.Reg)
				ctx.W.EmitAddInt64(r179, d161.Reg)
				d163 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d163)
			}
			if d163.Loc == scm.LocReg && d122.Loc == scm.LocReg && d163.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d161)
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			var d165 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r180, fieldAddr)
				ctx.W.EmitMovRegMem64(r181, fieldAddr+8)
				d165 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d165)
				ctx.BindReg(r181, &d165)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r182 := ctx.AllocReg()
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r182, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off+8)
				d165 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
				ctx.BindReg(r182, &d165)
				ctx.BindReg(r183, &d165)
			}
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			r184 := ctx.AllocReg()
			r185 := ctx.AllocRegExcept(r184)
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			if d163.Loc == scm.LocStack || d163.Loc == scm.LocStackPair { ctx.EnsureDesc(&d163) }
			if d165.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r184, uint64(d165.Imm.Int()))
			} else if d165.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r184, d165.Reg)
			} else {
				ctx.W.EmitMovRegReg(r184, d165.Reg)
			}
			if d122.Loc == scm.LocImm {
				if d122.Imm.Int() != 0 {
					if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r184, int32(d122.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
						ctx.W.EmitAddInt64(r184, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r184, d122.Reg)
			}
			if d163.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r185, uint64(d163.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r185, d163.Reg)
			}
			if d122.Loc == scm.LocImm {
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r185, int32(d122.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.W.EmitSubInt64(r185, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r185, d122.Reg)
			}
			d166 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d166)
			ctx.BindReg(r185, &d166)
			ctx.FreeDesc(&d122)
			ctx.FreeDesc(&d163)
			r186 := ctx.AllocReg()
			r187 := ctx.AllocRegExcept(r186)
			d167 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d167)
			ctx.BindReg(r187, &d167)
			d168 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d166}, 2)
			ctx.EmitMovPairToResult(&d168, &d167)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl11)
			var d169 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r188 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r188, thisptr.Reg, off)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r188}
				ctx.BindReg(r188, &d169)
			}
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d170 scm.JITValueDesc
			if d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d169.Imm.Int()))))}
			} else {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r189, d169.Reg)
				ctx.W.EmitShlRegImm8(r189, 32)
				ctx.W.EmitShrRegImm8(r189, 32)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r189}
				ctx.BindReg(r189, &d170)
			}
			ctx.FreeDesc(&d169)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d171 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d41.Imm.Int()) == uint64(d170.Imm.Int()))}
			} else if d170.Loc == scm.LocImm {
				r190 := ctx.AllocRegExcept(d41.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d41.Reg, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitCmpInt64(d41.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r190, scm.CcE)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r190}
				ctx.BindReg(r190, &d171)
			} else if d41.Loc == scm.LocImm {
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d170.Reg)
				ctx.W.EmitSetcc(r191, scm.CcE)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r191}
				ctx.BindReg(r191, &d171)
			} else {
				r192 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitCmpInt64(d41.Reg, d170.Reg)
				ctx.W.EmitSetcc(r192, scm.CcE)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r192}
				ctx.BindReg(r192, &d171)
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d170)
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d171.Loc == scm.LocImm {
				if d171.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d171.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d171)
			ctx.W.MarkLabel(lbl21)
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			d172 := d0
			_ = d172
			r193 := d0.Loc == scm.LocReg
			r194 := d0.Reg
			if r193 { ctx.ProtectReg(r194) }
			lbl37 := ctx.W.ReserveLabel()
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			var d173 scm.JITValueDesc
			if d172.Loc == scm.LocImm {
				d173 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d172.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r195, d172.Reg)
				ctx.W.EmitShlRegImm8(r195, 32)
				ctx.W.EmitShrRegImm8(r195, 32)
				d173 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d173)
			}
			var d174 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r196 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r196, thisptr.Reg, off)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r196}
				ctx.BindReg(r196, &d174)
			}
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			var d175 scm.JITValueDesc
			if d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d174.Imm.Int()))))}
			} else {
				r197 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r197, d174.Reg)
				ctx.W.EmitShlRegImm8(r197, 56)
				ctx.W.EmitShrRegImm8(r197, 56)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r197}
				ctx.BindReg(r197, &d175)
			}
			ctx.FreeDesc(&d174)
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			var d176 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d173.Imm.Int() * d175.Imm.Int())}
			} else if d173.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d173.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(scratch, d173.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d175.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else {
				r198 := ctx.AllocRegExcept(d173.Reg, d175.Reg)
				ctx.W.EmitMovRegReg(r198, d173.Reg)
				ctx.W.EmitImulInt64(r198, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d176)
			}
			if d176.Loc == scm.LocReg && d173.Loc == scm.LocReg && d176.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			ctx.FreeDesc(&d175)
			var d177 scm.JITValueDesc
			r199 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r199, uint64(dataPtr))
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199, StackOff: int32(sliceLen)}
				ctx.BindReg(r199, &d177)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r199, thisptr.Reg, off)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r199}
				ctx.BindReg(r199, &d177)
			}
			ctx.BindReg(r199, &d177)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d178 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() / 64)}
			} else {
				r200 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r200, d176.Reg)
				ctx.W.EmitShrRegImm8(r200, 6)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r200}
				ctx.BindReg(r200, &d178)
			}
			if d178.Loc == scm.LocReg && d176.Loc == scm.LocReg && d178.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			r201 := ctx.AllocReg()
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			if d178.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r201, uint64(d178.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r201, d178.Reg)
				ctx.W.EmitShlRegImm8(r201, 3)
			}
			if d177.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(r201, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r201, d177.Reg)
			}
			r202 := ctx.AllocRegExcept(r201)
			ctx.W.EmitMovRegMem(r202, r201, 0)
			ctx.FreeReg(r201)
			d179 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
			ctx.BindReg(r202, &d179)
			ctx.FreeDesc(&d178)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d180 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r203 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r203, d176.Reg)
				ctx.W.EmitAndRegImm32(r203, 63)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d180)
			}
			if d180.Loc == scm.LocReg && d176.Loc == scm.LocReg && d180.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			var d181 scm.JITValueDesc
			if d179.Loc == scm.LocImm && d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d179.Imm.Int()) << uint64(d180.Imm.Int())))}
			} else if d180.Loc == scm.LocImm {
				r204 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r204, d179.Reg)
				ctx.W.EmitShlRegImm8(r204, uint8(d180.Imm.Int()))
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d181)
			} else {
				{
					shiftSrc := d179.Reg
					r205 := ctx.AllocRegExcept(d179.Reg)
					ctx.W.EmitMovRegReg(r205, d179.Reg)
					shiftSrc = r205
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d180.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d180.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d180.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d181)
				}
			}
			if d181.Loc == scm.LocReg && d179.Loc == scm.LocReg && d181.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.FreeDesc(&d180)
			var d182 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r206, thisptr.Reg, off)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r206}
				ctx.BindReg(r206, &d182)
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d182.Loc == scm.LocImm {
				if d182.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
			d183 := d181
			if d183.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			ctx.EmitStoreToStack(d183, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d182.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
			d184 := d181
			if d184.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			ctx.EmitStoreToStack(d184, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d182)
			ctx.W.MarkLabel(lbl39)
			d185 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d186 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r207 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r207, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r207}
				ctx.BindReg(r207, &d186)
			}
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d186.Imm.Int()))))}
			} else {
				r208 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r208, d186.Reg)
				ctx.W.EmitShlRegImm8(r208, 56)
				ctx.W.EmitShrRegImm8(r208, 56)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d187)
			}
			ctx.FreeDesc(&d186)
			d188 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair { ctx.EnsureDesc(&d188) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair { ctx.EnsureDesc(&d188) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d189 scm.JITValueDesc
			if d188.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d188.Imm.Int() - d187.Imm.Int())}
			} else if d187.Loc == scm.LocImm && d187.Imm.Int() == 0 {
				r209 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(r209, d188.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d189)
			} else if d188.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d188.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d187.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d189)
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitMovRegReg(scratch, d188.Reg)
				if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d189)
			} else {
				r210 := ctx.AllocRegExcept(d188.Reg, d187.Reg)
				ctx.W.EmitMovRegReg(r210, d188.Reg)
				ctx.W.EmitSubInt64(r210, d187.Reg)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r210}
				ctx.BindReg(r210, &d189)
			}
			if d189.Loc == scm.LocReg && d188.Loc == scm.LocReg && d189.Reg == d188.Reg {
				ctx.TransferReg(d188.Reg)
				d188.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d187)
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			if d189.Loc == scm.LocStack || d189.Loc == scm.LocStackPair { ctx.EnsureDesc(&d189) }
			var d190 scm.JITValueDesc
			if d185.Loc == scm.LocImm && d189.Loc == scm.LocImm {
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d185.Imm.Int()) >> uint64(d189.Imm.Int())))}
			} else if d189.Loc == scm.LocImm {
				r211 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r211, d185.Reg)
				ctx.W.EmitShrRegImm8(r211, uint8(d189.Imm.Int()))
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d190)
			} else {
				{
					shiftSrc := d185.Reg
					r212 := ctx.AllocRegExcept(d185.Reg)
					ctx.W.EmitMovRegReg(r212, d185.Reg)
					shiftSrc = r212
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d189.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d189.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d189.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d190 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d190)
				}
			}
			if d190.Loc == scm.LocReg && d185.Loc == scm.LocReg && d190.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			ctx.FreeDesc(&d189)
			r213 := ctx.AllocReg()
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			if d190.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r213, d190)
			}
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl38)
			d185 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d191 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d191 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r214 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r214, d176.Reg)
				ctx.W.EmitAndRegImm32(r214, 63)
				d191 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d191)
			}
			if d191.Loc == scm.LocReg && d176.Loc == scm.LocReg && d191.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			var d192 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r215 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r215, thisptr.Reg, off)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r215}
				ctx.BindReg(r215, &d192)
			}
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			var d193 scm.JITValueDesc
			if d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d192.Imm.Int()))))}
			} else {
				r216 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r216, d192.Reg)
				ctx.W.EmitShlRegImm8(r216, 56)
				ctx.W.EmitShrRegImm8(r216, 56)
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d193)
			}
			ctx.FreeDesc(&d192)
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			var d194 scm.JITValueDesc
			if d191.Loc == scm.LocImm && d193.Loc == scm.LocImm {
				d194 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d191.Imm.Int() + d193.Imm.Int())}
			} else if d193.Loc == scm.LocImm && d193.Imm.Int() == 0 {
				r217 := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(r217, d191.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r217}
				ctx.BindReg(r217, &d194)
			} else if d191.Loc == scm.LocImm && d191.Imm.Int() == 0 {
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d193.Reg}
				ctx.BindReg(d193.Reg, &d194)
			} else if d191.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d193.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d191.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d194)
			} else if d193.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d191.Reg)
				ctx.W.EmitMovRegReg(scratch, d191.Reg)
				if d193.Imm.Int() >= -2147483648 && d193.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d193.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d193.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d194)
			} else {
				r218 := ctx.AllocRegExcept(d191.Reg, d193.Reg)
				ctx.W.EmitMovRegReg(r218, d191.Reg)
				ctx.W.EmitAddInt64(r218, d193.Reg)
				d194 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d194)
			}
			if d194.Loc == scm.LocReg && d191.Loc == scm.LocReg && d194.Reg == d191.Reg {
				ctx.TransferReg(d191.Reg)
				d191.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d191)
			ctx.FreeDesc(&d193)
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			var d195 scm.JITValueDesc
			if d194.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d194.Imm.Int()) > uint64(64))}
			} else {
				r219 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitCmpRegImm32(d194.Reg, 64)
				ctx.W.EmitSetcc(r219, scm.CcA)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r219}
				ctx.BindReg(r219, &d195)
			}
			ctx.FreeDesc(&d194)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d195.Loc == scm.LocImm {
				if d195.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
			d196 := d181
			if d196.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d196.Loc == scm.LocStack || d196.Loc == scm.LocStackPair { ctx.EnsureDesc(&d196) }
			ctx.EmitStoreToStack(d196, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d195.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
			d197 := d181
			if d197.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			ctx.EmitStoreToStack(d197, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d195)
			ctx.W.MarkLabel(lbl41)
			d185 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d198 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() / 64)}
			} else {
				r220 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r220, d176.Reg)
				ctx.W.EmitShrRegImm8(r220, 6)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d198)
			}
			if d198.Loc == scm.LocReg && d176.Loc == scm.LocReg && d198.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			if d198.Loc == scm.LocStack || d198.Loc == scm.LocStackPair { ctx.EnsureDesc(&d198) }
			var d199 scm.JITValueDesc
			if d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d198.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegReg(scratch, d198.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			}
			if d199.Loc == scm.LocReg && d198.Loc == scm.LocReg && d199.Reg == d198.Reg {
				ctx.TransferReg(d198.Reg)
				d198.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d198)
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			r221 := ctx.AllocReg()
			if d199.Loc == scm.LocStack || d199.Loc == scm.LocStackPair { ctx.EnsureDesc(&d199) }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			if d199.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r221, uint64(d199.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r221, d199.Reg)
				ctx.W.EmitShlRegImm8(r221, 3)
			}
			if d177.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(r221, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r221, d177.Reg)
			}
			r222 := ctx.AllocRegExcept(r221)
			ctx.W.EmitMovRegMem(r222, r221, 0)
			ctx.FreeReg(r221)
			d200 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r222}
			ctx.BindReg(r222, &d200)
			ctx.FreeDesc(&d199)
			if d176.Loc == scm.LocStack || d176.Loc == scm.LocStackPair { ctx.EnsureDesc(&d176) }
			var d201 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d176.Imm.Int() % 64)}
			} else {
				r223 := ctx.AllocRegExcept(d176.Reg)
				ctx.W.EmitMovRegReg(r223, d176.Reg)
				ctx.W.EmitAndRegImm32(r223, 63)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d201)
			}
			if d201.Loc == scm.LocReg && d176.Loc == scm.LocReg && d201.Reg == d176.Reg {
				ctx.TransferReg(d176.Reg)
				d176.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d176)
			d202 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			var d203 scm.JITValueDesc
			if d202.Loc == scm.LocImm && d201.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() - d201.Imm.Int())}
			} else if d201.Loc == scm.LocImm && d201.Imm.Int() == 0 {
				r224 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r224, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d201.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d201.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d201.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(scratch, d202.Reg)
				if d201.Imm.Int() >= -2147483648 && d201.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d201.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d201.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r225 := ctx.AllocRegExcept(d202.Reg, d201.Reg)
				ctx.W.EmitMovRegReg(r225, d202.Reg)
				ctx.W.EmitSubInt64(r225, d201.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d203)
			}
			if d203.Loc == scm.LocReg && d202.Loc == scm.LocReg && d203.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d201)
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			var d204 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d200.Imm.Int()) >> uint64(d203.Imm.Int())))}
			} else if d203.Loc == scm.LocImm {
				r226 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r226, d200.Reg)
				ctx.W.EmitShrRegImm8(r226, uint8(d203.Imm.Int()))
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d204)
			} else {
				{
					shiftSrc := d200.Reg
					r227 := ctx.AllocRegExcept(d200.Reg)
					ctx.W.EmitMovRegReg(r227, d200.Reg)
					shiftSrc = r227
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d203.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d203.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d203.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d204)
				}
			}
			if d204.Loc == scm.LocReg && d200.Loc == scm.LocReg && d204.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d203)
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			var d205 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() | d204.Imm.Int())}
			} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d204.Reg}
				ctx.BindReg(d204.Reg, &d205)
			} else if d204.Loc == scm.LocImm && d204.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r228, d181.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d205)
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d204.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d181.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d205)
			} else if d204.Loc == scm.LocImm {
				r229 := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(r229, d181.Reg)
				if d204.Imm.Int() >= -2147483648 && d204.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r229, int32(d204.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d204.Imm.Int()))
					ctx.W.EmitOrInt64(r229, scm.RegR11)
				}
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d205)
			} else {
				r230 := ctx.AllocRegExcept(d181.Reg, d204.Reg)
				ctx.W.EmitMovRegReg(r230, d181.Reg)
				ctx.W.EmitOrInt64(r230, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d205)
			}
			if d205.Loc == scm.LocReg && d181.Loc == scm.LocReg && d205.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d204)
			d206 := d205
			if d206.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			ctx.EmitStoreToStack(d206, 32)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl37)
			d207 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
			ctx.BindReg(r213, &d207)
			ctx.BindReg(r213, &d207)
			if r193 { ctx.UnprotectReg(r194) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d207.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d207.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d208)
			}
			ctx.FreeDesc(&d207)
			var d209 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r232 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r232, thisptr.Reg, off)
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r232}
				ctx.BindReg(r232, &d209)
			}
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair { ctx.EnsureDesc(&d209) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair { ctx.EnsureDesc(&d209) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			if d209.Loc == scm.LocStack || d209.Loc == scm.LocStackPair { ctx.EnsureDesc(&d209) }
			var d210 scm.JITValueDesc
			if d208.Loc == scm.LocImm && d209.Loc == scm.LocImm {
				d210 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d208.Imm.Int() + d209.Imm.Int())}
			} else if d209.Loc == scm.LocImm && d209.Imm.Int() == 0 {
				r233 := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(r233, d208.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d210)
			} else if d208.Loc == scm.LocImm && d208.Imm.Int() == 0 {
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d209.Reg}
				ctx.BindReg(d209.Reg, &d210)
			} else if d208.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d209.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d208.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d209.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d210)
			} else if d209.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(scratch, d208.Reg)
				if d209.Imm.Int() >= -2147483648 && d209.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d209.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d209.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d210)
			} else {
				r234 := ctx.AllocRegExcept(d208.Reg, d209.Reg)
				ctx.W.EmitMovRegReg(r234, d208.Reg)
				ctx.W.EmitAddInt64(r234, d209.Reg)
				d210 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d210)
			}
			if d210.Loc == scm.LocReg && d208.Loc == scm.LocReg && d210.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.FreeDesc(&d209)
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			if d210.Loc == scm.LocStack || d210.Loc == scm.LocStackPair { ctx.EnsureDesc(&d210) }
			var d211 scm.JITValueDesc
			if d210.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d210.Imm.Int()))))}
			} else {
				r235 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r235, d210.Reg)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d211)
			}
			ctx.FreeDesc(&d210)
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			var d212 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d82.Imm.Int()))))}
			} else {
				r236 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r236, d82.Reg)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d212)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d211.Loc == scm.LocStack || d211.Loc == scm.LocStackPair { ctx.EnsureDesc(&d211) }
			var d213 scm.JITValueDesc
			if d82.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d82.Imm.Int() + d211.Imm.Int())}
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r237 := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitMovRegReg(r237, d82.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d213)
			} else if d82.Loc == scm.LocImm && d82.Imm.Int() == 0 {
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d211.Reg}
				ctx.BindReg(d211.Reg, &d213)
			} else if d82.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d82.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitMovRegReg(scratch, d82.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r238 := ctx.AllocRegExcept(d82.Reg, d211.Reg)
				ctx.W.EmitMovRegReg(r238, d82.Reg)
				ctx.W.EmitAddInt64(r238, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d213)
			}
			if d213.Loc == scm.LocReg && d82.Loc == scm.LocReg && d213.Reg == d82.Reg {
				ctx.TransferReg(d82.Reg)
				d82.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair { ctx.EnsureDesc(&d213) }
			if d213.Loc == scm.LocStack || d213.Loc == scm.LocStackPair { ctx.EnsureDesc(&d213) }
			var d214 scm.JITValueDesc
			if d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d213.Imm.Int()))))}
			} else {
				r239 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r239, d213.Reg)
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r239}
				ctx.BindReg(r239, &d214)
			}
			ctx.FreeDesc(&d213)
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			r240 := ctx.AllocReg()
			r241 := ctx.AllocRegExcept(r240)
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			if d214.Loc == scm.LocStack || d214.Loc == scm.LocStackPair { ctx.EnsureDesc(&d214) }
			if d165.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r240, uint64(d165.Imm.Int()))
			} else if d165.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r240, d165.Reg)
			} else {
				ctx.W.EmitMovRegReg(r240, d165.Reg)
			}
			if d212.Loc == scm.LocImm {
				if d212.Imm.Int() != 0 {
					if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r240, int32(d212.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
						ctx.W.EmitAddInt64(r240, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r240, d212.Reg)
			}
			if d214.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r241, uint64(d214.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r241, d214.Reg)
			}
			if d212.Loc == scm.LocImm {
				if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r241, int32(d212.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
					ctx.W.EmitSubInt64(r241, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r241, d212.Reg)
			}
			d215 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r240, Reg2: r241}
			ctx.BindReg(r240, &d215)
			ctx.BindReg(r241, &d215)
			ctx.FreeDesc(&d212)
			ctx.FreeDesc(&d214)
			d216 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d216)
			ctx.BindReg(r187, &d216)
			d217 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d215}, 2)
			ctx.EmitMovPairToResult(&d217, &d216)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl20)
			var d218 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r242, thisptr.Reg, off)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r242}
				ctx.BindReg(r242, &d218)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d218.Loc == scm.LocStack || d218.Loc == scm.LocStackPair { ctx.EnsureDesc(&d218) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d218.Loc == scm.LocStack || d218.Loc == scm.LocStackPair { ctx.EnsureDesc(&d218) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d218.Loc == scm.LocStack || d218.Loc == scm.LocStackPair { ctx.EnsureDesc(&d218) }
			var d219 scm.JITValueDesc
			if d82.Loc == scm.LocImm && d218.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d82.Imm.Int()) == uint64(d218.Imm.Int()))}
			} else if d218.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d82.Reg)
				if d218.Imm.Int() >= -2147483648 && d218.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d82.Reg, int32(d218.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d218.Imm.Int()))
					ctx.W.EmitCmpInt64(d82.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r243, scm.CcE)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d219)
			} else if d82.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d82.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d218.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d219)
			} else {
				r245 := ctx.AllocRegExcept(d82.Reg)
				ctx.W.EmitCmpInt64(d82.Reg, d218.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d219)
			}
			ctx.FreeDesc(&d82)
			ctx.FreeDesc(&d218)
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d219.Loc == scm.LocImm {
				if d219.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d219.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl44)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl44)
				ctx.W.EmitJmp(lbl43)
			}
			ctx.FreeDesc(&d219)
			ctx.W.MarkLabel(lbl35)
			d220 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d220)
			ctx.BindReg(r187, &d220)
			ctx.W.EmitMakeNil(d220)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl43)
			d221 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d221)
			ctx.BindReg(r187, &d221)
			ctx.W.EmitMakeNil(d221)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			d222 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r186, Reg2: r187}
			ctx.BindReg(r186, &d222)
			ctx.BindReg(r187, &d222)
			ctx.BindReg(r186, &d222)
			ctx.BindReg(r187, &d222)
			if r2 { ctx.UnprotectReg(r3) }
			d224 := d222
			d223 := ctx.EmitTagEquals(&d224, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d223.Loc == scm.LocImm {
				if d223.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d223.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d223)
			ctx.W.MarkLabel(lbl46)
			d226 := d222
			d225 := ctx.EmitTagEquals(&d226, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			if d225.Loc == scm.LocImm {
				if d225.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d225.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d225)
			ctx.W.MarkLabel(lbl45)
			d227 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d227)
			ctx.BindReg(r1, &d227)
			ctx.W.EmitMakeNil(d227)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl49)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl48)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			d228 := idxInt
			_ = d228
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			lbl51 := ctx.W.ReserveLabel()
			if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair { ctx.EnsureDesc(&d228) }
			if d228.Loc == scm.LocStack || d228.Loc == scm.LocStackPair { ctx.EnsureDesc(&d228) }
			var d229 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d228.Imm.Int()))))}
			} else {
				r248 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r248, d228.Reg)
				ctx.W.EmitShlRegImm8(r248, 32)
				ctx.W.EmitShrRegImm8(r248, 32)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r248}
				ctx.BindReg(r248, &d229)
			}
			var d230 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r249 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r249, thisptr.Reg, off)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r249}
				ctx.BindReg(r249, &d230)
			}
			if d230.Loc == scm.LocStack || d230.Loc == scm.LocStackPair { ctx.EnsureDesc(&d230) }
			if d230.Loc == scm.LocStack || d230.Loc == scm.LocStackPair { ctx.EnsureDesc(&d230) }
			var d231 scm.JITValueDesc
			if d230.Loc == scm.LocImm {
				d231 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d230.Imm.Int()))))}
			} else {
				r250 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r250, d230.Reg)
				ctx.W.EmitShlRegImm8(r250, 56)
				ctx.W.EmitShrRegImm8(r250, 56)
				d231 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r250}
				ctx.BindReg(r250, &d231)
			}
			ctx.FreeDesc(&d230)
			if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair { ctx.EnsureDesc(&d229) }
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair { ctx.EnsureDesc(&d229) }
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair { ctx.EnsureDesc(&d229) }
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			var d232 scm.JITValueDesc
			if d229.Loc == scm.LocImm && d231.Loc == scm.LocImm {
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d229.Imm.Int() * d231.Imm.Int())}
			} else if d229.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d231.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d229.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d231.Reg)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d232)
			} else if d231.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d229.Reg)
				ctx.W.EmitMovRegReg(scratch, d229.Reg)
				if d231.Imm.Int() >= -2147483648 && d231.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d231.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d231.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d232)
			} else {
				r251 := ctx.AllocRegExcept(d229.Reg, d231.Reg)
				ctx.W.EmitMovRegReg(r251, d229.Reg)
				ctx.W.EmitImulInt64(r251, d231.Reg)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r251}
				ctx.BindReg(r251, &d232)
			}
			if d232.Loc == scm.LocReg && d229.Loc == scm.LocReg && d232.Reg == d229.Reg {
				ctx.TransferReg(d229.Reg)
				d229.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d229)
			ctx.FreeDesc(&d231)
			var d233 scm.JITValueDesc
			r252 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r252, uint64(dataPtr))
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252, StackOff: int32(sliceLen)}
				ctx.BindReg(r252, &d233)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.W.EmitMovRegMem(r252, thisptr.Reg, off)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r252}
				ctx.BindReg(r252, &d233)
			}
			ctx.BindReg(r252, &d233)
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d234 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() / 64)}
			} else {
				r253 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r253, d232.Reg)
				ctx.W.EmitShrRegImm8(r253, 6)
				d234 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d234)
			}
			if d234.Loc == scm.LocReg && d232.Loc == scm.LocReg && d234.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			if d234.Loc == scm.LocStack || d234.Loc == scm.LocStackPair { ctx.EnsureDesc(&d234) }
			r254 := ctx.AllocReg()
			if d234.Loc == scm.LocStack || d234.Loc == scm.LocStackPair { ctx.EnsureDesc(&d234) }
			if d233.Loc == scm.LocStack || d233.Loc == scm.LocStackPair { ctx.EnsureDesc(&d233) }
			if d234.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r254, uint64(d234.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r254, d234.Reg)
				ctx.W.EmitShlRegImm8(r254, 3)
			}
			if d233.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d233.Imm.Int()))
				ctx.W.EmitAddInt64(r254, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r254, d233.Reg)
			}
			r255 := ctx.AllocRegExcept(r254)
			ctx.W.EmitMovRegMem(r255, r254, 0)
			ctx.FreeReg(r254)
			d235 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
			ctx.BindReg(r255, &d235)
			ctx.FreeDesc(&d234)
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d236 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() % 64)}
			} else {
				r256 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r256, d232.Reg)
				ctx.W.EmitAndRegImm32(r256, 63)
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r256}
				ctx.BindReg(r256, &d236)
			}
			if d236.Loc == scm.LocReg && d232.Loc == scm.LocReg && d236.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			if d235.Loc == scm.LocStack || d235.Loc == scm.LocStackPair { ctx.EnsureDesc(&d235) }
			if d236.Loc == scm.LocStack || d236.Loc == scm.LocStackPair { ctx.EnsureDesc(&d236) }
			var d237 scm.JITValueDesc
			if d235.Loc == scm.LocImm && d236.Loc == scm.LocImm {
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d235.Imm.Int()) << uint64(d236.Imm.Int())))}
			} else if d236.Loc == scm.LocImm {
				r257 := ctx.AllocRegExcept(d235.Reg)
				ctx.W.EmitMovRegReg(r257, d235.Reg)
				ctx.W.EmitShlRegImm8(r257, uint8(d236.Imm.Int()))
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d237)
			} else {
				{
					shiftSrc := d235.Reg
					r258 := ctx.AllocRegExcept(d235.Reg)
					ctx.W.EmitMovRegReg(r258, d235.Reg)
					shiftSrc = r258
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d236.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d236.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d236.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d237 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d237)
				}
			}
			if d237.Loc == scm.LocReg && d235.Loc == scm.LocReg && d237.Reg == d235.Reg {
				ctx.TransferReg(d235.Reg)
				d235.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d235)
			ctx.FreeDesc(&d236)
			var d238 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r259 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r259, thisptr.Reg, off)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r259}
				ctx.BindReg(r259, &d238)
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d238.Loc == scm.LocImm {
				if d238.Imm.Bool() {
					ctx.W.EmitJmp(lbl52)
				} else {
			d239 := d237
			if d239.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d239.Loc == scm.LocStack || d239.Loc == scm.LocStackPair { ctx.EnsureDesc(&d239) }
			ctx.EmitStoreToStack(d239, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d238.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl54)
			d240 := d237
			if d240.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d240.Loc == scm.LocStack || d240.Loc == scm.LocStackPair { ctx.EnsureDesc(&d240) }
			ctx.EmitStoreToStack(d240, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d238)
			ctx.W.MarkLabel(lbl53)
			d241 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			var d242 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d242 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r260 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r260, thisptr.Reg, off)
				d242 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r260}
				ctx.BindReg(r260, &d242)
			}
			if d242.Loc == scm.LocStack || d242.Loc == scm.LocStackPair { ctx.EnsureDesc(&d242) }
			if d242.Loc == scm.LocStack || d242.Loc == scm.LocStackPair { ctx.EnsureDesc(&d242) }
			var d243 scm.JITValueDesc
			if d242.Loc == scm.LocImm {
				d243 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d242.Imm.Int()))))}
			} else {
				r261 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r261, d242.Reg)
				ctx.W.EmitShlRegImm8(r261, 56)
				ctx.W.EmitShrRegImm8(r261, 56)
				d243 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r261}
				ctx.BindReg(r261, &d243)
			}
			ctx.FreeDesc(&d242)
			d244 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair { ctx.EnsureDesc(&d243) }
			if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair { ctx.EnsureDesc(&d244) }
			if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair { ctx.EnsureDesc(&d243) }
			if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair { ctx.EnsureDesc(&d244) }
			if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair { ctx.EnsureDesc(&d243) }
			var d245 scm.JITValueDesc
			if d244.Loc == scm.LocImm && d243.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d244.Imm.Int() - d243.Imm.Int())}
			} else if d243.Loc == scm.LocImm && d243.Imm.Int() == 0 {
				r262 := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegReg(r262, d244.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r262}
				ctx.BindReg(r262, &d245)
			} else if d244.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d243.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d244.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d243.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d245)
			} else if d243.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegReg(scratch, d244.Reg)
				if d243.Imm.Int() >= -2147483648 && d243.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d243.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d243.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d245)
			} else {
				r263 := ctx.AllocRegExcept(d244.Reg, d243.Reg)
				ctx.W.EmitMovRegReg(r263, d244.Reg)
				ctx.W.EmitSubInt64(r263, d243.Reg)
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d245)
			}
			if d245.Loc == scm.LocReg && d244.Loc == scm.LocReg && d245.Reg == d244.Reg {
				ctx.TransferReg(d244.Reg)
				d244.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d243)
			if d241.Loc == scm.LocStack || d241.Loc == scm.LocStackPair { ctx.EnsureDesc(&d241) }
			if d245.Loc == scm.LocStack || d245.Loc == scm.LocStackPair { ctx.EnsureDesc(&d245) }
			var d246 scm.JITValueDesc
			if d241.Loc == scm.LocImm && d245.Loc == scm.LocImm {
				d246 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d241.Imm.Int()) >> uint64(d245.Imm.Int())))}
			} else if d245.Loc == scm.LocImm {
				r264 := ctx.AllocRegExcept(d241.Reg)
				ctx.W.EmitMovRegReg(r264, d241.Reg)
				ctx.W.EmitShrRegImm8(r264, uint8(d245.Imm.Int()))
				d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r264}
				ctx.BindReg(r264, &d246)
			} else {
				{
					shiftSrc := d241.Reg
					r265 := ctx.AllocRegExcept(d241.Reg)
					ctx.W.EmitMovRegReg(r265, d241.Reg)
					shiftSrc = r265
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d245.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d245.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d245.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d246 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d246)
				}
			}
			if d246.Loc == scm.LocReg && d241.Loc == scm.LocReg && d246.Reg == d241.Reg {
				ctx.TransferReg(d241.Reg)
				d241.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d241)
			ctx.FreeDesc(&d245)
			r266 := ctx.AllocReg()
			if d246.Loc == scm.LocStack || d246.Loc == scm.LocStackPair { ctx.EnsureDesc(&d246) }
			if d246.Loc == scm.LocStack || d246.Loc == scm.LocStackPair { ctx.EnsureDesc(&d246) }
			if d246.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r266, d246)
			}
			ctx.W.EmitJmp(lbl51)
			ctx.W.MarkLabel(lbl52)
			d241 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d247 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() % 64)}
			} else {
				r267 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r267, d232.Reg)
				ctx.W.EmitAndRegImm32(r267, 63)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d247)
			}
			if d247.Loc == scm.LocReg && d232.Loc == scm.LocReg && d247.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			var d248 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d248 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r268 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r268, thisptr.Reg, off)
				d248 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r268}
				ctx.BindReg(r268, &d248)
			}
			if d248.Loc == scm.LocStack || d248.Loc == scm.LocStackPair { ctx.EnsureDesc(&d248) }
			if d248.Loc == scm.LocStack || d248.Loc == scm.LocStackPair { ctx.EnsureDesc(&d248) }
			var d249 scm.JITValueDesc
			if d248.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d248.Imm.Int()))))}
			} else {
				r269 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r269, d248.Reg)
				ctx.W.EmitShlRegImm8(r269, 56)
				ctx.W.EmitShrRegImm8(r269, 56)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d249)
			}
			ctx.FreeDesc(&d248)
			if d247.Loc == scm.LocStack || d247.Loc == scm.LocStackPair { ctx.EnsureDesc(&d247) }
			if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair { ctx.EnsureDesc(&d249) }
			if d247.Loc == scm.LocStack || d247.Loc == scm.LocStackPair { ctx.EnsureDesc(&d247) }
			if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair { ctx.EnsureDesc(&d249) }
			if d247.Loc == scm.LocStack || d247.Loc == scm.LocStackPair { ctx.EnsureDesc(&d247) }
			if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair { ctx.EnsureDesc(&d249) }
			var d250 scm.JITValueDesc
			if d247.Loc == scm.LocImm && d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d247.Imm.Int() + d249.Imm.Int())}
			} else if d249.Loc == scm.LocImm && d249.Imm.Int() == 0 {
				r270 := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegReg(r270, d247.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r270}
				ctx.BindReg(r270, &d250)
			} else if d247.Loc == scm.LocImm && d247.Imm.Int() == 0 {
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d249.Reg}
				ctx.BindReg(d249.Reg, &d250)
			} else if d247.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d249.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d247.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d249.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d250)
			} else if d249.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegReg(scratch, d247.Reg)
				if d249.Imm.Int() >= -2147483648 && d249.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d249.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d249.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d250)
			} else {
				r271 := ctx.AllocRegExcept(d247.Reg, d249.Reg)
				ctx.W.EmitMovRegReg(r271, d247.Reg)
				ctx.W.EmitAddInt64(r271, d249.Reg)
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r271}
				ctx.BindReg(r271, &d250)
			}
			if d250.Loc == scm.LocReg && d247.Loc == scm.LocReg && d250.Reg == d247.Reg {
				ctx.TransferReg(d247.Reg)
				d247.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d247)
			ctx.FreeDesc(&d249)
			if d250.Loc == scm.LocStack || d250.Loc == scm.LocStackPair { ctx.EnsureDesc(&d250) }
			var d251 scm.JITValueDesc
			if d250.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d250.Imm.Int()) > uint64(64))}
			} else {
				r272 := ctx.AllocRegExcept(d250.Reg)
				ctx.W.EmitCmpRegImm32(d250.Reg, 64)
				ctx.W.EmitSetcc(r272, scm.CcA)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r272}
				ctx.BindReg(r272, &d251)
			}
			ctx.FreeDesc(&d250)
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d251.Loc == scm.LocImm {
				if d251.Imm.Bool() {
					ctx.W.EmitJmp(lbl55)
				} else {
			d252 := d237
			if d252.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair { ctx.EnsureDesc(&d252) }
			ctx.EmitStoreToStack(d252, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d251.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
			d253 := d237
			if d253.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d253.Loc == scm.LocStack || d253.Loc == scm.LocStackPair { ctx.EnsureDesc(&d253) }
			ctx.EmitStoreToStack(d253, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
			}
			ctx.FreeDesc(&d251)
			ctx.W.MarkLabel(lbl55)
			d241 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d254 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() / 64)}
			} else {
				r273 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r273, d232.Reg)
				ctx.W.EmitShrRegImm8(r273, 6)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d254)
			}
			if d254.Loc == scm.LocReg && d232.Loc == scm.LocReg && d254.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			if d254.Loc == scm.LocStack || d254.Loc == scm.LocStackPair { ctx.EnsureDesc(&d254) }
			var d255 scm.JITValueDesc
			if d254.Loc == scm.LocImm {
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d254.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(scratch, d254.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d255)
			}
			if d255.Loc == scm.LocReg && d254.Loc == scm.LocReg && d255.Reg == d254.Reg {
				ctx.TransferReg(d254.Reg)
				d254.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d254)
			if d255.Loc == scm.LocStack || d255.Loc == scm.LocStackPair { ctx.EnsureDesc(&d255) }
			r274 := ctx.AllocReg()
			if d255.Loc == scm.LocStack || d255.Loc == scm.LocStackPair { ctx.EnsureDesc(&d255) }
			if d233.Loc == scm.LocStack || d233.Loc == scm.LocStackPair { ctx.EnsureDesc(&d233) }
			if d255.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r274, uint64(d255.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r274, d255.Reg)
				ctx.W.EmitShlRegImm8(r274, 3)
			}
			if d233.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d233.Imm.Int()))
				ctx.W.EmitAddInt64(r274, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r274, d233.Reg)
			}
			r275 := ctx.AllocRegExcept(r274)
			ctx.W.EmitMovRegMem(r275, r274, 0)
			ctx.FreeReg(r274)
			d256 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r275}
			ctx.BindReg(r275, &d256)
			ctx.FreeDesc(&d255)
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d257 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d257 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d232.Imm.Int() % 64)}
			} else {
				r276 := ctx.AllocRegExcept(d232.Reg)
				ctx.W.EmitMovRegReg(r276, d232.Reg)
				ctx.W.EmitAndRegImm32(r276, 63)
				d257 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r276}
				ctx.BindReg(r276, &d257)
			}
			if d257.Loc == scm.LocReg && d232.Loc == scm.LocReg && d257.Reg == d232.Reg {
				ctx.TransferReg(d232.Reg)
				d232.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d232)
			d258 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair { ctx.EnsureDesc(&d257) }
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair { ctx.EnsureDesc(&d257) }
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair { ctx.EnsureDesc(&d257) }
			var d259 scm.JITValueDesc
			if d258.Loc == scm.LocImm && d257.Loc == scm.LocImm {
				d259 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d258.Imm.Int() - d257.Imm.Int())}
			} else if d257.Loc == scm.LocImm && d257.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegReg(r277, d258.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d259)
			} else if d258.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d257.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d258.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d257.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else if d257.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d258.Reg)
				ctx.W.EmitMovRegReg(scratch, d258.Reg)
				if d257.Imm.Int() >= -2147483648 && d257.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d257.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d257.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d259)
			} else {
				r278 := ctx.AllocRegExcept(d258.Reg, d257.Reg)
				ctx.W.EmitMovRegReg(r278, d258.Reg)
				ctx.W.EmitSubInt64(r278, d257.Reg)
				d259 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d259)
			}
			if d259.Loc == scm.LocReg && d258.Loc == scm.LocReg && d259.Reg == d258.Reg {
				ctx.TransferReg(d258.Reg)
				d258.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d257)
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			if d259.Loc == scm.LocStack || d259.Loc == scm.LocStackPair { ctx.EnsureDesc(&d259) }
			var d260 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d259.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d256.Imm.Int()) >> uint64(d259.Imm.Int())))}
			} else if d259.Loc == scm.LocImm {
				r279 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitMovRegReg(r279, d256.Reg)
				ctx.W.EmitShrRegImm8(r279, uint8(d259.Imm.Int()))
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d260)
			} else {
				{
					shiftSrc := d256.Reg
					r280 := ctx.AllocRegExcept(d256.Reg)
					ctx.W.EmitMovRegReg(r280, d256.Reg)
					shiftSrc = r280
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d259.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d259.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d259.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d260)
				}
			}
			if d260.Loc == scm.LocReg && d256.Loc == scm.LocReg && d260.Reg == d256.Reg {
				ctx.TransferReg(d256.Reg)
				d256.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d256)
			ctx.FreeDesc(&d259)
			if d237.Loc == scm.LocStack || d237.Loc == scm.LocStackPair { ctx.EnsureDesc(&d237) }
			if d260.Loc == scm.LocStack || d260.Loc == scm.LocStackPair { ctx.EnsureDesc(&d260) }
			var d261 scm.JITValueDesc
			if d237.Loc == scm.LocImm && d260.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d237.Imm.Int() | d260.Imm.Int())}
			} else if d237.Loc == scm.LocImm && d237.Imm.Int() == 0 {
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d260.Reg}
				ctx.BindReg(d260.Reg, &d261)
			} else if d260.Loc == scm.LocImm && d260.Imm.Int() == 0 {
				r281 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r281, d237.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r281}
				ctx.BindReg(r281, &d261)
			} else if d237.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d260.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d237.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d261)
			} else if d260.Loc == scm.LocImm {
				r282 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r282, d237.Reg)
				if d260.Imm.Int() >= -2147483648 && d260.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r282, int32(d260.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d260.Imm.Int()))
					ctx.W.EmitOrInt64(r282, scm.RegR11)
				}
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d261)
			} else {
				r283 := ctx.AllocRegExcept(d237.Reg, d260.Reg)
				ctx.W.EmitMovRegReg(r283, d237.Reg)
				ctx.W.EmitOrInt64(r283, d260.Reg)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d261)
			}
			if d261.Loc == scm.LocReg && d237.Loc == scm.LocReg && d261.Reg == d237.Reg {
				ctx.TransferReg(d237.Reg)
				d237.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d260)
			d262 := d261
			if d262.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d262.Loc == scm.LocStack || d262.Loc == scm.LocStackPair { ctx.EnsureDesc(&d262) }
			ctx.EmitStoreToStack(d262, 40)
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl51)
			d263 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r266}
			ctx.BindReg(r266, &d263)
			ctx.BindReg(r266, &d263)
			if r246 { ctx.UnprotectReg(r247) }
			ctx.FreeDesc(&idxInt)
			if d263.Loc == scm.LocStack || d263.Loc == scm.LocStackPair { ctx.EnsureDesc(&d263) }
			if d263.Loc == scm.LocStack || d263.Loc == scm.LocStackPair { ctx.EnsureDesc(&d263) }
			var d264 scm.JITValueDesc
			if d263.Loc == scm.LocImm {
				d264 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d263.Imm.Int()))))}
			} else {
				r284 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r284, d263.Reg)
				d264 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r284}
				ctx.BindReg(r284, &d264)
			}
			ctx.FreeDesc(&d263)
			var d265 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d265 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r285 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r285, thisptr.Reg, off)
				d265 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r285}
				ctx.BindReg(r285, &d265)
			}
			if d264.Loc == scm.LocStack || d264.Loc == scm.LocStackPair { ctx.EnsureDesc(&d264) }
			if d265.Loc == scm.LocStack || d265.Loc == scm.LocStackPair { ctx.EnsureDesc(&d265) }
			if d264.Loc == scm.LocStack || d264.Loc == scm.LocStackPair { ctx.EnsureDesc(&d264) }
			if d265.Loc == scm.LocStack || d265.Loc == scm.LocStackPair { ctx.EnsureDesc(&d265) }
			if d264.Loc == scm.LocStack || d264.Loc == scm.LocStackPair { ctx.EnsureDesc(&d264) }
			if d265.Loc == scm.LocStack || d265.Loc == scm.LocStackPair { ctx.EnsureDesc(&d265) }
			var d266 scm.JITValueDesc
			if d264.Loc == scm.LocImm && d265.Loc == scm.LocImm {
				d266 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d264.Imm.Int() + d265.Imm.Int())}
			} else if d265.Loc == scm.LocImm && d265.Imm.Int() == 0 {
				r286 := ctx.AllocRegExcept(d264.Reg)
				ctx.W.EmitMovRegReg(r286, d264.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r286}
				ctx.BindReg(r286, &d266)
			} else if d264.Loc == scm.LocImm && d264.Imm.Int() == 0 {
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d265.Reg}
				ctx.BindReg(d265.Reg, &d266)
			} else if d264.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d265.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d264.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d265.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d266)
			} else if d265.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d264.Reg)
				ctx.W.EmitMovRegReg(scratch, d264.Reg)
				if d265.Imm.Int() >= -2147483648 && d265.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d265.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d265.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d266)
			} else {
				r287 := ctx.AllocRegExcept(d264.Reg, d265.Reg)
				ctx.W.EmitMovRegReg(r287, d264.Reg)
				ctx.W.EmitAddInt64(r287, d265.Reg)
				d266 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r287}
				ctx.BindReg(r287, &d266)
			}
			if d266.Loc == scm.LocReg && d264.Loc == scm.LocReg && d266.Reg == d264.Reg {
				ctx.TransferReg(d264.Reg)
				d264.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d264)
			ctx.FreeDesc(&d265)
			var d267 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r288 := ctx.AllocReg()
				r289 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r288, fieldAddr)
				ctx.W.EmitMovRegMem64(r289, fieldAddr+8)
				d267 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r288, Reg2: r289}
				ctx.BindReg(r288, &d267)
				ctx.BindReg(r289, &d267)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r290 := ctx.AllocReg()
				r291 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r290, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r291, thisptr.Reg, off+8)
				d267 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r290, Reg2: r291}
				ctx.BindReg(r290, &d267)
				ctx.BindReg(r291, &d267)
			}
			var d268 scm.JITValueDesc
			if d267.Loc == scm.LocImm {
				d268 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d267.StackOff))}
			} else {
				if d267.Loc == scm.LocStack || d267.Loc == scm.LocStackPair { ctx.EnsureDesc(&d267) }
				if d267.Loc == scm.LocRegPair {
					d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d267.Reg2}
					ctx.BindReg(d267.Reg2, &d268)
					ctx.BindReg(d267.Reg2, &d268)
				} else if d267.Loc == scm.LocReg {
					d268 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d267.Reg}
					ctx.BindReg(d267.Reg, &d268)
					ctx.BindReg(d267.Reg, &d268)
				} else {
					panic("len on unsupported descriptor location")
				}
			}
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			if d268.Loc == scm.LocStack || d268.Loc == scm.LocStackPair { ctx.EnsureDesc(&d268) }
			var d270 scm.JITValueDesc
			if d266.Loc == scm.LocImm && d268.Loc == scm.LocImm {
				d270 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d266.Imm.Int() >= d268.Imm.Int())}
			} else if d268.Loc == scm.LocImm {
				r292 := ctx.AllocRegExcept(d266.Reg)
				if d268.Imm.Int() >= -2147483648 && d268.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d266.Reg, int32(d268.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d268.Imm.Int()))
					ctx.W.EmitCmpInt64(d266.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r292, scm.CcGE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r292}
				ctx.BindReg(r292, &d270)
			} else if d266.Loc == scm.LocImm {
				r293 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d266.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d268.Reg)
				ctx.W.EmitSetcc(r293, scm.CcGE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r293}
				ctx.BindReg(r293, &d270)
			} else {
				r294 := ctx.AllocRegExcept(d266.Reg)
				ctx.W.EmitCmpInt64(d266.Reg, d268.Reg)
				ctx.W.EmitSetcc(r294, scm.CcGE)
				d270 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r294}
				ctx.BindReg(r294, &d270)
			}
			ctx.FreeDesc(&d268)
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			if d270.Loc == scm.LocImm {
				if d270.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d270.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl59)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d270)
			ctx.W.MarkLabel(lbl58)
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			var d271 scm.JITValueDesc
			if d266.Loc == scm.LocImm {
				d271 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d266.Imm.Int() < 0)}
			} else {
				r295 := ctx.AllocRegExcept(d266.Reg)
				ctx.W.EmitCmpRegImm32(d266.Reg, 0)
				ctx.W.EmitSetcc(r295, scm.CcL)
				d271 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r295}
				ctx.BindReg(r295, &d271)
			}
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d271.Loc == scm.LocImm {
				if d271.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl60)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d271.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl61)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d271)
			ctx.W.MarkLabel(lbl57)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl60)
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			r296 := ctx.AllocReg()
			if d266.Loc == scm.LocStack || d266.Loc == scm.LocStackPair { ctx.EnsureDesc(&d266) }
			if d267.Loc == scm.LocStack || d267.Loc == scm.LocStackPair { ctx.EnsureDesc(&d267) }
			if d266.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r296, uint64(d266.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r296, d266.Reg)
				ctx.W.EmitShlRegImm8(r296, 4)
			}
			if d267.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d267.Imm.Int()))
				ctx.W.EmitAddInt64(r296, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r296, d267.Reg)
			}
			r297 := ctx.AllocRegExcept(r296)
			r298 := ctx.AllocRegExcept(r296, r297)
			ctx.W.EmitMovRegMem(r297, r296, 0)
			ctx.W.EmitMovRegMem(r298, r296, 8)
			ctx.FreeReg(r296)
			d272 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r297, Reg2: r298}
			ctx.BindReg(r297, &d272)
			ctx.BindReg(r298, &d272)
			ctx.FreeDesc(&d266)
			if d222.Loc != scm.LocImm && d222.Type == scm.JITTypeUnknown {
				panic("jit: scm.Scmer.String on unknown dynamic type")
			}
			d273 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d222}, 2)
			ctx.FreeDesc(&d222)
			if d272.Loc == scm.LocStack || d272.Loc == scm.LocStackPair { ctx.EnsureDesc(&d272) }
			if d273.Loc == scm.LocStack || d273.Loc == scm.LocStackPair { ctx.EnsureDesc(&d273) }
			d274 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d272, d273}, 2)
			ctx.FreeDesc(&d272)
			d275 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d275)
			ctx.BindReg(r1, &d275)
			d276 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d274}, 2)
			ctx.EmitMovPairToResult(&d276, &d275)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d277 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d277)
			ctx.BindReg(r1, &d277)
			ctx.EmitMovPairToResult(&d277, &result)
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
