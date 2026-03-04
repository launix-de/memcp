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
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&idxInt)
			d0 := idxInt
			_ = d0
			r2 := idxInt.Loc == scm.LocReg
			r3 := idxInt.Reg
			if r2 { ctx.ProtectReg(r3) }
			lbl3 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_4 := int32(-1)
			_ = bbpos_1_4
			bbpos_1_5 := int32(-1)
			_ = bbpos_1_5
			bbpos_1_6 := int32(-1)
			_ = bbpos_1_6
			bbpos_1_7 := int32(-1)
			_ = bbpos_1_7
			bbpos_1_8 := int32(-1)
			_ = bbpos_1_8
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d3 := d0
			_ = d3
			r5 := d0.Loc == scm.LocReg
			r6 := d0.Reg
			if r5 { ctx.ProtectReg(r6) }
			r7 := ctx.W.EmitSubRSP32Fixup()
			lbl8 := ctx.W.ReserveLabel()
			bbpos_2_0 := int32(-1)
			_ = bbpos_2_0
			bbpos_2_1 := int32(-1)
			_ = bbpos_2_1
			bbpos_2_2 := int32(-1)
			_ = bbpos_2_2
			bbpos_2_3 := int32(-1)
			_ = bbpos_2_3
			bbpos_2_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d14.Loc == scm.LocImm {
				if d14.Imm.Bool() {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				} else {
					ctx.W.MarkLabel(lbl12)
			d15 := d12
			if d15.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d15)
			ctx.EmitStoreToStack(d15, 0)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d14.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl12)
			d16 := d12
			if d16.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d16)
			ctx.EmitStoreToStack(d16, 0)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d13)
			bbpos_2_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl10)
			ctx.W.ResolveFixups()
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
			bbpos_2_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl9)
			ctx.W.ResolveFixups()
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
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d28.Loc == scm.LocImm {
				if d28.Imm.Bool() {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.MarkLabel(lbl15)
			d29 := d12
			if d29.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d29)
			ctx.EmitStoreToStack(d29, 0)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d28.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl15)
			d30 := d12
			if d30.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d30)
			ctx.EmitStoreToStack(d30, 0)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d27)
			bbpos_2_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl13)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl10)
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
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d45)
			bbpos_1_7 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl17)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d44)
			d47 := d44
			_ = d47
			r50 := d44.Loc == scm.LocReg
			r51 := d44.Reg
			if r50 { ctx.ProtectReg(r51) }
			lbl20 := ctx.W.ReserveLabel()
			bbpos_3_0 := int32(-1)
			_ = bbpos_3_0
			bbpos_3_1 := int32(-1)
			_ = bbpos_3_1
			bbpos_3_2 := int32(-1)
			_ = bbpos_3_2
			bbpos_3_3 := int32(-1)
			_ = bbpos_3_3
			bbpos_3_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			lbl24 := ctx.W.ReserveLabel()
			if d58.Loc == scm.LocImm {
				if d58.Imm.Bool() {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				} else {
					ctx.W.MarkLabel(lbl24)
			d59 := d56
			if d59.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			ctx.EmitStoreToStack(d59, 8)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d58.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl23)
				ctx.W.EmitJmp(lbl24)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
				ctx.W.MarkLabel(lbl24)
			d60 := d56
			if d60.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d60)
			ctx.EmitStoreToStack(d60, 8)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d57)
			bbpos_3_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl22)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl20)
			bbpos_3_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl21)
			ctx.W.ResolveFixups()
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
			lbl25 := ctx.W.ReserveLabel()
			lbl26 := ctx.W.ReserveLabel()
			lbl27 := ctx.W.ReserveLabel()
			if d72.Loc == scm.LocImm {
				if d72.Imm.Bool() {
					ctx.W.MarkLabel(lbl26)
					ctx.W.EmitJmp(lbl25)
				} else {
					ctx.W.MarkLabel(lbl27)
			d73 := d56
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, 8)
					ctx.W.EmitJmp(lbl22)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d72.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl26)
				ctx.W.EmitJmp(lbl27)
				ctx.W.MarkLabel(lbl26)
				ctx.W.EmitJmp(lbl25)
				ctx.W.MarkLabel(lbl27)
			d74 := d56
			if d74.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d74)
			ctx.EmitStoreToStack(d74, 8)
				ctx.W.EmitJmp(lbl22)
			}
			ctx.FreeDesc(&d71)
			bbpos_3_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl25)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl20)
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
			ctx.EnsureDesc(&d44)
			d88 := d44
			_ = d88
			r92 := d44.Loc == scm.LocReg
			r93 := d44.Reg
			if r92 { ctx.ProtectReg(r93) }
			lbl28 := ctx.W.ReserveLabel()
			bbpos_4_0 := int32(-1)
			_ = bbpos_4_0
			bbpos_4_1 := int32(-1)
			_ = bbpos_4_1
			bbpos_4_2 := int32(-1)
			_ = bbpos_4_2
			bbpos_4_3 := int32(-1)
			_ = bbpos_4_3
			bbpos_4_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d88)
			ctx.EnsureDesc(&d88)
			var d89 scm.JITValueDesc
			if d88.Loc == scm.LocImm {
				d89 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d88.Imm.Int()))))}
			} else {
				r94 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r94, d88.Reg)
				ctx.W.EmitShlRegImm8(r94, 32)
				ctx.W.EmitShrRegImm8(r94, 32)
				d89 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r94}
				ctx.BindReg(r94, &d89)
			}
			var d90 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d90 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r95 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r95, thisptr.Reg, off)
				d90 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r95}
				ctx.BindReg(r95, &d90)
			}
			ctx.EnsureDesc(&d90)
			ctx.EnsureDesc(&d90)
			var d91 scm.JITValueDesc
			if d90.Loc == scm.LocImm {
				d91 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d90.Imm.Int()))))}
			} else {
				r96 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r96, d90.Reg)
				ctx.W.EmitShlRegImm8(r96, 56)
				ctx.W.EmitShrRegImm8(r96, 56)
				d91 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r96}
				ctx.BindReg(r96, &d91)
			}
			ctx.FreeDesc(&d90)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d91)
			ctx.EnsureDesc(&d89)
			ctx.EnsureDesc(&d91)
			var d92 scm.JITValueDesc
			if d89.Loc == scm.LocImm && d91.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d89.Imm.Int() * d91.Imm.Int())}
			} else if d89.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d91.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d89.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			} else if d91.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d89.Reg)
				ctx.W.EmitMovRegReg(scratch, d89.Reg)
				if d91.Imm.Int() >= -2147483648 && d91.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d91.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d91.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d92)
			} else {
				r97 := ctx.AllocRegExcept(d89.Reg, d91.Reg)
				ctx.W.EmitMovRegReg(r97, d89.Reg)
				ctx.W.EmitImulInt64(r97, d91.Reg)
				d92 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r97}
				ctx.BindReg(r97, &d92)
			}
			if d92.Loc == scm.LocReg && d89.Loc == scm.LocReg && d92.Reg == d89.Reg {
				ctx.TransferReg(d89.Reg)
				d89.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d89)
			ctx.FreeDesc(&d91)
			var d93 scm.JITValueDesc
			r98 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r98, uint64(dataPtr))
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98, StackOff: int32(sliceLen)}
				ctx.BindReg(r98, &d93)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r98, thisptr.Reg, off)
				d93 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r98}
				ctx.BindReg(r98, &d93)
			}
			ctx.BindReg(r98, &d93)
			ctx.EnsureDesc(&d92)
			var d94 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() / 64)}
			} else {
				r99 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r99, d92.Reg)
				ctx.W.EmitShrRegImm8(r99, 6)
				d94 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r99}
				ctx.BindReg(r99, &d94)
			}
			if d94.Loc == scm.LocReg && d92.Loc == scm.LocReg && d94.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d94)
			r100 := ctx.AllocReg()
			ctx.EnsureDesc(&d94)
			ctx.EnsureDesc(&d93)
			if d94.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r100, uint64(d94.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r100, d94.Reg)
				ctx.W.EmitShlRegImm8(r100, 3)
			}
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d93.Imm.Int()))
				ctx.W.EmitAddInt64(r100, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r100, d93.Reg)
			}
			r101 := ctx.AllocRegExcept(r100)
			ctx.W.EmitMovRegMem(r101, r100, 0)
			ctx.FreeReg(r100)
			d95 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r101}
			ctx.BindReg(r101, &d95)
			ctx.FreeDesc(&d94)
			ctx.EnsureDesc(&d92)
			var d96 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d96 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() % 64)}
			} else {
				r102 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r102, d92.Reg)
				ctx.W.EmitAndRegImm32(r102, 63)
				d96 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r102}
				ctx.BindReg(r102, &d96)
			}
			if d96.Loc == scm.LocReg && d92.Loc == scm.LocReg && d96.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d95)
			ctx.EnsureDesc(&d96)
			var d97 scm.JITValueDesc
			if d95.Loc == scm.LocImm && d96.Loc == scm.LocImm {
				d97 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d95.Imm.Int()) << uint64(d96.Imm.Int())))}
			} else if d96.Loc == scm.LocImm {
				r103 := ctx.AllocRegExcept(d95.Reg)
				ctx.W.EmitMovRegReg(r103, d95.Reg)
				ctx.W.EmitShlRegImm8(r103, uint8(d96.Imm.Int()))
				d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r103}
				ctx.BindReg(r103, &d97)
			} else {
				{
					shiftSrc := d95.Reg
					r104 := ctx.AllocRegExcept(d95.Reg)
					ctx.W.EmitMovRegReg(r104, d95.Reg)
					shiftSrc = r104
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d96.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d96.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d96.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d97 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d97)
				}
			}
			if d97.Loc == scm.LocReg && d95.Loc == scm.LocReg && d97.Reg == d95.Reg {
				ctx.TransferReg(d95.Reg)
				d95.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d95)
			ctx.FreeDesc(&d96)
			var d98 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d98 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r105 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r105, thisptr.Reg, off)
				d98 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r105}
				ctx.BindReg(r105, &d98)
			}
			d99 := d98
			ctx.EnsureDesc(&d99)
			if d99.Loc != scm.LocImm && d99.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl29 := ctx.W.ReserveLabel()
			lbl30 := ctx.W.ReserveLabel()
			lbl31 := ctx.W.ReserveLabel()
			lbl32 := ctx.W.ReserveLabel()
			if d99.Loc == scm.LocImm {
				if d99.Imm.Bool() {
					ctx.W.MarkLabel(lbl31)
					ctx.W.EmitJmp(lbl29)
				} else {
					ctx.W.MarkLabel(lbl32)
			d100 := d97
			if d100.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d100)
			ctx.EmitStoreToStack(d100, 16)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d99.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl31)
				ctx.W.EmitJmp(lbl32)
				ctx.W.MarkLabel(lbl31)
				ctx.W.EmitJmp(lbl29)
				ctx.W.MarkLabel(lbl32)
			d101 := d97
			if d101.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d101)
			ctx.EmitStoreToStack(d101, 16)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d98)
			bbpos_4_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl30)
			ctx.W.ResolveFixups()
			d102 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d103 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d103 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r106 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r106, thisptr.Reg, off)
				d103 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r106}
				ctx.BindReg(r106, &d103)
			}
			ctx.EnsureDesc(&d103)
			ctx.EnsureDesc(&d103)
			var d104 scm.JITValueDesc
			if d103.Loc == scm.LocImm {
				d104 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d103.Imm.Int()))))}
			} else {
				r107 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r107, d103.Reg)
				ctx.W.EmitShlRegImm8(r107, 56)
				ctx.W.EmitShrRegImm8(r107, 56)
				d104 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r107}
				ctx.BindReg(r107, &d104)
			}
			ctx.FreeDesc(&d103)
			d105 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			ctx.EnsureDesc(&d105)
			ctx.EnsureDesc(&d104)
			var d106 scm.JITValueDesc
			if d105.Loc == scm.LocImm && d104.Loc == scm.LocImm {
				d106 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d105.Imm.Int() - d104.Imm.Int())}
			} else if d104.Loc == scm.LocImm && d104.Imm.Int() == 0 {
				r108 := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(r108, d105.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r108}
				ctx.BindReg(r108, &d106)
			} else if d105.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d104.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d105.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else if d104.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d105.Reg)
				ctx.W.EmitMovRegReg(scratch, d105.Reg)
				if d104.Imm.Int() >= -2147483648 && d104.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d104.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d104.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d106)
			} else {
				r109 := ctx.AllocRegExcept(d105.Reg, d104.Reg)
				ctx.W.EmitMovRegReg(r109, d105.Reg)
				ctx.W.EmitSubInt64(r109, d104.Reg)
				d106 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r109}
				ctx.BindReg(r109, &d106)
			}
			if d106.Loc == scm.LocReg && d105.Loc == scm.LocReg && d106.Reg == d105.Reg {
				ctx.TransferReg(d105.Reg)
				d105.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d104)
			ctx.EnsureDesc(&d102)
			ctx.EnsureDesc(&d106)
			var d107 scm.JITValueDesc
			if d102.Loc == scm.LocImm && d106.Loc == scm.LocImm {
				d107 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d102.Imm.Int()) >> uint64(d106.Imm.Int())))}
			} else if d106.Loc == scm.LocImm {
				r110 := ctx.AllocRegExcept(d102.Reg)
				ctx.W.EmitMovRegReg(r110, d102.Reg)
				ctx.W.EmitShrRegImm8(r110, uint8(d106.Imm.Int()))
				d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r110}
				ctx.BindReg(r110, &d107)
			} else {
				{
					shiftSrc := d102.Reg
					r111 := ctx.AllocRegExcept(d102.Reg)
					ctx.W.EmitMovRegReg(r111, d102.Reg)
					shiftSrc = r111
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d106.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d106.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d106.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d107 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d107)
				}
			}
			if d107.Loc == scm.LocReg && d102.Loc == scm.LocReg && d107.Reg == d102.Reg {
				ctx.TransferReg(d102.Reg)
				d102.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d102)
			ctx.FreeDesc(&d106)
			r112 := ctx.AllocReg()
			ctx.EnsureDesc(&d107)
			ctx.EnsureDesc(&d107)
			if d107.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r112, d107)
			}
			ctx.W.EmitJmp(lbl28)
			bbpos_4_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl29)
			ctx.W.ResolveFixups()
			d102 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d92)
			var d108 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d108 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() % 64)}
			} else {
				r113 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r113, d92.Reg)
				ctx.W.EmitAndRegImm32(r113, 63)
				d108 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r113}
				ctx.BindReg(r113, &d108)
			}
			if d108.Loc == scm.LocReg && d92.Loc == scm.LocReg && d108.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			var d109 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d109 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r114 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r114, thisptr.Reg, off)
				d109 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r114}
				ctx.BindReg(r114, &d109)
			}
			ctx.EnsureDesc(&d109)
			ctx.EnsureDesc(&d109)
			var d110 scm.JITValueDesc
			if d109.Loc == scm.LocImm {
				d110 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d109.Imm.Int()))))}
			} else {
				r115 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r115, d109.Reg)
				ctx.W.EmitShlRegImm8(r115, 56)
				ctx.W.EmitShrRegImm8(r115, 56)
				d110 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r115}
				ctx.BindReg(r115, &d110)
			}
			ctx.FreeDesc(&d109)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			ctx.EnsureDesc(&d108)
			ctx.EnsureDesc(&d110)
			var d111 scm.JITValueDesc
			if d108.Loc == scm.LocImm && d110.Loc == scm.LocImm {
				d111 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d108.Imm.Int() + d110.Imm.Int())}
			} else if d110.Loc == scm.LocImm && d110.Imm.Int() == 0 {
				r116 := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(r116, d108.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r116}
				ctx.BindReg(r116, &d111)
			} else if d108.Loc == scm.LocImm && d108.Imm.Int() == 0 {
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d110.Reg}
				ctx.BindReg(d110.Reg, &d111)
			} else if d108.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d110.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d108.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else if d110.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d108.Reg)
				ctx.W.EmitMovRegReg(scratch, d108.Reg)
				if d110.Imm.Int() >= -2147483648 && d110.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d110.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d110.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d111)
			} else {
				r117 := ctx.AllocRegExcept(d108.Reg, d110.Reg)
				ctx.W.EmitMovRegReg(r117, d108.Reg)
				ctx.W.EmitAddInt64(r117, d110.Reg)
				d111 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r117}
				ctx.BindReg(r117, &d111)
			}
			if d111.Loc == scm.LocReg && d108.Loc == scm.LocReg && d111.Reg == d108.Reg {
				ctx.TransferReg(d108.Reg)
				d108.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d108)
			ctx.FreeDesc(&d110)
			ctx.EnsureDesc(&d111)
			var d112 scm.JITValueDesc
			if d111.Loc == scm.LocImm {
				d112 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d111.Imm.Int()) > uint64(64))}
			} else {
				r118 := ctx.AllocRegExcept(d111.Reg)
				ctx.W.EmitCmpRegImm32(d111.Reg, 64)
				ctx.W.EmitSetcc(r118, scm.CcA)
				d112 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r118}
				ctx.BindReg(r118, &d112)
			}
			ctx.FreeDesc(&d111)
			d113 := d112
			ctx.EnsureDesc(&d113)
			if d113.Loc != scm.LocImm && d113.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl33 := ctx.W.ReserveLabel()
			lbl34 := ctx.W.ReserveLabel()
			lbl35 := ctx.W.ReserveLabel()
			if d113.Loc == scm.LocImm {
				if d113.Imm.Bool() {
					ctx.W.MarkLabel(lbl34)
					ctx.W.EmitJmp(lbl33)
				} else {
					ctx.W.MarkLabel(lbl35)
			d114 := d97
			if d114.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d114)
			ctx.EmitStoreToStack(d114, 16)
					ctx.W.EmitJmp(lbl30)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d113.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl34)
				ctx.W.EmitJmp(lbl35)
				ctx.W.MarkLabel(lbl34)
				ctx.W.EmitJmp(lbl33)
				ctx.W.MarkLabel(lbl35)
			d115 := d97
			if d115.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d115)
			ctx.EmitStoreToStack(d115, 16)
				ctx.W.EmitJmp(lbl30)
			}
			ctx.FreeDesc(&d112)
			bbpos_4_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl33)
			ctx.W.ResolveFixups()
			d102 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d92)
			var d116 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d116 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() / 64)}
			} else {
				r119 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r119, d92.Reg)
				ctx.W.EmitShrRegImm8(r119, 6)
				d116 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r119}
				ctx.BindReg(r119, &d116)
			}
			if d116.Loc == scm.LocReg && d92.Loc == scm.LocReg && d116.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d116)
			ctx.EnsureDesc(&d116)
			var d117 scm.JITValueDesc
			if d116.Loc == scm.LocImm {
				d117 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d116.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d116.Reg)
				ctx.W.EmitMovRegReg(scratch, d116.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d117 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d117)
			}
			if d117.Loc == scm.LocReg && d116.Loc == scm.LocReg && d117.Reg == d116.Reg {
				ctx.TransferReg(d116.Reg)
				d116.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d116)
			ctx.EnsureDesc(&d117)
			r120 := ctx.AllocReg()
			ctx.EnsureDesc(&d117)
			ctx.EnsureDesc(&d93)
			if d117.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r120, uint64(d117.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r120, d117.Reg)
				ctx.W.EmitShlRegImm8(r120, 3)
			}
			if d93.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d93.Imm.Int()))
				ctx.W.EmitAddInt64(r120, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r120, d93.Reg)
			}
			r121 := ctx.AllocRegExcept(r120)
			ctx.W.EmitMovRegMem(r121, r120, 0)
			ctx.FreeReg(r120)
			d118 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r121}
			ctx.BindReg(r121, &d118)
			ctx.FreeDesc(&d117)
			ctx.EnsureDesc(&d92)
			var d119 scm.JITValueDesc
			if d92.Loc == scm.LocImm {
				d119 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d92.Imm.Int() % 64)}
			} else {
				r122 := ctx.AllocRegExcept(d92.Reg)
				ctx.W.EmitMovRegReg(r122, d92.Reg)
				ctx.W.EmitAndRegImm32(r122, 63)
				d119 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r122}
				ctx.BindReg(r122, &d119)
			}
			if d119.Loc == scm.LocReg && d92.Loc == scm.LocReg && d119.Reg == d92.Reg {
				ctx.TransferReg(d92.Reg)
				d92.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d92)
			d120 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			ctx.EnsureDesc(&d120)
			ctx.EnsureDesc(&d119)
			var d121 scm.JITValueDesc
			if d120.Loc == scm.LocImm && d119.Loc == scm.LocImm {
				d121 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d120.Imm.Int() - d119.Imm.Int())}
			} else if d119.Loc == scm.LocImm && d119.Imm.Int() == 0 {
				r123 := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(r123, d120.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r123}
				ctx.BindReg(r123, &d121)
			} else if d120.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d119.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d120.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else if d119.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d120.Reg)
				ctx.W.EmitMovRegReg(scratch, d120.Reg)
				if d119.Imm.Int() >= -2147483648 && d119.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d119.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d119.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d121)
			} else {
				r124 := ctx.AllocRegExcept(d120.Reg, d119.Reg)
				ctx.W.EmitMovRegReg(r124, d120.Reg)
				ctx.W.EmitSubInt64(r124, d119.Reg)
				d121 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r124}
				ctx.BindReg(r124, &d121)
			}
			if d121.Loc == scm.LocReg && d120.Loc == scm.LocReg && d121.Reg == d120.Reg {
				ctx.TransferReg(d120.Reg)
				d120.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d119)
			ctx.EnsureDesc(&d118)
			ctx.EnsureDesc(&d121)
			var d122 scm.JITValueDesc
			if d118.Loc == scm.LocImm && d121.Loc == scm.LocImm {
				d122 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d118.Imm.Int()) >> uint64(d121.Imm.Int())))}
			} else if d121.Loc == scm.LocImm {
				r125 := ctx.AllocRegExcept(d118.Reg)
				ctx.W.EmitMovRegReg(r125, d118.Reg)
				ctx.W.EmitShrRegImm8(r125, uint8(d121.Imm.Int()))
				d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r125}
				ctx.BindReg(r125, &d122)
			} else {
				{
					shiftSrc := d118.Reg
					r126 := ctx.AllocRegExcept(d118.Reg)
					ctx.W.EmitMovRegReg(r126, d118.Reg)
					shiftSrc = r126
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d121.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d121.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d121.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d122 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d122)
				}
			}
			if d122.Loc == scm.LocReg && d118.Loc == scm.LocReg && d122.Reg == d118.Reg {
				ctx.TransferReg(d118.Reg)
				d118.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d118)
			ctx.FreeDesc(&d121)
			ctx.EnsureDesc(&d97)
			ctx.EnsureDesc(&d122)
			var d123 scm.JITValueDesc
			if d97.Loc == scm.LocImm && d122.Loc == scm.LocImm {
				d123 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d97.Imm.Int() | d122.Imm.Int())}
			} else if d97.Loc == scm.LocImm && d97.Imm.Int() == 0 {
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d122.Reg}
				ctx.BindReg(d122.Reg, &d123)
			} else if d122.Loc == scm.LocImm && d122.Imm.Int() == 0 {
				r127 := ctx.AllocRegExcept(d97.Reg)
				ctx.W.EmitMovRegReg(r127, d97.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r127}
				ctx.BindReg(r127, &d123)
			} else if d97.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d122.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d97.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d123)
			} else if d122.Loc == scm.LocImm {
				r128 := ctx.AllocRegExcept(d97.Reg)
				ctx.W.EmitMovRegReg(r128, d97.Reg)
				if d122.Imm.Int() >= -2147483648 && d122.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r128, int32(d122.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d122.Imm.Int()))
					ctx.W.EmitOrInt64(r128, scm.RegR11)
				}
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r128}
				ctx.BindReg(r128, &d123)
			} else {
				r129 := ctx.AllocRegExcept(d97.Reg, d122.Reg)
				ctx.W.EmitMovRegReg(r129, d97.Reg)
				ctx.W.EmitOrInt64(r129, d122.Reg)
				d123 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r129}
				ctx.BindReg(r129, &d123)
			}
			if d123.Loc == scm.LocReg && d97.Loc == scm.LocReg && d123.Reg == d97.Reg {
				ctx.TransferReg(d97.Reg)
				d97.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d122)
			d124 := d123
			if d124.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d124)
			ctx.EmitStoreToStack(d124, 16)
			ctx.W.EmitJmp(lbl30)
			ctx.W.MarkLabel(lbl28)
			d125 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r112}
			ctx.BindReg(r112, &d125)
			ctx.BindReg(r112, &d125)
			if r92 { ctx.UnprotectReg(r93) }
			ctx.EnsureDesc(&d125)
			ctx.EnsureDesc(&d125)
			var d126 scm.JITValueDesc
			if d125.Loc == scm.LocImm {
				d126 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d125.Imm.Int()))))}
			} else {
				r130 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r130, d125.Reg)
				d126 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r130}
				ctx.BindReg(r130, &d126)
			}
			ctx.FreeDesc(&d125)
			var d127 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d127 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r131 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r131, thisptr.Reg, off)
				d127 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r131}
				ctx.BindReg(r131, &d127)
			}
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			ctx.EnsureDesc(&d126)
			ctx.EnsureDesc(&d127)
			var d128 scm.JITValueDesc
			if d126.Loc == scm.LocImm && d127.Loc == scm.LocImm {
				d128 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d126.Imm.Int() + d127.Imm.Int())}
			} else if d127.Loc == scm.LocImm && d127.Imm.Int() == 0 {
				r132 := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(r132, d126.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r132}
				ctx.BindReg(r132, &d128)
			} else if d126.Loc == scm.LocImm && d126.Imm.Int() == 0 {
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d127.Reg}
				ctx.BindReg(d127.Reg, &d128)
			} else if d126.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d127.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d126.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else if d127.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d126.Reg)
				ctx.W.EmitMovRegReg(scratch, d126.Reg)
				if d127.Imm.Int() >= -2147483648 && d127.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d127.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d127.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d128)
			} else {
				r133 := ctx.AllocRegExcept(d126.Reg, d127.Reg)
				ctx.W.EmitMovRegReg(r133, d126.Reg)
				ctx.W.EmitAddInt64(r133, d127.Reg)
				d128 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r133}
				ctx.BindReg(r133, &d128)
			}
			if d128.Loc == scm.LocReg && d126.Loc == scm.LocReg && d128.Reg == d126.Reg {
				ctx.TransferReg(d126.Reg)
				d126.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d126)
			ctx.FreeDesc(&d127)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d128)
			var d130 scm.JITValueDesc
			if d87.Loc == scm.LocImm && d128.Loc == scm.LocImm {
				d130 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d87.Imm.Int() + d128.Imm.Int())}
			} else if d128.Loc == scm.LocImm && d128.Imm.Int() == 0 {
				r134 := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(r134, d87.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r134}
				ctx.BindReg(r134, &d130)
			} else if d87.Loc == scm.LocImm && d87.Imm.Int() == 0 {
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d128.Reg}
				ctx.BindReg(d128.Reg, &d130)
			} else if d87.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d128.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d87.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else if d128.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d87.Reg)
				ctx.W.EmitMovRegReg(scratch, d87.Reg)
				if d128.Imm.Int() >= -2147483648 && d128.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d128.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d128.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d130)
			} else {
				r135 := ctx.AllocRegExcept(d87.Reg, d128.Reg)
				ctx.W.EmitMovRegReg(r135, d87.Reg)
				ctx.W.EmitAddInt64(r135, d128.Reg)
				d130 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r135}
				ctx.BindReg(r135, &d130)
			}
			if d130.Loc == scm.LocReg && d87.Loc == scm.LocReg && d130.Reg == d87.Reg {
				ctx.TransferReg(d87.Reg)
				d87.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d128)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d130)
			var d132 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 72
				r136 := ctx.AllocReg()
				r137 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r136, fieldAddr)
				ctx.W.EmitMovRegMem64(r137, fieldAddr+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r136, Reg2: r137}
				ctx.BindReg(r136, &d132)
				ctx.BindReg(r137, &d132)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 72)
				r138 := ctx.AllocReg()
				r139 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r138, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r139, thisptr.Reg, off+8)
				d132 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r138, Reg2: r139}
				ctx.BindReg(r138, &d132)
				ctx.BindReg(r139, &d132)
			}
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			r140 := ctx.AllocReg()
			r141 := ctx.AllocRegExcept(r140)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d87)
			ctx.EnsureDesc(&d130)
			if d132.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r140, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r140, d132.Reg)
			} else {
				ctx.W.EmitMovRegReg(r140, d132.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() != 0 {
					if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r140, int32(d87.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
						ctx.W.EmitAddInt64(r140, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r140, d87.Reg)
			}
			if d130.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r141, uint64(d130.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r141, d130.Reg)
			}
			if d87.Loc == scm.LocImm {
				if d87.Imm.Int() >= -2147483648 && d87.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r141, int32(d87.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d87.Imm.Int()))
					ctx.W.EmitSubInt64(r141, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r141, d87.Reg)
			}
			d133 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r140, Reg2: r141}
			ctx.BindReg(r140, &d133)
			ctx.BindReg(r141, &d133)
			ctx.FreeDesc(&d87)
			ctx.FreeDesc(&d130)
			r142 := ctx.AllocReg()
			r143 := ctx.AllocRegExcept(r142)
			d134 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d134)
			ctx.BindReg(r143, &d134)
			d135 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d133}, 2)
			ctx.EmitMovPairToResult(&d135, &d134)
			ctx.W.EmitJmp(lbl3)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl4)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d136 := d0
			_ = d136
			r144 := d0.Loc == scm.LocReg
			r145 := d0.Reg
			if r144 { ctx.ProtectReg(r145) }
			lbl36 := ctx.W.ReserveLabel()
			bbpos_5_0 := int32(-1)
			_ = bbpos_5_0
			bbpos_5_1 := int32(-1)
			_ = bbpos_5_1
			bbpos_5_2 := int32(-1)
			_ = bbpos_5_2
			bbpos_5_3 := int32(-1)
			_ = bbpos_5_3
			bbpos_5_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d136)
			ctx.EnsureDesc(&d136)
			var d137 scm.JITValueDesc
			if d136.Loc == scm.LocImm {
				d137 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d136.Imm.Int()))))}
			} else {
				r146 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r146, d136.Reg)
				ctx.W.EmitShlRegImm8(r146, 32)
				ctx.W.EmitShrRegImm8(r146, 32)
				d137 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r146}
				ctx.BindReg(r146, &d137)
			}
			var d138 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d138 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r147 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r147, thisptr.Reg, off)
				d138 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r147}
				ctx.BindReg(r147, &d138)
			}
			ctx.EnsureDesc(&d138)
			ctx.EnsureDesc(&d138)
			var d139 scm.JITValueDesc
			if d138.Loc == scm.LocImm {
				d139 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d138.Imm.Int()))))}
			} else {
				r148 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r148, d138.Reg)
				ctx.W.EmitShlRegImm8(r148, 56)
				ctx.W.EmitShrRegImm8(r148, 56)
				d139 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r148}
				ctx.BindReg(r148, &d139)
			}
			ctx.FreeDesc(&d138)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d139)
			ctx.EnsureDesc(&d137)
			ctx.EnsureDesc(&d139)
			var d140 scm.JITValueDesc
			if d137.Loc == scm.LocImm && d139.Loc == scm.LocImm {
				d140 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d137.Imm.Int() * d139.Imm.Int())}
			} else if d137.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d139.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d137.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else if d139.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d137.Reg)
				ctx.W.EmitMovRegReg(scratch, d137.Reg)
				if d139.Imm.Int() >= -2147483648 && d139.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d139.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d139.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d140)
			} else {
				r149 := ctx.AllocRegExcept(d137.Reg, d139.Reg)
				ctx.W.EmitMovRegReg(r149, d137.Reg)
				ctx.W.EmitImulInt64(r149, d139.Reg)
				d140 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r149}
				ctx.BindReg(r149, &d140)
			}
			if d140.Loc == scm.LocReg && d137.Loc == scm.LocReg && d140.Reg == d137.Reg {
				ctx.TransferReg(d137.Reg)
				d137.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d137)
			ctx.FreeDesc(&d139)
			var d141 scm.JITValueDesc
			r150 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r150, uint64(dataPtr))
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150, StackOff: int32(sliceLen)}
				ctx.BindReg(r150, &d141)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 0)
				ctx.W.EmitMovRegMem(r150, thisptr.Reg, off)
				d141 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r150}
				ctx.BindReg(r150, &d141)
			}
			ctx.BindReg(r150, &d141)
			ctx.EnsureDesc(&d140)
			var d142 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d142 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() / 64)}
			} else {
				r151 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r151, d140.Reg)
				ctx.W.EmitShrRegImm8(r151, 6)
				d142 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r151}
				ctx.BindReg(r151, &d142)
			}
			if d142.Loc == scm.LocReg && d140.Loc == scm.LocReg && d142.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d142)
			r152 := ctx.AllocReg()
			ctx.EnsureDesc(&d142)
			ctx.EnsureDesc(&d141)
			if d142.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r152, uint64(d142.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r152, d142.Reg)
				ctx.W.EmitShlRegImm8(r152, 3)
			}
			if d141.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.W.EmitAddInt64(r152, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r152, d141.Reg)
			}
			r153 := ctx.AllocRegExcept(r152)
			ctx.W.EmitMovRegMem(r153, r152, 0)
			ctx.FreeReg(r152)
			d143 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r153}
			ctx.BindReg(r153, &d143)
			ctx.FreeDesc(&d142)
			ctx.EnsureDesc(&d140)
			var d144 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d144 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
			} else {
				r154 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r154, d140.Reg)
				ctx.W.EmitAndRegImm32(r154, 63)
				d144 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r154}
				ctx.BindReg(r154, &d144)
			}
			if d144.Loc == scm.LocReg && d140.Loc == scm.LocReg && d144.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d143)
			ctx.EnsureDesc(&d144)
			var d145 scm.JITValueDesc
			if d143.Loc == scm.LocImm && d144.Loc == scm.LocImm {
				d145 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d143.Imm.Int()) << uint64(d144.Imm.Int())))}
			} else if d144.Loc == scm.LocImm {
				r155 := ctx.AllocRegExcept(d143.Reg)
				ctx.W.EmitMovRegReg(r155, d143.Reg)
				ctx.W.EmitShlRegImm8(r155, uint8(d144.Imm.Int()))
				d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r155}
				ctx.BindReg(r155, &d145)
			} else {
				{
					shiftSrc := d143.Reg
					r156 := ctx.AllocRegExcept(d143.Reg)
					ctx.W.EmitMovRegReg(r156, d143.Reg)
					shiftSrc = r156
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d144.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d144.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d144.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d145 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d145)
				}
			}
			if d145.Loc == scm.LocReg && d143.Loc == scm.LocReg && d145.Reg == d143.Reg {
				ctx.TransferReg(d143.Reg)
				d143.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d143)
			ctx.FreeDesc(&d144)
			var d146 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d146 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 25)
				r157 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r157, thisptr.Reg, off)
				d146 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r157}
				ctx.BindReg(r157, &d146)
			}
			d147 := d146
			ctx.EnsureDesc(&d147)
			if d147.Loc != scm.LocImm && d147.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl37 := ctx.W.ReserveLabel()
			lbl38 := ctx.W.ReserveLabel()
			lbl39 := ctx.W.ReserveLabel()
			lbl40 := ctx.W.ReserveLabel()
			if d147.Loc == scm.LocImm {
				if d147.Imm.Bool() {
					ctx.W.MarkLabel(lbl39)
					ctx.W.EmitJmp(lbl37)
				} else {
					ctx.W.MarkLabel(lbl40)
			d148 := d145
			if d148.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d148)
			ctx.EmitStoreToStack(d148, 24)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d147.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl39)
				ctx.W.EmitJmp(lbl40)
				ctx.W.MarkLabel(lbl39)
				ctx.W.EmitJmp(lbl37)
				ctx.W.MarkLabel(lbl40)
			d149 := d145
			if d149.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d149)
			ctx.EmitStoreToStack(d149, 24)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d146)
			bbpos_5_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl38)
			ctx.W.ResolveFixups()
			d150 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			var d151 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r158 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r158, thisptr.Reg, off)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r158}
				ctx.BindReg(r158, &d151)
			}
			ctx.EnsureDesc(&d151)
			ctx.EnsureDesc(&d151)
			var d152 scm.JITValueDesc
			if d151.Loc == scm.LocImm {
				d152 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d151.Imm.Int()))))}
			} else {
				r159 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r159, d151.Reg)
				ctx.W.EmitShlRegImm8(r159, 56)
				ctx.W.EmitShrRegImm8(r159, 56)
				d152 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r159}
				ctx.BindReg(r159, &d152)
			}
			ctx.FreeDesc(&d151)
			d153 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			ctx.EnsureDesc(&d153)
			ctx.EnsureDesc(&d152)
			var d154 scm.JITValueDesc
			if d153.Loc == scm.LocImm && d152.Loc == scm.LocImm {
				d154 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d153.Imm.Int() - d152.Imm.Int())}
			} else if d152.Loc == scm.LocImm && d152.Imm.Int() == 0 {
				r160 := ctx.AllocRegExcept(d153.Reg)
				ctx.W.EmitMovRegReg(r160, d153.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r160}
				ctx.BindReg(r160, &d154)
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
				r161 := ctx.AllocRegExcept(d153.Reg, d152.Reg)
				ctx.W.EmitMovRegReg(r161, d153.Reg)
				ctx.W.EmitSubInt64(r161, d152.Reg)
				d154 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r161}
				ctx.BindReg(r161, &d154)
			}
			if d154.Loc == scm.LocReg && d153.Loc == scm.LocReg && d154.Reg == d153.Reg {
				ctx.TransferReg(d153.Reg)
				d153.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d152)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d154)
			var d155 scm.JITValueDesc
			if d150.Loc == scm.LocImm && d154.Loc == scm.LocImm {
				d155 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d150.Imm.Int()) >> uint64(d154.Imm.Int())))}
			} else if d154.Loc == scm.LocImm {
				r162 := ctx.AllocRegExcept(d150.Reg)
				ctx.W.EmitMovRegReg(r162, d150.Reg)
				ctx.W.EmitShrRegImm8(r162, uint8(d154.Imm.Int()))
				d155 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r162}
				ctx.BindReg(r162, &d155)
			} else {
				{
					shiftSrc := d150.Reg
					r163 := ctx.AllocRegExcept(d150.Reg)
					ctx.W.EmitMovRegReg(r163, d150.Reg)
					shiftSrc = r163
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
			if d155.Loc == scm.LocReg && d150.Loc == scm.LocReg && d155.Reg == d150.Reg {
				ctx.TransferReg(d150.Reg)
				d150.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d150)
			ctx.FreeDesc(&d154)
			r164 := ctx.AllocReg()
			ctx.EnsureDesc(&d155)
			ctx.EnsureDesc(&d155)
			if d155.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r164, d155)
			}
			ctx.W.EmitJmp(lbl36)
			bbpos_5_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl37)
			ctx.W.ResolveFixups()
			d150 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d140)
			var d156 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d156 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
			} else {
				r165 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r165, d140.Reg)
				ctx.W.EmitAndRegImm32(r165, 63)
				d156 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r165}
				ctx.BindReg(r165, &d156)
			}
			if d156.Loc == scm.LocReg && d140.Loc == scm.LocReg && d156.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			var d157 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d157 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 24)
				r166 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r166, thisptr.Reg, off)
				d157 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r166}
				ctx.BindReg(r166, &d157)
			}
			ctx.EnsureDesc(&d157)
			ctx.EnsureDesc(&d157)
			var d158 scm.JITValueDesc
			if d157.Loc == scm.LocImm {
				d158 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d157.Imm.Int()))))}
			} else {
				r167 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r167, d157.Reg)
				ctx.W.EmitShlRegImm8(r167, 56)
				ctx.W.EmitShrRegImm8(r167, 56)
				d158 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r167}
				ctx.BindReg(r167, &d158)
			}
			ctx.FreeDesc(&d157)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			ctx.EnsureDesc(&d156)
			ctx.EnsureDesc(&d158)
			var d159 scm.JITValueDesc
			if d156.Loc == scm.LocImm && d158.Loc == scm.LocImm {
				d159 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d156.Imm.Int() + d158.Imm.Int())}
			} else if d158.Loc == scm.LocImm && d158.Imm.Int() == 0 {
				r168 := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(r168, d156.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r168}
				ctx.BindReg(r168, &d159)
			} else if d156.Loc == scm.LocImm && d156.Imm.Int() == 0 {
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d158.Reg}
				ctx.BindReg(d158.Reg, &d159)
			} else if d156.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d158.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d156.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else if d158.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d156.Reg)
				ctx.W.EmitMovRegReg(scratch, d156.Reg)
				if d158.Imm.Int() >= -2147483648 && d158.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d158.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d158.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d159)
			} else {
				r169 := ctx.AllocRegExcept(d156.Reg, d158.Reg)
				ctx.W.EmitMovRegReg(r169, d156.Reg)
				ctx.W.EmitAddInt64(r169, d158.Reg)
				d159 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r169}
				ctx.BindReg(r169, &d159)
			}
			if d159.Loc == scm.LocReg && d156.Loc == scm.LocReg && d159.Reg == d156.Reg {
				ctx.TransferReg(d156.Reg)
				d156.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d156)
			ctx.FreeDesc(&d158)
			ctx.EnsureDesc(&d159)
			var d160 scm.JITValueDesc
			if d159.Loc == scm.LocImm {
				d160 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d159.Imm.Int()) > uint64(64))}
			} else {
				r170 := ctx.AllocRegExcept(d159.Reg)
				ctx.W.EmitCmpRegImm32(d159.Reg, 64)
				ctx.W.EmitSetcc(r170, scm.CcA)
				d160 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r170}
				ctx.BindReg(r170, &d160)
			}
			ctx.FreeDesc(&d159)
			d161 := d160
			ctx.EnsureDesc(&d161)
			if d161.Loc != scm.LocImm && d161.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl41 := ctx.W.ReserveLabel()
			lbl42 := ctx.W.ReserveLabel()
			lbl43 := ctx.W.ReserveLabel()
			if d161.Loc == scm.LocImm {
				if d161.Imm.Bool() {
					ctx.W.MarkLabel(lbl42)
					ctx.W.EmitJmp(lbl41)
				} else {
					ctx.W.MarkLabel(lbl43)
			d162 := d145
			if d162.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d162)
			ctx.EmitStoreToStack(d162, 24)
					ctx.W.EmitJmp(lbl38)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d161.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl42)
				ctx.W.EmitJmp(lbl43)
				ctx.W.MarkLabel(lbl42)
				ctx.W.EmitJmp(lbl41)
				ctx.W.MarkLabel(lbl43)
			d163 := d145
			if d163.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d163)
			ctx.EmitStoreToStack(d163, 24)
				ctx.W.EmitJmp(lbl38)
			}
			ctx.FreeDesc(&d160)
			bbpos_5_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl41)
			ctx.W.ResolveFixups()
			d150 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(24)}
			ctx.EnsureDesc(&d140)
			var d164 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d164 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() / 64)}
			} else {
				r171 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r171, d140.Reg)
				ctx.W.EmitShrRegImm8(r171, 6)
				d164 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r171}
				ctx.BindReg(r171, &d164)
			}
			if d164.Loc == scm.LocReg && d140.Loc == scm.LocReg && d164.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d164)
			ctx.EnsureDesc(&d164)
			var d165 scm.JITValueDesc
			if d164.Loc == scm.LocImm {
				d165 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d164.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d164.Reg)
				ctx.W.EmitMovRegReg(scratch, d164.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d165 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d165)
			}
			if d165.Loc == scm.LocReg && d164.Loc == scm.LocReg && d165.Reg == d164.Reg {
				ctx.TransferReg(d164.Reg)
				d164.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d164)
			ctx.EnsureDesc(&d165)
			r172 := ctx.AllocReg()
			ctx.EnsureDesc(&d165)
			ctx.EnsureDesc(&d141)
			if d165.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r172, uint64(d165.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r172, d165.Reg)
				ctx.W.EmitShlRegImm8(r172, 3)
			}
			if d141.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d141.Imm.Int()))
				ctx.W.EmitAddInt64(r172, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r172, d141.Reg)
			}
			r173 := ctx.AllocRegExcept(r172)
			ctx.W.EmitMovRegMem(r173, r172, 0)
			ctx.FreeReg(r172)
			d166 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r173}
			ctx.BindReg(r173, &d166)
			ctx.FreeDesc(&d165)
			ctx.EnsureDesc(&d140)
			var d167 scm.JITValueDesc
			if d140.Loc == scm.LocImm {
				d167 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d140.Imm.Int() % 64)}
			} else {
				r174 := ctx.AllocRegExcept(d140.Reg)
				ctx.W.EmitMovRegReg(r174, d140.Reg)
				ctx.W.EmitAndRegImm32(r174, 63)
				d167 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r174}
				ctx.BindReg(r174, &d167)
			}
			if d167.Loc == scm.LocReg && d140.Loc == scm.LocReg && d167.Reg == d140.Reg {
				ctx.TransferReg(d140.Reg)
				d140.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d140)
			d168 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			ctx.EnsureDesc(&d168)
			ctx.EnsureDesc(&d167)
			var d169 scm.JITValueDesc
			if d168.Loc == scm.LocImm && d167.Loc == scm.LocImm {
				d169 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d168.Imm.Int() - d167.Imm.Int())}
			} else if d167.Loc == scm.LocImm && d167.Imm.Int() == 0 {
				r175 := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(r175, d168.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r175}
				ctx.BindReg(r175, &d169)
			} else if d168.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d167.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d168.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else if d167.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d168.Reg)
				ctx.W.EmitMovRegReg(scratch, d168.Reg)
				if d167.Imm.Int() >= -2147483648 && d167.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d167.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d167.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d169)
			} else {
				r176 := ctx.AllocRegExcept(d168.Reg, d167.Reg)
				ctx.W.EmitMovRegReg(r176, d168.Reg)
				ctx.W.EmitSubInt64(r176, d167.Reg)
				d169 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r176}
				ctx.BindReg(r176, &d169)
			}
			if d169.Loc == scm.LocReg && d168.Loc == scm.LocReg && d169.Reg == d168.Reg {
				ctx.TransferReg(d168.Reg)
				d168.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d167)
			ctx.EnsureDesc(&d166)
			ctx.EnsureDesc(&d169)
			var d170 scm.JITValueDesc
			if d166.Loc == scm.LocImm && d169.Loc == scm.LocImm {
				d170 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d166.Imm.Int()) >> uint64(d169.Imm.Int())))}
			} else if d169.Loc == scm.LocImm {
				r177 := ctx.AllocRegExcept(d166.Reg)
				ctx.W.EmitMovRegReg(r177, d166.Reg)
				ctx.W.EmitShrRegImm8(r177, uint8(d169.Imm.Int()))
				d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r177}
				ctx.BindReg(r177, &d170)
			} else {
				{
					shiftSrc := d166.Reg
					r178 := ctx.AllocRegExcept(d166.Reg)
					ctx.W.EmitMovRegReg(r178, d166.Reg)
					shiftSrc = r178
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d169.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d169.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d169.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d170 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d170)
				}
			}
			if d170.Loc == scm.LocReg && d166.Loc == scm.LocReg && d170.Reg == d166.Reg {
				ctx.TransferReg(d166.Reg)
				d166.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d166)
			ctx.FreeDesc(&d169)
			ctx.EnsureDesc(&d145)
			ctx.EnsureDesc(&d170)
			var d171 scm.JITValueDesc
			if d145.Loc == scm.LocImm && d170.Loc == scm.LocImm {
				d171 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d145.Imm.Int() | d170.Imm.Int())}
			} else if d145.Loc == scm.LocImm && d145.Imm.Int() == 0 {
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d170.Reg}
				ctx.BindReg(d170.Reg, &d171)
			} else if d170.Loc == scm.LocImm && d170.Imm.Int() == 0 {
				r179 := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegReg(r179, d145.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r179}
				ctx.BindReg(r179, &d171)
			} else if d145.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d170.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d145.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d171)
			} else if d170.Loc == scm.LocImm {
				r180 := ctx.AllocRegExcept(d145.Reg)
				ctx.W.EmitMovRegReg(r180, d145.Reg)
				if d170.Imm.Int() >= -2147483648 && d170.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r180, int32(d170.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d170.Imm.Int()))
					ctx.W.EmitOrInt64(r180, scm.RegR11)
				}
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r180}
				ctx.BindReg(r180, &d171)
			} else {
				r181 := ctx.AllocRegExcept(d145.Reg, d170.Reg)
				ctx.W.EmitMovRegReg(r181, d145.Reg)
				ctx.W.EmitOrInt64(r181, d170.Reg)
				d171 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r181}
				ctx.BindReg(r181, &d171)
			}
			if d171.Loc == scm.LocReg && d145.Loc == scm.LocReg && d171.Reg == d145.Reg {
				ctx.TransferReg(d145.Reg)
				d145.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d170)
			d172 := d171
			if d172.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d172)
			ctx.EmitStoreToStack(d172, 24)
			ctx.W.EmitJmp(lbl38)
			ctx.W.MarkLabel(lbl36)
			d173 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r164}
			ctx.BindReg(r164, &d173)
			ctx.BindReg(r164, &d173)
			if r144 { ctx.UnprotectReg(r145) }
			ctx.EnsureDesc(&d173)
			ctx.EnsureDesc(&d173)
			var d174 scm.JITValueDesc
			if d173.Loc == scm.LocImm {
				d174 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d173.Imm.Int()))))}
			} else {
				r182 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r182, d173.Reg)
				d174 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r182}
				ctx.BindReg(r182, &d174)
			}
			ctx.FreeDesc(&d173)
			var d175 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d175 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 32)
				r183 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r183, thisptr.Reg, off)
				d175 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r183}
				ctx.BindReg(r183, &d175)
			}
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			ctx.EnsureDesc(&d174)
			ctx.EnsureDesc(&d175)
			var d176 scm.JITValueDesc
			if d174.Loc == scm.LocImm && d175.Loc == scm.LocImm {
				d176 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d174.Imm.Int() + d175.Imm.Int())}
			} else if d175.Loc == scm.LocImm && d175.Imm.Int() == 0 {
				r184 := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(r184, d174.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r184}
				ctx.BindReg(r184, &d176)
			} else if d174.Loc == scm.LocImm && d174.Imm.Int() == 0 {
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d175.Reg}
				ctx.BindReg(d175.Reg, &d176)
			} else if d174.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d175.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d174.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else if d175.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d174.Reg)
				ctx.W.EmitMovRegReg(scratch, d174.Reg)
				if d175.Imm.Int() >= -2147483648 && d175.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d175.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d175.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d176)
			} else {
				r185 := ctx.AllocRegExcept(d174.Reg, d175.Reg)
				ctx.W.EmitMovRegReg(r185, d174.Reg)
				ctx.W.EmitAddInt64(r185, d175.Reg)
				d176 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r185}
				ctx.BindReg(r185, &d176)
			}
			if d176.Loc == scm.LocReg && d174.Loc == scm.LocReg && d176.Reg == d174.Reg {
				ctx.TransferReg(d174.Reg)
				d174.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d174)
			ctx.FreeDesc(&d175)
			ctx.EnsureDesc(&d176)
			ctx.EnsureDesc(&d176)
			var d177 scm.JITValueDesc
			if d176.Loc == scm.LocImm {
				d177 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d176.Imm.Int()))))}
			} else {
				r186 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r186, d176.Reg)
				d177 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r186}
				ctx.BindReg(r186, &d177)
			}
			ctx.FreeDesc(&d176)
			var d178 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d178 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 88 + 56)
				r187 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r187, thisptr.Reg, off)
				d178 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r187}
				ctx.BindReg(r187, &d178)
			}
			d179 := d178
			ctx.EnsureDesc(&d179)
			if d179.Loc != scm.LocImm && d179.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl44 := ctx.W.ReserveLabel()
			lbl45 := ctx.W.ReserveLabel()
			lbl46 := ctx.W.ReserveLabel()
			lbl47 := ctx.W.ReserveLabel()
			if d179.Loc == scm.LocImm {
				if d179.Imm.Bool() {
					ctx.W.MarkLabel(lbl46)
					ctx.W.EmitJmp(lbl44)
				} else {
					ctx.W.MarkLabel(lbl47)
					ctx.W.EmitJmp(lbl45)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d179.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl46)
				ctx.W.EmitJmp(lbl47)
				ctx.W.MarkLabel(lbl46)
				ctx.W.EmitJmp(lbl44)
				ctx.W.MarkLabel(lbl47)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d178)
			bbpos_1_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl45)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&d0)
			d180 := d0
			_ = d180
			r188 := d0.Loc == scm.LocReg
			r189 := d0.Reg
			if r188 { ctx.ProtectReg(r189) }
			lbl48 := ctx.W.ReserveLabel()
			bbpos_6_0 := int32(-1)
			_ = bbpos_6_0
			bbpos_6_1 := int32(-1)
			_ = bbpos_6_1
			bbpos_6_2 := int32(-1)
			_ = bbpos_6_2
			bbpos_6_3 := int32(-1)
			_ = bbpos_6_3
			bbpos_6_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.EnsureDesc(&d180)
			ctx.EnsureDesc(&d180)
			var d181 scm.JITValueDesc
			if d180.Loc == scm.LocImm {
				d181 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d180.Imm.Int()))))}
			} else {
				r190 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r190, d180.Reg)
				ctx.W.EmitShlRegImm8(r190, 32)
				ctx.W.EmitShrRegImm8(r190, 32)
				d181 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r190}
				ctx.BindReg(r190, &d181)
			}
			var d182 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d182 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r191 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r191, thisptr.Reg, off)
				d182 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r191}
				ctx.BindReg(r191, &d182)
			}
			ctx.EnsureDesc(&d182)
			ctx.EnsureDesc(&d182)
			var d183 scm.JITValueDesc
			if d182.Loc == scm.LocImm {
				d183 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d182.Imm.Int()))))}
			} else {
				r192 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r192, d182.Reg)
				ctx.W.EmitShlRegImm8(r192, 56)
				ctx.W.EmitShrRegImm8(r192, 56)
				d183 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r192}
				ctx.BindReg(r192, &d183)
			}
			ctx.FreeDesc(&d182)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d183)
			ctx.EnsureDesc(&d181)
			ctx.EnsureDesc(&d183)
			var d184 scm.JITValueDesc
			if d181.Loc == scm.LocImm && d183.Loc == scm.LocImm {
				d184 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d181.Imm.Int() * d183.Imm.Int())}
			} else if d181.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d183.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d181.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d183.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d184)
			} else if d183.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d181.Reg)
				ctx.W.EmitMovRegReg(scratch, d181.Reg)
				if d183.Imm.Int() >= -2147483648 && d183.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d183.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d183.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d184)
			} else {
				r193 := ctx.AllocRegExcept(d181.Reg, d183.Reg)
				ctx.W.EmitMovRegReg(r193, d181.Reg)
				ctx.W.EmitImulInt64(r193, d183.Reg)
				d184 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r193}
				ctx.BindReg(r193, &d184)
			}
			if d184.Loc == scm.LocReg && d181.Loc == scm.LocReg && d184.Reg == d181.Reg {
				ctx.TransferReg(d181.Reg)
				d181.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d181)
			ctx.FreeDesc(&d183)
			var d185 scm.JITValueDesc
			r194 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r194, uint64(dataPtr))
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194, StackOff: int32(sliceLen)}
				ctx.BindReg(r194, &d185)
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 0)
				ctx.W.EmitMovRegMem(r194, thisptr.Reg, off)
				d185 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r194}
				ctx.BindReg(r194, &d185)
			}
			ctx.BindReg(r194, &d185)
			ctx.EnsureDesc(&d184)
			var d186 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d186 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() / 64)}
			} else {
				r195 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r195, d184.Reg)
				ctx.W.EmitShrRegImm8(r195, 6)
				d186 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r195}
				ctx.BindReg(r195, &d186)
			}
			if d186.Loc == scm.LocReg && d184.Loc == scm.LocReg && d186.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d186)
			r196 := ctx.AllocReg()
			ctx.EnsureDesc(&d186)
			ctx.EnsureDesc(&d185)
			if d186.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r196, uint64(d186.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r196, d186.Reg)
				ctx.W.EmitShlRegImm8(r196, 3)
			}
			if d185.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
				ctx.W.EmitAddInt64(r196, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r196, d185.Reg)
			}
			r197 := ctx.AllocRegExcept(r196)
			ctx.W.EmitMovRegMem(r197, r196, 0)
			ctx.FreeReg(r196)
			d187 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r197}
			ctx.BindReg(r197, &d187)
			ctx.FreeDesc(&d186)
			ctx.EnsureDesc(&d184)
			var d188 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d188 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() % 64)}
			} else {
				r198 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r198, d184.Reg)
				ctx.W.EmitAndRegImm32(r198, 63)
				d188 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r198}
				ctx.BindReg(r198, &d188)
			}
			if d188.Loc == scm.LocReg && d184.Loc == scm.LocReg && d188.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d187)
			ctx.EnsureDesc(&d188)
			var d189 scm.JITValueDesc
			if d187.Loc == scm.LocImm && d188.Loc == scm.LocImm {
				d189 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d187.Imm.Int()) << uint64(d188.Imm.Int())))}
			} else if d188.Loc == scm.LocImm {
				r199 := ctx.AllocRegExcept(d187.Reg)
				ctx.W.EmitMovRegReg(r199, d187.Reg)
				ctx.W.EmitShlRegImm8(r199, uint8(d188.Imm.Int()))
				d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r199}
				ctx.BindReg(r199, &d189)
			} else {
				{
					shiftSrc := d187.Reg
					r200 := ctx.AllocRegExcept(d187.Reg)
					ctx.W.EmitMovRegReg(r200, d187.Reg)
					shiftSrc = r200
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d188.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d188.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d188.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d189 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d189)
				}
			}
			if d189.Loc == scm.LocReg && d187.Loc == scm.LocReg && d189.Reg == d187.Reg {
				ctx.TransferReg(d187.Reg)
				d187.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d187)
			ctx.FreeDesc(&d188)
			var d190 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d190 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 25)
				r201 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r201, thisptr.Reg, off)
				d190 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r201}
				ctx.BindReg(r201, &d190)
			}
			d191 := d190
			ctx.EnsureDesc(&d191)
			if d191.Loc != scm.LocImm && d191.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl49 := ctx.W.ReserveLabel()
			lbl50 := ctx.W.ReserveLabel()
			lbl51 := ctx.W.ReserveLabel()
			lbl52 := ctx.W.ReserveLabel()
			if d191.Loc == scm.LocImm {
				if d191.Imm.Bool() {
					ctx.W.MarkLabel(lbl51)
					ctx.W.EmitJmp(lbl49)
				} else {
					ctx.W.MarkLabel(lbl52)
			d192 := d189
			if d192.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d192)
			ctx.EmitStoreToStack(d192, 32)
					ctx.W.EmitJmp(lbl50)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d191.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl51)
				ctx.W.EmitJmp(lbl52)
				ctx.W.MarkLabel(lbl51)
				ctx.W.EmitJmp(lbl49)
				ctx.W.MarkLabel(lbl52)
			d193 := d189
			if d193.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d193)
			ctx.EmitStoreToStack(d193, 32)
				ctx.W.EmitJmp(lbl50)
			}
			ctx.FreeDesc(&d190)
			bbpos_6_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl50)
			ctx.W.ResolveFixups()
			d194 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			var d195 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d195 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r202 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r202, thisptr.Reg, off)
				d195 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r202}
				ctx.BindReg(r202, &d195)
			}
			ctx.EnsureDesc(&d195)
			ctx.EnsureDesc(&d195)
			var d196 scm.JITValueDesc
			if d195.Loc == scm.LocImm {
				d196 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d195.Imm.Int()))))}
			} else {
				r203 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r203, d195.Reg)
				ctx.W.EmitShlRegImm8(r203, 56)
				ctx.W.EmitShrRegImm8(r203, 56)
				d196 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r203}
				ctx.BindReg(r203, &d196)
			}
			ctx.FreeDesc(&d195)
			d197 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			ctx.EnsureDesc(&d197)
			ctx.EnsureDesc(&d196)
			var d198 scm.JITValueDesc
			if d197.Loc == scm.LocImm && d196.Loc == scm.LocImm {
				d198 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d197.Imm.Int() - d196.Imm.Int())}
			} else if d196.Loc == scm.LocImm && d196.Imm.Int() == 0 {
				r204 := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(r204, d197.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r204}
				ctx.BindReg(r204, &d198)
			} else if d197.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d196.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d197.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else if d196.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d197.Reg)
				ctx.W.EmitMovRegReg(scratch, d197.Reg)
				if d196.Imm.Int() >= -2147483648 && d196.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d196.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d196.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d198)
			} else {
				r205 := ctx.AllocRegExcept(d197.Reg, d196.Reg)
				ctx.W.EmitMovRegReg(r205, d197.Reg)
				ctx.W.EmitSubInt64(r205, d196.Reg)
				d198 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r205}
				ctx.BindReg(r205, &d198)
			}
			if d198.Loc == scm.LocReg && d197.Loc == scm.LocReg && d198.Reg == d197.Reg {
				ctx.TransferReg(d197.Reg)
				d197.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d196)
			ctx.EnsureDesc(&d194)
			ctx.EnsureDesc(&d198)
			var d199 scm.JITValueDesc
			if d194.Loc == scm.LocImm && d198.Loc == scm.LocImm {
				d199 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d194.Imm.Int()) >> uint64(d198.Imm.Int())))}
			} else if d198.Loc == scm.LocImm {
				r206 := ctx.AllocRegExcept(d194.Reg)
				ctx.W.EmitMovRegReg(r206, d194.Reg)
				ctx.W.EmitShrRegImm8(r206, uint8(d198.Imm.Int()))
				d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r206}
				ctx.BindReg(r206, &d199)
			} else {
				{
					shiftSrc := d194.Reg
					r207 := ctx.AllocRegExcept(d194.Reg)
					ctx.W.EmitMovRegReg(r207, d194.Reg)
					shiftSrc = r207
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d198.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d198.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d198.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d199 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d199)
				}
			}
			if d199.Loc == scm.LocReg && d194.Loc == scm.LocReg && d199.Reg == d194.Reg {
				ctx.TransferReg(d194.Reg)
				d194.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d194)
			ctx.FreeDesc(&d198)
			r208 := ctx.AllocReg()
			ctx.EnsureDesc(&d199)
			ctx.EnsureDesc(&d199)
			if d199.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r208, d199)
			}
			ctx.W.EmitJmp(lbl48)
			bbpos_6_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl49)
			ctx.W.ResolveFixups()
			d194 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d184)
			var d200 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d200 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() % 64)}
			} else {
				r209 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r209, d184.Reg)
				ctx.W.EmitAndRegImm32(r209, 63)
				d200 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r209}
				ctx.BindReg(r209, &d200)
			}
			if d200.Loc == scm.LocReg && d184.Loc == scm.LocReg && d200.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			var d201 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d201 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 24)
				r210 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r210, thisptr.Reg, off)
				d201 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r210}
				ctx.BindReg(r210, &d201)
			}
			ctx.EnsureDesc(&d201)
			ctx.EnsureDesc(&d201)
			var d202 scm.JITValueDesc
			if d201.Loc == scm.LocImm {
				d202 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d201.Imm.Int()))))}
			} else {
				r211 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r211, d201.Reg)
				ctx.W.EmitShlRegImm8(r211, 56)
				ctx.W.EmitShrRegImm8(r211, 56)
				d202 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r211}
				ctx.BindReg(r211, &d202)
			}
			ctx.FreeDesc(&d201)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			ctx.EnsureDesc(&d200)
			ctx.EnsureDesc(&d202)
			var d203 scm.JITValueDesc
			if d200.Loc == scm.LocImm && d202.Loc == scm.LocImm {
				d203 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d200.Imm.Int() + d202.Imm.Int())}
			} else if d202.Loc == scm.LocImm && d202.Imm.Int() == 0 {
				r212 := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(r212, d200.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r212}
				ctx.BindReg(r212, &d203)
			} else if d200.Loc == scm.LocImm && d200.Imm.Int() == 0 {
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d202.Reg}
				ctx.BindReg(d202.Reg, &d203)
			} else if d200.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d202.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d200.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else if d202.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d200.Reg)
				ctx.W.EmitMovRegReg(scratch, d200.Reg)
				if d202.Imm.Int() >= -2147483648 && d202.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d202.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d202.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d203)
			} else {
				r213 := ctx.AllocRegExcept(d200.Reg, d202.Reg)
				ctx.W.EmitMovRegReg(r213, d200.Reg)
				ctx.W.EmitAddInt64(r213, d202.Reg)
				d203 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r213}
				ctx.BindReg(r213, &d203)
			}
			if d203.Loc == scm.LocReg && d200.Loc == scm.LocReg && d203.Reg == d200.Reg {
				ctx.TransferReg(d200.Reg)
				d200.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d200)
			ctx.FreeDesc(&d202)
			ctx.EnsureDesc(&d203)
			var d204 scm.JITValueDesc
			if d203.Loc == scm.LocImm {
				d204 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d203.Imm.Int()) > uint64(64))}
			} else {
				r214 := ctx.AllocRegExcept(d203.Reg)
				ctx.W.EmitCmpRegImm32(d203.Reg, 64)
				ctx.W.EmitSetcc(r214, scm.CcA)
				d204 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r214}
				ctx.BindReg(r214, &d204)
			}
			ctx.FreeDesc(&d203)
			d205 := d204
			ctx.EnsureDesc(&d205)
			if d205.Loc != scm.LocImm && d205.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl53 := ctx.W.ReserveLabel()
			lbl54 := ctx.W.ReserveLabel()
			lbl55 := ctx.W.ReserveLabel()
			if d205.Loc == scm.LocImm {
				if d205.Imm.Bool() {
					ctx.W.MarkLabel(lbl54)
					ctx.W.EmitJmp(lbl53)
				} else {
					ctx.W.MarkLabel(lbl55)
			d206 := d189
			if d206.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d206)
			ctx.EmitStoreToStack(d206, 32)
					ctx.W.EmitJmp(lbl50)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d205.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl54)
				ctx.W.EmitJmp(lbl55)
				ctx.W.MarkLabel(lbl54)
				ctx.W.EmitJmp(lbl53)
				ctx.W.MarkLabel(lbl55)
			d207 := d189
			if d207.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d207)
			ctx.EmitStoreToStack(d207, 32)
				ctx.W.EmitJmp(lbl50)
			}
			ctx.FreeDesc(&d204)
			bbpos_6_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl53)
			ctx.W.ResolveFixups()
			d194 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(32)}
			ctx.EnsureDesc(&d184)
			var d208 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d208 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() / 64)}
			} else {
				r215 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r215, d184.Reg)
				ctx.W.EmitShrRegImm8(r215, 6)
				d208 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r215}
				ctx.BindReg(r215, &d208)
			}
			if d208.Loc == scm.LocReg && d184.Loc == scm.LocReg && d208.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d208)
			ctx.EnsureDesc(&d208)
			var d209 scm.JITValueDesc
			if d208.Loc == scm.LocImm {
				d209 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d208.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d208.Reg)
				ctx.W.EmitMovRegReg(scratch, d208.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d209 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d209)
			}
			if d209.Loc == scm.LocReg && d208.Loc == scm.LocReg && d209.Reg == d208.Reg {
				ctx.TransferReg(d208.Reg)
				d208.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d208)
			ctx.EnsureDesc(&d209)
			r216 := ctx.AllocReg()
			ctx.EnsureDesc(&d209)
			ctx.EnsureDesc(&d185)
			if d209.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r216, uint64(d209.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r216, d209.Reg)
				ctx.W.EmitShlRegImm8(r216, 3)
			}
			if d185.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d185.Imm.Int()))
				ctx.W.EmitAddInt64(r216, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r216, d185.Reg)
			}
			r217 := ctx.AllocRegExcept(r216)
			ctx.W.EmitMovRegMem(r217, r216, 0)
			ctx.FreeReg(r216)
			d210 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r217}
			ctx.BindReg(r217, &d210)
			ctx.FreeDesc(&d209)
			ctx.EnsureDesc(&d184)
			var d211 scm.JITValueDesc
			if d184.Loc == scm.LocImm {
				d211 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d184.Imm.Int() % 64)}
			} else {
				r218 := ctx.AllocRegExcept(d184.Reg)
				ctx.W.EmitMovRegReg(r218, d184.Reg)
				ctx.W.EmitAndRegImm32(r218, 63)
				d211 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r218}
				ctx.BindReg(r218, &d211)
			}
			if d211.Loc == scm.LocReg && d184.Loc == scm.LocReg && d211.Reg == d184.Reg {
				ctx.TransferReg(d184.Reg)
				d184.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d184)
			d212 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			ctx.EnsureDesc(&d212)
			ctx.EnsureDesc(&d211)
			var d213 scm.JITValueDesc
			if d212.Loc == scm.LocImm && d211.Loc == scm.LocImm {
				d213 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d212.Imm.Int() - d211.Imm.Int())}
			} else if d211.Loc == scm.LocImm && d211.Imm.Int() == 0 {
				r219 := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(r219, d212.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r219}
				ctx.BindReg(r219, &d213)
			} else if d212.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d211.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d212.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else if d211.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d212.Reg)
				ctx.W.EmitMovRegReg(scratch, d212.Reg)
				if d211.Imm.Int() >= -2147483648 && d211.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d211.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d211.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d213)
			} else {
				r220 := ctx.AllocRegExcept(d212.Reg, d211.Reg)
				ctx.W.EmitMovRegReg(r220, d212.Reg)
				ctx.W.EmitSubInt64(r220, d211.Reg)
				d213 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r220}
				ctx.BindReg(r220, &d213)
			}
			if d213.Loc == scm.LocReg && d212.Loc == scm.LocReg && d213.Reg == d212.Reg {
				ctx.TransferReg(d212.Reg)
				d212.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d211)
			ctx.EnsureDesc(&d210)
			ctx.EnsureDesc(&d213)
			var d214 scm.JITValueDesc
			if d210.Loc == scm.LocImm && d213.Loc == scm.LocImm {
				d214 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d210.Imm.Int()) >> uint64(d213.Imm.Int())))}
			} else if d213.Loc == scm.LocImm {
				r221 := ctx.AllocRegExcept(d210.Reg)
				ctx.W.EmitMovRegReg(r221, d210.Reg)
				ctx.W.EmitShrRegImm8(r221, uint8(d213.Imm.Int()))
				d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r221}
				ctx.BindReg(r221, &d214)
			} else {
				{
					shiftSrc := d210.Reg
					r222 := ctx.AllocRegExcept(d210.Reg)
					ctx.W.EmitMovRegReg(r222, d210.Reg)
					shiftSrc = r222
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d213.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d213.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d213.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d214 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d214)
				}
			}
			if d214.Loc == scm.LocReg && d210.Loc == scm.LocReg && d214.Reg == d210.Reg {
				ctx.TransferReg(d210.Reg)
				d210.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d210)
			ctx.FreeDesc(&d213)
			ctx.EnsureDesc(&d189)
			ctx.EnsureDesc(&d214)
			var d215 scm.JITValueDesc
			if d189.Loc == scm.LocImm && d214.Loc == scm.LocImm {
				d215 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d189.Imm.Int() | d214.Imm.Int())}
			} else if d189.Loc == scm.LocImm && d189.Imm.Int() == 0 {
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d214.Reg}
				ctx.BindReg(d214.Reg, &d215)
			} else if d214.Loc == scm.LocImm && d214.Imm.Int() == 0 {
				r223 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r223, d189.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r223}
				ctx.BindReg(r223, &d215)
			} else if d189.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d214.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d189.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d215)
			} else if d214.Loc == scm.LocImm {
				r224 := ctx.AllocRegExcept(d189.Reg)
				ctx.W.EmitMovRegReg(r224, d189.Reg)
				if d214.Imm.Int() >= -2147483648 && d214.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r224, int32(d214.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d214.Imm.Int()))
					ctx.W.EmitOrInt64(r224, scm.RegR11)
				}
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r224}
				ctx.BindReg(r224, &d215)
			} else {
				r225 := ctx.AllocRegExcept(d189.Reg, d214.Reg)
				ctx.W.EmitMovRegReg(r225, d189.Reg)
				ctx.W.EmitOrInt64(r225, d214.Reg)
				d215 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r225}
				ctx.BindReg(r225, &d215)
			}
			if d215.Loc == scm.LocReg && d189.Loc == scm.LocReg && d215.Reg == d189.Reg {
				ctx.TransferReg(d189.Reg)
				d189.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d214)
			d216 := d215
			if d216.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d216)
			ctx.EmitStoreToStack(d216, 32)
			ctx.W.EmitJmp(lbl50)
			ctx.W.MarkLabel(lbl48)
			d217 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r208}
			ctx.BindReg(r208, &d217)
			ctx.BindReg(r208, &d217)
			if r188 { ctx.UnprotectReg(r189) }
			ctx.EnsureDesc(&d217)
			ctx.EnsureDesc(&d217)
			var d218 scm.JITValueDesc
			if d217.Loc == scm.LocImm {
				d218 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d217.Imm.Int()))))}
			} else {
				r226 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r226, d217.Reg)
				d218 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r226}
				ctx.BindReg(r226, &d218)
			}
			ctx.FreeDesc(&d217)
			var d219 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d219 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 160 + 32)
				r227 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r227, thisptr.Reg, off)
				d219 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r227}
				ctx.BindReg(r227, &d219)
			}
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			ctx.EnsureDesc(&d218)
			ctx.EnsureDesc(&d219)
			var d220 scm.JITValueDesc
			if d218.Loc == scm.LocImm && d219.Loc == scm.LocImm {
				d220 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d218.Imm.Int() + d219.Imm.Int())}
			} else if d219.Loc == scm.LocImm && d219.Imm.Int() == 0 {
				r228 := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(r228, d218.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r228}
				ctx.BindReg(r228, &d220)
			} else if d218.Loc == scm.LocImm && d218.Imm.Int() == 0 {
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d219.Reg}
				ctx.BindReg(d219.Reg, &d220)
			} else if d218.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d219.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d218.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else if d219.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d218.Reg)
				ctx.W.EmitMovRegReg(scratch, d218.Reg)
				if d219.Imm.Int() >= -2147483648 && d219.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d219.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d219.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d220)
			} else {
				r229 := ctx.AllocRegExcept(d218.Reg, d219.Reg)
				ctx.W.EmitMovRegReg(r229, d218.Reg)
				ctx.W.EmitAddInt64(r229, d219.Reg)
				d220 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r229}
				ctx.BindReg(r229, &d220)
			}
			if d220.Loc == scm.LocReg && d218.Loc == scm.LocReg && d220.Reg == d218.Reg {
				ctx.TransferReg(d218.Reg)
				d218.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d218)
			ctx.FreeDesc(&d219)
			ctx.EnsureDesc(&d220)
			ctx.EnsureDesc(&d220)
			var d221 scm.JITValueDesc
			if d220.Loc == scm.LocImm {
				d221 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d220.Imm.Int()))))}
			} else {
				r230 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r230, d220.Reg)
				d221 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r230}
				ctx.BindReg(r230, &d221)
			}
			ctx.FreeDesc(&d220)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d177)
			var d222 scm.JITValueDesc
			if d177.Loc == scm.LocImm {
				d222 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d177.Imm.Int()))))}
			} else {
				r231 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r231, d177.Reg)
				d222 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r231}
				ctx.BindReg(r231, &d222)
			}
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d221)
			var d223 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d221.Loc == scm.LocImm {
				d223 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d177.Imm.Int() + d221.Imm.Int())}
			} else if d221.Loc == scm.LocImm && d221.Imm.Int() == 0 {
				r232 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(r232, d177.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r232}
				ctx.BindReg(r232, &d223)
			} else if d177.Loc == scm.LocImm && d177.Imm.Int() == 0 {
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d221.Reg}
				ctx.BindReg(d221.Reg, &d223)
			} else if d177.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d221.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d177.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else if d221.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitMovRegReg(scratch, d177.Reg)
				if d221.Imm.Int() >= -2147483648 && d221.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d221.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d221.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d223)
			} else {
				r233 := ctx.AllocRegExcept(d177.Reg, d221.Reg)
				ctx.W.EmitMovRegReg(r233, d177.Reg)
				ctx.W.EmitAddInt64(r233, d221.Reg)
				d223 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r233}
				ctx.BindReg(r233, &d223)
			}
			if d223.Loc == scm.LocReg && d177.Loc == scm.LocReg && d223.Reg == d177.Reg {
				ctx.TransferReg(d177.Reg)
				d177.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d221)
			ctx.EnsureDesc(&d223)
			ctx.EnsureDesc(&d223)
			var d224 scm.JITValueDesc
			if d223.Loc == scm.LocImm {
				d224 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(int64(uint64(d223.Imm.Int()))))}
			} else {
				r234 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r234, d223.Reg)
				d224 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r234}
				ctx.BindReg(r234, &d224)
			}
			ctx.FreeDesc(&d223)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			r235 := ctx.AllocReg()
			r236 := ctx.AllocRegExcept(r235)
			ctx.EnsureDesc(&d132)
			ctx.EnsureDesc(&d222)
			ctx.EnsureDesc(&d224)
			if d132.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r235, uint64(d132.Imm.Int()))
			} else if d132.Loc == scm.LocRegPair {
				ctx.W.EmitMovRegReg(r235, d132.Reg)
			} else {
				ctx.W.EmitMovRegReg(r235, d132.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() != 0 {
					if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
						ctx.W.EmitAddRegImm32(r235, int32(d222.Imm.Int()))
					} else {
						ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
						ctx.W.EmitAddInt64(r235, scm.RegR11)
					}
				}
			} else {
				ctx.W.EmitAddInt64(r235, d222.Reg)
			}
			if d224.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r236, uint64(d224.Imm.Int()))
			} else {
				ctx.W.EmitMovRegReg(r236, d224.Reg)
			}
			if d222.Loc == scm.LocImm {
				if d222.Imm.Int() >= -2147483648 && d222.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(r236, int32(d222.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d222.Imm.Int()))
					ctx.W.EmitSubInt64(r236, scm.RegR11)
				}
			} else {
				ctx.W.EmitSubInt64(r236, d222.Reg)
			}
			d225 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r235, Reg2: r236}
			ctx.BindReg(r235, &d225)
			ctx.BindReg(r236, &d225)
			ctx.FreeDesc(&d222)
			ctx.FreeDesc(&d224)
			d226 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d226)
			ctx.BindReg(r143, &d226)
			d227 := ctx.EmitGoCallScalar(scm.GoFuncAddr(scm.NewString), []scm.JITValueDesc{d225}, 2)
			ctx.EmitMovPairToResult(&d227, &d226)
			ctx.W.EmitJmp(lbl3)
			bbpos_1_8 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl16)
			ctx.W.ResolveFixups()
			var d228 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64
				val := *(*uint64)(unsafe.Pointer(fieldAddr))
				d228 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StoragePrefix)(nil).values) + 0 + 64)
				r237 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r237, thisptr.Reg, off)
				d228 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r237}
				ctx.BindReg(r237, &d228)
			}
			ctx.EnsureDesc(&d228)
			ctx.EnsureDesc(&d228)
			var d229 scm.JITValueDesc
			if d228.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d228.Imm.Int()))))}
			} else {
				r238 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r238, d228.Reg)
				ctx.W.EmitShlRegImm8(r238, 32)
				ctx.W.EmitShrRegImm8(r238, 32)
				d229 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r238}
				ctx.BindReg(r238, &d229)
			}
			ctx.FreeDesc(&d228)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d229)
			var d230 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d229.Loc == scm.LocImm {
				d230 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) == uint64(d229.Imm.Int()))}
			} else if d229.Loc == scm.LocImm {
				r239 := ctx.AllocRegExcept(d44.Reg)
				if d229.Imm.Int() >= -2147483648 && d229.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d44.Reg, int32(d229.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d229.Imm.Int()))
					ctx.W.EmitCmpInt64(d44.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r239, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r239}
				ctx.BindReg(r239, &d230)
			} else if d44.Loc == scm.LocImm {
				r240 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d229.Reg)
				ctx.W.EmitSetcc(r240, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r240}
				ctx.BindReg(r240, &d230)
			} else {
				r241 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitCmpInt64(d44.Reg, d229.Reg)
				ctx.W.EmitSetcc(r241, scm.CcE)
				d230 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r241}
				ctx.BindReg(r241, &d230)
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d229)
			d231 := d230
			ctx.EnsureDesc(&d231)
			if d231.Loc != scm.LocImm && d231.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl56 := ctx.W.ReserveLabel()
			lbl57 := ctx.W.ReserveLabel()
			lbl58 := ctx.W.ReserveLabel()
			if d231.Loc == scm.LocImm {
				if d231.Imm.Bool() {
					ctx.W.MarkLabel(lbl57)
					ctx.W.EmitJmp(lbl56)
				} else {
					ctx.W.MarkLabel(lbl58)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d231.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl57)
				ctx.W.EmitJmp(lbl58)
				ctx.W.MarkLabel(lbl57)
				ctx.W.EmitJmp(lbl56)
				ctx.W.MarkLabel(lbl58)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d230)
			bbpos_1_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl44)
			ctx.W.ResolveFixups()
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
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			ctx.EnsureDesc(&d177)
			ctx.EnsureDesc(&d232)
			var d233 scm.JITValueDesc
			if d177.Loc == scm.LocImm && d232.Loc == scm.LocImm {
				d233 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d177.Imm.Int()) == uint64(d232.Imm.Int()))}
			} else if d232.Loc == scm.LocImm {
				r243 := ctx.AllocRegExcept(d177.Reg)
				if d232.Imm.Int() >= -2147483648 && d232.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d177.Reg, int32(d232.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d232.Imm.Int()))
					ctx.W.EmitCmpInt64(d177.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r243, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r243}
				ctx.BindReg(r243, &d233)
			} else if d177.Loc == scm.LocImm {
				r244 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d177.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d232.Reg)
				ctx.W.EmitSetcc(r244, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r244}
				ctx.BindReg(r244, &d233)
			} else {
				r245 := ctx.AllocRegExcept(d177.Reg)
				ctx.W.EmitCmpInt64(d177.Reg, d232.Reg)
				ctx.W.EmitSetcc(r245, scm.CcE)
				d233 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r245}
				ctx.BindReg(r245, &d233)
			}
			ctx.FreeDesc(&d177)
			ctx.FreeDesc(&d232)
			d234 := d233
			ctx.EnsureDesc(&d234)
			if d234.Loc != scm.LocImm && d234.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl59 := ctx.W.ReserveLabel()
			lbl60 := ctx.W.ReserveLabel()
			lbl61 := ctx.W.ReserveLabel()
			if d234.Loc == scm.LocImm {
				if d234.Imm.Bool() {
					ctx.W.MarkLabel(lbl60)
					ctx.W.EmitJmp(lbl59)
				} else {
					ctx.W.MarkLabel(lbl61)
					ctx.W.EmitJmp(lbl45)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d234.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl60)
				ctx.W.EmitJmp(lbl61)
				ctx.W.MarkLabel(lbl60)
				ctx.W.EmitJmp(lbl59)
				ctx.W.MarkLabel(lbl61)
				ctx.W.EmitJmp(lbl45)
			}
			ctx.FreeDesc(&d233)
			bbpos_1_6 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl56)
			ctx.W.ResolveFixups()
			d235 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d235)
			ctx.BindReg(r143, &d235)
			ctx.W.EmitMakeNil(d235)
			ctx.W.EmitJmp(lbl3)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl59)
			ctx.W.ResolveFixups()
			d236 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d236)
			ctx.BindReg(r143, &d236)
			ctx.W.EmitMakeNil(d236)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl3)
			d237 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r142, Reg2: r143}
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			ctx.BindReg(r142, &d237)
			ctx.BindReg(r143, &d237)
			if r2 { ctx.UnprotectReg(r3) }
			d239 := d237
			d239.ID = 0
			d238 := ctx.EmitTagEquals(&d239, scm.TagNil, scm.JITValueDesc{Loc: scm.LocAny})
			d240 := d238
			ctx.EnsureDesc(&d240)
			if d240.Loc != scm.LocImm && d240.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl62 := ctx.W.ReserveLabel()
			lbl63 := ctx.W.ReserveLabel()
			lbl64 := ctx.W.ReserveLabel()
			lbl65 := ctx.W.ReserveLabel()
			if d240.Loc == scm.LocImm {
				if d240.Imm.Bool() {
					ctx.W.MarkLabel(lbl64)
					ctx.W.EmitJmp(lbl62)
				} else {
					ctx.W.MarkLabel(lbl65)
					ctx.W.EmitJmp(lbl63)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d240.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl64)
				ctx.W.EmitJmp(lbl65)
				ctx.W.MarkLabel(lbl64)
				ctx.W.EmitJmp(lbl62)
				ctx.W.MarkLabel(lbl65)
				ctx.W.EmitJmp(lbl63)
			}
			ctx.FreeDesc(&d238)
			bbpos_0_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl63)
			ctx.W.ResolveFixups()
			d242 := d237
			d242.ID = 0
			d241 := ctx.EmitTagEquals(&d242, scm.TagString, scm.JITValueDesc{Loc: scm.LocAny})
			d243 := d241
			ctx.EnsureDesc(&d243)
			if d243.Loc != scm.LocImm && d243.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl66 := ctx.W.ReserveLabel()
			lbl67 := ctx.W.ReserveLabel()
			lbl68 := ctx.W.ReserveLabel()
			lbl69 := ctx.W.ReserveLabel()
			if d243.Loc == scm.LocImm {
				if d243.Imm.Bool() {
					ctx.W.MarkLabel(lbl68)
					ctx.W.EmitJmp(lbl66)
				} else {
					ctx.W.MarkLabel(lbl69)
					ctx.W.EmitJmp(lbl67)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d243.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl68)
				ctx.W.EmitJmp(lbl69)
				ctx.W.MarkLabel(lbl68)
				ctx.W.EmitJmp(lbl66)
				ctx.W.MarkLabel(lbl69)
				ctx.W.EmitJmp(lbl67)
			}
			ctx.FreeDesc(&d241)
			bbpos_0_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl67)
			ctx.W.ResolveFixups()
			ctx.W.EmitByte(0xCC)
			bbpos_0_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl62)
			ctx.W.ResolveFixups()
			d244 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d244)
			ctx.BindReg(r1, &d244)
			ctx.W.EmitMakeNil(d244)
			ctx.W.EmitJmp(lbl0)
			bbpos_0_4 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl66)
			ctx.W.ResolveFixups()
			ctx.EnsureDesc(&idxInt)
			d245 := idxInt
			_ = d245
			r246 := idxInt.Loc == scm.LocReg
			r247 := idxInt.Reg
			if r246 { ctx.ProtectReg(r247) }
			lbl70 := ctx.W.ReserveLabel()
			bbpos_7_0 := int32(-1)
			_ = bbpos_7_0
			bbpos_7_1 := int32(-1)
			_ = bbpos_7_1
			bbpos_7_2 := int32(-1)
			_ = bbpos_7_2
			bbpos_7_3 := int32(-1)
			_ = bbpos_7_3
			bbpos_7_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
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
			lbl71 := ctx.W.ReserveLabel()
			lbl72 := ctx.W.ReserveLabel()
			lbl73 := ctx.W.ReserveLabel()
			lbl74 := ctx.W.ReserveLabel()
			if d256.Loc == scm.LocImm {
				if d256.Imm.Bool() {
					ctx.W.MarkLabel(lbl73)
					ctx.W.EmitJmp(lbl71)
				} else {
					ctx.W.MarkLabel(lbl74)
			d257 := d254
			if d257.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d257)
			ctx.EmitStoreToStack(d257, 40)
					ctx.W.EmitJmp(lbl72)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d256.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl73)
				ctx.W.EmitJmp(lbl74)
				ctx.W.MarkLabel(lbl73)
				ctx.W.EmitJmp(lbl71)
				ctx.W.MarkLabel(lbl74)
			d258 := d254
			if d258.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d258)
			ctx.EmitStoreToStack(d258, 40)
				ctx.W.EmitJmp(lbl72)
			}
			ctx.FreeDesc(&d255)
			bbpos_7_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl72)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl70)
			bbpos_7_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl71)
			ctx.W.ResolveFixups()
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
			lbl75 := ctx.W.ReserveLabel()
			lbl76 := ctx.W.ReserveLabel()
			lbl77 := ctx.W.ReserveLabel()
			if d270.Loc == scm.LocImm {
				if d270.Imm.Bool() {
					ctx.W.MarkLabel(lbl76)
					ctx.W.EmitJmp(lbl75)
				} else {
					ctx.W.MarkLabel(lbl77)
			d271 := d254
			if d271.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d271)
			ctx.EmitStoreToStack(d271, 40)
					ctx.W.EmitJmp(lbl72)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d270.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl76)
				ctx.W.EmitJmp(lbl77)
				ctx.W.MarkLabel(lbl76)
				ctx.W.EmitJmp(lbl75)
				ctx.W.MarkLabel(lbl77)
			d272 := d254
			if d272.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d272)
			ctx.EmitStoreToStack(d272, 40)
				ctx.W.EmitJmp(lbl72)
			}
			ctx.FreeDesc(&d269)
			bbpos_7_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl75)
			ctx.W.ResolveFixups()
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
			ctx.W.EmitJmp(lbl72)
			ctx.W.MarkLabel(lbl70)
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
			lbl78 := ctx.W.ReserveLabel()
			lbl79 := ctx.W.ReserveLabel()
			lbl80 := ctx.W.ReserveLabel()
			if d290.Loc == scm.LocImm {
				if d290.Imm.Bool() {
					ctx.W.MarkLabel(lbl79)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.MarkLabel(lbl80)
					ctx.W.EmitJmp(lbl78)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d290.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl79)
				ctx.W.EmitJmp(lbl80)
				ctx.W.MarkLabel(lbl79)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl80)
				ctx.W.EmitJmp(lbl78)
			}
			ctx.FreeDesc(&d289)
			bbpos_0_7 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl78)
			ctx.W.ResolveFixups()
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
			lbl81 := ctx.W.ReserveLabel()
			lbl82 := ctx.W.ReserveLabel()
			if d292.Loc == scm.LocImm {
				if d292.Imm.Bool() {
					ctx.W.MarkLabel(lbl81)
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.MarkLabel(lbl82)
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d292.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl81)
				ctx.W.EmitJmp(lbl82)
				ctx.W.MarkLabel(lbl81)
				ctx.W.EmitJmp(lbl1)
				ctx.W.MarkLabel(lbl82)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d291)
			bbpos_0_6 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl2)
			ctx.W.ResolveFixups()
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
			bbpos_0_5 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl1)
			ctx.W.ResolveFixups()
			ctx.W.EmitByte(0xCC)
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
