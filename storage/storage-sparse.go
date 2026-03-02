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
import "unsafe"
import "encoding/json"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

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
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
			ctx.EmitStoreToStack(d0, 8)
			lbl1 := ctx.W.ReserveLabel()
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl1)
			r3 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r3, 0)
			ctx.ProtectReg(r3)
			d2 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r3}
			r4 := ctx.AllocRegExcept(r3)
			ctx.EmitLoadFromStack(r4, 8)
			ctx.ProtectReg(r4)
			d3 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r4}
			ctx.UnprotectReg(r3)
			ctx.UnprotectReg(r4)
			var d4 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d3.Loc == scm.LocImm {
				d4 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d2.Imm.Int() == d3.Imm.Int())}
			} else if d3.Loc == scm.LocImm {
				r5 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpRegImm32(d2.Reg, int32(d3.Imm.Int()))
				ctx.W.EmitSetcc(r5, scm.CcE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r5}
			} else if d2.Loc == scm.LocImm {
				r6 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, d3.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r6, scm.CcE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
			} else {
				r7 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d3.Reg)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d4 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
			}
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			lbl4 := ctx.W.ReserveLabel()
			if d4.Loc == scm.LocImm {
				if d4.Imm.Bool() {
					ctx.W.EmitJmp(lbl2)
				} else {
					ctx.W.EmitJmp(lbl3)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d4.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl4)
				ctx.W.EmitJmp(lbl3)
				ctx.W.MarkLabel(lbl4)
				ctx.W.EmitJmp(lbl2)
			}
			ctx.FreeDesc(&d4)
			ctx.W.MarkLabel(lbl3)
			var d5 scm.JITValueDesc
			if d2.Loc == scm.LocImm && d3.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d2.Imm.Int() + d3.Imm.Int())}
			} else if d3.Loc == scm.LocImm && d3.Imm.Int() == 0 {
				r8 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(r8, d2.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r8}
			} else if d2.Loc == scm.LocImm && d2.Imm.Int() == 0 {
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d3.Reg}
			} else if d2.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d2.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d3.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d3.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d3.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d2.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r9 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitMovRegReg(r9, d2.Reg)
				ctx.W.EmitAddInt64(r9, d3.Reg)
				d5 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
			}
			if d5.Loc == scm.LocReg && d2.Loc == scm.LocReg && d5.Reg == d2.Reg {
				ctx.TransferReg(d2.Reg)
				d2.Loc = scm.LocNone
			}
			var d6 scm.JITValueDesc
			if d5.Loc == scm.LocImm {
				d6 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d5.Imm.Int() / 2)}
			} else {
				ctx.W.EmitShrRegImm8(d5.Reg, 1)
				d6 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d5.Reg}
			}
			if d6.Loc == scm.LocReg && d5.Loc == scm.LocReg && d6.Reg == d5.Reg {
				ctx.TransferReg(d5.Reg)
				d5.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d5)
			r10 := d6.Loc == scm.LocReg
			r11 := d6.Reg
			if r10 { ctx.ProtectReg(r11) }
			r12 := ctx.AllocReg()
			lbl5 := ctx.W.ReserveLabel()
			var d8 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d8 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r13 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r13, thisptr.Reg, off)
				d8 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r13}
			}
			var d10 scm.JITValueDesc
			if d6.Loc == scm.LocImm && d8.Loc == scm.LocImm {
				d10 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() * d8.Imm.Int())}
			} else if d6.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d6.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d8.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d8.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d8.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d6.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else {
				r14 := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegReg(r14, d6.Reg)
				ctx.W.EmitImulInt64(r14, d8.Reg)
				d10 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
			}
			if d10.Loc == scm.LocReg && d6.Loc == scm.LocReg && d10.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d8)
			var d11 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(dataPtr)), StackOff: int32(sliceLen)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r15, thisptr.Reg, off)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
			}
			var d12 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d12 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() / 64)}
			} else {
				r16 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r16, d10.Reg)
				ctx.W.EmitShrRegImm8(r16, 6)
				d12 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
			}
			if d12.Loc == scm.LocReg && d10.Loc == scm.LocReg && d12.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			r17 := ctx.AllocReg()
			if d12.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r17, uint64(d12.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r17, d12.Reg)
				ctx.W.EmitShlRegImm8(r17, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r17, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r17, d11.Reg)
			}
			r18 := ctx.AllocRegExcept(r17)
			ctx.W.EmitMovRegMem(r18, r17, 0)
			ctx.FreeReg(r17)
			d13 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
			ctx.FreeDesc(&d12)
			var d14 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d14 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				r19 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r19, d10.Reg)
				ctx.W.EmitAndRegImm32(r19, 63)
				d14 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
			}
			if d14.Loc == scm.LocReg && d10.Loc == scm.LocReg && d14.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			var d15 scm.JITValueDesc
			if d13.Loc == scm.LocImm && d14.Loc == scm.LocImm {
				d15 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d13.Imm.Int()) << uint64(d14.Imm.Int())))}
			} else if d14.Loc == scm.LocImm {
				ctx.W.EmitShlRegImm8(d13.Reg, uint8(d14.Imm.Int()))
				d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d13.Reg}
			} else {
				{
					shiftSrc := d13.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d14.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d14.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d14.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d15 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
				}
			}
			if d15.Loc == scm.LocReg && d13.Loc == scm.LocReg && d15.Reg == d13.Reg {
				ctx.TransferReg(d13.Reg)
				d13.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d13)
			ctx.FreeDesc(&d14)
			var d16 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d16 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r20 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r20, thisptr.Reg, off)
				d16 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r20}
			}
			lbl6 := ctx.W.ReserveLabel()
			lbl7 := ctx.W.ReserveLabel()
			lbl8 := ctx.W.ReserveLabel()
			if d16.Loc == scm.LocImm {
				if d16.Imm.Bool() {
					ctx.W.EmitJmp(lbl6)
				} else {
					ctx.EmitStoreToStack(d15, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d16.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl8)
				ctx.EmitStoreToStack(d15, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl8)
				ctx.W.EmitJmp(lbl6)
			}
			ctx.FreeDesc(&d16)
			ctx.W.MarkLabel(lbl7)
			r21 := ctx.AllocReg()
			ctx.EmitLoadFromStack(r21, 16)
			ctx.ProtectReg(r21)
			d17 := scm.JITValueDesc{Loc: scm.LocReg, Type: scm.JITTypeUnknown, Reg: r21}
			ctx.UnprotectReg(r21)
			var d18 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d18 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r22 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r22, thisptr.Reg, off)
				d18 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r22}
			}
			d20 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d21 scm.JITValueDesc
			if d20.Loc == scm.LocImm && d18.Loc == scm.LocImm {
				d21 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() - d18.Imm.Int())}
			} else if d18.Loc == scm.LocImm && d18.Imm.Int() == 0 {
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			} else if d20.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d20.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d18.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d18.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d18.Imm.Int()))
				ctx.W.EmitSubInt64(d20.Reg, scm.RegR11)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			} else {
				ctx.W.EmitSubInt64(d20.Reg, d18.Reg)
				d21 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d20.Reg}
			}
			if d21.Loc == scm.LocReg && d20.Loc == scm.LocReg && d21.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d18)
			var d22 scm.JITValueDesc
			if d17.Loc == scm.LocImm && d21.Loc == scm.LocImm {
				d22 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d17.Imm.Int()) >> uint64(d21.Imm.Int())))}
			} else if d21.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d17.Reg, uint8(d21.Imm.Int()))
				d22 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d17.Reg}
			} else {
				{
					shiftSrc := d17.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d22.Loc == scm.LocReg && d17.Loc == scm.LocReg && d22.Reg == d17.Reg {
				ctx.TransferReg(d17.Reg)
				d17.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d17)
			ctx.FreeDesc(&d21)
			ctx.EmitMovToReg(r12, d22)
			ctx.W.EmitJmp(lbl5)
			ctx.FreeDesc(&d22)
			ctx.W.MarkLabel(lbl6)
			var d23 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				r23 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r23, d10.Reg)
				ctx.W.EmitAndRegImm32(r23, 63)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
			}
			if d23.Loc == scm.LocReg && d10.Loc == scm.LocReg && d23.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r24 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r24, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r24}
			}
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d24.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() + d24.Imm.Int())}
			} else if d24.Loc == scm.LocImm && d24.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else if d23.Loc == scm.LocImm && d23.Imm.Int() == 0 {
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d24.Reg}
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d24.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d24.Imm.Int()))
				ctx.W.EmitAddInt64(d23.Reg, scm.RegR11)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			} else {
				ctx.W.EmitAddInt64(d23.Reg, d24.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d23.Reg}
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d24)
			var d27 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d27 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d26.Imm.Int() > 64)}
			} else {
				r25 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d26.Reg, 64)
				ctx.W.EmitSetcc(r25, scm.CcG)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r25}
			}
			ctx.FreeDesc(&d26)
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			if d27.Loc == scm.LocImm {
				if d27.Imm.Bool() {
					ctx.W.EmitJmp(lbl9)
				} else {
					ctx.EmitStoreToStack(d15, 16)
					ctx.W.EmitJmp(lbl7)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d27.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl10)
				ctx.EmitStoreToStack(d15, 16)
				ctx.W.EmitJmp(lbl7)
				ctx.W.MarkLabel(lbl10)
				ctx.W.EmitJmp(lbl9)
			}
			ctx.FreeDesc(&d27)
			ctx.W.MarkLabel(lbl9)
			var d28 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() / 64)}
			} else {
				r26 := ctx.AllocRegExcept(d10.Reg)
				ctx.W.EmitMovRegReg(r26, d10.Reg)
				ctx.W.EmitShrRegImm8(r26, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r26}
			}
			if d28.Loc == scm.LocReg && d10.Loc == scm.LocReg && d28.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			var d29 scm.JITValueDesc
			if d28.Loc == scm.LocImm {
				d29 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d28.Imm.Int() + 1)}
			} else {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(1))
				ctx.W.EmitAddInt64(d28.Reg, scm.RegR11)
				d29 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d28.Reg}
			}
			if d29.Loc == scm.LocReg && d28.Loc == scm.LocReg && d29.Reg == d28.Reg {
				ctx.TransferReg(d28.Reg)
				d28.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d28)
			r27 := ctx.AllocReg()
			if d29.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r27, uint64(d29.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r27, d29.Reg)
				ctx.W.EmitShlRegImm8(r27, 3)
			}
			if d11.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d11.Imm.Int()))
				ctx.W.EmitAddInt64(r27, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r27, d11.Reg)
			}
			r28 := ctx.AllocRegExcept(r27)
			ctx.W.EmitMovRegMem(r28, r27, 0)
			ctx.FreeReg(r27)
			d30 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r28}
			ctx.FreeDesc(&d29)
			var d31 scm.JITValueDesc
			if d10.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d10.Imm.Int() % 64)}
			} else {
				ctx.W.EmitAndRegImm32(d10.Reg, 63)
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d10.Reg}
			}
			if d31.Loc == scm.LocReg && d10.Loc == scm.LocReg && d31.Reg == d10.Reg {
				ctx.TransferReg(d10.Reg)
				d10.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d10)
			d32 := scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			var d33 scm.JITValueDesc
			if d32.Loc == scm.LocImm && d31.Loc == scm.LocImm {
				d33 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d32.Imm.Int() - d31.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else if d32.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d32.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d31.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d31.Imm.Int()))
				ctx.W.EmitSubInt64(d32.Reg, scm.RegR11)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			} else {
				ctx.W.EmitSubInt64(d32.Reg, d31.Reg)
				d33 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d32.Reg}
			}
			if d33.Loc == scm.LocReg && d32.Loc == scm.LocReg && d33.Reg == d32.Reg {
				ctx.TransferReg(d32.Reg)
				d32.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d31)
			var d34 scm.JITValueDesc
			if d30.Loc == scm.LocImm && d33.Loc == scm.LocImm {
				d34 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d30.Imm.Int()) >> uint64(d33.Imm.Int())))}
			} else if d33.Loc == scm.LocImm {
				ctx.W.EmitShrRegImm8(d30.Reg, uint8(d33.Imm.Int()))
				d34 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d30.Reg}
			} else {
				{
					shiftSrc := d30.Reg
					if shiftSrc == scm.RegRCX {
						newReg := ctx.AllocReg()
						ctx.W.EmitMovRegReg(newReg, scm.RegRCX)
						shiftSrc = newReg
					}
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
				}
			}
			if d34.Loc == scm.LocReg && d30.Loc == scm.LocReg && d34.Reg == d30.Reg {
				ctx.TransferReg(d30.Reg)
				d30.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d30)
			ctx.FreeDesc(&d33)
			var d35 scm.JITValueDesc
			if d15.Loc == scm.LocImm && d34.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d15.Imm.Int() | d34.Imm.Int())}
			} else if d15.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d15.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d34.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d34.Imm.Int()))
				ctx.W.EmitOrInt64(d15.Reg, scratch)
				ctx.FreeReg(scratch)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			} else {
				ctx.W.EmitOrInt64(d15.Reg, d34.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d15.Reg}
			}
			if d35.Loc == scm.LocReg && d15.Loc == scm.LocReg && d35.Reg == d15.Reg {
				ctx.TransferReg(d15.Reg)
				d15.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d34)
			ctx.EmitStoreToStack(d35, 16)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl5)
			ctx.W.ResolveFixups()
			d36 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r12}
			if r10 { ctx.UnprotectReg(r11) }
			var d37 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r29 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r29, thisptr.Reg, off)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r29}
			}
			var d39 scm.JITValueDesc
			if d36.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else if d36.Loc == scm.LocImm && d36.Imm.Int() == 0 {
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d37.Reg}
			} else if d36.Loc == scm.LocImm {
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d36.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			} else if d37.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d37.Imm.Int()))
				ctx.W.EmitAddInt64(d36.Reg, scm.RegR11)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			} else {
				ctx.W.EmitAddInt64(d36.Reg, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d36.Reg}
			}
			if d39.Loc == scm.LocReg && d36.Loc == scm.LocReg && d39.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d36)
			ctx.FreeDesc(&d37)
			var d41 scm.JITValueDesc
			if d39.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d39.Imm.Int() == idxInt.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpRegImm32(d39.Reg, int32(idxInt.Imm.Int()))
				ctx.W.EmitSetcc(r30, scm.CcE)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r30}
			} else if d39.Loc == scm.LocImm {
				r31 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, idxInt.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r31, scm.CcE)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r31}
			} else {
				r32 := ctx.AllocRegExcept(d39.Reg)
				ctx.W.EmitCmpInt64(d39.Reg, idxInt.Reg)
				ctx.W.EmitSetcc(r32, scm.CcE)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r32}
			}
			lbl11 := ctx.W.ReserveLabel()
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			if d41.Loc == scm.LocImm {
				if d41.Imm.Bool() {
					ctx.W.EmitJmp(lbl11)
				} else {
					ctx.W.EmitJmp(lbl12)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d41.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl13)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl13)
				ctx.W.EmitJmp(lbl11)
			}
			ctx.FreeDesc(&d41)
			ctx.W.MarkLabel(lbl2)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl12)
			var d43 scm.JITValueDesc
			if d39.Loc == scm.LocImm && idxInt.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d39.Imm.Int() < idxInt.Imm.Int())}
			} else if idxInt.Loc == scm.LocImm {
				r33 := ctx.AllocReg()
				ctx.W.EmitCmpRegImm32(d39.Reg, int32(idxInt.Imm.Int()))
				ctx.W.EmitSetcc(r33, scm.CcL)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r33}
			} else if d39.Loc == scm.LocImm {
				r34 := ctx.AllocReg()
				scratch := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.W.EmitCmpInt64(scratch, idxInt.Reg)
				ctx.FreeReg(scratch)
				ctx.W.EmitSetcc(r34, scm.CcL)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r34}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitCmpInt64(d39.Reg, idxInt.Reg)
				ctx.W.EmitSetcc(r35, scm.CcL)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r35}
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&idxInt)
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			lbl16 := ctx.W.ReserveLabel()
			if d43.Loc == scm.LocImm {
				if d43.Imm.Bool() {
					ctx.W.EmitJmp(lbl14)
				} else {
					ctx.W.EmitJmp(lbl15)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d43.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl16)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl16)
				ctx.W.EmitJmp(lbl14)
			}
			ctx.FreeDesc(&d43)
			ctx.W.MarkLabel(lbl11)
			var d44 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r36 := ctx.AllocReg()
				r37 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r36, fieldAddr)
				ctx.W.EmitMovRegMem64(r37, fieldAddr+8)
				d44 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r36, Reg2: r37}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r38 := ctx.AllocReg()
				r39 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r38, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r39, thisptr.Reg, off+8)
				d44 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r38, Reg2: r39}
			}
			r40 := ctx.AllocReg()
			if d6.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r40, uint64(d6.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r40, d6.Reg)
				ctx.W.EmitShlRegImm8(r40, 4)
			}
			if d44.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d44.Imm.Int()))
				ctx.W.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r40, d44.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			r42 := ctx.AllocRegExcept(r40, r41)
			ctx.W.EmitMovRegMem(r41, r40, 0)
			ctx.W.EmitMovRegMem(r42, r40, 8)
			ctx.FreeReg(r40)
			d45 := scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r41, Reg2: r42}
			ctx.EmitMovPairToResult(&d45, &result)
			result.Type = d45.Type
			ctx.W.EmitJmp(lbl0)
			ctx.FreeDesc(&d45)
			ctx.W.MarkLabel(lbl15)
			ctx.EmitStoreToStack(d2, 0)
			ctx.EmitStoreToStack(d6, 8)
			ctx.W.EmitJmp(lbl1)
			ctx.W.MarkLabel(lbl14)
			var d46 scm.JITValueDesc
			if d6.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d6.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d6.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(1))
				ctx.W.EmitAddInt64(scratch, d6.Reg)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
			}
			if d46.Loc == scm.LocReg && d6.Loc == scm.LocReg && d46.Reg == d6.Reg {
				ctx.TransferReg(d6.Reg)
				d6.Loc = scm.LocNone
			}
			ctx.EmitStoreToStack(d46, 0)
			ctx.EmitStoreToStack(d3, 8)
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
