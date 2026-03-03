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
			if d2.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: d2.Type, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d2.Reg, 32)
				ctx.W.EmitShrRegImm8(d2.Reg, 32)
			}
			ctx.EmitStoreToStack(d2, 8)
			ctx.W.MarkLabel(lbl1)
			r4 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r4, 0)
			ctx.ProtectReg(r4)
			d3 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r4}
			ctx.BindReg(r4, &d3)
			r5 := ctx.AllocRegExcept(r4)
			ctx.EmitLoadFromStack(r5, 8)
			ctx.ProtectReg(r5)
			d4 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r5}
			ctx.BindReg(r5, &d4)
			ctx.UnprotectReg(r4)
			ctx.UnprotectReg(r5)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d3.Imm.Int()) == uint64(d4.Imm.Int()))}
			} else if d4.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d3.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitCmpInt64(d3.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r6, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d5)
			} else if d3.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d3.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d4.Reg)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d5)
			} else {
				r8 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitCmpInt64(d3.Reg, d4.Reg)
				ctx.W.EmitSetcc(r8, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d5)
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d5.Loc == scm.LocImm {
				if d5.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d5.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d5)
			ctx.W.MarkLabel(lbl3)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			if d4.Loc == scm.LocStack || d4.Loc == scm.LocStackPair { ctx.EnsureDesc(&d4) }
			var d6 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() + d4.Imm.Int())}
			} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d6)
			} else if d3.Loc == scm.LocImm && d3.Imm.Int() == 0 {
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d4.Reg}
				ctx.BindReg(d4.Reg, &d6)
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d4.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d4.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else if d4.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(scratch, d3.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d4.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d4.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d6)
			} else {
				r10 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r10, d3.Reg)
				ctx.W.EmitAddInt64(r10, d4.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d6)
			}
			if d6.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: d6.Type, Imm: scm.NewInt(int64(uint64(d6.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d6.Reg, 32)
				ctx.W.EmitShrRegImm8(d6.Reg, 32)
			}
			if d6.Loc == scm.LocReg && d3.Loc == scm.LocReg && d6.Reg == d3.Reg {
				ctx.TransferReg(d3.Reg)
				d3.Loc = scm.LocNone
			}
			if d6.Loc == scm.LocStack || d6.Loc == scm.LocStackPair { ctx.EnsureDesc(&d6) }
			var d7 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r11, d6.Reg)
				ctx.W.EmitShrRegImm8(r11, 1)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d7)
			}
			if d7.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: d7.Type, Imm: scm.NewInt(int64(uint64(d7.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d7.Reg, 32)
				ctx.W.EmitShrRegImm8(d7.Reg, 32)
			}
			if d7.Loc == scm.LocReg && d6.Loc == scm.LocReg && d7.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d6)
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			r12 := d7.Loc == scm.LocReg
			r13 := d7.Reg
			if r12 { ctx.ProtectReg(r13) }
			lbl5 := ctx.W.ReserveLabel()
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			var d8 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d7.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d7.Reg)
				ctx.W.EmitShlRegImm8(r14, 32)
				ctx.W.EmitShrRegImm8(r14, 32)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d8)
			}
			var d9 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d9 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d9 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d9)
			}
			if d9.Loc == scm.LocStack || d9.Loc == scm.LocStackPair { ctx.EnsureDesc(&d9) }
			var d10 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d9.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d9.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d10)
			}
			ctx.FreeDesc(&d9)
			if d8.Loc == scm.LocStack || d8.Loc == scm.LocStackPair { ctx.EnsureDesc(&d8) }
			if d10.Loc == scm.LocStack || d10.Loc == scm.LocStackPair { ctx.EnsureDesc(&d10) }
			var d11 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() * d10.Imm.Int())}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d10.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d11)
			} else if d10.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(scratch, d8.Reg)
				if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d10.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d11)
			} else {
				r17 := ctx.AllocRegExcept(d8.Reg)
				ctx.W.EmitMovRegReg(r17, d8.Reg)
				ctx.W.EmitImulInt64(r17, d10.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d11)
			}
			if d11.Loc == scm.LocReg && d8.Loc == scm.LocReg && d11.Reg == d8.Reg {
				ctx.TransferReg(d8.Reg)
				d8.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			ctx.FreeDesc(&d10)
			var d12 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				r18 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r18, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d12)
			}
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d13 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() / 64)}
			} else {
				r19 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r19, d11.Reg)
				ctx.W.EmitShrRegImm8(r19, 6)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d13)
			}
			if d13.Loc == scm.LocReg && d11.Loc == scm.LocReg && d13.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			if d13.Loc == scm.LocStack || d13.Loc == scm.LocStackPair { ctx.EnsureDesc(&d13) }
			r20 := ctx.AllocReg()
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r20, uint64(d13.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r20, d13.Reg)
				ctx.W.EmitShlRegImm8(r20, 3)
			}
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(r20, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r20, d12.Reg)
			}
			r21 := ctx.AllocRegExcept(r20)
			ctx.W.EmitMovRegMem(r21, r20, 0)
			ctx.FreeReg(r20)
			d14 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d14)
			ctx.FreeDesc(&d13)
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d15 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r22, d11.Reg)
				ctx.W.EmitAndRegImm32(r22, 63)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d15)
			}
			if d15.Loc == scm.LocReg && d11.Loc == scm.LocReg && d15.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			if d14.Loc == scm.LocStack || d14.Loc == scm.LocStackPair { ctx.EnsureDesc(&d14) }
			if d15.Loc == scm.LocStack || d15.Loc == scm.LocStackPair { ctx.EnsureDesc(&d15) }
			var d16 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) << uint64(d15.Imm.Int())))}
			} else if d15.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d14.Reg)
				ctx.W.EmitMovRegReg(r23, d14.Reg)
				ctx.W.EmitShlRegImm8(r23, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d16)
			} else {
				{
					shiftSrc := d14.Reg
					r24 := ctx.AllocRegExcept(d14.Reg)
					ctx.W.EmitMovRegReg(r24, d14.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d15.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d15.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d15.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d16)
				}
			}
			if d16.Loc == scm.LocReg && d14.Loc == scm.LocReg && d16.Reg == d14.Reg {
				ctx.TransferReg(d14.Reg)
				d14.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d14)
			ctx.FreeDesc(&d15)
			var d17 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d17 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d17)
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d17.Loc == scm.LocImm {
				if d17.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
			ctx.EmitStoreToStack(d16, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d17.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
			ctx.EmitStoreToStack(d16, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d17)
			ctx.W.MarkLabel(lbl7)
			r26 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r26, 16)
			ctx.ProtectReg(r26)
			d18 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r26}
			ctx.BindReg(r26, &d18)
			ctx.UnprotectReg(r26)
			var d19 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r27, thisptr.Reg, off)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
				ctx.BindReg(r27, &d19)
			}
			if d19.Loc == scm.LocStack || d19.Loc == scm.LocStackPair { ctx.EnsureDesc(&d19) }
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d19.Imm.Int()))))}
			} else {
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r28, d19.Reg)
				ctx.W.EmitShlRegImm8(r28, 56)
				ctx.W.EmitShrRegImm8(r28, 56)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d20)
			}
			ctx.FreeDesc(&d19)
			d21 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d20.Loc == scm.LocStack || d20.Loc == scm.LocStackPair { ctx.EnsureDesc(&d20) }
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() - d20.Imm.Int())}
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				r29 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r29, d21.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d22)
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d20.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(scratch, d21.Reg)
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d20.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d20.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d22)
			} else {
				r30 := ctx.AllocRegExcept(d21.Reg)
				ctx.W.EmitMovRegReg(r30, d21.Reg)
				ctx.W.EmitSubInt64(r30, d20.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d22)
			}
			if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			if d18.Loc == scm.LocStack || d18.Loc == scm.LocStackPair { ctx.EnsureDesc(&d18) }
			if d22.Loc == scm.LocStack || d22.Loc == scm.LocStackPair { ctx.EnsureDesc(&d22) }
			var d23 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d18.Imm.Int()) >> uint64(d22.Imm.Int())))}
			} else if d22.Loc == scm.LocImm {
				r31 := ctx.AllocRegExcept(d18.Reg)
				ctx.W.EmitMovRegReg(r31, d18.Reg)
				ctx.W.EmitShrRegImm8(r31, uint8(d22.Imm.Int()))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r31}
				ctx.BindReg(r31, &d23)
			} else {
				{
					shiftSrc := d18.Reg
					r32 := ctx.AllocRegExcept(d18.Reg)
					ctx.W.EmitMovRegReg(r32, d18.Reg)
					shiftSrc = r32
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d22.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d22.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d22.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d23)
				}
			}
			if d23.Loc == scm.LocReg && d18.Loc == scm.LocReg && d23.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.FreeDesc(&d22)
			r33 := ctx.AllocReg()
			if d23.Loc == scm.LocStack || d23.Loc == scm.LocStackPair { ctx.EnsureDesc(&d23) }
			ctx.EmitMovToReg(r33, d23)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl6)
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d24 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				r34 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r34, d11.Reg)
				ctx.W.EmitAndRegImm32(r34, 63)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
				ctx.BindReg(r34, &d24)
			}
			if d24.Loc == scm.LocReg && d11.Loc == scm.LocReg && d24.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			var d25 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r35, thisptr.Reg, off)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r35}
				ctx.BindReg(r35, &d25)
			}
			if d25.Loc == scm.LocStack || d25.Loc == scm.LocStackPair { ctx.EnsureDesc(&d25) }
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d25.Imm.Int()))))}
			} else {
				r36 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r36, d25.Reg)
				ctx.W.EmitShlRegImm8(r36, 56)
				ctx.W.EmitShrRegImm8(r36, 56)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d26)
			}
			ctx.FreeDesc(&d25)
			if d24.Loc == scm.LocStack || d24.Loc == scm.LocStackPair { ctx.EnsureDesc(&d24) }
			if d26.Loc == scm.LocStack || d26.Loc == scm.LocStackPair { ctx.EnsureDesc(&d26) }
			var d27 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() + d26.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				r37 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r37, d24.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d27)
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
				ctx.BindReg(d26.Reg, &d27)
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else if d26.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(scratch, d24.Reg)
				if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d26.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d27)
			} else {
				r38 := ctx.AllocRegExcept(d24.Reg)
				ctx.W.EmitMovRegReg(r38, d24.Reg)
				ctx.W.EmitAddInt64(r38, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
				ctx.BindReg(r38, &d27)
			}
			if d27.Loc == scm.LocReg && d24.Loc == scm.LocReg && d27.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d26)
			if d27.Loc == scm.LocStack || d27.Loc == scm.LocStackPair { ctx.EnsureDesc(&d27) }
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d27.Imm.Int()) > uint64(64))}
			} else {
				r39 := ctx.AllocRegExcept(d27.Reg)
				ctx.W.EmitCmpRegImm32(d27.Reg, 64)
				ctx.W.EmitSetcc(r39, scm.CcA)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r39}
				ctx.BindReg(r39, &d28)
			}
			ctx.FreeDesc(&d27)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d28.Loc == scm.LocImm {
				if d28.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
			ctx.EmitStoreToStack(d16, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d28.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
			ctx.EmitStoreToStack(d16, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d28)
			ctx.W.MarkLabel(lbl9)
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d29 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() / 64)}
			} else {
				r40 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r40, d11.Reg)
				ctx.W.EmitShrRegImm8(r40, 6)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r40}
				ctx.BindReg(r40, &d29)
			}
			if d29.Loc == scm.LocReg && d11.Loc == scm.LocReg && d29.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			if d29.Loc == scm.LocStack || d29.Loc == scm.LocStackPair { ctx.EnsureDesc(&d29) }
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
			if d30.Loc == scm.LocStack || d30.Loc == scm.LocStackPair { ctx.EnsureDesc(&d30) }
			r41 := ctx.AllocReg()
			if d30.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r41, uint64(d30.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r41, d30.Reg)
				ctx.W.EmitShlRegImm8(r41, 3)
			}
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(r41, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r41, d12.Reg)
			}
			r42 := ctx.AllocRegExcept(r41)
			ctx.W.EmitMovRegMem(r42, r41, 0)
			ctx.FreeReg(r41)
			d31 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r42}
			ctx.BindReg(r42, &d31)
			ctx.FreeDesc(&d30)
			if d11.Loc == scm.LocStack || d11.Loc == scm.LocStackPair { ctx.EnsureDesc(&d11) }
			var d32 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				r43 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r43, d11.Reg)
				ctx.W.EmitAndRegImm32(r43, 63)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d32)
			}
			if d32.Loc == scm.LocReg && d11.Loc == scm.LocReg && d32.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			d33 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			if d32.Loc == scm.LocStack || d32.Loc == scm.LocStackPair { ctx.EnsureDesc(&d32) }
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() - d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				r44 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r44, d33.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d34)
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
				r45 := ctx.AllocRegExcept(d33.Reg)
				ctx.W.EmitMovRegReg(r45, d33.Reg)
				ctx.W.EmitSubInt64(r45, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d34)
			}
			if d34.Loc == scm.LocReg && d33.Loc == scm.LocReg && d34.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			if d31.Loc == scm.LocStack || d31.Loc == scm.LocStackPair { ctx.EnsureDesc(&d31) }
			if d34.Loc == scm.LocStack || d34.Loc == scm.LocStackPair { ctx.EnsureDesc(&d34) }
			var d35 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d31.Imm.Int()) >> uint64(d34.Imm.Int())))}
			} else if d34.Loc == scm.LocImm {
				r46 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r46, d31.Reg)
				ctx.W.EmitShrRegImm8(r46, uint8(d34.Imm.Int()))
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r46}
				ctx.BindReg(r46, &d35)
			} else {
				{
					shiftSrc := d31.Reg
					r47 := ctx.AllocRegExcept(d31.Reg)
					ctx.W.EmitMovRegReg(r47, d31.Reg)
					shiftSrc = r47
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
			if d16.Loc == scm.LocStack || d16.Loc == scm.LocStackPair { ctx.EnsureDesc(&d16) }
			if d35.Loc == scm.LocStack || d35.Loc == scm.LocStackPair { ctx.EnsureDesc(&d35) }
			var d36 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() | d35.Imm.Int())}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d35.Reg}
				ctx.BindReg(d35.Reg, &d36)
			} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
				r48 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r48, d16.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d36)
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d35.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d36)
			} else if d35.Loc == scm.LocImm {
				r49 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r49, d16.Reg)
				if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r49, int32(d35.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d35.Imm.Int()))
					ctx.W.EmitOrInt64(r49, scm.RegR11)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d36)
			} else {
				r50 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r50, d16.Reg)
				ctx.W.EmitOrInt64(r50, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r50}
				ctx.BindReg(r50, &d36)
			}
			if d36.Loc == scm.LocReg && d16.Loc == scm.LocReg && d36.Reg == d16.Reg {
				ctx.TransferReg(d16.Reg)
				d16.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EmitStoreToStack(d36, 16)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r33}
			ctx.BindReg(r33, &d37)
			ctx.BindReg(r33, &d37)
			if r12 { ctx.UnprotectReg(r13) }
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r51, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r51}
				ctx.BindReg(r51, &d38)
			}
			if d38.Loc == scm.LocStack || d38.Loc == scm.LocStackPair { ctx.EnsureDesc(&d38) }
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d38.Imm.Int()))))}
			} else {
				r52 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r52, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d39)
			}
			ctx.FreeDesc(&d38)
			if d37.Loc == scm.LocStack || d37.Loc == scm.LocStackPair { ctx.EnsureDesc(&d37) }
			if d39.Loc == scm.LocStack || d39.Loc == scm.LocStackPair { ctx.EnsureDesc(&d39) }
			var d40 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				r53 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r53, d37.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d40)
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
				ctx.BindReg(d39.Reg, &d40)
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(scratch, d37.Reg)
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d39.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d40)
			} else {
				r54 := ctx.AllocRegExcept(d37.Reg)
				ctx.W.EmitMovRegReg(r54, d37.Reg)
				ctx.W.EmitAddInt64(r54, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d40)
			}
			if d40.Loc == scm.LocReg && d37.Loc == scm.LocReg && d40.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d39)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d41 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r55 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r55, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r55, 32)
				ctx.W.EmitShrRegImm8(r55, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r55}
				ctx.BindReg(r55, &d41)
			}
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			if d41.Loc == scm.LocStack || d41.Loc == scm.LocStackPair { ctx.EnsureDesc(&d41) }
			var d42 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d40.Imm.Int()) == uint64(d41.Imm.Int()))}
			} else if d41.Loc == scm.LocImm {
				r56 := ctx.AllocRegExcept(d40.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d41.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r56, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d42)
			} else if d40.Loc == scm.LocImm {
				r57 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d41.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d42)
			} else {
				r58 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitCmpInt64(d40.Reg, d41.Reg)
				ctx.W.EmitSetcc(r58, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r58}
				ctx.BindReg(r58, &d42)
			}
			ctx.FreeDesc(&d41)
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
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			var d43 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r59, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r59, 32)
				ctx.W.EmitShrRegImm8(r59, 32)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r59}
				ctx.BindReg(r59, &d43)
			}
			ctx.FreeDesc(&idxInt)
			if d40.Loc == scm.LocStack || d40.Loc == scm.LocStackPair { ctx.EnsureDesc(&d40) }
			if d43.Loc == scm.LocStack || d43.Loc == scm.LocStackPair { ctx.EnsureDesc(&d43) }
			var d44 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d40.Imm.Int()) < uint64(d43.Imm.Int()))}
			} else if d43.Loc == scm.LocImm {
				r60 := ctx.AllocRegExcept(d40.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r60, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r60}
				ctx.BindReg(r60, &d44)
			} else if d40.Loc == scm.LocImm {
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d43.Reg)
				ctx.W.EmitSetcc(r61, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r61}
				ctx.BindReg(r61, &d44)
			} else {
				r62 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitCmpInt64(d40.Reg, d43.Reg)
				ctx.W.EmitSetcc(r62, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r62}
				ctx.BindReg(r62, &d44)
			}
			ctx.FreeDesc(&d40)
			ctx.FreeDesc(&d43)
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d44.Loc == scm.LocImm {
				if d44.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d44.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d44)
			ctx.W.MarkLabel(lbl11)
			var d45 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r63 := ctx.AllocReg()
				r64 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r63, fieldAddr)
				ctx.W.EmitMovRegMem64(r64, fieldAddr+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r63, Reg2: r64}
				ctx.BindReg(r63, &d45)
				ctx.BindReg(r64, &d45)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r65 := ctx.AllocReg()
				r66 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r65, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r66, thisptr.Reg, off+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r65, Reg2: r66}
				ctx.BindReg(r65, &d45)
				ctx.BindReg(r66, &d45)
			}
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			r67 := ctx.AllocReg()
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r67, uint64(d7.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r67, d7.Reg)
				ctx.W.EmitShlRegImm8(r67, 4)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r67, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r67, d45.Reg)
			}
			r68 := ctx.AllocRegExcept(r67)
			r69 := ctx.AllocRegExcept(r67, r68)
			ctx.W.EmitMovRegMem(r68, r67, 0)
			ctx.W.EmitMovRegMem(r69, r67, 8)
			ctx.FreeReg(r67)
			d46 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r68, Reg2: r69}
			ctx.BindReg(r68, &d46)
			ctx.BindReg(r69, &d46)
			ctx.EmitMovPairToResult(&d46, &result)
			result.Type = d46.Type
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl15)
			d47 := d3
			if d47.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: d47.Type, Imm: scm.NewInt(int64(uint64(d47.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d47.Reg, 32)
				ctx.W.EmitShrRegImm8(d47.Reg, 32)
			}
			ctx.EmitStoreToStack(d47, 0)
			d48 := d7
			if d48.Loc == scm.LocImm {
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: d48.Type, Imm: scm.NewInt(int64(uint64(d48.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d48.Reg, 32)
				ctx.W.EmitShrRegImm8(d48.Reg, 32)
			}
			ctx.EmitStoreToStack(d48, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl14)
			if d7.Loc == scm.LocStack || d7.Loc == scm.LocStackPair { ctx.EnsureDesc(&d7) }
			var d49 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(scratch, d7.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d49)
			}
			if d49.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: d49.Type, Imm: scm.NewInt(int64(uint64(d49.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d49.Reg, 32)
				ctx.W.EmitShrRegImm8(d49.Reg, 32)
			}
			if d49.Loc == scm.LocReg && d7.Loc == scm.LocReg && d49.Reg == d7.Reg {
				ctx.TransferReg(d7.Reg)
				d7.Loc = scm.LocNone
			}
			d50 := d49
			if d50.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: d50.Type, Imm: scm.NewInt(int64(uint64(d50.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d50.Reg, 32)
				ctx.W.EmitShrRegImm8(d50.Reg, 32)
			}
			ctx.EmitStoreToStack(d50, 0)
			d51 := d4
			if d51.Loc == scm.LocImm {
				d51 = scm.JITValueDesc{Loc: scm.LocImm, Type: d51.Type, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d51.Reg, 32)
				ctx.W.EmitShrRegImm8(d51.Reg, 32)
			}
			ctx.EmitStoreToStack(d51, 8)
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
