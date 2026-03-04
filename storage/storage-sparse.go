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
					ctx.W.EmitJmp(lbl5)
				} else {
					ctx.W.EmitJmp(lbl6)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d7.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl5)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.W.MarkLabel(lbl5)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl6)
			ctx.W.EmitJmp(lbl4)
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl4)
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
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
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d21.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl11)
				ctx.W.EmitJmp(lbl12)
			}
			ctx.W.MarkLabel(lbl11)
			ctx.W.EmitJmp(lbl9)
			ctx.W.MarkLabel(lbl12)
			d22 := d19
			if d22.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d22)
			ctx.EmitStoreToStack(d22, 16)
			ctx.W.EmitJmp(lbl10)
			ctx.FreeDesc(&d20)
			ctx.W.MarkLabel(lbl10)
			d23 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d24.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d25)
			}
			ctx.FreeDesc(&d24)
			d26 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d26)
			ctx.EnsureDesc(&d25)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() - d25.Imm.Int())}
			} else if d25.Loc == scm.LocImm && d25.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r28, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d27)
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
				r29 := ctx.AllocRegExcept(d26.Reg, d25.Reg)
				ctx.W.EmitMovRegReg(r29, d26.Reg)
				ctx.W.EmitSubInt64(r29, d25.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d27)
			}
			if d27.Loc == scm.LocReg && d26.Loc == scm.LocReg && d27.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d27)
			var d28 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d23.Imm.Int()) >> uint64(d27.Imm.Int())))}
			} else if d27.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(r30, d23.Reg)
				ctx.W.EmitShrRegImm8(r30, uint8(d27.Imm.Int()))
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d28)
			} else {
				{
					shiftSrc := d23.Reg
					r31 := ctx.AllocRegExcept(d23.Reg)
					ctx.W.EmitMovRegReg(r31, d23.Reg)
					shiftSrc = r31
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
			if d28.Loc == scm.LocReg && d23.Loc == scm.LocReg && d28.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d27)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d28)
			if d28.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d28)
			}
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl9)
			d23 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d14)
			var d29 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r33, d14.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d29)
			}
			if d29.Loc == scm.LocReg && d14.Loc == scm.LocReg && d29.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			var d30 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
				ctx.BindReg(r34, &d30)
			}
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d30.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r35, d30.Reg)
				ctx.W.EmitShlRegImm8(r35, 56)
				ctx.W.EmitShrRegImm8(r35, 56)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d31)
			}
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r36, d29.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d32)
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
				ctx.BindReg(d31.Reg, &d32)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d29.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(scratch, d29.Reg)
				if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d31.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d32)
			} else {
				r37 := ctx.AllocRegExcept(d29.Reg, d31.Reg)
				ctx.W.EmitMovRegReg(r37, d29.Reg)
				ctx.W.EmitAddInt64(r37, d31.Reg)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d32)
			}
			if d32.Loc == scm.LocReg && d29.Loc == scm.LocReg && d32.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d31)
			ctx.EnsureDesc(&d32)
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d32.Imm.Int()) > uint64(64))}
			} else {
				r38 := ctx.AllocRegExcept(d32.Reg)
				ctx.W.EmitCmpRegImm32(d32.Reg, 64)
				ctx.W.EmitSetcc(r38, scm.CcA)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
				ctx.BindReg(r38, &d33)
			}
			ctx.FreeDesc(&d32)
			d34 := d33
			ctx.EnsureDesc(&d34)
			if d34.Loc != scm.LocImm && d34.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d34.Loc == scm.LocImm {
				if d34.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d34.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
			}
			ctx.W.MarkLabel(lbl14)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl15)
			d35 := d19
			if d35.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			ctx.EmitStoreToStack(d35, 16)
			ctx.W.EmitJmp(lbl10)
			ctx.FreeDesc(&d33)
			ctx.W.MarkLabel(lbl13)
			d23 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d14)
			var d36 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r39, d14.Reg)
				ctx.W.EmitShrRegImm8(r39, 6)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d36)
			}
			if d36.Loc == scm.LocReg && d14.Loc == scm.LocReg && d36.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(scratch, d36.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d37)
			}
			if d37.Loc == scm.LocReg && d36.Loc == scm.LocReg && d37.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d37)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d15)
			if d37.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r40, uint64(d37.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r40, d37.Reg)
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
			d38 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d38)
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d14)
			var d39 scm.JITValueDesc
			if d14.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d14.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r42, d14.Reg)
				ctx.W.EmitAndRegImm32(r42, 63)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d39)
			}
			if d39.Loc == scm.LocReg && d14.Loc == scm.LocReg && d39.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			d40 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d39)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d40.Imm.Int() - d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(r43, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d41)
			} else if d40.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d39.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegReg(scratch, d40.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else {
				r44 := ctx.AllocRegExcept(d40.Reg, d39.Reg)
				ctx.W.EmitMovRegReg(r44, d40.Reg)
				ctx.W.EmitSubInt64(r44, d39.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d41)
			}
			if d41.Loc == scm.LocReg && d40.Loc == scm.LocReg && d41.Reg == d40.Reg {
				ctx.TransferReg(d40.Reg)
				d40.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) >> uint64(d41.Imm.Int())))}
			} else if d41.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r45, d38.Reg)
				ctx.W.EmitShrRegImm8(r45, uint8(d41.Imm.Int()))
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d42)
			} else {
				{
					shiftSrc := d38.Reg
					r46 := ctx.AllocRegExcept(d38.Reg)
					ctx.W.EmitMovRegReg(r46, d38.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d41.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d41.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d41.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d42)
				}
			}
			if d42.Loc == scm.LocReg && d38.Loc == scm.LocReg && d42.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d41)
			ctx.EnsureDesc(&d19)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d19.Loc == scm.LocImm && d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() | d42.Imm.Int())}
			} else if d19.Loc == scm.LocImm && d19.Imm.Int() == 0 {
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d42.Reg}
				ctx.BindReg(d42.Reg, &d43)
			} else if d42.Loc == scm.LocImm && d42.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r47, d19.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d43)
			} else if d19.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d42.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d19.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d43)
			} else if d42.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r48, d19.Reg)
				if d42.Imm.Int() >= -2147483648 && d42.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r48, int32(d42.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d42.Imm.Int()))
					ctx.W.EmitOrInt64(r48, scm.RegR11)
				}
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d43)
			} else {
				r49 := ctx.AllocRegExcept(d19.Reg, d42.Reg)
				ctx.W.EmitMovRegReg(r49, d19.Reg)
				ctx.W.EmitOrInt64(r49, d42.Reg)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d43)
			}
			if d43.Loc == scm.LocReg && d19.Loc == scm.LocReg && d43.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d42)
			d44 := d43
			if d44.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d44)
			ctx.EmitStoreToStack(d44, 16)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl7)
			d45 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d45)
			ctx.BindReg(r32, &d45)
			if r12 { ctx.UnprotectReg(r13) }
			var d46 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d46)
			}
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d46.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d47)
			}
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d47)
			var d48 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d45.Imm.Int() + d47.Imm.Int())}
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				r52 := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(r52, d45.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d48)
			} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d47.Reg}
				ctx.BindReg(d47.Reg, &d48)
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegReg(scratch, d45.Reg)
				if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d47.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d48)
			} else {
				r53 := ctx.AllocRegExcept(d45.Reg, d47.Reg)
				ctx.W.EmitMovRegReg(r53, d45.Reg)
				ctx.W.EmitAddInt64(r53, d47.Reg)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d48)
			}
			if d48.Loc == scm.LocReg && d45.Loc == scm.LocReg && d48.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.FreeDesc(&d47)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d49 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r54, 32)
				ctx.W.EmitShrRegImm8(r54, 32)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d49)
			}
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d48.Imm.Int()) == uint64(d49.Imm.Int()))}
			} else if d49.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d48.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d48.Reg, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitCmpInt64(d48.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcE)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d50)
			} else if d48.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d49.Reg)
				ctx.W.EmitSetcc(r56, scm.CcE)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d50)
			} else {
				r57 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitCmpInt64(d48.Reg, d49.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d50)
			}
			ctx.FreeDesc(&d49)
			d51 := d50
			ctx.EnsureDesc(&d51)
			if d51.Loc != scm.LocImm && d51.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			lbl19 := ctx.W.ReserveLabel()
			if d51.Loc == scm.LocImm {
				if d51.Imm.Bool() {
					ctx.W.EmitJmp(lbl18)
				} else {
					ctx.W.EmitJmp(lbl19)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d51.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl18)
				ctx.W.EmitJmp(lbl19)
			}
			ctx.W.MarkLabel(lbl18)
			ctx.W.EmitJmp(lbl16)
			ctx.W.MarkLabel(lbl19)
			ctx.W.EmitJmp(lbl17)
			ctx.FreeDesc(&d50)
			ctx.W.MarkLabel(lbl3)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d52 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d52)
			ctx.BindReg(r2, &d52)
			ctx.W.EmitMakeNil(d52)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl17)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d53 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r58, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r58, 32)
				ctx.W.EmitShrRegImm8(r58, 32)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d53)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d53)
			var d54 scm.JITValueDesc
			if d48.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d48.Imm.Int()) < uint64(d53.Imm.Int()))}
			} else if d53.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d48.Reg)
				if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d48.Reg, int32(d53.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
					ctx.W.EmitCmpInt64(d48.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r59, scm.CcB)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d54)
			} else if d48.Loc == scm.LocImm {
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d53.Reg)
				ctx.W.EmitSetcc(r60, scm.CcB)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d54)
			} else {
				r61 := ctx.AllocRegExcept(d48.Reg)
				ctx.W.EmitCmpInt64(d48.Reg, d53.Reg)
				ctx.W.EmitSetcc(r61, scm.CcB)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d54)
			}
			ctx.FreeDesc(&d48)
			ctx.FreeDesc(&d53)
			d55 := d54
			ctx.EnsureDesc(&d55)
			if d55.Loc != scm.LocImm && d55.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl20 := ctx.W.ReserveLabel()
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			lbl23 := ctx.W.ReserveLabel()
			if d55.Loc == scm.LocImm {
				if d55.Imm.Bool() {
					ctx.W.EmitJmp(lbl22)
				} else {
					ctx.W.EmitJmp(lbl23)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d55.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl22)
				ctx.W.EmitJmp(lbl23)
			}
			ctx.W.MarkLabel(lbl22)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl23)
			ctx.W.EmitJmp(lbl21)
			ctx.FreeDesc(&d54)
			ctx.W.MarkLabel(lbl16)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d56 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r62, fieldAddr)
				ctx.W.EmitMovRegMem64(r63, fieldAddr+8)
				d56 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d56)
				ctx.BindReg(r63, &d56)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r64 := ctx.AllocReg()
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r64, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r65, thisptr.Reg, off+8)
				d56 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r64, Reg2: r65}
				ctx.BindReg(r64, &d56)
				ctx.BindReg(r65, &d56)
			}
			ctx.EnsureDesc(&d9)
			r66 := ctx.AllocReg()
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d56)
			if d9.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r66, uint64(d9.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r66, d9.Reg)
				ctx.W.EmitShlRegImm8(r66, 4)
			}
			if d56.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d56.Imm.Int()))
				ctx.W.EmitAddInt64(r66, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r66, d56.Reg)
			}
			r67 := ctx.AllocRegExcept(r66)
			r68 := ctx.AllocRegExcept(r66, r67)
			ctx.W.EmitMovRegMem(r67, r66, 0)
			ctx.W.EmitMovRegMem(r68, r66, 8)
			ctx.FreeReg(r66)
			d57 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r67, Reg2: r68}
			ctx.BindReg(r67, &d57)
			ctx.BindReg(r68, &d57)
			d58 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d58)
			ctx.BindReg(r2, &d58)
			ctx.EnsureDesc(&d57)
			if d57.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d57, &d58)
			} else {
				switch d57.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(d58, d57)
				case scm.TagInt:
					ctx.W.EmitMakeInt(d58, d57)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(d58, d57)
				case scm.TagNil:
					ctx.W.EmitMakeNil(d58)
				default:
					ctx.EmitMovPairToResult(&d57, &d58)
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl21)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d59 := d4
			if d59.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d59)
			d60 := d59
			if d60.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: d60.Type, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d60.Reg, 32)
				ctx.W.EmitShrRegImm8(d60.Reg, 32)
			}
			ctx.EmitStoreToStack(d60, 0)
			d61 := d9
			if d61.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d61)
			d62 := d61
			if d62.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: d62.Type, Imm: scm.NewInt(int64(uint64(d62.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d62.Reg, 32)
				ctx.W.EmitShrRegImm8(d62.Reg, 32)
			}
			ctx.EmitStoreToStack(d62, 8)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl20)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			var d63 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(scratch, d9.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d63)
			}
			if d63.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: d63.Type, Imm: scm.NewInt(int64(uint64(d63.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d63.Reg, 32)
				ctx.W.EmitShrRegImm8(d63.Reg, 32)
			}
			if d63.Loc == scm.LocReg && d9.Loc == scm.LocReg && d63.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			d64 := d63
			if d64.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d64)
			d65 := d64
			if d65.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: d65.Type, Imm: scm.NewInt(int64(uint64(d65.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d65.Reg, 32)
				ctx.W.EmitShrRegImm8(d65.Reg, 32)
			}
			ctx.EmitStoreToStack(d65, 0)
			d66 := d5
			if d66.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d66)
			d67 := d66
			if d67.Loc == scm.LocImm {
				d67 = scm.JITValueDesc{Loc: scm.LocImm, Type: d67.Type, Imm: scm.NewInt(int64(uint64(d67.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d67.Reg, 32)
				ctx.W.EmitShrRegImm8(d67.Reg, 32)
			}
			ctx.EmitStoreToStack(d67, 8)
			ctx.W.EmitJmp(lbl2)
			ctx.W.MarkLabel(lbl0)
			d68 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d68)
			ctx.BindReg(r2, &d68)
			ctx.EmitMovPairToResult(&d68, &result)
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
