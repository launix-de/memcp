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
			r10 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r10, uint64(dataPtr))
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10, StackOff: int32(sliceLen)}
				ctx.BindReg(r10, &d5)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 0)
				ctx.W.EmitMovRegMem(r10, thisptr.Reg, off)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
				ctx.BindReg(r10, &d5)
			}
			ctx.BindReg(r10, &d5)
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
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
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
			d11 := d9
			if d11.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			ctx.EmitStoreToStack(d11, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d10.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			d12 := d9
			if d12.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			ctx.EmitStoreToStack(d12, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d10)
			ctx.W.MarkLabel(lbl7)
			d13 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			var d14 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r18, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d14)
			}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			var d15 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d14.Imm.Int()))))}
			} else {
				r19 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r19, d14.Reg)
				ctx.W.EmitShlRegImm8(r19, 56)
				ctx.W.EmitShrRegImm8(r19, 56)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d15)
			}
			ctx.FreeDesc(&d14)
			d16 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d17 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() - d15.Imm.Int())}
			} else if d15.Loc == scm.LocImm && d15.Imm.Int() == 0 {
				r20 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r20, d16.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d17)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(scratch, d16.Reg)
				if d15.Imm.Int() >= -2147483648 && d15.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d15.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d17)
			} else {
				r21 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r21, d16.Reg)
				ctx.W.EmitSubInt64(r21, d15.Reg)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d17)
			}
			if d17.Loc == scm.LocReg && d16.Loc == scm.LocReg && d17.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			var d18 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) >> uint64(d17.Imm.Int())))}
			} else if d17.Loc == scm.LocImm {
				r22 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r22, d13.Reg)
				ctx.W.EmitShrRegImm8(r22, uint8(d17.Imm.Int()))
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d18)
			} else {
				{
					shiftSrc := d13.Reg
					r23 := ctx.AllocRegExcept(d13.Reg)
					ctx.W.EmitMovRegReg(r23, d13.Reg)
					shiftSrc = r23
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d17.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d17.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d17.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d18)
				}
			}
			if d18.Loc == scm.LocReg && d13.Loc == scm.LocReg && d18.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d17)
			r24 := ctx.AllocReg()
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d18.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r24, d18.Reg2)
			} else {
				ctx.EmitMovToReg(r24, d18)
			}
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d19 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r25 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r25, d4.Reg)
				ctx.W.EmitAndRegImm32(r25, 63)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d19)
			}
			if d19.Loc == scm.LocReg && d4.Loc == scm.LocReg && d19.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d20)
			}
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d20.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d20.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d21)
			}
			ctx.FreeDesc(&d20)
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			var d22 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() + d21.Imm.Int())}
			} else if d21.Loc == scm.LocImm && d21.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r28, d19.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d22)
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
				ctx.BindReg(d21.Reg, &d22)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(scratch, d19.Reg)
				if d21.Imm.Int() >= -2147483648 && d21.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d21.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d21.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else {
				r29 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r29, d19.Reg)
				ctx.W.EmitAddInt64(r29, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d22)
			}
			if d22.Loc == scm.LocReg && d19.Loc == scm.LocReg && d22.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.FreeDesc(&d21)
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d22.Imm.Int()) > uint64(64))}
			} else {
				r30 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitCmpRegImm32(d22.Reg, 64)
				ctx.W.EmitSetcc(r30, scm.CcA)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
				ctx.BindReg(r30, &d23)
			}
			ctx.FreeDesc(&d22)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d23.Loc == scm.LocImm {
				if d23.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d24 := d9
			if d24.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			ctx.EmitStoreToStack(d24, 0)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d23.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			d25 := d9
			if d25.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			ctx.EmitStoreToStack(d25, 0)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d23)
			ctx.W.MarkLabel(lbl9)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d26 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() / 64)}
			} else {
				r31 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r31, d4.Reg)
				ctx.W.EmitShrRegImm8(r31, 6)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d26)
			}
			if d26.Loc == scm.LocReg && d4.Loc == scm.LocReg && d26.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(scratch, d26.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			r32 := ctx.AllocReg()
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r32, uint64(d27.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r32, d27.Reg)
				ctx.W.EmitShlRegImm8(r32, 3)
			}
			if d5.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
				ctx.W.EmitAddInt64(r32, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r32, d5.Reg)
			}
			r33 := ctx.AllocRegExcept(r32)
			ctx.W.EmitMovRegMem(r33, r32, 0)
			ctx.FreeReg(r32)
			d28 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d28)
			ctx.FreeDesc(&d27)
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d29 scm.JITValueDesc
			if d4.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r34, d4.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d29)
			}
			if d29.Loc == scm.LocReg && d4.Loc == scm.LocReg && d29.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d4)
			d30 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() - d29.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r35 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r35, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d31)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r36 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r36, d30.Reg)
				ctx.W.EmitSubInt64(r36, d29.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d31)
			}
			if d31.Loc == scm.LocReg && d30.Loc == scm.LocReg && d31.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			var d32 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d28.Imm.Int()) >> uint64(d31.Imm.Int())))}
			} else if d31.Loc == scm.LocImm {
				r37 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r37, d28.Reg)
				ctx.W.EmitShrRegImm8(r37, uint8(d31.Imm.Int()))
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d32)
			} else {
				{
					shiftSrc := d28.Reg
					r38 := ctx.AllocRegExcept(d28.Reg)
					ctx.W.EmitMovRegReg(r38, d28.Reg)
					shiftSrc = r38
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d31.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d31.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d31.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d32)
				}
			}
			if d32.Loc == scm.LocReg && d28.Loc == scm.LocReg && d32.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d31)
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d33 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() | d32.Imm.Int())}
			} else if d9.Loc == scm.LocImm && d9.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d33)
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r39 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r39, d9.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d33)
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == scm.LocImm {
				r40 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r40, d9.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r40, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitOrInt64(r40, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d33)
			} else {
				r41 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r41, d9.Reg)
				ctx.W.EmitOrInt64(r41, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d33)
			}
			if d33.Loc == scm.LocReg && d9.Loc == scm.LocReg && d33.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			d34 := d33
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			ctx.EmitStoreToStack(d34, 0)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d35 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			ctx.BindReg(r24, &d35)
			ctx.BindReg(r24, &d35)
			if r3 { ctx.UnprotectReg(r4) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d35.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d36)
			}
			ctx.FreeDesc(&d35)
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 32)
				r43 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r43, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r43}
				ctx.BindReg(r43, &d37)
			}
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d38 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r44, d36.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d38)
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
				ctx.BindReg(d37.Reg, &d38)
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d38)
			} else {
				r45 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r45, d36.Reg)
				ctx.W.EmitAddInt64(r45, d37.Reg)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d38)
			}
			if d38.Loc == scm.LocReg && d36.Loc == scm.LocReg && d38.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d37)
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(int64(d38.Imm.Int()))))}
			} else {
				r46 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r46, d38.Reg)
				ctx.W.EmitShlRegImm8(r46, 32)
				ctx.W.EmitShrRegImm8(r46, 32)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d39)
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 56)
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r47, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r47}
				ctx.BindReg(r47, &d40)
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d40.Loc == scm.LocImm {
				if d40.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d40.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d40)
			ctx.W.MarkLabel(lbl2)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r48 := idxInt.Loc == scm.LocReg
			r49 := idxInt.Reg
			if r48 { ctx.ProtectReg(r49) }
			lbl14 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d41 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r50, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r50, 32)
				ctx.W.EmitShrRegImm8(r50, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d41)
			}
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r51, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d42)
			}
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d42.Reg)
				ctx.W.EmitShlRegImm8(r52, 56)
				ctx.W.EmitShrRegImm8(r52, 56)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d43)
			}
			ctx.FreeDesc(&d42)
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() * d43.Imm.Int())}
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r53 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r53, d41.Reg)
				ctx.W.EmitImulInt64(r53, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d44)
			}
			if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d43)
			var d45 scm.JITValueDesc
			r54 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r54, uint64(dataPtr))
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54, StackOff: int32(sliceLen)}
				ctx.BindReg(r54, &d45)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r54, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r54}
				ctx.BindReg(r54, &d45)
			}
			ctx.BindReg(r54, &d45)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d46 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r55 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r55, d44.Reg)
				ctx.W.EmitShrRegImm8(r55, 6)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d46)
			}
			if d46.Loc == scm.LocReg && d44.Loc == scm.LocReg && d46.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			r56 := ctx.AllocReg()
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d46.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r56, uint64(d46.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r56, d46.Reg)
				ctx.W.EmitShlRegImm8(r56, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r56, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r56, d45.Reg)
			}
			r57 := ctx.AllocRegExcept(r56)
			ctx.W.EmitMovRegMem(r57, r56, 0)
			ctx.FreeReg(r56)
			d47 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r57}
			ctx.BindReg(r57, &d47)
			ctx.FreeDesc(&d46)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d48 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r58 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r58, d44.Reg)
				ctx.W.EmitAndRegImm32(r58, 63)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d48)
			}
			if d48.Loc == scm.LocReg && d44.Loc == scm.LocReg && d48.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			if d48.Loc == scm.LocStack || d48.Loc == scm.LocStackPair { ctx.EnsureDesc(&d48) }
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) << uint64(d48.Imm.Int())))}
			} else if d48.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r59, d47.Reg)
				ctx.W.EmitShlRegImm8(r59, uint8(d48.Imm.Int()))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d49)
			} else {
				{
					shiftSrc := d47.Reg
					r60 := ctx.AllocRegExcept(d47.Reg)
					ctx.W.EmitMovRegReg(r60, d47.Reg)
					shiftSrc = r60
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d48.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d48.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d48.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d49)
				}
			}
			if d49.Loc == scm.LocReg && d47.Loc == scm.LocReg && d49.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d48)
			var d50 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r61, thisptr.Reg, off)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r61}
				ctx.BindReg(r61, &d50)
			}
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			if d50.Loc == scm.LocImm {
				if d50.Imm.Bool() {
					ctx.W.EmitJmp(lbl15)
				} else {
			d51 := d49
			if d51.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d51.Loc == scm.LocStack || d51.Loc == scm.LocStackPair { ctx.EnsureDesc(&d51) }
			ctx.EmitStoreToStack(d51, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d50.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
			d52 := d49
			if d52.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d52.Loc == scm.LocStack || d52.Loc == scm.LocStackPair { ctx.EnsureDesc(&d52) }
			ctx.EmitStoreToStack(d52, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.FreeDesc(&d50)
			ctx.W.MarkLabel(lbl16)
			d53 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d54 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r62 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r62, thisptr.Reg, off)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r62}
				ctx.BindReg(r62, &d54)
			}
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			if d54.Loc == scm.LocStack || d54.Loc == scm.LocStackPair { ctx.EnsureDesc(&d54) }
			var d55 scm.JITValueDesc
			if d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d54.Imm.Int()))))}
			} else {
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r63, d54.Reg)
				ctx.W.EmitShlRegImm8(r63, 56)
				ctx.W.EmitShrRegImm8(r63, 56)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r63}
				ctx.BindReg(r63, &d55)
			}
			ctx.FreeDesc(&d54)
			d56 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			var d57 scm.JITValueDesc
			if d56.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d56.Imm.Int() - d55.Imm.Int())}
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				r64 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r64, d56.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r64}
				ctx.BindReg(r64, &d57)
			} else if d56.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d56.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else if d55.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(scratch, d56.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			} else {
				r65 := ctx.AllocRegExcept(d56.Reg)
				ctx.W.EmitMovRegReg(r65, d56.Reg)
				ctx.W.EmitSubInt64(r65, d55.Reg)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d57)
			}
			if d57.Loc == scm.LocReg && d56.Loc == scm.LocReg && d57.Reg == d56.Reg {
				ctx.TransferReg(d56.Reg)
				d56.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			if d57.Loc == scm.LocStack || d57.Loc == scm.LocStackPair { ctx.EnsureDesc(&d57) }
			var d58 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d57.Loc == scm.LocImm {
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d53.Imm.Int()) >> uint64(d57.Imm.Int())))}
			} else if d57.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r66, d53.Reg)
				ctx.W.EmitShrRegImm8(r66, uint8(d57.Imm.Int()))
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r66}
				ctx.BindReg(r66, &d58)
			} else {
				{
					shiftSrc := d53.Reg
					r67 := ctx.AllocRegExcept(d53.Reg)
					ctx.W.EmitMovRegReg(r67, d53.Reg)
					shiftSrc = r67
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d57.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d57.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d57.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d58 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d58)
				}
			}
			if d58.Loc == scm.LocReg && d53.Loc == scm.LocReg && d58.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.FreeDesc(&d57)
			r68 := ctx.AllocReg()
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			if d58.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r68, d58.Reg2)
			} else {
				ctx.EmitMovToReg(r68, d58)
			}
			ctx.W.EmitJmp(lbl14)
			ctx.W.MarkLabel(lbl15)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d59 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r69 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r69, d44.Reg)
				ctx.W.EmitAndRegImm32(r69, 63)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r69}
				ctx.BindReg(r69, &d59)
			}
			if d59.Loc == scm.LocReg && d44.Loc == scm.LocReg && d59.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			var d60 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r70 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r70, thisptr.Reg, off)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r70}
				ctx.BindReg(r70, &d60)
			}
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d60.Imm.Int()))))}
			} else {
				r71 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r71, d60.Reg)
				ctx.W.EmitShlRegImm8(r71, 56)
				ctx.W.EmitShrRegImm8(r71, 56)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r71}
				ctx.BindReg(r71, &d61)
			}
			ctx.FreeDesc(&d60)
			if d59.Loc == scm.LocStack || d59.Loc == scm.LocStackPair { ctx.EnsureDesc(&d59) }
			if d61.Loc == scm.LocStack || d61.Loc == scm.LocStackPair { ctx.EnsureDesc(&d61) }
			var d62 scm.JITValueDesc
			if d59.Loc == scm.LocImm && d61.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d59.Imm.Int() + d61.Imm.Int())}
			} else if d61.Loc == scm.LocImm && d61.Imm.Int() == 0 {
				r72 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r72, d59.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r72}
				ctx.BindReg(r72, &d62)
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d61.Reg}
				ctx.BindReg(d61.Reg, &d62)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d59.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else if d61.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(scratch, d59.Reg)
				if d61.Imm.Int() >= -2147483648 && d61.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d61.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d62)
			} else {
				r73 := ctx.AllocRegExcept(d59.Reg)
				ctx.W.EmitMovRegReg(r73, d59.Reg)
				ctx.W.EmitAddInt64(r73, d61.Reg)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r73}
				ctx.BindReg(r73, &d62)
			}
			if d62.Loc == scm.LocReg && d59.Loc == scm.LocReg && d62.Reg == d59.Reg {
				ctx.TransferReg(d59.Reg)
				d59.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d59)
			ctx.FreeDesc(&d61)
			if d62.Loc == scm.LocStack || d62.Loc == scm.LocStackPair { ctx.EnsureDesc(&d62) }
			var d63 scm.JITValueDesc
			if d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d62.Imm.Int()) > uint64(64))}
			} else {
				r74 := ctx.AllocRegExcept(d62.Reg)
				ctx.W.EmitCmpRegImm32(d62.Reg, 64)
				ctx.W.EmitSetcc(r74, scm.CcA)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r74}
				ctx.BindReg(r74, &d63)
			}
			ctx.FreeDesc(&d62)
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d63.Loc == scm.LocImm {
				if d63.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
			d64 := d49
			if d64.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d64.Loc == scm.LocStack || d64.Loc == scm.LocStackPair { ctx.EnsureDesc(&d64) }
			ctx.EmitStoreToStack(d64, 8)
					ctx.W.EmitJmp(lbl16)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d63.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl19)
			d65 := d49
			if d65.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d65.Loc == scm.LocStack || d65.Loc == scm.LocStackPair { ctx.EnsureDesc(&d65) }
			ctx.EmitStoreToStack(d65, 8)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl18)
			}
			ctx.FreeDesc(&d63)
			ctx.W.MarkLabel(lbl18)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d66 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() / 64)}
			} else {
				r75 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r75, d44.Reg)
				ctx.W.EmitShrRegImm8(r75, 6)
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r75}
				ctx.BindReg(r75, &d66)
			}
			if d66.Loc == scm.LocReg && d44.Loc == scm.LocReg && d66.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			if d66.Loc == scm.LocStack || d66.Loc == scm.LocStackPair { ctx.EnsureDesc(&d66) }
			var d67 scm.JITValueDesc
			if d66.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d66.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d66.Reg)
				ctx.W.EmitMovRegReg(scratch, d66.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d67 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d67)
			}
			if d67.Loc == scm.LocReg && d66.Loc == scm.LocReg && d67.Reg == d66.Reg {
				ctx.TransferReg(d66.Reg)
				d66.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d66)
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			r76 := ctx.AllocReg()
			if d67.Loc == scm.LocStack || d67.Loc == scm.LocStackPair { ctx.EnsureDesc(&d67) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			if d67.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r76, uint64(d67.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r76, d67.Reg)
				ctx.W.EmitShlRegImm8(r76, 3)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r76, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r76, d45.Reg)
			}
			r77 := ctx.AllocRegExcept(r76)
			ctx.W.EmitMovRegMem(r77, r76, 0)
			ctx.FreeReg(r76)
			d68 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r77}
			ctx.BindReg(r77, &d68)
			ctx.FreeDesc(&d67)
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d69 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() % 64)}
			} else {
				r78 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r78, d44.Reg)
				ctx.W.EmitAndRegImm32(r78, 63)
				d69 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r78}
				ctx.BindReg(r78, &d69)
			}
			if d69.Loc == scm.LocReg && d44.Loc == scm.LocReg && d69.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			d70 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d69.Loc == scm.LocStack || d69.Loc == scm.LocStackPair { ctx.EnsureDesc(&d69) }
			var d71 scm.JITValueDesc
			if d70.Loc == scm.LocImm && d69.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d70.Imm.Int() - d69.Imm.Int())}
			} else if d69.Loc == scm.LocImm && d69.Imm.Int() == 0 {
				r79 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r79, d70.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r79}
				ctx.BindReg(r79, &d71)
			} else if d70.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d70.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d69.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(scratch, d70.Reg)
				if d69.Imm.Int() >= -2147483648 && d69.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d69.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d69.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d71)
			} else {
				r80 := ctx.AllocRegExcept(d70.Reg)
				ctx.W.EmitMovRegReg(r80, d70.Reg)
				ctx.W.EmitSubInt64(r80, d69.Reg)
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r80}
				ctx.BindReg(r80, &d71)
			}
			if d71.Loc == scm.LocReg && d70.Loc == scm.LocReg && d71.Reg == d70.Reg {
				ctx.TransferReg(d70.Reg)
				d70.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d69)
			if d68.Loc == scm.LocStack || d68.Loc == scm.LocStackPair { ctx.EnsureDesc(&d68) }
			if d71.Loc == scm.LocStack || d71.Loc == scm.LocStackPair { ctx.EnsureDesc(&d71) }
			var d72 scm.JITValueDesc
			if d68.Loc == scm.LocImm && d71.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d68.Imm.Int()) >> uint64(d71.Imm.Int())))}
			} else if d71.Loc == scm.LocImm {
				r81 := ctx.AllocRegExcept(d68.Reg)
				ctx.W.EmitMovRegReg(r81, d68.Reg)
				ctx.W.EmitShrRegImm8(r81, uint8(d71.Imm.Int()))
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r81}
				ctx.BindReg(r81, &d72)
			} else {
				{
					shiftSrc := d68.Reg
					r82 := ctx.AllocRegExcept(d68.Reg)
					ctx.W.EmitMovRegReg(r82, d68.Reg)
					shiftSrc = r82
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d71.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d71.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d71.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d72)
				}
			}
			if d72.Loc == scm.LocReg && d68.Loc == scm.LocReg && d72.Reg == d68.Reg {
				ctx.TransferReg(d68.Reg)
				d68.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.FreeDesc(&d71)
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			if d72.Loc == scm.LocStack || d72.Loc == scm.LocStackPair { ctx.EnsureDesc(&d72) }
			var d73 scm.JITValueDesc
			if d49.Loc == scm.LocImm && d72.Loc == scm.LocImm {
				d73 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() | d72.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d72.Reg}
				ctx.BindReg(d72.Reg, &d73)
			} else if d72.Loc == scm.LocImm && d72.Imm.Int() == 0 {
				r83 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r83, d49.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r83}
				ctx.BindReg(r83, &d73)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d72.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d49.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d73)
			} else if d72.Loc == scm.LocImm {
				r84 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r84, d49.Reg)
				if d72.Imm.Int() >= -2147483648 && d72.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r84, int32(d72.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d72.Imm.Int()))
					ctx.W.EmitOrInt64(r84, scm.RegR11)
				}
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r84}
				ctx.BindReg(r84, &d73)
			} else {
				r85 := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(r85, d49.Reg)
				ctx.W.EmitOrInt64(r85, d72.Reg)
				d73 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r85}
				ctx.BindReg(r85, &d73)
			}
			if d73.Loc == scm.LocReg && d49.Loc == scm.LocReg && d73.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d72)
			d74 := d73
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d74.Loc == scm.LocStack || d74.Loc == scm.LocStackPair { ctx.EnsureDesc(&d74) }
			ctx.EmitStoreToStack(d74, 8)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl14)
			ctx.W.ResolveFixups()
			d75 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r68}
			ctx.BindReg(r68, &d75)
			ctx.BindReg(r68, &d75)
			if r48 { ctx.UnprotectReg(r49) }
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			if d75.Loc == scm.LocStack || d75.Loc == scm.LocStackPair { ctx.EnsureDesc(&d75) }
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d75.Imm.Int()))))}
			} else {
				r86 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r86, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r86}
				ctx.BindReg(r86, &d76)
			}
			ctx.FreeDesc(&d75)
			var d77 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r87 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r87, thisptr.Reg, off)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r87}
				ctx.BindReg(r87, &d77)
			}
			if d76.Loc == scm.LocStack || d76.Loc == scm.LocStackPair { ctx.EnsureDesc(&d76) }
			if d77.Loc == scm.LocStack || d77.Loc == scm.LocStackPair { ctx.EnsureDesc(&d77) }
			var d78 scm.JITValueDesc
			if d76.Loc == scm.LocImm && d77.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d76.Imm.Int() + d77.Imm.Int())}
			} else if d77.Loc == scm.LocImm && d77.Imm.Int() == 0 {
				r88 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(r88, d76.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r88}
				ctx.BindReg(r88, &d78)
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d77.Reg}
				ctx.BindReg(d77.Reg, &d78)
			} else if d76.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d77.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d76.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else if d77.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(scratch, d76.Reg)
				if d77.Imm.Int() >= -2147483648 && d77.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d77.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d78)
			} else {
				r89 := ctx.AllocRegExcept(d76.Reg)
				ctx.W.EmitMovRegReg(r89, d76.Reg)
				ctx.W.EmitAddInt64(r89, d77.Reg)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r89}
				ctx.BindReg(r89, &d78)
			}
			if d78.Loc == scm.LocReg && d76.Loc == scm.LocReg && d78.Reg == d76.Reg {
				ctx.TransferReg(d76.Reg)
				d76.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d76)
			ctx.FreeDesc(&d77)
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			if d78.Loc == scm.LocStack || d78.Loc == scm.LocStackPair { ctx.EnsureDesc(&d78) }
			var d79 scm.JITValueDesc
			if d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d78.Imm.Int()))))}
			} else {
				r90 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r90, d78.Reg)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r90}
				ctx.BindReg(r90, &d79)
			}
			ctx.FreeDesc(&d78)
			var d80 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d80 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r91 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r91, thisptr.Reg, off)
				d80 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r91}
				ctx.BindReg(r91, &d80)
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d80.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
			}
			ctx.FreeDesc(&d80)
			ctx.W.MarkLabel(lbl12)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			r92 := d39.Loc == scm.LocReg
			r93 := d39.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl23 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d81 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d81 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d39.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d81 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d81)
			}
			var d82 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d82 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d82 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d82)
			}
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			if d82.Loc == scm.LocStack || d82.Loc == scm.LocStackPair { ctx.EnsureDesc(&d82) }
			var d83 scm.JITValueDesc
			if d82.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d82.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d82.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d83)
			}
			ctx.FreeDesc(&d82)
			if d81.Loc == scm.LocStack || d81.Loc == scm.LocStackPair { ctx.EnsureDesc(&d81) }
			if d83.Loc == scm.LocStack || d83.Loc == scm.LocStackPair { ctx.EnsureDesc(&d83) }
			var d84 scm.JITValueDesc
			if d81.Loc == scm.LocImm && d83.Loc == scm.LocImm {
				d84 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d81.Imm.Int() * d83.Imm.Int())}
			} else if d81.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d83.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d81.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else if d83.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(scratch, d81.Reg)
				if d83.Imm.Int() >= -2147483648 && d83.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d83.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d83.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d84)
			} else {
				r97 := ctx.AllocRegExcept(d81.Reg)
				ctx.W.EmitMovRegReg(r97, d81.Reg)
				ctx.W.EmitImulInt64(r97, d83.Reg)
				d84 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d84)
			}
			if d84.Loc == scm.LocReg && d81.Loc == scm.LocReg && d84.Reg == d81.Reg {
				ctx.TransferReg(d81.Reg)
				d81.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d81)
			ctx.FreeDesc(&d83)
			var d85 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d85)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d85 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d85)
			}
			ctx.BindReg(r98, &d85)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d86 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d86 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r99, d84.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d86 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d86)
			}
			if d86.Loc == scm.LocReg && d84.Loc == scm.LocReg && d86.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			r100 := ctx.AllocReg()
			if d86.Loc == scm.LocStack || d86.Loc == scm.LocStackPair { ctx.EnsureDesc(&d86) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d86.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d86.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d86.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d85.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d85.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d87 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d87)
			ctx.FreeDesc(&d86)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d88 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d88 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r102, d84.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d88 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d88)
			}
			if d88.Loc == scm.LocReg && d84.Loc == scm.LocReg && d88.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d87.Loc == scm.LocStack || d87.Loc == scm.LocStackPair { ctx.EnsureDesc(&d87) }
			if d88.Loc == scm.LocStack || d88.Loc == scm.LocStackPair { ctx.EnsureDesc(&d88) }
			var d89 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d87.Imm.Int()) << uint64(d88.Imm.Int())))}
			} else if d88.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r103, d87.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d88.Imm.Int()))
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d89)
			} else {
				{
					shiftSrc := d87.Reg
					r104 := ctx.AllocRegExcept(d87.Reg)
					ctx.W.EmitMovRegReg(r104, d87.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d88.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d88.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d88.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d89)
				}
			}
			if d89.Loc == scm.LocReg && d87.Loc == scm.LocReg && d89.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d88)
			var d90 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d90)
			}
			lbl24 := ctx.W.ReserveLabel()
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			if d90.Loc == scm.LocImm {
				if d90.Imm.Bool() {
					ctx.W.EmitJmp(lbl24)
				} else {
			d91 := d89
			if d91.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d91.Loc == scm.LocStack || d91.Loc == scm.LocStackPair { ctx.EnsureDesc(&d91) }
			ctx.EmitStoreToStack(d91, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d90.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
			d92 := d89
			if d92.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d92.Loc == scm.LocStack || d92.Loc == scm.LocStackPair { ctx.EnsureDesc(&d92) }
			ctx.EmitStoreToStack(d92, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl24)
			}
			ctx.FreeDesc(&d90)
			ctx.W.MarkLabel(lbl25)
			d93 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d94 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d94)
			}
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			if d94.Loc == scm.LocStack || d94.Loc == scm.LocStackPair { ctx.EnsureDesc(&d94) }
			var d95 scm.JITValueDesc
			if d94.Loc == scm.LocImm {
				d95 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d94.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d94.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d95 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d95)
			}
			ctx.FreeDesc(&d94)
			d96 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d95.Loc == scm.LocStack || d95.Loc == scm.LocStackPair { ctx.EnsureDesc(&d95) }
			var d97 scm.JITValueDesc
			if d96.Loc == scm.LocImm && d95.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d96.Imm.Int() - d95.Imm.Int())}
			} else if d95.Loc == scm.LocImm && d95.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r108, d96.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d97)
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
				r109 := ctx.AllocRegExcept(d96.Reg)
				ctx.W.EmitMovRegReg(r109, d96.Reg)
				ctx.W.EmitSubInt64(r109, d95.Reg)
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d97)
			}
			if d97.Loc == scm.LocReg && d96.Loc == scm.LocReg && d97.Reg == d96.Reg {
				ctx.TransferReg(d96.Reg)
				d96.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			if d93.Loc == scm.LocStack || d93.Loc == scm.LocStackPair { ctx.EnsureDesc(&d93) }
			if d97.Loc == scm.LocStack || d97.Loc == scm.LocStackPair { ctx.EnsureDesc(&d97) }
			var d98 scm.JITValueDesc
			if d93.Loc == scm.LocImm && d97.Loc == scm.LocImm {
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d93.Imm.Int()) >> uint64(d97.Imm.Int())))}
			} else if d97.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d93.Reg)
				ctx.W.EmitMovRegReg(r110, d93.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d97.Imm.Int()))
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d98)
			} else {
				{
					shiftSrc := d93.Reg
					r111 := ctx.AllocRegExcept(d93.Reg)
					ctx.W.EmitMovRegReg(r111, d93.Reg)
					shiftSrc = r111
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
			if d98.Loc == scm.LocReg && d93.Loc == scm.LocReg && d98.Reg == d93.Reg {
				ctx.TransferReg(d93.Reg)
				d93.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d93)
			ctx.FreeDesc(&d97)
			r112 := ctx.AllocReg()
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			if d98.Loc == scm.LocStack || d98.Loc == scm.LocStackPair { ctx.EnsureDesc(&d98) }
			if d98.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r112, d98.Reg2)
			} else {
				ctx.EmitMovToReg(r112, d98)
			}
			ctx.W.EmitJmp(lbl23)
			ctx.W.MarkLabel(lbl24)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d99 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d99 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r113, d84.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d99 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d99)
			}
			if d99.Loc == scm.LocReg && d84.Loc == scm.LocReg && d99.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			var d100 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d100 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d100 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d100)
			}
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			if d100.Loc == scm.LocStack || d100.Loc == scm.LocStackPair { ctx.EnsureDesc(&d100) }
			var d101 scm.JITValueDesc
			if d100.Loc == scm.LocImm {
				d101 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d100.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d100.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d101 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d101)
			}
			ctx.FreeDesc(&d100)
			if d99.Loc == scm.LocStack || d99.Loc == scm.LocStackPair { ctx.EnsureDesc(&d99) }
			if d101.Loc == scm.LocStack || d101.Loc == scm.LocStackPair { ctx.EnsureDesc(&d101) }
			var d102 scm.JITValueDesc
			if d99.Loc == scm.LocImm && d101.Loc == scm.LocImm {
				d102 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d99.Imm.Int() + d101.Imm.Int())}
			} else if d101.Loc == scm.LocImm && d101.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r116, d99.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d102)
			} else if d99.Loc == scm.LocImm && d99.Imm.Int() == 0 {
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d101.Reg}
				ctx.BindReg(d101.Reg, &d102)
			} else if d99.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d101.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d99.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d102)
			} else if d101.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(scratch, d99.Reg)
				if d101.Imm.Int() >= -2147483648 && d101.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d101.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d101.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d102)
			} else {
				r117 := ctx.AllocRegExcept(d99.Reg)
				ctx.W.EmitMovRegReg(r117, d99.Reg)
				ctx.W.EmitAddInt64(r117, d101.Reg)
				d102 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d102)
			}
			if d102.Loc == scm.LocReg && d99.Loc == scm.LocReg && d102.Reg == d99.Reg {
				ctx.TransferReg(d99.Reg)
				d99.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d99)
			ctx.FreeDesc(&d101)
			if d102.Loc == scm.LocStack || d102.Loc == scm.LocStackPair { ctx.EnsureDesc(&d102) }
			var d103 scm.JITValueDesc
			if d102.Loc == scm.LocImm {
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d102.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitCmpRegImm32(d102.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d103)
			}
			ctx.FreeDesc(&d102)
			lbl27 := ctx.W.ReserveLabel()
			lbl28 := ctx.W.ReserveLabel()
			if d103.Loc == scm.LocImm {
				if d103.Imm.Bool() {
					ctx.W.EmitJmp(lbl27)
				} else {
			d104 := d89
			if d104.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d104.Loc == scm.LocStack || d104.Loc == scm.LocStackPair { ctx.EnsureDesc(&d104) }
			ctx.EmitStoreToStack(d104, 16)
					ctx.W.EmitJmp(lbl25)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d103.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl28)
			d105 := d89
			if d105.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d105.Loc == scm.LocStack || d105.Loc == scm.LocStackPair { ctx.EnsureDesc(&d105) }
			ctx.EmitStoreToStack(d105, 16)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl28)
				ctx.W.EmitJmp(lbl27)
			}
			ctx.FreeDesc(&d103)
			ctx.W.MarkLabel(lbl27)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d106 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r119, d84.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d106)
			}
			if d106.Loc == scm.LocReg && d84.Loc == scm.LocReg && d106.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			if d106.Loc == scm.LocStack || d106.Loc == scm.LocStackPair { ctx.EnsureDesc(&d106) }
			var d107 scm.JITValueDesc
			if d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d106.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d106.Reg)
				ctx.W.EmitMovRegReg(scratch, d106.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d107)
			}
			if d107.Loc == scm.LocReg && d106.Loc == scm.LocReg && d107.Reg == d106.Reg {
				ctx.TransferReg(d106.Reg)
				d106.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d106)
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			r120 := ctx.AllocReg()
			if d107.Loc == scm.LocStack || d107.Loc == scm.LocStackPair { ctx.EnsureDesc(&d107) }
			if d85.Loc == scm.LocStack || d85.Loc == scm.LocStackPair { ctx.EnsureDesc(&d85) }
			if d107.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d107.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d107.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d85.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d85.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d85.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d108 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d108)
			ctx.FreeDesc(&d107)
			if d84.Loc == scm.LocStack || d84.Loc == scm.LocStackPair { ctx.EnsureDesc(&d84) }
			var d109 scm.JITValueDesc
			if d84.Loc == scm.LocImm {
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d84.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d84.Reg)
				ctx.W.EmitMovRegReg(r122, d84.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d109)
			}
			if d109.Loc == scm.LocReg && d84.Loc == scm.LocReg && d109.Reg == d84.Reg {
				ctx.TransferReg(d84.Reg)
				d84.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d84)
			d110 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d109.Loc == scm.LocStack || d109.Loc == scm.LocStackPair { ctx.EnsureDesc(&d109) }
			var d111 scm.JITValueDesc
			if d110.Loc == scm.LocImm && d109.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d110.Imm.Int() - d109.Imm.Int())}
			} else if d109.Loc == scm.LocImm && d109.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r123, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d111)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d109.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d110.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else if d109.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(scratch, d110.Reg)
				if d109.Imm.Int() >= -2147483648 && d109.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d109.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d109.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else {
				r124 := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegReg(r124, d110.Reg)
				ctx.W.EmitSubInt64(r124, d109.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d111)
			}
			if d111.Loc == scm.LocReg && d110.Loc == scm.LocReg && d111.Reg == d110.Reg {
				ctx.TransferReg(d110.Reg)
				d110.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d109)
			if d108.Loc == scm.LocStack || d108.Loc == scm.LocStackPair { ctx.EnsureDesc(&d108) }
			if d111.Loc == scm.LocStack || d111.Loc == scm.LocStackPair { ctx.EnsureDesc(&d111) }
			var d112 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d108.Imm.Int()) >> uint64(d111.Imm.Int())))}
			} else if d111.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r125, d108.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d111.Imm.Int()))
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d112)
			} else {
				{
					shiftSrc := d108.Reg
					r126 := ctx.AllocRegExcept(d108.Reg)
					ctx.W.EmitMovRegReg(r126, d108.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d111.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d111.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d111.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d112)
				}
			}
			if d112.Loc == scm.LocReg && d108.Loc == scm.LocReg && d112.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d111)
			if d89.Loc == scm.LocStack || d89.Loc == scm.LocStackPair { ctx.EnsureDesc(&d89) }
			if d112.Loc == scm.LocStack || d112.Loc == scm.LocStackPair { ctx.EnsureDesc(&d112) }
			var d113 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d112.Loc == scm.LocImm {
				d113 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() | d112.Imm.Int())}
			} else if d89.Loc == scm.LocImm && d89.Imm.Int() == 0 {
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d112.Reg}
				ctx.BindReg(d112.Reg, &d113)
			} else if d112.Loc == scm.LocImm && d112.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r127, d89.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d113)
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d112.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d113)
			} else if d112.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r128, d89.Reg)
				if d112.Imm.Int() >= -2147483648 && d112.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d112.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d112.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d113)
			} else {
				r129 := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(r129, d89.Reg)
				ctx.W.EmitOrInt64(r129, d112.Reg)
				d113 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d113)
			}
			if d113.Loc == scm.LocReg && d89.Loc == scm.LocReg && d113.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d112)
			d114 := d113
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d114.Loc == scm.LocStack || d114.Loc == scm.LocStackPair { ctx.EnsureDesc(&d114) }
			ctx.EmitStoreToStack(d114, 16)
			ctx.W.EmitJmp(lbl25)
			ctx.W.MarkLabel(lbl23)
			ctx.W.ResolveFixups()
			d115 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d115)
			ctx.BindReg(r112, &d115)
			if r92 { ctx.UnprotectReg(r93) }
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			if d115.Loc == scm.LocStack || d115.Loc == scm.LocStackPair { ctx.EnsureDesc(&d115) }
			var d116 scm.JITValueDesc
			if d115.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d115.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d115.Reg)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d116)
			}
			ctx.FreeDesc(&d115)
			var d117 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d117)
			}
			if d116.Loc == scm.LocStack || d116.Loc == scm.LocStackPair { ctx.EnsureDesc(&d116) }
			if d117.Loc == scm.LocStack || d117.Loc == scm.LocStackPair { ctx.EnsureDesc(&d117) }
			var d118 scm.JITValueDesc
			if d116.Loc == scm.LocImm && d117.Loc == scm.LocImm {
				d118 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + d117.Imm.Int())}
			} else if d117.Loc == scm.LocImm && d117.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r132, d116.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d118)
			} else if d116.Loc == scm.LocImm && d116.Imm.Int() == 0 {
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d117.Reg}
				ctx.BindReg(d117.Reg, &d118)
			} else if d116.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d117.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d116.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else if d117.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(scratch, d116.Reg)
				if d117.Imm.Int() >= -2147483648 && d117.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d117.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d117.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d118)
			} else {
				r133 := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(r133, d116.Reg)
				ctx.W.EmitAddInt64(r133, d117.Reg)
				d118 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d118)
			}
			if d118.Loc == scm.LocReg && d116.Loc == scm.LocReg && d118.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.FreeDesc(&d117)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			r134 := d39.Loc == scm.LocReg
			r135 := d39.Reg
			if r134 { ctx.ProtectReg(r135) }
			lbl29 := ctx.W.ReserveLabel()
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d119 scm.JITValueDesc
			if d39.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d39.Imm.Int()))))}
			} else {
				r136 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r136, d39.Reg)
				ctx.W.EmitShlRegImm8(r136, 32)
				ctx.W.EmitShrRegImm8(r136, 32)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r136}
				ctx.BindReg(r136, &d119)
			}
			var d120 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d120 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r137, thisptr.Reg, off)
				d120 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r137}
				ctx.BindReg(r137, &d120)
			}
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			if d120.Loc == scm.LocStack || d120.Loc == scm.LocStackPair { ctx.EnsureDesc(&d120) }
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d120.Imm.Int()))))}
			} else {
				r138 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r138, d120.Reg)
				ctx.W.EmitShlRegImm8(r138, 56)
				ctx.W.EmitShrRegImm8(r138, 56)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r138}
				ctx.BindReg(r138, &d121)
			}
			ctx.FreeDesc(&d120)
			if d119.Loc == scm.LocStack || d119.Loc == scm.LocStackPair { ctx.EnsureDesc(&d119) }
			if d121.Loc == scm.LocStack || d121.Loc == scm.LocStackPair { ctx.EnsureDesc(&d121) }
			var d122 scm.JITValueDesc
			if d119.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d119.Imm.Int() * d121.Imm.Int())}
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d121.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d119.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else if d121.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(scratch, d119.Reg)
				if d121.Imm.Int() >= -2147483648 && d121.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d121.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d121.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d122)
			} else {
				r139 := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegReg(r139, d119.Reg)
				ctx.W.EmitImulInt64(r139, d121.Reg)
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r139}
				ctx.BindReg(r139, &d122)
			}
			if d122.Loc == scm.LocReg && d119.Loc == scm.LocReg && d122.Reg == d119.Reg {
				ctx.TransferReg(d119.Reg)
				d119.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.FreeDesc(&d121)
			var d123 scm.JITValueDesc
			r140 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r140, uint64(dataPtr))
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140, StackOff: int32(sliceLen)}
				ctx.BindReg(r140, &d123)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r140, thisptr.Reg, off)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r140}
				ctx.BindReg(r140, &d123)
			}
			ctx.BindReg(r140, &d123)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d124 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d124 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() / 64)}
			} else {
				r141 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r141, d122.Reg)
				ctx.W.EmitShrRegImm8(r141, 6)
				d124 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r141}
				ctx.BindReg(r141, &d124)
			}
			if d124.Loc == scm.LocReg && d122.Loc == scm.LocReg && d124.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			r142 := ctx.AllocReg()
			if d124.Loc == scm.LocStack || d124.Loc == scm.LocStackPair { ctx.EnsureDesc(&d124) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			if d124.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r142, uint64(d124.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r142, d124.Reg)
				ctx.W.EmitShlRegImm8(r142, 3)
			}
			if d123.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
				ctx.W.EmitAddInt64(r142, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r142, d123.Reg)
			}
			r143 := ctx.AllocRegExcept(r142)
			ctx.W.EmitMovRegMem(r143, r142, 0)
			ctx.FreeReg(r142)
			d125 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r143}
			ctx.BindReg(r143, &d125)
			ctx.FreeDesc(&d124)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d126 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r144 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r144, d122.Reg)
				ctx.W.EmitAndRegImm32(r144, 63)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r144}
				ctx.BindReg(r144, &d126)
			}
			if d126.Loc == scm.LocReg && d122.Loc == scm.LocReg && d126.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d125.Loc == scm.LocStack || d125.Loc == scm.LocStackPair { ctx.EnsureDesc(&d125) }
			if d126.Loc == scm.LocStack || d126.Loc == scm.LocStackPair { ctx.EnsureDesc(&d126) }
			var d127 scm.JITValueDesc
			if d125.Loc == scm.LocImm && d126.Loc == scm.LocImm {
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d125.Imm.Int()) << uint64(d126.Imm.Int())))}
			} else if d126.Loc == scm.LocImm {
				r145 := ctx.AllocRegExcept(d125.Reg)
				ctx.W.EmitMovRegReg(r145, d125.Reg)
				ctx.W.EmitShlRegImm8(r145, uint8(d126.Imm.Int()))
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r145}
				ctx.BindReg(r145, &d127)
			} else {
				{
					shiftSrc := d125.Reg
					r146 := ctx.AllocRegExcept(d125.Reg)
					ctx.W.EmitMovRegReg(r146, d125.Reg)
					shiftSrc = r146
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d126.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d126.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d126.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d127 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d127)
				}
			}
			if d127.Loc == scm.LocReg && d125.Loc == scm.LocReg && d127.Reg == d125.Reg {
				ctx.TransferReg(d125.Reg)
				d125.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d125)
			ctx.FreeDesc(&d126)
			var d128 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d128)
			}
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d128.Loc == scm.LocImm {
				if d128.Imm.Bool() {
					ctx.W.EmitJmp(lbl30)
				} else {
			d129 := d127
			if d129.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d129.Loc == scm.LocStack || d129.Loc == scm.LocStackPair { ctx.EnsureDesc(&d129) }
			ctx.EmitStoreToStack(d129, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d128.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl32)
			d130 := d127
			if d130.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d130.Loc == scm.LocStack || d130.Loc == scm.LocStackPair { ctx.EnsureDesc(&d130) }
			ctx.EmitStoreToStack(d130, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl32)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d128)
			ctx.W.MarkLabel(lbl31)
			d131 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d132 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r148, thisptr.Reg, off)
				d132 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r148}
				ctx.BindReg(r148, &d132)
			}
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			if d132.Loc == scm.LocStack || d132.Loc == scm.LocStackPair { ctx.EnsureDesc(&d132) }
			var d133 scm.JITValueDesc
			if d132.Loc == scm.LocImm {
				d133 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d132.Imm.Int()))))}
			} else {
				r149 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r149, d132.Reg)
				ctx.W.EmitShlRegImm8(r149, 56)
				ctx.W.EmitShrRegImm8(r149, 56)
				d133 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d133)
			}
			ctx.FreeDesc(&d132)
			d134 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d133.Loc == scm.LocStack || d133.Loc == scm.LocStackPair { ctx.EnsureDesc(&d133) }
			var d135 scm.JITValueDesc
			if d134.Loc == scm.LocImm && d133.Loc == scm.LocImm {
				d135 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d134.Imm.Int() - d133.Imm.Int())}
			} else if d133.Loc == scm.LocImm && d133.Imm.Int() == 0 {
				r150 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r150, d134.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r150}
				ctx.BindReg(r150, &d135)
			} else if d134.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d133.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d134.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d133.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else if d133.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(scratch, d134.Reg)
				if d133.Imm.Int() >= -2147483648 && d133.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d133.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d133.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d135)
			} else {
				r151 := ctx.AllocRegExcept(d134.Reg)
				ctx.W.EmitMovRegReg(r151, d134.Reg)
				ctx.W.EmitSubInt64(r151, d133.Reg)
				d135 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d135)
			}
			if d135.Loc == scm.LocReg && d134.Loc == scm.LocReg && d135.Reg == d134.Reg {
				ctx.TransferReg(d134.Reg)
				d134.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d133)
			if d131.Loc == scm.LocStack || d131.Loc == scm.LocStackPair { ctx.EnsureDesc(&d131) }
			if d135.Loc == scm.LocStack || d135.Loc == scm.LocStackPair { ctx.EnsureDesc(&d135) }
			var d136 scm.JITValueDesc
			if d131.Loc == scm.LocImm && d135.Loc == scm.LocImm {
				d136 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d131.Imm.Int()) >> uint64(d135.Imm.Int())))}
			} else if d135.Loc == scm.LocImm {
				r152 := ctx.AllocRegExcept(d131.Reg)
				ctx.W.EmitMovRegReg(r152, d131.Reg)
				ctx.W.EmitShrRegImm8(r152, uint8(d135.Imm.Int()))
				d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r152}
				ctx.BindReg(r152, &d136)
			} else {
				{
					shiftSrc := d131.Reg
					r153 := ctx.AllocRegExcept(d131.Reg)
					ctx.W.EmitMovRegReg(r153, d131.Reg)
					shiftSrc = r153
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d135.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d135.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d135.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d136 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d136)
				}
			}
			if d136.Loc == scm.LocReg && d131.Loc == scm.LocReg && d136.Reg == d131.Reg {
				ctx.TransferReg(d131.Reg)
				d131.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d131)
			ctx.FreeDesc(&d135)
			r154 := ctx.AllocReg()
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d136.Loc == scm.LocStack || d136.Loc == scm.LocStackPair { ctx.EnsureDesc(&d136) }
			if d136.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r154, d136.Reg2)
			} else {
				ctx.EmitMovToReg(r154, d136)
			}
			ctx.W.EmitJmp(lbl29)
			ctx.W.MarkLabel(lbl30)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d137 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r155 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r155, d122.Reg)
				ctx.W.EmitAndRegImm32(r155, 63)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d137)
			}
			if d137.Loc == scm.LocReg && d122.Loc == scm.LocReg && d137.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			var d138 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r156 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r156, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r156}
				ctx.BindReg(r156, &d138)
			}
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			if d138.Loc == scm.LocStack || d138.Loc == scm.LocStackPair { ctx.EnsureDesc(&d138) }
			var d139 scm.JITValueDesc
			if d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
			} else {
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r157, d138.Reg)
				ctx.W.EmitShlRegImm8(r157, 56)
				ctx.W.EmitShrRegImm8(r157, 56)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r157}
				ctx.BindReg(r157, &d139)
			}
			ctx.FreeDesc(&d138)
			if d137.Loc == scm.LocStack || d137.Loc == scm.LocStackPair { ctx.EnsureDesc(&d137) }
			if d139.Loc == scm.LocStack || d139.Loc == scm.LocStackPair { ctx.EnsureDesc(&d139) }
			var d140 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() + d139.Imm.Int())}
			} else if d139.Loc == scm.LocImm && d139.Imm.Int() == 0 {
				r158 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r158, d137.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r158}
				ctx.BindReg(r158, &d140)
			} else if d137.Loc == scm.LocImm && d137.Imm.Int() == 0 {
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d139.Reg}
				ctx.BindReg(d139.Reg, &d140)
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(scratch, d137.Reg)
				if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else {
				r159 := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(r159, d137.Reg)
				ctx.W.EmitAddInt64(r159, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d140)
			}
			if d140.Loc == scm.LocReg && d137.Loc == scm.LocReg && d140.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d139)
			if d140.Loc == scm.LocStack || d140.Loc == scm.LocStackPair { ctx.EnsureDesc(&d140) }
			var d141 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d141 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d140.Imm.Int()) > uint64(64))}
			} else {
				r160 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitCmpRegImm32(d140.Reg, 64)
				ctx.W.EmitSetcc(r160, scm.CcA)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r160}
				ctx.BindReg(r160, &d141)
			}
			ctx.FreeDesc(&d140)
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			if d141.Loc == scm.LocImm {
				if d141.Imm.Bool() {
					ctx.W.EmitJmp(lbl33)
				} else {
			d142 := d127
			if d142.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d142.Loc == scm.LocStack || d142.Loc == scm.LocStackPair { ctx.EnsureDesc(&d142) }
			ctx.EmitStoreToStack(d142, 24)
					ctx.W.EmitJmp(lbl31)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d141.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
			d143 := d127
			if d143.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d143.Loc == scm.LocStack || d143.Loc == scm.LocStackPair { ctx.EnsureDesc(&d143) }
			ctx.EmitStoreToStack(d143, 24)
				ctx.W.EmitJmp(lbl31)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
			}
			ctx.FreeDesc(&d141)
			ctx.W.MarkLabel(lbl33)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d144 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() / 64)}
			} else {
				r161 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r161, d122.Reg)
				ctx.W.EmitShrRegImm8(r161, 6)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d144)
			}
			if d144.Loc == scm.LocReg && d122.Loc == scm.LocReg && d144.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			if d144.Loc == scm.LocStack || d144.Loc == scm.LocStackPair { ctx.EnsureDesc(&d144) }
			var d145 scm.JITValueDesc
			if d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d144.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d144.Reg)
				ctx.W.EmitMovRegReg(scratch, d144.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d145)
			}
			if d145.Loc == scm.LocReg && d144.Loc == scm.LocReg && d145.Reg == d144.Reg {
				ctx.TransferReg(d144.Reg)
				d144.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d144)
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			r162 := ctx.AllocReg()
			if d145.Loc == scm.LocStack || d145.Loc == scm.LocStackPair { ctx.EnsureDesc(&d145) }
			if d123.Loc == scm.LocStack || d123.Loc == scm.LocStackPair { ctx.EnsureDesc(&d123) }
			if d145.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r162, uint64(d145.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r162, d145.Reg)
				ctx.W.EmitShlRegImm8(r162, 3)
			}
			if d123.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d123.Imm.Int()))
				ctx.W.EmitAddInt64(r162, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r162, d123.Reg)
			}
			r163 := ctx.AllocRegExcept(r162)
			ctx.W.EmitMovRegMem(r163, r162, 0)
			ctx.FreeReg(r162)
			d146 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r163}
			ctx.BindReg(r163, &d146)
			ctx.FreeDesc(&d145)
			if d122.Loc == scm.LocStack || d122.Loc == scm.LocStackPair { ctx.EnsureDesc(&d122) }
			var d147 scm.JITValueDesc
			if d122.Loc == scm.LocImm {
				d147 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d122.Imm.Int() % 64)}
			} else {
				r164 := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegReg(r164, d122.Reg)
				ctx.W.EmitAndRegImm32(r164, 63)
				d147 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r164}
				ctx.BindReg(r164, &d147)
			}
			if d147.Loc == scm.LocReg && d122.Loc == scm.LocReg && d147.Reg == d122.Reg {
				ctx.TransferReg(d122.Reg)
				d122.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			d148 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d147.Loc == scm.LocStack || d147.Loc == scm.LocStackPair { ctx.EnsureDesc(&d147) }
			var d149 scm.JITValueDesc
			if d148.Loc == scm.LocImm && d147.Loc == scm.LocImm {
				d149 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d148.Imm.Int() - d147.Imm.Int())}
			} else if d147.Loc == scm.LocImm && d147.Imm.Int() == 0 {
				r165 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r165, d148.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d149)
			} else if d148.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d147.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d148.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else if d147.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(scratch, d148.Reg)
				if d147.Imm.Int() >= -2147483648 && d147.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d147.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d149)
			} else {
				r166 := ctx.AllocRegExcept(d148.Reg)
				ctx.W.EmitMovRegReg(r166, d148.Reg)
				ctx.W.EmitSubInt64(r166, d147.Reg)
				d149 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r166}
				ctx.BindReg(r166, &d149)
			}
			if d149.Loc == scm.LocReg && d148.Loc == scm.LocReg && d149.Reg == d148.Reg {
				ctx.TransferReg(d148.Reg)
				d148.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d147)
			if d146.Loc == scm.LocStack || d146.Loc == scm.LocStackPair { ctx.EnsureDesc(&d146) }
			if d149.Loc == scm.LocStack || d149.Loc == scm.LocStackPair { ctx.EnsureDesc(&d149) }
			var d150 scm.JITValueDesc
			if d146.Loc == scm.LocImm && d149.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d146.Imm.Int()) >> uint64(d149.Imm.Int())))}
			} else if d149.Loc == scm.LocImm {
				r167 := ctx.AllocRegExcept(d146.Reg)
				ctx.W.EmitMovRegReg(r167, d146.Reg)
				ctx.W.EmitShrRegImm8(r167, uint8(d149.Imm.Int()))
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d150)
			} else {
				{
					shiftSrc := d146.Reg
					r168 := ctx.AllocRegExcept(d146.Reg)
					ctx.W.EmitMovRegReg(r168, d146.Reg)
					shiftSrc = r168
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d149.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d149.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d149.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d150)
				}
			}
			if d150.Loc == scm.LocReg && d146.Loc == scm.LocReg && d150.Reg == d146.Reg {
				ctx.TransferReg(d146.Reg)
				d146.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d146)
			ctx.FreeDesc(&d149)
			if d127.Loc == scm.LocStack || d127.Loc == scm.LocStackPair { ctx.EnsureDesc(&d127) }
			if d150.Loc == scm.LocStack || d150.Loc == scm.LocStackPair { ctx.EnsureDesc(&d150) }
			var d151 scm.JITValueDesc
			if d127.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d127.Imm.Int() | d150.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d150.Reg}
				ctx.BindReg(d150.Reg, &d151)
			} else if d150.Loc == scm.LocImm && d150.Imm.Int() == 0 {
				r169 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r169, d127.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d151)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d127.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d151)
			} else if d150.Loc == scm.LocImm {
				r170 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r170, d127.Reg)
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r170, int32(d150.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.W.EmitOrInt64(r170, scm.RegR11)
				}
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r170}
				ctx.BindReg(r170, &d151)
			} else {
				r171 := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegReg(r171, d127.Reg)
				ctx.W.EmitOrInt64(r171, d150.Reg)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d151)
			}
			if d151.Loc == scm.LocReg && d127.Loc == scm.LocReg && d151.Reg == d127.Reg {
				ctx.TransferReg(d127.Reg)
				d127.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			d152 := d151
			if d152.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d152.Loc == scm.LocStack || d152.Loc == scm.LocStackPair { ctx.EnsureDesc(&d152) }
			ctx.EmitStoreToStack(d152, 24)
			ctx.W.EmitJmp(lbl31)
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d153 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r154}
			ctx.BindReg(r154, &d153)
			ctx.BindReg(r154, &d153)
			if r134 { ctx.UnprotectReg(r135) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			if d153.Loc == scm.LocStack || d153.Loc == scm.LocStackPair { ctx.EnsureDesc(&d153) }
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d153.Imm.Int()))))}
			} else {
				r172 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r172, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r172}
				ctx.BindReg(r172, &d154)
			}
			ctx.FreeDesc(&d153)
			var d155 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r173 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r173, thisptr.Reg, off)
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
				ctx.BindReg(r173, &d155)
			}
			if d154.Loc == scm.LocStack || d154.Loc == scm.LocStackPair { ctx.EnsureDesc(&d154) }
			if d155.Loc == scm.LocStack || d155.Loc == scm.LocStackPair { ctx.EnsureDesc(&d155) }
			var d156 scm.JITValueDesc
			if d154.Loc == scm.LocImm && d155.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d154.Imm.Int() + d155.Imm.Int())}
			} else if d155.Loc == scm.LocImm && d155.Imm.Int() == 0 {
				r174 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r174, d154.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d156)
			} else if d154.Loc == scm.LocImm && d154.Imm.Int() == 0 {
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d155.Reg}
				ctx.BindReg(d155.Reg, &d156)
			} else if d154.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d155.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d154.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else if d155.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(scratch, d154.Reg)
				if d155.Imm.Int() >= -2147483648 && d155.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d155.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d155.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d156)
			} else {
				r175 := ctx.AllocRegExcept(d154.Reg)
				ctx.W.EmitMovRegReg(r175, d154.Reg)
				ctx.W.EmitAddInt64(r175, d155.Reg)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d156)
			}
			if d156.Loc == scm.LocReg && d154.Loc == scm.LocReg && d156.Reg == d154.Reg {
				ctx.TransferReg(d154.Reg)
				d154.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d154)
			ctx.FreeDesc(&d155)
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d156.Loc == scm.LocStack || d156.Loc == scm.LocStackPair { ctx.EnsureDesc(&d156) }
			var d158 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d156.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d118.Imm.Int() + d156.Imm.Int())}
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				r176 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r176, d118.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d158)
			} else if d118.Loc == scm.LocImm && d118.Imm.Int() == 0 {
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d156.Reg}
				ctx.BindReg(d156.Reg, &d158)
			} else if d118.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d118.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(scratch, d118.Reg)
				if d156.Imm.Int() >= -2147483648 && d156.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d156.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d156.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d158)
			} else {
				r177 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r177, d118.Reg)
				ctx.W.EmitAddInt64(r177, d156.Reg)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d158)
			}
			if d158.Loc == scm.LocReg && d118.Loc == scm.LocReg && d158.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			var d160 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r178 := ctx.AllocReg()
				r179 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r178, fieldAddr)
				ctx.W.EmitMovRegMem64(r179, fieldAddr+8)
				d160 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r178, Reg2: r179}
				ctx.BindReg(r178, &d160)
				ctx.BindReg(r179, &d160)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r180 := ctx.AllocReg()
				r181 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r180, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r181, thisptr.Reg, off+8)
				d160 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r180, Reg2: r181}
				ctx.BindReg(r180, &d160)
				ctx.BindReg(r181, &d160)
			}
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			r182 := ctx.AllocReg()
			r183 := ctx.AllocRegExcept(r182)
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d118.Loc == scm.LocStack || d118.Loc == scm.LocStackPair { ctx.EnsureDesc(&d118) }
			if d158.Loc == scm.LocStack || d158.Loc == scm.LocStackPair { ctx.EnsureDesc(&d158) }
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r182, uint64(d160.Imm.Int()))
			} else if d160.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r182, d160.Reg)
			} else {
				ctx.W.EmitMovRegReg(r182, d160.Reg)
			}
			if d118.Loc == scm.LocImm {
				if d118.Imm.Int() != 0 {
					if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r182, int32(d118.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
						ctx.W.EmitAddInt64(r182, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r182, d118.Reg)
			}
			if d158.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r183, uint64(d158.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r183, d158.Reg)
			}
			if d118.Loc == scm.LocImm {
				if d118.Imm.Int() >= -2147483648 && d118.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r183, int32(d118.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d118.Imm.Int()))
					ctx.W.EmitSubInt64(r183, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r183, d118.Reg)
			}
			d161 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r182, Reg2: r183}
			ctx.BindReg(r182, &d161)
			ctx.BindReg(r183, &d161)
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d158)
			r184 := ctx.AllocReg()
			r185 := ctx.AllocRegExcept(r184)
			d162 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d162)
			ctx.BindReg(r185, &d162)
			d163 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d161}, 2)
			ctx.EmitMovPairToResult(&d163, &d162)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl11)
			var d164 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r186 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r186, thisptr.Reg, off)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r186}
				ctx.BindReg(r186, &d164)
			}
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			if d164.Loc == scm.LocStack || d164.Loc == scm.LocStackPair { ctx.EnsureDesc(&d164) }
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d164.Imm.Int()))))}
			} else {
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r187, d164.Reg)
				ctx.W.EmitShlRegImm8(r187, 32)
				ctx.W.EmitShrRegImm8(r187, 32)
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r187}
				ctx.BindReg(r187, &d165)
			}
			ctx.FreeDesc(&d164)
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			if d165.Loc == scm.LocStack || d165.Loc == scm.LocStackPair { ctx.EnsureDesc(&d165) }
			var d166 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d165.Loc == scm.LocImm {
				d166 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d39.Imm.Int()) == uint64(d165.Imm.Int()))}
			} else if d165.Loc == scm.LocImm {
				r188 := ctx.AllocRegExcept(d39.Reg)
				if d165.Imm.Int() >= -2147483648 && d165.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d39.Reg, int32(d165.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d165.Imm.Int()))
					ctx.W.EmitCmpInt64(d39.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r188, scm.CcE)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r188}
				ctx.BindReg(r188, &d166)
			} else if d39.Loc == scm.LocImm {
				r189 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d165.Reg)
				ctx.W.EmitSetcc(r189, scm.CcE)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r189}
				ctx.BindReg(r189, &d166)
			} else {
				r190 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpInt64(d39.Reg, d165.Reg)
				ctx.W.EmitSetcc(r190, scm.CcE)
				d166 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r190}
				ctx.BindReg(r190, &d166)
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d165)
			lbl35 := ctx.W.ReserveLabel()
			lbl36 := ctx.W.ReserveLabel()
			if d166.Loc == scm.LocImm {
				if d166.Imm.Bool() {
					ctx.W.EmitJmp(lbl35)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d166.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl36)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl36)
				ctx.W.EmitJmp(lbl35)
			}
			ctx.FreeDesc(&d166)
			ctx.W.MarkLabel(lbl21)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r191 := idxInt.Loc == scm.LocReg
			r192 := idxInt.Reg
			if r191 { ctx.ProtectReg(r192) }
			lbl37 := ctx.W.ReserveLabel()
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d167 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r193 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r193, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r193, 32)
				ctx.W.EmitShrRegImm8(r193, 32)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d167)
			}
			var d168 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d168 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r194 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r194, thisptr.Reg, off)
				d168 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d168)
			}
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			if d168.Loc == scm.LocStack || d168.Loc == scm.LocStackPair { ctx.EnsureDesc(&d168) }
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d168.Imm.Int()))))}
			} else {
				r195 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r195, d168.Reg)
				ctx.W.EmitShlRegImm8(r195, 56)
				ctx.W.EmitShrRegImm8(r195, 56)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d169)
			}
			ctx.FreeDesc(&d168)
			if d167.Loc == scm.LocStack || d167.Loc == scm.LocStackPair { ctx.EnsureDesc(&d167) }
			if d169.Loc == scm.LocStack || d169.Loc == scm.LocStackPair { ctx.EnsureDesc(&d169) }
			var d170 scm.JITValueDesc
			if d167.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d167.Imm.Int() * d169.Imm.Int())}
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d169.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d167.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else if d169.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(scratch, d167.Reg)
				if d169.Imm.Int() >= -2147483648 && d169.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d169.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d169.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d170)
			} else {
				r196 := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegReg(r196, d167.Reg)
				ctx.W.EmitImulInt64(r196, d169.Reg)
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r196}
				ctx.BindReg(r196, &d170)
			}
			if d170.Loc == scm.LocReg && d167.Loc == scm.LocReg && d170.Reg == d167.Reg {
				ctx.TransferReg(d167.Reg)
				d167.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.FreeDesc(&d169)
			var d171 scm.JITValueDesc
			r197 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r197, uint64(dataPtr))
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197, StackOff: int32(sliceLen)}
				ctx.BindReg(r197, &d171)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r197, thisptr.Reg, off)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
				ctx.BindReg(r197, &d171)
			}
			ctx.BindReg(r197, &d171)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d172 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d172 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() / 64)}
			} else {
				r198 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r198, d170.Reg)
				ctx.W.EmitShrRegImm8(r198, 6)
				d172 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d172)
			}
			if d172.Loc == scm.LocReg && d170.Loc == scm.LocReg && d172.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			r199 := ctx.AllocReg()
			if d172.Loc == scm.LocStack || d172.Loc == scm.LocStackPair { ctx.EnsureDesc(&d172) }
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			if d172.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r199, uint64(d172.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r199, d172.Reg)
				ctx.W.EmitShlRegImm8(r199, 3)
			}
			if d171.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
				ctx.W.EmitAddInt64(r199, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r199, d171.Reg)
			}
			r200 := ctx.AllocRegExcept(r199)
			ctx.W.EmitMovRegMem(r200, r199, 0)
			ctx.FreeReg(r199)
			d173 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r200}
			ctx.BindReg(r200, &d173)
			ctx.FreeDesc(&d172)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d174 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				r201 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r201, d170.Reg)
				ctx.W.EmitAndRegImm32(r201, 63)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r201}
				ctx.BindReg(r201, &d174)
			}
			if d174.Loc == scm.LocReg && d170.Loc == scm.LocReg && d174.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			if d173.Loc == scm.LocStack || d173.Loc == scm.LocStackPair { ctx.EnsureDesc(&d173) }
			if d174.Loc == scm.LocStack || d174.Loc == scm.LocStackPair { ctx.EnsureDesc(&d174) }
			var d175 scm.JITValueDesc
			if d173.Loc == scm.LocImm && d174.Loc == scm.LocImm {
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d173.Imm.Int()) << uint64(d174.Imm.Int())))}
			} else if d174.Loc == scm.LocImm {
				r202 := ctx.AllocRegExcept(d173.Reg)
				ctx.W.EmitMovRegReg(r202, d173.Reg)
				ctx.W.EmitShlRegImm8(r202, uint8(d174.Imm.Int()))
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r202}
				ctx.BindReg(r202, &d175)
			} else {
				{
					shiftSrc := d173.Reg
					r203 := ctx.AllocRegExcept(d173.Reg)
					ctx.W.EmitMovRegReg(r203, d173.Reg)
					shiftSrc = r203
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d174.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d174.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d174.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d175 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d175)
				}
			}
			if d175.Loc == scm.LocReg && d173.Loc == scm.LocReg && d175.Reg == d173.Reg {
				ctx.TransferReg(d173.Reg)
				d173.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d173)
			ctx.FreeDesc(&d174)
			var d176 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r204 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r204, thisptr.Reg, off)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r204}
				ctx.BindReg(r204, &d176)
			}
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d176.Loc == scm.LocImm {
				if d176.Imm.Bool() {
					ctx.W.EmitJmp(lbl38)
				} else {
			d177 := d175
			if d177.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d177.Loc == scm.LocStack || d177.Loc == scm.LocStackPair { ctx.EnsureDesc(&d177) }
			ctx.EmitStoreToStack(d177, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d176.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl40)
			d178 := d175
			if d178.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d178.Loc == scm.LocStack || d178.Loc == scm.LocStackPair { ctx.EnsureDesc(&d178) }
			ctx.EmitStoreToStack(d178, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl40)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d176)
			ctx.W.MarkLabel(lbl39)
			d179 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d180 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d180 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r205 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r205, thisptr.Reg, off)
				d180 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r205}
				ctx.BindReg(r205, &d180)
			}
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			if d180.Loc == scm.LocStack || d180.Loc == scm.LocStackPair { ctx.EnsureDesc(&d180) }
			var d181 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d180.Imm.Int()))))}
			} else {
				r206 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r206, d180.Reg)
				ctx.W.EmitShlRegImm8(r206, 56)
				ctx.W.EmitShrRegImm8(r206, 56)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d181)
			}
			ctx.FreeDesc(&d180)
			d182 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d181.Loc == scm.LocStack || d181.Loc == scm.LocStackPair { ctx.EnsureDesc(&d181) }
			var d183 scm.JITValueDesc
			if d182.Loc == scm.LocImm && d181.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d182.Imm.Int() - d181.Imm.Int())}
			} else if d181.Loc == scm.LocImm && d181.Imm.Int() == 0 {
				r207 := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(r207, d182.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r207}
				ctx.BindReg(r207, &d183)
			} else if d182.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d182.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d181.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(scratch, d182.Reg)
				if d181.Imm.Int() >= -2147483648 && d181.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d181.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d181.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d183)
			} else {
				r208 := ctx.AllocRegExcept(d182.Reg)
				ctx.W.EmitMovRegReg(r208, d182.Reg)
				ctx.W.EmitSubInt64(r208, d181.Reg)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r208}
				ctx.BindReg(r208, &d183)
			}
			if d183.Loc == scm.LocReg && d182.Loc == scm.LocReg && d183.Reg == d182.Reg {
				ctx.TransferReg(d182.Reg)
				d182.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			if d179.Loc == scm.LocStack || d179.Loc == scm.LocStackPair { ctx.EnsureDesc(&d179) }
			if d183.Loc == scm.LocStack || d183.Loc == scm.LocStackPair { ctx.EnsureDesc(&d183) }
			var d184 scm.JITValueDesc
			if d179.Loc == scm.LocImm && d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d179.Imm.Int()) >> uint64(d183.Imm.Int())))}
			} else if d183.Loc == scm.LocImm {
				r209 := ctx.AllocRegExcept(d179.Reg)
				ctx.W.EmitMovRegReg(r209, d179.Reg)
				ctx.W.EmitShrRegImm8(r209, uint8(d183.Imm.Int()))
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d184)
			} else {
				{
					shiftSrc := d179.Reg
					r210 := ctx.AllocRegExcept(d179.Reg)
					ctx.W.EmitMovRegReg(r210, d179.Reg)
					shiftSrc = r210
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d183.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d183.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d183.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d184)
				}
			}
			if d184.Loc == scm.LocReg && d179.Loc == scm.LocReg && d184.Reg == d179.Reg {
				ctx.TransferReg(d179.Reg)
				d179.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d179)
			ctx.FreeDesc(&d183)
			r211 := ctx.AllocReg()
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			if d184.Loc == scm.LocStack || d184.Loc == scm.LocStackPair { ctx.EnsureDesc(&d184) }
			if d184.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r211, d184.Reg2)
			} else {
				ctx.EmitMovToReg(r211, d184)
			}
			ctx.W.EmitJmp(lbl37)
			ctx.W.MarkLabel(lbl38)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d185 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d185 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				r212 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r212, d170.Reg)
				ctx.W.EmitAndRegImm32(r212, 63)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d185)
			}
			if d185.Loc == scm.LocReg && d170.Loc == scm.LocReg && d185.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			var d186 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r213 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r213, thisptr.Reg, off)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r213}
				ctx.BindReg(r213, &d186)
			}
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			if d186.Loc == scm.LocStack || d186.Loc == scm.LocStackPair { ctx.EnsureDesc(&d186) }
			var d187 scm.JITValueDesc
			if d186.Loc == scm.LocImm {
				d187 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d186.Imm.Int()))))}
			} else {
				r214 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r214, d186.Reg)
				ctx.W.EmitShlRegImm8(r214, 56)
				ctx.W.EmitShrRegImm8(r214, 56)
				d187 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r214}
				ctx.BindReg(r214, &d187)
			}
			ctx.FreeDesc(&d186)
			if d185.Loc == scm.LocStack || d185.Loc == scm.LocStackPair { ctx.EnsureDesc(&d185) }
			if d187.Loc == scm.LocStack || d187.Loc == scm.LocStackPair { ctx.EnsureDesc(&d187) }
			var d188 scm.JITValueDesc
			if d185.Loc == scm.LocImm && d187.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d185.Imm.Int() + d187.Imm.Int())}
			} else if d187.Loc == scm.LocImm && d187.Imm.Int() == 0 {
				r215 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r215, d185.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d188)
			} else if d185.Loc == scm.LocImm && d185.Imm.Int() == 0 {
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d187.Reg}
				ctx.BindReg(d187.Reg, &d188)
			} else if d185.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d185.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else if d187.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(scratch, d185.Reg)
				if d187.Imm.Int() >= -2147483648 && d187.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d187.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d187.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d188)
			} else {
				r216 := ctx.AllocRegExcept(d185.Reg)
				ctx.W.EmitMovRegReg(r216, d185.Reg)
				ctx.W.EmitAddInt64(r216, d187.Reg)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r216}
				ctx.BindReg(r216, &d188)
			}
			if d188.Loc == scm.LocReg && d185.Loc == scm.LocReg && d188.Reg == d185.Reg {
				ctx.TransferReg(d185.Reg)
				d185.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d185)
			ctx.FreeDesc(&d187)
			if d188.Loc == scm.LocStack || d188.Loc == scm.LocStackPair { ctx.EnsureDesc(&d188) }
			var d189 scm.JITValueDesc
			if d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d188.Imm.Int()) > uint64(64))}
			} else {
				r217 := ctx.AllocRegExcept(d188.Reg)
				ctx.W.EmitCmpRegImm32(d188.Reg, 64)
				ctx.W.EmitSetcc(r217, scm.CcA)
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r217}
				ctx.BindReg(r217, &d189)
			}
			ctx.FreeDesc(&d188)
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			if d189.Loc == scm.LocImm {
				if d189.Imm.Bool() {
					ctx.W.EmitJmp(lbl41)
				} else {
			d190 := d175
			if d190.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d190.Loc == scm.LocStack || d190.Loc == scm.LocStackPair { ctx.EnsureDesc(&d190) }
			ctx.EmitStoreToStack(d190, 32)
					ctx.W.EmitJmp(lbl39)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d189.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
			d191 := d175
			if d191.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d191.Loc == scm.LocStack || d191.Loc == scm.LocStackPair { ctx.EnsureDesc(&d191) }
			ctx.EmitStoreToStack(d191, 32)
				ctx.W.EmitJmp(lbl39)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
			}
			ctx.FreeDesc(&d189)
			ctx.W.MarkLabel(lbl41)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d192 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d192 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() / 64)}
			} else {
				r218 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r218, d170.Reg)
				ctx.W.EmitShrRegImm8(r218, 6)
				d192 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d192)
			}
			if d192.Loc == scm.LocReg && d170.Loc == scm.LocReg && d192.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			if d192.Loc == scm.LocStack || d192.Loc == scm.LocStackPair { ctx.EnsureDesc(&d192) }
			var d193 scm.JITValueDesc
			if d192.Loc == scm.LocImm {
				d193 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d192.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d192.Reg)
				ctx.W.EmitMovRegReg(scratch, d192.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d193 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d193)
			}
			if d193.Loc == scm.LocReg && d192.Loc == scm.LocReg && d193.Reg == d192.Reg {
				ctx.TransferReg(d192.Reg)
				d192.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d192)
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			r219 := ctx.AllocReg()
			if d193.Loc == scm.LocStack || d193.Loc == scm.LocStackPair { ctx.EnsureDesc(&d193) }
			if d171.Loc == scm.LocStack || d171.Loc == scm.LocStackPair { ctx.EnsureDesc(&d171) }
			if d193.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r219, uint64(d193.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r219, d193.Reg)
				ctx.W.EmitShlRegImm8(r219, 3)
			}
			if d171.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d171.Imm.Int()))
				ctx.W.EmitAddInt64(r219, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r219, d171.Reg)
			}
			r220 := ctx.AllocRegExcept(r219)
			ctx.W.EmitMovRegMem(r220, r219, 0)
			ctx.FreeReg(r219)
			d194 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r220}
			ctx.BindReg(r220, &d194)
			ctx.FreeDesc(&d193)
			if d170.Loc == scm.LocStack || d170.Loc == scm.LocStackPair { ctx.EnsureDesc(&d170) }
			var d195 scm.JITValueDesc
			if d170.Loc == scm.LocImm {
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d170.Imm.Int() % 64)}
			} else {
				r221 := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegReg(r221, d170.Reg)
				ctx.W.EmitAndRegImm32(r221, 63)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d195)
			}
			if d195.Loc == scm.LocReg && d170.Loc == scm.LocReg && d195.Reg == d170.Reg {
				ctx.TransferReg(d170.Reg)
				d170.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			d196 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d195.Loc == scm.LocStack || d195.Loc == scm.LocStackPair { ctx.EnsureDesc(&d195) }
			var d197 scm.JITValueDesc
			if d196.Loc == scm.LocImm && d195.Loc == scm.LocImm {
				d197 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d196.Imm.Int() - d195.Imm.Int())}
			} else if d195.Loc == scm.LocImm && d195.Imm.Int() == 0 {
				r222 := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(r222, d196.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r222}
				ctx.BindReg(r222, &d197)
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
				r223 := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegReg(r223, d196.Reg)
				ctx.W.EmitSubInt64(r223, d195.Reg)
				d197 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d197)
			}
			if d197.Loc == scm.LocReg && d196.Loc == scm.LocReg && d197.Reg == d196.Reg {
				ctx.TransferReg(d196.Reg)
				d196.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d195)
			if d194.Loc == scm.LocStack || d194.Loc == scm.LocStackPair { ctx.EnsureDesc(&d194) }
			if d197.Loc == scm.LocStack || d197.Loc == scm.LocStackPair { ctx.EnsureDesc(&d197) }
			var d198 scm.JITValueDesc
			if d194.Loc == scm.LocImm && d197.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d194.Imm.Int()) >> uint64(d197.Imm.Int())))}
			} else if d197.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(r224, d194.Reg)
				ctx.W.EmitShrRegImm8(r224, uint8(d197.Imm.Int()))
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d198)
			} else {
				{
					shiftSrc := d194.Reg
					r225 := ctx.AllocRegExcept(d194.Reg)
					ctx.W.EmitMovRegReg(r225, d194.Reg)
					shiftSrc = r225
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
			if d198.Loc == scm.LocReg && d194.Loc == scm.LocReg && d198.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			ctx.FreeDesc(&d197)
			if d175.Loc == scm.LocStack || d175.Loc == scm.LocStackPair { ctx.EnsureDesc(&d175) }
			if d198.Loc == scm.LocStack || d198.Loc == scm.LocStackPair { ctx.EnsureDesc(&d198) }
			var d199 scm.JITValueDesc
			if d175.Loc == scm.LocImm && d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d175.Imm.Int() | d198.Imm.Int())}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d198.Reg}
				ctx.BindReg(d198.Reg, &d199)
			} else if d198.Loc == scm.LocImm && d198.Imm.Int() == 0 {
				r226 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r226, d175.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d199)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d198.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d175.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d198.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d199)
			} else if d198.Loc == scm.LocImm {
				r227 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r227, d175.Reg)
				if d198.Imm.Int() >= -2147483648 && d198.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r227, int32(d198.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d198.Imm.Int()))
					ctx.W.EmitOrInt64(r227, scm.RegR11)
				}
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r227}
				ctx.BindReg(r227, &d199)
			} else {
				r228 := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegReg(r228, d175.Reg)
				ctx.W.EmitOrInt64(r228, d198.Reg)
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d199)
			}
			if d199.Loc == scm.LocReg && d175.Loc == scm.LocReg && d199.Reg == d175.Reg {
				ctx.TransferReg(d175.Reg)
				d175.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d198)
			d200 := d199
			if d200.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d200.Loc == scm.LocStack || d200.Loc == scm.LocStackPair { ctx.EnsureDesc(&d200) }
			ctx.EmitStoreToStack(d200, 32)
			ctx.W.EmitJmp(lbl39)
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d201 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r211}
			ctx.BindReg(r211, &d201)
			ctx.BindReg(r211, &d201)
			if r191 { ctx.UnprotectReg(r192) }
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			if d201.Loc == scm.LocStack || d201.Loc == scm.LocStackPair { ctx.EnsureDesc(&d201) }
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d201.Imm.Int()))))}
			} else {
				r229 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r229, d201.Reg)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d202)
			}
			ctx.FreeDesc(&d201)
			var d203 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r230, thisptr.Reg, off)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r230}
				ctx.BindReg(r230, &d203)
			}
			if d202.Loc == scm.LocStack || d202.Loc == scm.LocStackPair { ctx.EnsureDesc(&d202) }
			if d203.Loc == scm.LocStack || d203.Loc == scm.LocStackPair { ctx.EnsureDesc(&d203) }
			var d204 scm.JITValueDesc
			if d202.Loc == scm.LocImm && d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d202.Imm.Int() + d203.Imm.Int())}
			} else if d203.Loc == scm.LocImm && d203.Imm.Int() == 0 {
				r231 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r231, d202.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d204)
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d203.Reg}
				ctx.BindReg(d203.Reg, &d204)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			} else if d203.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(scratch, d202.Reg)
				if d203.Imm.Int() >= -2147483648 && d203.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d203.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d203.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d204)
			} else {
				r232 := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegReg(r232, d202.Reg)
				ctx.W.EmitAddInt64(r232, d203.Reg)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d204)
			}
			if d204.Loc == scm.LocReg && d202.Loc == scm.LocReg && d204.Reg == d202.Reg {
				ctx.TransferReg(d202.Reg)
				d202.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d202)
			ctx.FreeDesc(&d203)
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			if d204.Loc == scm.LocStack || d204.Loc == scm.LocStackPair { ctx.EnsureDesc(&d204) }
			var d205 scm.JITValueDesc
			if d204.Loc == scm.LocImm {
				d205 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d204.Imm.Int()))))}
			} else {
				r233 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r233, d204.Reg)
				d205 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d205)
			}
			ctx.FreeDesc(&d204)
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			var d206 scm.JITValueDesc
			if d79.Loc == scm.LocImm {
				d206 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d79.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d79.Reg)
				d206 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d206)
			}
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d205.Loc == scm.LocStack || d205.Loc == scm.LocStackPair { ctx.EnsureDesc(&d205) }
			var d207 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d205.Loc == scm.LocImm {
				d207 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d79.Imm.Int() + d205.Imm.Int())}
			} else if d205.Loc == scm.LocImm && d205.Imm.Int() == 0 {
				r235 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r235, d79.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r235}
				ctx.BindReg(r235, &d207)
			} else if d79.Loc == scm.LocImm && d79.Imm.Int() == 0 {
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d205.Reg}
				ctx.BindReg(d205.Reg, &d207)
			} else if d79.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d205.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d79.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else if d205.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(scratch, d79.Reg)
				if d205.Imm.Int() >= -2147483648 && d205.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d205.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d205.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d207)
			} else {
				r236 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitMovRegReg(r236, d79.Reg)
				ctx.W.EmitAddInt64(r236, d205.Reg)
				d207 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r236}
				ctx.BindReg(r236, &d207)
			}
			if d207.Loc == scm.LocReg && d79.Loc == scm.LocReg && d207.Reg == d79.Reg {
				ctx.TransferReg(d79.Reg)
				d79.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d205)
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			if d207.Loc == scm.LocStack || d207.Loc == scm.LocStackPair { ctx.EnsureDesc(&d207) }
			var d208 scm.JITValueDesc
			if d207.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d207.Imm.Int()))))}
			} else {
				r237 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r237, d207.Reg)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r237}
				ctx.BindReg(r237, &d208)
			}
			ctx.FreeDesc(&d207)
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			r238 := ctx.AllocReg()
			r239 := ctx.AllocRegExcept(r238)
			if d160.Loc == scm.LocStack || d160.Loc == scm.LocStackPair { ctx.EnsureDesc(&d160) }
			if d206.Loc == scm.LocStack || d206.Loc == scm.LocStackPair { ctx.EnsureDesc(&d206) }
			if d208.Loc == scm.LocStack || d208.Loc == scm.LocStackPair { ctx.EnsureDesc(&d208) }
			if d160.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r238, uint64(d160.Imm.Int()))
			} else if d160.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r238, d160.Reg)
			} else {
				ctx.W.EmitMovRegReg(r238, d160.Reg)
			}
			if d206.Loc == scm.LocImm {
				if d206.Imm.Int() != 0 {
					if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r238, int32(d206.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
						ctx.W.EmitAddInt64(r238, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r238, d206.Reg)
			}
			if d208.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r239, uint64(d208.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r239, d208.Reg)
			}
			if d206.Loc == scm.LocImm {
				if d206.Imm.Int() >= -2147483648 && d206.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r239, int32(d206.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d206.Imm.Int()))
					ctx.W.EmitSubInt64(r239, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r239, d206.Reg)
			}
			d209 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r238, Reg2: r239}
			ctx.BindReg(r238, &d209)
			ctx.BindReg(r239, &d209)
			ctx.FreeDesc(&d206)
			ctx.FreeDesc(&d208)
			d210 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d210)
			ctx.BindReg(r185, &d210)
			d211 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d209}, 2)
			ctx.EmitMovPairToResult(&d211, &d210)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl20)
			var d212 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d212 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 64)
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r240, thisptr.Reg, off)
				d212 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r240}
				ctx.BindReg(r240, &d212)
			}
			if d79.Loc == scm.LocStack || d79.Loc == scm.LocStackPair { ctx.EnsureDesc(&d79) }
			if d212.Loc == scm.LocStack || d212.Loc == scm.LocStackPair { ctx.EnsureDesc(&d212) }
			var d213 scm.JITValueDesc
			if d79.Loc == scm.LocImm && d212.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d79.Imm.Int()) == uint64(d212.Imm.Int()))}
			} else if d212.Loc == scm.LocImm {
				r241 := ctx.AllocRegExcept(d79.Reg)
				if d212.Imm.Int() >= -2147483648 && d212.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d79.Reg, int32(d212.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d212.Imm.Int()))
					ctx.W.EmitCmpInt64(d79.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r241, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d213)
			} else if d79.Loc == scm.LocImm {
				r242 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d79.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d212.Reg)
				ctx.W.EmitSetcc(r242, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r242}
				ctx.BindReg(r242, &d213)
			} else {
				r243 := ctx.AllocRegExcept(d79.Reg)
				ctx.W.EmitCmpInt64(d79.Reg, d212.Reg)
				ctx.W.EmitSetcc(r243, scm.CcE)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d213)
			}
			ctx.FreeDesc(&d79)
			ctx.FreeDesc(&d212)
			lbl43 := ctx.W.ReserveLabel()
			lbl44 := ctx.W.ReserveLabel()
			if d213.Loc == scm.LocImm {
				if d213.Imm.Bool() {
					ctx.W.EmitJmp(lbl43)
				} else {
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d213.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl44)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl44)
				ctx.W.EmitJmp(lbl43)
			}
			ctx.FreeDesc(&d213)
			ctx.W.MarkLabel(lbl35)
			d214 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d214)
			ctx.BindReg(r185, &d214)
			ctx.W.EmitMakeNil(d214)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl43)
			d215 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d215)
			ctx.BindReg(r185, &d215)
			ctx.W.EmitMakeNil(d215)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			d216 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r184, Reg2: r185}
			ctx.BindReg(r184, &d216)
			ctx.BindReg(r185, &d216)
			ctx.BindReg(r184, &d216)
			ctx.BindReg(r185, &d216)
			if r0 { ctx.UnprotectReg(r1) }
			d217 := ctx.EmitTagEquals(&d216, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d217.Loc == scm.LocImm {
				if d217.Imm.Bool() {
					ctx.W.EmitJmp(lbl45)
				} else {
					ctx.W.EmitJmp(lbl46)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d217.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl47)
				ctx.W.EmitJmp(lbl46)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d217)
			ctx.W.MarkLabel(lbl46)
			d218 := ctx.EmitTagEquals(&d216, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			lbl48 := ctx.W.ReserveLabel()
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			if d218.Loc == scm.LocImm {
				if d218.Imm.Bool() {
					ctx.W.EmitJmp(lbl48)
				} else {
					ctx.W.EmitJmp(lbl49)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d218.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl50)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl50)
				ctx.W.EmitJmp(lbl48)
			}
			ctx.FreeDesc(&d218)
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
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d219 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r244, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r244, 32)
				ctx.W.EmitShrRegImm8(r244, 32)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r244}
				ctx.BindReg(r244, &d219)
			}
			ctx.FreeDesc(&idxInt)
			var d220 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r245 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r245, thisptr.Reg, off)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r245}
				ctx.BindReg(r245, &d220)
			}
			if d220.Loc == scm.LocStack || d220.Loc == scm.LocStackPair { ctx.EnsureDesc(&d220) }
			if d220.Loc == scm.LocStack || d220.Loc == scm.LocStackPair { ctx.EnsureDesc(&d220) }
			var d221 scm.JITValueDesc
			if d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d220.Imm.Int()))))}
			} else {
				r246 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r246, d220.Reg)
				ctx.W.EmitShlRegImm8(r246, 56)
				ctx.W.EmitShrRegImm8(r246, 56)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r246}
				ctx.BindReg(r246, &d221)
			}
			ctx.FreeDesc(&d220)
			if d219.Loc == scm.LocStack || d219.Loc == scm.LocStackPair { ctx.EnsureDesc(&d219) }
			if d221.Loc == scm.LocStack || d221.Loc == scm.LocStackPair { ctx.EnsureDesc(&d221) }
			var d222 scm.JITValueDesc
			if d219.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d219.Imm.Int() * d221.Imm.Int())}
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d219.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegReg(scratch, d219.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d222)
			} else {
				r247 := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegReg(r247, d219.Reg)
				ctx.W.EmitImulInt64(r247, d221.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r247}
				ctx.BindReg(r247, &d222)
			}
			if d222.Loc == scm.LocReg && d219.Loc == scm.LocReg && d222.Reg == d219.Reg {
				ctx.TransferReg(d219.Reg)
				d219.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d219)
			ctx.FreeDesc(&d221)
			var d223 scm.JITValueDesc
			r248 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r248, uint64(dataPtr))
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r248, StackOff: int32(sliceLen)}
				ctx.BindReg(r248, &d223)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 0)
				ctx.W.EmitMovRegMem(r248, thisptr.Reg, off)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r248}
				ctx.BindReg(r248, &d223)
			}
			ctx.BindReg(r248, &d223)
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			var d224 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() / 64)}
			} else {
				r249 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r249, d222.Reg)
				ctx.W.EmitShrRegImm8(r249, 6)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r249}
				ctx.BindReg(r249, &d224)
			}
			if d224.Loc == scm.LocReg && d222.Loc == scm.LocReg && d224.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			if d224.Loc == scm.LocStack || d224.Loc == scm.LocStackPair { ctx.EnsureDesc(&d224) }
			r250 := ctx.AllocReg()
			if d224.Loc == scm.LocStack || d224.Loc == scm.LocStackPair { ctx.EnsureDesc(&d224) }
			if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair { ctx.EnsureDesc(&d223) }
			if d224.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r250, uint64(d224.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r250, d224.Reg)
				ctx.W.EmitShlRegImm8(r250, 3)
			}
			if d223.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d223.Imm.Int()))
				ctx.W.EmitAddInt64(r250, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r250, d223.Reg)
			}
			r251 := ctx.AllocRegExcept(r250)
			ctx.W.EmitMovRegMem(r251, r250, 0)
			ctx.FreeReg(r250)
			d225 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r251}
			ctx.BindReg(r251, &d225)
			ctx.FreeDesc(&d224)
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			var d226 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d226 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() % 64)}
			} else {
				r252 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r252, d222.Reg)
				ctx.W.EmitAndRegImm32(r252, 63)
				d226 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r252}
				ctx.BindReg(r252, &d226)
			}
			if d226.Loc == scm.LocReg && d222.Loc == scm.LocReg && d226.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			if d225.Loc == scm.LocStack || d225.Loc == scm.LocStackPair { ctx.EnsureDesc(&d225) }
			if d226.Loc == scm.LocStack || d226.Loc == scm.LocStackPair { ctx.EnsureDesc(&d226) }
			var d227 scm.JITValueDesc
			if d225.Loc == scm.LocImm && d226.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d225.Imm.Int()) << uint64(d226.Imm.Int())))}
			} else if d226.Loc == scm.LocImm {
				r253 := ctx.AllocRegExcept(d225.Reg)
				ctx.W.EmitMovRegReg(r253, d225.Reg)
				ctx.W.EmitShlRegImm8(r253, uint8(d226.Imm.Int()))
				d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r253}
				ctx.BindReg(r253, &d227)
			} else {
				{
					shiftSrc := d225.Reg
					r254 := ctx.AllocRegExcept(d225.Reg)
					ctx.W.EmitMovRegReg(r254, d225.Reg)
					shiftSrc = r254
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d226.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d226.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d226.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d227 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d227)
				}
			}
			if d227.Loc == scm.LocReg && d225.Loc == scm.LocReg && d227.Reg == d225.Reg {
				ctx.TransferReg(d225.Reg)
				d225.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d225)
			ctx.FreeDesc(&d226)
			var d228 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 25)
				r255 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r255, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r255}
				ctx.BindReg(r255, &d228)
			}
			lbl52 := ctx.W.ReserveLabel()
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			if d228.Loc == scm.LocImm {
				if d228.Imm.Bool() {
					ctx.W.EmitJmp(lbl52)
				} else {
			d229 := d227
			if d229.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d229.Loc == scm.LocStack || d229.Loc == scm.LocStackPair { ctx.EnsureDesc(&d229) }
			ctx.EmitStoreToStack(d229, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d228.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl54)
			d230 := d227
			if d230.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d230.Loc == scm.LocStack || d230.Loc == scm.LocStackPair { ctx.EnsureDesc(&d230) }
			ctx.EmitStoreToStack(d230, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl52)
			}
			ctx.FreeDesc(&d228)
			ctx.W.MarkLabel(lbl53)
			d231 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(40)}
			var d232 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d232 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r256 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r256, thisptr.Reg, off)
				d232 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r256}
				ctx.BindReg(r256, &d232)
			}
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			if d232.Loc == scm.LocStack || d232.Loc == scm.LocStackPair { ctx.EnsureDesc(&d232) }
			var d233 scm.JITValueDesc
			if d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d232.Imm.Int()))))}
			} else {
				r257 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r257, d232.Reg)
				ctx.W.EmitShlRegImm8(r257, 56)
				ctx.W.EmitShrRegImm8(r257, 56)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r257}
				ctx.BindReg(r257, &d233)
			}
			ctx.FreeDesc(&d232)
			d234 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d233.Loc == scm.LocStack || d233.Loc == scm.LocStackPair { ctx.EnsureDesc(&d233) }
			var d235 scm.JITValueDesc
			if d234.Loc == scm.LocImm && d233.Loc == scm.LocImm {
				d235 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d234.Imm.Int() - d233.Imm.Int())}
			} else if d233.Loc == scm.LocImm && d233.Imm.Int() == 0 {
				r258 := ctx.AllocRegExcept(d234.Reg)
				ctx.W.EmitMovRegReg(r258, d234.Reg)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r258}
				ctx.BindReg(r258, &d235)
			} else if d234.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d233.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d234.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d233.Reg)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d235)
			} else if d233.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d234.Reg)
				ctx.W.EmitMovRegReg(scratch, d234.Reg)
				if d233.Imm.Int() >= -2147483648 && d233.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d233.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d233.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d235)
			} else {
				r259 := ctx.AllocRegExcept(d234.Reg)
				ctx.W.EmitMovRegReg(r259, d234.Reg)
				ctx.W.EmitSubInt64(r259, d233.Reg)
				d235 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r259}
				ctx.BindReg(r259, &d235)
			}
			if d235.Loc == scm.LocReg && d234.Loc == scm.LocReg && d235.Reg == d234.Reg {
				ctx.TransferReg(d234.Reg)
				d234.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d233)
			if d231.Loc == scm.LocStack || d231.Loc == scm.LocStackPair { ctx.EnsureDesc(&d231) }
			if d235.Loc == scm.LocStack || d235.Loc == scm.LocStackPair { ctx.EnsureDesc(&d235) }
			var d236 scm.JITValueDesc
			if d231.Loc == scm.LocImm && d235.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d231.Imm.Int()) >> uint64(d235.Imm.Int())))}
			} else if d235.Loc == scm.LocImm {
				r260 := ctx.AllocRegExcept(d231.Reg)
				ctx.W.EmitMovRegReg(r260, d231.Reg)
				ctx.W.EmitShrRegImm8(r260, uint8(d235.Imm.Int()))
				d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r260}
				ctx.BindReg(r260, &d236)
			} else {
				{
					shiftSrc := d231.Reg
					r261 := ctx.AllocRegExcept(d231.Reg)
					ctx.W.EmitMovRegReg(r261, d231.Reg)
					shiftSrc = r261
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d235.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d235.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d235.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d236 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d236)
				}
			}
			if d236.Loc == scm.LocReg && d231.Loc == scm.LocReg && d236.Reg == d231.Reg {
				ctx.TransferReg(d231.Reg)
				d231.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d231)
			ctx.FreeDesc(&d235)
			r262 := ctx.AllocReg()
			if d236.Loc == scm.LocStack || d236.Loc == scm.LocStackPair { ctx.EnsureDesc(&d236) }
			if d236.Loc == scm.LocStack || d236.Loc == scm.LocStackPair { ctx.EnsureDesc(&d236) }
			if d236.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r262, d236.Reg2)
			} else {
				ctx.EmitMovToReg(r262, d236)
			}
			ctx.W.EmitJmp(lbl51)
			ctx.W.MarkLabel(lbl52)
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			var d237 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d237 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() % 64)}
			} else {
				r263 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r263, d222.Reg)
				ctx.W.EmitAndRegImm32(r263, 63)
				d237 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r263}
				ctx.BindReg(r263, &d237)
			}
			if d237.Loc == scm.LocReg && d222.Loc == scm.LocReg && d237.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			var d238 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d238 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 24)
				r264 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r264, thisptr.Reg, off)
				d238 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r264}
				ctx.BindReg(r264, &d238)
			}
			if d238.Loc == scm.LocStack || d238.Loc == scm.LocStackPair { ctx.EnsureDesc(&d238) }
			if d238.Loc == scm.LocStack || d238.Loc == scm.LocStackPair { ctx.EnsureDesc(&d238) }
			var d239 scm.JITValueDesc
			if d238.Loc == scm.LocImm {
				d239 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d238.Imm.Int()))))}
			} else {
				r265 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r265, d238.Reg)
				ctx.W.EmitShlRegImm8(r265, 56)
				ctx.W.EmitShrRegImm8(r265, 56)
				d239 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r265}
				ctx.BindReg(r265, &d239)
			}
			ctx.FreeDesc(&d238)
			if d237.Loc == scm.LocStack || d237.Loc == scm.LocStackPair { ctx.EnsureDesc(&d237) }
			if d239.Loc == scm.LocStack || d239.Loc == scm.LocStackPair { ctx.EnsureDesc(&d239) }
			var d240 scm.JITValueDesc
			if d237.Loc == scm.LocImm && d239.Loc == scm.LocImm {
				d240 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d237.Imm.Int() + d239.Imm.Int())}
			} else if d239.Loc == scm.LocImm && d239.Imm.Int() == 0 {
				r266 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r266, d237.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r266}
				ctx.BindReg(r266, &d240)
			} else if d237.Loc == scm.LocImm && d237.Imm.Int() == 0 {
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d239.Reg}
				ctx.BindReg(d239.Reg, &d240)
			} else if d237.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d239.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d237.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d239.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else if d239.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(scratch, d237.Reg)
				if d239.Imm.Int() >= -2147483648 && d239.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d239.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d239.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d240)
			} else {
				r267 := ctx.AllocRegExcept(d237.Reg)
				ctx.W.EmitMovRegReg(r267, d237.Reg)
				ctx.W.EmitAddInt64(r267, d239.Reg)
				d240 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r267}
				ctx.BindReg(r267, &d240)
			}
			if d240.Loc == scm.LocReg && d237.Loc == scm.LocReg && d240.Reg == d237.Reg {
				ctx.TransferReg(d237.Reg)
				d237.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d237)
			ctx.FreeDesc(&d239)
			if d240.Loc == scm.LocStack || d240.Loc == scm.LocStackPair { ctx.EnsureDesc(&d240) }
			var d241 scm.JITValueDesc
			if d240.Loc == scm.LocImm {
				d241 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d240.Imm.Int()) > uint64(64))}
			} else {
				r268 := ctx.AllocRegExcept(d240.Reg)
				ctx.W.EmitCmpRegImm32(d240.Reg, 64)
				ctx.W.EmitSetcc(r268, scm.CcA)
				d241 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r268}
				ctx.BindReg(r268, &d241)
			}
			ctx.FreeDesc(&d240)
			lbl55 := ctx.W.ReserveLabel()
			lbl56 := ctx.W.ReserveLabel()
			if d241.Loc == scm.LocImm {
				if d241.Imm.Bool() {
					ctx.W.EmitJmp(lbl55)
				} else {
			d242 := d227
			if d242.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d242.Loc == scm.LocStack || d242.Loc == scm.LocStackPair { ctx.EnsureDesc(&d242) }
			ctx.EmitStoreToStack(d242, 40)
					ctx.W.EmitJmp(lbl53)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d241.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl56)
			d243 := d227
			if d243.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d243.Loc == scm.LocStack || d243.Loc == scm.LocStackPair { ctx.EnsureDesc(&d243) }
			ctx.EmitStoreToStack(d243, 40)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl56)
				ctx.W.EmitJmp(lbl55)
			}
			ctx.FreeDesc(&d241)
			ctx.W.MarkLabel(lbl55)
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			var d244 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d244 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() / 64)}
			} else {
				r269 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r269, d222.Reg)
				ctx.W.EmitShrRegImm8(r269, 6)
				d244 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r269}
				ctx.BindReg(r269, &d244)
			}
			if d244.Loc == scm.LocReg && d222.Loc == scm.LocReg && d244.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			if d244.Loc == scm.LocStack || d244.Loc == scm.LocStackPair { ctx.EnsureDesc(&d244) }
			var d245 scm.JITValueDesc
			if d244.Loc == scm.LocImm {
				d245 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d244.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d244.Reg)
				ctx.W.EmitMovRegReg(scratch, d244.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d245 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d245)
			}
			if d245.Loc == scm.LocReg && d244.Loc == scm.LocReg && d245.Reg == d244.Reg {
				ctx.TransferReg(d244.Reg)
				d244.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d244)
			if d245.Loc == scm.LocStack || d245.Loc == scm.LocStackPair { ctx.EnsureDesc(&d245) }
			r270 := ctx.AllocReg()
			if d245.Loc == scm.LocStack || d245.Loc == scm.LocStackPair { ctx.EnsureDesc(&d245) }
			if d223.Loc == scm.LocStack || d223.Loc == scm.LocStackPair { ctx.EnsureDesc(&d223) }
			if d245.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r270, uint64(d245.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r270, d245.Reg)
				ctx.W.EmitShlRegImm8(r270, 3)
			}
			if d223.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d223.Imm.Int()))
				ctx.W.EmitAddInt64(r270, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r270, d223.Reg)
			}
			r271 := ctx.AllocRegExcept(r270)
			ctx.W.EmitMovRegMem(r271, r270, 0)
			ctx.FreeReg(r270)
			d246 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r271}
			ctx.BindReg(r271, &d246)
			ctx.FreeDesc(&d245)
			if d222.Loc == scm.LocStack || d222.Loc == scm.LocStackPair { ctx.EnsureDesc(&d222) }
			var d247 scm.JITValueDesc
			if d222.Loc == scm.LocImm {
				d247 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d222.Imm.Int() % 64)}
			} else {
				r272 := ctx.AllocRegExcept(d222.Reg)
				ctx.W.EmitMovRegReg(r272, d222.Reg)
				ctx.W.EmitAndRegImm32(r272, 63)
				d247 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r272}
				ctx.BindReg(r272, &d247)
			}
			if d247.Loc == scm.LocReg && d222.Loc == scm.LocReg && d247.Reg == d222.Reg {
				ctx.TransferReg(d222.Reg)
				d222.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d222)
			d248 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d247.Loc == scm.LocStack || d247.Loc == scm.LocStackPair { ctx.EnsureDesc(&d247) }
			var d249 scm.JITValueDesc
			if d248.Loc == scm.LocImm && d247.Loc == scm.LocImm {
				d249 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d248.Imm.Int() - d247.Imm.Int())}
			} else if d247.Loc == scm.LocImm && d247.Imm.Int() == 0 {
				r273 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r273, d248.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r273}
				ctx.BindReg(r273, &d249)
			} else if d248.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d247.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d248.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d247.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else if d247.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(scratch, d248.Reg)
				if d247.Imm.Int() >= -2147483648 && d247.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d247.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d247.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d249)
			} else {
				r274 := ctx.AllocRegExcept(d248.Reg)
				ctx.W.EmitMovRegReg(r274, d248.Reg)
				ctx.W.EmitSubInt64(r274, d247.Reg)
				d249 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r274}
				ctx.BindReg(r274, &d249)
			}
			if d249.Loc == scm.LocReg && d248.Loc == scm.LocReg && d249.Reg == d248.Reg {
				ctx.TransferReg(d248.Reg)
				d248.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d247)
			if d246.Loc == scm.LocStack || d246.Loc == scm.LocStackPair { ctx.EnsureDesc(&d246) }
			if d249.Loc == scm.LocStack || d249.Loc == scm.LocStackPair { ctx.EnsureDesc(&d249) }
			var d250 scm.JITValueDesc
			if d246.Loc == scm.LocImm && d249.Loc == scm.LocImm {
				d250 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d246.Imm.Int()) >> uint64(d249.Imm.Int())))}
			} else if d249.Loc == scm.LocImm {
				r275 := ctx.AllocRegExcept(d246.Reg)
				ctx.W.EmitMovRegReg(r275, d246.Reg)
				ctx.W.EmitShrRegImm8(r275, uint8(d249.Imm.Int()))
				d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r275}
				ctx.BindReg(r275, &d250)
			} else {
				{
					shiftSrc := d246.Reg
					r276 := ctx.AllocRegExcept(d246.Reg)
					ctx.W.EmitMovRegReg(r276, d246.Reg)
					shiftSrc = r276
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d249.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d249.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d249.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d250 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d250)
				}
			}
			if d250.Loc == scm.LocReg && d246.Loc == scm.LocReg && d250.Reg == d246.Reg {
				ctx.TransferReg(d246.Reg)
				d246.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d246)
			ctx.FreeDesc(&d249)
			if d227.Loc == scm.LocStack || d227.Loc == scm.LocStackPair { ctx.EnsureDesc(&d227) }
			if d250.Loc == scm.LocStack || d250.Loc == scm.LocStackPair { ctx.EnsureDesc(&d250) }
			var d251 scm.JITValueDesc
			if d227.Loc == scm.LocImm && d250.Loc == scm.LocImm {
				d251 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d227.Imm.Int() | d250.Imm.Int())}
			} else if d227.Loc == scm.LocImm && d227.Imm.Int() == 0 {
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d250.Reg}
				ctx.BindReg(d250.Reg, &d251)
			} else if d250.Loc == scm.LocImm && d250.Imm.Int() == 0 {
				r277 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r277, d227.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r277}
				ctx.BindReg(r277, &d251)
			} else if d227.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d250.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d227.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d250.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d251)
			} else if d250.Loc == scm.LocImm {
				r278 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r278, d227.Reg)
				if d250.Imm.Int() >= -2147483648 && d250.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r278, int32(d250.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d250.Imm.Int()))
					ctx.W.EmitOrInt64(r278, scm.RegR11)
				}
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r278}
				ctx.BindReg(r278, &d251)
			} else {
				r279 := ctx.AllocRegExcept(d227.Reg)
				ctx.W.EmitMovRegReg(r279, d227.Reg)
				ctx.W.EmitOrInt64(r279, d250.Reg)
				d251 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r279}
				ctx.BindReg(r279, &d251)
			}
			if d251.Loc == scm.LocReg && d227.Loc == scm.LocReg && d251.Reg == d227.Reg {
				ctx.TransferReg(d227.Reg)
				d227.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d250)
			d252 := d251
			if d252.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d252.Loc == scm.LocStack || d252.Loc == scm.LocStackPair { ctx.EnsureDesc(&d252) }
			ctx.EmitStoreToStack(d252, 40)
			ctx.W.EmitJmp(lbl53)
			ctx.W.MarkLabel(lbl51)
			ctx.W.ResolveFixups()
			d253 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r262}
			ctx.BindReg(r262, &d253)
			ctx.BindReg(r262, &d253)
			ctx.FreeDesc(&idxInt)
			if d253.Loc == scm.LocStack || d253.Loc == scm.LocStackPair { ctx.EnsureDesc(&d253) }
			if d253.Loc == scm.LocStack || d253.Loc == scm.LocStackPair { ctx.EnsureDesc(&d253) }
			var d254 scm.JITValueDesc
			if d253.Loc == scm.LocImm {
				d254 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d253.Imm.Int()))))}
			} else {
				r280 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r280, d253.Reg)
				d254 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r280}
				ctx.BindReg(r280, &d254)
			}
			ctx.FreeDesc(&d253)
			var d255 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d255 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixes) + 32)
				r281 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r281, thisptr.Reg, off)
				d255 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r281}
				ctx.BindReg(r281, &d255)
			}
			if d254.Loc == scm.LocStack || d254.Loc == scm.LocStackPair { ctx.EnsureDesc(&d254) }
			if d255.Loc == scm.LocStack || d255.Loc == scm.LocStackPair { ctx.EnsureDesc(&d255) }
			var d256 scm.JITValueDesc
			if d254.Loc == scm.LocImm && d255.Loc == scm.LocImm {
				d256 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d254.Imm.Int() + d255.Imm.Int())}
			} else if d255.Loc == scm.LocImm && d255.Imm.Int() == 0 {
				r282 := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(r282, d254.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r282}
				ctx.BindReg(r282, &d256)
			} else if d254.Loc == scm.LocImm && d254.Imm.Int() == 0 {
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d255.Reg}
				ctx.BindReg(d255.Reg, &d256)
			} else if d254.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d255.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d254.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d255.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d256)
			} else if d255.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(scratch, d254.Reg)
				if d255.Imm.Int() >= -2147483648 && d255.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d255.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d255.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d256)
			} else {
				r283 := ctx.AllocRegExcept(d254.Reg)
				ctx.W.EmitMovRegReg(r283, d254.Reg)
				ctx.W.EmitAddInt64(r283, d255.Reg)
				d256 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r283}
				ctx.BindReg(r283, &d256)
			}
			if d256.Loc == scm.LocReg && d254.Loc == scm.LocReg && d256.Reg == d254.Reg {
				ctx.TransferReg(d254.Reg)
				d254.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d254)
			ctx.FreeDesc(&d255)
			var d257 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary)
				r284 := ctx.AllocReg()
				r285 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r284, fieldAddr)
				ctx.W.EmitMovRegMem64(r285, fieldAddr+8)
				d257 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r284, Reg2: r285}
				ctx.BindReg(r284, &d257)
				ctx.BindReg(r285, &d257)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).prefixdictionary))
				r286 := ctx.AllocReg()
				r287 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r286, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r287, thisptr.Reg, off+8)
				d257 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r286, Reg2: r287}
				ctx.BindReg(r286, &d257)
				ctx.BindReg(r287, &d257)
			}
			var d258 scm.JITValueDesc
			if d257.Loc == scm.LocImm {
				d258 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(d257.StackOff))}
			} else {
				d258 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d257.Reg2}
				ctx.BindReg(d257.Reg2, &d258)
			}
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			if d258.Loc == scm.LocStack || d258.Loc == scm.LocStackPair { ctx.EnsureDesc(&d258) }
			var d260 scm.JITValueDesc
			if d256.Loc == scm.LocImm && d258.Loc == scm.LocImm {
				d260 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d256.Imm.Int() >= d258.Imm.Int())}
			} else if d258.Loc == scm.LocImm {
				r288 := ctx.AllocRegExcept(d256.Reg)
				if d258.Imm.Int() >= -2147483648 && d258.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d256.Reg, int32(d258.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d258.Imm.Int()))
					ctx.W.EmitCmpInt64(d256.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r288, scm.CcGE)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r288}
				ctx.BindReg(r288, &d260)
			} else if d256.Loc == scm.LocImm {
				r289 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d256.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d258.Reg)
				ctx.W.EmitSetcc(r289, scm.CcGE)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r289}
				ctx.BindReg(r289, &d260)
			} else {
				r290 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitCmpInt64(d256.Reg, d258.Reg)
				ctx.W.EmitSetcc(r290, scm.CcGE)
				d260 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r290}
				ctx.BindReg(r290, &d260)
			}
			ctx.FreeDesc(&d258)
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			lbl59 := ctx.W.ReserveLabel()
			if d260.Loc == scm.LocImm {
				if d260.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl58)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d260.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl59)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl59)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d260)
			ctx.W.MarkLabel(lbl58)
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			var d261 scm.JITValueDesc
			if d256.Loc == scm.LocImm {
				d261 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d256.Imm.Int() < 0)}
			} else {
				r291 := ctx.AllocRegExcept(d256.Reg)
				ctx.W.EmitCmpRegImm32(d256.Reg, 0)
				ctx.W.EmitSetcc(r291, scm.CcL)
				d261 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r291}
				ctx.BindReg(r291, &d261)
			}
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d261.Loc == scm.LocImm {
				if d261.Imm.Bool() {
					ctx.W.EmitJmp(lbl57)
				} else {
					ctx.W.EmitJmp(lbl60)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d261.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl61)
				ctx.W.EmitJmp(lbl60)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl57)
			}
			ctx.FreeDesc(&d261)
			ctx.W.MarkLabel(lbl57)
			ctx.W.EmitByte(0xCC)
			ctx.W.MarkLabel(lbl60)
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			r292 := ctx.AllocReg()
			if d256.Loc == scm.LocStack || d256.Loc == scm.LocStackPair { ctx.EnsureDesc(&d256) }
			if d257.Loc == scm.LocStack || d257.Loc == scm.LocStackPair { ctx.EnsureDesc(&d257) }
			if d256.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r292, uint64(d256.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r292, d256.Reg)
				ctx.W.EmitShlRegImm8(r292, 4)
			}
			if d257.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d257.Imm.Int()))
				ctx.W.EmitAddInt64(r292, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r292, d257.Reg)
			}
			r293 := ctx.AllocRegExcept(r292)
			r294 := ctx.AllocRegExcept(r292, r293)
			ctx.W.EmitMovRegMem(r293, r292, 0)
			ctx.W.EmitMovRegMem(r294, r292, 8)
			ctx.FreeReg(r292)
			d262 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r293, Reg2: r294}
			ctx.BindReg(r293, &d262)
			ctx.BindReg(r294, &d262)
			ctx.FreeDesc(&d256)
			d263 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.Scmer.String), []scm.JITValueDesc{d216}, 2)
			ctx.FreeDesc(&d216)
			if d262.Loc == scm.LocStack || d262.Loc == scm.LocStackPair { ctx.EnsureDesc(&d262) }
			if d263.Loc == scm.LocStack || d263.Loc == scm.LocStackPair { ctx.EnsureDesc(&d263) }
			d264 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.ConcatStrings), []scm.JITValueDesc{d262, d263}, 2)
			ctx.FreeDesc(&d262)
			d265 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d264}, 2)
			ctx.EmitMovPairToResult(&d265, &result)
			result.Type = scm.TagString
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
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
