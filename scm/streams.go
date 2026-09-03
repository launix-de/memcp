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
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["streamString"].Fn, args, result)
				}
				declaration := declarations["streamString"]
				inline := declaration.RetainsCallArgs
				knownTypes, knownShapes, knownArgs := 0, 0, 0
				hasVirtualArgs := false
				knownCallback, hasCallback := false, false
				for index, arg := range args {
					if arg.Type != JITTypeUnknown {
						knownTypes++
					}
					hasKnownShape := arg.Loc == LocImm || arg.SliceSizeKnown || arg.Loc == LocVirtualSlice
					hasVirtualArgs = hasVirtualArgs || arg.Loc == LocVirtualSlice
					if hasKnownShape {
						knownShapes++
					}
					if arg.Type != JITTypeUnknown || hasKnownShape {
						knownArgs++
					}
					parameter := jitDeclarationParam(declaration, index)
					if parameter != nil && parameter.Kind == "func" {
						hasCallback = true
						if (arg.Loc == LocLambdaTemplate && arg.Lambda != nil) ||
							(arg.Loc == LocImm && (arg.Imm.GetTag() == tagProc || arg.Imm.GetTag() == tagFunc)) {
							knownCallback = true
						}
					}
				}
				cost := int(declaration.Type.JITInlineCost)
				if !inline && hasCallback {
					inline = declaration.Type.JITInlineCallbacks && knownCallback
				} else if !inline {
					switch {
					case declaration.Type.JITVirtualArgs && cost <= jitTrivialVirtualInlineCost && (jitDirectSliceBuilder(len(args)) != 0 || len(args) > 8):
						inline = true
					case declaration.Type.JITVirtualArgs && hasVirtualArgs && declaration.Type.JITInlineCost <= 32:
						inline = true
					case len(args) > 0 && knownTypes == len(args) && cost <= 256:
						inline = true
					case knownShapes == len(args) && knownArgs == len(args) && cost <= 32:
						inline = true
					}
					if declaration.Type.JITVirtualArgs && cost > jitTrivialVirtualInlineCost && !hasVirtualArgs && knownShapes != len(args) {
						inline = false
					}
					if declaration.Type.JITVirtualArgs && cost > 32 && knownShapes == 0 {
						inline = false
					}
				}
				if cost == 65535 || !declaration.RetainsCallArgs && ctx.BuiltinInlineCost+cost > jitBuiltinInlineBudget {
					inline = false
				}
				if !inline {
					ctx.Coverage.NativeCalls++
					return jitEmitGoVariadicCallFromDescs(ctx, declaration.Fn, args, result)
				}
				ctx.BuiltinInlineCost += cost
				ctx.Coverage.InlinedCalls++
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d0)
				d0 = JITPrepareScmerGoArg(ctx, d0)
				ctx.SyncDesc(&d0)
				d1 := ctx.EmitGoCallScalar(GoFuncAddr((Scmer).Stream), []JITValueDesc{d0}, 2)
				d1.NoHeapPointer = false
				ctx.BindReg(d1.Reg, &d1)
				ctx.BindReg(d1.Reg2, &d1)
				ctx.FreeDesc(&d0)
				ctx.EnsureDesc(&d1)
				d2 := ctx.EmitGoCallScalar(GoFuncAddr(jitReaderToAny), []JITValueDesc{d1}, 2)
				ctx.FreeDesc(&d1)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				ctx.EnsureDesc(&d2)
				if d2.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d2.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d2)
					} else if d2.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d2.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d2 = tmpPair
				} else if d2.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d2.Type, Reg: ctx.AllocRegExcept(d2.Reg), Reg2: ctx.AllocRegExcept(d2.Reg)}
					switch d2.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d2)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d2)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d2)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d2)
					d2 = tmpPair
				}
				if d2.Loc != LocRegPair && d2.Loc != LocStackPair && d2.Loc != LocInputPair {
					panic("jit: generic call arg expects 2-word value (NewAny arg0)")
				}
				ctx.SyncDesc(&d2)
				d3 := ctx.EmitGoCallScalar(GoFuncAddr(NewAny), []JITValueDesc{d2}, 2)
				d3.NoHeapPointer = false
				ctx.BindReg(d3.Reg, &d3)
				ctx.BindReg(d3.Reg2, &d3)
				if d3.Loc == LocImm {
					if result.Loc == LocAny {
						return d3
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.SyncDesc(&d3)
				if d3.Loc == LocRegPair || d3.Loc == LocStackPair || d3.Loc == LocInputPair {
					ctx.EmitMovPairToResult(&d3, &result)
					result.Type = d3.Type
				} else {
					switch d3.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d3)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d3)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d3)
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
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
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
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["zcat"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
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

			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: interface type assertion.
				ctx.Coverage.NativeCalls++
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["xzcat"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})
}
