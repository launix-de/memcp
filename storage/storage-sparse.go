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
			var d2 scm.JITValueDesc
			_ = d2
			var d3 scm.JITValueDesc
			_ = d3
			var d4 scm.JITValueDesc
			_ = d4
			var d5 scm.JITValueDesc
			_ = d5
			var d7 scm.JITValueDesc
			_ = d7
			var d8 scm.JITValueDesc
			_ = d8
			var d9 scm.JITValueDesc
			_ = d9
			var d10 scm.JITValueDesc
			_ = d10
			var d11 scm.JITValueDesc
			_ = d11
			var d12 scm.JITValueDesc
			_ = d12
			var d15 scm.JITValueDesc
			_ = d15
			var d16 scm.JITValueDesc
			_ = d16
			var d34 scm.JITValueDesc
			_ = d34
			var d35 scm.JITValueDesc
			_ = d35
			var d36 scm.JITValueDesc
			_ = d36
			var d37 scm.JITValueDesc
			_ = d37
			var d38 scm.JITValueDesc
			_ = d38
			var d39 scm.JITValueDesc
			_ = d39
			var d40 scm.JITValueDesc
			_ = d40
			var d41 scm.JITValueDesc
			_ = d41
			var d42 scm.JITValueDesc
			_ = d42
			var d43 scm.JITValueDesc
			_ = d43
			var d44 scm.JITValueDesc
			_ = d44
			var d45 scm.JITValueDesc
			_ = d45
			var d46 scm.JITValueDesc
			_ = d46
			var d47 scm.JITValueDesc
			_ = d47
			var d48 scm.JITValueDesc
			_ = d48
			var d49 scm.JITValueDesc
			_ = d49
			var d50 scm.JITValueDesc
			_ = d50
			var d51 scm.JITValueDesc
			_ = d51
			var d52 scm.JITValueDesc
			_ = d52
			var d53 scm.JITValueDesc
			_ = d53
			var d54 scm.JITValueDesc
			_ = d54
			var d55 scm.JITValueDesc
			_ = d55
			var d56 scm.JITValueDesc
			_ = d56
			var d57 scm.JITValueDesc
			_ = d57
			var d58 scm.JITValueDesc
			_ = d58
			var d59 scm.JITValueDesc
			_ = d59
			var d60 scm.JITValueDesc
			_ = d60
			var d61 scm.JITValueDesc
			_ = d61
			var d62 scm.JITValueDesc
			_ = d62
			var d63 scm.JITValueDesc
			_ = d63
			var d64 scm.JITValueDesc
			_ = d64
			var d65 scm.JITValueDesc
			_ = d65
			var d66 scm.JITValueDesc
			_ = d66
			var d67 scm.JITValueDesc
			_ = d67
			var d68 scm.JITValueDesc
			_ = d68
			var d69 scm.JITValueDesc
			_ = d69
			var d70 scm.JITValueDesc
			_ = d70
			var d71 scm.JITValueDesc
			_ = d71
			var d72 scm.JITValueDesc
			_ = d72
			var d73 scm.JITValueDesc
			_ = d73
			var d74 scm.JITValueDesc
			_ = d74
			var d75 scm.JITValueDesc
			_ = d75
			var d76 scm.JITValueDesc
			_ = d76
			var d77 scm.JITValueDesc
			_ = d77
			var d78 scm.JITValueDesc
			_ = d78
			var d79 scm.JITValueDesc
			_ = d79
			var d80 scm.JITValueDesc
			_ = d80
			var d147 scm.JITValueDesc
			_ = d147
			var d148 scm.JITValueDesc
			_ = d148
			var d149 scm.JITValueDesc
			_ = d149
			var d150 scm.JITValueDesc
			_ = d150
			var d151 scm.JITValueDesc
			_ = d151
			var d152 scm.JITValueDesc
			_ = d152
			var d225 scm.JITValueDesc
			_ = d225
			var d226 scm.JITValueDesc
			_ = d226
			var d227 scm.JITValueDesc
			_ = d227
			var d228 scm.JITValueDesc
			_ = d228
			var d229 scm.JITValueDesc
			_ = d229
			var d231 scm.JITValueDesc
			_ = d231
			var d232 scm.JITValueDesc
			_ = d232
			var d233 scm.JITValueDesc
			_ = d233
			var d234 scm.JITValueDesc
			_ = d234
			var d235 scm.JITValueDesc
			_ = d235
			var d236 scm.JITValueDesc
			_ = d236
			var d238 scm.JITValueDesc
			_ = d238
			var d239 scm.JITValueDesc
			_ = d239
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
			r0 := ctx.EmitSubRSP32Fixup()
			_ = r0
			if thisptr.MemPtr == 0 && (thisptr.Loc == scm.LocStack || thisptr.Loc == scm.LocStackPair) {
				thisptr.StackOff += int32(32)
			}
			if idxInt.MemPtr == 0 && (idxInt.Loc == scm.LocStack || idxInt.Loc == scm.LocStackPair) {
				idxInt.StackOff += int32(32)
			}
			if result.MemPtr == 0 && (result.Loc == scm.LocStack || result.Loc == scm.LocStackPair) {
				result.StackOff += int32(32)
			}
			d0 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			var bbs [8]scm.BBDescriptor
			bbs[1].PhiBase = int32(0)
			bbs[1].PhiCount = uint16(2)
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
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
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.ReserveLabel()
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			var d2 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).i)
				r3 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r3, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
				ctx.BindReg(r3, &d2)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r4 := ctx.AllocReg()
				ctx.EmitMovRegMem(r4, thisptr.Reg, off)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r4}
				ctx.BindReg(r4, &d2)
			}
			ctx.EnsureDesc(&d2)
			ctx.EnsureDesc(&d2)
			var d3 scm.JITValueDesc
			if d2.Loc == scm.LocImm {
				d3 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint32(uint64(d2.Imm.Int()))))}
			} else {
				r5 := ctx.AllocReg()
				ctx.EmitMovRegReg(r5, d2.Reg)
				ctx.EmitShlRegImm8(r5, 32)
				ctx.EmitShrRegImm8(r5, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d3)
			}
			ctx.EnsureDesc(&d3)
			if d3.Loc == scm.LocReg {
				ctx.ProtectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.ProtectReg(d3.Reg)
				ctx.ProtectReg(d3.Reg2)
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}, int32(bbs[1].PhiBase)+int32(0))
			d4 = d3
			if d4.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d4)
			d5 = d4
			if d5.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: d5.Type, Imm: scm.NewInt(int64(uint64(d5.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d5.Reg, 32)
				ctx.EmitShrRegImm8(d5.Reg, 32)
			}
			ctx.EmitStoreToStack(d5, int32(bbs[1].PhiBase)+int32(16))
			if d3.Loc == scm.LocReg {
				ctx.UnprotectReg(d3.Reg)
			} else if d3.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d3.Reg)
				ctx.UnprotectReg(d3.Reg2)
			}
			ps6 := scm.PhiState{General: ps.General}
			ps6.OverlayValues = make([]scm.JITValueDesc, 6)
			ps6.OverlayValues[0] = d0
			ps6.OverlayValues[1] = d1
			ps6.OverlayValues[2] = d2
			ps6.OverlayValues[3] = d3
			ps6.OverlayValues[4] = d4
			ps6.OverlayValues[5] = d5
			ps6.PhiValues = make([]scm.JITValueDesc, 2)
			d7 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(0)}
			ps6.PhiValues[0] = d7
			d8 = d3
			ps6.PhiValues[1] = d8
			if ps6.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps6)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d9 := ps.PhiValues[0]
					ctx.EnsureDesc(&d9)
					ctx.EmitStoreToStack(d9, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d10 := ps.PhiValues[1]
					ctx.EnsureDesc(&d10)
					ctx.EmitStoreToStack(d10, int32(bbs[1].PhiBase)+int32(16))
				}
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
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
				d0 = ps.PhiValues[0]
			}
			if !ps.General && len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
				d1 = ps.PhiValues[1]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d11 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d11 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d0.Imm.Int()) == uint64(d1.Imm.Int()))}
			} else if d1.Loc == scm.LocImm {
				r6 := ctx.AllocRegExcept(d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d0.Reg, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.EmitCmpInt64(d0.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r6, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d11)
			} else if d0.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d1.Reg)
				ctx.EmitSetcc(r7, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d11)
			} else {
				r8 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitCmpInt64(d0.Reg, d1.Reg)
				ctx.EmitSetcc(r8, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r8}
				ctx.BindReg(r8, &d11)
			}
			d12 = d11
			ctx.EnsureDesc(&d12)
			if d12.Loc != scm.LocImm && d12.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d12.Loc == scm.LocImm {
				if d12.Imm.Bool() {
			ps13 := scm.PhiState{General: ps.General}
			ps13.OverlayValues = make([]scm.JITValueDesc, 13)
			ps13.OverlayValues[0] = d0
			ps13.OverlayValues[1] = d1
			ps13.OverlayValues[2] = d2
			ps13.OverlayValues[3] = d3
			ps13.OverlayValues[4] = d4
			ps13.OverlayValues[5] = d5
			ps13.OverlayValues[7] = d7
			ps13.OverlayValues[8] = d8
			ps13.OverlayValues[9] = d9
			ps13.OverlayValues[10] = d10
			ps13.OverlayValues[11] = d11
			ps13.OverlayValues[12] = d12
					return bbs[2].RenderPS(ps13)
				}
			ps14 := scm.PhiState{General: ps.General}
			ps14.OverlayValues = make([]scm.JITValueDesc, 13)
			ps14.OverlayValues[0] = d0
			ps14.OverlayValues[1] = d1
			ps14.OverlayValues[2] = d2
			ps14.OverlayValues[3] = d3
			ps14.OverlayValues[4] = d4
			ps14.OverlayValues[5] = d5
			ps14.OverlayValues[7] = d7
			ps14.OverlayValues[8] = d8
			ps14.OverlayValues[9] = d9
			ps14.OverlayValues[10] = d10
			ps14.OverlayValues[11] = d11
			ps14.OverlayValues[12] = d12
				return bbs[3].RenderPS(ps14)
			}
			if !ps.General {
				if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
					d15 := ps.PhiValues[0]
					ctx.EnsureDesc(&d15)
					ctx.EmitStoreToStack(d15, int32(bbs[1].PhiBase)+int32(0))
				}
				if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
					d16 := ps.PhiValues[1]
					ctx.EnsureDesc(&d16)
					ctx.EmitStoreToStack(d16, int32(bbs[1].PhiBase)+int32(16))
				}
				ps.General = true
				return bbs[1].RenderPS(ps)
			}
			lbl9 := ctx.ReserveLabel()
			lbl10 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d12.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl9)
			ctx.EmitJmp(lbl10)
			ctx.MarkLabel(lbl9)
			ctx.EmitJmp(lbl3)
			ctx.MarkLabel(lbl10)
			ctx.EmitJmp(lbl4)
			ps17 := scm.PhiState{General: true}
			ps17.OverlayValues = make([]scm.JITValueDesc, 17)
			ps17.OverlayValues[0] = d0
			ps17.OverlayValues[1] = d1
			ps17.OverlayValues[2] = d2
			ps17.OverlayValues[3] = d3
			ps17.OverlayValues[4] = d4
			ps17.OverlayValues[5] = d5
			ps17.OverlayValues[7] = d7
			ps17.OverlayValues[8] = d8
			ps17.OverlayValues[9] = d9
			ps17.OverlayValues[10] = d10
			ps17.OverlayValues[11] = d11
			ps17.OverlayValues[12] = d12
			ps17.OverlayValues[15] = d15
			ps17.OverlayValues[16] = d16
			ps18 := scm.PhiState{General: true}
			ps18.OverlayValues = make([]scm.JITValueDesc, 17)
			ps18.OverlayValues[0] = d0
			ps18.OverlayValues[1] = d1
			ps18.OverlayValues[2] = d2
			ps18.OverlayValues[3] = d3
			ps18.OverlayValues[4] = d4
			ps18.OverlayValues[5] = d5
			ps18.OverlayValues[7] = d7
			ps18.OverlayValues[8] = d8
			ps18.OverlayValues[9] = d9
			ps18.OverlayValues[10] = d10
			ps18.OverlayValues[11] = d11
			ps18.OverlayValues[12] = d12
			ps18.OverlayValues[15] = d15
			ps18.OverlayValues[16] = d16
			snap19 := d0
			snap20 := d1
			snap21 := d2
			snap22 := d3
			snap23 := d4
			snap24 := d5
			snap25 := d7
			snap26 := d8
			snap27 := d9
			snap28 := d10
			snap29 := d11
			snap30 := d12
			snap31 := d15
			snap32 := d16
			alloc33 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps18)
			}
			ctx.RestoreAllocState(alloc33)
			d0 = snap19
			d1 = snap20
			d2 = snap21
			d3 = snap22
			d4 = snap23
			d5 = snap24
			d7 = snap25
			d8 = snap26
			d9 = snap27
			d10 = snap28
			d11 = snap29
			d12 = snap30
			d15 = snap31
			d16 = snap32
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps17)
			}
			return result
			ctx.FreeDesc(&d11)
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
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			ctx.ReclaimUntrackedRegs()
			d34 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d34)
			ctx.BindReg(r2, &d34)
			ctx.EmitMakeNil(d34)
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[3].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[3].VisitCount >= 2 {
					ps.General = true
					return bbs[3].RenderPS(ps)
				}
			}
			bbs[3].VisitCount++
			if ps.General {
				if bbs[3].Rendered {
					ctx.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.MarkLabel(lbl4)
				ctx.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d35 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() + d1.Imm.Int())}
			} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(r9, d0.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d35)
			} else if d0.Loc == scm.LocImm && d0.Imm.Int() == 0 {
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d35)
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.EmitAddInt64(scratch, d1.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.EmitMovRegReg(scratch, d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d1.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d35)
			} else {
				r10 := ctx.AllocRegExcept(d0.Reg, d1.Reg)
				ctx.EmitMovRegReg(r10, d0.Reg)
				ctx.EmitAddInt64(r10, d1.Reg)
				d35 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d35)
			}
			if d35.Loc == scm.LocImm {
				d35 = scm.JITValueDesc{Loc: scm.LocImm, Type: d35.Type, Imm: scm.NewInt(int64(uint64(d35.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d35.Reg, 32)
				ctx.EmitShrRegImm8(d35.Reg, 32)
			}
			if d35.Loc == scm.LocReg && d0.Loc == scm.LocReg && d35.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d35)
			var d36 scm.JITValueDesc
			if d35.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d35.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d35.Reg)
				ctx.EmitMovRegReg(r11, d35.Reg)
				ctx.EmitShrRegImm8(r11, 1)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d36)
			}
			if d36.Loc == scm.LocImm {
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: d36.Type, Imm: scm.NewInt(int64(uint64(d36.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d36.Reg, 32)
				ctx.EmitShrRegImm8(d36.Reg, 32)
			}
			if d36.Loc == scm.LocReg && d35.Loc == scm.LocReg && d36.Reg == d35.Reg {
				ctx.TransferReg(d35.Reg)
				d35.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d35)
			ctx.EnsureDesc(&d36)
			d37 = d36
			_ = d37
			r12 := d36.Loc == scm.LocReg
			r13 := d36.Reg
			if r12 { ctx.ProtectReg(r13) }
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			lbl11 := ctx.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d37)
			var d39 scm.JITValueDesc
			if d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d37.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.EmitMovRegReg(r14, d37.Reg)
				ctx.EmitShlRegImm8(r14, 32)
				ctx.EmitShrRegImm8(r14, 32)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d39)
			}
			var d40 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r15, thisptr.Reg, off)
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d40)
			}
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			var d41 scm.JITValueDesc
			if d40.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d40.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.EmitMovRegReg(r16, d40.Reg)
				ctx.EmitShlRegImm8(r16, 56)
				ctx.EmitShrRegImm8(r16, 56)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d41)
			}
			ctx.FreeDesc(&d40)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d39)
			ctx.EnsureDesc(&d41)
			var d42 scm.JITValueDesc
			if d39.Loc == scm.LocImm && d41.Loc == scm.LocImm {
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d39.Imm.Int() * d41.Imm.Int())}
			} else if d39.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d39.Imm.Int()))
				ctx.EmitImulInt64(scratch, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d39.Reg)
				ctx.EmitMovRegReg(scratch, d39.Reg)
				if d41.Imm.Int() >= -2147483648 && d41.Imm.Int() <= 2147483647 {
					ctx.EmitImulRegImm32(scratch, int32(d41.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d41.Imm.Int()))
					ctx.EmitImulInt64(scratch, scm.RegR11)
				}
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d42)
			} else {
				r17 := ctx.AllocRegExcept(d39.Reg, d41.Reg)
				ctx.EmitMovRegReg(r17, d39.Reg)
				ctx.EmitImulInt64(r17, d41.Reg)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d42)
			}
			if d42.Loc == scm.LocReg && d39.Loc == scm.LocReg && d42.Reg == d39.Reg {
				ctx.TransferReg(d39.Reg)
				d39.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d39)
			ctx.FreeDesc(&d41)
			var d43 scm.JITValueDesc
			r18 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.EmitMovRegImm64(r18, uint64(dataPtr))
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18, StackOff: int32(sliceLen)}
				ctx.BindReg(r18, &d43)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				ctx.EmitMovRegMem(r18, thisptr.Reg, off)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d43)
			}
			ctx.BindReg(r18, &d43)
			ctx.EnsureDesc(&d42)
			var d44 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() / 64)}
			} else {
				r19 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r19, d42.Reg)
				ctx.EmitShrRegImm8(r19, 6)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d44)
			}
			if d44.Loc == scm.LocReg && d42.Loc == scm.LocReg && d44.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d44)
			r20 := ctx.AllocReg()
			ctx.EnsureDesc(&d44)
			ctx.EnsureDesc(&d43)
			if d44.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r20, uint64(d44.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r20, d44.Reg)
				ctx.EmitShlRegImm8(r20, 3)
			}
			if d43.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.EmitAddInt64(r20, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r20, d43.Reg)
			}
			r21 := ctx.AllocRegExcept(r20)
			ctx.EmitMovRegMem(r21, r20, 0)
			ctx.FreeReg(r20)
			d45 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d45)
			ctx.FreeDesc(&d44)
			ctx.EnsureDesc(&d42)
			var d46 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d46 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r22, d42.Reg)
				ctx.EmitAndRegImm32(r22, 63)
				d46 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d46)
			}
			if d46.Loc == scm.LocReg && d42.Loc == scm.LocReg && d46.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d45)
			ctx.EnsureDesc(&d46)
			var d47 scm.JITValueDesc
			if d45.Loc == scm.LocImm && d46.Loc == scm.LocImm {
				d47 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d45.Imm.Int()) << uint64(d46.Imm.Int())))}
			} else if d46.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d45.Reg)
				ctx.EmitMovRegReg(r23, d45.Reg)
				ctx.EmitShlRegImm8(r23, uint8(d46.Imm.Int()))
				d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d47)
			} else {
				{
					shiftSrc := d45.Reg
					r24 := ctx.AllocRegExcept(d45.Reg)
					ctx.EmitMovRegReg(r24, d45.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d46.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d46.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d46.Reg)
					}
					ctx.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d47 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d47)
				}
			}
			if d47.Loc == scm.LocReg && d45.Loc == scm.LocReg && d47.Reg == d45.Reg {
				ctx.TransferReg(d45.Reg)
				d45.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d45)
			ctx.FreeDesc(&d46)
			var d48 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d48 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r25 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r25, thisptr.Reg, off)
				d48 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d48)
			}
			d49 = d48
			ctx.EnsureDesc(&d49)
			if d49.Loc != scm.LocImm && d49.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.ReserveLabel()
			lbl13 := ctx.ReserveLabel()
			lbl14 := ctx.ReserveLabel()
			lbl15 := ctx.ReserveLabel()
			if d49.Loc == scm.LocImm {
				if d49.Imm.Bool() {
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl12)
				} else {
					ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d50 = d47
			if d50.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d50)
			ctx.EmitStoreToStack(d50, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
					ctx.EmitJmp(lbl13)
				}
			} else {
				ctx.EmitCmpRegImm32(d49.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl14)
				ctx.EmitJmp(lbl15)
				ctx.MarkLabel(lbl14)
				ctx.EmitJmp(lbl12)
				ctx.MarkLabel(lbl15)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d51 = d47
			if d51.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d51)
			ctx.EmitStoreToStack(d51, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
				ctx.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d48)
			bbpos_1_2 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl13)
			ctx.ResolveFixups()
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			var d52 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r26 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r26, thisptr.Reg, off)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d52)
			}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d52)
			var d53 scm.JITValueDesc
			if d52.Loc == scm.LocImm {
				d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d52.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.EmitMovRegReg(r27, d52.Reg)
				ctx.EmitShlRegImm8(r27, 56)
				ctx.EmitShrRegImm8(r27, 56)
				d53 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d53)
			}
			ctx.FreeDesc(&d52)
			d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d54)
			ctx.EnsureDesc(&d53)
			var d55 scm.JITValueDesc
			if d54.Loc == scm.LocImm && d53.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d54.Imm.Int() - d53.Imm.Int())}
			} else if d53.Loc == scm.LocImm && d53.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d54.Reg)
				ctx.EmitMovRegReg(r28, d54.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d55)
			} else if d54.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d54.Imm.Int()))
				ctx.EmitSubInt64(scratch, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d54.Reg)
				ctx.EmitMovRegReg(scratch, d54.Reg)
				if d53.Imm.Int() >= -2147483648 && d53.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d53.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d53.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d55)
			} else {
				r29 := ctx.AllocRegExcept(d54.Reg, d53.Reg)
				ctx.EmitMovRegReg(r29, d54.Reg)
				ctx.EmitSubInt64(r29, d53.Reg)
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d55)
			}
			if d55.Loc == scm.LocReg && d54.Loc == scm.LocReg && d55.Reg == d54.Reg {
				ctx.TransferReg(d54.Reg)
				d54.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d53)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d38.Imm.Int()) >> uint64(d55.Imm.Int())))}
			} else if d55.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d38.Reg)
				ctx.EmitMovRegReg(r30, d38.Reg)
				ctx.EmitShrRegImm8(r30, uint8(d55.Imm.Int()))
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d56)
			} else {
				{
					shiftSrc := d38.Reg
					r31 := ctx.AllocRegExcept(d38.Reg)
					ctx.EmitMovRegReg(r31, d38.Reg)
					shiftSrc = r31
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d55.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d55.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d55.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d56)
				}
			}
			if d56.Loc == scm.LocReg && d38.Loc == scm.LocReg && d56.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d38)
			ctx.FreeDesc(&d55)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d56)
			ctx.EnsureDesc(&d56)
			if d56.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d56)
			}
			ctx.EmitJmp(lbl11)
			bbpos_1_3 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl12)
			ctx.ResolveFixups()
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d42)
			var d57 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d57 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r33, d42.Reg)
				ctx.EmitAndRegImm32(r33, 63)
				d57 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d57)
			}
			if d57.Loc == scm.LocReg && d42.Loc == scm.LocReg && d57.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			var d58 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d58 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r34 := ctx.AllocReg()
				ctx.EmitMovRegMemB(r34, thisptr.Reg, off)
				d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
				ctx.BindReg(r34, &d58)
			}
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d58)
			var d59 scm.JITValueDesc
			if d58.Loc == scm.LocImm {
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d58.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.EmitMovRegReg(r35, d58.Reg)
				ctx.EmitShlRegImm8(r35, 56)
				ctx.EmitShrRegImm8(r35, 56)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d59)
			}
			ctx.FreeDesc(&d58)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d57)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d57.Loc == scm.LocImm && d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d57.Imm.Int() + d59.Imm.Int())}
			} else if d59.Loc == scm.LocImm && d59.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegReg(r36, d57.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d60)
			} else if d57.Loc == scm.LocImm && d57.Imm.Int() == 0 {
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d59.Reg}
				ctx.BindReg(d59.Reg, &d60)
			} else if d57.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d59.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d57.Imm.Int()))
				ctx.EmitAddInt64(scratch, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else if d59.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d57.Reg)
				ctx.EmitMovRegReg(scratch, d57.Reg)
				if d59.Imm.Int() >= -2147483648 && d59.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d59.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d59.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d60)
			} else {
				r37 := ctx.AllocRegExcept(d57.Reg, d59.Reg)
				ctx.EmitMovRegReg(r37, d57.Reg)
				ctx.EmitAddInt64(r37, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d60)
			}
			if d60.Loc == scm.LocReg && d57.Loc == scm.LocReg && d60.Reg == d57.Reg {
				ctx.TransferReg(d57.Reg)
				d57.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d57)
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d60.Imm.Int()) > uint64(64))}
			} else {
				r38 := ctx.AllocRegExcept(d60.Reg)
				ctx.EmitCmpRegImm32(d60.Reg, 64)
				ctx.EmitSetcc(r38, scm.CcA)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
				ctx.BindReg(r38, &d61)
			}
			ctx.FreeDesc(&d60)
			d62 = d61
			ctx.EnsureDesc(&d62)
			if d62.Loc != scm.LocImm && d62.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.ReserveLabel()
			lbl17 := ctx.ReserveLabel()
			lbl18 := ctx.ReserveLabel()
			if d62.Loc == scm.LocImm {
				if d62.Imm.Bool() {
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl16)
				} else {
					ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d63 = d47
			if d63.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d63)
			ctx.EmitStoreToStack(d63, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
					ctx.EmitJmp(lbl13)
				}
			} else {
				ctx.EmitCmpRegImm32(d62.Reg, 0)
				ctx.EmitJcc(scm.CcNE, lbl17)
				ctx.EmitJmp(lbl18)
				ctx.MarkLabel(lbl17)
				ctx.EmitJmp(lbl16)
				ctx.MarkLabel(lbl18)
			ctx.EnsureDesc(&d47)
			if d47.Loc == scm.LocReg {
				ctx.ProtectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.ProtectReg(d47.Reg)
				ctx.ProtectReg(d47.Reg2)
			}
			d64 = d47
			if d64.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d64)
			ctx.EmitStoreToStack(d64, int32(bbs[2].PhiBase)+int32(0))
			if d47.Loc == scm.LocReg {
				ctx.UnprotectReg(d47.Reg)
			} else if d47.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d47.Reg)
				ctx.UnprotectReg(d47.Reg2)
			}
				ctx.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d61)
			bbpos_1_1 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
			ctx.MarkLabel(lbl16)
			ctx.ResolveFixups()
			d38 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(32)}
			ctx.EnsureDesc(&d42)
			var d65 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d65 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r39, d42.Reg)
				ctx.EmitShrRegImm8(r39, 6)
				d65 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d65)
			}
			if d65.Loc == scm.LocReg && d42.Loc == scm.LocReg && d65.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d65)
			ctx.EnsureDesc(&d65)
			var d66 scm.JITValueDesc
			if d65.Loc == scm.LocImm {
				d66 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d65.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d65.Reg)
				ctx.EmitMovRegReg(scratch, d65.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d66 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d66)
			}
			if d66.Loc == scm.LocReg && d65.Loc == scm.LocReg && d66.Reg == d65.Reg {
				ctx.TransferReg(d65.Reg)
				d65.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d65)
			ctx.EnsureDesc(&d66)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d66)
			ctx.EnsureDesc(&d43)
			if d66.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r40, uint64(d66.Imm.Int()) * 8)
			} else {
				ctx.EmitMovRegReg(r40, d66.Reg)
				ctx.EmitShlRegImm8(r40, 3)
			}
			if d43.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
				ctx.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r40, d43.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			ctx.EmitMovRegMem(r41, r40, 0)
			ctx.FreeReg(r40)
			d67 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d67)
			ctx.FreeDesc(&d66)
			ctx.EnsureDesc(&d42)
			var d68 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d68 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d42.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d42.Reg)
				ctx.EmitMovRegReg(r42, d42.Reg)
				ctx.EmitAndRegImm32(r42, 63)
				d68 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d68)
			}
			if d68.Loc == scm.LocReg && d42.Loc == scm.LocReg && d68.Reg == d42.Reg {
				ctx.TransferReg(d42.Reg)
				d42.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d42)
			d69 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d68)
			ctx.EnsureDesc(&d69)
			ctx.EnsureDesc(&d68)
			var d70 scm.JITValueDesc
			if d69.Loc == scm.LocImm && d68.Loc == scm.LocImm {
				d70 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d69.Imm.Int() - d68.Imm.Int())}
			} else if d68.Loc == scm.LocImm && d68.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d69.Reg)
				ctx.EmitMovRegReg(r43, d69.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d70)
			} else if d69.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d68.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d69.Imm.Int()))
				ctx.EmitSubInt64(scratch, d68.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else if d68.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d69.Reg)
				ctx.EmitMovRegReg(scratch, d69.Reg)
				if d68.Imm.Int() >= -2147483648 && d68.Imm.Int() <= 2147483647 {
					ctx.EmitSubRegImm32(scratch, int32(d68.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d68.Imm.Int()))
					ctx.EmitSubInt64(scratch, scm.RegR11)
				}
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d70)
			} else {
				r44 := ctx.AllocRegExcept(d69.Reg, d68.Reg)
				ctx.EmitMovRegReg(r44, d69.Reg)
				ctx.EmitSubInt64(r44, d68.Reg)
				d70 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d70)
			}
			if d70.Loc == scm.LocReg && d69.Loc == scm.LocReg && d70.Reg == d69.Reg {
				ctx.TransferReg(d69.Reg)
				d69.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d68)
			ctx.EnsureDesc(&d67)
			ctx.EnsureDesc(&d70)
			var d71 scm.JITValueDesc
			if d67.Loc == scm.LocImm && d70.Loc == scm.LocImm {
				d71 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d67.Imm.Int()) >> uint64(d70.Imm.Int())))}
			} else if d70.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d67.Reg)
				ctx.EmitMovRegReg(r45, d67.Reg)
				ctx.EmitShrRegImm8(r45, uint8(d70.Imm.Int()))
				d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d71)
			} else {
				{
					shiftSrc := d67.Reg
					r46 := ctx.AllocRegExcept(d67.Reg)
					ctx.EmitMovRegReg(r46, d67.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d70.Reg != scm.RegRCX
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d70.Reg != scm.RegRCX {
						ctx.EmitMovRegReg(scm.RegRCX, d70.Reg)
					}
					ctx.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d71 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d71)
				}
			}
			if d71.Loc == scm.LocReg && d67.Loc == scm.LocReg && d71.Reg == d67.Reg {
				ctx.TransferReg(d67.Reg)
				d67.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d67)
			ctx.FreeDesc(&d70)
			ctx.EnsureDesc(&d47)
			ctx.EnsureDesc(&d71)
			var d72 scm.JITValueDesc
			if d47.Loc == scm.LocImm && d71.Loc == scm.LocImm {
				d72 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d47.Imm.Int() | d71.Imm.Int())}
			} else if d47.Loc == scm.LocImm && d47.Imm.Int() == 0 {
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d71.Reg}
				ctx.BindReg(d71.Reg, &d72)
			} else if d71.Loc == scm.LocImm && d71.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r47, d47.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d72)
			} else if d47.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d71.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
				ctx.EmitOrInt64(scratch, d71.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d72)
			} else if d71.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d47.Reg)
				ctx.EmitMovRegReg(r48, d47.Reg)
				if d71.Imm.Int() >= -2147483648 && d71.Imm.Int() <= 2147483647 {
					ctx.EmitOrRegImm32(r48, int32(d71.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
					ctx.EmitOrInt64(r48, scm.RegR11)
				}
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d72)
			} else {
				r49 := ctx.AllocRegExcept(d47.Reg, d71.Reg)
				ctx.EmitMovRegReg(r49, d47.Reg)
				ctx.EmitOrInt64(r49, d71.Reg)
				d72 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d72)
			}
			if d72.Loc == scm.LocReg && d47.Loc == scm.LocReg && d72.Reg == d47.Reg {
				ctx.TransferReg(d47.Reg)
				d47.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d71)
			ctx.EnsureDesc(&d72)
			if d72.Loc == scm.LocReg {
				ctx.ProtectReg(d72.Reg)
			} else if d72.Loc == scm.LocRegPair {
				ctx.ProtectReg(d72.Reg)
				ctx.ProtectReg(d72.Reg2)
			}
			d73 = d72
			if d73.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d73)
			ctx.EmitStoreToStack(d73, int32(bbs[2].PhiBase)+int32(0))
			if d72.Loc == scm.LocReg {
				ctx.UnprotectReg(d72.Reg)
			} else if d72.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d72.Reg)
				ctx.UnprotectReg(d72.Reg2)
			}
			ctx.EmitJmp(lbl13)
			ctx.MarkLabel(lbl11)
			d74 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d74)
			ctx.BindReg(r32, &d74)
			if r12 { ctx.UnprotectReg(r13) }
			var d75 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r50 := ctx.AllocReg()
				ctx.EmitMovRegMem(r50, thisptr.Reg, off)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d75)
			}
			ctx.EnsureDesc(&d75)
			ctx.EnsureDesc(&d75)
			var d76 scm.JITValueDesc
			if d75.Loc == scm.LocImm {
				d76 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d75.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.EmitMovRegReg(r51, d75.Reg)
				d76 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d76)
			}
			ctx.FreeDesc(&d75)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d76)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d76)
			var d77 scm.JITValueDesc
			if d74.Loc == scm.LocImm && d76.Loc == scm.LocImm {
				d77 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d74.Imm.Int() + d76.Imm.Int())}
			} else if d76.Loc == scm.LocImm && d76.Imm.Int() == 0 {
				r52 := ctx.AllocRegExcept(d74.Reg)
				ctx.EmitMovRegReg(r52, d74.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d77)
			} else if d74.Loc == scm.LocImm && d74.Imm.Int() == 0 {
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d76.Reg}
				ctx.BindReg(d76.Reg, &d77)
			} else if d74.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d76.Reg)
				ctx.EmitMovRegImm64(scratch, uint64(d74.Imm.Int()))
				ctx.EmitAddInt64(scratch, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else if d76.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d74.Reg)
				ctx.EmitMovRegReg(scratch, d74.Reg)
				if d76.Imm.Int() >= -2147483648 && d76.Imm.Int() <= 2147483647 {
					ctx.EmitAddRegImm32(scratch, int32(d76.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d76.Imm.Int()))
					ctx.EmitAddInt64(scratch, scm.RegR11)
				}
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d77)
			} else {
				r53 := ctx.AllocRegExcept(d74.Reg, d76.Reg)
				ctx.EmitMovRegReg(r53, d74.Reg)
				ctx.EmitAddInt64(r53, d76.Reg)
				d77 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d77)
			}
			if d77.Loc == scm.LocReg && d74.Loc == scm.LocReg && d77.Reg == d74.Reg {
				ctx.TransferReg(d74.Reg)
				d74.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d74)
			ctx.FreeDesc(&d76)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d78 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d78 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.EmitMovRegReg(r54, idxInt.Reg)
				ctx.EmitShlRegImm8(r54, 32)
				ctx.EmitShrRegImm8(r54, 32)
				d78 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d78)
			}
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d78)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d78)
			var d79 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d78.Loc == scm.LocImm {
				d79 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d77.Imm.Int()) == uint64(d78.Imm.Int()))}
			} else if d78.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d77.Reg)
				if d78.Imm.Int() >= -2147483648 && d78.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d77.Reg, int32(d78.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d78.Imm.Int()))
					ctx.EmitCmpInt64(d77.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r55, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d79)
			} else if d77.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d78.Reg)
				ctx.EmitSetcc(r56, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d79)
			} else {
				r57 := ctx.AllocRegExcept(d77.Reg)
				ctx.EmitCmpInt64(d77.Reg, d78.Reg)
				ctx.EmitSetcc(r57, scm.CcE)
				d79 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d79)
			}
			ctx.FreeDesc(&d78)
			d80 = d79
			ctx.EnsureDesc(&d80)
			if d80.Loc != scm.LocImm && d80.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d80.Loc == scm.LocImm {
				if d80.Imm.Bool() {
			ps81 := scm.PhiState{General: ps.General}
			ps81.OverlayValues = make([]scm.JITValueDesc, 81)
			ps81.OverlayValues[0] = d0
			ps81.OverlayValues[1] = d1
			ps81.OverlayValues[2] = d2
			ps81.OverlayValues[3] = d3
			ps81.OverlayValues[4] = d4
			ps81.OverlayValues[5] = d5
			ps81.OverlayValues[7] = d7
			ps81.OverlayValues[8] = d8
			ps81.OverlayValues[9] = d9
			ps81.OverlayValues[10] = d10
			ps81.OverlayValues[11] = d11
			ps81.OverlayValues[12] = d12
			ps81.OverlayValues[15] = d15
			ps81.OverlayValues[16] = d16
			ps81.OverlayValues[34] = d34
			ps81.OverlayValues[35] = d35
			ps81.OverlayValues[36] = d36
			ps81.OverlayValues[37] = d37
			ps81.OverlayValues[38] = d38
			ps81.OverlayValues[39] = d39
			ps81.OverlayValues[40] = d40
			ps81.OverlayValues[41] = d41
			ps81.OverlayValues[42] = d42
			ps81.OverlayValues[43] = d43
			ps81.OverlayValues[44] = d44
			ps81.OverlayValues[45] = d45
			ps81.OverlayValues[46] = d46
			ps81.OverlayValues[47] = d47
			ps81.OverlayValues[48] = d48
			ps81.OverlayValues[49] = d49
			ps81.OverlayValues[50] = d50
			ps81.OverlayValues[51] = d51
			ps81.OverlayValues[52] = d52
			ps81.OverlayValues[53] = d53
			ps81.OverlayValues[54] = d54
			ps81.OverlayValues[55] = d55
			ps81.OverlayValues[56] = d56
			ps81.OverlayValues[57] = d57
			ps81.OverlayValues[58] = d58
			ps81.OverlayValues[59] = d59
			ps81.OverlayValues[60] = d60
			ps81.OverlayValues[61] = d61
			ps81.OverlayValues[62] = d62
			ps81.OverlayValues[63] = d63
			ps81.OverlayValues[64] = d64
			ps81.OverlayValues[65] = d65
			ps81.OverlayValues[66] = d66
			ps81.OverlayValues[67] = d67
			ps81.OverlayValues[68] = d68
			ps81.OverlayValues[69] = d69
			ps81.OverlayValues[70] = d70
			ps81.OverlayValues[71] = d71
			ps81.OverlayValues[72] = d72
			ps81.OverlayValues[73] = d73
			ps81.OverlayValues[74] = d74
			ps81.OverlayValues[75] = d75
			ps81.OverlayValues[76] = d76
			ps81.OverlayValues[77] = d77
			ps81.OverlayValues[78] = d78
			ps81.OverlayValues[79] = d79
			ps81.OverlayValues[80] = d80
					return bbs[4].RenderPS(ps81)
				}
			ps82 := scm.PhiState{General: ps.General}
			ps82.OverlayValues = make([]scm.JITValueDesc, 81)
			ps82.OverlayValues[0] = d0
			ps82.OverlayValues[1] = d1
			ps82.OverlayValues[2] = d2
			ps82.OverlayValues[3] = d3
			ps82.OverlayValues[4] = d4
			ps82.OverlayValues[5] = d5
			ps82.OverlayValues[7] = d7
			ps82.OverlayValues[8] = d8
			ps82.OverlayValues[9] = d9
			ps82.OverlayValues[10] = d10
			ps82.OverlayValues[11] = d11
			ps82.OverlayValues[12] = d12
			ps82.OverlayValues[15] = d15
			ps82.OverlayValues[16] = d16
			ps82.OverlayValues[34] = d34
			ps82.OverlayValues[35] = d35
			ps82.OverlayValues[36] = d36
			ps82.OverlayValues[37] = d37
			ps82.OverlayValues[38] = d38
			ps82.OverlayValues[39] = d39
			ps82.OverlayValues[40] = d40
			ps82.OverlayValues[41] = d41
			ps82.OverlayValues[42] = d42
			ps82.OverlayValues[43] = d43
			ps82.OverlayValues[44] = d44
			ps82.OverlayValues[45] = d45
			ps82.OverlayValues[46] = d46
			ps82.OverlayValues[47] = d47
			ps82.OverlayValues[48] = d48
			ps82.OverlayValues[49] = d49
			ps82.OverlayValues[50] = d50
			ps82.OverlayValues[51] = d51
			ps82.OverlayValues[52] = d52
			ps82.OverlayValues[53] = d53
			ps82.OverlayValues[54] = d54
			ps82.OverlayValues[55] = d55
			ps82.OverlayValues[56] = d56
			ps82.OverlayValues[57] = d57
			ps82.OverlayValues[58] = d58
			ps82.OverlayValues[59] = d59
			ps82.OverlayValues[60] = d60
			ps82.OverlayValues[61] = d61
			ps82.OverlayValues[62] = d62
			ps82.OverlayValues[63] = d63
			ps82.OverlayValues[64] = d64
			ps82.OverlayValues[65] = d65
			ps82.OverlayValues[66] = d66
			ps82.OverlayValues[67] = d67
			ps82.OverlayValues[68] = d68
			ps82.OverlayValues[69] = d69
			ps82.OverlayValues[70] = d70
			ps82.OverlayValues[71] = d71
			ps82.OverlayValues[72] = d72
			ps82.OverlayValues[73] = d73
			ps82.OverlayValues[74] = d74
			ps82.OverlayValues[75] = d75
			ps82.OverlayValues[76] = d76
			ps82.OverlayValues[77] = d77
			ps82.OverlayValues[78] = d78
			ps82.OverlayValues[79] = d79
			ps82.OverlayValues[80] = d80
				return bbs[5].RenderPS(ps82)
			}
			if !ps.General {
				ps.General = true
				return bbs[3].RenderPS(ps)
			}
			lbl19 := ctx.ReserveLabel()
			lbl20 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d80.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl19)
			ctx.EmitJmp(lbl20)
			ctx.MarkLabel(lbl19)
			ctx.EmitJmp(lbl5)
			ctx.MarkLabel(lbl20)
			ctx.EmitJmp(lbl6)
			ps83 := scm.PhiState{General: true}
			ps83.OverlayValues = make([]scm.JITValueDesc, 81)
			ps83.OverlayValues[0] = d0
			ps83.OverlayValues[1] = d1
			ps83.OverlayValues[2] = d2
			ps83.OverlayValues[3] = d3
			ps83.OverlayValues[4] = d4
			ps83.OverlayValues[5] = d5
			ps83.OverlayValues[7] = d7
			ps83.OverlayValues[8] = d8
			ps83.OverlayValues[9] = d9
			ps83.OverlayValues[10] = d10
			ps83.OverlayValues[11] = d11
			ps83.OverlayValues[12] = d12
			ps83.OverlayValues[15] = d15
			ps83.OverlayValues[16] = d16
			ps83.OverlayValues[34] = d34
			ps83.OverlayValues[35] = d35
			ps83.OverlayValues[36] = d36
			ps83.OverlayValues[37] = d37
			ps83.OverlayValues[38] = d38
			ps83.OverlayValues[39] = d39
			ps83.OverlayValues[40] = d40
			ps83.OverlayValues[41] = d41
			ps83.OverlayValues[42] = d42
			ps83.OverlayValues[43] = d43
			ps83.OverlayValues[44] = d44
			ps83.OverlayValues[45] = d45
			ps83.OverlayValues[46] = d46
			ps83.OverlayValues[47] = d47
			ps83.OverlayValues[48] = d48
			ps83.OverlayValues[49] = d49
			ps83.OverlayValues[50] = d50
			ps83.OverlayValues[51] = d51
			ps83.OverlayValues[52] = d52
			ps83.OverlayValues[53] = d53
			ps83.OverlayValues[54] = d54
			ps83.OverlayValues[55] = d55
			ps83.OverlayValues[56] = d56
			ps83.OverlayValues[57] = d57
			ps83.OverlayValues[58] = d58
			ps83.OverlayValues[59] = d59
			ps83.OverlayValues[60] = d60
			ps83.OverlayValues[61] = d61
			ps83.OverlayValues[62] = d62
			ps83.OverlayValues[63] = d63
			ps83.OverlayValues[64] = d64
			ps83.OverlayValues[65] = d65
			ps83.OverlayValues[66] = d66
			ps83.OverlayValues[67] = d67
			ps83.OverlayValues[68] = d68
			ps83.OverlayValues[69] = d69
			ps83.OverlayValues[70] = d70
			ps83.OverlayValues[71] = d71
			ps83.OverlayValues[72] = d72
			ps83.OverlayValues[73] = d73
			ps83.OverlayValues[74] = d74
			ps83.OverlayValues[75] = d75
			ps83.OverlayValues[76] = d76
			ps83.OverlayValues[77] = d77
			ps83.OverlayValues[78] = d78
			ps83.OverlayValues[79] = d79
			ps83.OverlayValues[80] = d80
			ps84 := scm.PhiState{General: true}
			ps84.OverlayValues = make([]scm.JITValueDesc, 81)
			ps84.OverlayValues[0] = d0
			ps84.OverlayValues[1] = d1
			ps84.OverlayValues[2] = d2
			ps84.OverlayValues[3] = d3
			ps84.OverlayValues[4] = d4
			ps84.OverlayValues[5] = d5
			ps84.OverlayValues[7] = d7
			ps84.OverlayValues[8] = d8
			ps84.OverlayValues[9] = d9
			ps84.OverlayValues[10] = d10
			ps84.OverlayValues[11] = d11
			ps84.OverlayValues[12] = d12
			ps84.OverlayValues[15] = d15
			ps84.OverlayValues[16] = d16
			ps84.OverlayValues[34] = d34
			ps84.OverlayValues[35] = d35
			ps84.OverlayValues[36] = d36
			ps84.OverlayValues[37] = d37
			ps84.OverlayValues[38] = d38
			ps84.OverlayValues[39] = d39
			ps84.OverlayValues[40] = d40
			ps84.OverlayValues[41] = d41
			ps84.OverlayValues[42] = d42
			ps84.OverlayValues[43] = d43
			ps84.OverlayValues[44] = d44
			ps84.OverlayValues[45] = d45
			ps84.OverlayValues[46] = d46
			ps84.OverlayValues[47] = d47
			ps84.OverlayValues[48] = d48
			ps84.OverlayValues[49] = d49
			ps84.OverlayValues[50] = d50
			ps84.OverlayValues[51] = d51
			ps84.OverlayValues[52] = d52
			ps84.OverlayValues[53] = d53
			ps84.OverlayValues[54] = d54
			ps84.OverlayValues[55] = d55
			ps84.OverlayValues[56] = d56
			ps84.OverlayValues[57] = d57
			ps84.OverlayValues[58] = d58
			ps84.OverlayValues[59] = d59
			ps84.OverlayValues[60] = d60
			ps84.OverlayValues[61] = d61
			ps84.OverlayValues[62] = d62
			ps84.OverlayValues[63] = d63
			ps84.OverlayValues[64] = d64
			ps84.OverlayValues[65] = d65
			ps84.OverlayValues[66] = d66
			ps84.OverlayValues[67] = d67
			ps84.OverlayValues[68] = d68
			ps84.OverlayValues[69] = d69
			ps84.OverlayValues[70] = d70
			ps84.OverlayValues[71] = d71
			ps84.OverlayValues[72] = d72
			ps84.OverlayValues[73] = d73
			ps84.OverlayValues[74] = d74
			ps84.OverlayValues[75] = d75
			ps84.OverlayValues[76] = d76
			ps84.OverlayValues[77] = d77
			ps84.OverlayValues[78] = d78
			ps84.OverlayValues[79] = d79
			ps84.OverlayValues[80] = d80
			snap85 := d0
			snap86 := d1
			snap87 := d2
			snap88 := d3
			snap89 := d4
			snap90 := d5
			snap91 := d7
			snap92 := d8
			snap93 := d9
			snap94 := d10
			snap95 := d11
			snap96 := d12
			snap97 := d15
			snap98 := d16
			snap99 := d34
			snap100 := d35
			snap101 := d36
			snap102 := d37
			snap103 := d38
			snap104 := d39
			snap105 := d40
			snap106 := d41
			snap107 := d42
			snap108 := d43
			snap109 := d44
			snap110 := d45
			snap111 := d46
			snap112 := d47
			snap113 := d48
			snap114 := d49
			snap115 := d50
			snap116 := d51
			snap117 := d52
			snap118 := d53
			snap119 := d54
			snap120 := d55
			snap121 := d56
			snap122 := d57
			snap123 := d58
			snap124 := d59
			snap125 := d60
			snap126 := d61
			snap127 := d62
			snap128 := d63
			snap129 := d64
			snap130 := d65
			snap131 := d66
			snap132 := d67
			snap133 := d68
			snap134 := d69
			snap135 := d70
			snap136 := d71
			snap137 := d72
			snap138 := d73
			snap139 := d74
			snap140 := d75
			snap141 := d76
			snap142 := d77
			snap143 := d78
			snap144 := d79
			snap145 := d80
			alloc146 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps84)
			}
			ctx.RestoreAllocState(alloc146)
			d0 = snap85
			d1 = snap86
			d2 = snap87
			d3 = snap88
			d4 = snap89
			d5 = snap90
			d7 = snap91
			d8 = snap92
			d9 = snap93
			d10 = snap94
			d11 = snap95
			d12 = snap96
			d15 = snap97
			d16 = snap98
			d34 = snap99
			d35 = snap100
			d36 = snap101
			d37 = snap102
			d38 = snap103
			d39 = snap104
			d40 = snap105
			d41 = snap106
			d42 = snap107
			d43 = snap108
			d44 = snap109
			d45 = snap110
			d46 = snap111
			d47 = snap112
			d48 = snap113
			d49 = snap114
			d50 = snap115
			d51 = snap116
			d52 = snap117
			d53 = snap118
			d54 = snap119
			d55 = snap120
			d56 = snap121
			d57 = snap122
			d58 = snap123
			d59 = snap124
			d60 = snap125
			d61 = snap126
			d62 = snap127
			d63 = snap128
			d64 = snap129
			d65 = snap130
			d66 = snap131
			d67 = snap132
			d68 = snap133
			d69 = snap134
			d70 = snap135
			d71 = snap136
			d72 = snap137
			d73 = snap138
			d74 = snap139
			d75 = snap140
			d76 = snap141
			d77 = snap142
			d78 = snap143
			d79 = snap144
			d80 = snap145
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps83)
			}
			return result
			ctx.FreeDesc(&d79)
			return result
			}
			bbs[4].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[4].VisitCount >= 2 {
					ps.General = true
					return bbs[4].RenderPS(ps)
				}
			}
			bbs[4].VisitCount++
			if ps.General {
				if bbs[4].Rendered {
					ctx.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.MarkLabel(lbl5)
				ctx.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			ctx.ReclaimUntrackedRegs()
			var d147 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r58 := ctx.AllocReg()
				r59 := ctx.AllocReg()
				ctx.EmitMovRegMem64(r58, fieldAddr)
				ctx.EmitMovRegMem64(r59, fieldAddr+8)
				d147 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r58, Reg2: r59}
				ctx.BindReg(r58, &d147)
				ctx.BindReg(r59, &d147)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r60 := ctx.AllocReg()
				r61 := ctx.AllocReg()
				ctx.EmitMovRegMem(r60, thisptr.Reg, off)
				ctx.EmitMovRegMem(r61, thisptr.Reg, off+8)
				d147 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r60, Reg2: r61}
				ctx.BindReg(r60, &d147)
				ctx.BindReg(r61, &d147)
			}
			ctx.EnsureDesc(&d36)
			r62 := ctx.AllocReg()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d147)
			if d36.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(r62, uint64(d36.Imm.Int()) * 16)
			} else {
				ctx.EmitMovRegReg(r62, d36.Reg)
				ctx.EmitShlRegImm8(r62, 4)
			}
			if d147.Loc == scm.LocImm {
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d147.Imm.Int()))
				ctx.EmitAddInt64(r62, scm.RegR11)
			} else {
				ctx.EmitAddInt64(r62, d147.Reg)
			}
			r63 := ctx.AllocRegExcept(r62)
			r64 := ctx.AllocRegExcept(r62, r63)
			ctx.EmitMovRegMem(r63, r62, 0)
			ctx.EmitMovRegMem(r64, r62, 8)
			ctx.FreeReg(r62)
			d148 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r63, Reg2: r64}
			ctx.BindReg(r63, &d148)
			ctx.BindReg(r64, &d148)
			d149 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d149)
			ctx.BindReg(r2, &d149)
			ctx.EnsureDesc(&d148)
			if d148.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d148, &d149)
			} else {
				switch d148.Type {
				case scm.TagBool:
					ctx.EmitMakeBool(d149, d148)
				case scm.TagInt:
					ctx.EmitMakeInt(d149, d148)
				case scm.TagFloat:
					ctx.EmitMakeFloat(d149, d148)
				case scm.TagNil:
					ctx.EmitMakeNil(d149)
				default:
					ctx.EmitMovPairToResult(&d148, &d149)
				}
			}
			ctx.EmitJmp(lbl0)
			return result
			}
			bbs[5].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[5].VisitCount >= 2 {
					ps.General = true
					return bbs[5].RenderPS(ps)
				}
			}
			bbs[5].VisitCount++
			if ps.General {
				if bbs[5].Rendered {
					ctx.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.MarkLabel(lbl6)
				ctx.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d150 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d150 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.EmitMovRegReg(r65, idxInt.Reg)
				ctx.EmitShlRegImm8(r65, 32)
				ctx.EmitShrRegImm8(r65, 32)
				d150 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d150)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d150)
			ctx.EnsureDesc(&d77)
			ctx.EnsureDesc(&d150)
			var d151 scm.JITValueDesc
			if d77.Loc == scm.LocImm && d150.Loc == scm.LocImm {
				d151 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d77.Imm.Int()) < uint64(d150.Imm.Int()))}
			} else if d150.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d77.Reg)
				if d150.Imm.Int() >= -2147483648 && d150.Imm.Int() <= 2147483647 {
					ctx.EmitCmpRegImm32(d77.Reg, int32(d150.Imm.Int()))
				} else {
					ctx.EmitMovRegImm64(scm.RegR11, uint64(d150.Imm.Int()))
					ctx.EmitCmpInt64(d77.Reg, scm.RegR11)
				}
				ctx.EmitSetcc(r66, scm.CcB)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r66}
				ctx.BindReg(r66, &d151)
			} else if d77.Loc == scm.LocImm {
				r67 := ctx.AllocReg()
				ctx.EmitMovRegImm64(scm.RegR11, uint64(d77.Imm.Int()))
				ctx.EmitCmpInt64(scm.RegR11, d150.Reg)
				ctx.EmitSetcc(r67, scm.CcB)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r67}
				ctx.BindReg(r67, &d151)
			} else {
				r68 := ctx.AllocRegExcept(d77.Reg)
				ctx.EmitCmpInt64(d77.Reg, d150.Reg)
				ctx.EmitSetcc(r68, scm.CcB)
				d151 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
				ctx.BindReg(r68, &d151)
			}
			ctx.FreeDesc(&d77)
			ctx.FreeDesc(&d150)
			d152 = d151
			ctx.EnsureDesc(&d152)
			if d152.Loc != scm.LocImm && d152.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d152.Loc == scm.LocImm {
				if d152.Imm.Bool() {
			ps153 := scm.PhiState{General: ps.General}
			ps153.OverlayValues = make([]scm.JITValueDesc, 153)
			ps153.OverlayValues[0] = d0
			ps153.OverlayValues[1] = d1
			ps153.OverlayValues[2] = d2
			ps153.OverlayValues[3] = d3
			ps153.OverlayValues[4] = d4
			ps153.OverlayValues[5] = d5
			ps153.OverlayValues[7] = d7
			ps153.OverlayValues[8] = d8
			ps153.OverlayValues[9] = d9
			ps153.OverlayValues[10] = d10
			ps153.OverlayValues[11] = d11
			ps153.OverlayValues[12] = d12
			ps153.OverlayValues[15] = d15
			ps153.OverlayValues[16] = d16
			ps153.OverlayValues[34] = d34
			ps153.OverlayValues[35] = d35
			ps153.OverlayValues[36] = d36
			ps153.OverlayValues[37] = d37
			ps153.OverlayValues[38] = d38
			ps153.OverlayValues[39] = d39
			ps153.OverlayValues[40] = d40
			ps153.OverlayValues[41] = d41
			ps153.OverlayValues[42] = d42
			ps153.OverlayValues[43] = d43
			ps153.OverlayValues[44] = d44
			ps153.OverlayValues[45] = d45
			ps153.OverlayValues[46] = d46
			ps153.OverlayValues[47] = d47
			ps153.OverlayValues[48] = d48
			ps153.OverlayValues[49] = d49
			ps153.OverlayValues[50] = d50
			ps153.OverlayValues[51] = d51
			ps153.OverlayValues[52] = d52
			ps153.OverlayValues[53] = d53
			ps153.OverlayValues[54] = d54
			ps153.OverlayValues[55] = d55
			ps153.OverlayValues[56] = d56
			ps153.OverlayValues[57] = d57
			ps153.OverlayValues[58] = d58
			ps153.OverlayValues[59] = d59
			ps153.OverlayValues[60] = d60
			ps153.OverlayValues[61] = d61
			ps153.OverlayValues[62] = d62
			ps153.OverlayValues[63] = d63
			ps153.OverlayValues[64] = d64
			ps153.OverlayValues[65] = d65
			ps153.OverlayValues[66] = d66
			ps153.OverlayValues[67] = d67
			ps153.OverlayValues[68] = d68
			ps153.OverlayValues[69] = d69
			ps153.OverlayValues[70] = d70
			ps153.OverlayValues[71] = d71
			ps153.OverlayValues[72] = d72
			ps153.OverlayValues[73] = d73
			ps153.OverlayValues[74] = d74
			ps153.OverlayValues[75] = d75
			ps153.OverlayValues[76] = d76
			ps153.OverlayValues[77] = d77
			ps153.OverlayValues[78] = d78
			ps153.OverlayValues[79] = d79
			ps153.OverlayValues[80] = d80
			ps153.OverlayValues[147] = d147
			ps153.OverlayValues[148] = d148
			ps153.OverlayValues[149] = d149
			ps153.OverlayValues[150] = d150
			ps153.OverlayValues[151] = d151
			ps153.OverlayValues[152] = d152
					return bbs[6].RenderPS(ps153)
				}
			ps154 := scm.PhiState{General: ps.General}
			ps154.OverlayValues = make([]scm.JITValueDesc, 153)
			ps154.OverlayValues[0] = d0
			ps154.OverlayValues[1] = d1
			ps154.OverlayValues[2] = d2
			ps154.OverlayValues[3] = d3
			ps154.OverlayValues[4] = d4
			ps154.OverlayValues[5] = d5
			ps154.OverlayValues[7] = d7
			ps154.OverlayValues[8] = d8
			ps154.OverlayValues[9] = d9
			ps154.OverlayValues[10] = d10
			ps154.OverlayValues[11] = d11
			ps154.OverlayValues[12] = d12
			ps154.OverlayValues[15] = d15
			ps154.OverlayValues[16] = d16
			ps154.OverlayValues[34] = d34
			ps154.OverlayValues[35] = d35
			ps154.OverlayValues[36] = d36
			ps154.OverlayValues[37] = d37
			ps154.OverlayValues[38] = d38
			ps154.OverlayValues[39] = d39
			ps154.OverlayValues[40] = d40
			ps154.OverlayValues[41] = d41
			ps154.OverlayValues[42] = d42
			ps154.OverlayValues[43] = d43
			ps154.OverlayValues[44] = d44
			ps154.OverlayValues[45] = d45
			ps154.OverlayValues[46] = d46
			ps154.OverlayValues[47] = d47
			ps154.OverlayValues[48] = d48
			ps154.OverlayValues[49] = d49
			ps154.OverlayValues[50] = d50
			ps154.OverlayValues[51] = d51
			ps154.OverlayValues[52] = d52
			ps154.OverlayValues[53] = d53
			ps154.OverlayValues[54] = d54
			ps154.OverlayValues[55] = d55
			ps154.OverlayValues[56] = d56
			ps154.OverlayValues[57] = d57
			ps154.OverlayValues[58] = d58
			ps154.OverlayValues[59] = d59
			ps154.OverlayValues[60] = d60
			ps154.OverlayValues[61] = d61
			ps154.OverlayValues[62] = d62
			ps154.OverlayValues[63] = d63
			ps154.OverlayValues[64] = d64
			ps154.OverlayValues[65] = d65
			ps154.OverlayValues[66] = d66
			ps154.OverlayValues[67] = d67
			ps154.OverlayValues[68] = d68
			ps154.OverlayValues[69] = d69
			ps154.OverlayValues[70] = d70
			ps154.OverlayValues[71] = d71
			ps154.OverlayValues[72] = d72
			ps154.OverlayValues[73] = d73
			ps154.OverlayValues[74] = d74
			ps154.OverlayValues[75] = d75
			ps154.OverlayValues[76] = d76
			ps154.OverlayValues[77] = d77
			ps154.OverlayValues[78] = d78
			ps154.OverlayValues[79] = d79
			ps154.OverlayValues[80] = d80
			ps154.OverlayValues[147] = d147
			ps154.OverlayValues[148] = d148
			ps154.OverlayValues[149] = d149
			ps154.OverlayValues[150] = d150
			ps154.OverlayValues[151] = d151
			ps154.OverlayValues[152] = d152
				return bbs[7].RenderPS(ps154)
			}
			if !ps.General {
				ps.General = true
				return bbs[5].RenderPS(ps)
			}
			lbl21 := ctx.ReserveLabel()
			lbl22 := ctx.ReserveLabel()
			ctx.EmitCmpRegImm32(d152.Reg, 0)
			ctx.EmitJcc(scm.CcNE, lbl21)
			ctx.EmitJmp(lbl22)
			ctx.MarkLabel(lbl21)
			ctx.EmitJmp(lbl7)
			ctx.MarkLabel(lbl22)
			ctx.EmitJmp(lbl8)
			ps155 := scm.PhiState{General: true}
			ps155.OverlayValues = make([]scm.JITValueDesc, 153)
			ps155.OverlayValues[0] = d0
			ps155.OverlayValues[1] = d1
			ps155.OverlayValues[2] = d2
			ps155.OverlayValues[3] = d3
			ps155.OverlayValues[4] = d4
			ps155.OverlayValues[5] = d5
			ps155.OverlayValues[7] = d7
			ps155.OverlayValues[8] = d8
			ps155.OverlayValues[9] = d9
			ps155.OverlayValues[10] = d10
			ps155.OverlayValues[11] = d11
			ps155.OverlayValues[12] = d12
			ps155.OverlayValues[15] = d15
			ps155.OverlayValues[16] = d16
			ps155.OverlayValues[34] = d34
			ps155.OverlayValues[35] = d35
			ps155.OverlayValues[36] = d36
			ps155.OverlayValues[37] = d37
			ps155.OverlayValues[38] = d38
			ps155.OverlayValues[39] = d39
			ps155.OverlayValues[40] = d40
			ps155.OverlayValues[41] = d41
			ps155.OverlayValues[42] = d42
			ps155.OverlayValues[43] = d43
			ps155.OverlayValues[44] = d44
			ps155.OverlayValues[45] = d45
			ps155.OverlayValues[46] = d46
			ps155.OverlayValues[47] = d47
			ps155.OverlayValues[48] = d48
			ps155.OverlayValues[49] = d49
			ps155.OverlayValues[50] = d50
			ps155.OverlayValues[51] = d51
			ps155.OverlayValues[52] = d52
			ps155.OverlayValues[53] = d53
			ps155.OverlayValues[54] = d54
			ps155.OverlayValues[55] = d55
			ps155.OverlayValues[56] = d56
			ps155.OverlayValues[57] = d57
			ps155.OverlayValues[58] = d58
			ps155.OverlayValues[59] = d59
			ps155.OverlayValues[60] = d60
			ps155.OverlayValues[61] = d61
			ps155.OverlayValues[62] = d62
			ps155.OverlayValues[63] = d63
			ps155.OverlayValues[64] = d64
			ps155.OverlayValues[65] = d65
			ps155.OverlayValues[66] = d66
			ps155.OverlayValues[67] = d67
			ps155.OverlayValues[68] = d68
			ps155.OverlayValues[69] = d69
			ps155.OverlayValues[70] = d70
			ps155.OverlayValues[71] = d71
			ps155.OverlayValues[72] = d72
			ps155.OverlayValues[73] = d73
			ps155.OverlayValues[74] = d74
			ps155.OverlayValues[75] = d75
			ps155.OverlayValues[76] = d76
			ps155.OverlayValues[77] = d77
			ps155.OverlayValues[78] = d78
			ps155.OverlayValues[79] = d79
			ps155.OverlayValues[80] = d80
			ps155.OverlayValues[147] = d147
			ps155.OverlayValues[148] = d148
			ps155.OverlayValues[149] = d149
			ps155.OverlayValues[150] = d150
			ps155.OverlayValues[151] = d151
			ps155.OverlayValues[152] = d152
			ps156 := scm.PhiState{General: true}
			ps156.OverlayValues = make([]scm.JITValueDesc, 153)
			ps156.OverlayValues[0] = d0
			ps156.OverlayValues[1] = d1
			ps156.OverlayValues[2] = d2
			ps156.OverlayValues[3] = d3
			ps156.OverlayValues[4] = d4
			ps156.OverlayValues[5] = d5
			ps156.OverlayValues[7] = d7
			ps156.OverlayValues[8] = d8
			ps156.OverlayValues[9] = d9
			ps156.OverlayValues[10] = d10
			ps156.OverlayValues[11] = d11
			ps156.OverlayValues[12] = d12
			ps156.OverlayValues[15] = d15
			ps156.OverlayValues[16] = d16
			ps156.OverlayValues[34] = d34
			ps156.OverlayValues[35] = d35
			ps156.OverlayValues[36] = d36
			ps156.OverlayValues[37] = d37
			ps156.OverlayValues[38] = d38
			ps156.OverlayValues[39] = d39
			ps156.OverlayValues[40] = d40
			ps156.OverlayValues[41] = d41
			ps156.OverlayValues[42] = d42
			ps156.OverlayValues[43] = d43
			ps156.OverlayValues[44] = d44
			ps156.OverlayValues[45] = d45
			ps156.OverlayValues[46] = d46
			ps156.OverlayValues[47] = d47
			ps156.OverlayValues[48] = d48
			ps156.OverlayValues[49] = d49
			ps156.OverlayValues[50] = d50
			ps156.OverlayValues[51] = d51
			ps156.OverlayValues[52] = d52
			ps156.OverlayValues[53] = d53
			ps156.OverlayValues[54] = d54
			ps156.OverlayValues[55] = d55
			ps156.OverlayValues[56] = d56
			ps156.OverlayValues[57] = d57
			ps156.OverlayValues[58] = d58
			ps156.OverlayValues[59] = d59
			ps156.OverlayValues[60] = d60
			ps156.OverlayValues[61] = d61
			ps156.OverlayValues[62] = d62
			ps156.OverlayValues[63] = d63
			ps156.OverlayValues[64] = d64
			ps156.OverlayValues[65] = d65
			ps156.OverlayValues[66] = d66
			ps156.OverlayValues[67] = d67
			ps156.OverlayValues[68] = d68
			ps156.OverlayValues[69] = d69
			ps156.OverlayValues[70] = d70
			ps156.OverlayValues[71] = d71
			ps156.OverlayValues[72] = d72
			ps156.OverlayValues[73] = d73
			ps156.OverlayValues[74] = d74
			ps156.OverlayValues[75] = d75
			ps156.OverlayValues[76] = d76
			ps156.OverlayValues[77] = d77
			ps156.OverlayValues[78] = d78
			ps156.OverlayValues[79] = d79
			ps156.OverlayValues[80] = d80
			ps156.OverlayValues[147] = d147
			ps156.OverlayValues[148] = d148
			ps156.OverlayValues[149] = d149
			ps156.OverlayValues[150] = d150
			ps156.OverlayValues[151] = d151
			ps156.OverlayValues[152] = d152
			snap157 := d0
			snap158 := d1
			snap159 := d2
			snap160 := d3
			snap161 := d4
			snap162 := d5
			snap163 := d7
			snap164 := d8
			snap165 := d9
			snap166 := d10
			snap167 := d11
			snap168 := d12
			snap169 := d15
			snap170 := d16
			snap171 := d34
			snap172 := d35
			snap173 := d36
			snap174 := d37
			snap175 := d38
			snap176 := d39
			snap177 := d40
			snap178 := d41
			snap179 := d42
			snap180 := d43
			snap181 := d44
			snap182 := d45
			snap183 := d46
			snap184 := d47
			snap185 := d48
			snap186 := d49
			snap187 := d50
			snap188 := d51
			snap189 := d52
			snap190 := d53
			snap191 := d54
			snap192 := d55
			snap193 := d56
			snap194 := d57
			snap195 := d58
			snap196 := d59
			snap197 := d60
			snap198 := d61
			snap199 := d62
			snap200 := d63
			snap201 := d64
			snap202 := d65
			snap203 := d66
			snap204 := d67
			snap205 := d68
			snap206 := d69
			snap207 := d70
			snap208 := d71
			snap209 := d72
			snap210 := d73
			snap211 := d74
			snap212 := d75
			snap213 := d76
			snap214 := d77
			snap215 := d78
			snap216 := d79
			snap217 := d80
			snap218 := d147
			snap219 := d148
			snap220 := d149
			snap221 := d150
			snap222 := d151
			snap223 := d152
			alloc224 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps156)
			}
			ctx.RestoreAllocState(alloc224)
			d0 = snap157
			d1 = snap158
			d2 = snap159
			d3 = snap160
			d4 = snap161
			d5 = snap162
			d7 = snap163
			d8 = snap164
			d9 = snap165
			d10 = snap166
			d11 = snap167
			d12 = snap168
			d15 = snap169
			d16 = snap170
			d34 = snap171
			d35 = snap172
			d36 = snap173
			d37 = snap174
			d38 = snap175
			d39 = snap176
			d40 = snap177
			d41 = snap178
			d42 = snap179
			d43 = snap180
			d44 = snap181
			d45 = snap182
			d46 = snap183
			d47 = snap184
			d48 = snap185
			d49 = snap186
			d50 = snap187
			d51 = snap188
			d52 = snap189
			d53 = snap190
			d54 = snap191
			d55 = snap192
			d56 = snap193
			d57 = snap194
			d58 = snap195
			d59 = snap196
			d60 = snap197
			d61 = snap198
			d62 = snap199
			d63 = snap200
			d64 = snap201
			d65 = snap202
			d66 = snap203
			d67 = snap204
			d68 = snap205
			d69 = snap206
			d70 = snap207
			d71 = snap208
			d72 = snap209
			d73 = snap210
			d74 = snap211
			d75 = snap212
			d76 = snap213
			d77 = snap214
			d78 = snap215
			d79 = snap216
			d80 = snap217
			d147 = snap218
			d148 = snap219
			d149 = snap220
			d150 = snap221
			d151 = snap222
			d152 = snap223
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps155)
			}
			return result
			ctx.FreeDesc(&d151)
			return result
			}
			bbs[6].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[6].VisitCount >= 2 {
					ps.General = true
					return bbs[6].RenderPS(ps)
				}
			}
			bbs[6].VisitCount++
			if ps.General {
				if bbs[6].Rendered {
					ctx.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.MarkLabel(lbl7)
				ctx.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d225 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d36.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d36.Reg)
				ctx.EmitMovRegReg(scratch, d36.Reg)
				ctx.EmitAddRegImm32(scratch, int32(1))
				d225 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d225)
			}
			if d225.Loc == scm.LocImm {
				d225 = scm.JITValueDesc{Loc: scm.LocImm, Type: d225.Type, Imm: scm.NewInt(int64(uint64(d225.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d225.Reg, 32)
				ctx.EmitShrRegImm8(d225.Reg, 32)
			}
			if d225.Loc == scm.LocReg && d36.Loc == scm.LocReg && d225.Reg == d36.Reg {
				ctx.TransferReg(d36.Reg)
				d36.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d1)
			if d1.Loc == scm.LocReg {
				ctx.ProtectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.ProtectReg(d1.Reg)
				ctx.ProtectReg(d1.Reg2)
			}
			ctx.EnsureDesc(&d225)
			if d225.Loc == scm.LocReg {
				ctx.ProtectReg(d225.Reg)
			} else if d225.Loc == scm.LocRegPair {
				ctx.ProtectReg(d225.Reg)
				ctx.ProtectReg(d225.Reg2)
			}
			d226 = d225
			if d226.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d226)
			d227 = d226
			if d227.Loc == scm.LocImm {
				d227 = scm.JITValueDesc{Loc: scm.LocImm, Type: d227.Type, Imm: scm.NewInt(int64(uint64(d227.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d227.Reg, 32)
				ctx.EmitShrRegImm8(d227.Reg, 32)
			}
			ctx.EmitStoreToStack(d227, int32(bbs[1].PhiBase)+int32(0))
			d228 = d1
			if d228.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d228)
			d229 = d228
			if d229.Loc == scm.LocImm {
				d229 = scm.JITValueDesc{Loc: scm.LocImm, Type: d229.Type, Imm: scm.NewInt(int64(uint64(d229.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d229.Reg, 32)
				ctx.EmitShrRegImm8(d229.Reg, 32)
			}
			ctx.EmitStoreToStack(d229, int32(bbs[1].PhiBase)+int32(16))
			if d1.Loc == scm.LocReg {
				ctx.UnprotectReg(d1.Reg)
			} else if d1.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d1.Reg)
				ctx.UnprotectReg(d1.Reg2)
			}
			if d225.Loc == scm.LocReg {
				ctx.UnprotectReg(d225.Reg)
			} else if d225.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d225.Reg)
				ctx.UnprotectReg(d225.Reg2)
			}
			ps230 := scm.PhiState{General: ps.General}
			ps230.OverlayValues = make([]scm.JITValueDesc, 230)
			ps230.OverlayValues[0] = d0
			ps230.OverlayValues[1] = d1
			ps230.OverlayValues[2] = d2
			ps230.OverlayValues[3] = d3
			ps230.OverlayValues[4] = d4
			ps230.OverlayValues[5] = d5
			ps230.OverlayValues[7] = d7
			ps230.OverlayValues[8] = d8
			ps230.OverlayValues[9] = d9
			ps230.OverlayValues[10] = d10
			ps230.OverlayValues[11] = d11
			ps230.OverlayValues[12] = d12
			ps230.OverlayValues[15] = d15
			ps230.OverlayValues[16] = d16
			ps230.OverlayValues[34] = d34
			ps230.OverlayValues[35] = d35
			ps230.OverlayValues[36] = d36
			ps230.OverlayValues[37] = d37
			ps230.OverlayValues[38] = d38
			ps230.OverlayValues[39] = d39
			ps230.OverlayValues[40] = d40
			ps230.OverlayValues[41] = d41
			ps230.OverlayValues[42] = d42
			ps230.OverlayValues[43] = d43
			ps230.OverlayValues[44] = d44
			ps230.OverlayValues[45] = d45
			ps230.OverlayValues[46] = d46
			ps230.OverlayValues[47] = d47
			ps230.OverlayValues[48] = d48
			ps230.OverlayValues[49] = d49
			ps230.OverlayValues[50] = d50
			ps230.OverlayValues[51] = d51
			ps230.OverlayValues[52] = d52
			ps230.OverlayValues[53] = d53
			ps230.OverlayValues[54] = d54
			ps230.OverlayValues[55] = d55
			ps230.OverlayValues[56] = d56
			ps230.OverlayValues[57] = d57
			ps230.OverlayValues[58] = d58
			ps230.OverlayValues[59] = d59
			ps230.OverlayValues[60] = d60
			ps230.OverlayValues[61] = d61
			ps230.OverlayValues[62] = d62
			ps230.OverlayValues[63] = d63
			ps230.OverlayValues[64] = d64
			ps230.OverlayValues[65] = d65
			ps230.OverlayValues[66] = d66
			ps230.OverlayValues[67] = d67
			ps230.OverlayValues[68] = d68
			ps230.OverlayValues[69] = d69
			ps230.OverlayValues[70] = d70
			ps230.OverlayValues[71] = d71
			ps230.OverlayValues[72] = d72
			ps230.OverlayValues[73] = d73
			ps230.OverlayValues[74] = d74
			ps230.OverlayValues[75] = d75
			ps230.OverlayValues[76] = d76
			ps230.OverlayValues[77] = d77
			ps230.OverlayValues[78] = d78
			ps230.OverlayValues[79] = d79
			ps230.OverlayValues[80] = d80
			ps230.OverlayValues[147] = d147
			ps230.OverlayValues[148] = d148
			ps230.OverlayValues[149] = d149
			ps230.OverlayValues[150] = d150
			ps230.OverlayValues[151] = d151
			ps230.OverlayValues[152] = d152
			ps230.OverlayValues[225] = d225
			ps230.OverlayValues[226] = d226
			ps230.OverlayValues[227] = d227
			ps230.OverlayValues[228] = d228
			ps230.OverlayValues[229] = d229
			ps230.PhiValues = make([]scm.JITValueDesc, 2)
			d231 = d225
			ps230.PhiValues[0] = d231
			d232 = d1
			ps230.PhiValues[1] = d232
			if ps230.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps230)
			return result
			}
			bbs[7].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[7].VisitCount >= 2 {
					ps.General = true
					return bbs[7].RenderPS(ps)
				}
			}
			bbs[7].VisitCount++
			if ps.General {
				if bbs[7].Rendered {
					ctx.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.MarkLabel(lbl8)
				ctx.ResolveFixups()
			}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(16)}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.TagInt, StackOff: int32(0)}
			if !ps.General && len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != scm.LocNone {
				d0 = ps.OverlayValues[0]
			}
			if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != scm.LocNone {
				d1 = ps.OverlayValues[1]
			}
			if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != scm.LocNone {
				d2 = ps.OverlayValues[2]
			}
			if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != scm.LocNone {
				d3 = ps.OverlayValues[3]
			}
			if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != scm.LocNone {
				d4 = ps.OverlayValues[4]
			}
			if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != scm.LocNone {
				d5 = ps.OverlayValues[5]
			}
			if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != scm.LocNone {
				d7 = ps.OverlayValues[7]
			}
			if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != scm.LocNone {
				d8 = ps.OverlayValues[8]
			}
			if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != scm.LocNone {
				d9 = ps.OverlayValues[9]
			}
			if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != scm.LocNone {
				d10 = ps.OverlayValues[10]
			}
			if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != scm.LocNone {
				d11 = ps.OverlayValues[11]
			}
			if len(ps.OverlayValues) > 12 && ps.OverlayValues[12].Loc != scm.LocNone {
				d12 = ps.OverlayValues[12]
			}
			if len(ps.OverlayValues) > 15 && ps.OverlayValues[15].Loc != scm.LocNone {
				d15 = ps.OverlayValues[15]
			}
			if len(ps.OverlayValues) > 16 && ps.OverlayValues[16].Loc != scm.LocNone {
				d16 = ps.OverlayValues[16]
			}
			if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != scm.LocNone {
				d34 = ps.OverlayValues[34]
			}
			if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != scm.LocNone {
				d35 = ps.OverlayValues[35]
			}
			if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != scm.LocNone {
				d36 = ps.OverlayValues[36]
			}
			if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != scm.LocNone {
				d37 = ps.OverlayValues[37]
			}
			if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != scm.LocNone {
				d38 = ps.OverlayValues[38]
			}
			if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != scm.LocNone {
				d39 = ps.OverlayValues[39]
			}
			if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != scm.LocNone {
				d40 = ps.OverlayValues[40]
			}
			if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != scm.LocNone {
				d41 = ps.OverlayValues[41]
			}
			if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != scm.LocNone {
				d42 = ps.OverlayValues[42]
			}
			if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != scm.LocNone {
				d43 = ps.OverlayValues[43]
			}
			if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != scm.LocNone {
				d44 = ps.OverlayValues[44]
			}
			if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != scm.LocNone {
				d45 = ps.OverlayValues[45]
			}
			if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != scm.LocNone {
				d46 = ps.OverlayValues[46]
			}
			if len(ps.OverlayValues) > 47 && ps.OverlayValues[47].Loc != scm.LocNone {
				d47 = ps.OverlayValues[47]
			}
			if len(ps.OverlayValues) > 48 && ps.OverlayValues[48].Loc != scm.LocNone {
				d48 = ps.OverlayValues[48]
			}
			if len(ps.OverlayValues) > 49 && ps.OverlayValues[49].Loc != scm.LocNone {
				d49 = ps.OverlayValues[49]
			}
			if len(ps.OverlayValues) > 50 && ps.OverlayValues[50].Loc != scm.LocNone {
				d50 = ps.OverlayValues[50]
			}
			if len(ps.OverlayValues) > 51 && ps.OverlayValues[51].Loc != scm.LocNone {
				d51 = ps.OverlayValues[51]
			}
			if len(ps.OverlayValues) > 52 && ps.OverlayValues[52].Loc != scm.LocNone {
				d52 = ps.OverlayValues[52]
			}
			if len(ps.OverlayValues) > 53 && ps.OverlayValues[53].Loc != scm.LocNone {
				d53 = ps.OverlayValues[53]
			}
			if len(ps.OverlayValues) > 54 && ps.OverlayValues[54].Loc != scm.LocNone {
				d54 = ps.OverlayValues[54]
			}
			if len(ps.OverlayValues) > 55 && ps.OverlayValues[55].Loc != scm.LocNone {
				d55 = ps.OverlayValues[55]
			}
			if len(ps.OverlayValues) > 56 && ps.OverlayValues[56].Loc != scm.LocNone {
				d56 = ps.OverlayValues[56]
			}
			if len(ps.OverlayValues) > 57 && ps.OverlayValues[57].Loc != scm.LocNone {
				d57 = ps.OverlayValues[57]
			}
			if len(ps.OverlayValues) > 58 && ps.OverlayValues[58].Loc != scm.LocNone {
				d58 = ps.OverlayValues[58]
			}
			if len(ps.OverlayValues) > 59 && ps.OverlayValues[59].Loc != scm.LocNone {
				d59 = ps.OverlayValues[59]
			}
			if len(ps.OverlayValues) > 60 && ps.OverlayValues[60].Loc != scm.LocNone {
				d60 = ps.OverlayValues[60]
			}
			if len(ps.OverlayValues) > 61 && ps.OverlayValues[61].Loc != scm.LocNone {
				d61 = ps.OverlayValues[61]
			}
			if len(ps.OverlayValues) > 62 && ps.OverlayValues[62].Loc != scm.LocNone {
				d62 = ps.OverlayValues[62]
			}
			if len(ps.OverlayValues) > 63 && ps.OverlayValues[63].Loc != scm.LocNone {
				d63 = ps.OverlayValues[63]
			}
			if len(ps.OverlayValues) > 64 && ps.OverlayValues[64].Loc != scm.LocNone {
				d64 = ps.OverlayValues[64]
			}
			if len(ps.OverlayValues) > 65 && ps.OverlayValues[65].Loc != scm.LocNone {
				d65 = ps.OverlayValues[65]
			}
			if len(ps.OverlayValues) > 66 && ps.OverlayValues[66].Loc != scm.LocNone {
				d66 = ps.OverlayValues[66]
			}
			if len(ps.OverlayValues) > 67 && ps.OverlayValues[67].Loc != scm.LocNone {
				d67 = ps.OverlayValues[67]
			}
			if len(ps.OverlayValues) > 68 && ps.OverlayValues[68].Loc != scm.LocNone {
				d68 = ps.OverlayValues[68]
			}
			if len(ps.OverlayValues) > 69 && ps.OverlayValues[69].Loc != scm.LocNone {
				d69 = ps.OverlayValues[69]
			}
			if len(ps.OverlayValues) > 70 && ps.OverlayValues[70].Loc != scm.LocNone {
				d70 = ps.OverlayValues[70]
			}
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != scm.LocNone {
				d74 = ps.OverlayValues[74]
			}
			if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != scm.LocNone {
				d75 = ps.OverlayValues[75]
			}
			if len(ps.OverlayValues) > 76 && ps.OverlayValues[76].Loc != scm.LocNone {
				d76 = ps.OverlayValues[76]
			}
			if len(ps.OverlayValues) > 77 && ps.OverlayValues[77].Loc != scm.LocNone {
				d77 = ps.OverlayValues[77]
			}
			if len(ps.OverlayValues) > 78 && ps.OverlayValues[78].Loc != scm.LocNone {
				d78 = ps.OverlayValues[78]
			}
			if len(ps.OverlayValues) > 79 && ps.OverlayValues[79].Loc != scm.LocNone {
				d79 = ps.OverlayValues[79]
			}
			if len(ps.OverlayValues) > 80 && ps.OverlayValues[80].Loc != scm.LocNone {
				d80 = ps.OverlayValues[80]
			}
			if len(ps.OverlayValues) > 147 && ps.OverlayValues[147].Loc != scm.LocNone {
				d147 = ps.OverlayValues[147]
			}
			if len(ps.OverlayValues) > 148 && ps.OverlayValues[148].Loc != scm.LocNone {
				d148 = ps.OverlayValues[148]
			}
			if len(ps.OverlayValues) > 149 && ps.OverlayValues[149].Loc != scm.LocNone {
				d149 = ps.OverlayValues[149]
			}
			if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != scm.LocNone {
				d150 = ps.OverlayValues[150]
			}
			if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != scm.LocNone {
				d151 = ps.OverlayValues[151]
			}
			if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != scm.LocNone {
				d152 = ps.OverlayValues[152]
			}
			if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != scm.LocNone {
				d225 = ps.OverlayValues[225]
			}
			if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != scm.LocNone {
				d226 = ps.OverlayValues[226]
			}
			if len(ps.OverlayValues) > 227 && ps.OverlayValues[227].Loc != scm.LocNone {
				d227 = ps.OverlayValues[227]
			}
			if len(ps.OverlayValues) > 228 && ps.OverlayValues[228].Loc != scm.LocNone {
				d228 = ps.OverlayValues[228]
			}
			if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != scm.LocNone {
				d229 = ps.OverlayValues[229]
			}
			if len(ps.OverlayValues) > 231 && ps.OverlayValues[231].Loc != scm.LocNone {
				d231 = ps.OverlayValues[231]
			}
			if len(ps.OverlayValues) > 232 && ps.OverlayValues[232].Loc != scm.LocNone {
				d232 = ps.OverlayValues[232]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			if d0.Loc == scm.LocReg {
				ctx.ProtectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.ProtectReg(d0.Reg)
				ctx.ProtectReg(d0.Reg2)
			}
			ctx.EnsureDesc(&d36)
			if d36.Loc == scm.LocReg {
				ctx.ProtectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.ProtectReg(d36.Reg)
				ctx.ProtectReg(d36.Reg2)
			}
			d233 = d0
			if d233.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d233)
			d234 = d233
			if d234.Loc == scm.LocImm {
				d234 = scm.JITValueDesc{Loc: scm.LocImm, Type: d234.Type, Imm: scm.NewInt(int64(uint64(d234.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d234.Reg, 32)
				ctx.EmitShrRegImm8(d234.Reg, 32)
			}
			ctx.EmitStoreToStack(d234, int32(bbs[1].PhiBase)+int32(0))
			d235 = d36
			if d235.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d235)
			d236 = d235
			if d236.Loc == scm.LocImm {
				d236 = scm.JITValueDesc{Loc: scm.LocImm, Type: d236.Type, Imm: scm.NewInt(int64(uint64(d236.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.EmitShlRegImm8(d236.Reg, 32)
				ctx.EmitShrRegImm8(d236.Reg, 32)
			}
			ctx.EmitStoreToStack(d236, int32(bbs[1].PhiBase)+int32(16))
			if d0.Loc == scm.LocReg {
				ctx.UnprotectReg(d0.Reg)
			} else if d0.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d0.Reg)
				ctx.UnprotectReg(d0.Reg2)
			}
			if d36.Loc == scm.LocReg {
				ctx.UnprotectReg(d36.Reg)
			} else if d36.Loc == scm.LocRegPair {
				ctx.UnprotectReg(d36.Reg)
				ctx.UnprotectReg(d36.Reg2)
			}
			ps237 := scm.PhiState{General: ps.General}
			ps237.OverlayValues = make([]scm.JITValueDesc, 237)
			ps237.OverlayValues[0] = d0
			ps237.OverlayValues[1] = d1
			ps237.OverlayValues[2] = d2
			ps237.OverlayValues[3] = d3
			ps237.OverlayValues[4] = d4
			ps237.OverlayValues[5] = d5
			ps237.OverlayValues[7] = d7
			ps237.OverlayValues[8] = d8
			ps237.OverlayValues[9] = d9
			ps237.OverlayValues[10] = d10
			ps237.OverlayValues[11] = d11
			ps237.OverlayValues[12] = d12
			ps237.OverlayValues[15] = d15
			ps237.OverlayValues[16] = d16
			ps237.OverlayValues[34] = d34
			ps237.OverlayValues[35] = d35
			ps237.OverlayValues[36] = d36
			ps237.OverlayValues[37] = d37
			ps237.OverlayValues[38] = d38
			ps237.OverlayValues[39] = d39
			ps237.OverlayValues[40] = d40
			ps237.OverlayValues[41] = d41
			ps237.OverlayValues[42] = d42
			ps237.OverlayValues[43] = d43
			ps237.OverlayValues[44] = d44
			ps237.OverlayValues[45] = d45
			ps237.OverlayValues[46] = d46
			ps237.OverlayValues[47] = d47
			ps237.OverlayValues[48] = d48
			ps237.OverlayValues[49] = d49
			ps237.OverlayValues[50] = d50
			ps237.OverlayValues[51] = d51
			ps237.OverlayValues[52] = d52
			ps237.OverlayValues[53] = d53
			ps237.OverlayValues[54] = d54
			ps237.OverlayValues[55] = d55
			ps237.OverlayValues[56] = d56
			ps237.OverlayValues[57] = d57
			ps237.OverlayValues[58] = d58
			ps237.OverlayValues[59] = d59
			ps237.OverlayValues[60] = d60
			ps237.OverlayValues[61] = d61
			ps237.OverlayValues[62] = d62
			ps237.OverlayValues[63] = d63
			ps237.OverlayValues[64] = d64
			ps237.OverlayValues[65] = d65
			ps237.OverlayValues[66] = d66
			ps237.OverlayValues[67] = d67
			ps237.OverlayValues[68] = d68
			ps237.OverlayValues[69] = d69
			ps237.OverlayValues[70] = d70
			ps237.OverlayValues[71] = d71
			ps237.OverlayValues[72] = d72
			ps237.OverlayValues[73] = d73
			ps237.OverlayValues[74] = d74
			ps237.OverlayValues[75] = d75
			ps237.OverlayValues[76] = d76
			ps237.OverlayValues[77] = d77
			ps237.OverlayValues[78] = d78
			ps237.OverlayValues[79] = d79
			ps237.OverlayValues[80] = d80
			ps237.OverlayValues[147] = d147
			ps237.OverlayValues[148] = d148
			ps237.OverlayValues[149] = d149
			ps237.OverlayValues[150] = d150
			ps237.OverlayValues[151] = d151
			ps237.OverlayValues[152] = d152
			ps237.OverlayValues[225] = d225
			ps237.OverlayValues[226] = d226
			ps237.OverlayValues[227] = d227
			ps237.OverlayValues[228] = d228
			ps237.OverlayValues[229] = d229
			ps237.OverlayValues[231] = d231
			ps237.OverlayValues[232] = d232
			ps237.OverlayValues[233] = d233
			ps237.OverlayValues[234] = d234
			ps237.OverlayValues[235] = d235
			ps237.OverlayValues[236] = d236
			ps237.PhiValues = make([]scm.JITValueDesc, 2)
			d238 = d0
			ps237.PhiValues[0] = d238
			d239 = d36
			ps237.PhiValues[1] = d239
			if ps237.General && bbs[1].Rendered {
				ctx.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps237)
			return result
			}
			ps240 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps240)
			ctx.MarkLabel(lbl0)
			d241 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d241)
			ctx.BindReg(r2, &d241)
			ctx.EmitMovPairToResult(&d241, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
			ctx.PatchInt32(r0, int32(48))
			ctx.EmitAddRSP32(int32(48))
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
