/*
Copyright (C) 2024-2026  Carl-Philip Hänsch

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
package scm

import (
	"bufio"
	"compress/gzip"
	"io"

	"github.com/ulikunitz/xz"
)

func init_streams() {
	// string functions
	DeclareTitle("Streams")

	Declare(&Globalenv, &Declaration{
		Name: "streamString",

		Fn: func(a ...Scmer) Scmer {
			return NewAny(a[0].Stream())
		},
		Type: &TypeDescriptor{Kind: "func", Description: "creates a stream that contains a string",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "string", Label: "content", Description: "content to put into the stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["streamString"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				if d0.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d0.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d0)
					} else if d0.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d0.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d0 = tmpPair
				} else if d0.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
					switch d0.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d0)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d0)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d0)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d0)
					d0 = tmpPair
				}
				if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value ((Scmer).Stream arg0)")
				}
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Stream), []JITValueDesc{d0}, 2)
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(jitReaderToAny), []JITValueDesc{d1}, 2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				d3 := d2
				_ = d3
				ctx.StabilizeDescForControlFlow(&d3)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				d4 := ctx.EmitGoCallScalar(GoFuncAddr(func() *any { return new(any) }), nil, 1)
				ctx.BindReg(d4.Reg, &d4)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d3)
				ctx.EmitGoCallVoid(GoFuncAddr(func(dst *any, value any) { *dst = value }), []JITValueDesc{d4, d3})
				ctx.ReclaimUntrackedRegs()
				r0 := ctx.AllocReg()
				r1 := ctx.AllocRegExcept(r0)
				ctx.EmitMovRegImm64(r0, 0)
				ctx.EmitMovRegImm64(r1, 0)
				d5 := JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r0, Reg2: r1}
				ctx.BindReg(r0, &d5)
				ctx.BindReg(r1, &d5)
				ctx.ReclaimUntrackedRegs()
				d6 := args[0]
				d6.ID = 0
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.ReclaimUntrackedRegs()
				d9 := args[0]
				d9.ID = 0
				ctx.ReclaimUntrackedRegs()
				d10 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(17)}
				d11 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
				d12 := d10
				_ = d12
				ctx.StabilizeDescForControlFlow(&d12)
				d13 := d11
				_ = d13
				ctx.StabilizeDescForControlFlow(&d13)
				bbpos_2_0 := int32(-1)
				_ = bbpos_2_0
				bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d13)
				var d14 JITValueDesc
				if d13.Loc == LocImm {
					d14 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d13.Imm.Int()) << 8))}
				} else {
					ctx.EmitShlRegImm8(d13.Reg, 8)
					d14 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d13.Reg}
					ctx.BindReg(d13.Reg, &d14)
				}
				if d14.Loc == LocReg && d13.Loc == LocReg && d14.Reg == d13.Reg {
					ctx.TransferReg(d13.Reg)
					d13.Loc = LocNone
				}
				ctx.FreeDesc(&d13)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d12)
				var d15 JITValueDesc
				if d12.Loc == LocImm {
					d15 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d12.Imm.Int() & 255)}
				} else {
					ctx.EmitAndRegImm32(d12.Reg, int32(255))
					d15 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d12.Reg}
					ctx.BindReg(d12.Reg, &d15)
				}
				if d15.Loc == LocImm {
					d15 = JITValueDesc{Loc: LocImm, Type: d15.Type, Imm: NewInt(int64(uint64(d15.Imm.Int()) & 0xff))}
				} else {
					ctx.EmitShlRegImm8(d15.Reg, 56)
					ctx.EmitShrRegImm8(d15.Reg, 56)
				}
				if d15.Loc == LocReg && d12.Loc == LocReg && d15.Reg == d12.Reg {
					ctx.TransferReg(d12.Reg)
					d12.Loc = LocNone
				}
				ctx.FreeDesc(&d12)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d15)
				ctx.EnsureDesc(&d15)
				var d16 JITValueDesc
				if d15.Loc == LocImm {
					d16 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d15.Imm.Int()))))}
				} else {
					r2 := ctx.AllocReg()
					ctx.EmitMovRegReg(r2, d15.Reg)
					ctx.EmitShlRegImm8(r2, 56)
					ctx.EmitShrRegImm8(r2, 56)
					d16 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
					ctx.BindReg(r2, &d16)
				}
				ctx.FreeDesc(&d15)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d14)
				ctx.EnsureDesc(&d16)
				var d17 JITValueDesc
				if d14.Loc == LocImm && d16.Loc == LocImm {
					d17 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d14.Imm.Int() | d16.Imm.Int())}
				} else if d14.Loc == LocImm && d14.Imm.Int() == 0 {
					d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d16.Reg}
					ctx.BindReg(d16.Reg, &d17)
				} else if d16.Loc == LocImm && d16.Imm.Int() == 0 {
					d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d17)
				} else if d14.Loc == LocImm {
					scratch := ctx.AllocRegExcept(d16.Reg)
					ctx.EmitMovRegImm64(scratch, uint64(d14.Imm.Int()))
					ctx.EmitOrInt64(scratch, d16.Reg)
					d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
					ctx.BindReg(scratch, &d17)
				} else if d16.Loc == LocImm {
					if d16.Imm.Int() >= -2147483648 && d16.Imm.Int() <= 2147483647 {
						ctx.EmitOrRegImm32(d14.Reg, int32(d16.Imm.Int()))
					} else {
						ctx.EmitMovRegImm64(RegR11, uint64(d16.Imm.Int()))
						ctx.EmitOrInt64(d14.Reg, RegR11)
					}
					d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d17)
				} else {
					ctx.EmitOrInt64(d14.Reg, d16.Reg)
					d17 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d14.Reg}
					ctx.BindReg(d14.Reg, &d17)
				}
				if d17.Loc == LocReg && d14.Loc == LocReg && d17.Reg == d14.Reg {
					ctx.TransferReg(d14.Reg)
					d14.Loc = LocNone
				}
				ctx.FreeDesc(&d14)
				ctx.FreeDesc(&d16)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d17)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EmitMovToReg(d6.Reg, d4)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d17)
				ctx.EnsureDesc(&d17)
				ctx.EmitMovToReg(d9.Reg2, d17)
				ctx.FreeDesc(&d17)
				ctx.ReclaimUntrackedRegs()
				d18 := d5
				_ = d18
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d18)
				if d18.Loc == LocImm {
					if result.Loc == LocAny {
						return d18
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d18)
				if d18.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d18, &result)
					result.Type = d18.Type
				} else {
					switch d18.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d18)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d18)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d18)
						result.Type = tagFloat
					case tagNil:
						ctx.EmitMakeNil(result)
						result.Type = tagNil
					default:
						panic("jit: single-block scalar return with unknown type")
					}
				}
				return result
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  6,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "gzip",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("gzip expects a stream")
			}
			reader, writer := io.Pipe()
			bwriter := bufio.NewWriterSize(writer, 16*1024)
			zip := gzip.NewWriter(bwriter)
			go func() {
				io.Copy(zip, stream)
				zip.Close()
				bwriter.Flush()
				writer.Close()
			}()
			return NewAny(io.Reader(reader))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compresses a stream with gzip. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["gzip"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "xz",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("xz expects a stream")
			}
			reader, writer := io.Pipe()
			bwriter := bufio.NewWriterSize(writer, 16*1024)
			zip, err := xz.NewWriter(bwriter)
			go func() {
				io.Copy(zip, stream)
				zip.Close()
				bwriter.Flush()
				writer.Close()
			}()
			if err != nil {
				panic(err)
			}
			return NewAny(io.Reader(reader))
		},
		Type: &TypeDescriptor{Kind: "func", Description: "compresses a stream with xz. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["xz"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "zcat",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("zcat expects a stream")
			}
			reader, err := gzip.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a compressed gzip stream into a stream of uncompressed data. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["zcat"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [5]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
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
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					ctx.EnsureDesc(&d0)
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d0.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d0.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d0 = tmpPair
					} else if d0.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
						switch d0.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d0)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d0)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d0)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d0)
						d0 = tmpPair
					}
					if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((Scmer).Any arg0)")
					}
					ctx.SyncDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d0}, 2)
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d1)
					callResults2 := JITEmitGoCallResults(ctx, GoFuncAddr(jitAssertReader), []JITValueDesc{d1}, []uint8{2, 1}, []uint8{3, 0})
					d3 = callResults2[0]
					d4 = callResults2[1]
					_ = d3
					_ = d4
					ctx.EmitAndRegImm32(d4.Reg, 1)
					d4.Type = tagBool
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d3)
					d5 = d4
					ctx.EnsureDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							if ps.General {
							}
							ps6 := PhiState{General: ps.General}
							ps6.OverlayValues = make([]JITValueDesc, 6)
							ps6.OverlayValues[0] = d0
							ps6.OverlayValues[1] = d1
							ps6.OverlayValues[3] = d3
							ps6.OverlayValues[4] = d4
							ps6.OverlayValues[5] = d5
							return bbs[2].RenderPS(ps6)
						}
						if ps.General {
						}
						ps7 := PhiState{General: ps.General}
						ps7.OverlayValues = make([]JITValueDesc, 6)
						ps7.OverlayValues[0] = d0
						ps7.OverlayValues[1] = d1
						ps7.OverlayValues[3] = d3
						ps7.OverlayValues[4] = d4
						ps7.OverlayValues[5] = d5
						return bbs[1].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 6)
					ps8.OverlayValues[0] = d0
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					ps8.OverlayValues[5] = d5
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 6)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					snap10 := d0
					snap11 := d1
					snap12 := d3
					snap13 := d4
					snap14 := d5
					alloc15 := ctx.SnapshotAllocState()
					if !bbs[1].Rendered {
						bbs[1].RenderPS(ps9)
					}
					ctx.RestoreAllocState(alloc15)
					d0 = snap10
					d1 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps8)
					}
					return result
					ctx.FreeDesc(&d4)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["zcat"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d3.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d3.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d3 = tmpPair
					} else if d3.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
						switch d3.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d3)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d3)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d3)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d3)
						d3 = tmpPair
					}
					if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (gzip.NewReader arg0)")
					}
					ctx.SyncDesc(&d3)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(gzip.NewReader), []JITValueDesc{d3}, []uint8{1, 2}, []uint8{1, 3})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.FreeDesc(&d3)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d18)
					var d19 JITValueDesc
					if d18.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d18)
						if d18.Loc != LocReg && d18.Loc != LocRegPair && d18.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[0] = d0
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[3].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[0] = d0
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[4].RenderPS(ps22)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 21)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[17] = d17
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					ps23.OverlayValues[20] = d20
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 21)
					ps24.OverlayValues[0] = d0
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[17] = d17
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					snap25 := d0
					snap26 := d1
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d17
					snap31 := d18
					snap32 := d19
					snap33 := d20
					alloc34 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps24)
					}
					ctx.RestoreAllocState(alloc34)
					d0 = snap25
					d1 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d17 = snap30
					d18 = snap31
					d19 = snap32
					d20 = snap33
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps23)
					}
					return result
					ctx.FreeDesc(&d19)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["zcat"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d17)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *gzip.Reader) any { return value }), []JITValueDesc{d17}, 2)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d35)
					d36 = d35
					_ = d36
					ctx.StabilizeDescForControlFlow(&d36)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(func() *any { return new(any) }), nil, 1)
					ctx.BindReg(d37.Reg, &d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *any, value any) { *dst = value }), []JITValueDesc{d37, d36})
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d38 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d38)
					ctx.BindReg(r2, &d38)
					ctx.ReclaimUntrackedRegs()
					d39 = args[0]
					d39.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.ReclaimUntrackedRegs()
					d42 = args[0]
					d42.ID = 0
					ctx.ReclaimUntrackedRegs()
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(17)}
					d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d45 = d43
					_ = d45
					ctx.StabilizeDescForControlFlow(&d45)
					d46 = d44
					_ = d46
					ctx.StabilizeDescForControlFlow(&d46)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					var d47 JITValueDesc
					if d46.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d46.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d46.Reg, 8)
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg}
						ctx.BindReg(d46.Reg, &d47)
					}
					if d47.Loc == LocReg && d46.Loc == LocReg && d47.Reg == d46.Reg {
						ctx.TransferReg(d46.Reg)
						d46.Loc = LocNone
					}
					ctx.FreeDesc(&d46)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d45)
					var d48 JITValueDesc
					if d45.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d45.Reg, int32(255))
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg}
						ctx.BindReg(d45.Reg, &d48)
					}
					if d48.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: d48.Type, Imm: NewInt(int64(uint64(d48.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d48.Reg, 56)
						ctx.EmitShrRegImm8(d48.Reg, 56)
					}
					if d48.Loc == LocReg && d45.Loc == LocReg && d48.Reg == d45.Reg {
						ctx.TransferReg(d45.Reg)
						d45.Loc = LocNone
					}
					ctx.FreeDesc(&d45)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					var d49 JITValueDesc
					if d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d48.Imm.Int()))))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegReg(r3, d48.Reg)
						ctx.EmitShlRegImm8(r3, 56)
						ctx.EmitShrRegImm8(r3, 56)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d49)
					}
					ctx.FreeDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d49)
					var d50 JITValueDesc
					if d47.Loc == LocImm && d49.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() | d49.Imm.Int())}
					} else if d47.Loc == LocImm && d47.Imm.Int() == 0 {
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d49.Reg}
						ctx.BindReg(d49.Reg, &d50)
					} else if d49.Loc == LocImm && d49.Imm.Int() == 0 {
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					} else if d47.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
						ctx.EmitOrInt64(scratch, d49.Reg)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d50)
					} else if d49.Loc == LocImm {
						if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d47.Reg, int32(d49.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
							ctx.EmitOrInt64(d47.Reg, RegR11)
						}
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					} else {
						ctx.EmitOrInt64(d47.Reg, d49.Reg)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					}
					if d50.Loc == LocReg && d47.Loc == LocReg && d50.Reg == d47.Reg {
						ctx.TransferReg(d47.Reg)
						d47.Loc = LocNone
					}
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d50)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.EmitMovToReg(d39.Reg, d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d50)
					ctx.EmitMovToReg(d42.Reg2, d50)
					ctx.FreeDesc(&d50)
					ctx.ReclaimUntrackedRegs()
					d51 = d38
					_ = d51
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d51, &result)
						result.Type = d51.Type
					} else {
						switch d51.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d51)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d51)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d51)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d51, &result)
							result.Type = d51.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps52 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps52)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
		},
	})
	Declare(&Globalenv, &Declaration{
		Name: "xzcat",

		Fn: func(a ...Scmer) Scmer {
			stream, ok := a[0].Any().(io.Reader)
			if !ok {
				panic("xzcat expects a stream")
			}
			reader, err := xz.NewReader(stream)
			if err != nil {
				panic(err)
			}
			return NewAny(reader)
		},
		Type: &TypeDescriptor{Kind: "func", Description: "turns a compressed xz stream into a stream of uncompressed data. Create streams with (stream filename)",
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "stream", Label: "stream", Description: "input stream"}},
			Return: &TypeDescriptor{Kind: "stream"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["xzcat"].Fn, args, result)
				}
				var d0 JITValueDesc
				_ = d0
				var d1 JITValueDesc
				_ = d1
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d17 JITValueDesc
				_ = d17
				var d18 JITValueDesc
				_ = d18
				var d19 JITValueDesc
				_ = d19
				var d20 JITValueDesc
				_ = d20
				var d35 JITValueDesc
				_ = d35
				var d36 JITValueDesc
				_ = d36
				var d37 JITValueDesc
				_ = d37
				var d38 JITValueDesc
				_ = d38
				var d39 JITValueDesc
				_ = d39
				var d40 JITValueDesc
				_ = d40
				var d41 JITValueDesc
				_ = d41
				var d42 JITValueDesc
				_ = d42
				var d43 JITValueDesc
				_ = d43
				var d44 JITValueDesc
				_ = d44
				var d45 JITValueDesc
				_ = d45
				var d46 JITValueDesc
				_ = d46
				var d47 JITValueDesc
				_ = d47
				var d48 JITValueDesc
				_ = d48
				var d49 JITValueDesc
				_ = d49
				var d50 JITValueDesc
				_ = d50
				var d51 JITValueDesc
				_ = d51
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				var bbs [5]BBDescriptor
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
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
				bbs[0].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[0].VisitCount >= 0 {
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
					d0 = args[0]
					d0.ID = 0
					ctx.EnsureDesc(&d0)
					ctx.EnsureDesc(&d0)
					ctx.EnsureDesc(&d0)
					if d0.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d0.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d0)
						} else if d0.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d0.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d0 = tmpPair
					} else if d0.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d0.Type, Reg: ctx.AllocRegExcept(d0.Reg), Reg2: ctx.AllocRegExcept(d0.Reg)}
						switch d0.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d0)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d0)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d0)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d0)
						d0 = tmpPair
					}
					if d0.Loc != LocRegPair && d0.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value ((Scmer).Any arg0)")
					}
					ctx.SyncDesc(&d0)
					d1 = ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Any), []JITValueDesc{d0}, 2)
					ctx.BindReg(d1.Reg, &d1)
					ctx.BindReg(d1.Reg2, &d1)
					ctx.FreeDesc(&d0)
					ctx.EnsureDesc(&d1)
					callResults2 := JITEmitGoCallResults(ctx, GoFuncAddr(jitAssertReader), []JITValueDesc{d1}, []uint8{2, 1}, []uint8{3, 0})
					d3 = callResults2[0]
					d4 = callResults2[1]
					_ = d3
					_ = d4
					ctx.EmitAndRegImm32(d4.Reg, 1)
					d4.Type = tagBool
					ctx.FreeDesc(&d1)
					ctx.StabilizeDescForControlFlow(&d3)
					d5 = d4
					ctx.EnsureDesc(&d5)
					if d5.Loc != LocImm && d5.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d5.Loc == LocImm {
						if d5.Imm.Bool() {
							if ps.General {
							}
							ps6 := PhiState{General: ps.General}
							ps6.OverlayValues = make([]JITValueDesc, 6)
							ps6.OverlayValues[0] = d0
							ps6.OverlayValues[1] = d1
							ps6.OverlayValues[3] = d3
							ps6.OverlayValues[4] = d4
							ps6.OverlayValues[5] = d5
							return bbs[2].RenderPS(ps6)
						}
						if ps.General {
						}
						ps7 := PhiState{General: ps.General}
						ps7.OverlayValues = make([]JITValueDesc, 6)
						ps7.OverlayValues[0] = d0
						ps7.OverlayValues[1] = d1
						ps7.OverlayValues[3] = d3
						ps7.OverlayValues[4] = d4
						ps7.OverlayValues[5] = d5
						return bbs[1].RenderPS(ps7)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl6 := ctx.ReserveLabel()
					lbl7 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d5.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl6)
					ctx.EmitJmp(lbl7)
					ctx.MarkLabel(lbl6)
					ctx.EmitJmp(lbl3)
					ctx.MarkLabel(lbl7)
					ctx.EmitJmp(lbl2)
					ps8 := PhiState{General: true}
					ps8.OverlayValues = make([]JITValueDesc, 6)
					ps8.OverlayValues[0] = d0
					ps8.OverlayValues[1] = d1
					ps8.OverlayValues[3] = d3
					ps8.OverlayValues[4] = d4
					ps8.OverlayValues[5] = d5
					ps9 := PhiState{General: true}
					ps9.OverlayValues = make([]JITValueDesc, 6)
					ps9.OverlayValues[0] = d0
					ps9.OverlayValues[1] = d1
					ps9.OverlayValues[3] = d3
					ps9.OverlayValues[4] = d4
					ps9.OverlayValues[5] = d5
					snap10 := d0
					snap11 := d1
					snap12 := d3
					snap13 := d4
					snap14 := d5
					alloc15 := ctx.SnapshotAllocState()
					if !bbs[1].Rendered {
						bbs[1].RenderPS(ps9)
					}
					ctx.RestoreAllocState(alloc15)
					d0 = snap10
					d1 = snap11
					d3 = snap12
					d4 = snap13
					d5 = snap14
					if !bbs[2].Rendered {
						return bbs[2].RenderPS(ps8)
					}
					return result
					ctx.FreeDesc(&d4)
					return result
				}
				bbs[1].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[1].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["xzcat"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[2].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[2].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d3.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d3)
						} else if d3.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d3.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d3 = tmpPair
					} else if d3.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d3.Type, Reg: ctx.AllocRegExcept(d3.Reg), Reg2: ctx.AllocRegExcept(d3.Reg)}
						switch d3.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d3)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d3)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d3)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d3)
						d3 = tmpPair
					}
					if d3.Loc != LocRegPair && d3.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (xz.NewReader arg0)")
					}
					ctx.SyncDesc(&d3)
					callResults16 := JITEmitGoCallResults(ctx, GoFuncAddr(xz.NewReader), []JITValueDesc{d3}, []uint8{1, 2}, []uint8{1, 3})
					d17 = callResults16[0]
					_ = d17
					d18 = callResults16[1]
					_ = d18
					ctx.FreeDesc(&d3)
					ctx.StabilizeDescForControlFlow(&d17)
					ctx.StabilizeDescForControlFlow(&d18)
					ctx.EnsureDesc(&d18)
					var d19 JITValueDesc
					if d18.Loc == LocImm {
						d19 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d18.Imm.IsNil() != true)}
					} else {
						ctx.EnsureDesc(&d18)
						if d18.Loc != LocReg && d18.Loc != LocRegPair && d18.Loc != LocRegTriple {
							panic("jit: nil comparison requires a register value")
						}
						r0 := ctx.AllocRegExcept(d18.Reg)
						ctx.EmitCmpRegImm32(d18.Reg, 0)
						ctx.EmitSetcc(r0, CondNotEqual)
						d19 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d19)
					}
					d20 = d19
					ctx.EnsureDesc(&d20)
					if d20.Loc != LocImm && d20.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d20.Loc == LocImm {
						if d20.Imm.Bool() {
							if ps.General {
							}
							ps21 := PhiState{General: ps.General}
							ps21.OverlayValues = make([]JITValueDesc, 21)
							ps21.OverlayValues[0] = d0
							ps21.OverlayValues[1] = d1
							ps21.OverlayValues[3] = d3
							ps21.OverlayValues[4] = d4
							ps21.OverlayValues[5] = d5
							ps21.OverlayValues[17] = d17
							ps21.OverlayValues[18] = d18
							ps21.OverlayValues[19] = d19
							ps21.OverlayValues[20] = d20
							return bbs[3].RenderPS(ps21)
						}
						if ps.General {
						}
						ps22 := PhiState{General: ps.General}
						ps22.OverlayValues = make([]JITValueDesc, 21)
						ps22.OverlayValues[0] = d0
						ps22.OverlayValues[1] = d1
						ps22.OverlayValues[3] = d3
						ps22.OverlayValues[4] = d4
						ps22.OverlayValues[5] = d5
						ps22.OverlayValues[17] = d17
						ps22.OverlayValues[18] = d18
						ps22.OverlayValues[19] = d19
						ps22.OverlayValues[20] = d20
						return bbs[4].RenderPS(ps22)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl8 := ctx.ReserveLabel()
					lbl9 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d20.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl8)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl8)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl9)
					ctx.EmitJmp(lbl5)
					ps23 := PhiState{General: true}
					ps23.OverlayValues = make([]JITValueDesc, 21)
					ps23.OverlayValues[0] = d0
					ps23.OverlayValues[1] = d1
					ps23.OverlayValues[3] = d3
					ps23.OverlayValues[4] = d4
					ps23.OverlayValues[5] = d5
					ps23.OverlayValues[17] = d17
					ps23.OverlayValues[18] = d18
					ps23.OverlayValues[19] = d19
					ps23.OverlayValues[20] = d20
					ps24 := PhiState{General: true}
					ps24.OverlayValues = make([]JITValueDesc, 21)
					ps24.OverlayValues[0] = d0
					ps24.OverlayValues[1] = d1
					ps24.OverlayValues[3] = d3
					ps24.OverlayValues[4] = d4
					ps24.OverlayValues[5] = d5
					ps24.OverlayValues[17] = d17
					ps24.OverlayValues[18] = d18
					ps24.OverlayValues[19] = d19
					ps24.OverlayValues[20] = d20
					snap25 := d0
					snap26 := d1
					snap27 := d3
					snap28 := d4
					snap29 := d5
					snap30 := d17
					snap31 := d18
					snap32 := d19
					snap33 := d20
					alloc34 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps24)
					}
					ctx.RestoreAllocState(alloc34)
					d0 = snap25
					d1 = snap26
					d3 = snap27
					d4 = snap28
					d5 = snap29
					d17 = snap30
					d18 = snap31
					d19 = snap32
					d20 = snap33
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps23)
					}
					return result
					ctx.FreeDesc(&d19)
					return result
				}
				bbs[3].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[3].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["xzcat"].Fn, args, result)
					ctx.EmitGoPanic("jit: builtin panic boundary unexpectedly returned")
					return result
				}
				bbs[4].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[4].VisitCount >= 0 {
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
					if len(ps.OverlayValues) > 0 && ps.OverlayValues[0].Loc != LocNone {
						d0 = ps.OverlayValues[0]
					}
					if len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 3 && ps.OverlayValues[3].Loc != LocNone {
						d3 = ps.OverlayValues[3]
					}
					if len(ps.OverlayValues) > 4 && ps.OverlayValues[4].Loc != LocNone {
						d4 = ps.OverlayValues[4]
					}
					if len(ps.OverlayValues) > 5 && ps.OverlayValues[5].Loc != LocNone {
						d5 = ps.OverlayValues[5]
					}
					if len(ps.OverlayValues) > 17 && ps.OverlayValues[17].Loc != LocNone {
						d17 = ps.OverlayValues[17]
					}
					if len(ps.OverlayValues) > 18 && ps.OverlayValues[18].Loc != LocNone {
						d18 = ps.OverlayValues[18]
					}
					if len(ps.OverlayValues) > 19 && ps.OverlayValues[19].Loc != LocNone {
						d19 = ps.OverlayValues[19]
					}
					if len(ps.OverlayValues) > 20 && ps.OverlayValues[20].Loc != LocNone {
						d20 = ps.OverlayValues[20]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d17)
					d35 = ctx.EmitGoCallScalar(GoFuncAddr(func(value *xz.Reader) any { return value }), []JITValueDesc{d17}, 2)
					ctx.FreeDesc(&d17)
					ctx.EnsureDesc(&d35)
					d36 = d35
					_ = d36
					ctx.StabilizeDescForControlFlow(&d36)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					d37 = ctx.EmitGoCallScalar(GoFuncAddr(func() *any { return new(any) }), nil, 1)
					ctx.BindReg(d37.Reg, &d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d36)
					ctx.EmitGoCallVoid(GoFuncAddr(func(dst *any, value any) { *dst = value }), []JITValueDesc{d37, d36})
					ctx.ReclaimUntrackedRegs()
					r1 := ctx.AllocReg()
					r2 := ctx.AllocRegExcept(r1)
					ctx.EmitMovRegImm64(r1, 0)
					ctx.EmitMovRegImm64(r2, 0)
					d38 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: r1, Reg2: r2}
					ctx.BindReg(r1, &d38)
					ctx.BindReg(r2, &d38)
					ctx.ReclaimUntrackedRegs()
					d39 = args[0]
					d39.ID = 0
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.ReclaimUntrackedRegs()
					d42 = args[0]
					d42.ID = 0
					ctx.ReclaimUntrackedRegs()
					d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(17)}
					d44 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d45 = d43
					_ = d45
					ctx.StabilizeDescForControlFlow(&d45)
					d46 = d44
					_ = d46
					ctx.StabilizeDescForControlFlow(&d46)
					bbpos_2_0 := int32(-1)
					_ = bbpos_2_0
					bbpos_2_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d46)
					var d47 JITValueDesc
					if d46.Loc == LocImm {
						d47 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(d46.Imm.Int()) << 8))}
					} else {
						ctx.EmitShlRegImm8(d46.Reg, 8)
						d47 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d46.Reg}
						ctx.BindReg(d46.Reg, &d47)
					}
					if d47.Loc == LocReg && d46.Loc == LocReg && d47.Reg == d46.Reg {
						ctx.TransferReg(d46.Reg)
						d46.Loc = LocNone
					}
					ctx.FreeDesc(&d46)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d45)
					var d48 JITValueDesc
					if d45.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d45.Imm.Int() & 255)}
					} else {
						ctx.EmitAndRegImm32(d45.Reg, int32(255))
						d48 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d45.Reg}
						ctx.BindReg(d45.Reg, &d48)
					}
					if d48.Loc == LocImm {
						d48 = JITValueDesc{Loc: LocImm, Type: d48.Type, Imm: NewInt(int64(uint64(d48.Imm.Int()) & 0xff))}
					} else {
						ctx.EmitShlRegImm8(d48.Reg, 56)
						ctx.EmitShrRegImm8(d48.Reg, 56)
					}
					if d48.Loc == LocReg && d45.Loc == LocReg && d48.Reg == d45.Reg {
						ctx.TransferReg(d45.Reg)
						d45.Loc = LocNone
					}
					ctx.FreeDesc(&d45)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d48)
					ctx.EnsureDesc(&d48)
					var d49 JITValueDesc
					if d48.Loc == LocImm {
						d49 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uint64(uint8(d48.Imm.Int()))))}
					} else {
						r3 := ctx.AllocReg()
						ctx.EmitMovRegReg(r3, d48.Reg)
						ctx.EmitShlRegImm8(r3, 56)
						ctx.EmitShrRegImm8(r3, 56)
						d49 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d49)
					}
					ctx.FreeDesc(&d48)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d47)
					ctx.EnsureDesc(&d49)
					var d50 JITValueDesc
					if d47.Loc == LocImm && d49.Loc == LocImm {
						d50 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d47.Imm.Int() | d49.Imm.Int())}
					} else if d47.Loc == LocImm && d47.Imm.Int() == 0 {
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d49.Reg}
						ctx.BindReg(d49.Reg, &d50)
					} else if d49.Loc == LocImm && d49.Imm.Int() == 0 {
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					} else if d47.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d49.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d47.Imm.Int()))
						ctx.EmitOrInt64(scratch, d49.Reg)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d50)
					} else if d49.Loc == LocImm {
						if d49.Imm.Int() >= -2147483648 && d49.Imm.Int() <= 2147483647 {
							ctx.EmitOrRegImm32(d47.Reg, int32(d49.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d49.Imm.Int()))
							ctx.EmitOrInt64(d47.Reg, RegR11)
						}
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					} else {
						ctx.EmitOrInt64(d47.Reg, d49.Reg)
						d50 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d47.Reg}
						ctx.BindReg(d47.Reg, &d50)
					}
					if d50.Loc == LocReg && d47.Loc == LocReg && d50.Reg == d47.Reg {
						ctx.TransferReg(d47.Reg)
						d47.Loc = LocNone
					}
					ctx.FreeDesc(&d47)
					ctx.FreeDesc(&d49)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d50)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.EmitMovToReg(d39.Reg, d37)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d50)
					ctx.EnsureDesc(&d50)
					ctx.EmitMovToReg(d42.Reg2, d50)
					ctx.FreeDesc(&d50)
					ctx.ReclaimUntrackedRegs()
					d51 = d38
					_ = d51
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d51)
					ctx.EnsureDesc(&d51)
					if d51.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d51, &result)
						result.Type = d51.Type
					} else {
						switch d51.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d51)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d51)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d51)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d51, &result)
							result.Type = d51.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps52 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps52)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  19,
		},
	})
}
