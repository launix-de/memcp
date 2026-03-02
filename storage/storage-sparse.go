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
			var idxInt scm.JITValueDesc
			if idx.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(idx.Imm.Int())}
			} else if idx.Loc == scm.LocRegPair {
				ctx.FreeReg(idx.Reg)
				idxInt = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: idx.Reg2}
			} else {
				idxInt = idx
			}
			if idxInt.Loc == scm.LocImm {
				idxInt = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(idxInt.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
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
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r2 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r2, thisptr.Reg, off)
				d0 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r2}
			}
			var d1 scm.JITValueDesc
			if d0.Loc == scm.LocImm {
				d1 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d0.Imm.Int()))))}
			} else {
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r3, d0.Reg)
				ctx.W.EmitShlRegImm8(r3, 32)
				ctx.W.EmitShrRegImm8(r3, 32)
				d1 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r3}
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
			d2 := d1
			if d2.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: d2.Type, Imm: scm.NewInt(int64(uint64(d2.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d2.Reg, 32)
				ctx.W.EmitShrRegImm8(d2.Reg, 32)
			}
			ctx.EmitStoreToStack(d2, 8)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r4 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r4, 0)
			ctx.ProtectReg(r4)
			d3 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r4}
			r5 := ctx.AllocRegExcept(r4)
			ctx.EmitLoadFromStack(r5, 8)
			ctx.ProtectReg(r5)
			d4 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r5}
			ctx.UnprotectReg(r4)
			ctx.UnprotectReg(r5)
			var d5 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d3.Imm.Int()) == uint64(d4.Imm.Int()))}
			} else if d4.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d3.Reg)
				if d4.Imm.Int() >= -2147483648 && d4.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d3.Reg, int32(d4.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d4.Imm.Int()))
					ctx.W.EmitCmpInt64(d3.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r6, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
			} else if d3.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d4.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
			} else {
				r8 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitCmpInt64(d3.Reg, d4.Reg)
				ctx.W.EmitSetcc(r8, scm.CcE)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
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
			var d6 scm.JITValueDesc
			if d3.Loc == scm.LocImm && d4.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d3.Imm.Int() + d4.Imm.Int())}
			} else if d4.Loc == scm.LocImm && d4.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r9, d3.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			} else if d3.Loc == scm.LocImm && d3.Imm.Int() == 0 {
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d4.Reg}
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d4.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
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
			} else {
				r10 := ctx.AllocRegExcept(d3.Reg)
				ctx.W.EmitMovRegReg(r10, d3.Reg)
				ctx.W.EmitAddInt64(r10, d4.Reg)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
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
			var d7 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() / 2)}
			} else {
				ctx.W.EmitShrRegImm8(d6.Reg, 1)
				d7 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d6.Reg}
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
			r11 := d7.Loc == scm.LocReg
			r12 := d7.Reg
			if r11 { ctx.ProtectReg(r12) }
			r13 := ctx.AllocReg()
			lbl5 := ctx.W.ReserveLabel()
			var d8 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d7.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d7.Reg)
				ctx.W.EmitShlRegImm8(r14, 32)
				ctx.W.EmitShrRegImm8(r14, 32)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
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
			}
			var d10 scm.JITValueDesc
			if d9.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d9.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d9.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			ctx.FreeDesc(&d9)
			var d11 scm.JITValueDesc
			if d8.Loc == scm.LocImm && d10.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d8.Imm.Int() * d10.Imm.Int())}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d10.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d10.Loc == scm.LocImm {
				if d10.Imm.Int() >= -2147483648 && d10.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(d8.Reg, int32(d10.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d10.Imm.Int()))
				ctx.W.EmitImulInt64(d8.Reg, scm.RegR11)
				}
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
			} else {
				ctx.W.EmitImulInt64(d8.Reg, d10.Reg)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d8.Reg}
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
				r17 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r17, thisptr.Reg, off)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r17}
			}
			var d13 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d13 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() / 64)}
			} else {
				r18 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r18, d11.Reg)
				ctx.W.EmitShrRegImm8(r18, 6)
				d13 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r18}
			}
			if d13.Loc == scm.LocReg && d11.Loc == scm.LocReg && d13.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			r19 := ctx.AllocReg()
			if d13.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r19, uint64(d13.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r19, d13.Reg)
				ctx.W.EmitShlRegImm8(r19, 3)
			}
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(r19, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r19, d12.Reg)
			}
			r20 := ctx.AllocRegExcept(r19)
			ctx.W.EmitMovRegMem(r20, r19, 0)
			ctx.FreeReg(r19)
			d14 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			ctx.FreeDesc(&d13)
			var d15 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				r21 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r21, d11.Reg)
				ctx.W.EmitAndRegImm32(r21, 63)
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r21}
			}
			if d15.Loc == scm.LocReg && d11.Loc == scm.LocReg && d15.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			var d16 scm.JITValueDesc
			if d14.Loc == scm.LocImm && d15.Loc == scm.LocImm {
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d14.Imm.Int()) << uint64(d15.Imm.Int())))}
			} else if d15.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d14.Reg, uint8(d15.Imm.Int()))
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d14.Reg}
			} else {
				{
					shiftSrc := d14.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r22, thisptr.Reg, off)
				d17 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
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
			r23 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r23, 16)
			ctx.ProtectReg(r23)
			d18 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r23}
			ctx.UnprotectReg(r23)
			var d19 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			}
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d19.Imm.Int()))))}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r25, d19.Reg)
				ctx.W.EmitShlRegImm8(r25, 56)
				ctx.W.EmitShrRegImm8(r25, 56)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r25}
			}
			ctx.FreeDesc(&d19)
			d21 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d22 scm.JITValueDesc
			if d21.Loc == scm.LocImm && d20.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d21.Imm.Int() - d20.Imm.Int())}
			} else if d20.Loc == scm.LocImm && d20.Imm.Int() == 0 {
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
			} else if d21.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d21.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d20.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d20.Loc == scm.LocImm {
				if d20.Imm.Int() >= -2147483648 && d20.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d21.Reg, int32(d20.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(d21.Reg, scm.RegR11)
				}
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
			} else {
				ctx.W.EmitSubInt64(d21.Reg, d20.Reg)
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d21.Reg}
			}
			if d22.Loc == scm.LocReg && d21.Loc == scm.LocReg && d22.Reg == d21.Reg {
				ctx.TransferReg(d21.Reg)
				d21.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d20)
			var d23 scm.JITValueDesc
			if d18.Loc == scm.LocImm && d22.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d18.Imm.Int()) >> uint64(d22.Imm.Int())))}
			} else if d22.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d18.Reg, uint8(d22.Imm.Int()))
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d18.Reg}
			} else {
				{
					shiftSrc := d18.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d23.Loc == scm.LocReg && d18.Loc == scm.LocReg && d23.Reg == d18.Reg {
				ctx.TransferReg(d18.Reg)
				d18.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			ctx.FreeDesc(&d22)
			ctx.EmitMovToReg(r13, d23)
			ctx.W.EmitJmp(lbl5)
			ctx.FreeDesc(&d23)
			ctx.W.MarkLabel(lbl6)
			var d24 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				r26 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r26, d11.Reg)
				ctx.W.EmitAndRegImm32(r26, 63)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
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
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r27, thisptr.Reg, off)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r27}
			}
			var d26 scm.JITValueDesc
			if d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d25.Imm.Int()))))}
			} else {
				r28 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r28, d25.Reg)
				ctx.W.EmitShlRegImm8(r28, 56)
				ctx.W.EmitShrRegImm8(r28, 56)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
			}
			ctx.FreeDesc(&d25)
			var d27 scm.JITValueDesc
			if d24.Loc == scm.LocImm && d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d24.Imm.Int() + d26.Imm.Int())}
			} else if d26.Loc == scm.LocImm && d26.Imm.Int() == 0 {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d26.Reg}
			} else if d24.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d26.Loc == scm.LocImm {
				if d26.Imm.Int() >= -2147483648 && d26.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d24.Reg, int32(d26.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d26.Imm.Int()))
				ctx.W.EmitAddInt64(d24.Reg, scm.RegR11)
				}
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else {
				ctx.W.EmitAddInt64(d24.Reg, d26.Reg)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			}
			if d27.Loc == scm.LocReg && d24.Loc == scm.LocReg && d27.Reg == d24.Reg {
				ctx.TransferReg(d24.Reg)
				d24.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d24)
			ctx.FreeDesc(&d26)
			var d28 scm.JITValueDesc
			if d27.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d27.Imm.Int()) > uint64(64))}
			} else {
				r29 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d27.Reg, 64)
				ctx.W.EmitSetcc(r29, scm.CcA)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r29}
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
			var d29 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() / 64)}
			} else {
				r30 := ctx.AllocRegExcept(d11.Reg)
				ctx.W.EmitMovRegReg(r30, d11.Reg)
				ctx.W.EmitShrRegImm8(r30, 6)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
			}
			if d29.Loc == scm.LocReg && d11.Loc == scm.LocReg && d29.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			var d30 scm.JITValueDesc
			if d29.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d29.Imm.Int() + 1)}
			} else {
				ctx.W.EmitAddRegImm32(d29.Reg, int32(1))
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d29.Reg}
			}
			if d30.Loc == scm.LocReg && d29.Loc == scm.LocReg && d30.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			r31 := ctx.AllocReg()
			if d30.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r31, uint64(d30.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r31, d30.Reg)
				ctx.W.EmitShlRegImm8(r31, 3)
			}
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d12.Imm.Int()))
				ctx.W.EmitAddInt64(r31, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r31, d12.Reg)
			}
			r32 := ctx.AllocRegExcept(r31)
			ctx.W.EmitMovRegMem(r32, r31, 0)
			ctx.FreeReg(r31)
			d31 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if d11.Loc == scm.LocImm {
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d11.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d11.Reg, 63)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d11.Reg}
			}
			if d32.Loc == scm.LocReg && d11.Loc == scm.LocReg && d32.Reg == d11.Reg {
				ctx.TransferReg(d11.Reg)
				d11.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d11)
			d33 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d34 scm.JITValueDesc
			if d33.Loc == scm.LocImm && d32.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d33.Imm.Int() - d32.Imm.Int())}
			} else if d32.Loc == scm.LocImm && d32.Imm.Int() == 0 {
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			} else if d33.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d33.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d32.Loc == scm.LocImm {
				if d32.Imm.Int() >= -2147483648 && d32.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(d33.Reg, int32(d32.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d32.Imm.Int()))
				ctx.W.EmitSubInt64(d33.Reg, scm.RegR11)
				}
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			} else {
				ctx.W.EmitSubInt64(d33.Reg, d32.Reg)
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d33.Reg}
			}
			if d34.Loc == scm.LocReg && d33.Loc == scm.LocReg && d34.Reg == d33.Reg {
				ctx.TransferReg(d33.Reg)
				d33.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d32)
			var d35 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d31.Imm.Int()) >> uint64(d34.Imm.Int())))}
			} else if d34.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d31.Reg, uint8(d34.Imm.Int()))
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d31.Reg}
			} else {
				{
					shiftSrc := d31.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d35.Loc == scm.LocReg && d31.Loc == scm.LocReg && d35.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			ctx.FreeDesc(&d34)
			var d36 scm.JITValueDesc
			if d16.Loc == scm.LocImm && d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d16.Imm.Int() | d35.Imm.Int())}
			} else if d16.Loc == scm.LocImm && d16.Imm.Int() == 0 {
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d35.Reg}
			} else if d35.Loc == scm.LocImm && d35.Imm.Int() == 0 {
				r33 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r33, d16.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
			} else if d16.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d16.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d35.Loc == scm.LocImm {
				r34 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r34, d16.Reg)
				if d35.Imm.Int() >= -2147483648 && d35.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r34, int32(d35.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d35.Imm.Int()))
					ctx.W.EmitOrInt64(r34, scratch)
					ctx.FreeReg(scratch)
				}
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r34}
			} else {
				r35 := ctx.AllocRegExcept(d16.Reg)
				ctx.W.EmitMovRegReg(r35, d16.Reg)
				ctx.W.EmitOrInt64(r35, d35.Reg)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
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
			d37 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			if r11 { ctx.UnprotectReg(r12) }
			var d38 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r36 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r36, thisptr.Reg, off)
				d38 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r36}
			}
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d38.Imm.Int()))))}
			} else {
				r37 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r37, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
			}
			ctx.FreeDesc(&d38)
			var d40 scm.JITValueDesc
			if d37.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d37.Imm.Int() + d39.Imm.Int())}
			} else if d39.Loc == scm.LocImm && d39.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d39.Reg}
			} else if d37.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d37.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d39.Loc == scm.LocImm {
				if d39.Imm.Int() >= -2147483648 && d39.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(d37.Reg, int32(d39.Imm.Int()))
				} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d39.Imm.Int()))
				ctx.W.EmitAddInt64(d37.Reg, scm.RegR11)
				}
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
			} else {
				ctx.W.EmitAddInt64(d37.Reg, d39.Reg)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
			}
			if d40.Loc == scm.LocReg && d37.Loc == scm.LocReg && d40.Reg == d37.Reg {
				ctx.TransferReg(d37.Reg)
				d37.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.FreeDesc(&d39)
			var d41 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r38 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r38, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r38, 32)
				ctx.W.EmitShrRegImm8(r38, 32)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r38}
			}
			var d42 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d40.Imm.Int()) == uint64(d41.Imm.Int()))}
			} else if d41.Loc == scm.LocImm {
				r39 := ctx.AllocRegExcept(d40.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d41.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r39, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r39}
			} else if d40.Loc == scm.LocImm {
				r40 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d41.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r40, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r40}
			} else {
				r41 := ctx.AllocRegExcept(d40.Reg)
				ctx.W.EmitCmpInt64(d40.Reg, d41.Reg)
				ctx.W.EmitSetcc(r41, scm.CcE)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r41}
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
			var d43 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r42 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r42, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r42, 32)
				ctx.W.EmitShrRegImm8(r42, 32)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
			}
			ctx.FreeDesc(&idxInt)
			var d44 scm.JITValueDesc
			if d40.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d40.Imm.Int()) < uint64(d43.Imm.Int()))}
			} else if d43.Loc == scm.LocImm {
				r43 := ctx.AllocReg()
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d40.Reg, int32(d43.Imm.Int()))
				} else {
					scratch := ctx.AllocReg()
					ctx.W.EmitMovRegImm64(scratch, uint64(d43.Imm.Int()))
					ctx.W.EmitCmpInt64(d40.Reg, scratch)
					ctx.FreeReg(scratch)
				}
				ctx.W.EmitSetcc(r43, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r43}
			} else if d40.Loc == scm.LocImm {
				r44 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d40.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d43.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r44, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r44}
			} else {
				r45 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d40.Reg, d43.Reg)
				ctx.W.EmitSetcc(r45, scm.CcB)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r45}
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
				r46 := ctx.AllocReg()
				r47 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r46, fieldAddr)
				ctx.W.EmitMovRegMem64(r47, fieldAddr+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r46, Reg2: r47}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r48 := ctx.AllocReg()
				r49 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r48, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r49, thisptr.Reg, off+8)
				d45 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r48, Reg2: r49}
			}
			r50 := ctx.AllocReg()
			if d7.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r50, uint64(d7.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r50, d7.Reg)
				ctx.W.EmitShlRegImm8(r50, 4)
			}
			if d45.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d45.Imm.Int()))
				ctx.W.EmitAddInt64(r50, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r50, d45.Reg)
			}
			r51 := ctx.AllocRegExcept(r50)
			r52 := ctx.AllocRegExcept(r50, r51)
			ctx.W.EmitMovRegMem(r51, r50, 0)
			ctx.W.EmitMovRegMem(r52, r50, 8)
			ctx.FreeReg(r50)
			d46 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r51, Reg2: r52}
			ctx.EmitMovPairToResult(&d46, &result)
			result.Type = d46.Type
			ctx.W.EmitJmp(lbl0)
			ctx.FreeDesc(&d46)
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
			var d49 scm.JITValueDesc
			if d7.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d7.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d7.Reg)
				ctx.W.EmitMovRegReg(scratch, d7.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
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
