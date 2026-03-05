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
			var d0 scm.JITValueDesc
			_ = d0
			var d1 scm.JITValueDesc
			_ = d1
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var d13 scm.JITValueDesc
			_ = d13
			var d14 scm.JITValueDesc
			_ = d14
			var d15 scm.JITValueDesc
			_ = d15
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
				ctx.EmitShlRegImm8(idxInt.Reg, 32)
				ctx.EmitShrRegImm8(idxInt.Reg, 32)
				ctx.BindReg(idxInt.Reg, &idxInt)
			}
			idxPinned := idxInt.Loc == scm.LocReg
			idxPinnedReg := idxInt.Reg
			if idxPinned { ctx.ProtectReg(idxPinnedReg) }
			var bbs [3]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r0 := ctx.AllocReg()
			r1 := ctx.AllocRegExcept(r0)
			lbl0 := ctx.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.ReserveLabel()
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
					ctx.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.MarkLabel(lbl1)
				ctx.ResolveFixups()
			}
			ctx.ReclaimUntrackedRegs()
			var d0 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageFloat)(nil).values)
				r2 := ctx.AllocReg()
				r3 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r2, fieldAddr)
				ctx.EmitMovRegMem64(r3, fieldAddr+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r2, Reg2: r3}
				ctx.BindReg(r2, &d0)
				ctx.BindReg(r3, &d0)
			} else {
				off := int32(unsafe.Offsetof((*StorageFloat)(nil).values))
				r4 := ctx.AllocReg()
				r5 := ctx.AllocReg()
				ctx.EmitMovRegMem(r4, thisptr.Reg, off)
				ctx.EmitMovRegMem(r5, thisptr.Reg, off+8)
				d0 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r4, Reg2: r5}
				ctx.BindReg(r4, &d0)
				ctx.BindReg(r5, &d0)
			}
			ctx.EnsureDesc(&idxInt)
			r6 := ctx.AllocReg()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d0)
			if idxInt.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r6, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r6, idxInt.Reg)
				ctx.EmitShlRegImm8(r6, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(r6, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r6, d0.Reg)
			}
			r7 := ctx.AllocRegExcept(r6)
			ctx.EmitMovRegMem(r7, r6, 0)
			ctx.FreeReg(r6)
			d1 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r7}
			ctx.BindReg(r7, &d1)
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocRegPair || d1.Loc == scm.LocStackPair {
				panic("jit: generic call arg expects 1-word value")
			}
			d2 = ctx.EmitGoCallScalar(scm.GoFuncAddr(math.IsNaN), []scm.JITValueDesc{d1}, 1)
			ctx.FreeDesc(&d1)
			d3 = d2
			ctx.EnsureDesc(&d3)
			if d3.Loc != scm.LocImm && d3.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d3.Loc == scm.LocImm {
				if d3.Imm.Bool() {
			ps4 := scm.PhiState{General: ps.General}
			ps4.OverlayValues = make([]scm.JITValueDesc, 4)
			ps4.OverlayValues[0] = d0
			ps4.OverlayValues[1] = d1
			ps4.OverlayValues[2] = d2
			ps4.OverlayValues[3] = d3
					return bbs[1].RenderPS(ps4)
				}
			ps5 := scm.PhiState{General: ps.General}
			ps5.OverlayValues = make([]scm.JITValueDesc, 4)
			ps5.OverlayValues[0] = d0
			ps5.OverlayValues[1] = d1
			ps5.OverlayValues[2] = d2
			ps5.OverlayValues[3] = d3
				return bbs[2].RenderPS(ps5)
			}
			if !ps.General {
				ps.General = true
				return bbs[0].RenderPS(ps)
			}
			lbl4 := ctx.ReserveLabel()
			lbl5 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d3.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl4)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl4)
			ctx.EmitJmp(lbl2)
			ctx.MarkLabel(lbl5)
			ctx.EmitJmp(lbl3)
			ps6 := scm.PhiState{General: true}
			ps6.OverlayValues = make([]scm.JITValueDesc, 4)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps7 := scm.PhiState{General: true}
			ps7.OverlayValues = make([]scm.JITValueDesc, 4)
			ps7.OverlayValues[0] = d0
			ps7.OverlayValues[1] = d1
			ps7.OverlayValues[2] = d2
			ps7.OverlayValues[3] = d3
			snap8 := d0
			snap9 := d1
			snap10 := d2
			snap11 := d3
			alloc12 := ctx.SnapshotAllocState()
			if !bbs[2].Rendered {
				bbs[2].RenderPS(ps7)
			}
			ctx.RestoreAllocState(alloc12)
			d0 = snap8
			d1 = snap9
			d2 = snap10
			d3 = snap11
			if !bbs[1].Rendered {
				return bbs[1].RenderPS(ps6)
			}
			return result
			ctx.FreeDesc(&d2)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.MarkLabel(lbl2)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			ctx.ReclaimUntrackedRegs()
			d13 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d13)
			ctx.BindReg(r1, &d13)
			ctx.EmitMakeNil(d13)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[2].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[2].VisitCount >= 2 {
					ps.General = true
					return bbs[2].RenderPS(ps)
				}
			}
			bbs[2].VisitCount++
			if ps.General {
				if bbs[2].Rendered {
					ctx.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.MarkLabel(lbl3)
				ctx.ResolveFixups()
			}
			if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 13 && ps.OverlayValues[13].Loc != scm.LocNone {
				d13 = ps.OverlayValues[13]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			r8 := ctx.AllocReg()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&d0)
			if idxInt.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r8, uint64(idxInt.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r8, idxInt.Reg)
				ctx.EmitShlRegImm8(r8, 3)
			}
			if d0.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(r8, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r8, d0.Reg)
			}
			r9 := ctx.AllocRegExcept(r8)
			ctx.EmitMovRegMem(r9, r8, 0)
			ctx.FreeReg(r8)
			d14 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r9}
			ctx.BindReg(r9, &d14)
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d14)
			d15 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d15)
			ctx.BindReg(r1, &d15)
			ctx.EnsureDesc(&d14)
			ctx.EmitMakeFloat(d15, d14)
			if d14.Loc == scm.LocReg { ctx.FreeReg(d14.Reg) }
			ctx.EmitJmp(lbl0)
			return result
			}
			ps16 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps16)
			ctx.MarkLabel(lbl0)
			d17 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r0, Reg2: r1}
			ctx.BindReg(r0, &d17)
			ctx.BindReg(r1, &d17)
			ctx.EmitMovPairToResult(&d17, &result)
			ctx.FreeReg(r0)
			ctx.FreeReg(r1)
			ctx.ResolveFixups()
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
