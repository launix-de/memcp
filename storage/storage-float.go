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
				ctx.W.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.W.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
			}
			lbl0 := ctx.W.ReserveLabel()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageFloat)(nil).values)
				r0 := ctx.AllocReg()
				r1 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r0, fieldAddr)
				ctx.W.EmitMovRegMem64(r1, fieldAddr+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
				ctx.BindReg(r0, &d0)
				ctx.BindReg(r1, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageFloat)(nil).values))
				r2 := ctx.AllocReg()
				r3 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r2, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r3, thisptr.Reg, off+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d0)
				ctx.BindReg(r3, &d0)
			}
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r4 := ctx.AllocReg()
			if idxInt.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r4, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r4, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r4, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(r4, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r4, d0.Reg)
			}
			r5 := ctx.AllocRegExcept(r4)
			ctx.W.EmitMovRegMem(r5, r4, 0)
			ctx.FreeReg(r4)
			d1 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r5}
			ctx.BindReg(r5, &d1)
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			if d1.Loc == scm.LocStack || d1.Loc == scm.LocStackPair { ctx.EnsureDesc(&d1) }
			var d2 scm.JITValueDesc
			if d1.Loc == scm.LocImm {
				d2 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(d1.Imm.Int() != d1.Imm.Int())}
			} else if d1.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d1.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d1.Reg, int32(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.W.EmitCmpInt64(d1.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r6, scm.CcNE)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d2)
			} else if d1.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d1.Reg)
				ctx.W.EmitSetcc(r7, scm.CcNE)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d2)
			} else {
				r8 := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitCmpInt64(d1.Reg, d1.Reg)
				ctx.W.EmitSetcc(r8, scm.CcNE)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d2)
			}
			ctx.FreeDesc(&d1)
			if d2.Loc == scm.LocStack || d2.Loc == scm.LocStackPair { ctx.EnsureDesc(&d2) }
			ctx.FreeDesc(&d1)
			lbl1 := ctx.W.ReserveLabel()
			lbl2 := ctx.W.ReserveLabel()
			lbl3 := ctx.W.ReserveLabel()
			if d2.Loc == scm.LocImm {
				if d2.Imm.Bool() {
					ctx.W.EmitJmp(lbl1)
				} else {
					ctx.W.EmitJmp(lbl2)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d2.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl3)
				ctx.W.EmitJmp(lbl2)
				ctx.W.MarkLabel(lbl3)
				ctx.W.EmitJmp(lbl1)
			}
			ctx.FreeDesc(&d2)
			ctx.W.MarkLabel(lbl2)
			if idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair { ctx.EnsureDesc(&idxInt) }
			r9 := ctx.AllocReg()
			if idxInt.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r9, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r9, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r9, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(r9, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r9, d0.Reg)
			}
			r10 := ctx.AllocRegExcept(r9)
			ctx.W.EmitMovRegMem(r10, r9, 0)
			ctx.FreeReg(r9)
			d3 := scm.JITValueDesc{Loc: scm.LocReg, Reg: r10}
			ctx.BindReg(r10, &d3)
			ctx.FreeDesc(&idxInt)
			if d3.Loc == scm.LocStack || d3.Loc == scm.LocStackPair { ctx.EnsureDesc(&d3) }
			ctx.W.EmitMakeFloat(result, d3)
			if d3.Loc == scm.LocReg { ctx.FreeReg(d3.Reg) }
			result.Type = scm.TagFloat
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl1)
			ctx.W.EmitMakeNil(result)
			result.Type = scm.TagNil
			ctx.W.EmitJmp(lbl0)
			ctx.W.MarkLabel(lbl0)
			ctx.W.ResolveFixups()
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
