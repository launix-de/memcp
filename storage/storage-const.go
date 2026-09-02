/*
Copyright (C) 2026  Carl-Philip Hänsch

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

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/launix-de/memcp/scm"
	"io"
)
import "unsafe"

// StorageConst stores a column where every row has the same value.
// Zero per-element overhead: only the single constant value is stored.
type StorageConst struct {
	value scm.Scmer `jit:"immutable-after-finish"`
	count uint64
}

func (s *StorageConst) String() string {
	return fmt.Sprintf("const[%s]", s.value.String())
}

func (s *StorageConst) ComputeSize() uint {
	return 48 + scm.ComputeSize(s.value)
}

func (s *StorageConst) GetValue(i uint32) scm.Scmer {
	return s.value
}

// GetValueRange and GetValueMulti fill target with the single constant
// value directly; there is no per-row work to batch.
func (s *StorageConst) GetValueRange(recid uint32, count uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for k := uint32(0); k < count; k++ {
		target[idx] = s.value
		idx += stride
	}
}

func (s *StorageConst) GetValueMulti(recids []uint32, target []scm.Scmer, stride int) {
	if stride <= 0 {
		stride = 1
	}
	idx := 0
	for range recids {
		target[idx] = s.value
		idx += stride
	}
}

func (s *StorageConst) GetCachedReader() ColumnReader { return s }

func (s *StorageConst) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
	ctx.FreeDesc(&idx)
	var d0 scm.JITValueDesc
	if thisptr.Loc == scm.LocImm {
		fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageConst)(nil).value)
		val := *(*scm.Scmer)(unsafe.Pointer(fieldAddr))
		ctx.TrackImm(val)
		d0 = scm.JITValueDesc{Loc: scm.LocImm, Type: val.GetTag(), Imm: val}
	} else {
		off := int32(unsafe.Offsetof((*StorageConst)(nil).value))
		r0 := ctx.AllocReg()
		r1 := ctx.AllocRegExcept(r0)
		ctx.EmitMovRegMem(r0, thisptr.Reg, off)
		ctx.EmitMovRegMem(r1, thisptr.Reg, off+8)
		d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r0, Reg2: r1}
		ctx.BindReg(r0, &d0)
		ctx.BindReg(r1, &d0)
		ctx.BindReg(r0, &d0)
		ctx.BindReg(r1, &d0)
	}
	if d0.Loc == scm.LocImm {
		if result.Loc == scm.LocAny {
			return d0
		}
	}
	if result.Loc == scm.LocAny {
		result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
		ctx.BindReg(result.Reg, &result)
		ctx.BindReg(result.Reg2, &result)
	}
	ctx.SyncDesc(&d0)
	if d0.Loc == scm.LocRegPair || d0.Loc == scm.LocStackPair || d0.Loc == scm.LocInputPair {
		ctx.EmitMovPairToResult(&d0, &result)
		result.Type = d0.Type
	} else {
		switch d0.Type {
		case scm.TagBool:
			ctx.EmitMakeBool(result, d0)
			result.Type = scm.TagBool
		case scm.TagInt:
			ctx.EmitMakeInt(result, d0)
			result.Type = scm.TagInt
		case scm.TagFloat:
			ctx.EmitMakeFloat(result, d0)
			result.Type = scm.TagFloat
		case scm.TagNil:
			ctx.EmitMakeNil(result)
			result.Type = scm.TagNil
		default:
			panic("jit: single-block scalar return with unknown type")
		}
	}
	return result
	return result
}

func (s *StorageConst) prepare()                                  {}
func (s *StorageConst) scan(i uint32, value scm.Scmer)            {}
func (s *StorageConst) proposeCompression(i uint32) ColumnStorage { return nil }
func (s *StorageConst) init(i uint32)                             { s.count = uint64(i) }
func (s *StorageConst) build(i uint32, value scm.Scmer)           { s.value = value } // all rows identical; last assignment wins
func (s *StorageConst) finish()                                   {}

// Serialize: magic 41 + uint64 count + JSON-encoded value
func (s *StorageConst) Serialize(f io.Writer) {
	binary.Write(f, binary.LittleEndian, uint8(41))
	binary.Write(f, binary.LittleEndian, s.count)
	b, _ := json.Marshal(s.value)
	binary.Write(f, binary.LittleEndian, uint32(len(b)))
	f.Write(b)
}

func (s *StorageConst) Deserialize(f io.Reader) uint {
	binary.Read(f, binary.LittleEndian, &s.count)
	var vlen uint32
	binary.Read(f, binary.LittleEndian, &vlen)
	buf := make([]byte, vlen)
	io.ReadFull(f, buf)
	if err := json.Unmarshal(buf, &s.value); err != nil {
		panic(err)
	}
	return uint(s.count)
}

func (s *StorageConst) DistinctCount() uint { return 1 }
