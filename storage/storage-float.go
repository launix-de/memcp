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
import "math"
import "unsafe"
import "encoding/binary"
import "github.com/launix-de/memcp/scm"

// main type for storage: can store any value, is inefficient but does type analysis how to optimize
type StorageFloat struct {
	values []float64
}

func (s *StorageFloat) ComputeSize() uint {
	return 16 + 8*uint(len(s.values)) + 24 /* a slice */
}

func (s *StorageFloat) String() string {
	return "float64"
}

func (s *StorageFloat) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(12)) // 12 = StorageFloat
	io.WriteString(f, "1234567")                    // fill up to 64 bit alignment
	binary.Write(f, binary.LittleEndian, uint64(len(s.values)))
	// now at offset 16 begin data
	rawdata := unsafe.Slice((*byte)(unsafe.Pointer(&s.values[0])), 8*len(s.values))
	f.Write(rawdata)
	// free allocated memory and mmap
	/* TODO: runtime.SetFinalizer(s, func(s *StorageSCMER) {f.Close()})
	newrawdata = mmap.Map(f, RDWR, 0)
	s.values = unsafe.Slice((*float64)&newrawdata[16], len(s.values))
	*/
}
func (s *StorageFloat) Deserialize(f io.Reader) uint {
	var dummy [7]byte
	f.Read(dummy[:])
	var l uint64
	binary.Read(f, binary.LittleEndian, &l)
	/* TODO: runtime.SetFinalizer(s, func(s *StorageSCMER) { f.Close() })
	rawdata := mmap.Map(f, RDWR, 0)
	*/
	rawdata := make([]byte, 8*l)
	f.Read(rawdata)
	s.values = unsafe.Slice((*float64)(unsafe.Pointer(&rawdata[0])), l)
	return uint(l)
}

func (s *StorageFloat) GetCachedReader() ColumnReader { return s }

func (s *StorageFloat) GetValue(i uint32) scm.Scmer {
	// NULL is encoded as NaN in SQL
	if math.IsNaN(s.values[i]) {
		return scm.NewNil()
	}
	return scm.NewFloat(s.values[i])
}
func (s *StorageFloat) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
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
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageFloat)(nil).values)
				r2 := ctx.AllocReg()
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r2, fieldAddr)
				ctx.W.EmitMovRegMem64(r3, fieldAddr+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d0)
				ctx.BindReg(r3, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageFloat)(nil).values))
				r4 := ctx.AllocReg()
				r5 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r4, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r5, thisptr.Reg, off+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d0)
				ctx.BindReg(r5, &d0)
			}
			ctx.EnsureDesc(&idxInt)
			r6 := ctx.AllocReg()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d0)
			if idxInt.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r6, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r6, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r6, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(r6, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r6, d0.Reg)
			}
			r7 := ctx.AllocRegExcept(r6)
			ctx.W.EmitMovRegMem(r7, r6, 0)
			ctx.FreeReg(r6)
			d1 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d1)
			ctx.EnsureDesc(&d1)
			d2 := d1
			_ = d2
			r8 := d1.Loc == scm.LocReg
			r9 := d1.Reg
			if r8 { ctx.ProtectReg(r9) }
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d2.Imm.Int() != d2.Imm.Int())}
			} else if d2.Loc == scm.LocImm {
				r10 := ctx.AllocRegExcept(d2.Reg)
				if d2.Imm.Int() >= -2147483648 && d2.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d2.Reg, int32(d2.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
					ctx.W.EmitCmpInt64(d2.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r10, scm.CcNE)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r10}
				ctx.BindReg(r10, &d3)
			} else if d2.Loc == scm.LocImm {
				r11 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d2.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d2.Reg)
				ctx.W.EmitSetcc(r11, scm.CcNE)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r11}
				ctx.BindReg(r11, &d3)
			} else {
				r12 := ctx.AllocRegExcept(d2.Reg)
				ctx.W.EmitCmpInt64(d2.Reg, d2.Reg)
				ctx.W.EmitSetcc(r12, scm.CcNE)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r12}
				ctx.BindReg(r12, &d3)
			}
			ctx.EnsureDesc(&d3)
			if r8 { ctx.UnprotectReg(r9) }
			ctx.FreeDesc(&d1)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d3.Loc == scm.LocImm {
				if d3.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d3.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d3)
			ctx.W.MarkLabel(lbl2)
			ctx.EnsureDesc(&idxInt)
			r13 := ctx.AllocReg()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d0)
			if idxInt.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r13, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r13, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r13, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(r13, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r13, d0.Reg)
			}
			r14 := ctx.AllocRegExcept(r13)
			ctx.W.EmitMovRegMem(r14, r13, 0)
			ctx.FreeReg(r13)
			d4 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r14}
			ctx.BindReg(r14, &d4)
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d4)
			d5 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d5)
			ctx.BindReg(r1, &d5)
			ctx.EnsureDesc(&d4)
			ctx.W.EmitMakeFloat(d5, d4)
			if d4.Loc == scm.LocReg { ctx.FreeReg(d4.Reg) }
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			d6 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d6)
			ctx.BindReg(r1, &d6)
			ctx.W.EmitMakeNil(d6)
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			d7 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d7)
			ctx.BindReg(r1, &d7)
			ctx.EmitMovPairToResult(&d7, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			return result
}

func (s *StorageFloat) scan(i uint32, value scm.Scmer) {
}
func (s *StorageFloat) prepare() {
}
func (s *StorageFloat) init(i uint32) {
	// allocate
	s.values = make([]float64, i)
}
func (s *StorageFloat) build(i uint32, value scm.Scmer) {
	// store
	if value.IsNil() {
		s.values[i] = math.NaN()
	} else {
		s.values[i] = value.Float()
	}
}
func (s *StorageFloat) finish() {
}

func (s *StorageFloat) proposeCompression(i uint32) ColumnStorage {
	// dont't propose another pass
	return nil
}
