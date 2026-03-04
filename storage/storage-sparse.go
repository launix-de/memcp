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
import "bufio"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"
import "unsafe"

type StorageSparse struct {
	i, count uint64
	recids   StorageInt
	values   []scm.Scmer // TODO: embed other formats as values (ColumnStorage with a proposeCompression loop)
}

func (s *StorageSparse) ComputeSize() uint {
	var sz uint = 16 + 8 + 24 + s.recids.ComputeSize() + 8*uint(len(s.values))
	for _, v := range s.values {
		sz += scm.ComputeSize(v)
	}
	return sz
}

func (s *StorageSparse) String() string {
	return "SCMER-sparse"
}
func (s *StorageSparse) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(2)) // 2 = StorageSparse
	binary.Write(f, binary.LittleEndian, uint64(s.count))
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	for k, v := range s.values {
		vbytes, err := json.Marshal(uint64(s.recids.GetValueUInt(uint32(k)) + uint64(s.recids.offset)))
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file beginning at byte 9
		vbytes, err = json.Marshal(v)
		if err != nil {
			panic(err)
		}
		f.Write(vbytes)
		f.Write([]byte("\n")) // endline so the serialized file becomes a jsonl file beginning at byte 9
	}
}
func (s *StorageSparse) Deserialize(f io.Reader) uint {
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	s.count = l
	var l2 uint64
	binary.Read(f, binary.LittleEndian, &l2)
	s.values = make([]scm.Scmer, l2)
	s.i = l2
	scanner := bufio.NewScanner(f)
	s.recids.prepare()
	s.recids.scan(0, scm.NewInt(0))
	s.recids.scan(uint32(l2-1), scm.NewInt(int64(l-1)))
	s.recids.init(uint32(l2))
	i := 0
	for {
		var k uint64
		if !scanner.Scan() {
			break
		}
		json.Unmarshal(scanner.Bytes(), &k)
		if !scanner.Scan() {
			break
		}
		var v any
		json.Unmarshal(scanner.Bytes(), &v)
		s.recids.build(uint32(i), scm.NewInt(int64(k)))
		s.values[i] = scm.TransformFromJSON(v)
		i++
	}
	s.recids.finish()
	return uint(l)
}

func (s *StorageSparse) GetCachedReader() ColumnReader { return s }

