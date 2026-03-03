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
				if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
				if idxInt.Loc != scm.LocReg { panic("jit: idxInt not in register") }
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			r0 := ctx.W.EmitSubRSP32Fixup()
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
				r1 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r1, fieldAddr)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r1}
				ctx.BindReg(r1, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r2, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
				ctx.BindReg(r2, &d0)
			}
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			if d0.Loc == scm.LocStack || d0.Loc == scm.LocStackPair { ctx.EnsureDesc(&d0) }
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d0.Imm.Int()))))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r3, d0.Reg)
				ctx.W.EmitShlRegImm8(r3, 32)
				ctx.W.EmitShrRegImm8(r3, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
				ctx.BindReg(r3, &d1)
			}
			lbl1 := ctx.W.ReserveLabel()
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
			d2 := d1
			if d2.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			d3 := d2
			if d3.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: d3.Type, Imm: scm.NewInt(int64(uint64(d3.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d3.Reg, 32)
				ctx.W.EmitShrRegImm8(d3.Reg, 32)
			}
			ctx.EmitStoreToStack(d3, 8)
			ctx.W.MarkLabel(lbl1)
			d4 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d5 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			var d6 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d4.Imm.Int()) == uint64(d5.Imm.Int()))}
			} else if d5.Loc == scm.LocImm {
				r4 := ctx.AllocRegExcept(d4.Reg)
				if d5.Imm.Int() >= -2147483648 && d5.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d4.Reg, int32(d5.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d5.Imm.Int()))
					ctx.W.EmitCmpInt64(d4.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r4, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r4}
				ctx.BindReg(r4, &d6)
			} else if d4.Loc == scm.LocImm {
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d5.Reg)
				ctx.W.EmitSetcc(r5, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
				ctx.BindReg(r5, &d6)
			} else {
				r6 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitCmpInt64(d4.Reg, d5.Reg)
				ctx.W.EmitSetcc(r6, scm.CcE)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d6)
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
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			if d5.Loc == scm.LocStack || d5.Loc == scm.LocStackPair { ctx.EnsureDesc(&d5) }
			var d7 scm.JITValueDesc
			if d4.Loc == scm.LocImm && d5.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d4.Imm.Int() + d5.Imm.Int())}
			} else if d5.Loc == scm.LocImm && d5.Imm.Int() == 0 {
				r7 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r7, d4.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r7}
				ctx.BindReg(r7, &d7)
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
				r8 := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegReg(r8, d4.Reg)
				ctx.W.EmitAddInt64(r8, d5.Reg)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
				ctx.BindReg(r8, &d7)
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
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			var d8 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() / 2)}
			} else {
				r9 := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(r9, d7.Reg)
				ctx.W.EmitShrRegImm8(r9, 1)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d8)
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
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			r10 := d8.Loc == scm.LocReg
			r11 := d8.Reg
			if r10 { ctx.ProtectReg(r11) }
			lbl5 := ctx.W.ReserveLabel()
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			var d9 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d8.Imm.Int()))))}
			} else {
				r12 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r12, d8.Reg)
				ctx.W.EmitShlRegImm8(r12, 32)
				ctx.W.EmitShrRegImm8(r12, 32)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r12}
				ctx.BindReg(r12, &d9)
			}
			var d10 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r13, thisptr.Reg, off)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
				ctx.BindReg(r13, &d10)
			}
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			var d11 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d10.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d10.Reg)
				ctx.W.EmitShlRegImm8(r14, 56)
				ctx.W.EmitShrRegImm8(r14, 56)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d11)
			}
			ctx.FreeDesc(&d10)
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d12 scm.JITValueDesc
			if d9.Loc == scm.LocImm && d11.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d9.Imm.Int() * d11.Imm.Int())}
			} else if d9.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d9.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else if d11.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(scratch, d9.Reg)
				if d11.Imm.Int() >= -2147483648 && d11.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d11.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d12)
			} else {
				r15 := ctx.AllocRegExcept(d9.Reg)
				ctx.W.EmitMovRegReg(r15, d9.Reg)
				ctx.W.EmitImulInt64(r15, d11.Reg)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r15}
				ctx.BindReg(r15, &d12)
			}
			if d12.Loc == scm.LocReg && d9.Loc == scm.LocReg && d12.Reg == d9.Reg {
				ctx.TransferReg(d9.Reg)
				d9.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d9)
			ctx.FreeDesc(&d11)
			var d13 scm.JITValueDesc
			r16 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r16, uint64(dataPtr))
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16, StackOff: int32(sliceLen)}
				ctx.BindReg(r16, &d13)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				ctx.W.EmitMovRegMem(r16, thisptr.Reg, off)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r16}
				ctx.BindReg(r16, &d13)
			}
			ctx.BindReg(r16, &d13)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d14 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r17 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r17, d12.Reg)
				ctx.W.EmitShrRegImm8(r17, 6)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d14)
			}
			if d14.Loc == scm.LocReg && d12.Loc == scm.LocReg && d14.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			r18 := ctx.AllocReg()
			if d14.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r18, uint64(d14.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r18, d14.Reg)
				ctx.W.EmitShlRegImm8(r18, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r18, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r18, d13.Reg)
			}
			r19 := ctx.AllocRegExcept(r18)
			ctx.W.EmitMovRegMem(r19, r18, 0)
			ctx.FreeReg(r18)
			d15 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r19}
			ctx.BindReg(r19, &d15)
			ctx.FreeDesc(&d14)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d16 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r20 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r20, d12.Reg)
				ctx.W.EmitAndRegImm32(r20, 63)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r20}
				ctx.BindReg(r20, &d16)
			}
			if d16.Loc == scm.LocReg && d12.Loc == scm.LocReg && d16.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			var d17 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d16.Loc == scm.LocImm {
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d15.Imm.Int()) << uint64(d16.Imm.Int())))}
			} else if d16.Loc == scm.LocImm {
				r21 := ctx.AllocRegExcept(d15.Reg)
				ctx.W.EmitMovRegReg(r21, d15.Reg)
				ctx.W.EmitShlRegImm8(r21, uint8(d16.Imm.Int()))
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
				ctx.BindReg(r21, &d17)
			} else {
				{
					shiftSrc := d15.Reg
					r22 := ctx.AllocRegExcept(d15.Reg)
					ctx.W.EmitMovRegReg(r22, d15.Reg)
					shiftSrc = r22
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d16.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d16.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d16.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d17 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d17)
				}
			}
			if d17.Loc == scm.LocReg && d15.Loc == scm.LocReg && d17.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d15)
			ctx.FreeDesc(&d16)
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r23 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r23, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r23}
				ctx.BindReg(r23, &d18)
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d18.Loc == scm.LocImm {
				if d18.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			d19 := d17
			if d19.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			ctx.EmitStoreToStack(d19, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d18.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			d20 := d17
			if d20.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			ctx.EmitStoreToStack(d20, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d18)
			ctx.W.MarkLabel(lbl7)
			d21 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d22 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
				ctx.BindReg(r24, &d22)
			}
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d22.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d22.Reg)
				ctx.W.EmitShlRegImm8(r25, 56)
				ctx.W.EmitShrRegImm8(r25, 56)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
				ctx.BindReg(r25, &d23)
			}
			ctx.FreeDesc(&d22)
			d24 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d23.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() - d23.Imm.Int())}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				r26 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r26, d24.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
				ctx.BindReg(r26, &d25)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d23.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(scratch, d24.Reg)
				if d23.Imm.Int() >= -2147483648 && d23.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d23.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d23.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d25)
			} else {
				r27 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r27, d24.Reg)
				ctx.W.EmitSubInt64(r27, d23.Reg)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d25)
			}
			if d25.Loc == scm.LocReg && d24.Loc == scm.LocReg && d25.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			if d21.Loc == scm.LocStack || d21.Loc == scm.LocStackPair { ctx.EnsureDesc(&d21) }
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			var d26 scm.JITValueDesc
			if d21.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d21.Imm.Int()) >> uint64(d25.Imm.Int())))}
			} else if d25.Loc == scm.LocImm {
				r28 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r28, d21.Reg)
				ctx.W.EmitShrRegImm8(r28, uint8(d25.Imm.Int()))
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d26)
			} else {
				{
					shiftSrc := d21.Reg
					r29 := ctx.AllocRegExcept(d21.Reg)
					ctx.W.EmitMovRegReg(r29, d21.Reg)
					shiftSrc = r29
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d25.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d25.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d25.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d26)
				}
			}
			if d26.Loc == scm.LocReg && d21.Loc == scm.LocReg && d26.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d21)
			ctx.FreeDesc(&d25)
			r30 := ctx.AllocReg()
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			ctx.EmitMovToReg(r30, d26)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d27 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r31 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r31, d12.Reg)
				ctx.W.EmitAndRegImm32(r31, 63)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d27)
			}
			if d27.Loc == scm.LocReg && d12.Loc == scm.LocReg && d27.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			var d28 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r32 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r32, thisptr.Reg, off)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
				ctx.BindReg(r32, &d28)
			}
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			if d28.Loc == scm.LocStack || d28.Loc == scm.LocStackPair { ctx.EnsureDesc(&d28) }
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d28.Imm.Int()))))}
			} else {
				r33 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r33, d28.Reg)
				ctx.W.EmitShlRegImm8(r33, 56)
				ctx.W.EmitShrRegImm8(r33, 56)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d29)
			}
			ctx.FreeDesc(&d28)
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
			var d30 scm.JITValueDesc
			if d27.Loc == scm.LocImm && d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d27.Imm.Int() + d29.Imm.Int())}
			} else if d29.Loc == scm.LocImm && d29.Imm.Int() == 0 {
				r34 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r34, d27.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d30)
			} else if d27.Loc == scm.LocImm && d27.Imm.Int() == 0 {
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
				ctx.BindReg(d29.Reg, &d30)
			} else if d27.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else if d29.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(scratch, d27.Reg)
				if d29.Imm.Int() >= -2147483648 && d29.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d29.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d29.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d30)
			} else {
				r35 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitMovRegReg(r35, d27.Reg)
				ctx.W.EmitAddInt64(r35, d29.Reg)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d30)
			}
			if d30.Loc == scm.LocReg && d27.Loc == scm.LocReg && d30.Reg == d27.Reg {
				ctx.TransferReg(d27.Reg)
				d27.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d27)
			ctx.FreeDesc(&d29)
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			var d31 scm.JITValueDesc
			if d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d30.Imm.Int()) > uint64(64))}
			} else {
				r36 := ctx.AllocRegExcept(d30.Reg)
				ctx.W.EmitCmpRegImm32(d30.Reg, 64)
				ctx.W.EmitSetcc(r36, scm.CcA)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r36}
				ctx.BindReg(r36, &d31)
			}
			ctx.FreeDesc(&d30)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d31.Loc == scm.LocImm {
				if d31.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			d32 := d17
			if d32.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			ctx.EmitStoreToStack(d32, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d31.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			d33 := d17
			if d33.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d33.Loc == scm.LocStack || d33.Loc == scm.LocStackPair { ctx.EnsureDesc(&d33) }
			ctx.EmitStoreToStack(d33, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d31)
			ctx.W.MarkLabel(lbl9)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d34 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() / 64)}
			} else {
				r37 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r37, d12.Reg)
				ctx.W.EmitShrRegImm8(r37, 6)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d34)
			}
			if d34.Loc == scm.LocReg && d12.Loc == scm.LocReg && d34.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d35 scm.JITValueDesc
			if d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d34.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d34.Reg)
				ctx.W.EmitMovRegReg(scratch, d34.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			}
			if d35.Loc == scm.LocReg && d34.Loc == scm.LocReg && d35.Reg == d34.Reg {
				ctx.TransferReg(d34.Reg)
				d34.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			r38 := ctx.AllocReg()
			if d35.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r38, uint64(d35.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r38, d35.Reg)
				ctx.W.EmitShlRegImm8(r38, 3)
			}
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d13.Imm.Int()))
				ctx.W.EmitAddInt64(r38, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r38, d13.Reg)
			}
			r39 := ctx.AllocRegExcept(r38)
			ctx.W.EmitMovRegMem(r39, r38, 0)
			ctx.FreeReg(r38)
			d36 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r39}
			ctx.BindReg(r39, &d36)
			ctx.FreeDesc(&d35)
			if d12.Loc == scm.LocStack || d12.Loc == scm.LocStackPair { ctx.EnsureDesc(&d12) }
			var d37 scm.JITValueDesc
			if d12.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d12.Imm.Int() % 64)}
			} else {
				r40 := ctx.AllocRegExcept(d12.Reg)
				ctx.W.EmitMovRegReg(r40, d12.Reg)
				ctx.W.EmitAndRegImm32(r40, 63)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d37)
			}
			if d37.Loc == scm.LocReg && d12.Loc == scm.LocReg && d37.Reg == d12.Reg {
				ctx.TransferReg(d12.Reg)
				d12.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d12)
			d38 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() - d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r41 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r41, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r41}
				ctx.BindReg(r41, &d39)
			} else if d38.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d38.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(scratch, d38.Reg)
				if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d37.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d39)
			} else {
				r42 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r42, d38.Reg)
				ctx.W.EmitSubInt64(r42, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d39)
			}
			if d39.Loc == scm.LocReg && d38.Loc == scm.LocReg && d39.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			if d36.Loc == scm.LocStack || d36.Loc == scm.LocStackPair { ctx.EnsureDesc(&d36) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) >> uint64(d39.Imm.Int())))}
			} else if d39.Loc == scm.LocImm {
				r43 := ctx.AllocRegExcept(d36.Reg)
				ctx.W.EmitMovRegReg(r43, d36.Reg)
				ctx.W.EmitShrRegImm8(r43, uint8(d39.Imm.Int()))
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d40)
			} else {
				{
					shiftSrc := d36.Reg
					r44 := ctx.AllocRegExcept(d36.Reg)
					ctx.W.EmitMovRegReg(r44, d36.Reg)
					shiftSrc = r44
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d39.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d39.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d39.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d40)
				}
			}
			if d40.Loc == scm.LocReg && d36.Loc == scm.LocReg && d40.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d39)
			if d17.Loc == scm.LocStack || d17.Loc == scm.LocStackPair { ctx.EnsureDesc(&d17) }
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			var d41 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d17.Imm.Int() | d40.Imm.Int())}
			} else if d17.Loc == scm.LocImm && d17.Imm.Int() == 0 {
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d40.Reg}
				ctx.BindReg(d40.Reg, &d41)
			} else if d40.Loc == scm.LocImm && d40.Imm.Int() == 0 {
				r45 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r45, d17.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d41)
			} else if d17.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d17.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d41)
			} else if d40.Loc == scm.LocImm {
				r46 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r46, d17.Reg)
				if d40.Imm.Int() >= -2147483648 && d40.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r46, int32(d40.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
					ctx.W.EmitOrInt64(r46, scm.RegR11)
				}
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d41)
			} else {
				r47 := ctx.AllocRegExcept(d17.Reg)
				ctx.W.EmitMovRegReg(r47, d17.Reg)
				ctx.W.EmitOrInt64(r47, d40.Reg)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d41)
			}
			if d41.Loc == scm.LocReg && d17.Loc == scm.LocReg && d41.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d40)
			d42 := d41
			if d42.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d42.Loc == scm.LocStack || d42.Loc == scm.LocStackPair { ctx.EnsureDesc(&d42) }
			ctx.EmitStoreToStack(d42, 16)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d43 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r30}
			ctx.BindReg(r30, &d43)
			ctx.BindReg(r30, &d43)
			if r10 { ctx.UnprotectReg(r11) }
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r48 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r48, thisptr.Reg, off)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r48}
				ctx.BindReg(r48, &d44)
			}
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			if d44.Loc == scm.LocStack || d44.Loc == scm.LocStackPair { ctx.EnsureDesc(&d44) }
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d44.Imm.Int()))))}
			} else {
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r49, d44.Reg)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d45)
			}
			ctx.FreeDesc(&d44)
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			if d45.Loc == scm.LocStack || d45.Loc == scm.LocStackPair { ctx.EnsureDesc(&d45) }
			var d46 scm.JITValueDesc
			if d43.Loc == scm.LocImm && d45.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d43.Imm.Int() + d45.Imm.Int())}
			} else if d45.Loc == scm.LocImm && d45.Imm.Int() == 0 {
				r50 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(r50, d43.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d46)
			} else if d43.Loc == scm.LocImm && d43.Imm.Int() == 0 {
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d45.Reg}
				ctx.BindReg(d45.Reg, &d46)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d45.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else if d45.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(scratch, d43.Reg)
				if d45.Imm.Int() >= -2147483648 && d45.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d45.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d46)
			} else {
				r51 := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegReg(r51, d43.Reg)
				ctx.W.EmitAddInt64(r51, d45.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d46)
			}
			if d46.Loc == scm.LocReg && d43.Loc == scm.LocReg && d46.Reg == d43.Reg {
				ctx.TransferReg(d43.Reg)
				d43.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d43)
			ctx.FreeDesc(&d45)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d47 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r52, 32)
				ctx.W.EmitShrRegImm8(r52, 32)
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d47)
			}
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d47.Loc == scm.LocStack || d47.Loc == scm.LocStackPair { ctx.EnsureDesc(&d47) }
			var d48 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d47.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d46.Imm.Int()) == uint64(d47.Imm.Int()))}
			} else if d47.Loc == scm.LocImm {
				r53 := ctx.AllocRegExcept(d46.Reg)
				if d47.Imm.Int() >= -2147483648 && d47.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d46.Reg, int32(d47.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d47.Imm.Int()))
					ctx.W.EmitCmpInt64(d46.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r53, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r53}
				ctx.BindReg(r53, &d48)
			} else if d46.Loc == scm.LocImm {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d47.Reg)
				ctx.W.EmitSetcc(r54, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r54}
				ctx.BindReg(r54, &d48)
			} else {
				r55 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitCmpInt64(d46.Reg, d47.Reg)
				ctx.W.EmitSetcc(r55, scm.CcE)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d48)
			}
			ctx.FreeDesc(&d47)
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d48.Loc == scm.LocImm {
				if d48.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d48.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d48)
			ctx.W.MarkLabel(lbl2)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d49 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r56, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r56, 32)
				ctx.W.EmitShrRegImm8(r56, 32)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r56}
				ctx.BindReg(r56, &d49)
			}
			ctx.FreeDesc(&idxInt)
			if d46.Loc == scm.LocStack || d46.Loc == scm.LocStackPair { ctx.EnsureDesc(&d46) }
			if d49.Loc == scm.LocStack || d49.Loc == scm.LocStackPair { ctx.EnsureDesc(&d49) }
			var d50 scm.JITValueDesc
			if d46.Loc == scm.LocImm && d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d46.Imm.Int()) < uint64(d49.Imm.Int()))}
			} else if d49.Loc == scm.LocImm {
				r57 := ctx.AllocRegExcept(d46.Reg)
				if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d46.Reg, int32(d49.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d49.Imm.Int()))
					ctx.W.EmitCmpInt64(d46.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r57, scm.CcB)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d50)
			} else if d46.Loc == scm.LocImm {
				r58 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d46.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d49.Reg)
				ctx.W.EmitSetcc(r58, scm.CcB)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d50)
			} else {
				r59 := ctx.AllocRegExcept(d46.Reg)
				ctx.W.EmitCmpInt64(d46.Reg, d49.Reg)
				ctx.W.EmitSetcc(r59, scm.CcB)
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r59}
				ctx.BindReg(r59, &d50)
			}
			ctx.FreeDesc(&d46)
			ctx.FreeDesc(&d49)
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d50.Loc == scm.LocImm {
				if d50.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d50.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d50)
			ctx.W.MarkLabel(lbl11)
			var d51 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r60 := ctx.AllocReg()
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r60, fieldAddr)
				ctx.W.EmitMovRegMem64(r61, fieldAddr+8)
				d51 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r60, Reg2: r61}
				ctx.BindReg(r60, &d51)
				ctx.BindReg(r61, &d51)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r62 := ctx.AllocReg()
				r63 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r62, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r63, thisptr.Reg, off+8)
				d51 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r62, Reg2: r63}
				ctx.BindReg(r62, &d51)
				ctx.BindReg(r63, &d51)
			}
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			r64 := ctx.AllocReg()
			if d8.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r64, uint64(d8.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r64, d8.Reg)
				ctx.W.EmitShlRegImm8(r64, 4)
			}
			if d51.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d51.Imm.Int()))
				ctx.W.EmitAddInt64(r64, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r64, d51.Reg)
			}
			r65 := ctx.AllocRegExcept(r64)
			r66 := ctx.AllocRegExcept(r64, r65)
			ctx.W.EmitMovRegMem(r65, r64, 0)
			ctx.W.EmitMovRegMem(r66, r64, 8)
			ctx.FreeReg(r64)
			d52 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r65, Reg2: r66}
			ctx.BindReg(r65, &d52)
			ctx.BindReg(r66, &d52)
			ctx.EmitMovPairToResult(&d52, &result)
			result.Type = d52.Type
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			d53 := d4
			if d53.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d53.Loc == scm.LocStack || d53.Loc == scm.LocStackPair { ctx.EnsureDesc(&d53) }
			d54 := d53
			if d54.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: d54.Type, Imm: scm.NewInt(int64(uint64(d54.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d54.Reg, 32)
				ctx.W.EmitShrRegImm8(d54.Reg, 32)
			}
			ctx.EmitStoreToStack(d54, 0)
			d55 := d8
			if d55.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d55.Loc == scm.LocStack || d55.Loc == scm.LocStackPair { ctx.EnsureDesc(&d55) }
			d56 := d55
			if d56.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: d56.Type, Imm: scm.NewInt(int64(uint64(d56.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d56.Reg, 32)
				ctx.W.EmitShrRegImm8(d56.Reg, 32)
			}
			ctx.EmitStoreToStack(d56, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl14)
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			var d57 scm.JITValueDesc
			if d8.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(scratch, d8.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d57)
			}
			if d57.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: d57.Type, Imm: scm.NewInt(int64(uint64(d57.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d57.Reg, 32)
				ctx.W.EmitShrRegImm8(d57.Reg, 32)
			}
			if d57.Loc == scm.LocReg && d8.Loc == scm.LocReg && d57.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			d58 := d57
			if d58.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d58.Loc == scm.LocStack || d58.Loc == scm.LocStackPair { ctx.EnsureDesc(&d58) }
			d59 := d58
			if d59.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: d59.Type, Imm: scm.NewInt(int64(uint64(d59.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d59.Reg, 32)
				ctx.W.EmitShrRegImm8(d59.Reg, 32)
			}
			ctx.EmitStoreToStack(d59, 0)
			d60 := d5
			if d60.Loc == scm.LocNone { panic("jit: phi source has no location") }
			if d60.Loc == scm.LocStack || d60.Loc == scm.LocStackPair { ctx.EnsureDesc(&d60) }
			d61 := d60
			if d61.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: d61.Type, Imm: scm.NewInt(int64(uint64(d61.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d61.Reg, 32)
				ctx.W.EmitShrRegImm8(d61.Reg, 32)
			}
			ctx.EmitStoreToStack(d61, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
