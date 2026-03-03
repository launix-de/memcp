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
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			lbl0 := ctx.W.ReserveLabel()
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
			lbl1 := ctx.W.ReserveLabel()
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
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
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
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d6.Loc == scm.LocImm {
				if d6.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d6.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d6)
			ctx.W.MarkLabel(lbl3)
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			ctx.EnsureDesc(&d4)
			ctx.EnsureDesc(&d5)
			var d7 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + d5.Imm.Int())}
			} else if d5.Loc == scm.LocImm && d5.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r9, d4.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d7)
			} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d5.Reg}
				ctx.BindReg(d5.Reg, &d7)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d5.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d5.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d7)
			} else if d5.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(scratch, d4.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d7)
			} else {
				r10 := ctx.AllocRegExcept(d4.Reg, d5.Reg)
				ctx.W.EmitMovRegReg(r10, d4.Reg)
				ctx.W.EmitAddInt64(r10, d5.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d7)
			}
			if d7.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: d7.Type, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d7.Reg, 32)
				ctx.W.EmitShrRegImm8(d7.Reg, 32)
			}
			if d7.Loc == scm.LocReg && d4.Loc == scm.LocReg && d7.Reg == d4.Reg {
				ctx.TransferReg(d4.Reg)
				d4.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d7)
			var d8 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r11, d7.Reg)
				ctx.W.EmitShrRegImm8(r11, 1)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d8)
			}
			if d8.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: d8.Type, Imm: scm.NewInt(int64(uint64(d8.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d8.Reg, 32)
				ctx.W.EmitShrRegImm8(d8.Reg, 32)
			}
			if d8.Loc == scm.LocReg && d7.Loc == scm.LocReg && d8.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d7)
			ctx.EnsureDesc(&d8)
			d9 := d8
			_ = d9
			r12 := d8.Loc == scm.LocReg
			r13 := d8.Reg
			if r12 { ctx.ProtectReg(r13) }
			lbl5 := ctx.W.ReserveLabel()
			ctx.EnsureDesc(&d9)
			ctx.EnsureDesc(&d9)
			var d10 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d9.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d9.Reg)
				ctx.W.EmitShlRegImm8(r14, 32)
				ctx.W.EmitShrRegImm8(r14, 32)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d10)
			}
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d11)
			}
			ctx.EnsureDesc(&d11)
			ctx.EnsureDesc(&d11)
			var d12 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d11.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d11.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d12)
			}
			ctx.FreeDesc(&d11)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d12)
			ctx.EnsureDesc(&d10)
			ctx.EnsureDesc(&d12)
			var d13 scm.JITValueDesc
			if d10.Loc == scm.LocImm && d12.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() * d12.Imm.Int())}
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d10.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d12.Reg)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d13)
			} else if d12.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(scratch, d10.Reg)
				if d12.Imm.Int() >= -2147483648 && d12.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d12.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d13)
			} else {
				r17 := ctx.AllocRegExcept(d10.Reg, d12.Reg)
				ctx.W.EmitMovRegReg(r17, d10.Reg)
				ctx.W.EmitImulInt64(r17, d12.Reg)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d13)
			}
			if d13.Loc == scm.LocReg && d10.Loc == scm.LocReg && d13.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			ctx.FreeDesc(&d12)
			var d14 scm.JITValueDesc
			r18 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r18, uint64(dataPtr))
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18, StackOff: int32(sliceLen)}
				ctx.BindReg(r18, &d14)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				ctx.W.EmitMovRegMem(r18, thisptr.Reg, off)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d14)
			}
			ctx.BindReg(r18, &d14)
			ctx.EnsureDesc(&d13)
			var d15 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() / 64)}
			} else {
				r19 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r19, d13.Reg)
				ctx.W.EmitShrRegImm8(r19, 6)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d15)
			}
			if d15.Loc == scm.LocReg && d13.Loc == scm.LocReg && d15.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d15)
			r20 := ctx.AllocReg()
			ctx.EnsureDesc(&d15)
			ctx.EnsureDesc(&d14)
			if d15.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r20, uint64(d15.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r20, d15.Reg)
				ctx.W.EmitShlRegImm8(r20, 3)
			}
			if d14.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.W.EmitAddInt64(r20, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r20, d14.Reg)
			}
			r21 := ctx.AllocRegExcept(r20)
			ctx.W.EmitMovRegMem(r21, r20, 0)
			ctx.FreeReg(r20)
			d16 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d16)
			ctx.FreeDesc(&d15)
			ctx.EnsureDesc(&d13)
			var d17 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r22, d13.Reg)
				ctx.W.EmitAndRegImm32(r22, 63)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d17)
			}
			if d17.Loc == scm.LocReg && d13.Loc == scm.LocReg && d17.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d16)
			ctx.EnsureDesc(&d17)
			var d18 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d17.Loc == scm.LocImm {
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d16.Imm.Int()) << uint64(d17.Imm.Int())))}
			} else if d17.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r23, d16.Reg)
				ctx.W.EmitShlRegImm8(r23, uint8(d17.Imm.Int()))
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d18)
			} else {
				{
					shiftSrc := d16.Reg
					r24 := ctx.AllocRegExcept(d16.Reg)
					ctx.W.EmitMovRegReg(r24, d16.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d17.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d17.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d17.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d18 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d18)
				}
			}
			if d18.Loc == scm.LocReg && d16.Loc == scm.LocReg && d18.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d16)
			ctx.FreeDesc(&d17)
			var d19 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d19)
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d19.Loc == scm.LocImm {
				if d19.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			d20 := d18
			if d20.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d20)
			ctx.EmitStoreToStack(d20, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d19.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			d21 := d18
			if d21.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d21)
			ctx.EmitStoreToStack(d21, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d19)
			ctx.W.MarkLabel(lbl7)
			d22 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d23 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d23)
			}
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d23)
			var d24 scm.JITValueDesc
			if d23.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d23.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d23.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d24)
			}
			ctx.FreeDesc(&d23)
			d25 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d24)
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d25.Imm.Int() - d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(r28, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d25.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegReg(scratch, d25.Reg)
				if d24.Imm.Int() >= -2147483648 && d24.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d24.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r29 := ctx.AllocRegExcept(d25.Reg, d24.Reg)
				ctx.W.EmitMovRegReg(r29, d25.Reg)
				ctx.W.EmitSubInt64(r29, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d26)
			}
			if d26.Loc == scm.LocReg && d25.Loc == scm.LocReg && d26.Reg == d25.Reg {
				ctx.TransferReg(d25.Reg)
				d25.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d26)
			var d27 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d22.Imm.Int()) >> uint64(d26.Imm.Int())))}
			} else if d26.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r30, d22.Reg)
				ctx.W.EmitShrRegImm8(r30, uint8(d26.Imm.Int()))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d27)
			} else {
				{
					shiftSrc := d22.Reg
					r31 := ctx.AllocRegExcept(d22.Reg)
					ctx.W.EmitMovRegReg(r31, d22.Reg)
					shiftSrc = r31
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d26.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d26.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d26.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d27)
				}
			}
			if d27.Loc == scm.LocReg && d22.Loc == scm.LocReg && d27.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.FreeDesc(&d26)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d27)
			ctx.EnsureDesc(&d27)
			if d27.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d27)
			}
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d13)
			var d28 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r33, d13.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d28)
			}
			if d28.Loc == scm.LocReg && d13.Loc == scm.LocReg && d28.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			var d29 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
				ctx.BindReg(r34, &d29)
			}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d29)
			var d30 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d29.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r35, d29.Reg)
				ctx.W.EmitShlRegImm8(r35, 56)
				ctx.W.EmitShrRegImm8(r35, 56)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d30)
			}
			ctx.FreeDesc(&d29)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d30)
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d28.Loc == scm.LocImm && d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + d30.Imm.Int())}
			} else if d30.Loc == scm.LocImm && d30.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(r36, d28.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d31)
			} else if d28.Loc == scm.LocImm && d28.Imm.Int() == 0 {
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
				ctx.BindReg(d30.Reg, &d31)
			} else if d28.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d28.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else if d30.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d28.Reg)
				ctx.W.EmitMovRegReg(scratch, d28.Reg)
				if d30.Imm.Int() >= -2147483648 && d30.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d30.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d30.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d31)
			} else {
				r37 := ctx.AllocRegExcept(d28.Reg, d30.Reg)
				ctx.W.EmitMovRegReg(r37, d28.Reg)
				ctx.W.EmitAddInt64(r37, d30.Reg)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d31)
			}
			if d31.Loc == scm.LocReg && d28.Loc == scm.LocReg && d31.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			ctx.FreeDesc(&d30)
			ctx.EnsureDesc(&d31)
			var d32 scm.JITValueDesc
			if d31.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d31.Imm.Int()) > uint64(64))}
			} else {
				r38 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitCmpRegImm32(d31.Reg, 64)
				ctx.W.EmitSetcc(r38, scm.CcA)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
				ctx.BindReg(r38, &d32)
			}
			ctx.FreeDesc(&d31)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d32.Loc == scm.LocImm {
				if d32.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d33 := d18
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d33)
			ctx.EmitStoreToStack(d33, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d32.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			d34 := d18
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d34)
			ctx.EmitStoreToStack(d34, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d32)
			ctx.W.MarkLabel(lbl9)
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d13)
			var d35 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r39, d13.Reg)
				ctx.W.EmitShrRegImm8(r39, 6)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d35)
			}
			if d35.Loc == scm.LocReg && d13.Loc == scm.LocReg && d35.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegReg(scratch, d35.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d36)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d14)
			if d36.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r40, uint64(d36.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r40, d36.Reg)
				ctx.W.EmitShlRegImm8(r40, 3)
			}
			if d14.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d14.Imm.Int()))
				ctx.W.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r40, d14.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			ctx.W.EmitMovRegMem(r41, r40, 0)
			ctx.FreeReg(r40)
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d37)
			ctx.FreeDesc(&d36)
			ctx.EnsureDesc(&d13)
			var d38 scm.JITValueDesc
			if d13.Loc == scm.LocImm {
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d13.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d13.Reg)
				ctx.W.EmitMovRegReg(r42, d13.Reg)
				ctx.W.EmitAndRegImm32(r42, 63)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d38)
			}
			if d38.Loc == scm.LocReg && d13.Loc == scm.LocReg && d38.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			d39 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d38)
			var d40 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d38.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() - d38.Imm.Int())}
			} else if d38.Loc == scm.LocImm && d38.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(r43, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegReg(scratch, d39.Reg)
				if d38.Imm.Int() >= -2147483648 && d38.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d38.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d38.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r44 := ctx.AllocRegExcept(d39.Reg, d38.Reg)
				ctx.W.EmitMovRegReg(r44, d39.Reg)
				ctx.W.EmitSubInt64(r44, d38.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d40)
			}
			if d40.Loc == scm.LocReg && d39.Loc == scm.LocReg && d40.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d37.Imm.Int()) >> uint64(d40.Imm.Int())))}
			} else if d40.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r45, d37.Reg)
				ctx.W.EmitShrRegImm8(r45, uint8(d40.Imm.Int()))
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d41)
			} else {
				{
					shiftSrc := d37.Reg
					r46 := ctx.AllocRegExcept(d37.Reg)
					ctx.W.EmitMovRegReg(r46, d37.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d40.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d40.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d40.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d41)
				}
			}
			if d41.Loc == scm.LocReg && d37.Loc == scm.LocReg && d41.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d18)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d18.Imm.Int() | d41.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d41.Reg}
				ctx.BindReg(d41.Reg, &d42)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r47, d18.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d42)
			} else if d18.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d18.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r48, d18.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r48, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitOrInt64(r48, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d42)
			} else {
				r49 := ctx.AllocRegExcept(d18.Reg, d41.Reg)
				ctx.W.EmitMovRegReg(r49, d18.Reg)
				ctx.W.EmitOrInt64(r49, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d42)
			}
			if d42.Loc == scm.LocReg && d18.Loc == scm.LocReg && d42.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			d43 := d42
			if d43.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d43)
			ctx.EmitStoreToStack(d43, 16)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			d44 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d44)
			ctx.BindReg(r32, &d44)
			if r12 { ctx.UnprotectReg(r13) }
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d45)
			}
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d45)
			var d46 scm.JITValueDesc
			if d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d45.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d46)
			}
			ctx.FreeDesc(&d45)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d46)
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d44.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d44.Imm.Int() + d46.Imm.Int())}
			} else if d46.Loc == scm.LocImm && d46.Imm.Int() == 0 {
				r52 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(r52, d44.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d47)
			} else if d44.Loc == scm.LocImm && d44.Imm.Int() == 0 {
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d46.Reg}
				ctx.BindReg(d46.Reg, &d47)
			} else if d44.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d44.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else if d46.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitMovRegReg(scratch, d44.Reg)
				if d46.Imm.Int() >= -2147483648 && d46.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d46.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d47)
			} else {
				r53 := ctx.AllocRegExcept(d44.Reg, d46.Reg)
				ctx.W.EmitMovRegReg(r53, d44.Reg)
				ctx.W.EmitAddInt64(r53, d46.Reg)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d47)
			}
			if d47.Loc == scm.LocReg && d44.Loc == scm.LocReg && d47.Reg == d44.Reg {
				ctx.TransferReg(d44.Reg)
				d44.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d44)
			ctx.FreeDesc(&d46)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d48 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r54, 32)
				ctx.W.EmitShrRegImm8(r54, 32)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d48)
			}
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d48)
			var d49 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d48.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d47.Imm.Int()) == uint64(d48.Imm.Int()))}
			} else if d48.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d47.Reg)
				if d48.Imm.Int() >= -2147483648 && d48.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d47.Reg, int32(d48.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d48.Imm.Int()))
					ctx.W.EmitCmpInt64(d47.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcE)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d49)
			} else if d47.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d48.Reg)
				ctx.W.EmitSetcc(r56, scm.CcE)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d49)
			} else {
				r57 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitCmpInt64(d47.Reg, d48.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d49)
			}
			ctx.FreeDesc(&d48)
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d49.Loc == scm.LocImm {
				if d49.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d49.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d49)
			ctx.W.MarkLabel(lbl2)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d50 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d50)
			ctx.BindReg(r2, &d50)
			ctx.W.EmitMakeNil(d50)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d51 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r58, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r58, 32)
				ctx.W.EmitShrRegImm8(r58, 32)
				d51 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r58}
				ctx.BindReg(r58, &d51)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d51)
			var d52 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d51.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d47.Imm.Int()) < uint64(d51.Imm.Int()))}
			} else if d51.Loc == scm.LocImm {
				r59 := ctx.AllocRegExcept(d47.Reg)
				if d51.Imm.Int() >= -2147483648 && d51.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d47.Reg, int32(d51.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
					ctx.W.EmitCmpInt64(d47.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r59, scm.CcB)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d52)
			} else if d47.Loc == scm.LocImm {
				r60 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d51.Reg)
				ctx.W.EmitSetcc(r60, scm.CcB)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d52)
			} else {
				r61 := ctx.AllocRegExcept(d47.Reg)
				ctx.W.EmitCmpInt64(d47.Reg, d51.Reg)
				ctx.W.EmitSetcc(r61, scm.CcB)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d52)
			}
			ctx.FreeDesc(&d47)
			ctx.FreeDesc(&d51)
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d52.Loc == scm.LocImm {
				if d52.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d52.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d52)
			ctx.W.MarkLabel(lbl11)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var d53 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r62, fieldAddr)
				ctx.W.EmitMovRegMem64(r63, fieldAddr+8)
				d53 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d53)
				ctx.BindReg(r63, &d53)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r64 := ctx.AllocReg()
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r64, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r65, thisptr.Reg, off+8)
				d53 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r64, Reg2: r65}
				ctx.BindReg(r64, &d53)
				ctx.BindReg(r65, &d53)
			}
			ctx.EnsureDesc(&d8)
			r66 := ctx.AllocReg()
			ctx.EnsureDesc(&d8)
			ctx.EnsureDesc(&d53)
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r66, uint64(d8.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r66, d8.Reg)
				ctx.W.EmitShlRegImm8(r66, 4)
			}
			if d53.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
				ctx.W.EmitAddInt64(r66, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r66, d53.Reg)
			}
			r67 := ctx.AllocRegExcept(r66)
			r68 := ctx.AllocRegExcept(r66, r67)
			ctx.W.EmitMovRegMem(r67, r66, 0)
			ctx.W.EmitMovRegMem(r68, r66, 8)
			ctx.FreeReg(r66)
			d54 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r67, Reg2: r68}
			ctx.BindReg(r67, &d54)
			ctx.BindReg(r68, &d54)
			d55 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d55)
			ctx.BindReg(r2, &d55)
			ctx.EnsureDesc(&d54)
			if d54.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d54, &d55)
			} else {
				switch d54.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(d55, d54)
				case scm.TagInt:
					ctx.W.EmitMakeInt(d55, d54)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(d55, d54)
				case scm.TagNil:
					ctx.W.EmitMakeNil(d55)
				default:
					ctx.EmitMovPairToResult(&d54, &d55)
				}
			}
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			d56 := d4
			if d56.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d56)
			d57 := d56
			if d57.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: d57.Type, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d57.Reg, 32)
				ctx.W.EmitShrRegImm8(d57.Reg, 32)
			}
			ctx.EmitStoreToStack(d57, 0)
			d58 := d8
			if d58.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d58)
			d59 := d58
			if d59.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: d59.Type, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d59.Reg, 32)
				ctx.W.EmitShrRegImm8(d59.Reg, 32)
			}
			ctx.EmitStoreToStack(d59, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl14)
			d4 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			ctx.EnsureDesc(&d8)
			var d60 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(scratch, d8.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			}
			if d60.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: d60.Type, Imm: scm.NewInt(int64(uint64(d60.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d60.Reg, 32)
				ctx.W.EmitShrRegImm8(d60.Reg, 32)
			}
			if d60.Loc == scm.LocReg && d8.Loc == scm.LocReg && d60.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			d61 := d60
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
			d63 := d5
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
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			d65 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d65)
			ctx.BindReg(r2, &d65)
			ctx.EmitMovPairToResult(&d65, &result)
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
