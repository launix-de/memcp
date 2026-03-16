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

func (s *StorageConst) GetCachedReader() ColumnReader { return s }

func (s *StorageConst) JITEmit(ctx *scm.JITContext, thisptr scm.JITValueDesc, idx scm.JITValueDesc, result scm.JITValueDesc) scm.JITValueDesc {
			var d0 scm.JITValueDesc
			_ = d0
	/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
			ctx.FreeDesc(&idx)
			var bbs [1]scm.BBDescriptor
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl0 := ctx.ReserveLabel()
			bbs[0].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[0].VisitCount >= 2 {
					ps.General = true
					return bbs[0].RenderPS(ps)
				}
			}
			bbs[0].VisitCount++
			if ps.General {
				if bbs[0].Rendered {
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
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
			ctx.ResolveFixups()
			if d0.Loc == scm.LocImm {
				if result.Loc == scm.LocAny { return d0 }
			}
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			ctx.EnsureDesc(&d0)
			if d0.Loc == scm.LocRegPair {
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
			ps1 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps1)
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
	var v any
	json.Unmarshal(buf, &v)
	s.value = scm.TransformFromJSON(v)
	return uint(s.count)
}
