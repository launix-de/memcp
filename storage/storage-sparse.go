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
			var d18 scm.JITValueDesc
			_ = d18
			var d19 scm.JITValueDesc
			_ = d19
			var d20 scm.JITValueDesc
			_ = d20
			var d21 scm.JITValueDesc
			_ = d21
			var d22 scm.JITValueDesc
			_ = d22
			var d23 scm.JITValueDesc
			_ = d23
			var d24 scm.JITValueDesc
			_ = d24
			var d25 scm.JITValueDesc
			_ = d25
			var d26 scm.JITValueDesc
			_ = d26
			var d27 scm.JITValueDesc
			_ = d27
			var d28 scm.JITValueDesc
			_ = d28
			var d29 scm.JITValueDesc
			_ = d29
			var d30 scm.JITValueDesc
			_ = d30
			var d31 scm.JITValueDesc
			_ = d31
			var d32 scm.JITValueDesc
			_ = d32
			var d33 scm.JITValueDesc
			_ = d33
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
			var d83 scm.JITValueDesc
			_ = d83
			var d84 scm.JITValueDesc
			_ = d84
			var d85 scm.JITValueDesc
			_ = d85
			var d86 scm.JITValueDesc
			_ = d86
			var d87 scm.JITValueDesc
			_ = d87
			var d89 scm.JITValueDesc
			_ = d89
			var d90 scm.JITValueDesc
			_ = d90
			var d91 scm.JITValueDesc
			_ = d91
			var d92 scm.JITValueDesc
			_ = d92
			var d93 scm.JITValueDesc
			_ = d93
			var d94 scm.JITValueDesc
			_ = d94
			var d96 scm.JITValueDesc
			_ = d96
			var d97 scm.JITValueDesc
			_ = d97
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
			r0 := ctx.W.EmitSubRSP32Fixup()
			_ = r0
			d0 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 := scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
			var bbs [8]scm.BBDescriptor
			if result.Loc == scm.LocAny {
				result = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
				ctx.BindReg(result.Reg, &result)
				ctx.BindReg(result.Reg2, &result)
			}
			r1 := ctx.AllocReg()
			r2 := ctx.AllocRegExcept(r1)
			lbl0 := ctx.W.ReserveLabel()
			bbpos_0_0 := int32(-1)
			_ = bbpos_0_0
			lbl1 := ctx.W.ReserveLabel()
			bbpos_0_1 := int32(-1)
			_ = bbpos_0_1
			lbl2 := ctx.W.ReserveLabel()
			bbpos_0_2 := int32(-1)
			_ = bbpos_0_2
			lbl3 := ctx.W.ReserveLabel()
			bbpos_0_3 := int32(-1)
			_ = bbpos_0_3
			lbl4 := ctx.W.ReserveLabel()
			bbpos_0_4 := int32(-1)
			_ = bbpos_0_4
			lbl5 := ctx.W.ReserveLabel()
			bbpos_0_5 := int32(-1)
			_ = bbpos_0_5
			lbl6 := ctx.W.ReserveLabel()
			bbpos_0_6 := int32(-1)
			_ = bbpos_0_6
			lbl7 := ctx.W.ReserveLabel()
			bbpos_0_7 := int32(-1)
			_ = bbpos_0_7
			lbl8 := ctx.W.ReserveLabel()
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
					ctx.W.EmitJmp(lbl1)
					return result
				}
				bbs[0].Rendered = true
				bbs[0].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_0 = bbs[0].Address
				ctx.W.MarkLabel(lbl1)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
				ctx.W.EmitMovRegMem64(r3, fieldAddr)
				d2 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r3}
				ctx.BindReg(r3, &d2)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).i))
				r4 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r4, thisptr.Reg, off)
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
				ctx.W.EmitMovRegReg(r5, d2.Reg)
				ctx.W.EmitShlRegImm8(r5, 32)
				ctx.W.EmitShrRegImm8(r5, 32)
				d3 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r5}
				ctx.BindReg(r5, &d3)
			}
			ctx.EmitStoreToStack(scm.JITValueDesc{Loc: scm.LocImm, Imm: scm.NewInt(0)}, 0)
			d4 = d3
			if d4.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d4)
			d5 = d4
			if d5.Loc == scm.LocImm {
				d5 = scm.JITValueDesc{Loc: scm.LocImm, Type: d5.Type, Imm: scm.NewInt(int64(uint64(d5.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d5.Reg, 32)
				ctx.W.EmitShrRegImm8(d5.Reg, 32)
			}
			ctx.EmitStoreToStack(d5, 8)
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
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps6)
			return result
			}
			bbs[1].RenderPS = func(ps scm.PhiState) scm.JITValueDesc {
			if !ps.General {
				if bbs[1].VisitCount >= 2 {
					if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != scm.LocNone {
						d9 := ps.PhiValues[0]
						ctx.EnsureDesc(&d9)
						ctx.EmitStoreToStack(d9, 0)
					}
					if len(ps.PhiValues) > 1 && ps.PhiValues[1].Loc != scm.LocNone {
						d10 := ps.PhiValues[1]
						ctx.EnsureDesc(&d10)
						ctx.EmitStoreToStack(d10, 8)
					}
					ps.General = true
					return bbs[1].RenderPS(ps)
				}
			}
			bbs[1].VisitCount++
			if ps.General {
				if bbs[1].Rendered {
					ctx.W.EmitJmp(lbl2)
					return result
				}
				bbs[1].Rendered = true
				bbs[1].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_1 = bbs[1].Address
				ctx.W.MarkLabel(lbl2)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
					ctx.W.EmitCmpRegImm32(d0.Reg, int32(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.W.EmitCmpInt64(d0.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r6, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r6}
				ctx.BindReg(r6, &d11)
			} else if d0.Loc == scm.LocImm {
				r7 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d0.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d1.Reg)
				ctx.W.EmitSetcc(r7, scm.CcE)
				d11 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r7}
				ctx.BindReg(r7, &d11)
			} else {
				r8 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitCmpInt64(d0.Reg, d1.Reg)
				ctx.W.EmitSetcc(r8, scm.CcE)
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
			lbl9 := ctx.W.ReserveLabel()
			lbl10 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d12.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl9)
			ctx.W.EmitJmp(lbl10)
			ctx.W.MarkLabel(lbl9)
			ctx.W.EmitJmp(lbl3)
			ctx.W.MarkLabel(lbl10)
			ctx.W.EmitJmp(lbl4)
			ps15 := scm.PhiState{General: true}
			ps15.OverlayValues = make([]scm.JITValueDesc, 13)
			ps15.OverlayValues[0] = d0
			ps15.OverlayValues[1] = d1
			ps15.OverlayValues[2] = d2
			ps15.OverlayValues[3] = d3
			ps15.OverlayValues[4] = d4
			ps15.OverlayValues[5] = d5
			ps15.OverlayValues[7] = d7
			ps15.OverlayValues[8] = d8
			ps15.OverlayValues[9] = d9
			ps15.OverlayValues[10] = d10
			ps15.OverlayValues[11] = d11
			ps15.OverlayValues[12] = d12
			ps16 := scm.PhiState{General: true}
			ps16.OverlayValues = make([]scm.JITValueDesc, 13)
			ps16.OverlayValues[0] = d0
			ps16.OverlayValues[1] = d1
			ps16.OverlayValues[2] = d2
			ps16.OverlayValues[3] = d3
			ps16.OverlayValues[4] = d4
			ps16.OverlayValues[5] = d5
			ps16.OverlayValues[7] = d7
			ps16.OverlayValues[8] = d8
			ps16.OverlayValues[9] = d9
			ps16.OverlayValues[10] = d10
			ps16.OverlayValues[11] = d11
			ps16.OverlayValues[12] = d12
			alloc17 := ctx.SnapshotAllocState()
			if !bbs[3].Rendered {
				bbs[3].RenderPS(ps16)
			}
			ctx.RestoreAllocState(alloc17)
			if !bbs[2].Rendered {
				return bbs[2].RenderPS(ps15)
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
					ctx.W.EmitJmp(lbl3)
					return result
				}
				bbs[2].Rendered = true
				bbs[2].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_2 = bbs[2].Address
				ctx.W.MarkLabel(lbl3)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			ctx.ReclaimUntrackedRegs()
			d18 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d18)
			ctx.BindReg(r2, &d18)
			ctx.W.EmitMakeNil(d18)
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl4)
					return result
				}
				bbs[3].Rendered = true
				bbs[3].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_3 = bbs[3].Address
				ctx.W.MarkLabel(lbl4)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			ctx.EnsureDesc(&d0)
			ctx.EnsureDesc(&d1)
			var d19 scm.JITValueDesc
			if d0.Loc == scm.LocImm && d1.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d0.Imm.Int() + d1.Imm.Int())}
			} else if d1.Loc == scm.LocImm && d1.Imm.Int() == 0 {
				r9 := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(r9, d0.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r9}
				ctx.BindReg(r9, &d19)
			} else if d0.Loc == scm.LocImm && d0.Imm.Int() == 0 {
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d1.Reg}
				ctx.BindReg(d1.Reg, &d19)
			} else if d0.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d1.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d0.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d1.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else if d1.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d0.Reg)
				ctx.W.EmitMovRegReg(scratch, d0.Reg)
				if d1.Imm.Int() >= -2147483648 && d1.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d1.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d1.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d19)
			} else {
				r10 := ctx.AllocRegExcept(d0.Reg, d1.Reg)
				ctx.W.EmitMovRegReg(r10, d0.Reg)
				ctx.W.EmitAddInt64(r10, d1.Reg)
				d19 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r10}
				ctx.BindReg(r10, &d19)
			}
			if d19.Loc == scm.LocImm {
				d19 = scm.JITValueDesc{Loc: scm.LocImm, Type: d19.Type, Imm: scm.NewInt(int64(uint64(d19.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d19.Reg, 32)
				ctx.W.EmitShrRegImm8(d19.Reg, 32)
			}
			if d19.Loc == scm.LocReg && d0.Loc == scm.LocReg && d19.Reg == d0.Reg {
				ctx.TransferReg(d0.Reg)
				d0.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d19)
			var d20 scm.JITValueDesc
			if d19.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d19.Imm.Int() / 2)}
			} else {
				r11 := ctx.AllocRegExcept(d19.Reg)
				ctx.W.EmitMovRegReg(r11, d19.Reg)
				ctx.W.EmitShrRegImm8(r11, 1)
				d20 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r11}
				ctx.BindReg(r11, &d20)
			}
			if d20.Loc == scm.LocImm {
				d20 = scm.JITValueDesc{Loc: scm.LocImm, Type: d20.Type, Imm: scm.NewInt(int64(uint64(d20.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d20.Reg, 32)
				ctx.W.EmitShrRegImm8(d20.Reg, 32)
			}
			if d20.Loc == scm.LocReg && d19.Loc == scm.LocReg && d20.Reg == d19.Reg {
				ctx.TransferReg(d19.Reg)
				d19.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d19)
			ctx.EnsureDesc(&d20)
			d21 = d20
			_ = d21
			r12 := d20.Loc == scm.LocReg
			r13 := d20.Reg
			if r12 { ctx.ProtectReg(r13) }
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			lbl11 := ctx.W.ReserveLabel()
			bbpos_1_0 := int32(-1)
			_ = bbpos_1_0
			bbpos_1_1 := int32(-1)
			_ = bbpos_1_1
			bbpos_1_2 := int32(-1)
			_ = bbpos_1_2
			bbpos_1_3 := int32(-1)
			_ = bbpos_1_3
			bbpos_1_0 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d21)
			ctx.EnsureDesc(&d21)
			var d23 scm.JITValueDesc
			if d21.Loc == scm.LocImm {
				d23 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(d21.Imm.Int()))))}
			} else {
				r14 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r14, d21.Reg)
				ctx.W.EmitShlRegImm8(r14, 32)
				ctx.W.EmitShrRegImm8(r14, 32)
				d23 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r14}
				ctx.BindReg(r14, &d23)
			}
			var d24 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d24 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r15 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r15, thisptr.Reg, off)
				d24 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r15}
				ctx.BindReg(r15, &d24)
			}
			ctx.EnsureDesc(&d24)
			ctx.EnsureDesc(&d24)
			var d25 scm.JITValueDesc
			if d24.Loc == scm.LocImm {
				d25 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d24.Imm.Int()))))}
			} else {
				r16 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r16, d24.Reg)
				ctx.W.EmitShlRegImm8(r16, 56)
				ctx.W.EmitShrRegImm8(r16, 56)
				d25 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r16}
				ctx.BindReg(r16, &d25)
			}
			ctx.FreeDesc(&d24)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			ctx.EnsureDesc(&d23)
			ctx.EnsureDesc(&d25)
			var d26 scm.JITValueDesc
			if d23.Loc == scm.LocImm && d25.Loc == scm.LocImm {
				d26 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d23.Imm.Int() * d25.Imm.Int())}
			} else if d23.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d25.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d23.Imm.Int()))
				ctx.W.EmitImulInt64(scratch, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else if d25.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d23.Reg)
				ctx.W.EmitMovRegReg(scratch, d23.Reg)
				if d25.Imm.Int() >= -2147483648 && d25.Imm.Int() <= 2147483647 {
					ctx.W.EmitImulRegImm32(scratch, int32(d25.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d25.Imm.Int()))
					ctx.W.EmitImulInt64(scratch, scm.RegR11)
				}
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d26)
			} else {
				r17 := ctx.AllocRegExcept(d23.Reg, d25.Reg)
				ctx.W.EmitMovRegReg(r17, d23.Reg)
				ctx.W.EmitImulInt64(r17, d25.Reg)
				d26 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r17}
				ctx.BindReg(r17, &d26)
			}
			if d26.Loc == scm.LocReg && d23.Loc == scm.LocReg && d26.Reg == d23.Reg {
				ctx.TransferReg(d23.Reg)
				d23.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d23)
			ctx.FreeDesc(&d25)
			var d27 scm.JITValueDesc
			r18 := ctx.AllocReg()
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 0
				dataPtr := *(*uintptr)(unsafe.Pointer(fieldAddr))
				sliceLen := *(*int)(unsafe.Pointer(fieldAddr + 8))
				ctx.W.EmitMovRegImm64(r18, uint64(dataPtr))
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18, StackOff: int32(sliceLen)}
				ctx.BindReg(r18, &d27)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 0)
				ctx.W.EmitMovRegMem(r18, thisptr.Reg, off)
				d27 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r18}
				ctx.BindReg(r18, &d27)
			}
			ctx.BindReg(r18, &d27)
			ctx.EnsureDesc(&d26)
			var d28 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d28 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() / 64)}
			} else {
				r19 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r19, d26.Reg)
				ctx.W.EmitShrRegImm8(r19, 6)
				d28 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r19}
				ctx.BindReg(r19, &d28)
			}
			if d28.Loc == scm.LocReg && d26.Loc == scm.LocReg && d28.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d28)
			r20 := ctx.AllocReg()
			ctx.EnsureDesc(&d28)
			ctx.EnsureDesc(&d27)
			if d28.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r20, uint64(d28.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r20, d28.Reg)
				ctx.W.EmitShlRegImm8(r20, 3)
			}
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(r20, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r20, d27.Reg)
			}
			r21 := ctx.AllocRegExcept(r20)
			ctx.W.EmitMovRegMem(r21, r20, 0)
			ctx.FreeReg(r20)
			d29 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r21}
			ctx.BindReg(r21, &d29)
			ctx.FreeDesc(&d28)
			ctx.EnsureDesc(&d26)
			var d30 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d30 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() % 64)}
			} else {
				r22 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r22, d26.Reg)
				ctx.W.EmitAndRegImm32(r22, 63)
				d30 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r22}
				ctx.BindReg(r22, &d30)
			}
			if d30.Loc == scm.LocReg && d26.Loc == scm.LocReg && d30.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d29)
			ctx.EnsureDesc(&d30)
			var d31 scm.JITValueDesc
			if d29.Loc == scm.LocImm && d30.Loc == scm.LocImm {
				d31 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d29.Imm.Int()) << uint64(d30.Imm.Int())))}
			} else if d30.Loc == scm.LocImm {
				r23 := ctx.AllocRegExcept(d29.Reg)
				ctx.W.EmitMovRegReg(r23, d29.Reg)
				ctx.W.EmitShlRegImm8(r23, uint8(d30.Imm.Int()))
				d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r23}
				ctx.BindReg(r23, &d31)
			} else {
				{
					shiftSrc := d29.Reg
					r24 := ctx.AllocRegExcept(d29.Reg)
					ctx.W.EmitMovRegReg(r24, d29.Reg)
					shiftSrc = r24
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d30.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d30.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d30.Reg)
					}
					ctx.W.EmitShlRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d31 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d31)
				}
			}
			if d31.Loc == scm.LocReg && d29.Loc == scm.LocReg && d31.Reg == d29.Reg {
				ctx.TransferReg(d29.Reg)
				d29.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d29)
			ctx.FreeDesc(&d30)
			var d32 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 25
				val := *(*bool)(unsafe.Pointer(fieldAddr))
				d32 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 25)
				r25 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r25, thisptr.Reg, off)
				d32 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r25}
				ctx.BindReg(r25, &d32)
			}
			d33 = d32
			ctx.EnsureDesc(&d33)
			if d33.Loc != scm.LocImm && d33.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl12 := ctx.W.ReserveLabel()
			lbl13 := ctx.W.ReserveLabel()
			lbl14 := ctx.W.ReserveLabel()
			lbl15 := ctx.W.ReserveLabel()
			if d33.Loc == scm.LocImm {
				if d33.Imm.Bool() {
					ctx.W.MarkLabel(lbl14)
					ctx.W.EmitJmp(lbl12)
				} else {
					ctx.W.MarkLabel(lbl15)
			d34 = d31
			if d34.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d34)
			ctx.EmitStoreToStack(d34, 16)
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d33.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl14)
				ctx.W.EmitJmp(lbl15)
				ctx.W.MarkLabel(lbl14)
				ctx.W.EmitJmp(lbl12)
				ctx.W.MarkLabel(lbl15)
			d35 = d31
			if d35.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d35)
			ctx.EmitStoreToStack(d35, 16)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d32)
			bbpos_1_2 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl13)
			ctx.W.ResolveFixups()
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			var d36 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d36 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r26 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r26, thisptr.Reg, off)
				d36 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r26}
				ctx.BindReg(r26, &d36)
			}
			ctx.EnsureDesc(&d36)
			ctx.EnsureDesc(&d36)
			var d37 scm.JITValueDesc
			if d36.Loc == scm.LocImm {
				d37 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d36.Imm.Int()))))}
			} else {
				r27 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r27, d36.Reg)
				ctx.W.EmitShlRegImm8(r27, 56)
				ctx.W.EmitShrRegImm8(r27, 56)
				d37 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r27}
				ctx.BindReg(r27, &d37)
			}
			ctx.FreeDesc(&d36)
			d38 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			ctx.EnsureDesc(&d38)
			ctx.EnsureDesc(&d37)
			var d39 scm.JITValueDesc
			if d38.Loc == scm.LocImm && d37.Loc == scm.LocImm {
				d39 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d38.Imm.Int() - d37.Imm.Int())}
			} else if d37.Loc == scm.LocImm && d37.Imm.Int() == 0 {
				r28 := ctx.AllocRegExcept(d38.Reg)
				ctx.W.EmitMovRegReg(r28, d38.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r28}
				ctx.BindReg(r28, &d39)
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
				r29 := ctx.AllocRegExcept(d38.Reg, d37.Reg)
				ctx.W.EmitMovRegReg(r29, d38.Reg)
				ctx.W.EmitSubInt64(r29, d37.Reg)
				d39 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r29}
				ctx.BindReg(r29, &d39)
			}
			if d39.Loc == scm.LocReg && d38.Loc == scm.LocReg && d39.Reg == d38.Reg {
				ctx.TransferReg(d38.Reg)
				d38.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d37)
			ctx.EnsureDesc(&d22)
			ctx.EnsureDesc(&d39)
			var d40 scm.JITValueDesc
			if d22.Loc == scm.LocImm && d39.Loc == scm.LocImm {
				d40 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d22.Imm.Int()) >> uint64(d39.Imm.Int())))}
			} else if d39.Loc == scm.LocImm {
				r30 := ctx.AllocRegExcept(d22.Reg)
				ctx.W.EmitMovRegReg(r30, d22.Reg)
				ctx.W.EmitShrRegImm8(r30, uint8(d39.Imm.Int()))
				d40 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r30}
				ctx.BindReg(r30, &d40)
			} else {
				{
					shiftSrc := d22.Reg
					r31 := ctx.AllocRegExcept(d22.Reg)
					ctx.W.EmitMovRegReg(r31, d22.Reg)
					shiftSrc = r31
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
			if d40.Loc == scm.LocReg && d22.Loc == scm.LocReg && d40.Reg == d22.Reg {
				ctx.TransferReg(d22.Reg)
				d22.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d22)
			ctx.FreeDesc(&d39)
			r32 := ctx.AllocReg()
			ctx.EnsureDesc(&d40)
			ctx.EnsureDesc(&d40)
			if d40.Loc == scm.LocRegPair {
				panic("jit: scalar inline return has scm.LocRegPair")
			} else {
				ctx.EmitMovToReg(r32, d40)
			}
			ctx.W.EmitJmp(lbl11)
			bbpos_1_3 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl12)
			ctx.W.ResolveFixups()
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d26)
			var d41 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d41 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() % 64)}
			} else {
				r33 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r33, d26.Reg)
				ctx.W.EmitAndRegImm32(r33, 63)
				d41 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r33}
				ctx.BindReg(r33, &d41)
			}
			if d41.Loc == scm.LocReg && d26.Loc == scm.LocReg && d41.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			var d42 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 24
				val := *(*uint8)(unsafe.Pointer(fieldAddr))
				d42 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(val))}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 24)
				r34 := ctx.AllocReg()
				ctx.W.EmitMovRegMemB(r34, thisptr.Reg, off)
				d42 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r34}
				ctx.BindReg(r34, &d42)
			}
			ctx.EnsureDesc(&d42)
			ctx.EnsureDesc(&d42)
			var d43 scm.JITValueDesc
			if d42.Loc == scm.LocImm {
				d43 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint8(d42.Imm.Int()))))}
			} else {
				r35 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r35, d42.Reg)
				ctx.W.EmitShlRegImm8(r35, 56)
				ctx.W.EmitShrRegImm8(r35, 56)
				d43 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r35}
				ctx.BindReg(r35, &d43)
			}
			ctx.FreeDesc(&d42)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			ctx.EnsureDesc(&d41)
			ctx.EnsureDesc(&d43)
			var d44 scm.JITValueDesc
			if d41.Loc == scm.LocImm && d43.Loc == scm.LocImm {
				d44 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d41.Imm.Int() + d43.Imm.Int())}
			} else if d43.Loc == scm.LocImm && d43.Imm.Int() == 0 {
				r36 := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(r36, d41.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r36}
				ctx.BindReg(r36, &d44)
			} else if d41.Loc == scm.LocImm && d41.Imm.Int() == 0 {
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d43.Reg}
				ctx.BindReg(d43.Reg, &d44)
			} else if d41.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d43.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d41.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else if d43.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d41.Reg)
				ctx.W.EmitMovRegReg(scratch, d41.Reg)
				if d43.Imm.Int() >= -2147483648 && d43.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d43.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d43.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d44)
			} else {
				r37 := ctx.AllocRegExcept(d41.Reg, d43.Reg)
				ctx.W.EmitMovRegReg(r37, d41.Reg)
				ctx.W.EmitAddInt64(r37, d43.Reg)
				d44 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r37}
				ctx.BindReg(r37, &d44)
			}
			if d44.Loc == scm.LocReg && d41.Loc == scm.LocReg && d44.Reg == d41.Reg {
				ctx.TransferReg(d41.Reg)
				d41.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d41)
			ctx.FreeDesc(&d43)
			ctx.EnsureDesc(&d44)
			var d45 scm.JITValueDesc
			if d44.Loc == scm.LocImm {
				d45 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d44.Imm.Int()) > uint64(64))}
			} else {
				r38 := ctx.AllocRegExcept(d44.Reg)
				ctx.W.EmitCmpRegImm32(d44.Reg, 64)
				ctx.W.EmitSetcc(r38, scm.CcA)
				d45 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r38}
				ctx.BindReg(r38, &d45)
			}
			ctx.FreeDesc(&d44)
			d46 = d45
			ctx.EnsureDesc(&d46)
			if d46.Loc != scm.LocImm && d46.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			lbl16 := ctx.W.ReserveLabel()
			lbl17 := ctx.W.ReserveLabel()
			lbl18 := ctx.W.ReserveLabel()
			if d46.Loc == scm.LocImm {
				if d46.Imm.Bool() {
					ctx.W.MarkLabel(lbl17)
					ctx.W.EmitJmp(lbl16)
				} else {
					ctx.W.MarkLabel(lbl18)
			d47 = d31
			if d47.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d47)
			ctx.EmitStoreToStack(d47, 16)
					ctx.W.EmitJmp(lbl13)
				}
			} else {
				ctx.W.EmitCmpRegImm32(d46.Reg, 0)
				ctx.W.EmitJcc(scm.CcNE, lbl17)
				ctx.W.EmitJmp(lbl18)
				ctx.W.MarkLabel(lbl17)
				ctx.W.EmitJmp(lbl16)
				ctx.W.MarkLabel(lbl18)
			d48 = d31
			if d48.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d48)
			ctx.EmitStoreToStack(d48, 16)
				ctx.W.EmitJmp(lbl13)
			}
			ctx.FreeDesc(&d45)
			bbpos_1_1 = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
			ctx.W.MarkLabel(lbl16)
			ctx.W.ResolveFixups()
			d22 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(16)}
			ctx.EnsureDesc(&d26)
			var d49 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d49 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() / 64)}
			} else {
				r39 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r39, d26.Reg)
				ctx.W.EmitShrRegImm8(r39, 6)
				d49 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r39}
				ctx.BindReg(r39, &d49)
			}
			if d49.Loc == scm.LocReg && d26.Loc == scm.LocReg && d49.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.EnsureDesc(&d49)
			ctx.EnsureDesc(&d49)
			var d50 scm.JITValueDesc
			if d49.Loc == scm.LocImm {
				d50 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d49.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d49.Reg)
				ctx.W.EmitMovRegReg(scratch, d49.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d50 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d50)
			}
			if d50.Loc == scm.LocReg && d49.Loc == scm.LocReg && d50.Reg == d49.Reg {
				ctx.TransferReg(d49.Reg)
				d49.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d49)
			ctx.EnsureDesc(&d50)
			r40 := ctx.AllocReg()
			ctx.EnsureDesc(&d50)
			ctx.EnsureDesc(&d27)
			if d50.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r40, uint64(d50.Imm.Int()) * 8)
			} else {
				ctx.W.EmitMovRegReg(r40, d50.Reg)
				ctx.W.EmitShlRegImm8(r40, 3)
			}
			if d27.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d27.Imm.Int()))
				ctx.W.EmitAddInt64(r40, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r40, d27.Reg)
			}
			r41 := ctx.AllocRegExcept(r40)
			ctx.W.EmitMovRegMem(r41, r40, 0)
			ctx.FreeReg(r40)
			d51 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r41}
			ctx.BindReg(r41, &d51)
			ctx.FreeDesc(&d50)
			ctx.EnsureDesc(&d26)
			var d52 scm.JITValueDesc
			if d26.Loc == scm.LocImm {
				d52 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d26.Imm.Int() % 64)}
			} else {
				r42 := ctx.AllocRegExcept(d26.Reg)
				ctx.W.EmitMovRegReg(r42, d26.Reg)
				ctx.W.EmitAndRegImm32(r42, 63)
				d52 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r42}
				ctx.BindReg(r42, &d52)
			}
			if d52.Loc == scm.LocReg && d26.Loc == scm.LocReg && d52.Reg == d26.Reg {
				ctx.TransferReg(d26.Reg)
				d26.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d26)
			d53 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(64)}
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d52)
			ctx.EnsureDesc(&d53)
			ctx.EnsureDesc(&d52)
			var d54 scm.JITValueDesc
			if d53.Loc == scm.LocImm && d52.Loc == scm.LocImm {
				d54 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d53.Imm.Int() - d52.Imm.Int())}
			} else if d52.Loc == scm.LocImm && d52.Imm.Int() == 0 {
				r43 := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(r43, d53.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r43}
				ctx.BindReg(r43, &d54)
			} else if d53.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d52.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d53.Imm.Int()))
				ctx.W.EmitSubInt64(scratch, d52.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else if d52.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d53.Reg)
				ctx.W.EmitMovRegReg(scratch, d53.Reg)
				if d52.Imm.Int() >= -2147483648 && d52.Imm.Int() <= 2147483647 {
					ctx.W.EmitSubRegImm32(scratch, int32(d52.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d52.Imm.Int()))
					ctx.W.EmitSubInt64(scratch, scm.RegR11)
				}
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d54)
			} else {
				r44 := ctx.AllocRegExcept(d53.Reg, d52.Reg)
				ctx.W.EmitMovRegReg(r44, d53.Reg)
				ctx.W.EmitSubInt64(r44, d52.Reg)
				d54 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r44}
				ctx.BindReg(r44, &d54)
			}
			if d54.Loc == scm.LocReg && d53.Loc == scm.LocReg && d54.Reg == d53.Reg {
				ctx.TransferReg(d53.Reg)
				d53.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d52)
			ctx.EnsureDesc(&d51)
			ctx.EnsureDesc(&d54)
			var d55 scm.JITValueDesc
			if d51.Loc == scm.LocImm && d54.Loc == scm.LocImm {
				d55 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(d51.Imm.Int()) >> uint64(d54.Imm.Int())))}
			} else if d54.Loc == scm.LocImm {
				r45 := ctx.AllocRegExcept(d51.Reg)
				ctx.W.EmitMovRegReg(r45, d51.Reg)
				ctx.W.EmitShrRegImm8(r45, uint8(d54.Imm.Int()))
				d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r45}
				ctx.BindReg(r45, &d55)
			} else {
				{
					shiftSrc := d51.Reg
					r46 := ctx.AllocRegExcept(d51.Reg)
					ctx.W.EmitMovRegReg(r46, d51.Reg)
					shiftSrc = r46
					rcxUsed := ctx.FreeRegs & (1 << uint(scm.RegRCX)) == 0 && d54.Reg != scm.RegRCX
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegR11, scm.RegRCX)
					}
					if d54.Reg != scm.RegRCX {
						ctx.W.EmitMovRegReg(scm.RegRCX, d54.Reg)
					}
					ctx.W.EmitShrRegCl(shiftSrc)
					if rcxUsed {
						ctx.W.EmitMovRegReg(scm.RegRCX, scm.RegR11)
					}
					d55 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: shiftSrc}
					ctx.BindReg(shiftSrc, &d55)
				}
			}
			if d55.Loc == scm.LocReg && d51.Loc == scm.LocReg && d55.Reg == d51.Reg {
				ctx.TransferReg(d51.Reg)
				d51.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d51)
			ctx.FreeDesc(&d54)
			ctx.EnsureDesc(&d31)
			ctx.EnsureDesc(&d55)
			var d56 scm.JITValueDesc
			if d31.Loc == scm.LocImm && d55.Loc == scm.LocImm {
				d56 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d31.Imm.Int() | d55.Imm.Int())}
			} else if d31.Loc == scm.LocImm && d31.Imm.Int() == 0 {
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d55.Reg}
				ctx.BindReg(d55.Reg, &d56)
			} else if d55.Loc == scm.LocImm && d55.Imm.Int() == 0 {
				r47 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r47, d31.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r47}
				ctx.BindReg(r47, &d56)
			} else if d31.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d55.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d31.Imm.Int()))
				ctx.W.EmitOrInt64(scratch, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d56)
			} else if d55.Loc == scm.LocImm {
				r48 := ctx.AllocRegExcept(d31.Reg)
				ctx.W.EmitMovRegReg(r48, d31.Reg)
				if d55.Imm.Int() >= -2147483648 && d55.Imm.Int() <= 2147483647 {
					ctx.W.EmitOrRegImm32(r48, int32(d55.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d55.Imm.Int()))
					ctx.W.EmitOrInt64(r48, scm.RegR11)
				}
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r48}
				ctx.BindReg(r48, &d56)
			} else {
				r49 := ctx.AllocRegExcept(d31.Reg, d55.Reg)
				ctx.W.EmitMovRegReg(r49, d31.Reg)
				ctx.W.EmitOrInt64(r49, d55.Reg)
				d56 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r49}
				ctx.BindReg(r49, &d56)
			}
			if d56.Loc == scm.LocReg && d31.Loc == scm.LocReg && d56.Reg == d31.Reg {
				ctx.TransferReg(d31.Reg)
				d31.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d55)
			d57 = d56
			if d57.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d57)
			ctx.EmitStoreToStack(d57, 16)
			ctx.W.EmitJmp(lbl13)
			ctx.W.MarkLabel(lbl11)
			d58 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r32}
			ctx.BindReg(r32, &d58)
			ctx.BindReg(r32, &d58)
			if r12 { ctx.UnprotectReg(r13) }
			var d59 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).recids) + 32
				val := *(*int64)(unsafe.Pointer(fieldAddr))
				d59 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(val)}
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).recids) + 32)
				r50 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r50, thisptr.Reg, off)
				d59 = scm.JITValueDesc{Loc: scm.LocReg, Reg: r50}
				ctx.BindReg(r50, &d59)
			}
			ctx.EnsureDesc(&d59)
			ctx.EnsureDesc(&d59)
			var d60 scm.JITValueDesc
			if d59.Loc == scm.LocImm {
				d60 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(int64(d59.Imm.Int()))))}
			} else {
				r51 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r51, d59.Reg)
				d60 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r51}
				ctx.BindReg(r51, &d60)
			}
			ctx.FreeDesc(&d59)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d60)
			ctx.EnsureDesc(&d58)
			ctx.EnsureDesc(&d60)
			var d61 scm.JITValueDesc
			if d58.Loc == scm.LocImm && d60.Loc == scm.LocImm {
				d61 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d58.Imm.Int() + d60.Imm.Int())}
			} else if d60.Loc == scm.LocImm && d60.Imm.Int() == 0 {
				r52 := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(r52, d58.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r52}
				ctx.BindReg(r52, &d61)
			} else if d58.Loc == scm.LocImm && d58.Imm.Int() == 0 {
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: d60.Reg}
				ctx.BindReg(d60.Reg, &d61)
			} else if d58.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d60.Reg)
				ctx.W.EmitMovRegImm64(scratch, uint64(d58.Imm.Int()))
				ctx.W.EmitAddInt64(scratch, d60.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else if d60.Loc == scm.LocImm {
				scratch := ctx.AllocRegExcept(d58.Reg)
				ctx.W.EmitMovRegReg(scratch, d58.Reg)
				if d60.Imm.Int() >= -2147483648 && d60.Imm.Int() <= 2147483647 {
					ctx.W.EmitAddRegImm32(scratch, int32(d60.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d60.Imm.Int()))
					ctx.W.EmitAddInt64(scratch, scm.RegR11)
				}
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d61)
			} else {
				r53 := ctx.AllocRegExcept(d58.Reg, d60.Reg)
				ctx.W.EmitMovRegReg(r53, d58.Reg)
				ctx.W.EmitAddInt64(r53, d60.Reg)
				d61 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r53}
				ctx.BindReg(r53, &d61)
			}
			if d61.Loc == scm.LocReg && d58.Loc == scm.LocReg && d61.Reg == d58.Reg {
				ctx.TransferReg(d58.Reg)
				d58.Loc = scm.LocNone
			}
			ctx.FreeDesc(&d58)
			ctx.FreeDesc(&d60)
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d62 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d62 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r54 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r54, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r54, 32)
				ctx.W.EmitShrRegImm8(r54, 32)
				d62 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r54}
				ctx.BindReg(r54, &d62)
			}
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d62)
			var d63 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d62.Loc == scm.LocImm {
				d63 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d61.Imm.Int()) == uint64(d62.Imm.Int()))}
			} else if d62.Loc == scm.LocImm {
				r55 := ctx.AllocRegExcept(d61.Reg)
				if d62.Imm.Int() >= -2147483648 && d62.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d61.Reg, int32(d62.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d62.Imm.Int()))
					ctx.W.EmitCmpInt64(d61.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r55, scm.CcE)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r55}
				ctx.BindReg(r55, &d63)
			} else if d61.Loc == scm.LocImm {
				r56 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d62.Reg)
				ctx.W.EmitSetcc(r56, scm.CcE)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r56}
				ctx.BindReg(r56, &d63)
			} else {
				r57 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitCmpInt64(d61.Reg, d62.Reg)
				ctx.W.EmitSetcc(r57, scm.CcE)
				d63 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r57}
				ctx.BindReg(r57, &d63)
			}
			ctx.FreeDesc(&d62)
			d64 = d63
			ctx.EnsureDesc(&d64)
			if d64.Loc != scm.LocImm && d64.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d64.Loc == scm.LocImm {
				if d64.Imm.Bool() {
			ps65 := scm.PhiState{General: ps.General}
			ps65.OverlayValues = make([]scm.JITValueDesc, 65)
			ps65.OverlayValues[0] = d0
			ps65.OverlayValues[1] = d1
			ps65.OverlayValues[2] = d2
			ps65.OverlayValues[3] = d3
			ps65.OverlayValues[4] = d4
			ps65.OverlayValues[5] = d5
			ps65.OverlayValues[7] = d7
			ps65.OverlayValues[8] = d8
			ps65.OverlayValues[9] = d9
			ps65.OverlayValues[10] = d10
			ps65.OverlayValues[11] = d11
			ps65.OverlayValues[12] = d12
			ps65.OverlayValues[18] = d18
			ps65.OverlayValues[19] = d19
			ps65.OverlayValues[20] = d20
			ps65.OverlayValues[21] = d21
			ps65.OverlayValues[22] = d22
			ps65.OverlayValues[23] = d23
			ps65.OverlayValues[24] = d24
			ps65.OverlayValues[25] = d25
			ps65.OverlayValues[26] = d26
			ps65.OverlayValues[27] = d27
			ps65.OverlayValues[28] = d28
			ps65.OverlayValues[29] = d29
			ps65.OverlayValues[30] = d30
			ps65.OverlayValues[31] = d31
			ps65.OverlayValues[32] = d32
			ps65.OverlayValues[33] = d33
			ps65.OverlayValues[34] = d34
			ps65.OverlayValues[35] = d35
			ps65.OverlayValues[36] = d36
			ps65.OverlayValues[37] = d37
			ps65.OverlayValues[38] = d38
			ps65.OverlayValues[39] = d39
			ps65.OverlayValues[40] = d40
			ps65.OverlayValues[41] = d41
			ps65.OverlayValues[42] = d42
			ps65.OverlayValues[43] = d43
			ps65.OverlayValues[44] = d44
			ps65.OverlayValues[45] = d45
			ps65.OverlayValues[46] = d46
			ps65.OverlayValues[47] = d47
			ps65.OverlayValues[48] = d48
			ps65.OverlayValues[49] = d49
			ps65.OverlayValues[50] = d50
			ps65.OverlayValues[51] = d51
			ps65.OverlayValues[52] = d52
			ps65.OverlayValues[53] = d53
			ps65.OverlayValues[54] = d54
			ps65.OverlayValues[55] = d55
			ps65.OverlayValues[56] = d56
			ps65.OverlayValues[57] = d57
			ps65.OverlayValues[58] = d58
			ps65.OverlayValues[59] = d59
			ps65.OverlayValues[60] = d60
			ps65.OverlayValues[61] = d61
			ps65.OverlayValues[62] = d62
			ps65.OverlayValues[63] = d63
			ps65.OverlayValues[64] = d64
					return bbs[4].RenderPS(ps65)
				}
			ps66 := scm.PhiState{General: ps.General}
			ps66.OverlayValues = make([]scm.JITValueDesc, 65)
			ps66.OverlayValues[0] = d0
			ps66.OverlayValues[1] = d1
			ps66.OverlayValues[2] = d2
			ps66.OverlayValues[3] = d3
			ps66.OverlayValues[4] = d4
			ps66.OverlayValues[5] = d5
			ps66.OverlayValues[7] = d7
			ps66.OverlayValues[8] = d8
			ps66.OverlayValues[9] = d9
			ps66.OverlayValues[10] = d10
			ps66.OverlayValues[11] = d11
			ps66.OverlayValues[12] = d12
			ps66.OverlayValues[18] = d18
			ps66.OverlayValues[19] = d19
			ps66.OverlayValues[20] = d20
			ps66.OverlayValues[21] = d21
			ps66.OverlayValues[22] = d22
			ps66.OverlayValues[23] = d23
			ps66.OverlayValues[24] = d24
			ps66.OverlayValues[25] = d25
			ps66.OverlayValues[26] = d26
			ps66.OverlayValues[27] = d27
			ps66.OverlayValues[28] = d28
			ps66.OverlayValues[29] = d29
			ps66.OverlayValues[30] = d30
			ps66.OverlayValues[31] = d31
			ps66.OverlayValues[32] = d32
			ps66.OverlayValues[33] = d33
			ps66.OverlayValues[34] = d34
			ps66.OverlayValues[35] = d35
			ps66.OverlayValues[36] = d36
			ps66.OverlayValues[37] = d37
			ps66.OverlayValues[38] = d38
			ps66.OverlayValues[39] = d39
			ps66.OverlayValues[40] = d40
			ps66.OverlayValues[41] = d41
			ps66.OverlayValues[42] = d42
			ps66.OverlayValues[43] = d43
			ps66.OverlayValues[44] = d44
			ps66.OverlayValues[45] = d45
			ps66.OverlayValues[46] = d46
			ps66.OverlayValues[47] = d47
			ps66.OverlayValues[48] = d48
			ps66.OverlayValues[49] = d49
			ps66.OverlayValues[50] = d50
			ps66.OverlayValues[51] = d51
			ps66.OverlayValues[52] = d52
			ps66.OverlayValues[53] = d53
			ps66.OverlayValues[54] = d54
			ps66.OverlayValues[55] = d55
			ps66.OverlayValues[56] = d56
			ps66.OverlayValues[57] = d57
			ps66.OverlayValues[58] = d58
			ps66.OverlayValues[59] = d59
			ps66.OverlayValues[60] = d60
			ps66.OverlayValues[61] = d61
			ps66.OverlayValues[62] = d62
			ps66.OverlayValues[63] = d63
			ps66.OverlayValues[64] = d64
				return bbs[5].RenderPS(ps66)
			}
			lbl19 := ctx.W.ReserveLabel()
			lbl20 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d64.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl19)
			ctx.W.EmitJmp(lbl20)
			ctx.W.MarkLabel(lbl19)
			ctx.W.EmitJmp(lbl5)
			ctx.W.MarkLabel(lbl20)
			ctx.W.EmitJmp(lbl6)
			ps67 := scm.PhiState{General: true}
			ps67.OverlayValues = make([]scm.JITValueDesc, 65)
			ps67.OverlayValues[0] = d0
			ps67.OverlayValues[1] = d1
			ps67.OverlayValues[2] = d2
			ps67.OverlayValues[3] = d3
			ps67.OverlayValues[4] = d4
			ps67.OverlayValues[5] = d5
			ps67.OverlayValues[7] = d7
			ps67.OverlayValues[8] = d8
			ps67.OverlayValues[9] = d9
			ps67.OverlayValues[10] = d10
			ps67.OverlayValues[11] = d11
			ps67.OverlayValues[12] = d12
			ps67.OverlayValues[18] = d18
			ps67.OverlayValues[19] = d19
			ps67.OverlayValues[20] = d20
			ps67.OverlayValues[21] = d21
			ps67.OverlayValues[22] = d22
			ps67.OverlayValues[23] = d23
			ps67.OverlayValues[24] = d24
			ps67.OverlayValues[25] = d25
			ps67.OverlayValues[26] = d26
			ps67.OverlayValues[27] = d27
			ps67.OverlayValues[28] = d28
			ps67.OverlayValues[29] = d29
			ps67.OverlayValues[30] = d30
			ps67.OverlayValues[31] = d31
			ps67.OverlayValues[32] = d32
			ps67.OverlayValues[33] = d33
			ps67.OverlayValues[34] = d34
			ps67.OverlayValues[35] = d35
			ps67.OverlayValues[36] = d36
			ps67.OverlayValues[37] = d37
			ps67.OverlayValues[38] = d38
			ps67.OverlayValues[39] = d39
			ps67.OverlayValues[40] = d40
			ps67.OverlayValues[41] = d41
			ps67.OverlayValues[42] = d42
			ps67.OverlayValues[43] = d43
			ps67.OverlayValues[44] = d44
			ps67.OverlayValues[45] = d45
			ps67.OverlayValues[46] = d46
			ps67.OverlayValues[47] = d47
			ps67.OverlayValues[48] = d48
			ps67.OverlayValues[49] = d49
			ps67.OverlayValues[50] = d50
			ps67.OverlayValues[51] = d51
			ps67.OverlayValues[52] = d52
			ps67.OverlayValues[53] = d53
			ps67.OverlayValues[54] = d54
			ps67.OverlayValues[55] = d55
			ps67.OverlayValues[56] = d56
			ps67.OverlayValues[57] = d57
			ps67.OverlayValues[58] = d58
			ps67.OverlayValues[59] = d59
			ps67.OverlayValues[60] = d60
			ps67.OverlayValues[61] = d61
			ps67.OverlayValues[62] = d62
			ps67.OverlayValues[63] = d63
			ps67.OverlayValues[64] = d64
			ps68 := scm.PhiState{General: true}
			ps68.OverlayValues = make([]scm.JITValueDesc, 65)
			ps68.OverlayValues[0] = d0
			ps68.OverlayValues[1] = d1
			ps68.OverlayValues[2] = d2
			ps68.OverlayValues[3] = d3
			ps68.OverlayValues[4] = d4
			ps68.OverlayValues[5] = d5
			ps68.OverlayValues[7] = d7
			ps68.OverlayValues[8] = d8
			ps68.OverlayValues[9] = d9
			ps68.OverlayValues[10] = d10
			ps68.OverlayValues[11] = d11
			ps68.OverlayValues[12] = d12
			ps68.OverlayValues[18] = d18
			ps68.OverlayValues[19] = d19
			ps68.OverlayValues[20] = d20
			ps68.OverlayValues[21] = d21
			ps68.OverlayValues[22] = d22
			ps68.OverlayValues[23] = d23
			ps68.OverlayValues[24] = d24
			ps68.OverlayValues[25] = d25
			ps68.OverlayValues[26] = d26
			ps68.OverlayValues[27] = d27
			ps68.OverlayValues[28] = d28
			ps68.OverlayValues[29] = d29
			ps68.OverlayValues[30] = d30
			ps68.OverlayValues[31] = d31
			ps68.OverlayValues[32] = d32
			ps68.OverlayValues[33] = d33
			ps68.OverlayValues[34] = d34
			ps68.OverlayValues[35] = d35
			ps68.OverlayValues[36] = d36
			ps68.OverlayValues[37] = d37
			ps68.OverlayValues[38] = d38
			ps68.OverlayValues[39] = d39
			ps68.OverlayValues[40] = d40
			ps68.OverlayValues[41] = d41
			ps68.OverlayValues[42] = d42
			ps68.OverlayValues[43] = d43
			ps68.OverlayValues[44] = d44
			ps68.OverlayValues[45] = d45
			ps68.OverlayValues[46] = d46
			ps68.OverlayValues[47] = d47
			ps68.OverlayValues[48] = d48
			ps68.OverlayValues[49] = d49
			ps68.OverlayValues[50] = d50
			ps68.OverlayValues[51] = d51
			ps68.OverlayValues[52] = d52
			ps68.OverlayValues[53] = d53
			ps68.OverlayValues[54] = d54
			ps68.OverlayValues[55] = d55
			ps68.OverlayValues[56] = d56
			ps68.OverlayValues[57] = d57
			ps68.OverlayValues[58] = d58
			ps68.OverlayValues[59] = d59
			ps68.OverlayValues[60] = d60
			ps68.OverlayValues[61] = d61
			ps68.OverlayValues[62] = d62
			ps68.OverlayValues[63] = d63
			ps68.OverlayValues[64] = d64
			snap69 := d20
			alloc70 := ctx.SnapshotAllocState()
			if !bbs[5].Rendered {
				bbs[5].RenderPS(ps68)
			}
			ctx.RestoreAllocState(alloc70)
			d20 = snap69
			if !bbs[4].Rendered {
				return bbs[4].RenderPS(ps67)
			}
			return result
			ctx.FreeDesc(&d63)
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
					ctx.W.EmitJmp(lbl5)
					return result
				}
				bbs[4].Rendered = true
				bbs[4].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_4 = bbs[4].Address
				ctx.W.MarkLabel(lbl5)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
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
			ctx.ReclaimUntrackedRegs()
			var d71 scm.JITValueDesc
			if thisptr.Loc == scm.LocImm {
				fieldAddr := uintptr(thisptr.Imm.Int()) + unsafe.Offsetof((*StorageSparse)(nil).values)
				r58 := ctx.AllocReg()
				r59 := ctx.AllocReg()
				ctx.W.EmitMovRegMem64(r58, fieldAddr)
				ctx.W.EmitMovRegMem64(r59, fieldAddr+8)
				d71 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r58, Reg2: r59}
				ctx.BindReg(r58, &d71)
				ctx.BindReg(r59, &d71)
			} else {
				off := int32(unsafe.Offsetof((*StorageSparse)(nil).values))
				r60 := ctx.AllocReg()
				r61 := ctx.AllocReg()
				ctx.W.EmitMovRegMem(r60, thisptr.Reg, off)
				ctx.W.EmitMovRegMem(r61, thisptr.Reg, off+8)
				d71 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r60, Reg2: r61}
				ctx.BindReg(r60, &d71)
				ctx.BindReg(r61, &d71)
			}
			ctx.EnsureDesc(&d20)
			r62 := ctx.AllocReg()
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d71)
			if d20.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(r62, uint64(d20.Imm.Int()) * 16)
			} else {
				ctx.W.EmitMovRegReg(r62, d20.Reg)
				ctx.W.EmitShlRegImm8(r62, 4)
			}
			if d71.Loc == scm.LocImm {
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d71.Imm.Int()))
				ctx.W.EmitAddInt64(r62, scm.RegR11)
			} else {
				ctx.W.EmitAddInt64(r62, d71.Reg)
			}
			r63 := ctx.AllocRegExcept(r62)
			r64 := ctx.AllocRegExcept(r62, r63)
			ctx.W.EmitMovRegMem(r63, r62, 0)
			ctx.W.EmitMovRegMem(r64, r62, 8)
			ctx.FreeReg(r62)
			d72 = scm.JITValueDesc{Loc: scm.LocRegPair, Type: scm.JITTypeUnknown, Reg: r63, Reg2: r64}
			ctx.BindReg(r63, &d72)
			ctx.BindReg(r64, &d72)
			d73 = scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d73)
			ctx.BindReg(r2, &d73)
			ctx.EnsureDesc(&d72)
			if d72.Loc == scm.LocRegPair {
				ctx.EmitMovPairToResult(&d72, &d73)
			} else {
				switch d72.Type {
				case scm.TagBool:
					ctx.W.EmitMakeBool(d73, d72)
				case scm.TagInt:
					ctx.W.EmitMakeInt(d73, d72)
				case scm.TagFloat:
					ctx.W.EmitMakeFloat(d73, d72)
				case scm.TagNil:
					ctx.W.EmitMakeNil(d73)
				default:
					ctx.EmitMovPairToResult(&d72, &d73)
				}
			}
			ctx.W.EmitJmp(lbl0)
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
					ctx.W.EmitJmp(lbl6)
					return result
				}
				bbs[5].Rendered = true
				bbs[5].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_5 = bbs[5].Address
				ctx.W.MarkLabel(lbl6)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
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
			if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != scm.LocNone {
				d71 = ps.OverlayValues[71]
			}
			if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != scm.LocNone {
				d72 = ps.OverlayValues[72]
			}
			if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != scm.LocNone {
				d73 = ps.OverlayValues[73]
			}
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&idxInt)
			ctx.EnsureDesc(&idxInt)
			var d74 scm.JITValueDesc
			if idxInt.Loc == scm.LocImm {
				d74 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(int64(uint64(uint32(idxInt.Imm.Int()))))}
			} else {
				r65 := ctx.AllocReg()
				ctx.W.EmitMovRegReg(r65, idxInt.Reg)
				ctx.W.EmitShlRegImm8(r65, 32)
				ctx.W.EmitShrRegImm8(r65, 32)
				d74 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: r65}
				ctx.BindReg(r65, &d74)
			}
			ctx.FreeDesc(&idxInt)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d74)
			ctx.EnsureDesc(&d61)
			ctx.EnsureDesc(&d74)
			var d75 scm.JITValueDesc
			if d61.Loc == scm.LocImm && d74.Loc == scm.LocImm {
				d75 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagBool, Imm: scm.NewBool(uint64(d61.Imm.Int()) < uint64(d74.Imm.Int()))}
			} else if d74.Loc == scm.LocImm {
				r66 := ctx.AllocRegExcept(d61.Reg)
				if d74.Imm.Int() >= -2147483648 && d74.Imm.Int() <= 2147483647 {
					ctx.W.EmitCmpRegImm32(d61.Reg, int32(d74.Imm.Int()))
				} else {
					ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d74.Imm.Int()))
					ctx.W.EmitCmpInt64(d61.Reg, scm.RegR11)
				}
				ctx.W.EmitSetcc(r66, scm.CcB)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r66}
				ctx.BindReg(r66, &d75)
			} else if d61.Loc == scm.LocImm {
				r67 := ctx.AllocReg()
				ctx.W.EmitMovRegImm64(scm.RegR11, uint64(d61.Imm.Int()))
				ctx.W.EmitCmpInt64(scm.RegR11, d74.Reg)
				ctx.W.EmitSetcc(r67, scm.CcB)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r67}
				ctx.BindReg(r67, &d75)
			} else {
				r68 := ctx.AllocRegExcept(d61.Reg)
				ctx.W.EmitCmpInt64(d61.Reg, d74.Reg)
				ctx.W.EmitSetcc(r68, scm.CcB)
				d75 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagBool, Reg: r68}
				ctx.BindReg(r68, &d75)
			}
			ctx.FreeDesc(&d61)
			ctx.FreeDesc(&d74)
			d76 = d75
			ctx.EnsureDesc(&d76)
			if d76.Loc != scm.LocImm && d76.Loc != scm.LocReg {
				panic("jit: If condition is neither scm.LocImm nor scm.LocReg")
			}
			if d76.Loc == scm.LocImm {
				if d76.Imm.Bool() {
			ps77 := scm.PhiState{General: ps.General}
			ps77.OverlayValues = make([]scm.JITValueDesc, 77)
			ps77.OverlayValues[0] = d0
			ps77.OverlayValues[1] = d1
			ps77.OverlayValues[2] = d2
			ps77.OverlayValues[3] = d3
			ps77.OverlayValues[4] = d4
			ps77.OverlayValues[5] = d5
			ps77.OverlayValues[7] = d7
			ps77.OverlayValues[8] = d8
			ps77.OverlayValues[9] = d9
			ps77.OverlayValues[10] = d10
			ps77.OverlayValues[11] = d11
			ps77.OverlayValues[12] = d12
			ps77.OverlayValues[18] = d18
			ps77.OverlayValues[19] = d19
			ps77.OverlayValues[20] = d20
			ps77.OverlayValues[21] = d21
			ps77.OverlayValues[22] = d22
			ps77.OverlayValues[23] = d23
			ps77.OverlayValues[24] = d24
			ps77.OverlayValues[25] = d25
			ps77.OverlayValues[26] = d26
			ps77.OverlayValues[27] = d27
			ps77.OverlayValues[28] = d28
			ps77.OverlayValues[29] = d29
			ps77.OverlayValues[30] = d30
			ps77.OverlayValues[31] = d31
			ps77.OverlayValues[32] = d32
			ps77.OverlayValues[33] = d33
			ps77.OverlayValues[34] = d34
			ps77.OverlayValues[35] = d35
			ps77.OverlayValues[36] = d36
			ps77.OverlayValues[37] = d37
			ps77.OverlayValues[38] = d38
			ps77.OverlayValues[39] = d39
			ps77.OverlayValues[40] = d40
			ps77.OverlayValues[41] = d41
			ps77.OverlayValues[42] = d42
			ps77.OverlayValues[43] = d43
			ps77.OverlayValues[44] = d44
			ps77.OverlayValues[45] = d45
			ps77.OverlayValues[46] = d46
			ps77.OverlayValues[47] = d47
			ps77.OverlayValues[48] = d48
			ps77.OverlayValues[49] = d49
			ps77.OverlayValues[50] = d50
			ps77.OverlayValues[51] = d51
			ps77.OverlayValues[52] = d52
			ps77.OverlayValues[53] = d53
			ps77.OverlayValues[54] = d54
			ps77.OverlayValues[55] = d55
			ps77.OverlayValues[56] = d56
			ps77.OverlayValues[57] = d57
			ps77.OverlayValues[58] = d58
			ps77.OverlayValues[59] = d59
			ps77.OverlayValues[60] = d60
			ps77.OverlayValues[61] = d61
			ps77.OverlayValues[62] = d62
			ps77.OverlayValues[63] = d63
			ps77.OverlayValues[64] = d64
			ps77.OverlayValues[71] = d71
			ps77.OverlayValues[72] = d72
			ps77.OverlayValues[73] = d73
			ps77.OverlayValues[74] = d74
			ps77.OverlayValues[75] = d75
			ps77.OverlayValues[76] = d76
					return bbs[6].RenderPS(ps77)
				}
			ps78 := scm.PhiState{General: ps.General}
			ps78.OverlayValues = make([]scm.JITValueDesc, 77)
			ps78.OverlayValues[0] = d0
			ps78.OverlayValues[1] = d1
			ps78.OverlayValues[2] = d2
			ps78.OverlayValues[3] = d3
			ps78.OverlayValues[4] = d4
			ps78.OverlayValues[5] = d5
			ps78.OverlayValues[7] = d7
			ps78.OverlayValues[8] = d8
			ps78.OverlayValues[9] = d9
			ps78.OverlayValues[10] = d10
			ps78.OverlayValues[11] = d11
			ps78.OverlayValues[12] = d12
			ps78.OverlayValues[18] = d18
			ps78.OverlayValues[19] = d19
			ps78.OverlayValues[20] = d20
			ps78.OverlayValues[21] = d21
			ps78.OverlayValues[22] = d22
			ps78.OverlayValues[23] = d23
			ps78.OverlayValues[24] = d24
			ps78.OverlayValues[25] = d25
			ps78.OverlayValues[26] = d26
			ps78.OverlayValues[27] = d27
			ps78.OverlayValues[28] = d28
			ps78.OverlayValues[29] = d29
			ps78.OverlayValues[30] = d30
			ps78.OverlayValues[31] = d31
			ps78.OverlayValues[32] = d32
			ps78.OverlayValues[33] = d33
			ps78.OverlayValues[34] = d34
			ps78.OverlayValues[35] = d35
			ps78.OverlayValues[36] = d36
			ps78.OverlayValues[37] = d37
			ps78.OverlayValues[38] = d38
			ps78.OverlayValues[39] = d39
			ps78.OverlayValues[40] = d40
			ps78.OverlayValues[41] = d41
			ps78.OverlayValues[42] = d42
			ps78.OverlayValues[43] = d43
			ps78.OverlayValues[44] = d44
			ps78.OverlayValues[45] = d45
			ps78.OverlayValues[46] = d46
			ps78.OverlayValues[47] = d47
			ps78.OverlayValues[48] = d48
			ps78.OverlayValues[49] = d49
			ps78.OverlayValues[50] = d50
			ps78.OverlayValues[51] = d51
			ps78.OverlayValues[52] = d52
			ps78.OverlayValues[53] = d53
			ps78.OverlayValues[54] = d54
			ps78.OverlayValues[55] = d55
			ps78.OverlayValues[56] = d56
			ps78.OverlayValues[57] = d57
			ps78.OverlayValues[58] = d58
			ps78.OverlayValues[59] = d59
			ps78.OverlayValues[60] = d60
			ps78.OverlayValues[61] = d61
			ps78.OverlayValues[62] = d62
			ps78.OverlayValues[63] = d63
			ps78.OverlayValues[64] = d64
			ps78.OverlayValues[71] = d71
			ps78.OverlayValues[72] = d72
			ps78.OverlayValues[73] = d73
			ps78.OverlayValues[74] = d74
			ps78.OverlayValues[75] = d75
			ps78.OverlayValues[76] = d76
				return bbs[7].RenderPS(ps78)
			}
			lbl21 := ctx.W.ReserveLabel()
			lbl22 := ctx.W.ReserveLabel()
			ctx.W.EmitCmpRegImm32(d76.Reg, 0)
			ctx.W.EmitJcc(scm.CcNE, lbl21)
			ctx.W.EmitJmp(lbl22)
			ctx.W.MarkLabel(lbl21)
			ctx.W.EmitJmp(lbl7)
			ctx.W.MarkLabel(lbl22)
			ctx.W.EmitJmp(lbl8)
			ps79 := scm.PhiState{General: true}
			ps79.OverlayValues = make([]scm.JITValueDesc, 77)
			ps79.OverlayValues[0] = d0
			ps79.OverlayValues[1] = d1
			ps79.OverlayValues[2] = d2
			ps79.OverlayValues[3] = d3
			ps79.OverlayValues[4] = d4
			ps79.OverlayValues[5] = d5
			ps79.OverlayValues[7] = d7
			ps79.OverlayValues[8] = d8
			ps79.OverlayValues[9] = d9
			ps79.OverlayValues[10] = d10
			ps79.OverlayValues[11] = d11
			ps79.OverlayValues[12] = d12
			ps79.OverlayValues[18] = d18
			ps79.OverlayValues[19] = d19
			ps79.OverlayValues[20] = d20
			ps79.OverlayValues[21] = d21
			ps79.OverlayValues[22] = d22
			ps79.OverlayValues[23] = d23
			ps79.OverlayValues[24] = d24
			ps79.OverlayValues[25] = d25
			ps79.OverlayValues[26] = d26
			ps79.OverlayValues[27] = d27
			ps79.OverlayValues[28] = d28
			ps79.OverlayValues[29] = d29
			ps79.OverlayValues[30] = d30
			ps79.OverlayValues[31] = d31
			ps79.OverlayValues[32] = d32
			ps79.OverlayValues[33] = d33
			ps79.OverlayValues[34] = d34
			ps79.OverlayValues[35] = d35
			ps79.OverlayValues[36] = d36
			ps79.OverlayValues[37] = d37
			ps79.OverlayValues[38] = d38
			ps79.OverlayValues[39] = d39
			ps79.OverlayValues[40] = d40
			ps79.OverlayValues[41] = d41
			ps79.OverlayValues[42] = d42
			ps79.OverlayValues[43] = d43
			ps79.OverlayValues[44] = d44
			ps79.OverlayValues[45] = d45
			ps79.OverlayValues[46] = d46
			ps79.OverlayValues[47] = d47
			ps79.OverlayValues[48] = d48
			ps79.OverlayValues[49] = d49
			ps79.OverlayValues[50] = d50
			ps79.OverlayValues[51] = d51
			ps79.OverlayValues[52] = d52
			ps79.OverlayValues[53] = d53
			ps79.OverlayValues[54] = d54
			ps79.OverlayValues[55] = d55
			ps79.OverlayValues[56] = d56
			ps79.OverlayValues[57] = d57
			ps79.OverlayValues[58] = d58
			ps79.OverlayValues[59] = d59
			ps79.OverlayValues[60] = d60
			ps79.OverlayValues[61] = d61
			ps79.OverlayValues[62] = d62
			ps79.OverlayValues[63] = d63
			ps79.OverlayValues[64] = d64
			ps79.OverlayValues[71] = d71
			ps79.OverlayValues[72] = d72
			ps79.OverlayValues[73] = d73
			ps79.OverlayValues[74] = d74
			ps79.OverlayValues[75] = d75
			ps79.OverlayValues[76] = d76
			ps80 := scm.PhiState{General: true}
			ps80.OverlayValues = make([]scm.JITValueDesc, 77)
			ps80.OverlayValues[0] = d0
			ps80.OverlayValues[1] = d1
			ps80.OverlayValues[2] = d2
			ps80.OverlayValues[3] = d3
			ps80.OverlayValues[4] = d4
			ps80.OverlayValues[5] = d5
			ps80.OverlayValues[7] = d7
			ps80.OverlayValues[8] = d8
			ps80.OverlayValues[9] = d9
			ps80.OverlayValues[10] = d10
			ps80.OverlayValues[11] = d11
			ps80.OverlayValues[12] = d12
			ps80.OverlayValues[18] = d18
			ps80.OverlayValues[19] = d19
			ps80.OverlayValues[20] = d20
			ps80.OverlayValues[21] = d21
			ps80.OverlayValues[22] = d22
			ps80.OverlayValues[23] = d23
			ps80.OverlayValues[24] = d24
			ps80.OverlayValues[25] = d25
			ps80.OverlayValues[26] = d26
			ps80.OverlayValues[27] = d27
			ps80.OverlayValues[28] = d28
			ps80.OverlayValues[29] = d29
			ps80.OverlayValues[30] = d30
			ps80.OverlayValues[31] = d31
			ps80.OverlayValues[32] = d32
			ps80.OverlayValues[33] = d33
			ps80.OverlayValues[34] = d34
			ps80.OverlayValues[35] = d35
			ps80.OverlayValues[36] = d36
			ps80.OverlayValues[37] = d37
			ps80.OverlayValues[38] = d38
			ps80.OverlayValues[39] = d39
			ps80.OverlayValues[40] = d40
			ps80.OverlayValues[41] = d41
			ps80.OverlayValues[42] = d42
			ps80.OverlayValues[43] = d43
			ps80.OverlayValues[44] = d44
			ps80.OverlayValues[45] = d45
			ps80.OverlayValues[46] = d46
			ps80.OverlayValues[47] = d47
			ps80.OverlayValues[48] = d48
			ps80.OverlayValues[49] = d49
			ps80.OverlayValues[50] = d50
			ps80.OverlayValues[51] = d51
			ps80.OverlayValues[52] = d52
			ps80.OverlayValues[53] = d53
			ps80.OverlayValues[54] = d54
			ps80.OverlayValues[55] = d55
			ps80.OverlayValues[56] = d56
			ps80.OverlayValues[57] = d57
			ps80.OverlayValues[58] = d58
			ps80.OverlayValues[59] = d59
			ps80.OverlayValues[60] = d60
			ps80.OverlayValues[61] = d61
			ps80.OverlayValues[62] = d62
			ps80.OverlayValues[63] = d63
			ps80.OverlayValues[64] = d64
			ps80.OverlayValues[71] = d71
			ps80.OverlayValues[72] = d72
			ps80.OverlayValues[73] = d73
			ps80.OverlayValues[74] = d74
			ps80.OverlayValues[75] = d75
			ps80.OverlayValues[76] = d76
			snap81 := d20
			alloc82 := ctx.SnapshotAllocState()
			if !bbs[7].Rendered {
				bbs[7].RenderPS(ps80)
			}
			ctx.RestoreAllocState(alloc82)
			d20 = snap81
			if !bbs[6].Rendered {
				return bbs[6].RenderPS(ps79)
			}
			return result
			ctx.FreeDesc(&d75)
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
					ctx.W.EmitJmp(lbl7)
					return result
				}
				bbs[6].Rendered = true
				bbs[6].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_6 = bbs[6].Address
				ctx.W.MarkLabel(lbl7)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
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
			ctx.ReclaimUntrackedRegs()
			ctx.EnsureDesc(&d20)
			ctx.EnsureDesc(&d20)
			var d83 scm.JITValueDesc
			if d20.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: scm.TagInt, Imm: scm.NewInt(d20.Imm.Int() + 1)}
			} else {
				scratch := ctx.AllocRegExcept(d20.Reg)
				ctx.W.EmitMovRegReg(scratch, d20.Reg)
				ctx.W.EmitAddRegImm32(scratch, int32(1))
				d83 = scm.JITValueDesc{Loc: scm.LocReg, Type: scm.TagInt, Reg: scratch}
				ctx.BindReg(scratch, &d83)
			}
			if d83.Loc == scm.LocImm {
				d83 = scm.JITValueDesc{Loc: scm.LocImm, Type: d83.Type, Imm: scm.NewInt(int64(uint64(d83.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d83.Reg, 32)
				ctx.W.EmitShrRegImm8(d83.Reg, 32)
			}
			if d83.Loc == scm.LocReg && d20.Loc == scm.LocReg && d83.Reg == d20.Reg {
				ctx.TransferReg(d20.Reg)
				d20.Loc = scm.LocNone
			}
			d84 = d83
			if d84.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d84)
			d85 = d84
			if d85.Loc == scm.LocImm {
				d85 = scm.JITValueDesc{Loc: scm.LocImm, Type: d85.Type, Imm: scm.NewInt(int64(uint64(d85.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d85.Reg, 32)
				ctx.W.EmitShrRegImm8(d85.Reg, 32)
			}
			ctx.EmitStoreToStack(d85, 0)
			d86 = d1
			if d86.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d86)
			d87 = d86
			if d87.Loc == scm.LocImm {
				d87 = scm.JITValueDesc{Loc: scm.LocImm, Type: d87.Type, Imm: scm.NewInt(int64(uint64(d87.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d87.Reg, 32)
				ctx.W.EmitShrRegImm8(d87.Reg, 32)
			}
			ctx.EmitStoreToStack(d87, 8)
			ps88 := scm.PhiState{General: ps.General}
			ps88.OverlayValues = make([]scm.JITValueDesc, 88)
			ps88.OverlayValues[0] = d0
			ps88.OverlayValues[1] = d1
			ps88.OverlayValues[2] = d2
			ps88.OverlayValues[3] = d3
			ps88.OverlayValues[4] = d4
			ps88.OverlayValues[5] = d5
			ps88.OverlayValues[7] = d7
			ps88.OverlayValues[8] = d8
			ps88.OverlayValues[9] = d9
			ps88.OverlayValues[10] = d10
			ps88.OverlayValues[11] = d11
			ps88.OverlayValues[12] = d12
			ps88.OverlayValues[18] = d18
			ps88.OverlayValues[19] = d19
			ps88.OverlayValues[20] = d20
			ps88.OverlayValues[21] = d21
			ps88.OverlayValues[22] = d22
			ps88.OverlayValues[23] = d23
			ps88.OverlayValues[24] = d24
			ps88.OverlayValues[25] = d25
			ps88.OverlayValues[26] = d26
			ps88.OverlayValues[27] = d27
			ps88.OverlayValues[28] = d28
			ps88.OverlayValues[29] = d29
			ps88.OverlayValues[30] = d30
			ps88.OverlayValues[31] = d31
			ps88.OverlayValues[32] = d32
			ps88.OverlayValues[33] = d33
			ps88.OverlayValues[34] = d34
			ps88.OverlayValues[35] = d35
			ps88.OverlayValues[36] = d36
			ps88.OverlayValues[37] = d37
			ps88.OverlayValues[38] = d38
			ps88.OverlayValues[39] = d39
			ps88.OverlayValues[40] = d40
			ps88.OverlayValues[41] = d41
			ps88.OverlayValues[42] = d42
			ps88.OverlayValues[43] = d43
			ps88.OverlayValues[44] = d44
			ps88.OverlayValues[45] = d45
			ps88.OverlayValues[46] = d46
			ps88.OverlayValues[47] = d47
			ps88.OverlayValues[48] = d48
			ps88.OverlayValues[49] = d49
			ps88.OverlayValues[50] = d50
			ps88.OverlayValues[51] = d51
			ps88.OverlayValues[52] = d52
			ps88.OverlayValues[53] = d53
			ps88.OverlayValues[54] = d54
			ps88.OverlayValues[55] = d55
			ps88.OverlayValues[56] = d56
			ps88.OverlayValues[57] = d57
			ps88.OverlayValues[58] = d58
			ps88.OverlayValues[59] = d59
			ps88.OverlayValues[60] = d60
			ps88.OverlayValues[61] = d61
			ps88.OverlayValues[62] = d62
			ps88.OverlayValues[63] = d63
			ps88.OverlayValues[64] = d64
			ps88.OverlayValues[71] = d71
			ps88.OverlayValues[72] = d72
			ps88.OverlayValues[73] = d73
			ps88.OverlayValues[74] = d74
			ps88.OverlayValues[75] = d75
			ps88.OverlayValues[76] = d76
			ps88.OverlayValues[83] = d83
			ps88.OverlayValues[84] = d84
			ps88.OverlayValues[85] = d85
			ps88.OverlayValues[86] = d86
			ps88.OverlayValues[87] = d87
			ps88.PhiValues = make([]scm.JITValueDesc, 2)
			d89 = d83
			ps88.PhiValues[0] = d89
			d90 = d1
			ps88.PhiValues[1] = d90
			if ps88.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps88)
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
					ctx.W.EmitJmp(lbl8)
					return result
				}
				bbs[7].Rendered = true
				bbs[7].Address = int32(uintptr(ctx.W.Ptr) - uintptr(ctx.W.Start))
				bbpos_0_7 = bbs[7].Address
				ctx.W.MarkLabel(lbl8)
				ctx.W.ResolveFixups()
			}
			d0 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(0)}
			d1 = scm.JITValueDesc{Loc: scm.LocStack, Type: scm.JITTypeUnknown, StackOff: int32(8)}
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
			if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != scm.LocNone {
				d18 = ps.OverlayValues[18]
			}
			if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != scm.LocNone {
				d19 = ps.OverlayValues[19]
			}
			if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != scm.LocNone {
				d20 = ps.OverlayValues[20]
			}
			if len(ps.OverlayValues) > 21 && ps.OverlayValues[21].Loc != scm.LocNone {
				d21 = ps.OverlayValues[21]
			}
			if len(ps.OverlayValues) > 22 && ps.OverlayValues[22].Loc != scm.LocNone {
				d22 = ps.OverlayValues[22]
			}
			if len(ps.OverlayValues) > 23 && ps.OverlayValues[23].Loc != scm.LocNone {
				d23 = ps.OverlayValues[23]
			}
			if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != scm.LocNone {
				d24 = ps.OverlayValues[24]
			}
			if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != scm.LocNone {
				d25 = ps.OverlayValues[25]
			}
			if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != scm.LocNone {
				d26 = ps.OverlayValues[26]
			}
			if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != scm.LocNone {
				d27 = ps.OverlayValues[27]
			}
			if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != scm.LocNone {
				d28 = ps.OverlayValues[28]
			}
			if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != scm.LocNone {
				d29 = ps.OverlayValues[29]
			}
			if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != scm.LocNone {
				d30 = ps.OverlayValues[30]
			}
			if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != scm.LocNone {
				d31 = ps.OverlayValues[31]
			}
			if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != scm.LocNone {
				d32 = ps.OverlayValues[32]
			}
			if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != scm.LocNone {
				d33 = ps.OverlayValues[33]
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
			if len(ps.OverlayValues) > 83 && ps.OverlayValues[83].Loc != scm.LocNone {
				d83 = ps.OverlayValues[83]
			}
			if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != scm.LocNone {
				d84 = ps.OverlayValues[84]
			}
			if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != scm.LocNone {
				d85 = ps.OverlayValues[85]
			}
			if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != scm.LocNone {
				d86 = ps.OverlayValues[86]
			}
			if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != scm.LocNone {
				d87 = ps.OverlayValues[87]
			}
			if len(ps.OverlayValues) > 89 && ps.OverlayValues[89].Loc != scm.LocNone {
				d89 = ps.OverlayValues[89]
			}
			if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != scm.LocNone {
				d90 = ps.OverlayValues[90]
			}
			ctx.ReclaimUntrackedRegs()
			d91 = d0
			if d91.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d91)
			d92 = d91
			if d92.Loc == scm.LocImm {
				d92 = scm.JITValueDesc{Loc: scm.LocImm, Type: d92.Type, Imm: scm.NewInt(int64(uint64(d92.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d92.Reg, 32)
				ctx.W.EmitShrRegImm8(d92.Reg, 32)
			}
			ctx.EmitStoreToStack(d92, 0)
			d93 = d20
			if d93.Loc == scm.LocNone { panic("jit: phi source has no location") }
			ctx.EnsureDesc(&d93)
			d94 = d93
			if d94.Loc == scm.LocImm {
				d94 = scm.JITValueDesc{Loc: scm.LocImm, Type: d94.Type, Imm: scm.NewInt(int64(uint64(d94.Imm.Int()) & 0xffffffff))}
			} else {
				ctx.W.EmitShlRegImm8(d94.Reg, 32)
				ctx.W.EmitShrRegImm8(d94.Reg, 32)
			}
			ctx.EmitStoreToStack(d94, 8)
			ps95 := scm.PhiState{General: ps.General}
			ps95.OverlayValues = make([]scm.JITValueDesc, 95)
			ps95.OverlayValues[0] = d0
			ps95.OverlayValues[1] = d1
			ps95.OverlayValues[2] = d2
			ps95.OverlayValues[3] = d3
			ps95.OverlayValues[4] = d4
			ps95.OverlayValues[5] = d5
			ps95.OverlayValues[7] = d7
			ps95.OverlayValues[8] = d8
			ps95.OverlayValues[9] = d9
			ps95.OverlayValues[10] = d10
			ps95.OverlayValues[11] = d11
			ps95.OverlayValues[12] = d12
			ps95.OverlayValues[18] = d18
			ps95.OverlayValues[19] = d19
			ps95.OverlayValues[20] = d20
			ps95.OverlayValues[21] = d21
			ps95.OverlayValues[22] = d22
			ps95.OverlayValues[23] = d23
			ps95.OverlayValues[24] = d24
			ps95.OverlayValues[25] = d25
			ps95.OverlayValues[26] = d26
			ps95.OverlayValues[27] = d27
			ps95.OverlayValues[28] = d28
			ps95.OverlayValues[29] = d29
			ps95.OverlayValues[30] = d30
			ps95.OverlayValues[31] = d31
			ps95.OverlayValues[32] = d32
			ps95.OverlayValues[33] = d33
			ps95.OverlayValues[34] = d34
			ps95.OverlayValues[35] = d35
			ps95.OverlayValues[36] = d36
			ps95.OverlayValues[37] = d37
			ps95.OverlayValues[38] = d38
			ps95.OverlayValues[39] = d39
			ps95.OverlayValues[40] = d40
			ps95.OverlayValues[41] = d41
			ps95.OverlayValues[42] = d42
			ps95.OverlayValues[43] = d43
			ps95.OverlayValues[44] = d44
			ps95.OverlayValues[45] = d45
			ps95.OverlayValues[46] = d46
			ps95.OverlayValues[47] = d47
			ps95.OverlayValues[48] = d48
			ps95.OverlayValues[49] = d49
			ps95.OverlayValues[50] = d50
			ps95.OverlayValues[51] = d51
			ps95.OverlayValues[52] = d52
			ps95.OverlayValues[53] = d53
			ps95.OverlayValues[54] = d54
			ps95.OverlayValues[55] = d55
			ps95.OverlayValues[56] = d56
			ps95.OverlayValues[57] = d57
			ps95.OverlayValues[58] = d58
			ps95.OverlayValues[59] = d59
			ps95.OverlayValues[60] = d60
			ps95.OverlayValues[61] = d61
			ps95.OverlayValues[62] = d62
			ps95.OverlayValues[63] = d63
			ps95.OverlayValues[64] = d64
			ps95.OverlayValues[71] = d71
			ps95.OverlayValues[72] = d72
			ps95.OverlayValues[73] = d73
			ps95.OverlayValues[74] = d74
			ps95.OverlayValues[75] = d75
			ps95.OverlayValues[76] = d76
			ps95.OverlayValues[83] = d83
			ps95.OverlayValues[84] = d84
			ps95.OverlayValues[85] = d85
			ps95.OverlayValues[86] = d86
			ps95.OverlayValues[87] = d87
			ps95.OverlayValues[89] = d89
			ps95.OverlayValues[90] = d90
			ps95.OverlayValues[91] = d91
			ps95.OverlayValues[92] = d92
			ps95.OverlayValues[93] = d93
			ps95.OverlayValues[94] = d94
			ps95.PhiValues = make([]scm.JITValueDesc, 2)
			d96 = d0
			ps95.PhiValues[0] = d96
			d97 = d20
			ps95.PhiValues[1] = d97
			if ps95.General && bbs[1].Rendered {
				ctx.W.EmitJmp(lbl2)
				return result
			}
			return bbs[1].RenderPS(ps95)
			return result
			}
			ps98 := scm.PhiState{General: false}
			_ = bbs[0].RenderPS(ps98)
			ctx.W.MarkLabel(lbl0)
			d99 := scm.JITValueDesc{Loc: scm.LocRegPair, Reg: r1, Reg2: r2}
			ctx.BindReg(r1, &d99)
			ctx.BindReg(r2, &d99)
			ctx.EmitMovPairToResult(&d99, &result)
			ctx.FreeReg(r1)
			ctx.FreeReg(r2)
			ctx.W.ResolveFixups()
			if idxPinned { ctx.UnprotectReg(idxPinnedReg) }
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