func (s *StorageSparse) GetValue(i uint32) scm.Scmer {
	var lower uint32 = 0
	var upper uint32 = uint32(s.i)
	for {
		if lower == upper {
			return scm.NewNil() // sparse value
		}
		pivot := (lower + upper) / 2
		recid := s.recids.GetValueUInt(pivot) + uint64(s.recids.offset)
		if recid == uint64(i) {
			return s.values[pivot] // found the value
		}
		if recid < uint64(i) {
			lower = pivot + 1
		} else {
			upper = pivot
		}

	}
}
func (s *StorageSparse) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			lbl0 := ctx.W.ReserveLabel()
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl1)
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
				ctx.BindReg(r3, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r4, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d0)
			}
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d0)
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d0.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r5, d0.Reg)
				ctx.W.EmitShlRegImm8(r5, 32)
				ctx.W.EmitShrRegImm8(r5, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d1)
			}
			lbl2 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
			d2 := d1
			if d2.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d2)
			d3 := d2
			if d3.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: d3.Type, Imm: scm.NewInt(int64(uint64(d3.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d3.Reg, 32)
				ctx.W.EmitShrRegImm8(d3.Reg, 32)
			}
			ctx.EmitStoreToStack(d3, 8)
			ctx.W.MarkLabel(lbl2)
			d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d4.Imm.Int()) == uint64(d5.Imm.Int()))}
			} else if d5.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d4.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d4.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r6, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d6)
			} else if d4.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d5.Reg)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d6)
			} else {
				r8 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitCmpInt64(d4.Reg, d5.Reg)
				ctx.W.EmitSetcc(r8, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d6)
			}
			d7 := d6
			ctx.EnsureDesc(&d7)
			if d7.Loc != scm.LocImm && d7.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			lbl5 := ctx.W.ReserveLabel()
			lbl6 := ctx.W.ReserveLabel()
			if d7.Loc == scm.LocImm {
				if d7.Imm.Bool() {
					ctx.W.MarkLabel(lbl5)
					ctx.W.EmitJmp(lbl3)
				} else {
					ctx.W.MarkLabel(lbl6)
					ctx.W.EmitJmp(lbl4)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
				ctx.W.MarkLabel(lbl5)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl6)
				ctx.W.EmitJmp(lbl4)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl4)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d8 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + d5.Imm.Int())}
			} else if d5.Loc == scm.LocImm && d5.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d8)
			} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d8)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d5.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d8)
			} else {
				r10 := ctx.AllocRegExcept(d4.Reg, d5.Reg)
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitAddInt64(r10, d5.Reg)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d8)
			}
			if d8.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: d8.Type, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d8.Reg, 32)
				ctx.W.EmitShrRegImm8(d8.Reg, 32)
			}
			if d8.Loc == scm.LocReg && d4.Loc == scm.LocReg && d8.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d8)
			var d9 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r11, d8.Reg)
				ctx.W.EmitShrRegImm8(r11, 1)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d9)
			}
			if d9.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: d9.Type, Imm: scm.NewInt(int64(uint64(d9.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d9.Reg, 32)
				ctx.W.EmitShrRegImm8(d9.Reg, 32)
			}
			if d9.Loc == scm.LocReg && d8.Loc == scm.LocReg && d9.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.EnsureDesc(&d9)
			d10 := d9
			_ = d10
			r12 := d9.Loc == scm.LocReg
			r13 := d9.Reg
			if r12 { ctx.ProtectReg(r13) }
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			ctx.W.MarkLabel(lbl8)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d10)
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d10.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d10.Reg)
				ctx.W.EmitShlRegImm8(r14, 32)
				ctx.W.EmitShrRegImm8(r14, 32)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d11)
			}
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d12)
			}
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d12.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d12.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d13)
			}
			ctx.FreeDesc(&d12)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d13)
			var d14 scm.JITValueDesc
			if d11.Loc == scm.LocImm && d13.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() * d13.Imm.Int())}
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d11.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else if d13.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(scratch, d11.Reg)
				if d13.Imm.Int() >= -2147483648 && d13.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d13.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d14)
			} else {
				r17 := ctx.AllocRegExcept(d11.Reg, d13.Reg)
				ctx.W.EmitMovRegReg(r17, d11.Reg)
				ctx.W.EmitImulInt64(r17, d13.Reg)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d14)
			}
			if d14.Loc == scm.LocReg && d11.Loc == scm.LocReg && d14.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			ctx.FreeDesc(&d13)
			var d15 scm.JITValueDesc
			r18 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r18, uint64(dataPtr))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18, StackOff: int32(sliceLen)}
				ctx.BindReg(r18, &d15)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				ctx.W.EmitMovRegMem(r18, thisptr.Reg, off)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d15)
			}
			ctx.BindReg(r18, &d15)
			ctx.EnsureDesc(&d14)
			var d16 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
			} else {
				r19 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r19, d14.Reg)
				ctx.W.EmitShrRegImm8(r19, 6)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d16)
			}
			if d16.Loc == scm.LocReg && d14.Loc == scm.LocReg && d16.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d16)
			r20 := ctx.AllocReg()
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d15)
			if d16.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r20, uint64(d16.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r20, d16.Reg)
				ctx.W.EmitShlRegImm8(r20, 3)
			}
			if d15.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
				ctx.W.EmitAddInt64(r20, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r20, d15.Reg)
			}
			r21 := ctx.AllocRegExcept(r20)
			ctx.W.EmitMovRegMem(r21, r20, 0)
			ctx.FreeReg(r20)
			d17 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d17)
			ctx.FreeDesc(&d16)
			ctx.EnsureDesc(&d14)
			var d18 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r22, d14.Reg)
				ctx.W.EmitAndRegImm32(r22, 63)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d18)
			}
			if d18.Loc == scm.LocReg && d14.Loc == scm.LocReg && d18.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d17)
			ctx.EnsureDesc(&d18)
			var d19 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) << uint64(d18.Imm.Int())))}
			} else if d18.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r23, d17.Reg)
				ctx.W.EmitShlRegImm8(r23, uint8(d18.Imm.Int()))
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d19)
			} else {
				{
					shiftSrc := d17.Reg
					r24 := ctx.AllocRegExcept(d17.Reg)
					ctx.W.EmitMovRegReg(r24, d17.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d18.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d18.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d18.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d19)
				}
			}
			if d19.Loc == scm.LocReg && d17.Loc == scm.LocReg && d19.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d18)
			var d20 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d20)
			}
			d21 := d20
			ctx.EnsureDesc(&d21)
			if d21.Loc != scm.LocImm && d21.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			if d21.Loc == scm.LocImm {
				if d21.Imm.Bool() {
					ctx.W.MarkLabel(lbl11)
					ctx.W.EmitJmp(lbl9)
				} else {
					ctx.W.MarkLabel(lbl12)
			d22 := d19
			if d22.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, 16)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl11)
				ctx.W.EmitJmp(lbl9)
				ctx.W.MarkLabel(lbl12)
			d23 := d19
			if d23.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d23)
			ctx.EmitStoreToStack(d23, 16)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl10)
			d24 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d25 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d25)
			}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d25.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d25.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d26)
			}
			ctx.FreeDesc(&d25)
			d27 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d26)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() - d26.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r28, d27.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d28)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d26.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d28)
			} else {
				r29 := ctx.AllocRegExcept(d27.Reg, d26.Reg)
				ctx.W.EmitMovRegReg(r29, d27.Reg)
				ctx.W.EmitSubInt64(r29, d26.Reg)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d28)
			}
			if d28.Loc == scm.LocReg && d27.Loc == scm.LocReg && d28.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d28)
			var d29 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d24.Imm.Int()) >> uint64(d28.Imm.Int())))}
			} else if d28.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r30, d24.Reg)
				ctx.W.EmitShrRegImm8(r30, uint8(d28.Imm.Int()))
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d29)
			} else {
				{
					shiftSrc := d24.Reg
					r31 := ctx.AllocRegExcept(d24.Reg)
					ctx.W.EmitMovRegReg(r31, d24.Reg)
					shiftSrc = r31
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d28.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d28.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d28.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d29)
				}
			}
			if d29.Loc == scm.LocReg && d24.Loc == scm.LocReg && d29.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d28)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			if d29.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d29)
			}
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl9)
			d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d14)
			var d30 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r33, d14.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d30)
			}
			if d30.Loc == scm.LocReg && d14.Loc == scm.LocReg && d30.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			var d31 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
				ctx.BindReg(r34, &d31)
			}
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d31.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r35, d31.Reg)
				ctx.W.EmitShlRegImm8(r35, 56)
				ctx.W.EmitShrRegImm8(r35, 56)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d32)
			}
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d32)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d32)
			var d33 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d30.Imm.Int() + d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(r36, d30.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d33)
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
				ctx.BindReg(d32.Reg, &d33)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d30.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegReg(scratch, d30.Reg)
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d32.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d33)
			} else {
				r37 := ctx.AllocRegExcept(d30.Reg, d32.Reg)
				ctx.W.EmitMovRegReg(r37, d30.Reg)
				ctx.W.EmitAddInt64(r37, d32.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d33)
			}
			if d33.Loc == scm.LocReg && d30.Loc == scm.LocReg && d33.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d32)
			ctx.EnsureDesc(&d33)
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d33.Imm.Int()) > uint64(64))}
			} else {
				r38 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitCmpRegImm32(d33.Reg, 64)
				ctx.W.EmitSetcc(r38, scm.CcA)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
				ctx.BindReg(r38, &d34)
			}
			ctx.FreeDesc(&d33)
			d35 := d34
			ctx.EnsureDesc(&d35)
			if d35.Loc != scm.LocImm && d35.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d35.Loc == scm.LocImm {
				if d35.Imm.Bool() {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl13)
				} else {
					ctx.W.MarkLabel(lbl15)
			d36 := d19
			if d36.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d36)
			ctx.EmitStoreToStack(d36, 16)
					ctx.W.EmitJmp(lbl10)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d35.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl13)
				ctx.W.MarkLabel(lbl15)
			d37 := d19
			if d37.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d37)
			ctx.EmitStoreToStack(d37, 16)
				ctx.W.EmitJmp(lbl10)
			}
			ctx.FreeDesc(&d34)
			ctx.W.MarkLabel(lbl13)
			d24 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d14)
			var d38 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r39, d14.Reg)
				ctx.W.EmitShrRegImm8(r39, 6)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d38)
			}
			if d38.Loc == scm.LocReg && d14.Loc == scm.LocReg && d38.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d38)
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			}
			if d39.Loc == scm.LocReg && d38.Loc == scm.LocReg && d39.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d39)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d15)
			if d39.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r40, uint64(d39.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r40, d39.Reg)
				ctx.W.EmitShlRegImm8(r40, 3)
			}
			if d15.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d15.Imm.Int()))
				ctx.W.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r40, d15.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			ctx.W.EmitMovRegMem(r41, r40, 0)
			ctx.FreeReg(r40)
			d40 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d40)
			ctx.FreeDesc(&d39)
			ctx.EnsureDesc(&d14)
			var d41 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r42, d14.Reg)
				ctx.W.EmitAndRegImm32(r42, 63)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d41)
			}
			if d41.Loc == scm.LocReg && d14.Loc == scm.LocReg && d41.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			d42 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d41)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() - d41.Imm.Int())}
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegReg(r43, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d43)
			} else if d42.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d42.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d41.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegReg(scratch, d42.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else {
				r44 := ctx.AllocRegExcept(d42.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r44, d42.Reg)
				ctx.W.EmitSubInt64(r44, d41.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d43)
			}
			if d43.Loc == scm.LocReg && d42.Loc == scm.LocReg && d43.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d40.Imm.Int()) >> uint64(d43.Imm.Int())))}
			} else if d43.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r45, d40.Reg)
				ctx.W.EmitShrRegImm8(r45, uint8(d43.Imm.Int()))
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d44)
			} else {
				{
					shiftSrc := d40.Reg
					r46 := ctx.AllocRegExcept(d40.Reg)
					ctx.W.EmitMovRegReg(r46, d40.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d43.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d43.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d43.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d44)
				}
			}
			if d44.Loc == scm.LocReg && d40.Loc == scm.LocReg && d44.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d43)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() | d44.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d44.Reg}
				ctx.BindReg(d44.Reg, &d45)
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r47, d19.Reg)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d45)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d44.Reg)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d45)
			} else if d44.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r48, d19.Reg)
				if d44.Imm.Int() >= -2147483648 && d44.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r48, int32(d44.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
					ctx.W.EmitOrInt64(r48, scm.RegR11)
				}
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d45)
			} else {
				r49 := ctx.AllocRegExcept(d19.Reg, d44.Reg)
				ctx.W.EmitMovRegReg(r49, d19.Reg)
				ctx.W.EmitOrInt64(r49, d44.Reg)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d45)
			}
			if d45.Loc == scm.LocReg && d19.Loc == scm.LocReg && d45.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			d46 := d45
			if d46.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d46)
			ctx.EmitStoreToStack(d46, 16)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl7)
			d47 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d47)
			ctx.BindReg(r32, &d47)
			if r12 { ctx.UnprotectReg(r13) }
			var d48 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d48)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d48.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d48.Reg)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d49)
			}
			ctx.FreeDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() + d49.Imm.Int())}
			} else if d49.Loc == scm.LocImm && d49.Imm.Int() == 0 {
				r52 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(r52, d47.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d50)
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d49.Reg}
				ctx.BindReg(d49.Reg, &d50)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else if d49.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegReg(scratch, d47.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			} else {
				r53 := ctx.AllocRegExcept(d47.Reg, d49.Reg)
				ctx.W.EmitMovRegReg(r53, d47.Reg)
				ctx.W.EmitAddInt64(r53, d49.Reg)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d50)
			}
			if d50.Loc == scm.LocReg && d47.Loc == scm.LocReg && d50.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d51 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r54, 32)
				ctx.W.EmitShrRegImm8(r54, 32)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d51)
			}
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d50.Imm.Int()) == uint64(d51.Imm.Int()))}
			} else if d51.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d50.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d50.Reg, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitCmpInt64(d50.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcE)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d52)
			} else if d50.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d51.Reg)
				ctx.W.EmitSetcc(r56, scm.CcE)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d52)
			} else {
				r57 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitCmpInt64(d50.Reg, d51.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d52)
			}
			ctx.FreeDesc(&d51)
			d53 := d52
			ctx.EnsureDesc(&d53)
			if d53.Loc != scm.LocImm && d53.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d53.Loc == scm.LocImm {
				if d53.Imm.Bool() {
					ctx.W.MarkLabel(lbl18)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl19)
					ctx.W.EmitJmp(lbl17)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d53.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
				ctx.W.MarkLabel(lbl18)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl19)
				ctx.W.EmitJmp(lbl17)
			}
			ctx.FreeDesc(&d52)
			ctx.W.MarkLabel(lbl3)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d54 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d54)
			ctx.BindReg(r2, &d54)
			ctx.W.EmitMakeNil(d54)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d55 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r58, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r58, 32)
				ctx.W.EmitShrRegImm8(r58, 32)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d55)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d55)
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d50.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d50.Imm.Int()) < uint64(d55.Imm.Int()))}
			} else if d55.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d50.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d50.Reg, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitCmpInt64(d50.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r59, scm.CcB)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d56)
			} else if d50.Loc == scm.LocImm {
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d50.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d55.Reg)
				ctx.W.EmitSetcc(r60, scm.CcB)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d56)
			} else {
				r61 := ctx.AllocRegExcept(d50.Reg)
				ctx.W.EmitCmpInt64(d50.Reg, d55.Reg)
				ctx.W.EmitSetcc(r61, scm.CcB)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d56)
			}
			ctx.FreeDesc(&d50)
			ctx.FreeDesc(&d55)
			d57 := d56
			ctx.EnsureDesc(&d57)
			if d57.Loc != scm.LocImm && d57.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d57.Loc == scm.LocImm {
				if d57.Imm.Bool() {
					ctx.W.MarkLabel(lbl22)
					ctx.W.EmitJmp(lbl20)
				} else {
					ctx.W.MarkLabel(lbl23)
					ctx.W.EmitJmp(lbl21)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d57.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl23)
				ctx.W.MarkLabel(lbl22)
				ctx.W.EmitJmp(lbl20)
				ctx.W.MarkLabel(lbl23)
				ctx.W.EmitJmp(lbl21)
			}
			ctx.FreeDesc(&d56)
			ctx.W.MarkLabel(lbl16)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d58 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r62, fieldAddr)
				ctx.W.EmitMovRegMem64(r63, fieldAddr+8)
				d58 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d58)
				ctx.BindReg(r63, &d58)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r64 := ctx.AllocReg()
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r64, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r65, thisptr.Reg, off+8)
				d58 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r64, Reg2: r65}
				ctx.BindReg(r64, &d58)
				ctx.BindReg(r65, &d58)
			}
			ctx.EnsureDesc(&d9)
			r66 := ctx.AllocReg()
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d58)
			if d9.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r66, uint64(d9.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r66, d9.Reg)
				ctx.W.EmitShlRegImm8(r66, 4)
			}
			if d58.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d58.Imm.Int()))
				ctx.W.EmitAddInt64(r66, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r66, d58.Reg)
			}
			r67 := ctx.AllocRegExcept(r66)
			r68 := ctx.AllocRegExcept(r66, r67)
			ctx.W.EmitMovRegMem(r67, r66, 0)
			ctx.W.EmitMovRegMem(r68, r66, 8)
			ctx.FreeReg(r66)
			d59 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r67, Reg2: r68}
			ctx.BindReg(r67, &d59)
			ctx.BindReg(r68, &d59)
			d60 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d60)
			ctx.BindReg(r2, &d60)
			ctx.EnsureDesc(&d59)
			if d59.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d59, &d60)
			} else {
				switch d59.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(d60, d59)
				case scm.TagInt:
					ctx.W.EmitMakeInt(d60, d59)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(d60, d59)
				case scm.TagNil:
					ctx.W.EmitMakeNil(d60)
				default:
					ctx.EmitMovPairToResult(&d59, &d60)
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl21)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d61 := d4
			if d61.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			d62 := d61
			if d62.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: d62.Type, Imm: scm.NewInt(int64(uint64(d62.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d62.Reg, 32)
				ctx.W.EmitShrRegImm8(d62.Reg, 32)
			}
			ctx.EmitStoreToStack(d62, 0)
			d63 := d9
			if d63.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d63)
			d64 := d63
			if d64.Loc == scm.LocImm {
				d64 = scm.JITValueDesc{Loc: scm.LocImm, Type: d64.Type, Imm: scm.NewInt(int64(uint64(d64.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d64.Reg, 32)
				ctx.W.EmitShrRegImm8(d64.Reg, 32)
			}
			ctx.EmitStoreToStack(d64, 8)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl20)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			var d65 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(scratch, d9.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d65)
			}
			if d65.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: d65.Type, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d65.Reg, 32)
				ctx.W.EmitShrRegImm8(d65.Reg, 32)
			}
			if d65.Loc == scm.LocReg && d9.Loc == scm.LocReg && d65.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			d66 := d65
			if d66.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			d67 := d66
			if d67.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: d67.Type, Imm: scm.NewInt(int64(uint64(d67.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d67.Reg, 32)
				ctx.W.EmitShrRegImm8(d67.Reg, 32)
			}
			ctx.EmitStoreToStack(d67, 0)
			d68 := d5
			if d68.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d68)
			d69 := d68
			if d69.Loc == scm.LocImm {
				d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: d69.Type, Imm: scm.NewInt(int64(uint64(d69.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d69.Reg, 32)
				ctx.W.EmitShrRegImm8(d69.Reg, 32)
			}
			ctx.EmitStoreToStack(d69, 8)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl0)
			d70 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d70)
			ctx.BindReg(r2, &d70)
			ctx.EmitMovPairToResult(&d70, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.W.PatchInt32(r0, int32(24))
			ctx.W.EmitAddRSP32(int32(24))
			return result
}

func (s *StorageSparse) scan(i uint32, value scm.Scmer) {
	if !value.IsNil() {
		s.recids.scan(uint32(s.i), scm.NewInt(int64(i)))
		s.i++
	}
}
func (s *StorageSparse) prepare() {
	s.i = 0
}
func (s *StorageSparse) init(i uint32) {
	s.values = make([]scm.Scmer, s.i)
	s.count = uint64(i)
	s.recids.init(uint32(s.i))
	s.i = 0
}
func (s *StorageSparse) build(i uint32, value scm.Scmer) {
	// store
	if !value.IsNil() {
		s.recids.build(uint32(s.i), scm.NewInt(int64(i)))
		s.values[s.i] = value
		s.i++
	}
}
func (s *StorageSparse) finish() {
	s.recids.finish()
}

// soley to StorageSparse
func (s *StorageSparse) proposeCompression(i uint32) ColumnStorage {
	return nil
}
