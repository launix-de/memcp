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

package scm

import "sync"
import "unsafe"

/*
 Sliding window helpers for LEAD/LAG window functions.

 Accumulator layout (flat list):
   (skip_count counter stride slot_0_v0 slot_0_v1 ... slot_N_vM)

 - skip_count: rows to skip before first emit (LEAD offset, 0 for LAG)
 - counter: monotonic number of inserted positions
 - stride: number of values per slot
 - slots: window_size * stride values

 window_mut shifts vals into the caller-owned window, increments counter,
 and either decrements skip or calls emit_fn with all slot values
 ordered oldest-to-newest.

 window_flush shifts in count positions of nils, emitting each time.
*/

func windowMut(a ...Scmer) Scmer {
	win := asSlice(a[0], "window_mut")
	emitFn := a[1]
	vals := asSlice(a[2], "window_mut vals")

	if len(win) < 3 {
		panic("window_mut: window must have at least 3 elements (skip, counter, stride)")
	}

	skip := int(win[0].Int())
	counter := int(win[1].Int())
	stride := int(win[2].Int())
	slots := win[3:]
	if stride <= 0 || len(slots) == 0 || len(slots)%stride != 0 {
		panic("window_mut: invalid window dimensions")
	}

	// The accumulator belongs exclusively to the serial reducer. Keep its slots
	// physically oldest-to-newest so the variadic emit call can borrow the
	// contiguous backing array instead of allocating a rotated argument frame.
	copy(slots, slots[stride:])
	tail := slots[len(slots)-stride:]
	for i := range tail {
		if i < len(vals) {
			tail[i] = vals[i]
		} else {
			tail[i] = NewNil()
		}
	}
	win[1] = NewInt(int64(counter + 1))
	if skip > 0 {
		win[0] = NewInt(int64(skip - 1))
		return a[0]
	}
	Apply(emitFn, slots...)
	return a[0]
}

func windowFlush(a ...Scmer) Scmer {
	win := asSlice(a[0], "window_flush")
	emitFn := a[1]
	count := int(a[2].Int())
	if len(win) < 3 {
		panic("window_flush: window must have at least 3 elements")
	}
	stride := int(win[2].Int())
	slots := win[3:]
	if stride <= 0 || len(slots) == 0 || len(slots)%stride != 0 {
		panic("window_flush: invalid window dimensions")
	}
	for n := 0; n < count; n++ {
		copy(slots, slots[stride:])
		for i := len(slots) - stride; i < len(slots); i++ {
			slots[i] = NewNil()
		}
		win[1] = NewInt(win[1].Int() + 1)
		Apply(emitFn, slots...)
	}
	return NewNil()
}

func init_window() {
	DeclareTitle("Window Functions")

	Declare(&Globalenv, &Declaration{
		Name: "stream_emit",

		Fn: func(a ...Scmer) Scmer {
			return Apply(a[0], a[1])
		},
		Type: &TypeDescriptor{Kind: "func", Description: "invokes a streaming callback immediately; marks ordering-sensitive emission as an observable effect",
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "func", Label: "emit", Params: []*TypeDescriptor{{Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "value"},
			},
			Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["stream_emit"].Fn, args, result)
				}
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				d0 := args[0]
				d0.ID = 0
				d1 := args[1]
				d1.ID = 0
				stackArray2 := ctx.AllocStack(int32(16))
				_ = stackArray2
				ctx.EnsureDesc(&d1)
				ctx.EnsureDesc(&d1)
				ctx.EmitStoreScmerToStack(d1, int32(stackArray2)+int32(0))
				ctx.FreeDesc(&d1)
				d3 := JITValueDesc{Loc: LocVirtualSlice, Type: tagSlice, KnownSliceLen: int32(1), KnownSliceCap: int32(1), SliceSizeKnown: true}
				_ = d3
				ctx.EnsureDesc(&d0)
				ctx.EnsureDesc(&d3)
				d4 := d0
				_ = d4
				ctx.StabilizeDescForControlFlow(&d4)
				d5 := d3
				_ = d5
				ctx.StabilizeDescForControlFlow(&d5)
				bbpos_1_0 := int32(-1)
				_ = bbpos_1_0
				bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
				ctx.ReclaimUntrackedRegs()
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				ctx.EnsureDesc(&d4)
				if d4.Loc == LocImm {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					if d4.Imm.GetTag() == tagBool {
						ctx.EmitMakeBool(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagInt {
						ctx.EmitMakeInt(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagFloat {
						ctx.EmitMakeFloat(tmpPair, d4)
					} else if d4.Imm.GetTag() == tagNil {
						ctx.EmitMakeNil(tmpPair)
					} else {
						ptrWord, auxWord := d4.Imm.RawWords()
						ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
						ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
					}
					d4 = tmpPair
				} else if d4.Loc == LocReg {
					tmpPair := JITValueDesc{Loc: LocRegPair, Type: d4.Type, Reg: ctx.AllocRegExcept(d4.Reg), Reg2: ctx.AllocRegExcept(d4.Reg)}
					switch d4.Type {
					case tagBool:
						ctx.EmitMakeBool(tmpPair, d4)
					case tagInt:
						ctx.EmitMakeInt(tmpPair, d4)
					case tagFloat:
						ctx.EmitMakeFloat(tmpPair, d4)
					default:
						panic("jit: generic call arg scalar type unknown for 2-word value")
					}
					ctx.FreeDesc(&d4)
					d4 = tmpPair
				}
				if d4.Loc != LocRegPair && d4.Loc != LocStackPair {
					panic("jit: generic call arg expects 2-word value (ApplyEx arg0)")
				}
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
				ctx.EnsureDesc(&d5)
				if d5.Loc != LocRegTriple && d5.Loc != LocStackTriple {
					panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
				}
				d6 := JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
				if d6.Loc == LocRegPair || d6.Loc == LocStackPair || d6.Loc == LocRegTriple || d6.Loc == LocStackTriple {
					panic("jit: generic call arg expects 1-word value")
				}
				ctx.SyncDesc(&d4)
				ctx.SyncDesc(&d5)
				ctx.SyncDesc(&d6)
				d7 := ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d4, d5, d6}, 2)
				ctx.BindReg(d7.Reg, &d7)
				ctx.BindReg(d7.Reg2, &d7)
				ctx.ReclaimUntrackedRegs()
				ctx.EnsureDesc(&d7)
				ctx.FreeDesc(&d0)
				if d7.Loc == LocImm {
					if result.Loc == LocAny {
						return d7
					}
				}
				if result.Loc == LocAny {
					result = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
					ctx.BindReg(result.Reg, &result)
					ctx.BindReg(result.Reg2, &result)
				}
				ctx.EnsureDesc(&d7)
				if d7.Loc == LocRegPair {
					ctx.EmitMovPairToResult(&d7, &result)
					result.Type = d7.Type
				} else {
					switch d7.Type {
					case tagBool:
						ctx.EmitMakeBool(result, d7)
						result.Type = tagBool
					case tagInt:
						ctx.EmitMakeInt(result, d7)
						result.Type = tagInt
					case tagFloat:
						ctx.EmitMakeFloat(result, d7)
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
			JITInlineCost: 12,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "stream_window_reduce",

		Fn: func(a ...Scmer) (result Scmer) {
			offset := ToInt(a[0])
			limit := ToInt(a[1])
			if offset < 0 {
				panic("stream_window_reduce: offset must not be negative")
			}
			if limit < -1 {
				panic("stream_window_reduce: limit must be -1 or non-negative")
			}
			result = a[3]
			if limit == 0 {
				return result
			}
			reduceProgram := PrepareSerialProc(a[2])
			producerProgram := PrepareSerialProc(a[4])
			var reduceArgs [2]Scmer
			var producerArgs [1]Scmer
			seen := 0
			emitted := 0
			// A streaming producer may fan out over table shards and invoke emit
			// concurrently. The window state, reducer environment, and its borrowed
			// native-call frames are intentionally serial. Keep that contract at the
			// stream boundary instead of forcing every producer to abandon parallel
			// scans or making all prepared callbacks pay for synchronization.
			var emitMu sync.Mutex
			emit := NewFunc(func(values ...Scmer) Scmer {
				emitMu.Lock()
				defer emitMu.Unlock()
				if len(values) != 1 {
					panic("stream_window_reduce: emit expects exactly one complete value")
				}
				seen++
				if seen <= offset {
					return result
				}
				if limit >= 0 && emitted >= limit {
					return result
				}
				reduceArgs[0], reduceArgs[1] = result, values[0]
				result = reduceProgram.Call(reduceArgs[:])
				emitted++
				return result
			})
			producerArgs[0] = emit
			producerProgram.Call(producerArgs[:])
			return result
		},
		Type: &TypeDescriptor{Kind: "func", Description: "applies OFFSET/LIMIT and a serial reducer to complete values emitted by a nested streaming producer without collecting an intermediate relation",
			HasSideEffects: true,
			Params: []*TypeDescriptor{
				{Kind: "number", Label: "offset", Description: "number of complete producer values to skip"},
				{Kind: "number", Label: "limit", Description: "maximum values to reduce, or -1 for no limit"},
				{Kind: "func", Label: "reduce", Description: "serial accumulator over complete values", Params: []*TypeDescriptor{{Kind: "any", Label: "acc"}, {Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any"}},
				{Kind: "any", Label: "neutral", Description: "initial accumulator"},
				{Kind: "func", Label: "producer", Description: "nested streaming plan called with a one-value emit callback", Params: []*TypeDescriptor{{Kind: "func", Label: "emit", Description: "emits one complete value", Params: []*TypeDescriptor{{Kind: "any", Label: "value"}}, Return: &TypeDescriptor{Kind: "any", Label: "result"}}}, Return: &TypeDescriptor{Kind: "any"}},
			},
			Return: &TypeDescriptor{Kind: "any"},
			JITEmit: func(ctx *JITContext, _ []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				// JITGen native call boundary: escaping or recursive Go closure.
				return jitEmitGoVariadicCallFromDescs(ctx, declarations["stream_window_reduce"].Fn, args, result)
			},
			JITVirtualArgs: true,
			JITInlineCost:  65535,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "window_mut",

		Fn: windowMut,
		Type: &TypeDescriptor{Kind: "func", Description: "Owned sliding-window shift. (window_mut window emit_fn vals) mutates its serial accumulator in place, keeping values oldest-to-newest so emit_fn can borrow them without an allocation.", HasSideEffects: true,
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "list", Label: "window", Description: "caller-owned serial window accumulator"}, &TypeDescriptor{Kind: "func", Label: "emit_fn", Description: "callback receiving all window values oldest-to-newest", Params: []*TypeDescriptor{{Kind: "any", Label: "values", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}, &TypeDescriptor{Kind: "list", Label: "vals", Description: "list of stride values to insert"}},
			Return: &TypeDescriptor{Kind: "list"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["window_mut"].Fn, args, result)
				}
				var d2 JITValueDesc
				_ = d2
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d24 JITValueDesc
				_ = d24
				var d25 JITValueDesc
				_ = d25
				var d26 JITValueDesc
				_ = d26
				var d27 JITValueDesc
				_ = d27
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
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
				var d84 JITValueDesc
				_ = d84
				var d85 JITValueDesc
				_ = d85
				var d86 JITValueDesc
				_ = d86
				var d87 JITValueDesc
				_ = d87
				var d88 JITValueDesc
				_ = d88
				var d90 JITValueDesc
				_ = d90
				var d91 JITValueDesc
				_ = d91
				var d92 JITValueDesc
				_ = d92
				var d93 JITValueDesc
				_ = d93
				var d94 JITValueDesc
				_ = d94
				var d95 JITValueDesc
				_ = d95
				var d96 JITValueDesc
				_ = d96
				var d97 JITValueDesc
				_ = d97
				var d98 JITValueDesc
				_ = d98
				var d100 JITValueDesc
				_ = d100
				var d101 JITValueDesc
				_ = d101
				var d102 JITValueDesc
				_ = d102
				var d103 JITValueDesc
				_ = d103
				var d104 JITValueDesc
				_ = d104
				var d161 JITValueDesc
				_ = d161
				var d162 JITValueDesc
				_ = d162
				var d163 JITValueDesc
				_ = d163
				var d223 JITValueDesc
				_ = d223
				var d224 JITValueDesc
				_ = d224
				var d225 JITValueDesc
				_ = d225
				var d226 JITValueDesc
				_ = d226
				var d229 JITValueDesc
				_ = d229
				var d292 JITValueDesc
				_ = d292
				var d293 JITValueDesc
				_ = d293
				var d294 JITValueDesc
				_ = d294
				var d362 JITValueDesc
				_ = d362
				var d363 JITValueDesc
				_ = d363
				var d364 JITValueDesc
				_ = d364
				var d365 JITValueDesc
				_ = d365
				var d366 JITValueDesc
				_ = d366
				var d367 JITValueDesc
				_ = d367
				var d441 JITValueDesc
				_ = d441
				var d442 JITValueDesc
				_ = d442
				var d443 JITValueDesc
				_ = d443
				var d445 JITValueDesc
				_ = d445
				var d446 JITValueDesc
				_ = d446
				var d448 JITValueDesc
				_ = d448
				var d449 JITValueDesc
				_ = d449
				var d450 JITValueDesc
				_ = d450
				var d451 JITValueDesc
				_ = d451
				var d452 JITValueDesc
				_ = d452
				var d453 JITValueDesc
				_ = d453
				var d454 JITValueDesc
				_ = d454
				var d455 JITValueDesc
				_ = d455
				var d456 JITValueDesc
				_ = d456
				var d457 JITValueDesc
				_ = d457
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(16))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				var bbs [14]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
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
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
				bbpos_0_13 := int32(-1)
				_ = bbpos_0_13
				lbl14 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					ctx.ReclaimUntrackedRegs()
					d2 = args[0]
					d2.ID = 0
					var d3 JITValueDesc
					if d2.Type == tagSlice {
						d3 = jitKnownSliceHeader(ctx, &d2)
					} else {
						d3 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d2}, 3)
					}
					ctx.BindReg(d3.Reg, &d3)
					ctx.BindReg(d3.Reg2, &d3)
					ctx.BindReg(d3.Reg3, &d3)
					ctx.StabilizeDescForControlFlow(&d3)
					ctx.FreeDesc(&d2)
					d4 = args[1]
					d4.ID = 0
					ctx.StabilizeDescForControlFlow(&d4)
					d5 = args[2]
					d5.ID = 0
					var d6 JITValueDesc
					if d5.Type == tagSlice {
						d6 = jitKnownSliceHeader(ctx, &d5)
					} else {
						d6 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d5}, 3)
					}
					ctx.BindReg(d6.Reg, &d6)
					ctx.BindReg(d6.Reg2, &d6)
					ctx.BindReg(d6.Reg3, &d6)
					ctx.StabilizeDescForControlFlow(&d6)
					ctx.FreeDesc(&d5)
					var d7 JITValueDesc
					if d3.SliceSizeKnown {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.KnownSliceLen))}
					} else if d3.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d3.StackOff))}
					} else if d3.Loc == LocStackTriple {
						d7 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d3.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d3)
						if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2, ID: 0}
						} else if d3.Loc == LocReg {
							d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d7)
					var d8 JITValueDesc
					if d7.Loc == LocImm {
						d8 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d7.Imm.Int() < 3)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d7.Reg, 3)
						ctx.EmitSetcc(r0, CondSignedLess)
						d8 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d8)
					}
					ctx.FreeDesc(&d7)
					d9 = d8
					ctx.EnsureDesc(&d9)
					if d9.Loc != LocImm && d9.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d9.Loc == LocImm {
						if d9.Imm.Bool() {
							if ps.General {
							}
							ps10 := PhiState{General: ps.General}
							ps10.OverlayValues = make([]JITValueDesc, 10)
							ps10.OverlayValues[1] = d1
							ps10.OverlayValues[2] = d2
							ps10.OverlayValues[3] = d3
							ps10.OverlayValues[4] = d4
							ps10.OverlayValues[5] = d5
							ps10.OverlayValues[6] = d6
							ps10.OverlayValues[7] = d7
							ps10.OverlayValues[8] = d8
							ps10.OverlayValues[9] = d9
							return bbs[1].RenderPS(ps10)
						}
						if ps.General {
						}
						ps11 := PhiState{General: ps.General}
						ps11.OverlayValues = make([]JITValueDesc, 10)
						ps11.OverlayValues[1] = d1
						ps11.OverlayValues[2] = d2
						ps11.OverlayValues[3] = d3
						ps11.OverlayValues[4] = d4
						ps11.OverlayValues[5] = d5
						ps11.OverlayValues[6] = d6
						ps11.OverlayValues[7] = d7
						ps11.OverlayValues[8] = d8
						ps11.OverlayValues[9] = d9
						return bbs[2].RenderPS(ps11)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl15 := ctx.ReserveLabel()
					lbl16 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d9.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl15)
					ctx.EmitJmp(lbl16)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl3)
					ps12 := PhiState{General: true}
					ps12.OverlayValues = make([]JITValueDesc, 10)
					ps12.OverlayValues[1] = d1
					ps12.OverlayValues[2] = d2
					ps12.OverlayValues[3] = d3
					ps12.OverlayValues[4] = d4
					ps12.OverlayValues[5] = d5
					ps12.OverlayValues[6] = d6
					ps12.OverlayValues[7] = d7
					ps12.OverlayValues[8] = d8
					ps12.OverlayValues[9] = d9
					ps13 := PhiState{General: true}
					ps13.OverlayValues = make([]JITValueDesc, 10)
					ps13.OverlayValues[1] = d1
					ps13.OverlayValues[2] = d2
					ps13.OverlayValues[3] = d3
					ps13.OverlayValues[4] = d4
					ps13.OverlayValues[5] = d5
					ps13.OverlayValues[6] = d6
					ps13.OverlayValues[7] = d7
					ps13.OverlayValues[8] = d8
					ps13.OverlayValues[9] = d9
					snap14 := d1
					snap15 := d2
					snap16 := d3
					snap17 := d4
					snap18 := d5
					snap19 := d6
					snap20 := d7
					snap21 := d8
					snap22 := d9
					alloc23 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps13)
					}
					ctx.RestoreAllocState(alloc23)
					d1 = snap14
					d2 = snap15
					d3 = snap16
					d4 = snap17
					d5 = snap18
					d6 = snap19
					d7 = snap20
					d8 = snap21
					d9 = snap22
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps12)
					}
					return result
					ctx.FreeDesc(&d8)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_mut"].Fn, args, result)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					ctx.ReclaimUntrackedRegs()
					d24 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					d26 = ctx.EmitSliceElementAddress(&d3, &d24, 16)
					ctx.EnsureDesc(&d26)
					r1 := ctx.AllocRegExcept(d26.Reg)
					ctx.EmitMovRegMem(r1, d26.Reg, 8)
					ctx.EmitMovRegMem(d26.Reg, d26.Reg, 0)
					d25 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d26.Reg, Reg2: r1}
					ctx.BindReg(d26.Reg, &d25)
					ctx.BindReg(r1, &d25)
					var d27 JITValueDesc
					if d25.Loc == LocImm {
						d27 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d25.Imm.Int())}
					} else if d25.Type == tagInt && d25.Loc == LocRegPair {
						ctx.FreeReg(d25.Reg)
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg2}
						ctx.BindReg(d25.Reg2, &d27)
						ctx.BindReg(d25.Reg2, &d27)
					} else if d25.Type == tagInt && d25.Loc == LocReg {
						d27 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d25.Reg}
						ctx.BindReg(d25.Reg, &d27)
						ctx.BindReg(d25.Reg, &d27)
					} else {
						d27 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d25}, 1)
						d27.Type = tagInt
						ctx.BindReg(d27.Reg, &d27)
					}
					ctx.FreeDesc(&d25)
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					ctx.StabilizeDescForControlFlow(&d27)
					d29 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d31 = ctx.EmitSliceElementAddress(&d3, &d29, 16)
					ctx.EnsureDesc(&d31)
					r2 := ctx.AllocRegExcept(d31.Reg)
					ctx.EmitMovRegMem(r2, d31.Reg, 8)
					ctx.EmitMovRegMem(d31.Reg, d31.Reg, 0)
					d30 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d31.Reg, Reg2: r2}
					ctx.BindReg(d31.Reg, &d30)
					ctx.BindReg(r2, &d30)
					var d32 JITValueDesc
					if d30.Loc == LocImm {
						d32 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d30.Imm.Int())}
					} else if d30.Type == tagInt && d30.Loc == LocRegPair {
						ctx.FreeReg(d30.Reg)
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg2}
						ctx.BindReg(d30.Reg2, &d32)
						ctx.BindReg(d30.Reg2, &d32)
					} else if d30.Type == tagInt && d30.Loc == LocReg {
						d32 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d30.Reg}
						ctx.BindReg(d30.Reg, &d32)
						ctx.BindReg(d30.Reg, &d32)
					} else {
						d32 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d30}, 1)
						d32.Type = tagInt
						ctx.BindReg(d32.Reg, &d32)
					}
					ctx.FreeDesc(&d30)
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					ctx.StabilizeDescForControlFlow(&d32)
					d34 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d36 = ctx.EmitSliceElementAddress(&d3, &d34, 16)
					ctx.EnsureDesc(&d36)
					r3 := ctx.AllocRegExcept(d36.Reg)
					ctx.EmitMovRegMem(r3, d36.Reg, 8)
					ctx.EmitMovRegMem(d36.Reg, d36.Reg, 0)
					d35 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d36.Reg, Reg2: r3}
					ctx.BindReg(d36.Reg, &d35)
					ctx.BindReg(r3, &d35)
					var d37 JITValueDesc
					if d35.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d35.Imm.Int())}
					} else if d35.Type == tagInt && d35.Loc == LocRegPair {
						ctx.FreeReg(d35.Reg)
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg2}
						ctx.BindReg(d35.Reg2, &d37)
						ctx.BindReg(d35.Reg2, &d37)
					} else if d35.Type == tagInt && d35.Loc == LocReg {
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d35.Reg}
						ctx.BindReg(d35.Reg, &d37)
						ctx.BindReg(d35.Reg, &d37)
					} else {
						d37 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d35}, 1)
						d37.Type = tagInt
						ctx.BindReg(d37.Reg, &d37)
					}
					ctx.FreeDesc(&d35)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d37)
					ctx.StabilizeDescForControlFlow(&d37)
					d39 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d40 JITValueDesc
					ctx.EnsureDesc(&d3)
					if d3.Loc == LocRegPair || d3.Loc == LocRegTriple {
						d40 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d3.Reg2}
						ctx.BindReg(d3.Reg2, &d40)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d3)
					ctx.EnsureDesc(&d39)
					ctx.EnsureDesc(&d40)
					var d42 JITValueDesc
					if d40.Loc == LocImm && d39.Loc == LocImm {
						d42 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d40.Imm.Int() - d39.Imm.Int())}
					} else {
						r4 := ctx.AllocReg()
						if d40.Loc == LocImm {
							ctx.EmitMovRegImm64(r4, uint64(d40.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r4, d40.Reg)
						}
						if d39.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()))
							ctx.EmitSubInt64(r4, RegR11)
						} else {
							ctx.EmitSubInt64(r4, d39.Reg)
						}
						d42 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r4}
						ctx.BindReg(r4, &d42)
					}
					var d43 JITValueDesc
					if d3.Loc == LocImm && d39.Loc == LocImm {
						d43 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d3.Imm.Int() + d39.Imm.Int()*16)}
					} else {
						r5 := ctx.AllocReg()
						if d3.Loc == LocImm {
							ctx.EmitMovRegImm64(r5, uint64(d3.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r5, d3.Reg)
						}
						if d39.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d39.Imm.Int()*16))
							ctx.EmitAddInt64(r5, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r5, d39.Reg)
							ctx.EmitMovRegReg(offsetReg, d39.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r5, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d43 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r5}
						ctx.BindReg(r5, &d43)
					}
					var d44 JITValueDesc
					var r6 Reg
					var r7 Reg
					ctx.SyncDesc(&d43)
					ctx.EnsureDesc(&d43)
					if d43.Loc == LocImm {
						r6 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r6, uint64(d43.Imm.Int()))
					} else {
						r6 = d43.Reg
					}
					ctx.ProtectReg(r6)
					ctx.SyncDesc(&d42)
					ctx.EnsureDesc(&d42)
					if d42.Loc == LocImm {
						r7 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r7, uint64(d42.Imm.Int()))
					} else {
						r7 = d42.Reg
					}
					ctx.ProtectReg(r7)
					r8 := ctx.EmitSliceCapAfterLow(&d3, &d39, r6, r7)
					ctx.UnprotectReg(r7)
					ctx.UnprotectReg(r6)
					d44 = JITValueDesc{Loc: LocRegTriple, Reg: r6, Reg2: r7, Reg3: r8}
					ctx.BindReg(r6, &d44)
					ctx.BindReg(r7, &d44)
					ctx.BindReg(r8, &d44)
					ctx.BindReg(r6, &d44)
					ctx.BindReg(r7, &d44)
					ctx.BindReg(r8, &d44)
					ctx.StabilizeDescForControlFlow(&d44)
					ctx.EnsureDesc(&d37)
					var d45 JITValueDesc
					if d37.Loc == LocImm {
						d45 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d37.Imm.Int() <= 0)}
					} else {
						r9 := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitCmpRegImm32(d37.Reg, 0)
						ctx.EmitSetcc(r9, CondSignedLessOrEqual)
						d45 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d45)
					}
					d46 = d45
					ctx.EnsureDesc(&d46)
					if d46.Loc != LocImm && d46.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d46.Loc == LocImm {
						if d46.Imm.Bool() {
							if ps.General {
							}
							ps47 := PhiState{General: ps.General}
							ps47.OverlayValues = make([]JITValueDesc, 47)
							ps47.OverlayValues[1] = d1
							ps47.OverlayValues[2] = d2
							ps47.OverlayValues[3] = d3
							ps47.OverlayValues[4] = d4
							ps47.OverlayValues[5] = d5
							ps47.OverlayValues[6] = d6
							ps47.OverlayValues[7] = d7
							ps47.OverlayValues[8] = d8
							ps47.OverlayValues[9] = d9
							ps47.OverlayValues[24] = d24
							ps47.OverlayValues[25] = d25
							ps47.OverlayValues[26] = d26
							ps47.OverlayValues[27] = d27
							ps47.OverlayValues[28] = d28
							ps47.OverlayValues[29] = d29
							ps47.OverlayValues[30] = d30
							ps47.OverlayValues[31] = d31
							ps47.OverlayValues[32] = d32
							ps47.OverlayValues[33] = d33
							ps47.OverlayValues[34] = d34
							ps47.OverlayValues[35] = d35
							ps47.OverlayValues[36] = d36
							ps47.OverlayValues[37] = d37
							ps47.OverlayValues[38] = d38
							ps47.OverlayValues[39] = d39
							ps47.OverlayValues[40] = d40
							ps47.OverlayValues[41] = d41
							ps47.OverlayValues[42] = d42
							ps47.OverlayValues[43] = d43
							ps47.OverlayValues[44] = d44
							ps47.OverlayValues[45] = d45
							ps47.OverlayValues[46] = d46
							return bbs[3].RenderPS(ps47)
						}
						if ps.General {
						}
						ps48 := PhiState{General: ps.General}
						ps48.OverlayValues = make([]JITValueDesc, 47)
						ps48.OverlayValues[1] = d1
						ps48.OverlayValues[2] = d2
						ps48.OverlayValues[3] = d3
						ps48.OverlayValues[4] = d4
						ps48.OverlayValues[5] = d5
						ps48.OverlayValues[6] = d6
						ps48.OverlayValues[7] = d7
						ps48.OverlayValues[8] = d8
						ps48.OverlayValues[9] = d9
						ps48.OverlayValues[24] = d24
						ps48.OverlayValues[25] = d25
						ps48.OverlayValues[26] = d26
						ps48.OverlayValues[27] = d27
						ps48.OverlayValues[28] = d28
						ps48.OverlayValues[29] = d29
						ps48.OverlayValues[30] = d30
						ps48.OverlayValues[31] = d31
						ps48.OverlayValues[32] = d32
						ps48.OverlayValues[33] = d33
						ps48.OverlayValues[34] = d34
						ps48.OverlayValues[35] = d35
						ps48.OverlayValues[36] = d36
						ps48.OverlayValues[37] = d37
						ps48.OverlayValues[38] = d38
						ps48.OverlayValues[39] = d39
						ps48.OverlayValues[40] = d40
						ps48.OverlayValues[41] = d41
						ps48.OverlayValues[42] = d42
						ps48.OverlayValues[43] = d43
						ps48.OverlayValues[44] = d44
						ps48.OverlayValues[45] = d45
						ps48.OverlayValues[46] = d46
						return bbs[6].RenderPS(ps48)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl17 := ctx.ReserveLabel()
					lbl18 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d46.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl17)
					ctx.EmitJmp(lbl18)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl7)
					ps49 := PhiState{General: true}
					ps49.OverlayValues = make([]JITValueDesc, 47)
					ps49.OverlayValues[1] = d1
					ps49.OverlayValues[2] = d2
					ps49.OverlayValues[3] = d3
					ps49.OverlayValues[4] = d4
					ps49.OverlayValues[5] = d5
					ps49.OverlayValues[6] = d6
					ps49.OverlayValues[7] = d7
					ps49.OverlayValues[8] = d8
					ps49.OverlayValues[9] = d9
					ps49.OverlayValues[24] = d24
					ps49.OverlayValues[25] = d25
					ps49.OverlayValues[26] = d26
					ps49.OverlayValues[27] = d27
					ps49.OverlayValues[28] = d28
					ps49.OverlayValues[29] = d29
					ps49.OverlayValues[30] = d30
					ps49.OverlayValues[31] = d31
					ps49.OverlayValues[32] = d32
					ps49.OverlayValues[33] = d33
					ps49.OverlayValues[34] = d34
					ps49.OverlayValues[35] = d35
					ps49.OverlayValues[36] = d36
					ps49.OverlayValues[37] = d37
					ps49.OverlayValues[38] = d38
					ps49.OverlayValues[39] = d39
					ps49.OverlayValues[40] = d40
					ps49.OverlayValues[41] = d41
					ps49.OverlayValues[42] = d42
					ps49.OverlayValues[43] = d43
					ps49.OverlayValues[44] = d44
					ps49.OverlayValues[45] = d45
					ps49.OverlayValues[46] = d46
					ps50 := PhiState{General: true}
					ps50.OverlayValues = make([]JITValueDesc, 47)
					ps50.OverlayValues[1] = d1
					ps50.OverlayValues[2] = d2
					ps50.OverlayValues[3] = d3
					ps50.OverlayValues[4] = d4
					ps50.OverlayValues[5] = d5
					ps50.OverlayValues[6] = d6
					ps50.OverlayValues[7] = d7
					ps50.OverlayValues[8] = d8
					ps50.OverlayValues[9] = d9
					ps50.OverlayValues[24] = d24
					ps50.OverlayValues[25] = d25
					ps50.OverlayValues[26] = d26
					ps50.OverlayValues[27] = d27
					ps50.OverlayValues[28] = d28
					ps50.OverlayValues[29] = d29
					ps50.OverlayValues[30] = d30
					ps50.OverlayValues[31] = d31
					ps50.OverlayValues[32] = d32
					ps50.OverlayValues[33] = d33
					ps50.OverlayValues[34] = d34
					ps50.OverlayValues[35] = d35
					ps50.OverlayValues[36] = d36
					ps50.OverlayValues[37] = d37
					ps50.OverlayValues[38] = d38
					ps50.OverlayValues[39] = d39
					ps50.OverlayValues[40] = d40
					ps50.OverlayValues[41] = d41
					ps50.OverlayValues[42] = d42
					ps50.OverlayValues[43] = d43
					ps50.OverlayValues[44] = d44
					ps50.OverlayValues[45] = d45
					ps50.OverlayValues[46] = d46
					snap51 := d1
					snap52 := d2
					snap53 := d3
					snap54 := d4
					snap55 := d5
					snap56 := d6
					snap57 := d7
					snap58 := d8
					snap59 := d9
					snap60 := d24
					snap61 := d25
					snap62 := d26
					snap63 := d27
					snap64 := d28
					snap65 := d29
					snap66 := d30
					snap67 := d31
					snap68 := d32
					snap69 := d33
					snap70 := d34
					snap71 := d35
					snap72 := d36
					snap73 := d37
					snap74 := d38
					snap75 := d39
					snap76 := d40
					snap77 := d41
					snap78 := d42
					snap79 := d43
					snap80 := d44
					snap81 := d45
					snap82 := d46
					alloc83 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps50)
					}
					ctx.RestoreAllocState(alloc83)
					d1 = snap51
					d2 = snap52
					d3 = snap53
					d4 = snap54
					d5 = snap55
					d6 = snap56
					d7 = snap57
					d8 = snap58
					d9 = snap59
					d24 = snap60
					d25 = snap61
					d26 = snap62
					d27 = snap63
					d28 = snap64
					d29 = snap65
					d30 = snap66
					d31 = snap67
					d32 = snap68
					d33 = snap69
					d34 = snap70
					d35 = snap71
					d36 = snap72
					d37 = snap73
					d38 = snap74
					d39 = snap75
					d40 = snap76
					d41 = snap77
					d42 = snap78
					d43 = snap79
					d44 = snap80
					d45 = snap81
					d46 = snap82
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps49)
					}
					return result
					ctx.FreeDesc(&d45)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_mut"].Fn, args, result)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d37)
					var d84 JITValueDesc
					ctx.EnsureDesc(&d44)
					if d44.Loc == LocRegPair || d44.Loc == LocRegTriple {
						d84 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2}
						ctx.BindReg(d44.Reg2, &d84)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d84)
					var d86 JITValueDesc
					if d84.Loc == LocImm && d37.Loc == LocImm {
						d86 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d84.Imm.Int() - d37.Imm.Int())}
					} else {
						r10 := ctx.AllocReg()
						if d84.Loc == LocImm {
							ctx.EmitMovRegImm64(r10, uint64(d84.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r10, d84.Reg)
						}
						if d37.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()))
							ctx.EmitSubInt64(r10, RegR11)
						} else {
							ctx.EmitSubInt64(r10, d37.Reg)
						}
						d86 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r10}
						ctx.BindReg(r10, &d86)
					}
					var d87 JITValueDesc
					if d44.Loc == LocImm && d37.Loc == LocImm {
						d87 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + d37.Imm.Int()*16)}
					} else {
						r11 := ctx.AllocReg()
						if d44.Loc == LocImm {
							ctx.EmitMovRegImm64(r11, uint64(d44.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r11, d44.Reg)
						}
						if d37.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()*16))
							ctx.EmitAddInt64(r11, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r11, d37.Reg)
							ctx.EmitMovRegReg(offsetReg, d37.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r11, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d87 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r11}
						ctx.BindReg(r11, &d87)
					}
					var d88 JITValueDesc
					var r12 Reg
					var r13 Reg
					ctx.SyncDesc(&d87)
					ctx.EnsureDesc(&d87)
					if d87.Loc == LocImm {
						r12 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r12, uint64(d87.Imm.Int()))
					} else {
						r12 = d87.Reg
					}
					ctx.ProtectReg(r12)
					ctx.SyncDesc(&d86)
					ctx.EnsureDesc(&d86)
					if d86.Loc == LocImm {
						r13 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r13, uint64(d86.Imm.Int()))
					} else {
						r13 = d86.Reg
					}
					ctx.ProtectReg(r13)
					r14 := ctx.EmitSliceCapAfterLow(&d44, &d37, r12, r13)
					ctx.UnprotectReg(r13)
					ctx.UnprotectReg(r12)
					d88 = JITValueDesc{Loc: LocRegTriple, Reg: r12, Reg2: r13, Reg3: r14}
					ctx.BindReg(r12, &d88)
					ctx.BindReg(r13, &d88)
					ctx.BindReg(r14, &d88)
					ctx.BindReg(r12, &d88)
					ctx.BindReg(r13, &d88)
					ctx.BindReg(r14, &d88)
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d88)
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d88)
					callResults89 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d44, d88}, []uint8{1}, []uint8{0})
					d90 = callResults89[0]
					d90.Type = tagInt
					var d91 JITValueDesc
					if d44.SliceSizeKnown {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.KnownSliceLen))}
					} else if d44.Loc == LocImm {
						d91 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.StackOff))}
					} else if d44.Loc == LocStackTriple {
						d91 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d44.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d44)
						if d44.Loc == LocRegPair || d44.Loc == LocRegTriple {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2, ID: 0}
						} else if d44.Loc == LocReg {
							d91 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d91)
					ctx.EnsureDesc(&d37)
					ctx.EnsureDesc(&d91)
					ctx.ProtectReg(d91.Reg)
					ctx.EnsureDesc(&d37)
					ctx.UnprotectReg(d91.Reg)
					var d92 JITValueDesc
					if d91.Loc == LocImm && d37.Loc == LocImm {
						d92 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d91.Imm.Int() - d37.Imm.Int())}
					} else if d37.Loc == LocImm && d37.Imm.Int() == 0 {
						r15 := ctx.AllocRegExcept(d91.Reg)
						ctx.EmitMovRegReg(r15, d91.Reg)
						d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r15}
						ctx.BindReg(r15, &d92)
					} else if d91.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d37.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d91.Imm.Int()))
						ctx.EmitSubInt64(scratch, d37.Reg)
						d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d92)
					} else if d37.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d91.Reg)
						ctx.EmitMovRegReg(scratch, d91.Reg)
						if d37.Imm.Int() >= -2147483648 && d37.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d37.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d37.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d92)
					} else {
						r16 := ctx.AllocRegExcept(d91.Reg, d37.Reg)
						ctx.EmitMovRegReg(r16, d91.Reg)
						ctx.EmitSubInt64(r16, d37.Reg)
						d92 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r16}
						ctx.BindReg(r16, &d92)
					}
					if d92.Loc == LocReg && d91.Loc == LocReg && d92.Reg == d91.Reg {
						ctx.TransferReg(d91.Reg)
						d91.Loc = LocNone
					}
					ctx.FreeDesc(&d91)
					ctx.EnsureDesc(&d92)
					var d93 JITValueDesc
					ctx.EnsureDesc(&d44)
					if d44.Loc == LocRegPair || d44.Loc == LocRegTriple {
						d93 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2}
						ctx.BindReg(d44.Reg2, &d93)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d44)
					ctx.EnsureDesc(&d92)
					ctx.EnsureDesc(&d93)
					var d95 JITValueDesc
					if d93.Loc == LocImm && d92.Loc == LocImm {
						d95 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d93.Imm.Int() - d92.Imm.Int())}
					} else {
						r17 := ctx.AllocReg()
						if d93.Loc == LocImm {
							ctx.EmitMovRegImm64(r17, uint64(d93.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r17, d93.Reg)
						}
						if d92.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d92.Imm.Int()))
							ctx.EmitSubInt64(r17, RegR11)
						} else {
							ctx.EmitSubInt64(r17, d92.Reg)
						}
						d95 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r17}
						ctx.BindReg(r17, &d95)
					}
					var d96 JITValueDesc
					if d44.Loc == LocImm && d92.Loc == LocImm {
						d96 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d44.Imm.Int() + d92.Imm.Int()*16)}
					} else {
						r18 := ctx.AllocReg()
						if d44.Loc == LocImm {
							ctx.EmitMovRegImm64(r18, uint64(d44.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r18, d44.Reg)
						}
						if d92.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d92.Imm.Int()*16))
							ctx.EmitAddInt64(r18, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r18, d92.Reg)
							ctx.EmitMovRegReg(offsetReg, d92.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r18, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d96 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d96)
					}
					var d97 JITValueDesc
					var r19 Reg
					var r20 Reg
					ctx.SyncDesc(&d96)
					ctx.EnsureDesc(&d96)
					if d96.Loc == LocImm {
						r19 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r19, uint64(d96.Imm.Int()))
					} else {
						r19 = d96.Reg
					}
					ctx.ProtectReg(r19)
					ctx.SyncDesc(&d95)
					ctx.EnsureDesc(&d95)
					if d95.Loc == LocImm {
						r20 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r20, uint64(d95.Imm.Int()))
					} else {
						r20 = d95.Reg
					}
					ctx.ProtectReg(r20)
					r21 := ctx.EmitSliceCapAfterLow(&d44, &d92, r19, r20)
					ctx.UnprotectReg(r20)
					ctx.UnprotectReg(r19)
					d97 = JITValueDesc{Loc: LocRegTriple, Reg: r19, Reg2: r20, Reg3: r21}
					ctx.BindReg(r19, &d97)
					ctx.BindReg(r20, &d97)
					ctx.BindReg(r21, &d97)
					ctx.BindReg(r19, &d97)
					ctx.BindReg(r20, &d97)
					ctx.BindReg(r21, &d97)
					ctx.StabilizeDescForControlFlow(&d97)
					ctx.FreeDesc(&d92)
					var d98 JITValueDesc
					if d97.SliceSizeKnown {
						d98 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d97.KnownSliceLen))}
					} else if d97.Loc == LocImm {
						d98 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d97.StackOff))}
					} else if d97.Loc == LocStackTriple {
						d98 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d97.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d97)
						if d97.Loc == LocRegPair || d97.Loc == LocRegTriple {
							d98 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d97.Reg2, ID: 0}
						} else if d97.Loc == LocReg {
							d98 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d97.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.StabilizeDescForControlFlow(&d98)
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}, int32(bbs[7].PhiBase)+int32(0))
					}
					ps99 := PhiState{General: ps.General}
					ps99.OverlayValues = make([]JITValueDesc, 99)
					ps99.OverlayValues[1] = d1
					ps99.OverlayValues[2] = d2
					ps99.OverlayValues[3] = d3
					ps99.OverlayValues[4] = d4
					ps99.OverlayValues[5] = d5
					ps99.OverlayValues[6] = d6
					ps99.OverlayValues[7] = d7
					ps99.OverlayValues[8] = d8
					ps99.OverlayValues[9] = d9
					ps99.OverlayValues[24] = d24
					ps99.OverlayValues[25] = d25
					ps99.OverlayValues[26] = d26
					ps99.OverlayValues[27] = d27
					ps99.OverlayValues[28] = d28
					ps99.OverlayValues[29] = d29
					ps99.OverlayValues[30] = d30
					ps99.OverlayValues[31] = d31
					ps99.OverlayValues[32] = d32
					ps99.OverlayValues[33] = d33
					ps99.OverlayValues[34] = d34
					ps99.OverlayValues[35] = d35
					ps99.OverlayValues[36] = d36
					ps99.OverlayValues[37] = d37
					ps99.OverlayValues[38] = d38
					ps99.OverlayValues[39] = d39
					ps99.OverlayValues[40] = d40
					ps99.OverlayValues[41] = d41
					ps99.OverlayValues[42] = d42
					ps99.OverlayValues[43] = d43
					ps99.OverlayValues[44] = d44
					ps99.OverlayValues[45] = d45
					ps99.OverlayValues[46] = d46
					ps99.OverlayValues[84] = d84
					ps99.OverlayValues[85] = d85
					ps99.OverlayValues[86] = d86
					ps99.OverlayValues[87] = d87
					ps99.OverlayValues[88] = d88
					ps99.OverlayValues[90] = d90
					ps99.OverlayValues[91] = d91
					ps99.OverlayValues[92] = d92
					ps99.OverlayValues[93] = d93
					ps99.OverlayValues[94] = d94
					ps99.OverlayValues[95] = d95
					ps99.OverlayValues[96] = d96
					ps99.OverlayValues[97] = d97
					ps99.OverlayValues[98] = d98
					ps99.PhiValues = make([]JITValueDesc, 1)
					d100 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(-1)}
					ps99.PhiValues[0] = d100
					if ps99.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps99)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					ctx.ReclaimUntrackedRegs()
					var d101 JITValueDesc
					if d44.SliceSizeKnown {
						d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.KnownSliceLen))}
					} else if d44.Loc == LocImm {
						d101 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.StackOff))}
					} else if d44.Loc == LocStackTriple {
						d101 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d44.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d44)
						if d44.Loc == LocRegPair || d44.Loc == LocRegTriple {
							d101 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2, ID: 0}
						} else if d44.Loc == LocReg {
							d101 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d101)
					ctx.EnsureDesc(&d37)
					var d102 JITValueDesc
					if d101.Loc == LocImm && d37.Loc == LocImm {
						d102 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d101.Imm.Int() % d37.Imm.Int())}
					} else {
						d102 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d101, d37}, 1)
					}
					if d102.Loc == LocReg && d101.Loc == LocReg && d102.Reg == d101.Reg {
						ctx.TransferReg(d101.Reg)
						d101.Loc = LocNone
					}
					ctx.FreeDesc(&d101)
					ctx.FreeDesc(&d37)
					ctx.EnsureDesc(&d102)
					var d103 JITValueDesc
					if d102.Loc == LocImm {
						d103 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d102.Imm.Int() != 0)}
					} else {
						r22 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d102.Reg, 0)
						ctx.EmitSetcc(r22, CondNotEqual)
						d103 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d103)
					}
					ctx.FreeDesc(&d102)
					d104 = d103
					ctx.EnsureDesc(&d104)
					if d104.Loc != LocImm && d104.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d104.Loc == LocImm {
						if d104.Imm.Bool() {
							if ps.General {
							}
							ps105 := PhiState{General: ps.General}
							ps105.OverlayValues = make([]JITValueDesc, 105)
							ps105.OverlayValues[1] = d1
							ps105.OverlayValues[2] = d2
							ps105.OverlayValues[3] = d3
							ps105.OverlayValues[4] = d4
							ps105.OverlayValues[5] = d5
							ps105.OverlayValues[6] = d6
							ps105.OverlayValues[7] = d7
							ps105.OverlayValues[8] = d8
							ps105.OverlayValues[9] = d9
							ps105.OverlayValues[24] = d24
							ps105.OverlayValues[25] = d25
							ps105.OverlayValues[26] = d26
							ps105.OverlayValues[27] = d27
							ps105.OverlayValues[28] = d28
							ps105.OverlayValues[29] = d29
							ps105.OverlayValues[30] = d30
							ps105.OverlayValues[31] = d31
							ps105.OverlayValues[32] = d32
							ps105.OverlayValues[33] = d33
							ps105.OverlayValues[34] = d34
							ps105.OverlayValues[35] = d35
							ps105.OverlayValues[36] = d36
							ps105.OverlayValues[37] = d37
							ps105.OverlayValues[38] = d38
							ps105.OverlayValues[39] = d39
							ps105.OverlayValues[40] = d40
							ps105.OverlayValues[41] = d41
							ps105.OverlayValues[42] = d42
							ps105.OverlayValues[43] = d43
							ps105.OverlayValues[44] = d44
							ps105.OverlayValues[45] = d45
							ps105.OverlayValues[46] = d46
							ps105.OverlayValues[84] = d84
							ps105.OverlayValues[85] = d85
							ps105.OverlayValues[86] = d86
							ps105.OverlayValues[87] = d87
							ps105.OverlayValues[88] = d88
							ps105.OverlayValues[90] = d90
							ps105.OverlayValues[91] = d91
							ps105.OverlayValues[92] = d92
							ps105.OverlayValues[93] = d93
							ps105.OverlayValues[94] = d94
							ps105.OverlayValues[95] = d95
							ps105.OverlayValues[96] = d96
							ps105.OverlayValues[97] = d97
							ps105.OverlayValues[98] = d98
							ps105.OverlayValues[100] = d100
							ps105.OverlayValues[101] = d101
							ps105.OverlayValues[102] = d102
							ps105.OverlayValues[103] = d103
							ps105.OverlayValues[104] = d104
							return bbs[3].RenderPS(ps105)
						}
						if ps.General {
						}
						ps106 := PhiState{General: ps.General}
						ps106.OverlayValues = make([]JITValueDesc, 105)
						ps106.OverlayValues[1] = d1
						ps106.OverlayValues[2] = d2
						ps106.OverlayValues[3] = d3
						ps106.OverlayValues[4] = d4
						ps106.OverlayValues[5] = d5
						ps106.OverlayValues[6] = d6
						ps106.OverlayValues[7] = d7
						ps106.OverlayValues[8] = d8
						ps106.OverlayValues[9] = d9
						ps106.OverlayValues[24] = d24
						ps106.OverlayValues[25] = d25
						ps106.OverlayValues[26] = d26
						ps106.OverlayValues[27] = d27
						ps106.OverlayValues[28] = d28
						ps106.OverlayValues[29] = d29
						ps106.OverlayValues[30] = d30
						ps106.OverlayValues[31] = d31
						ps106.OverlayValues[32] = d32
						ps106.OverlayValues[33] = d33
						ps106.OverlayValues[34] = d34
						ps106.OverlayValues[35] = d35
						ps106.OverlayValues[36] = d36
						ps106.OverlayValues[37] = d37
						ps106.OverlayValues[38] = d38
						ps106.OverlayValues[39] = d39
						ps106.OverlayValues[40] = d40
						ps106.OverlayValues[41] = d41
						ps106.OverlayValues[42] = d42
						ps106.OverlayValues[43] = d43
						ps106.OverlayValues[44] = d44
						ps106.OverlayValues[45] = d45
						ps106.OverlayValues[46] = d46
						ps106.OverlayValues[84] = d84
						ps106.OverlayValues[85] = d85
						ps106.OverlayValues[86] = d86
						ps106.OverlayValues[87] = d87
						ps106.OverlayValues[88] = d88
						ps106.OverlayValues[90] = d90
						ps106.OverlayValues[91] = d91
						ps106.OverlayValues[92] = d92
						ps106.OverlayValues[93] = d93
						ps106.OverlayValues[94] = d94
						ps106.OverlayValues[95] = d95
						ps106.OverlayValues[96] = d96
						ps106.OverlayValues[97] = d97
						ps106.OverlayValues[98] = d98
						ps106.OverlayValues[100] = d100
						ps106.OverlayValues[101] = d101
						ps106.OverlayValues[102] = d102
						ps106.OverlayValues[103] = d103
						ps106.OverlayValues[104] = d104
						return bbs[4].RenderPS(ps106)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl19 := ctx.ReserveLabel()
					lbl20 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d104.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl19)
					ctx.EmitJmp(lbl20)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl5)
					ps107 := PhiState{General: true}
					ps107.OverlayValues = make([]JITValueDesc, 105)
					ps107.OverlayValues[1] = d1
					ps107.OverlayValues[2] = d2
					ps107.OverlayValues[3] = d3
					ps107.OverlayValues[4] = d4
					ps107.OverlayValues[5] = d5
					ps107.OverlayValues[6] = d6
					ps107.OverlayValues[7] = d7
					ps107.OverlayValues[8] = d8
					ps107.OverlayValues[9] = d9
					ps107.OverlayValues[24] = d24
					ps107.OverlayValues[25] = d25
					ps107.OverlayValues[26] = d26
					ps107.OverlayValues[27] = d27
					ps107.OverlayValues[28] = d28
					ps107.OverlayValues[29] = d29
					ps107.OverlayValues[30] = d30
					ps107.OverlayValues[31] = d31
					ps107.OverlayValues[32] = d32
					ps107.OverlayValues[33] = d33
					ps107.OverlayValues[34] = d34
					ps107.OverlayValues[35] = d35
					ps107.OverlayValues[36] = d36
					ps107.OverlayValues[37] = d37
					ps107.OverlayValues[38] = d38
					ps107.OverlayValues[39] = d39
					ps107.OverlayValues[40] = d40
					ps107.OverlayValues[41] = d41
					ps107.OverlayValues[42] = d42
					ps107.OverlayValues[43] = d43
					ps107.OverlayValues[44] = d44
					ps107.OverlayValues[45] = d45
					ps107.OverlayValues[46] = d46
					ps107.OverlayValues[84] = d84
					ps107.OverlayValues[85] = d85
					ps107.OverlayValues[86] = d86
					ps107.OverlayValues[87] = d87
					ps107.OverlayValues[88] = d88
					ps107.OverlayValues[90] = d90
					ps107.OverlayValues[91] = d91
					ps107.OverlayValues[92] = d92
					ps107.OverlayValues[93] = d93
					ps107.OverlayValues[94] = d94
					ps107.OverlayValues[95] = d95
					ps107.OverlayValues[96] = d96
					ps107.OverlayValues[97] = d97
					ps107.OverlayValues[98] = d98
					ps107.OverlayValues[100] = d100
					ps107.OverlayValues[101] = d101
					ps107.OverlayValues[102] = d102
					ps107.OverlayValues[103] = d103
					ps107.OverlayValues[104] = d104
					ps108 := PhiState{General: true}
					ps108.OverlayValues = make([]JITValueDesc, 105)
					ps108.OverlayValues[1] = d1
					ps108.OverlayValues[2] = d2
					ps108.OverlayValues[3] = d3
					ps108.OverlayValues[4] = d4
					ps108.OverlayValues[5] = d5
					ps108.OverlayValues[6] = d6
					ps108.OverlayValues[7] = d7
					ps108.OverlayValues[8] = d8
					ps108.OverlayValues[9] = d9
					ps108.OverlayValues[24] = d24
					ps108.OverlayValues[25] = d25
					ps108.OverlayValues[26] = d26
					ps108.OverlayValues[27] = d27
					ps108.OverlayValues[28] = d28
					ps108.OverlayValues[29] = d29
					ps108.OverlayValues[30] = d30
					ps108.OverlayValues[31] = d31
					ps108.OverlayValues[32] = d32
					ps108.OverlayValues[33] = d33
					ps108.OverlayValues[34] = d34
					ps108.OverlayValues[35] = d35
					ps108.OverlayValues[36] = d36
					ps108.OverlayValues[37] = d37
					ps108.OverlayValues[38] = d38
					ps108.OverlayValues[39] = d39
					ps108.OverlayValues[40] = d40
					ps108.OverlayValues[41] = d41
					ps108.OverlayValues[42] = d42
					ps108.OverlayValues[43] = d43
					ps108.OverlayValues[44] = d44
					ps108.OverlayValues[45] = d45
					ps108.OverlayValues[46] = d46
					ps108.OverlayValues[84] = d84
					ps108.OverlayValues[85] = d85
					ps108.OverlayValues[86] = d86
					ps108.OverlayValues[87] = d87
					ps108.OverlayValues[88] = d88
					ps108.OverlayValues[90] = d90
					ps108.OverlayValues[91] = d91
					ps108.OverlayValues[92] = d92
					ps108.OverlayValues[93] = d93
					ps108.OverlayValues[94] = d94
					ps108.OverlayValues[95] = d95
					ps108.OverlayValues[96] = d96
					ps108.OverlayValues[97] = d97
					ps108.OverlayValues[98] = d98
					ps108.OverlayValues[100] = d100
					ps108.OverlayValues[101] = d101
					ps108.OverlayValues[102] = d102
					ps108.OverlayValues[103] = d103
					ps108.OverlayValues[104] = d104
					snap109 := d1
					snap110 := d2
					snap111 := d3
					snap112 := d4
					snap113 := d5
					snap114 := d6
					snap115 := d7
					snap116 := d8
					snap117 := d9
					snap118 := d24
					snap119 := d25
					snap120 := d26
					snap121 := d27
					snap122 := d28
					snap123 := d29
					snap124 := d30
					snap125 := d31
					snap126 := d32
					snap127 := d33
					snap128 := d34
					snap129 := d35
					snap130 := d36
					snap131 := d37
					snap132 := d38
					snap133 := d39
					snap134 := d40
					snap135 := d41
					snap136 := d42
					snap137 := d43
					snap138 := d44
					snap139 := d45
					snap140 := d46
					snap141 := d84
					snap142 := d85
					snap143 := d86
					snap144 := d87
					snap145 := d88
					snap146 := d90
					snap147 := d91
					snap148 := d92
					snap149 := d93
					snap150 := d94
					snap151 := d95
					snap152 := d96
					snap153 := d97
					snap154 := d98
					snap155 := d100
					snap156 := d101
					snap157 := d102
					snap158 := d103
					snap159 := d104
					alloc160 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps108)
					}
					ctx.RestoreAllocState(alloc160)
					d1 = snap109
					d2 = snap110
					d3 = snap111
					d4 = snap112
					d5 = snap113
					d6 = snap114
					d7 = snap115
					d8 = snap116
					d9 = snap117
					d24 = snap118
					d25 = snap119
					d26 = snap120
					d27 = snap121
					d28 = snap122
					d29 = snap123
					d30 = snap124
					d31 = snap125
					d32 = snap126
					d33 = snap127
					d34 = snap128
					d35 = snap129
					d36 = snap130
					d37 = snap131
					d38 = snap132
					d39 = snap133
					d40 = snap134
					d41 = snap135
					d42 = snap136
					d43 = snap137
					d44 = snap138
					d45 = snap139
					d46 = snap140
					d84 = snap141
					d85 = snap142
					d86 = snap143
					d87 = snap144
					d88 = snap145
					d90 = snap146
					d91 = snap147
					d92 = snap148
					d93 = snap149
					d94 = snap150
					d95 = snap151
					d96 = snap152
					d97 = snap153
					d98 = snap154
					d100 = snap155
					d101 = snap156
					d102 = snap157
					d103 = snap158
					d104 = snap159
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps107)
					}
					return result
					ctx.FreeDesc(&d103)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					ctx.ReclaimUntrackedRegs()
					var d161 JITValueDesc
					if d44.SliceSizeKnown {
						d161 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.KnownSliceLen))}
					} else if d44.Loc == LocImm {
						d161 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d44.StackOff))}
					} else if d44.Loc == LocStackTriple {
						d161 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d44.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d44)
						if d44.Loc == LocRegPair || d44.Loc == LocRegTriple {
							d161 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg2, ID: 0}
						} else if d44.Loc == LocReg {
							d161 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d44.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d161)
					var d162 JITValueDesc
					if d161.Loc == LocImm {
						d162 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d161.Imm.Int() == 0)}
					} else {
						r23 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d161.Reg, 0)
						ctx.EmitSetcc(r23, CondEqual)
						d162 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r23}
						ctx.BindReg(r23, &d162)
					}
					ctx.FreeDesc(&d161)
					d163 = d162
					ctx.EnsureDesc(&d163)
					if d163.Loc != LocImm && d163.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d163.Loc == LocImm {
						if d163.Imm.Bool() {
							if ps.General {
							}
							ps164 := PhiState{General: ps.General}
							ps164.OverlayValues = make([]JITValueDesc, 164)
							ps164.OverlayValues[1] = d1
							ps164.OverlayValues[2] = d2
							ps164.OverlayValues[3] = d3
							ps164.OverlayValues[4] = d4
							ps164.OverlayValues[5] = d5
							ps164.OverlayValues[6] = d6
							ps164.OverlayValues[7] = d7
							ps164.OverlayValues[8] = d8
							ps164.OverlayValues[9] = d9
							ps164.OverlayValues[24] = d24
							ps164.OverlayValues[25] = d25
							ps164.OverlayValues[26] = d26
							ps164.OverlayValues[27] = d27
							ps164.OverlayValues[28] = d28
							ps164.OverlayValues[29] = d29
							ps164.OverlayValues[30] = d30
							ps164.OverlayValues[31] = d31
							ps164.OverlayValues[32] = d32
							ps164.OverlayValues[33] = d33
							ps164.OverlayValues[34] = d34
							ps164.OverlayValues[35] = d35
							ps164.OverlayValues[36] = d36
							ps164.OverlayValues[37] = d37
							ps164.OverlayValues[38] = d38
							ps164.OverlayValues[39] = d39
							ps164.OverlayValues[40] = d40
							ps164.OverlayValues[41] = d41
							ps164.OverlayValues[42] = d42
							ps164.OverlayValues[43] = d43
							ps164.OverlayValues[44] = d44
							ps164.OverlayValues[45] = d45
							ps164.OverlayValues[46] = d46
							ps164.OverlayValues[84] = d84
							ps164.OverlayValues[85] = d85
							ps164.OverlayValues[86] = d86
							ps164.OverlayValues[87] = d87
							ps164.OverlayValues[88] = d88
							ps164.OverlayValues[90] = d90
							ps164.OverlayValues[91] = d91
							ps164.OverlayValues[92] = d92
							ps164.OverlayValues[93] = d93
							ps164.OverlayValues[94] = d94
							ps164.OverlayValues[95] = d95
							ps164.OverlayValues[96] = d96
							ps164.OverlayValues[97] = d97
							ps164.OverlayValues[98] = d98
							ps164.OverlayValues[100] = d100
							ps164.OverlayValues[101] = d101
							ps164.OverlayValues[102] = d102
							ps164.OverlayValues[103] = d103
							ps164.OverlayValues[104] = d104
							ps164.OverlayValues[161] = d161
							ps164.OverlayValues[162] = d162
							ps164.OverlayValues[163] = d163
							return bbs[3].RenderPS(ps164)
						}
						if ps.General {
						}
						ps165 := PhiState{General: ps.General}
						ps165.OverlayValues = make([]JITValueDesc, 164)
						ps165.OverlayValues[1] = d1
						ps165.OverlayValues[2] = d2
						ps165.OverlayValues[3] = d3
						ps165.OverlayValues[4] = d4
						ps165.OverlayValues[5] = d5
						ps165.OverlayValues[6] = d6
						ps165.OverlayValues[7] = d7
						ps165.OverlayValues[8] = d8
						ps165.OverlayValues[9] = d9
						ps165.OverlayValues[24] = d24
						ps165.OverlayValues[25] = d25
						ps165.OverlayValues[26] = d26
						ps165.OverlayValues[27] = d27
						ps165.OverlayValues[28] = d28
						ps165.OverlayValues[29] = d29
						ps165.OverlayValues[30] = d30
						ps165.OverlayValues[31] = d31
						ps165.OverlayValues[32] = d32
						ps165.OverlayValues[33] = d33
						ps165.OverlayValues[34] = d34
						ps165.OverlayValues[35] = d35
						ps165.OverlayValues[36] = d36
						ps165.OverlayValues[37] = d37
						ps165.OverlayValues[38] = d38
						ps165.OverlayValues[39] = d39
						ps165.OverlayValues[40] = d40
						ps165.OverlayValues[41] = d41
						ps165.OverlayValues[42] = d42
						ps165.OverlayValues[43] = d43
						ps165.OverlayValues[44] = d44
						ps165.OverlayValues[45] = d45
						ps165.OverlayValues[46] = d46
						ps165.OverlayValues[84] = d84
						ps165.OverlayValues[85] = d85
						ps165.OverlayValues[86] = d86
						ps165.OverlayValues[87] = d87
						ps165.OverlayValues[88] = d88
						ps165.OverlayValues[90] = d90
						ps165.OverlayValues[91] = d91
						ps165.OverlayValues[92] = d92
						ps165.OverlayValues[93] = d93
						ps165.OverlayValues[94] = d94
						ps165.OverlayValues[95] = d95
						ps165.OverlayValues[96] = d96
						ps165.OverlayValues[97] = d97
						ps165.OverlayValues[98] = d98
						ps165.OverlayValues[100] = d100
						ps165.OverlayValues[101] = d101
						ps165.OverlayValues[102] = d102
						ps165.OverlayValues[103] = d103
						ps165.OverlayValues[104] = d104
						ps165.OverlayValues[161] = d161
						ps165.OverlayValues[162] = d162
						ps165.OverlayValues[163] = d163
						return bbs[5].RenderPS(ps165)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl21 := ctx.ReserveLabel()
					lbl22 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d163.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl21)
					ctx.EmitJmp(lbl22)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl6)
					ps166 := PhiState{General: true}
					ps166.OverlayValues = make([]JITValueDesc, 164)
					ps166.OverlayValues[1] = d1
					ps166.OverlayValues[2] = d2
					ps166.OverlayValues[3] = d3
					ps166.OverlayValues[4] = d4
					ps166.OverlayValues[5] = d5
					ps166.OverlayValues[6] = d6
					ps166.OverlayValues[7] = d7
					ps166.OverlayValues[8] = d8
					ps166.OverlayValues[9] = d9
					ps166.OverlayValues[24] = d24
					ps166.OverlayValues[25] = d25
					ps166.OverlayValues[26] = d26
					ps166.OverlayValues[27] = d27
					ps166.OverlayValues[28] = d28
					ps166.OverlayValues[29] = d29
					ps166.OverlayValues[30] = d30
					ps166.OverlayValues[31] = d31
					ps166.OverlayValues[32] = d32
					ps166.OverlayValues[33] = d33
					ps166.OverlayValues[34] = d34
					ps166.OverlayValues[35] = d35
					ps166.OverlayValues[36] = d36
					ps166.OverlayValues[37] = d37
					ps166.OverlayValues[38] = d38
					ps166.OverlayValues[39] = d39
					ps166.OverlayValues[40] = d40
					ps166.OverlayValues[41] = d41
					ps166.OverlayValues[42] = d42
					ps166.OverlayValues[43] = d43
					ps166.OverlayValues[44] = d44
					ps166.OverlayValues[45] = d45
					ps166.OverlayValues[46] = d46
					ps166.OverlayValues[84] = d84
					ps166.OverlayValues[85] = d85
					ps166.OverlayValues[86] = d86
					ps166.OverlayValues[87] = d87
					ps166.OverlayValues[88] = d88
					ps166.OverlayValues[90] = d90
					ps166.OverlayValues[91] = d91
					ps166.OverlayValues[92] = d92
					ps166.OverlayValues[93] = d93
					ps166.OverlayValues[94] = d94
					ps166.OverlayValues[95] = d95
					ps166.OverlayValues[96] = d96
					ps166.OverlayValues[97] = d97
					ps166.OverlayValues[98] = d98
					ps166.OverlayValues[100] = d100
					ps166.OverlayValues[101] = d101
					ps166.OverlayValues[102] = d102
					ps166.OverlayValues[103] = d103
					ps166.OverlayValues[104] = d104
					ps166.OverlayValues[161] = d161
					ps166.OverlayValues[162] = d162
					ps166.OverlayValues[163] = d163
					ps167 := PhiState{General: true}
					ps167.OverlayValues = make([]JITValueDesc, 164)
					ps167.OverlayValues[1] = d1
					ps167.OverlayValues[2] = d2
					ps167.OverlayValues[3] = d3
					ps167.OverlayValues[4] = d4
					ps167.OverlayValues[5] = d5
					ps167.OverlayValues[6] = d6
					ps167.OverlayValues[7] = d7
					ps167.OverlayValues[8] = d8
					ps167.OverlayValues[9] = d9
					ps167.OverlayValues[24] = d24
					ps167.OverlayValues[25] = d25
					ps167.OverlayValues[26] = d26
					ps167.OverlayValues[27] = d27
					ps167.OverlayValues[28] = d28
					ps167.OverlayValues[29] = d29
					ps167.OverlayValues[30] = d30
					ps167.OverlayValues[31] = d31
					ps167.OverlayValues[32] = d32
					ps167.OverlayValues[33] = d33
					ps167.OverlayValues[34] = d34
					ps167.OverlayValues[35] = d35
					ps167.OverlayValues[36] = d36
					ps167.OverlayValues[37] = d37
					ps167.OverlayValues[38] = d38
					ps167.OverlayValues[39] = d39
					ps167.OverlayValues[40] = d40
					ps167.OverlayValues[41] = d41
					ps167.OverlayValues[42] = d42
					ps167.OverlayValues[43] = d43
					ps167.OverlayValues[44] = d44
					ps167.OverlayValues[45] = d45
					ps167.OverlayValues[46] = d46
					ps167.OverlayValues[84] = d84
					ps167.OverlayValues[85] = d85
					ps167.OverlayValues[86] = d86
					ps167.OverlayValues[87] = d87
					ps167.OverlayValues[88] = d88
					ps167.OverlayValues[90] = d90
					ps167.OverlayValues[91] = d91
					ps167.OverlayValues[92] = d92
					ps167.OverlayValues[93] = d93
					ps167.OverlayValues[94] = d94
					ps167.OverlayValues[95] = d95
					ps167.OverlayValues[96] = d96
					ps167.OverlayValues[97] = d97
					ps167.OverlayValues[98] = d98
					ps167.OverlayValues[100] = d100
					ps167.OverlayValues[101] = d101
					ps167.OverlayValues[102] = d102
					ps167.OverlayValues[103] = d103
					ps167.OverlayValues[104] = d104
					ps167.OverlayValues[161] = d161
					ps167.OverlayValues[162] = d162
					ps167.OverlayValues[163] = d163
					snap168 := d1
					snap169 := d2
					snap170 := d3
					snap171 := d4
					snap172 := d5
					snap173 := d6
					snap174 := d7
					snap175 := d8
					snap176 := d9
					snap177 := d24
					snap178 := d25
					snap179 := d26
					snap180 := d27
					snap181 := d28
					snap182 := d29
					snap183 := d30
					snap184 := d31
					snap185 := d32
					snap186 := d33
					snap187 := d34
					snap188 := d35
					snap189 := d36
					snap190 := d37
					snap191 := d38
					snap192 := d39
					snap193 := d40
					snap194 := d41
					snap195 := d42
					snap196 := d43
					snap197 := d44
					snap198 := d45
					snap199 := d46
					snap200 := d84
					snap201 := d85
					snap202 := d86
					snap203 := d87
					snap204 := d88
					snap205 := d90
					snap206 := d91
					snap207 := d92
					snap208 := d93
					snap209 := d94
					snap210 := d95
					snap211 := d96
					snap212 := d97
					snap213 := d98
					snap214 := d100
					snap215 := d101
					snap216 := d102
					snap217 := d103
					snap218 := d104
					snap219 := d161
					snap220 := d162
					snap221 := d163
					alloc222 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps167)
					}
					ctx.RestoreAllocState(alloc222)
					d1 = snap168
					d2 = snap169
					d3 = snap170
					d4 = snap171
					d5 = snap172
					d6 = snap173
					d7 = snap174
					d8 = snap175
					d9 = snap176
					d24 = snap177
					d25 = snap178
					d26 = snap179
					d27 = snap180
					d28 = snap181
					d29 = snap182
					d30 = snap183
					d31 = snap184
					d32 = snap185
					d33 = snap186
					d34 = snap187
					d35 = snap188
					d36 = snap189
					d37 = snap190
					d38 = snap191
					d39 = snap192
					d40 = snap193
					d41 = snap194
					d42 = snap195
					d43 = snap196
					d44 = snap197
					d45 = snap198
					d46 = snap199
					d84 = snap200
					d85 = snap201
					d86 = snap202
					d87 = snap203
					d88 = snap204
					d90 = snap205
					d91 = snap206
					d92 = snap207
					d93 = snap208
					d94 = snap209
					d95 = snap210
					d96 = snap211
					d97 = snap212
					d98 = snap213
					d100 = snap214
					d101 = snap215
					d102 = snap216
					d103 = snap217
					d104 = snap218
					d161 = snap219
					d162 = snap220
					d163 = snap221
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps166)
					}
					return result
					ctx.FreeDesc(&d162)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d223 := ps.PhiValues[0]
							ctx.EnsureDesc(&d223)
							ctx.EmitStoreToStack(d223, int32(bbs[7].PhiBase)+int32(0))
						}
						if bbs[7].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d224 JITValueDesc
					if d1.Loc == LocImm {
						d224 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d224 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d224)
					}
					if d224.Loc == LocReg && d1.Loc == LocReg && d224.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d224)
					ctx.EmitStoreToStack(d224, int32(bbs[7].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d224)
					ctx.FreeDesc(&d1)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d98)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d98)
					var d225 JITValueDesc
					if d224.Loc == LocImm && d98.Loc == LocImm {
						d225 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d224.Imm.Int() < d98.Imm.Int())}
					} else if d98.Loc == LocImm {
						r24 := ctx.AllocRegExcept(d224.Reg)
						if d98.Imm.Int() >= -2147483648 && d98.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d224.Reg, int32(d98.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d98.Imm.Int()))
							ctx.EmitCmpInt64(d224.Reg, RegR11)
						}
						ctx.EmitSetcc(r24, CondSignedLess)
						d225 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r24}
						ctx.BindReg(r24, &d225)
					} else if d224.Loc == LocImm {
						r25 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d224.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d98.Reg)
						ctx.EmitSetcc(r25, CondSignedLess)
						d225 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r25}
						ctx.BindReg(r25, &d225)
					} else {
						r26 := ctx.AllocRegExcept(d224.Reg)
						ctx.EmitCmpInt64(d224.Reg, d98.Reg)
						ctx.EmitSetcc(r26, CondSignedLess)
						d225 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r26}
						ctx.BindReg(r26, &d225)
					}
					ctx.FreeDesc(&d98)
					d226 = d225
					ctx.EnsureDesc(&d226)
					if d226.Loc != LocImm && d226.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d226.Loc == LocImm {
						if d226.Imm.Bool() {
							if ps.General {
							}
							ps227 := PhiState{General: ps.General}
							ps227.OverlayValues = make([]JITValueDesc, 227)
							ps227.OverlayValues[1] = d1
							ps227.OverlayValues[2] = d2
							ps227.OverlayValues[3] = d3
							ps227.OverlayValues[4] = d4
							ps227.OverlayValues[5] = d5
							ps227.OverlayValues[6] = d6
							ps227.OverlayValues[7] = d7
							ps227.OverlayValues[8] = d8
							ps227.OverlayValues[9] = d9
							ps227.OverlayValues[24] = d24
							ps227.OverlayValues[25] = d25
							ps227.OverlayValues[26] = d26
							ps227.OverlayValues[27] = d27
							ps227.OverlayValues[28] = d28
							ps227.OverlayValues[29] = d29
							ps227.OverlayValues[30] = d30
							ps227.OverlayValues[31] = d31
							ps227.OverlayValues[32] = d32
							ps227.OverlayValues[33] = d33
							ps227.OverlayValues[34] = d34
							ps227.OverlayValues[35] = d35
							ps227.OverlayValues[36] = d36
							ps227.OverlayValues[37] = d37
							ps227.OverlayValues[38] = d38
							ps227.OverlayValues[39] = d39
							ps227.OverlayValues[40] = d40
							ps227.OverlayValues[41] = d41
							ps227.OverlayValues[42] = d42
							ps227.OverlayValues[43] = d43
							ps227.OverlayValues[44] = d44
							ps227.OverlayValues[45] = d45
							ps227.OverlayValues[46] = d46
							ps227.OverlayValues[84] = d84
							ps227.OverlayValues[85] = d85
							ps227.OverlayValues[86] = d86
							ps227.OverlayValues[87] = d87
							ps227.OverlayValues[88] = d88
							ps227.OverlayValues[90] = d90
							ps227.OverlayValues[91] = d91
							ps227.OverlayValues[92] = d92
							ps227.OverlayValues[93] = d93
							ps227.OverlayValues[94] = d94
							ps227.OverlayValues[95] = d95
							ps227.OverlayValues[96] = d96
							ps227.OverlayValues[97] = d97
							ps227.OverlayValues[98] = d98
							ps227.OverlayValues[100] = d100
							ps227.OverlayValues[101] = d101
							ps227.OverlayValues[102] = d102
							ps227.OverlayValues[103] = d103
							ps227.OverlayValues[104] = d104
							ps227.OverlayValues[161] = d161
							ps227.OverlayValues[162] = d162
							ps227.OverlayValues[163] = d163
							ps227.OverlayValues[223] = d223
							ps227.OverlayValues[224] = d224
							ps227.OverlayValues[225] = d225
							ps227.OverlayValues[226] = d226
							return bbs[8].RenderPS(ps227)
						}
						if ps.General {
						}
						ps228 := PhiState{General: ps.General}
						ps228.OverlayValues = make([]JITValueDesc, 227)
						ps228.OverlayValues[1] = d1
						ps228.OverlayValues[2] = d2
						ps228.OverlayValues[3] = d3
						ps228.OverlayValues[4] = d4
						ps228.OverlayValues[5] = d5
						ps228.OverlayValues[6] = d6
						ps228.OverlayValues[7] = d7
						ps228.OverlayValues[8] = d8
						ps228.OverlayValues[9] = d9
						ps228.OverlayValues[24] = d24
						ps228.OverlayValues[25] = d25
						ps228.OverlayValues[26] = d26
						ps228.OverlayValues[27] = d27
						ps228.OverlayValues[28] = d28
						ps228.OverlayValues[29] = d29
						ps228.OverlayValues[30] = d30
						ps228.OverlayValues[31] = d31
						ps228.OverlayValues[32] = d32
						ps228.OverlayValues[33] = d33
						ps228.OverlayValues[34] = d34
						ps228.OverlayValues[35] = d35
						ps228.OverlayValues[36] = d36
						ps228.OverlayValues[37] = d37
						ps228.OverlayValues[38] = d38
						ps228.OverlayValues[39] = d39
						ps228.OverlayValues[40] = d40
						ps228.OverlayValues[41] = d41
						ps228.OverlayValues[42] = d42
						ps228.OverlayValues[43] = d43
						ps228.OverlayValues[44] = d44
						ps228.OverlayValues[45] = d45
						ps228.OverlayValues[46] = d46
						ps228.OverlayValues[84] = d84
						ps228.OverlayValues[85] = d85
						ps228.OverlayValues[86] = d86
						ps228.OverlayValues[87] = d87
						ps228.OverlayValues[88] = d88
						ps228.OverlayValues[90] = d90
						ps228.OverlayValues[91] = d91
						ps228.OverlayValues[92] = d92
						ps228.OverlayValues[93] = d93
						ps228.OverlayValues[94] = d94
						ps228.OverlayValues[95] = d95
						ps228.OverlayValues[96] = d96
						ps228.OverlayValues[97] = d97
						ps228.OverlayValues[98] = d98
						ps228.OverlayValues[100] = d100
						ps228.OverlayValues[101] = d101
						ps228.OverlayValues[102] = d102
						ps228.OverlayValues[103] = d103
						ps228.OverlayValues[104] = d104
						ps228.OverlayValues[161] = d161
						ps228.OverlayValues[162] = d162
						ps228.OverlayValues[163] = d163
						ps228.OverlayValues[223] = d223
						ps228.OverlayValues[224] = d224
						ps228.OverlayValues[225] = d225
						ps228.OverlayValues[226] = d226
						return bbs[9].RenderPS(ps228)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d229 := ps.PhiValues[0]
							ctx.EnsureDesc(&d229)
							ctx.EmitStoreToStack(d229, int32(bbs[7].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl23 := ctx.ReserveLabel()
					lbl24 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d226.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl23)
					ctx.EmitJmp(lbl24)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl10)
					ps230 := PhiState{General: true}
					ps230.OverlayValues = make([]JITValueDesc, 230)
					ps230.OverlayValues[1] = d1
					ps230.OverlayValues[2] = d2
					ps230.OverlayValues[3] = d3
					ps230.OverlayValues[4] = d4
					ps230.OverlayValues[5] = d5
					ps230.OverlayValues[6] = d6
					ps230.OverlayValues[7] = d7
					ps230.OverlayValues[8] = d8
					ps230.OverlayValues[9] = d9
					ps230.OverlayValues[24] = d24
					ps230.OverlayValues[25] = d25
					ps230.OverlayValues[26] = d26
					ps230.OverlayValues[27] = d27
					ps230.OverlayValues[28] = d28
					ps230.OverlayValues[29] = d29
					ps230.OverlayValues[30] = d30
					ps230.OverlayValues[31] = d31
					ps230.OverlayValues[32] = d32
					ps230.OverlayValues[33] = d33
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
					ps230.OverlayValues[84] = d84
					ps230.OverlayValues[85] = d85
					ps230.OverlayValues[86] = d86
					ps230.OverlayValues[87] = d87
					ps230.OverlayValues[88] = d88
					ps230.OverlayValues[90] = d90
					ps230.OverlayValues[91] = d91
					ps230.OverlayValues[92] = d92
					ps230.OverlayValues[93] = d93
					ps230.OverlayValues[94] = d94
					ps230.OverlayValues[95] = d95
					ps230.OverlayValues[96] = d96
					ps230.OverlayValues[97] = d97
					ps230.OverlayValues[98] = d98
					ps230.OverlayValues[100] = d100
					ps230.OverlayValues[101] = d101
					ps230.OverlayValues[102] = d102
					ps230.OverlayValues[103] = d103
					ps230.OverlayValues[104] = d104
					ps230.OverlayValues[161] = d161
					ps230.OverlayValues[162] = d162
					ps230.OverlayValues[163] = d163
					ps230.OverlayValues[223] = d223
					ps230.OverlayValues[224] = d224
					ps230.OverlayValues[225] = d225
					ps230.OverlayValues[226] = d226
					ps230.OverlayValues[229] = d229
					ps231 := PhiState{General: true}
					ps231.OverlayValues = make([]JITValueDesc, 230)
					ps231.OverlayValues[1] = d1
					ps231.OverlayValues[2] = d2
					ps231.OverlayValues[3] = d3
					ps231.OverlayValues[4] = d4
					ps231.OverlayValues[5] = d5
					ps231.OverlayValues[6] = d6
					ps231.OverlayValues[7] = d7
					ps231.OverlayValues[8] = d8
					ps231.OverlayValues[9] = d9
					ps231.OverlayValues[24] = d24
					ps231.OverlayValues[25] = d25
					ps231.OverlayValues[26] = d26
					ps231.OverlayValues[27] = d27
					ps231.OverlayValues[28] = d28
					ps231.OverlayValues[29] = d29
					ps231.OverlayValues[30] = d30
					ps231.OverlayValues[31] = d31
					ps231.OverlayValues[32] = d32
					ps231.OverlayValues[33] = d33
					ps231.OverlayValues[34] = d34
					ps231.OverlayValues[35] = d35
					ps231.OverlayValues[36] = d36
					ps231.OverlayValues[37] = d37
					ps231.OverlayValues[38] = d38
					ps231.OverlayValues[39] = d39
					ps231.OverlayValues[40] = d40
					ps231.OverlayValues[41] = d41
					ps231.OverlayValues[42] = d42
					ps231.OverlayValues[43] = d43
					ps231.OverlayValues[44] = d44
					ps231.OverlayValues[45] = d45
					ps231.OverlayValues[46] = d46
					ps231.OverlayValues[84] = d84
					ps231.OverlayValues[85] = d85
					ps231.OverlayValues[86] = d86
					ps231.OverlayValues[87] = d87
					ps231.OverlayValues[88] = d88
					ps231.OverlayValues[90] = d90
					ps231.OverlayValues[91] = d91
					ps231.OverlayValues[92] = d92
					ps231.OverlayValues[93] = d93
					ps231.OverlayValues[94] = d94
					ps231.OverlayValues[95] = d95
					ps231.OverlayValues[96] = d96
					ps231.OverlayValues[97] = d97
					ps231.OverlayValues[98] = d98
					ps231.OverlayValues[100] = d100
					ps231.OverlayValues[101] = d101
					ps231.OverlayValues[102] = d102
					ps231.OverlayValues[103] = d103
					ps231.OverlayValues[104] = d104
					ps231.OverlayValues[161] = d161
					ps231.OverlayValues[162] = d162
					ps231.OverlayValues[163] = d163
					ps231.OverlayValues[223] = d223
					ps231.OverlayValues[224] = d224
					ps231.OverlayValues[225] = d225
					ps231.OverlayValues[226] = d226
					ps231.OverlayValues[229] = d229
					snap232 := d1
					snap233 := d2
					snap234 := d3
					snap235 := d4
					snap236 := d5
					snap237 := d6
					snap238 := d7
					snap239 := d8
					snap240 := d9
					snap241 := d24
					snap242 := d25
					snap243 := d26
					snap244 := d27
					snap245 := d28
					snap246 := d29
					snap247 := d30
					snap248 := d31
					snap249 := d32
					snap250 := d33
					snap251 := d34
					snap252 := d35
					snap253 := d36
					snap254 := d37
					snap255 := d38
					snap256 := d39
					snap257 := d40
					snap258 := d41
					snap259 := d42
					snap260 := d43
					snap261 := d44
					snap262 := d45
					snap263 := d46
					snap264 := d84
					snap265 := d85
					snap266 := d86
					snap267 := d87
					snap268 := d88
					snap269 := d90
					snap270 := d91
					snap271 := d92
					snap272 := d93
					snap273 := d94
					snap274 := d95
					snap275 := d96
					snap276 := d97
					snap277 := d98
					snap278 := d100
					snap279 := d101
					snap280 := d102
					snap281 := d103
					snap282 := d104
					snap283 := d161
					snap284 := d162
					snap285 := d163
					snap286 := d223
					snap287 := d224
					snap288 := d225
					snap289 := d226
					snap290 := d229
					alloc291 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps231)
					}
					ctx.RestoreAllocState(alloc291)
					d1 = snap232
					d2 = snap233
					d3 = snap234
					d4 = snap235
					d5 = snap236
					d6 = snap237
					d7 = snap238
					d8 = snap239
					d9 = snap240
					d24 = snap241
					d25 = snap242
					d26 = snap243
					d27 = snap244
					d28 = snap245
					d29 = snap246
					d30 = snap247
					d31 = snap248
					d32 = snap249
					d33 = snap250
					d34 = snap251
					d35 = snap252
					d36 = snap253
					d37 = snap254
					d38 = snap255
					d39 = snap256
					d40 = snap257
					d41 = snap258
					d42 = snap259
					d43 = snap260
					d44 = snap261
					d45 = snap262
					d46 = snap263
					d84 = snap264
					d85 = snap265
					d86 = snap266
					d87 = snap267
					d88 = snap268
					d90 = snap269
					d91 = snap270
					d92 = snap271
					d93 = snap272
					d94 = snap273
					d95 = snap274
					d96 = snap275
					d97 = snap276
					d98 = snap277
					d100 = snap278
					d101 = snap279
					d102 = snap280
					d103 = snap281
					d104 = snap282
					d161 = snap283
					d162 = snap284
					d163 = snap285
					d223 = snap286
					d224 = snap287
					d225 = snap288
					d226 = snap289
					d229 = snap290
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps230)
					}
					return result
					ctx.FreeDesc(&d225)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					ctx.ReclaimUntrackedRegs()
					var d292 JITValueDesc
					if d6.SliceSizeKnown {
						d292 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.KnownSliceLen))}
					} else if d6.Loc == LocImm {
						d292 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d6.StackOff))}
					} else if d6.Loc == LocStackTriple {
						d292 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d6.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d6)
						if d6.Loc == LocRegPair || d6.Loc == LocRegTriple {
							d292 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2, ID: 0}
						} else if d6.Loc == LocReg {
							d292 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d292)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d292)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d292)
					var d293 JITValueDesc
					if d224.Loc == LocImm && d292.Loc == LocImm {
						d293 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d224.Imm.Int() < d292.Imm.Int())}
					} else if d292.Loc == LocImm {
						r27 := ctx.AllocRegExcept(d224.Reg)
						if d292.Imm.Int() >= -2147483648 && d292.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d224.Reg, int32(d292.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d292.Imm.Int()))
							ctx.EmitCmpInt64(d224.Reg, RegR11)
						}
						ctx.EmitSetcc(r27, CondSignedLess)
						d293 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r27}
						ctx.BindReg(r27, &d293)
					} else if d224.Loc == LocImm {
						r28 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d224.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d292.Reg)
						ctx.EmitSetcc(r28, CondSignedLess)
						d293 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r28}
						ctx.BindReg(r28, &d293)
					} else {
						r29 := ctx.AllocRegExcept(d224.Reg)
						ctx.EmitCmpInt64(d224.Reg, d292.Reg)
						ctx.EmitSetcc(r29, CondSignedLess)
						d293 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r29}
						ctx.BindReg(r29, &d293)
					}
					ctx.FreeDesc(&d292)
					d294 = d293
					ctx.EnsureDesc(&d294)
					if d294.Loc != LocImm && d294.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d294.Loc == LocImm {
						if d294.Imm.Bool() {
							if ps.General {
							}
							ps295 := PhiState{General: ps.General}
							ps295.OverlayValues = make([]JITValueDesc, 295)
							ps295.OverlayValues[1] = d1
							ps295.OverlayValues[2] = d2
							ps295.OverlayValues[3] = d3
							ps295.OverlayValues[4] = d4
							ps295.OverlayValues[5] = d5
							ps295.OverlayValues[6] = d6
							ps295.OverlayValues[7] = d7
							ps295.OverlayValues[8] = d8
							ps295.OverlayValues[9] = d9
							ps295.OverlayValues[24] = d24
							ps295.OverlayValues[25] = d25
							ps295.OverlayValues[26] = d26
							ps295.OverlayValues[27] = d27
							ps295.OverlayValues[28] = d28
							ps295.OverlayValues[29] = d29
							ps295.OverlayValues[30] = d30
							ps295.OverlayValues[31] = d31
							ps295.OverlayValues[32] = d32
							ps295.OverlayValues[33] = d33
							ps295.OverlayValues[34] = d34
							ps295.OverlayValues[35] = d35
							ps295.OverlayValues[36] = d36
							ps295.OverlayValues[37] = d37
							ps295.OverlayValues[38] = d38
							ps295.OverlayValues[39] = d39
							ps295.OverlayValues[40] = d40
							ps295.OverlayValues[41] = d41
							ps295.OverlayValues[42] = d42
							ps295.OverlayValues[43] = d43
							ps295.OverlayValues[44] = d44
							ps295.OverlayValues[45] = d45
							ps295.OverlayValues[46] = d46
							ps295.OverlayValues[84] = d84
							ps295.OverlayValues[85] = d85
							ps295.OverlayValues[86] = d86
							ps295.OverlayValues[87] = d87
							ps295.OverlayValues[88] = d88
							ps295.OverlayValues[90] = d90
							ps295.OverlayValues[91] = d91
							ps295.OverlayValues[92] = d92
							ps295.OverlayValues[93] = d93
							ps295.OverlayValues[94] = d94
							ps295.OverlayValues[95] = d95
							ps295.OverlayValues[96] = d96
							ps295.OverlayValues[97] = d97
							ps295.OverlayValues[98] = d98
							ps295.OverlayValues[100] = d100
							ps295.OverlayValues[101] = d101
							ps295.OverlayValues[102] = d102
							ps295.OverlayValues[103] = d103
							ps295.OverlayValues[104] = d104
							ps295.OverlayValues[161] = d161
							ps295.OverlayValues[162] = d162
							ps295.OverlayValues[163] = d163
							ps295.OverlayValues[223] = d223
							ps295.OverlayValues[224] = d224
							ps295.OverlayValues[225] = d225
							ps295.OverlayValues[226] = d226
							ps295.OverlayValues[229] = d229
							ps295.OverlayValues[292] = d292
							ps295.OverlayValues[293] = d293
							ps295.OverlayValues[294] = d294
							return bbs[10].RenderPS(ps295)
						}
						if ps.General {
						}
						ps296 := PhiState{General: ps.General}
						ps296.OverlayValues = make([]JITValueDesc, 295)
						ps296.OverlayValues[1] = d1
						ps296.OverlayValues[2] = d2
						ps296.OverlayValues[3] = d3
						ps296.OverlayValues[4] = d4
						ps296.OverlayValues[5] = d5
						ps296.OverlayValues[6] = d6
						ps296.OverlayValues[7] = d7
						ps296.OverlayValues[8] = d8
						ps296.OverlayValues[9] = d9
						ps296.OverlayValues[24] = d24
						ps296.OverlayValues[25] = d25
						ps296.OverlayValues[26] = d26
						ps296.OverlayValues[27] = d27
						ps296.OverlayValues[28] = d28
						ps296.OverlayValues[29] = d29
						ps296.OverlayValues[30] = d30
						ps296.OverlayValues[31] = d31
						ps296.OverlayValues[32] = d32
						ps296.OverlayValues[33] = d33
						ps296.OverlayValues[34] = d34
						ps296.OverlayValues[35] = d35
						ps296.OverlayValues[36] = d36
						ps296.OverlayValues[37] = d37
						ps296.OverlayValues[38] = d38
						ps296.OverlayValues[39] = d39
						ps296.OverlayValues[40] = d40
						ps296.OverlayValues[41] = d41
						ps296.OverlayValues[42] = d42
						ps296.OverlayValues[43] = d43
						ps296.OverlayValues[44] = d44
						ps296.OverlayValues[45] = d45
						ps296.OverlayValues[46] = d46
						ps296.OverlayValues[84] = d84
						ps296.OverlayValues[85] = d85
						ps296.OverlayValues[86] = d86
						ps296.OverlayValues[87] = d87
						ps296.OverlayValues[88] = d88
						ps296.OverlayValues[90] = d90
						ps296.OverlayValues[91] = d91
						ps296.OverlayValues[92] = d92
						ps296.OverlayValues[93] = d93
						ps296.OverlayValues[94] = d94
						ps296.OverlayValues[95] = d95
						ps296.OverlayValues[96] = d96
						ps296.OverlayValues[97] = d97
						ps296.OverlayValues[98] = d98
						ps296.OverlayValues[100] = d100
						ps296.OverlayValues[101] = d101
						ps296.OverlayValues[102] = d102
						ps296.OverlayValues[103] = d103
						ps296.OverlayValues[104] = d104
						ps296.OverlayValues[161] = d161
						ps296.OverlayValues[162] = d162
						ps296.OverlayValues[163] = d163
						ps296.OverlayValues[223] = d223
						ps296.OverlayValues[224] = d224
						ps296.OverlayValues[225] = d225
						ps296.OverlayValues[226] = d226
						ps296.OverlayValues[229] = d229
						ps296.OverlayValues[292] = d292
						ps296.OverlayValues[293] = d293
						ps296.OverlayValues[294] = d294
						return bbs[11].RenderPS(ps296)
					}
					if !ps.General {
						ps.General = true
						return bbs[8].RenderPS(ps)
					}
					lbl25 := ctx.ReserveLabel()
					lbl26 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d294.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl25)
					ctx.EmitJmp(lbl26)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl11)
					ctx.MarkLabel(lbl26)
					ctx.EmitJmp(lbl12)
					ps297 := PhiState{General: true}
					ps297.OverlayValues = make([]JITValueDesc, 295)
					ps297.OverlayValues[1] = d1
					ps297.OverlayValues[2] = d2
					ps297.OverlayValues[3] = d3
					ps297.OverlayValues[4] = d4
					ps297.OverlayValues[5] = d5
					ps297.OverlayValues[6] = d6
					ps297.OverlayValues[7] = d7
					ps297.OverlayValues[8] = d8
					ps297.OverlayValues[9] = d9
					ps297.OverlayValues[24] = d24
					ps297.OverlayValues[25] = d25
					ps297.OverlayValues[26] = d26
					ps297.OverlayValues[27] = d27
					ps297.OverlayValues[28] = d28
					ps297.OverlayValues[29] = d29
					ps297.OverlayValues[30] = d30
					ps297.OverlayValues[31] = d31
					ps297.OverlayValues[32] = d32
					ps297.OverlayValues[33] = d33
					ps297.OverlayValues[34] = d34
					ps297.OverlayValues[35] = d35
					ps297.OverlayValues[36] = d36
					ps297.OverlayValues[37] = d37
					ps297.OverlayValues[38] = d38
					ps297.OverlayValues[39] = d39
					ps297.OverlayValues[40] = d40
					ps297.OverlayValues[41] = d41
					ps297.OverlayValues[42] = d42
					ps297.OverlayValues[43] = d43
					ps297.OverlayValues[44] = d44
					ps297.OverlayValues[45] = d45
					ps297.OverlayValues[46] = d46
					ps297.OverlayValues[84] = d84
					ps297.OverlayValues[85] = d85
					ps297.OverlayValues[86] = d86
					ps297.OverlayValues[87] = d87
					ps297.OverlayValues[88] = d88
					ps297.OverlayValues[90] = d90
					ps297.OverlayValues[91] = d91
					ps297.OverlayValues[92] = d92
					ps297.OverlayValues[93] = d93
					ps297.OverlayValues[94] = d94
					ps297.OverlayValues[95] = d95
					ps297.OverlayValues[96] = d96
					ps297.OverlayValues[97] = d97
					ps297.OverlayValues[98] = d98
					ps297.OverlayValues[100] = d100
					ps297.OverlayValues[101] = d101
					ps297.OverlayValues[102] = d102
					ps297.OverlayValues[103] = d103
					ps297.OverlayValues[104] = d104
					ps297.OverlayValues[161] = d161
					ps297.OverlayValues[162] = d162
					ps297.OverlayValues[163] = d163
					ps297.OverlayValues[223] = d223
					ps297.OverlayValues[224] = d224
					ps297.OverlayValues[225] = d225
					ps297.OverlayValues[226] = d226
					ps297.OverlayValues[229] = d229
					ps297.OverlayValues[292] = d292
					ps297.OverlayValues[293] = d293
					ps297.OverlayValues[294] = d294
					ps298 := PhiState{General: true}
					ps298.OverlayValues = make([]JITValueDesc, 295)
					ps298.OverlayValues[1] = d1
					ps298.OverlayValues[2] = d2
					ps298.OverlayValues[3] = d3
					ps298.OverlayValues[4] = d4
					ps298.OverlayValues[5] = d5
					ps298.OverlayValues[6] = d6
					ps298.OverlayValues[7] = d7
					ps298.OverlayValues[8] = d8
					ps298.OverlayValues[9] = d9
					ps298.OverlayValues[24] = d24
					ps298.OverlayValues[25] = d25
					ps298.OverlayValues[26] = d26
					ps298.OverlayValues[27] = d27
					ps298.OverlayValues[28] = d28
					ps298.OverlayValues[29] = d29
					ps298.OverlayValues[30] = d30
					ps298.OverlayValues[31] = d31
					ps298.OverlayValues[32] = d32
					ps298.OverlayValues[33] = d33
					ps298.OverlayValues[34] = d34
					ps298.OverlayValues[35] = d35
					ps298.OverlayValues[36] = d36
					ps298.OverlayValues[37] = d37
					ps298.OverlayValues[38] = d38
					ps298.OverlayValues[39] = d39
					ps298.OverlayValues[40] = d40
					ps298.OverlayValues[41] = d41
					ps298.OverlayValues[42] = d42
					ps298.OverlayValues[43] = d43
					ps298.OverlayValues[44] = d44
					ps298.OverlayValues[45] = d45
					ps298.OverlayValues[46] = d46
					ps298.OverlayValues[84] = d84
					ps298.OverlayValues[85] = d85
					ps298.OverlayValues[86] = d86
					ps298.OverlayValues[87] = d87
					ps298.OverlayValues[88] = d88
					ps298.OverlayValues[90] = d90
					ps298.OverlayValues[91] = d91
					ps298.OverlayValues[92] = d92
					ps298.OverlayValues[93] = d93
					ps298.OverlayValues[94] = d94
					ps298.OverlayValues[95] = d95
					ps298.OverlayValues[96] = d96
					ps298.OverlayValues[97] = d97
					ps298.OverlayValues[98] = d98
					ps298.OverlayValues[100] = d100
					ps298.OverlayValues[101] = d101
					ps298.OverlayValues[102] = d102
					ps298.OverlayValues[103] = d103
					ps298.OverlayValues[104] = d104
					ps298.OverlayValues[161] = d161
					ps298.OverlayValues[162] = d162
					ps298.OverlayValues[163] = d163
					ps298.OverlayValues[223] = d223
					ps298.OverlayValues[224] = d224
					ps298.OverlayValues[225] = d225
					ps298.OverlayValues[226] = d226
					ps298.OverlayValues[229] = d229
					ps298.OverlayValues[292] = d292
					ps298.OverlayValues[293] = d293
					ps298.OverlayValues[294] = d294
					snap299 := d1
					snap300 := d2
					snap301 := d3
					snap302 := d4
					snap303 := d5
					snap304 := d6
					snap305 := d7
					snap306 := d8
					snap307 := d9
					snap308 := d24
					snap309 := d25
					snap310 := d26
					snap311 := d27
					snap312 := d28
					snap313 := d29
					snap314 := d30
					snap315 := d31
					snap316 := d32
					snap317 := d33
					snap318 := d34
					snap319 := d35
					snap320 := d36
					snap321 := d37
					snap322 := d38
					snap323 := d39
					snap324 := d40
					snap325 := d41
					snap326 := d42
					snap327 := d43
					snap328 := d44
					snap329 := d45
					snap330 := d46
					snap331 := d84
					snap332 := d85
					snap333 := d86
					snap334 := d87
					snap335 := d88
					snap336 := d90
					snap337 := d91
					snap338 := d92
					snap339 := d93
					snap340 := d94
					snap341 := d95
					snap342 := d96
					snap343 := d97
					snap344 := d98
					snap345 := d100
					snap346 := d101
					snap347 := d102
					snap348 := d103
					snap349 := d104
					snap350 := d161
					snap351 := d162
					snap352 := d163
					snap353 := d223
					snap354 := d224
					snap355 := d225
					snap356 := d226
					snap357 := d229
					snap358 := d292
					snap359 := d293
					snap360 := d294
					alloc361 := ctx.SnapshotAllocState()
					if !bbs[11].Rendered {
						bbs[11].RenderPS(ps298)
					}
					ctx.RestoreAllocState(alloc361)
					d1 = snap299
					d2 = snap300
					d3 = snap301
					d4 = snap302
					d5 = snap303
					d6 = snap304
					d7 = snap305
					d8 = snap306
					d9 = snap307
					d24 = snap308
					d25 = snap309
					d26 = snap310
					d27 = snap311
					d28 = snap312
					d29 = snap313
					d30 = snap314
					d31 = snap315
					d32 = snap316
					d33 = snap317
					d34 = snap318
					d35 = snap319
					d36 = snap320
					d37 = snap321
					d38 = snap322
					d39 = snap323
					d40 = snap324
					d41 = snap325
					d42 = snap326
					d43 = snap327
					d44 = snap328
					d45 = snap329
					d46 = snap330
					d84 = snap331
					d85 = snap332
					d86 = snap333
					d87 = snap334
					d88 = snap335
					d90 = snap336
					d91 = snap337
					d92 = snap338
					d93 = snap339
					d94 = snap340
					d95 = snap341
					d96 = snap342
					d97 = snap343
					d98 = snap344
					d100 = snap345
					d101 = snap346
					d102 = snap347
					d103 = snap348
					d104 = snap349
					d161 = snap350
					d162 = snap351
					d163 = snap352
					d223 = snap353
					d224 = snap354
					d225 = snap355
					d226 = snap356
					d229 = snap357
					d292 = snap358
					d293 = snap359
					d294 = snap360
					if !bbs[10].Rendered {
						return bbs[10].RenderPS(ps297)
					}
					return result
					ctx.FreeDesc(&d293)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d32)
					ctx.EnsureDesc(&d32)
					var d362 JITValueDesc
					if d32.Loc == LocImm {
						d362 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d32.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d32.Reg)
						ctx.EmitMovRegReg(scratch, d32.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d362 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d362)
					}
					if d362.Loc == LocReg && d32.Loc == LocReg && d362.Reg == d32.Reg {
						ctx.TransferReg(d32.Reg)
						d32.Loc = LocNone
					}
					ctx.FreeDesc(&d32)
					ctx.EnsureDesc(&d362)
					ctx.EnsureDesc(&d362)
					ctx.EnsureDesc(&d362)
					d364 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.EnsureDesc(&d362)
					d365 = ctx.EmitSliceElementAddress(&d3, &d364, int32(16))
					ctx.EmitStoreScmerAt(&d365, &d362)
					ctx.FreeDesc(&d365)
					ctx.EnsureDesc(&d27)
					var d366 JITValueDesc
					if d27.Loc == LocImm {
						d366 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d27.Imm.Int() > 0)}
					} else {
						r30 := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitCmpRegImm32(d27.Reg, 0)
						ctx.EmitSetcc(r30, CondSignedGreater)
						d366 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r30}
						ctx.BindReg(r30, &d366)
					}
					d367 = d366
					ctx.EnsureDesc(&d367)
					if d367.Loc != LocImm && d367.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d367.Loc == LocImm {
						if d367.Imm.Bool() {
							if ps.General {
							}
							ps368 := PhiState{General: ps.General}
							ps368.OverlayValues = make([]JITValueDesc, 368)
							ps368.OverlayValues[1] = d1
							ps368.OverlayValues[2] = d2
							ps368.OverlayValues[3] = d3
							ps368.OverlayValues[4] = d4
							ps368.OverlayValues[5] = d5
							ps368.OverlayValues[6] = d6
							ps368.OverlayValues[7] = d7
							ps368.OverlayValues[8] = d8
							ps368.OverlayValues[9] = d9
							ps368.OverlayValues[24] = d24
							ps368.OverlayValues[25] = d25
							ps368.OverlayValues[26] = d26
							ps368.OverlayValues[27] = d27
							ps368.OverlayValues[28] = d28
							ps368.OverlayValues[29] = d29
							ps368.OverlayValues[30] = d30
							ps368.OverlayValues[31] = d31
							ps368.OverlayValues[32] = d32
							ps368.OverlayValues[33] = d33
							ps368.OverlayValues[34] = d34
							ps368.OverlayValues[35] = d35
							ps368.OverlayValues[36] = d36
							ps368.OverlayValues[37] = d37
							ps368.OverlayValues[38] = d38
							ps368.OverlayValues[39] = d39
							ps368.OverlayValues[40] = d40
							ps368.OverlayValues[41] = d41
							ps368.OverlayValues[42] = d42
							ps368.OverlayValues[43] = d43
							ps368.OverlayValues[44] = d44
							ps368.OverlayValues[45] = d45
							ps368.OverlayValues[46] = d46
							ps368.OverlayValues[84] = d84
							ps368.OverlayValues[85] = d85
							ps368.OverlayValues[86] = d86
							ps368.OverlayValues[87] = d87
							ps368.OverlayValues[88] = d88
							ps368.OverlayValues[90] = d90
							ps368.OverlayValues[91] = d91
							ps368.OverlayValues[92] = d92
							ps368.OverlayValues[93] = d93
							ps368.OverlayValues[94] = d94
							ps368.OverlayValues[95] = d95
							ps368.OverlayValues[96] = d96
							ps368.OverlayValues[97] = d97
							ps368.OverlayValues[98] = d98
							ps368.OverlayValues[100] = d100
							ps368.OverlayValues[101] = d101
							ps368.OverlayValues[102] = d102
							ps368.OverlayValues[103] = d103
							ps368.OverlayValues[104] = d104
							ps368.OverlayValues[161] = d161
							ps368.OverlayValues[162] = d162
							ps368.OverlayValues[163] = d163
							ps368.OverlayValues[223] = d223
							ps368.OverlayValues[224] = d224
							ps368.OverlayValues[225] = d225
							ps368.OverlayValues[226] = d226
							ps368.OverlayValues[229] = d229
							ps368.OverlayValues[292] = d292
							ps368.OverlayValues[293] = d293
							ps368.OverlayValues[294] = d294
							ps368.OverlayValues[362] = d362
							ps368.OverlayValues[363] = d363
							ps368.OverlayValues[364] = d364
							ps368.OverlayValues[365] = d365
							ps368.OverlayValues[366] = d366
							ps368.OverlayValues[367] = d367
							return bbs[12].RenderPS(ps368)
						}
						if ps.General {
						}
						ps369 := PhiState{General: ps.General}
						ps369.OverlayValues = make([]JITValueDesc, 368)
						ps369.OverlayValues[1] = d1
						ps369.OverlayValues[2] = d2
						ps369.OverlayValues[3] = d3
						ps369.OverlayValues[4] = d4
						ps369.OverlayValues[5] = d5
						ps369.OverlayValues[6] = d6
						ps369.OverlayValues[7] = d7
						ps369.OverlayValues[8] = d8
						ps369.OverlayValues[9] = d9
						ps369.OverlayValues[24] = d24
						ps369.OverlayValues[25] = d25
						ps369.OverlayValues[26] = d26
						ps369.OverlayValues[27] = d27
						ps369.OverlayValues[28] = d28
						ps369.OverlayValues[29] = d29
						ps369.OverlayValues[30] = d30
						ps369.OverlayValues[31] = d31
						ps369.OverlayValues[32] = d32
						ps369.OverlayValues[33] = d33
						ps369.OverlayValues[34] = d34
						ps369.OverlayValues[35] = d35
						ps369.OverlayValues[36] = d36
						ps369.OverlayValues[37] = d37
						ps369.OverlayValues[38] = d38
						ps369.OverlayValues[39] = d39
						ps369.OverlayValues[40] = d40
						ps369.OverlayValues[41] = d41
						ps369.OverlayValues[42] = d42
						ps369.OverlayValues[43] = d43
						ps369.OverlayValues[44] = d44
						ps369.OverlayValues[45] = d45
						ps369.OverlayValues[46] = d46
						ps369.OverlayValues[84] = d84
						ps369.OverlayValues[85] = d85
						ps369.OverlayValues[86] = d86
						ps369.OverlayValues[87] = d87
						ps369.OverlayValues[88] = d88
						ps369.OverlayValues[90] = d90
						ps369.OverlayValues[91] = d91
						ps369.OverlayValues[92] = d92
						ps369.OverlayValues[93] = d93
						ps369.OverlayValues[94] = d94
						ps369.OverlayValues[95] = d95
						ps369.OverlayValues[96] = d96
						ps369.OverlayValues[97] = d97
						ps369.OverlayValues[98] = d98
						ps369.OverlayValues[100] = d100
						ps369.OverlayValues[101] = d101
						ps369.OverlayValues[102] = d102
						ps369.OverlayValues[103] = d103
						ps369.OverlayValues[104] = d104
						ps369.OverlayValues[161] = d161
						ps369.OverlayValues[162] = d162
						ps369.OverlayValues[163] = d163
						ps369.OverlayValues[223] = d223
						ps369.OverlayValues[224] = d224
						ps369.OverlayValues[225] = d225
						ps369.OverlayValues[226] = d226
						ps369.OverlayValues[229] = d229
						ps369.OverlayValues[292] = d292
						ps369.OverlayValues[293] = d293
						ps369.OverlayValues[294] = d294
						ps369.OverlayValues[362] = d362
						ps369.OverlayValues[363] = d363
						ps369.OverlayValues[364] = d364
						ps369.OverlayValues[365] = d365
						ps369.OverlayValues[366] = d366
						ps369.OverlayValues[367] = d367
						return bbs[13].RenderPS(ps369)
					}
					if !ps.General {
						ps.General = true
						return bbs[9].RenderPS(ps)
					}
					lbl27 := ctx.ReserveLabel()
					lbl28 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d367.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl27)
					ctx.EmitJmp(lbl28)
					ctx.MarkLabel(lbl27)
					ctx.EmitJmp(lbl13)
					ctx.MarkLabel(lbl28)
					ctx.EmitJmp(lbl14)
					ps370 := PhiState{General: true}
					ps370.OverlayValues = make([]JITValueDesc, 368)
					ps370.OverlayValues[1] = d1
					ps370.OverlayValues[2] = d2
					ps370.OverlayValues[3] = d3
					ps370.OverlayValues[4] = d4
					ps370.OverlayValues[5] = d5
					ps370.OverlayValues[6] = d6
					ps370.OverlayValues[7] = d7
					ps370.OverlayValues[8] = d8
					ps370.OverlayValues[9] = d9
					ps370.OverlayValues[24] = d24
					ps370.OverlayValues[25] = d25
					ps370.OverlayValues[26] = d26
					ps370.OverlayValues[27] = d27
					ps370.OverlayValues[28] = d28
					ps370.OverlayValues[29] = d29
					ps370.OverlayValues[30] = d30
					ps370.OverlayValues[31] = d31
					ps370.OverlayValues[32] = d32
					ps370.OverlayValues[33] = d33
					ps370.OverlayValues[34] = d34
					ps370.OverlayValues[35] = d35
					ps370.OverlayValues[36] = d36
					ps370.OverlayValues[37] = d37
					ps370.OverlayValues[38] = d38
					ps370.OverlayValues[39] = d39
					ps370.OverlayValues[40] = d40
					ps370.OverlayValues[41] = d41
					ps370.OverlayValues[42] = d42
					ps370.OverlayValues[43] = d43
					ps370.OverlayValues[44] = d44
					ps370.OverlayValues[45] = d45
					ps370.OverlayValues[46] = d46
					ps370.OverlayValues[84] = d84
					ps370.OverlayValues[85] = d85
					ps370.OverlayValues[86] = d86
					ps370.OverlayValues[87] = d87
					ps370.OverlayValues[88] = d88
					ps370.OverlayValues[90] = d90
					ps370.OverlayValues[91] = d91
					ps370.OverlayValues[92] = d92
					ps370.OverlayValues[93] = d93
					ps370.OverlayValues[94] = d94
					ps370.OverlayValues[95] = d95
					ps370.OverlayValues[96] = d96
					ps370.OverlayValues[97] = d97
					ps370.OverlayValues[98] = d98
					ps370.OverlayValues[100] = d100
					ps370.OverlayValues[101] = d101
					ps370.OverlayValues[102] = d102
					ps370.OverlayValues[103] = d103
					ps370.OverlayValues[104] = d104
					ps370.OverlayValues[161] = d161
					ps370.OverlayValues[162] = d162
					ps370.OverlayValues[163] = d163
					ps370.OverlayValues[223] = d223
					ps370.OverlayValues[224] = d224
					ps370.OverlayValues[225] = d225
					ps370.OverlayValues[226] = d226
					ps370.OverlayValues[229] = d229
					ps370.OverlayValues[292] = d292
					ps370.OverlayValues[293] = d293
					ps370.OverlayValues[294] = d294
					ps370.OverlayValues[362] = d362
					ps370.OverlayValues[363] = d363
					ps370.OverlayValues[364] = d364
					ps370.OverlayValues[365] = d365
					ps370.OverlayValues[366] = d366
					ps370.OverlayValues[367] = d367
					ps371 := PhiState{General: true}
					ps371.OverlayValues = make([]JITValueDesc, 368)
					ps371.OverlayValues[1] = d1
					ps371.OverlayValues[2] = d2
					ps371.OverlayValues[3] = d3
					ps371.OverlayValues[4] = d4
					ps371.OverlayValues[5] = d5
					ps371.OverlayValues[6] = d6
					ps371.OverlayValues[7] = d7
					ps371.OverlayValues[8] = d8
					ps371.OverlayValues[9] = d9
					ps371.OverlayValues[24] = d24
					ps371.OverlayValues[25] = d25
					ps371.OverlayValues[26] = d26
					ps371.OverlayValues[27] = d27
					ps371.OverlayValues[28] = d28
					ps371.OverlayValues[29] = d29
					ps371.OverlayValues[30] = d30
					ps371.OverlayValues[31] = d31
					ps371.OverlayValues[32] = d32
					ps371.OverlayValues[33] = d33
					ps371.OverlayValues[34] = d34
					ps371.OverlayValues[35] = d35
					ps371.OverlayValues[36] = d36
					ps371.OverlayValues[37] = d37
					ps371.OverlayValues[38] = d38
					ps371.OverlayValues[39] = d39
					ps371.OverlayValues[40] = d40
					ps371.OverlayValues[41] = d41
					ps371.OverlayValues[42] = d42
					ps371.OverlayValues[43] = d43
					ps371.OverlayValues[44] = d44
					ps371.OverlayValues[45] = d45
					ps371.OverlayValues[46] = d46
					ps371.OverlayValues[84] = d84
					ps371.OverlayValues[85] = d85
					ps371.OverlayValues[86] = d86
					ps371.OverlayValues[87] = d87
					ps371.OverlayValues[88] = d88
					ps371.OverlayValues[90] = d90
					ps371.OverlayValues[91] = d91
					ps371.OverlayValues[92] = d92
					ps371.OverlayValues[93] = d93
					ps371.OverlayValues[94] = d94
					ps371.OverlayValues[95] = d95
					ps371.OverlayValues[96] = d96
					ps371.OverlayValues[97] = d97
					ps371.OverlayValues[98] = d98
					ps371.OverlayValues[100] = d100
					ps371.OverlayValues[101] = d101
					ps371.OverlayValues[102] = d102
					ps371.OverlayValues[103] = d103
					ps371.OverlayValues[104] = d104
					ps371.OverlayValues[161] = d161
					ps371.OverlayValues[162] = d162
					ps371.OverlayValues[163] = d163
					ps371.OverlayValues[223] = d223
					ps371.OverlayValues[224] = d224
					ps371.OverlayValues[225] = d225
					ps371.OverlayValues[226] = d226
					ps371.OverlayValues[229] = d229
					ps371.OverlayValues[292] = d292
					ps371.OverlayValues[293] = d293
					ps371.OverlayValues[294] = d294
					ps371.OverlayValues[362] = d362
					ps371.OverlayValues[363] = d363
					ps371.OverlayValues[364] = d364
					ps371.OverlayValues[365] = d365
					ps371.OverlayValues[366] = d366
					ps371.OverlayValues[367] = d367
					snap372 := d1
					snap373 := d2
					snap374 := d3
					snap375 := d4
					snap376 := d5
					snap377 := d6
					snap378 := d7
					snap379 := d8
					snap380 := d9
					snap381 := d24
					snap382 := d25
					snap383 := d26
					snap384 := d27
					snap385 := d28
					snap386 := d29
					snap387 := d30
					snap388 := d31
					snap389 := d32
					snap390 := d33
					snap391 := d34
					snap392 := d35
					snap393 := d36
					snap394 := d37
					snap395 := d38
					snap396 := d39
					snap397 := d40
					snap398 := d41
					snap399 := d42
					snap400 := d43
					snap401 := d44
					snap402 := d45
					snap403 := d46
					snap404 := d84
					snap405 := d85
					snap406 := d86
					snap407 := d87
					snap408 := d88
					snap409 := d90
					snap410 := d91
					snap411 := d92
					snap412 := d93
					snap413 := d94
					snap414 := d95
					snap415 := d96
					snap416 := d97
					snap417 := d98
					snap418 := d100
					snap419 := d101
					snap420 := d102
					snap421 := d103
					snap422 := d104
					snap423 := d161
					snap424 := d162
					snap425 := d163
					snap426 := d223
					snap427 := d224
					snap428 := d225
					snap429 := d226
					snap430 := d229
					snap431 := d292
					snap432 := d293
					snap433 := d294
					snap434 := d362
					snap435 := d363
					snap436 := d364
					snap437 := d365
					snap438 := d366
					snap439 := d367
					alloc440 := ctx.SnapshotAllocState()
					if !bbs[13].Rendered {
						bbs[13].RenderPS(ps371)
					}
					ctx.RestoreAllocState(alloc440)
					d1 = snap372
					d2 = snap373
					d3 = snap374
					d4 = snap375
					d5 = snap376
					d6 = snap377
					d7 = snap378
					d8 = snap379
					d9 = snap380
					d24 = snap381
					d25 = snap382
					d26 = snap383
					d27 = snap384
					d28 = snap385
					d29 = snap386
					d30 = snap387
					d31 = snap388
					d32 = snap389
					d33 = snap390
					d34 = snap391
					d35 = snap392
					d36 = snap393
					d37 = snap394
					d38 = snap395
					d39 = snap396
					d40 = snap397
					d41 = snap398
					d42 = snap399
					d43 = snap400
					d44 = snap401
					d45 = snap402
					d46 = snap403
					d84 = snap404
					d85 = snap405
					d86 = snap406
					d87 = snap407
					d88 = snap408
					d90 = snap409
					d91 = snap410
					d92 = snap411
					d93 = snap412
					d94 = snap413
					d95 = snap414
					d96 = snap415
					d97 = snap416
					d98 = snap417
					d100 = snap418
					d101 = snap419
					d102 = snap420
					d103 = snap421
					d104 = snap422
					d161 = snap423
					d162 = snap424
					d163 = snap425
					d223 = snap426
					d224 = snap427
					d225 = snap428
					d226 = snap429
					d229 = snap430
					d292 = snap431
					d293 = snap432
					d294 = snap433
					d362 = snap434
					d363 = snap435
					d364 = snap436
					d365 = snap437
					d366 = snap438
					d367 = snap439
					if !bbs[12].Rendered {
						return bbs[12].RenderPS(ps370)
					}
					return result
					ctx.FreeDesc(&d366)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != LocNone {
						d365 = ps.OverlayValues[365]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d224)
					d442 = ctx.EmitSliceElementAddress(&d6, &d224, 16)
					ctx.EnsureDesc(&d442)
					r31 := ctx.AllocRegExcept(d442.Reg)
					ctx.EmitMovRegMem(r31, d442.Reg, 8)
					ctx.EmitMovRegMem(d442.Reg, d442.Reg, 0)
					d441 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d442.Reg, Reg2: r31}
					ctx.BindReg(d442.Reg, &d441)
					ctx.BindReg(r31, &d441)
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d441)
					d443 = ctx.EmitSliceElementAddress(&d97, &d224, int32(16))
					ctx.EmitStoreScmerAt(&d443, &d441)
					ctx.FreeDesc(&d443)
					ctx.FreeDesc(&d441)
					if ps.General {
					}
					ps444 := PhiState{General: ps.General}
					ps444.OverlayValues = make([]JITValueDesc, 444)
					ps444.OverlayValues[1] = d1
					ps444.OverlayValues[2] = d2
					ps444.OverlayValues[3] = d3
					ps444.OverlayValues[4] = d4
					ps444.OverlayValues[5] = d5
					ps444.OverlayValues[6] = d6
					ps444.OverlayValues[7] = d7
					ps444.OverlayValues[8] = d8
					ps444.OverlayValues[9] = d9
					ps444.OverlayValues[24] = d24
					ps444.OverlayValues[25] = d25
					ps444.OverlayValues[26] = d26
					ps444.OverlayValues[27] = d27
					ps444.OverlayValues[28] = d28
					ps444.OverlayValues[29] = d29
					ps444.OverlayValues[30] = d30
					ps444.OverlayValues[31] = d31
					ps444.OverlayValues[32] = d32
					ps444.OverlayValues[33] = d33
					ps444.OverlayValues[34] = d34
					ps444.OverlayValues[35] = d35
					ps444.OverlayValues[36] = d36
					ps444.OverlayValues[37] = d37
					ps444.OverlayValues[38] = d38
					ps444.OverlayValues[39] = d39
					ps444.OverlayValues[40] = d40
					ps444.OverlayValues[41] = d41
					ps444.OverlayValues[42] = d42
					ps444.OverlayValues[43] = d43
					ps444.OverlayValues[44] = d44
					ps444.OverlayValues[45] = d45
					ps444.OverlayValues[46] = d46
					ps444.OverlayValues[84] = d84
					ps444.OverlayValues[85] = d85
					ps444.OverlayValues[86] = d86
					ps444.OverlayValues[87] = d87
					ps444.OverlayValues[88] = d88
					ps444.OverlayValues[90] = d90
					ps444.OverlayValues[91] = d91
					ps444.OverlayValues[92] = d92
					ps444.OverlayValues[93] = d93
					ps444.OverlayValues[94] = d94
					ps444.OverlayValues[95] = d95
					ps444.OverlayValues[96] = d96
					ps444.OverlayValues[97] = d97
					ps444.OverlayValues[98] = d98
					ps444.OverlayValues[100] = d100
					ps444.OverlayValues[101] = d101
					ps444.OverlayValues[102] = d102
					ps444.OverlayValues[103] = d103
					ps444.OverlayValues[104] = d104
					ps444.OverlayValues[161] = d161
					ps444.OverlayValues[162] = d162
					ps444.OverlayValues[163] = d163
					ps444.OverlayValues[223] = d223
					ps444.OverlayValues[224] = d224
					ps444.OverlayValues[225] = d225
					ps444.OverlayValues[226] = d226
					ps444.OverlayValues[229] = d229
					ps444.OverlayValues[292] = d292
					ps444.OverlayValues[293] = d293
					ps444.OverlayValues[294] = d294
					ps444.OverlayValues[362] = d362
					ps444.OverlayValues[363] = d363
					ps444.OverlayValues[364] = d364
					ps444.OverlayValues[365] = d365
					ps444.OverlayValues[366] = d366
					ps444.OverlayValues[367] = d367
					ps444.OverlayValues[441] = d441
					ps444.OverlayValues[442] = d442
					ps444.OverlayValues[443] = d443
					ps444.PhiValues = make([]JITValueDesc, 1)
					if ps444.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps444)
					return result
				}
				bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[11].VisitCount >= 0 {
							ps.General = true
							return bbs[11].RenderPS(ps)
						}
					}
					bbs[11].VisitCount++
					if ps.General {
						if bbs[11].Rendered {
							ctx.EmitJmp(lbl12)
							return result
						}
						bbs[11].Rendered = true
						bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_11 = bbs[11].Address
						ctx.MarkLabel(lbl12)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != LocNone {
						d365 = ps.OverlayValues[365]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != LocNone {
						d443 = ps.OverlayValues[443]
					}
					ctx.ReclaimUntrackedRegs()
					d445 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d224)
					ctx.EnsureDesc(&d445)
					d446 = ctx.EmitSliceElementAddress(&d97, &d224, int32(16))
					ctx.EmitStoreScmerAt(&d446, &d445)
					ctx.FreeDesc(&d446)
					ctx.FreeDesc(&d445)
					if ps.General {
					}
					ps447 := PhiState{General: ps.General}
					ps447.OverlayValues = make([]JITValueDesc, 447)
					ps447.OverlayValues[1] = d1
					ps447.OverlayValues[2] = d2
					ps447.OverlayValues[3] = d3
					ps447.OverlayValues[4] = d4
					ps447.OverlayValues[5] = d5
					ps447.OverlayValues[6] = d6
					ps447.OverlayValues[7] = d7
					ps447.OverlayValues[8] = d8
					ps447.OverlayValues[9] = d9
					ps447.OverlayValues[24] = d24
					ps447.OverlayValues[25] = d25
					ps447.OverlayValues[26] = d26
					ps447.OverlayValues[27] = d27
					ps447.OverlayValues[28] = d28
					ps447.OverlayValues[29] = d29
					ps447.OverlayValues[30] = d30
					ps447.OverlayValues[31] = d31
					ps447.OverlayValues[32] = d32
					ps447.OverlayValues[33] = d33
					ps447.OverlayValues[34] = d34
					ps447.OverlayValues[35] = d35
					ps447.OverlayValues[36] = d36
					ps447.OverlayValues[37] = d37
					ps447.OverlayValues[38] = d38
					ps447.OverlayValues[39] = d39
					ps447.OverlayValues[40] = d40
					ps447.OverlayValues[41] = d41
					ps447.OverlayValues[42] = d42
					ps447.OverlayValues[43] = d43
					ps447.OverlayValues[44] = d44
					ps447.OverlayValues[45] = d45
					ps447.OverlayValues[46] = d46
					ps447.OverlayValues[84] = d84
					ps447.OverlayValues[85] = d85
					ps447.OverlayValues[86] = d86
					ps447.OverlayValues[87] = d87
					ps447.OverlayValues[88] = d88
					ps447.OverlayValues[90] = d90
					ps447.OverlayValues[91] = d91
					ps447.OverlayValues[92] = d92
					ps447.OverlayValues[93] = d93
					ps447.OverlayValues[94] = d94
					ps447.OverlayValues[95] = d95
					ps447.OverlayValues[96] = d96
					ps447.OverlayValues[97] = d97
					ps447.OverlayValues[98] = d98
					ps447.OverlayValues[100] = d100
					ps447.OverlayValues[101] = d101
					ps447.OverlayValues[102] = d102
					ps447.OverlayValues[103] = d103
					ps447.OverlayValues[104] = d104
					ps447.OverlayValues[161] = d161
					ps447.OverlayValues[162] = d162
					ps447.OverlayValues[163] = d163
					ps447.OverlayValues[223] = d223
					ps447.OverlayValues[224] = d224
					ps447.OverlayValues[225] = d225
					ps447.OverlayValues[226] = d226
					ps447.OverlayValues[229] = d229
					ps447.OverlayValues[292] = d292
					ps447.OverlayValues[293] = d293
					ps447.OverlayValues[294] = d294
					ps447.OverlayValues[362] = d362
					ps447.OverlayValues[363] = d363
					ps447.OverlayValues[364] = d364
					ps447.OverlayValues[365] = d365
					ps447.OverlayValues[366] = d366
					ps447.OverlayValues[367] = d367
					ps447.OverlayValues[441] = d441
					ps447.OverlayValues[442] = d442
					ps447.OverlayValues[443] = d443
					ps447.OverlayValues[445] = d445
					ps447.OverlayValues[446] = d446
					ps447.PhiValues = make([]JITValueDesc, 1)
					if ps447.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps447)
					return result
				}
				bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[12].VisitCount >= 0 {
							ps.General = true
							return bbs[12].RenderPS(ps)
						}
					}
					bbs[12].VisitCount++
					if ps.General {
						if bbs[12].Rendered {
							ctx.EmitJmp(lbl13)
							return result
						}
						bbs[12].Rendered = true
						bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_12 = bbs[12].Address
						ctx.MarkLabel(lbl13)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != LocNone {
						d365 = ps.OverlayValues[365]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != LocNone {
						d443 = ps.OverlayValues[443]
					}
					if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != LocNone {
						d445 = ps.OverlayValues[445]
					}
					if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != LocNone {
						d446 = ps.OverlayValues[446]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d27)
					ctx.EnsureDesc(&d27)
					var d448 JITValueDesc
					if d27.Loc == LocImm {
						d448 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d27.Imm.Int() - 1)}
					} else {
						scratch := ctx.AllocRegExcept(d27.Reg)
						ctx.EmitMovRegReg(scratch, d27.Reg)
						ctx.EmitSubRegImm32(scratch, int32(1))
						d448 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d448)
					}
					if d448.Loc == LocReg && d27.Loc == LocReg && d448.Reg == d27.Reg {
						ctx.TransferReg(d27.Reg)
						d27.Loc = LocNone
					}
					ctx.FreeDesc(&d27)
					ctx.EnsureDesc(&d448)
					ctx.EnsureDesc(&d448)
					ctx.EnsureDesc(&d448)
					d450 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ctx.EnsureDesc(&d448)
					d451 = ctx.EmitSliceElementAddress(&d3, &d450, int32(16))
					ctx.EmitStoreScmerAt(&d451, &d448)
					ctx.FreeDesc(&d451)
					d452 = args[0]
					d452.ID = 0
					ctx.EnsureDesc(&d452)
					if d452.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d452, &result)
						result.Type = d452.Type
					} else {
						switch d452.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d452)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d452)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d452)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d452, &result)
							result.Type = d452.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[13].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[13].VisitCount >= 0 {
							ps.General = true
							return bbs[13].RenderPS(ps)
						}
					}
					bbs[13].VisitCount++
					if ps.General {
						if bbs[13].Rendered {
							ctx.EmitJmp(lbl14)
							return result
						}
						bbs[13].Rendered = true
						bbs[13].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_13 = bbs[13].Address
						ctx.MarkLabel(lbl14)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 24 && ps.OverlayValues[24].Loc != LocNone {
						d24 = ps.OverlayValues[24]
					}
					if len(ps.OverlayValues) > 25 && ps.OverlayValues[25].Loc != LocNone {
						d25 = ps.OverlayValues[25]
					}
					if len(ps.OverlayValues) > 26 && ps.OverlayValues[26].Loc != LocNone {
						d26 = ps.OverlayValues[26]
					}
					if len(ps.OverlayValues) > 27 && ps.OverlayValues[27].Loc != LocNone {
						d27 = ps.OverlayValues[27]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 41 && ps.OverlayValues[41].Loc != LocNone {
						d41 = ps.OverlayValues[41]
					}
					if len(ps.OverlayValues) > 42 && ps.OverlayValues[42].Loc != LocNone {
						d42 = ps.OverlayValues[42]
					}
					if len(ps.OverlayValues) > 43 && ps.OverlayValues[43].Loc != LocNone {
						d43 = ps.OverlayValues[43]
					}
					if len(ps.OverlayValues) > 44 && ps.OverlayValues[44].Loc != LocNone {
						d44 = ps.OverlayValues[44]
					}
					if len(ps.OverlayValues) > 45 && ps.OverlayValues[45].Loc != LocNone {
						d45 = ps.OverlayValues[45]
					}
					if len(ps.OverlayValues) > 46 && ps.OverlayValues[46].Loc != LocNone {
						d46 = ps.OverlayValues[46]
					}
					if len(ps.OverlayValues) > 84 && ps.OverlayValues[84].Loc != LocNone {
						d84 = ps.OverlayValues[84]
					}
					if len(ps.OverlayValues) > 85 && ps.OverlayValues[85].Loc != LocNone {
						d85 = ps.OverlayValues[85]
					}
					if len(ps.OverlayValues) > 86 && ps.OverlayValues[86].Loc != LocNone {
						d86 = ps.OverlayValues[86]
					}
					if len(ps.OverlayValues) > 87 && ps.OverlayValues[87].Loc != LocNone {
						d87 = ps.OverlayValues[87]
					}
					if len(ps.OverlayValues) > 88 && ps.OverlayValues[88].Loc != LocNone {
						d88 = ps.OverlayValues[88]
					}
					if len(ps.OverlayValues) > 90 && ps.OverlayValues[90].Loc != LocNone {
						d90 = ps.OverlayValues[90]
					}
					if len(ps.OverlayValues) > 91 && ps.OverlayValues[91].Loc != LocNone {
						d91 = ps.OverlayValues[91]
					}
					if len(ps.OverlayValues) > 92 && ps.OverlayValues[92].Loc != LocNone {
						d92 = ps.OverlayValues[92]
					}
					if len(ps.OverlayValues) > 93 && ps.OverlayValues[93].Loc != LocNone {
						d93 = ps.OverlayValues[93]
					}
					if len(ps.OverlayValues) > 94 && ps.OverlayValues[94].Loc != LocNone {
						d94 = ps.OverlayValues[94]
					}
					if len(ps.OverlayValues) > 95 && ps.OverlayValues[95].Loc != LocNone {
						d95 = ps.OverlayValues[95]
					}
					if len(ps.OverlayValues) > 96 && ps.OverlayValues[96].Loc != LocNone {
						d96 = ps.OverlayValues[96]
					}
					if len(ps.OverlayValues) > 97 && ps.OverlayValues[97].Loc != LocNone {
						d97 = ps.OverlayValues[97]
					}
					if len(ps.OverlayValues) > 98 && ps.OverlayValues[98].Loc != LocNone {
						d98 = ps.OverlayValues[98]
					}
					if len(ps.OverlayValues) > 100 && ps.OverlayValues[100].Loc != LocNone {
						d100 = ps.OverlayValues[100]
					}
					if len(ps.OverlayValues) > 101 && ps.OverlayValues[101].Loc != LocNone {
						d101 = ps.OverlayValues[101]
					}
					if len(ps.OverlayValues) > 102 && ps.OverlayValues[102].Loc != LocNone {
						d102 = ps.OverlayValues[102]
					}
					if len(ps.OverlayValues) > 103 && ps.OverlayValues[103].Loc != LocNone {
						d103 = ps.OverlayValues[103]
					}
					if len(ps.OverlayValues) > 104 && ps.OverlayValues[104].Loc != LocNone {
						d104 = ps.OverlayValues[104]
					}
					if len(ps.OverlayValues) > 161 && ps.OverlayValues[161].Loc != LocNone {
						d161 = ps.OverlayValues[161]
					}
					if len(ps.OverlayValues) > 162 && ps.OverlayValues[162].Loc != LocNone {
						d162 = ps.OverlayValues[162]
					}
					if len(ps.OverlayValues) > 163 && ps.OverlayValues[163].Loc != LocNone {
						d163 = ps.OverlayValues[163]
					}
					if len(ps.OverlayValues) > 223 && ps.OverlayValues[223].Loc != LocNone {
						d223 = ps.OverlayValues[223]
					}
					if len(ps.OverlayValues) > 224 && ps.OverlayValues[224].Loc != LocNone {
						d224 = ps.OverlayValues[224]
					}
					if len(ps.OverlayValues) > 225 && ps.OverlayValues[225].Loc != LocNone {
						d225 = ps.OverlayValues[225]
					}
					if len(ps.OverlayValues) > 226 && ps.OverlayValues[226].Loc != LocNone {
						d226 = ps.OverlayValues[226]
					}
					if len(ps.OverlayValues) > 229 && ps.OverlayValues[229].Loc != LocNone {
						d229 = ps.OverlayValues[229]
					}
					if len(ps.OverlayValues) > 292 && ps.OverlayValues[292].Loc != LocNone {
						d292 = ps.OverlayValues[292]
					}
					if len(ps.OverlayValues) > 293 && ps.OverlayValues[293].Loc != LocNone {
						d293 = ps.OverlayValues[293]
					}
					if len(ps.OverlayValues) > 294 && ps.OverlayValues[294].Loc != LocNone {
						d294 = ps.OverlayValues[294]
					}
					if len(ps.OverlayValues) > 362 && ps.OverlayValues[362].Loc != LocNone {
						d362 = ps.OverlayValues[362]
					}
					if len(ps.OverlayValues) > 363 && ps.OverlayValues[363].Loc != LocNone {
						d363 = ps.OverlayValues[363]
					}
					if len(ps.OverlayValues) > 364 && ps.OverlayValues[364].Loc != LocNone {
						d364 = ps.OverlayValues[364]
					}
					if len(ps.OverlayValues) > 365 && ps.OverlayValues[365].Loc != LocNone {
						d365 = ps.OverlayValues[365]
					}
					if len(ps.OverlayValues) > 366 && ps.OverlayValues[366].Loc != LocNone {
						d366 = ps.OverlayValues[366]
					}
					if len(ps.OverlayValues) > 367 && ps.OverlayValues[367].Loc != LocNone {
						d367 = ps.OverlayValues[367]
					}
					if len(ps.OverlayValues) > 441 && ps.OverlayValues[441].Loc != LocNone {
						d441 = ps.OverlayValues[441]
					}
					if len(ps.OverlayValues) > 442 && ps.OverlayValues[442].Loc != LocNone {
						d442 = ps.OverlayValues[442]
					}
					if len(ps.OverlayValues) > 443 && ps.OverlayValues[443].Loc != LocNone {
						d443 = ps.OverlayValues[443]
					}
					if len(ps.OverlayValues) > 445 && ps.OverlayValues[445].Loc != LocNone {
						d445 = ps.OverlayValues[445]
					}
					if len(ps.OverlayValues) > 446 && ps.OverlayValues[446].Loc != LocNone {
						d446 = ps.OverlayValues[446]
					}
					if len(ps.OverlayValues) > 448 && ps.OverlayValues[448].Loc != LocNone {
						d448 = ps.OverlayValues[448]
					}
					if len(ps.OverlayValues) > 449 && ps.OverlayValues[449].Loc != LocNone {
						d449 = ps.OverlayValues[449]
					}
					if len(ps.OverlayValues) > 450 && ps.OverlayValues[450].Loc != LocNone {
						d450 = ps.OverlayValues[450]
					}
					if len(ps.OverlayValues) > 451 && ps.OverlayValues[451].Loc != LocNone {
						d451 = ps.OverlayValues[451]
					}
					if len(ps.OverlayValues) > 452 && ps.OverlayValues[452].Loc != LocNone {
						d452 = ps.OverlayValues[452]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d44)
					d453 = d4
					_ = d453
					ctx.StabilizeDescForControlFlow(&d453)
					d454 = d44
					_ = d454
					ctx.StabilizeDescForControlFlow(&d454)
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d453)
					ctx.EnsureDesc(&d453)
					ctx.EnsureDesc(&d453)
					if d453.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d453.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d453.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d453)
						} else if d453.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d453)
						} else if d453.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d453)
						} else if d453.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d453.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d453 = tmpPair
					} else if d453.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d453.Type, Reg: ctx.AllocRegExcept(d453.Reg), Reg2: ctx.AllocRegExcept(d453.Reg)}
						switch d453.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d453)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d453)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d453)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d453)
						d453 = tmpPair
					}
					if d453.Loc != LocRegPair && d453.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (ApplyEx arg0)")
					}
					ctx.EnsureDesc(&d454)
					ctx.EnsureDesc(&d454)
					ctx.EnsureDesc(&d454)
					if d454.Loc != LocRegTriple && d454.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d455 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d455.Loc == LocRegPair || d455.Loc == LocStackPair || d455.Loc == LocRegTriple || d455.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d453)
					ctx.SyncDesc(&d454)
					ctx.SyncDesc(&d455)
					d456 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d453, d454, d455}, 2)
					ctx.BindReg(d456.Reg, &d456)
					ctx.BindReg(d456.Reg2, &d456)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d456)
					ctx.FreeDesc(&d4)
					d457 = args[0]
					d457.ID = 0
					ctx.EnsureDesc(&d457)
					if d457.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d457, &result)
						result.Type = d457.Type
					} else {
						switch d457.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d457)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d457)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d457)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d457, &result)
							result.Type = d457.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps458 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps458)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(16))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  81,
		},
	})

	Declare(&Globalenv, &Declaration{
		Name: "window_flush",

		Fn: windowFlush,
		Type: &TypeDescriptor{Kind: "func", Description: "Flush a caller-owned window by shifting in nils without allocating and invoking emit_fn for each displaced position.", HasSideEffects: true,
			Params: []*TypeDescriptor{&TypeDescriptor{Kind: "list", Label: "window", Description: "ring buffer accumulator"}, &TypeDescriptor{Kind: "func", Label: "emit_fn", Description: "callback receiving all window values oldest-to-newest", Params: []*TypeDescriptor{{Kind: "any", Label: "values", Variadic: true}}, Return: &TypeDescriptor{Kind: "any"}}, &TypeDescriptor{Kind: "number", Label: "count", Description: "number of nil positions to shift in"}},
			Return: &TypeDescriptor{Kind: "nil"},

			JITEmit: func(ctx *JITContext, sourceArgs []Scmer, args []JITValueDesc, result JITValueDesc) JITValueDesc {
				if !jitEnabled {
					return jitEmitGoVariadicCallFromDescs(ctx, declarations["window_flush"].Fn, args, result)
				}
				var d3 JITValueDesc
				_ = d3
				var d4 JITValueDesc
				_ = d4
				var d5 JITValueDesc
				_ = d5
				var d6 JITValueDesc
				_ = d6
				var d7 JITValueDesc
				_ = d7
				var d8 JITValueDesc
				_ = d8
				var d9 JITValueDesc
				_ = d9
				var d10 JITValueDesc
				_ = d10
				var d11 JITValueDesc
				_ = d11
				var d28 JITValueDesc
				_ = d28
				var d29 JITValueDesc
				_ = d29
				var d30 JITValueDesc
				_ = d30
				var d31 JITValueDesc
				_ = d31
				var d32 JITValueDesc
				_ = d32
				var d33 JITValueDesc
				_ = d33
				var d34 JITValueDesc
				_ = d34
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
				var d71 JITValueDesc
				_ = d71
				var d72 JITValueDesc
				_ = d72
				var d73 JITValueDesc
				_ = d73
				var d74 JITValueDesc
				_ = d74
				var d75 JITValueDesc
				_ = d75
				var d110 JITValueDesc
				_ = d110
				var d111 JITValueDesc
				_ = d111
				var d112 JITValueDesc
				_ = d112
				var d150 JITValueDesc
				_ = d150
				var d151 JITValueDesc
				_ = d151
				var d152 JITValueDesc
				_ = d152
				var d155 JITValueDesc
				_ = d155
				var d195 JITValueDesc
				_ = d195
				var d196 JITValueDesc
				_ = d196
				var d197 JITValueDesc
				_ = d197
				var d198 JITValueDesc
				_ = d198
				var d199 JITValueDesc
				_ = d199
				var d201 JITValueDesc
				_ = d201
				var d202 JITValueDesc
				_ = d202
				var d203 JITValueDesc
				_ = d203
				var d205 JITValueDesc
				_ = d205
				var d206 JITValueDesc
				_ = d206
				var d207 JITValueDesc
				_ = d207
				var d208 JITValueDesc
				_ = d208
				var d209 JITValueDesc
				_ = d209
				var d212 JITValueDesc
				_ = d212
				var d266 JITValueDesc
				_ = d266
				var d267 JITValueDesc
				_ = d267
				var d268 JITValueDesc
				_ = d268
				var d270 JITValueDesc
				_ = d270
				var d271 JITValueDesc
				_ = d271
				var d272 JITValueDesc
				_ = d272
				var d273 JITValueDesc
				_ = d273
				var d274 JITValueDesc
				_ = d274
				var d275 JITValueDesc
				_ = d275
				var d276 JITValueDesc
				_ = d276
				var d277 JITValueDesc
				_ = d277
				var d278 JITValueDesc
				_ = d278
				var d279 JITValueDesc
				_ = d279
				var d280 JITValueDesc
				_ = d280
				var d281 JITValueDesc
				_ = d281
				/* DO NEVER MANUALLY EDIT THIS SECTION. RUN make jitgen TO UPDATE */
				phiBase0 := ctx.AllocStack(int32(32))
				d1 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
				_ = d1
				d2 := JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
				_ = d2
				var bbs [13]BBDescriptor
				bbs[7].PhiBase = int32(phiBase0) + int32(0)
				bbs[7].PhiCount = uint16(1)
				bbs[10].PhiBase = int32(phiBase0) + int32(16)
				bbs[10].PhiCount = uint16(1)
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
				bbpos_0_5 := int32(-1)
				_ = bbpos_0_5
				lbl6 := ctx.ReserveLabel()
				bbpos_0_6 := int32(-1)
				_ = bbpos_0_6
				lbl7 := ctx.ReserveLabel()
				bbpos_0_7 := int32(-1)
				_ = bbpos_0_7
				lbl8 := ctx.ReserveLabel()
				bbpos_0_8 := int32(-1)
				_ = bbpos_0_8
				lbl9 := ctx.ReserveLabel()
				bbpos_0_9 := int32(-1)
				_ = bbpos_0_9
				lbl10 := ctx.ReserveLabel()
				bbpos_0_10 := int32(-1)
				_ = bbpos_0_10
				lbl11 := ctx.ReserveLabel()
				bbpos_0_11 := int32(-1)
				_ = bbpos_0_11
				lbl12 := ctx.ReserveLabel()
				bbpos_0_12 := int32(-1)
				_ = bbpos_0_12
				lbl13 := ctx.ReserveLabel()
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
					}
					ctx.ReclaimUntrackedRegs()
					d3 = args[0]
					d3.ID = 0
					var d4 JITValueDesc
					if d3.Type == tagSlice {
						d4 = jitKnownSliceHeader(ctx, &d3)
					} else {
						d4 = ctx.EmitGoCallScalar(GoFuncAddr(jitAsSlice), []JITValueDesc{d3}, 3)
					}
					ctx.BindReg(d4.Reg, &d4)
					ctx.BindReg(d4.Reg2, &d4)
					ctx.BindReg(d4.Reg3, &d4)
					ctx.StabilizeDescForControlFlow(&d4)
					ctx.FreeDesc(&d3)
					d5 = args[1]
					d5.ID = 0
					ctx.StabilizeDescForControlFlow(&d5)
					d6 = args[2]
					d6.ID = 0
					var d7 JITValueDesc
					if d6.Loc == LocImm {
						d7 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d6.Imm.Int())}
					} else if d6.Type == tagInt && d6.Loc == LocRegPair {
						ctx.FreeReg(d6.Reg)
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg2}
						ctx.BindReg(d6.Reg2, &d7)
						ctx.BindReg(d6.Reg2, &d7)
					} else if d6.Type == tagInt && d6.Loc == LocReg {
						d7 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d6.Reg}
						ctx.BindReg(d6.Reg, &d7)
						ctx.BindReg(d6.Reg, &d7)
					} else {
						d7 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d6}, 1)
						d7.Type = tagInt
						ctx.BindReg(d7.Reg, &d7)
					}
					ctx.FreeDesc(&d6)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d7)
					ctx.StabilizeDescForControlFlow(&d7)
					var d9 JITValueDesc
					if d4.SliceSizeKnown {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.KnownSliceLen))}
					} else if d4.Loc == LocImm {
						d9 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d4.StackOff))}
					} else if d4.Loc == LocStackTriple {
						d9 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d4.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d4)
						if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2, ID: 0}
						} else if d4.Loc == LocReg {
							d9 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d9)
					var d10 JITValueDesc
					if d9.Loc == LocImm {
						d10 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d9.Imm.Int() < 3)}
					} else {
						r0 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d9.Reg, 3)
						ctx.EmitSetcc(r0, CondSignedLess)
						d10 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r0}
						ctx.BindReg(r0, &d10)
					}
					ctx.FreeDesc(&d9)
					d11 = d10
					ctx.EnsureDesc(&d11)
					if d11.Loc != LocImm && d11.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d11.Loc == LocImm {
						if d11.Imm.Bool() {
							if ps.General {
							}
							ps12 := PhiState{General: ps.General}
							ps12.OverlayValues = make([]JITValueDesc, 12)
							ps12.OverlayValues[1] = d1
							ps12.OverlayValues[2] = d2
							ps12.OverlayValues[3] = d3
							ps12.OverlayValues[4] = d4
							ps12.OverlayValues[5] = d5
							ps12.OverlayValues[6] = d6
							ps12.OverlayValues[7] = d7
							ps12.OverlayValues[8] = d8
							ps12.OverlayValues[9] = d9
							ps12.OverlayValues[10] = d10
							ps12.OverlayValues[11] = d11
							return bbs[1].RenderPS(ps12)
						}
						if ps.General {
						}
						ps13 := PhiState{General: ps.General}
						ps13.OverlayValues = make([]JITValueDesc, 12)
						ps13.OverlayValues[1] = d1
						ps13.OverlayValues[2] = d2
						ps13.OverlayValues[3] = d3
						ps13.OverlayValues[4] = d4
						ps13.OverlayValues[5] = d5
						ps13.OverlayValues[6] = d6
						ps13.OverlayValues[7] = d7
						ps13.OverlayValues[8] = d8
						ps13.OverlayValues[9] = d9
						ps13.OverlayValues[10] = d10
						ps13.OverlayValues[11] = d11
						return bbs[2].RenderPS(ps13)
					}
					if !ps.General {
						ps.General = true
						return bbs[0].RenderPS(ps)
					}
					lbl14 := ctx.ReserveLabel()
					lbl15 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d11.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl14)
					ctx.EmitJmp(lbl15)
					ctx.MarkLabel(lbl14)
					ctx.EmitJmp(lbl2)
					ctx.MarkLabel(lbl15)
					ctx.EmitJmp(lbl3)
					ps14 := PhiState{General: true}
					ps14.OverlayValues = make([]JITValueDesc, 12)
					ps14.OverlayValues[1] = d1
					ps14.OverlayValues[2] = d2
					ps14.OverlayValues[3] = d3
					ps14.OverlayValues[4] = d4
					ps14.OverlayValues[5] = d5
					ps14.OverlayValues[6] = d6
					ps14.OverlayValues[7] = d7
					ps14.OverlayValues[8] = d8
					ps14.OverlayValues[9] = d9
					ps14.OverlayValues[10] = d10
					ps14.OverlayValues[11] = d11
					ps15 := PhiState{General: true}
					ps15.OverlayValues = make([]JITValueDesc, 12)
					ps15.OverlayValues[1] = d1
					ps15.OverlayValues[2] = d2
					ps15.OverlayValues[3] = d3
					ps15.OverlayValues[4] = d4
					ps15.OverlayValues[5] = d5
					ps15.OverlayValues[6] = d6
					ps15.OverlayValues[7] = d7
					ps15.OverlayValues[8] = d8
					ps15.OverlayValues[9] = d9
					ps15.OverlayValues[10] = d10
					ps15.OverlayValues[11] = d11
					snap16 := d1
					snap17 := d2
					snap18 := d3
					snap19 := d4
					snap20 := d5
					snap21 := d6
					snap22 := d7
					snap23 := d8
					snap24 := d9
					snap25 := d10
					snap26 := d11
					alloc27 := ctx.SnapshotAllocState()
					if !bbs[2].Rendered {
						bbs[2].RenderPS(ps15)
					}
					ctx.RestoreAllocState(alloc27)
					d1 = snap16
					d2 = snap17
					d3 = snap18
					d4 = snap19
					d5 = snap20
					d6 = snap21
					d7 = snap22
					d8 = snap23
					d9 = snap24
					d10 = snap25
					d11 = snap26
					if !bbs[1].Rendered {
						return bbs[1].RenderPS(ps14)
					}
					return result
					ctx.FreeDesc(&d10)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_flush"].Fn, args, result)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					ctx.ReclaimUntrackedRegs()
					d28 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(2)}
					d30 = ctx.EmitSliceElementAddress(&d4, &d28, 16)
					ctx.EnsureDesc(&d30)
					r1 := ctx.AllocRegExcept(d30.Reg)
					ctx.EmitMovRegMem(r1, d30.Reg, 8)
					ctx.EmitMovRegMem(d30.Reg, d30.Reg, 0)
					d29 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d30.Reg, Reg2: r1}
					ctx.BindReg(d30.Reg, &d29)
					ctx.BindReg(r1, &d29)
					var d31 JITValueDesc
					if d29.Loc == LocImm {
						d31 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d29.Imm.Int())}
					} else if d29.Type == tagInt && d29.Loc == LocRegPair {
						ctx.FreeReg(d29.Reg)
						d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg2}
						ctx.BindReg(d29.Reg2, &d31)
						ctx.BindReg(d29.Reg2, &d31)
					} else if d29.Type == tagInt && d29.Loc == LocReg {
						d31 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d29.Reg}
						ctx.BindReg(d29.Reg, &d31)
						ctx.BindReg(d29.Reg, &d31)
					} else {
						d31 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d29}, 1)
						d31.Type = tagInt
						ctx.BindReg(d31.Reg, &d31)
					}
					ctx.FreeDesc(&d29)
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d31)
					ctx.StabilizeDescForControlFlow(&d31)
					d33 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(3)}
					var d34 JITValueDesc
					ctx.EnsureDesc(&d4)
					if d4.Loc == LocRegPair || d4.Loc == LocRegTriple {
						d34 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d4.Reg2}
						ctx.BindReg(d4.Reg2, &d34)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d4)
					ctx.EnsureDesc(&d33)
					ctx.EnsureDesc(&d34)
					var d36 JITValueDesc
					if d34.Loc == LocImm && d33.Loc == LocImm {
						d36 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d34.Imm.Int() - d33.Imm.Int())}
					} else {
						r2 := ctx.AllocReg()
						if d34.Loc == LocImm {
							ctx.EmitMovRegImm64(r2, uint64(d34.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r2, d34.Reg)
						}
						if d33.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()))
							ctx.EmitSubInt64(r2, RegR11)
						} else {
							ctx.EmitSubInt64(r2, d33.Reg)
						}
						d36 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r2}
						ctx.BindReg(r2, &d36)
					}
					var d37 JITValueDesc
					if d4.Loc == LocImm && d33.Loc == LocImm {
						d37 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d4.Imm.Int() + d33.Imm.Int()*16)}
					} else {
						r3 := ctx.AllocReg()
						if d4.Loc == LocImm {
							ctx.EmitMovRegImm64(r3, uint64(d4.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r3, d4.Reg)
						}
						if d33.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d33.Imm.Int()*16))
							ctx.EmitAddInt64(r3, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r3, d33.Reg)
							ctx.EmitMovRegReg(offsetReg, d33.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r3, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d37 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r3}
						ctx.BindReg(r3, &d37)
					}
					var d38 JITValueDesc
					var r4 Reg
					var r5 Reg
					ctx.SyncDesc(&d37)
					ctx.EnsureDesc(&d37)
					if d37.Loc == LocImm {
						r4 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r4, uint64(d37.Imm.Int()))
					} else {
						r4 = d37.Reg
					}
					ctx.ProtectReg(r4)
					ctx.SyncDesc(&d36)
					ctx.EnsureDesc(&d36)
					if d36.Loc == LocImm {
						r5 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r5, uint64(d36.Imm.Int()))
					} else {
						r5 = d36.Reg
					}
					ctx.ProtectReg(r5)
					r6 := ctx.EmitSliceCapAfterLow(&d4, &d33, r4, r5)
					ctx.UnprotectReg(r5)
					ctx.UnprotectReg(r4)
					d38 = JITValueDesc{Loc: LocRegTriple, Reg: r4, Reg2: r5, Reg3: r6}
					ctx.BindReg(r4, &d38)
					ctx.BindReg(r5, &d38)
					ctx.BindReg(r6, &d38)
					ctx.BindReg(r4, &d38)
					ctx.BindReg(r5, &d38)
					ctx.BindReg(r6, &d38)
					ctx.StabilizeDescForControlFlow(&d38)
					ctx.EnsureDesc(&d31)
					var d39 JITValueDesc
					if d31.Loc == LocImm {
						d39 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d31.Imm.Int() <= 0)}
					} else {
						r7 := ctx.AllocRegExcept(d31.Reg)
						ctx.EmitCmpRegImm32(d31.Reg, 0)
						ctx.EmitSetcc(r7, CondSignedLessOrEqual)
						d39 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r7}
						ctx.BindReg(r7, &d39)
					}
					d40 = d39
					ctx.EnsureDesc(&d40)
					if d40.Loc != LocImm && d40.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d40.Loc == LocImm {
						if d40.Imm.Bool() {
							if ps.General {
							}
							ps41 := PhiState{General: ps.General}
							ps41.OverlayValues = make([]JITValueDesc, 41)
							ps41.OverlayValues[1] = d1
							ps41.OverlayValues[2] = d2
							ps41.OverlayValues[3] = d3
							ps41.OverlayValues[4] = d4
							ps41.OverlayValues[5] = d5
							ps41.OverlayValues[6] = d6
							ps41.OverlayValues[7] = d7
							ps41.OverlayValues[8] = d8
							ps41.OverlayValues[9] = d9
							ps41.OverlayValues[10] = d10
							ps41.OverlayValues[11] = d11
							ps41.OverlayValues[28] = d28
							ps41.OverlayValues[29] = d29
							ps41.OverlayValues[30] = d30
							ps41.OverlayValues[31] = d31
							ps41.OverlayValues[32] = d32
							ps41.OverlayValues[33] = d33
							ps41.OverlayValues[34] = d34
							ps41.OverlayValues[35] = d35
							ps41.OverlayValues[36] = d36
							ps41.OverlayValues[37] = d37
							ps41.OverlayValues[38] = d38
							ps41.OverlayValues[39] = d39
							ps41.OverlayValues[40] = d40
							return bbs[3].RenderPS(ps41)
						}
						if ps.General {
						}
						ps42 := PhiState{General: ps.General}
						ps42.OverlayValues = make([]JITValueDesc, 41)
						ps42.OverlayValues[1] = d1
						ps42.OverlayValues[2] = d2
						ps42.OverlayValues[3] = d3
						ps42.OverlayValues[4] = d4
						ps42.OverlayValues[5] = d5
						ps42.OverlayValues[6] = d6
						ps42.OverlayValues[7] = d7
						ps42.OverlayValues[8] = d8
						ps42.OverlayValues[9] = d9
						ps42.OverlayValues[10] = d10
						ps42.OverlayValues[11] = d11
						ps42.OverlayValues[28] = d28
						ps42.OverlayValues[29] = d29
						ps42.OverlayValues[30] = d30
						ps42.OverlayValues[31] = d31
						ps42.OverlayValues[32] = d32
						ps42.OverlayValues[33] = d33
						ps42.OverlayValues[34] = d34
						ps42.OverlayValues[35] = d35
						ps42.OverlayValues[36] = d36
						ps42.OverlayValues[37] = d37
						ps42.OverlayValues[38] = d38
						ps42.OverlayValues[39] = d39
						ps42.OverlayValues[40] = d40
						return bbs[6].RenderPS(ps42)
					}
					if !ps.General {
						ps.General = true
						return bbs[2].RenderPS(ps)
					}
					lbl16 := ctx.ReserveLabel()
					lbl17 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d40.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl16)
					ctx.EmitJmp(lbl17)
					ctx.MarkLabel(lbl16)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl17)
					ctx.EmitJmp(lbl7)
					ps43 := PhiState{General: true}
					ps43.OverlayValues = make([]JITValueDesc, 41)
					ps43.OverlayValues[1] = d1
					ps43.OverlayValues[2] = d2
					ps43.OverlayValues[3] = d3
					ps43.OverlayValues[4] = d4
					ps43.OverlayValues[5] = d5
					ps43.OverlayValues[6] = d6
					ps43.OverlayValues[7] = d7
					ps43.OverlayValues[8] = d8
					ps43.OverlayValues[9] = d9
					ps43.OverlayValues[10] = d10
					ps43.OverlayValues[11] = d11
					ps43.OverlayValues[28] = d28
					ps43.OverlayValues[29] = d29
					ps43.OverlayValues[30] = d30
					ps43.OverlayValues[31] = d31
					ps43.OverlayValues[32] = d32
					ps43.OverlayValues[33] = d33
					ps43.OverlayValues[34] = d34
					ps43.OverlayValues[35] = d35
					ps43.OverlayValues[36] = d36
					ps43.OverlayValues[37] = d37
					ps43.OverlayValues[38] = d38
					ps43.OverlayValues[39] = d39
					ps43.OverlayValues[40] = d40
					ps44 := PhiState{General: true}
					ps44.OverlayValues = make([]JITValueDesc, 41)
					ps44.OverlayValues[1] = d1
					ps44.OverlayValues[2] = d2
					ps44.OverlayValues[3] = d3
					ps44.OverlayValues[4] = d4
					ps44.OverlayValues[5] = d5
					ps44.OverlayValues[6] = d6
					ps44.OverlayValues[7] = d7
					ps44.OverlayValues[8] = d8
					ps44.OverlayValues[9] = d9
					ps44.OverlayValues[10] = d10
					ps44.OverlayValues[11] = d11
					ps44.OverlayValues[28] = d28
					ps44.OverlayValues[29] = d29
					ps44.OverlayValues[30] = d30
					ps44.OverlayValues[31] = d31
					ps44.OverlayValues[32] = d32
					ps44.OverlayValues[33] = d33
					ps44.OverlayValues[34] = d34
					ps44.OverlayValues[35] = d35
					ps44.OverlayValues[36] = d36
					ps44.OverlayValues[37] = d37
					ps44.OverlayValues[38] = d38
					ps44.OverlayValues[39] = d39
					ps44.OverlayValues[40] = d40
					snap45 := d1
					snap46 := d2
					snap47 := d3
					snap48 := d4
					snap49 := d5
					snap50 := d6
					snap51 := d7
					snap52 := d8
					snap53 := d9
					snap54 := d10
					snap55 := d11
					snap56 := d28
					snap57 := d29
					snap58 := d30
					snap59 := d31
					snap60 := d32
					snap61 := d33
					snap62 := d34
					snap63 := d35
					snap64 := d36
					snap65 := d37
					snap66 := d38
					snap67 := d39
					snap68 := d40
					alloc69 := ctx.SnapshotAllocState()
					if !bbs[6].Rendered {
						bbs[6].RenderPS(ps44)
					}
					ctx.RestoreAllocState(alloc69)
					d1 = snap45
					d2 = snap46
					d3 = snap47
					d4 = snap48
					d5 = snap49
					d6 = snap50
					d7 = snap51
					d8 = snap52
					d9 = snap53
					d10 = snap54
					d11 = snap55
					d28 = snap56
					d29 = snap57
					d30 = snap58
					d31 = snap59
					d32 = snap60
					d33 = snap61
					d34 = snap62
					d35 = snap63
					d36 = snap64
					d37 = snap65
					d38 = snap66
					d39 = snap67
					d40 = snap68
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps43)
					}
					return result
					ctx.FreeDesc(&d39)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					ctx.ReclaimUntrackedRegs()
					_ = jitEmitGoVariadicCallFromDescs(ctx, declarations["window_flush"].Fn, args, result)
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					ctx.ReclaimUntrackedRegs()
					if ps.General {
						ctx.EmitStoreToStack(JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}, int32(bbs[7].PhiBase)+int32(0))
					}
					ps70 := PhiState{General: ps.General}
					ps70.OverlayValues = make([]JITValueDesc, 41)
					ps70.OverlayValues[1] = d1
					ps70.OverlayValues[2] = d2
					ps70.OverlayValues[3] = d3
					ps70.OverlayValues[4] = d4
					ps70.OverlayValues[5] = d5
					ps70.OverlayValues[6] = d6
					ps70.OverlayValues[7] = d7
					ps70.OverlayValues[8] = d8
					ps70.OverlayValues[9] = d9
					ps70.OverlayValues[10] = d10
					ps70.OverlayValues[11] = d11
					ps70.OverlayValues[28] = d28
					ps70.OverlayValues[29] = d29
					ps70.OverlayValues[30] = d30
					ps70.OverlayValues[31] = d31
					ps70.OverlayValues[32] = d32
					ps70.OverlayValues[33] = d33
					ps70.OverlayValues[34] = d34
					ps70.OverlayValues[35] = d35
					ps70.OverlayValues[36] = d36
					ps70.OverlayValues[37] = d37
					ps70.OverlayValues[38] = d38
					ps70.OverlayValues[39] = d39
					ps70.OverlayValues[40] = d40
					ps70.PhiValues = make([]JITValueDesc, 1)
					d71 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(0)}
					ps70.PhiValues[0] = d71
					if ps70.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps70)
					return result
				}
				bbs[5].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[5].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					ctx.ReclaimUntrackedRegs()
					var d72 JITValueDesc
					if d38.SliceSizeKnown {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.KnownSliceLen))}
					} else if d38.Loc == LocImm {
						d72 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.StackOff))}
					} else if d38.Loc == LocStackTriple {
						d72 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
							d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2, ID: 0}
						} else if d38.Loc == LocReg {
							d72 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d72)
					ctx.EnsureDesc(&d31)
					var d73 JITValueDesc
					if d72.Loc == LocImm && d31.Loc == LocImm {
						d73 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d72.Imm.Int() % d31.Imm.Int())}
					} else {
						d73 = ctx.EmitGoCallScalar(GoFuncAddr(JITIntRem), []JITValueDesc{d72, d31}, 1)
					}
					if d73.Loc == LocReg && d72.Loc == LocReg && d73.Reg == d72.Reg {
						ctx.TransferReg(d72.Reg)
						d72.Loc = LocNone
					}
					ctx.FreeDesc(&d72)
					ctx.EnsureDesc(&d73)
					var d74 JITValueDesc
					if d73.Loc == LocImm {
						d74 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d73.Imm.Int() != 0)}
					} else {
						r8 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d73.Reg, 0)
						ctx.EmitSetcc(r8, CondNotEqual)
						d74 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r8}
						ctx.BindReg(r8, &d74)
					}
					ctx.FreeDesc(&d73)
					d75 = d74
					ctx.EnsureDesc(&d75)
					if d75.Loc != LocImm && d75.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d75.Loc == LocImm {
						if d75.Imm.Bool() {
							if ps.General {
							}
							ps76 := PhiState{General: ps.General}
							ps76.OverlayValues = make([]JITValueDesc, 76)
							ps76.OverlayValues[1] = d1
							ps76.OverlayValues[2] = d2
							ps76.OverlayValues[3] = d3
							ps76.OverlayValues[4] = d4
							ps76.OverlayValues[5] = d5
							ps76.OverlayValues[6] = d6
							ps76.OverlayValues[7] = d7
							ps76.OverlayValues[8] = d8
							ps76.OverlayValues[9] = d9
							ps76.OverlayValues[10] = d10
							ps76.OverlayValues[11] = d11
							ps76.OverlayValues[28] = d28
							ps76.OverlayValues[29] = d29
							ps76.OverlayValues[30] = d30
							ps76.OverlayValues[31] = d31
							ps76.OverlayValues[32] = d32
							ps76.OverlayValues[33] = d33
							ps76.OverlayValues[34] = d34
							ps76.OverlayValues[35] = d35
							ps76.OverlayValues[36] = d36
							ps76.OverlayValues[37] = d37
							ps76.OverlayValues[38] = d38
							ps76.OverlayValues[39] = d39
							ps76.OverlayValues[40] = d40
							ps76.OverlayValues[71] = d71
							ps76.OverlayValues[72] = d72
							ps76.OverlayValues[73] = d73
							ps76.OverlayValues[74] = d74
							ps76.OverlayValues[75] = d75
							return bbs[3].RenderPS(ps76)
						}
						if ps.General {
						}
						ps77 := PhiState{General: ps.General}
						ps77.OverlayValues = make([]JITValueDesc, 76)
						ps77.OverlayValues[1] = d1
						ps77.OverlayValues[2] = d2
						ps77.OverlayValues[3] = d3
						ps77.OverlayValues[4] = d4
						ps77.OverlayValues[5] = d5
						ps77.OverlayValues[6] = d6
						ps77.OverlayValues[7] = d7
						ps77.OverlayValues[8] = d8
						ps77.OverlayValues[9] = d9
						ps77.OverlayValues[10] = d10
						ps77.OverlayValues[11] = d11
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
						ps77.OverlayValues[71] = d71
						ps77.OverlayValues[72] = d72
						ps77.OverlayValues[73] = d73
						ps77.OverlayValues[74] = d74
						ps77.OverlayValues[75] = d75
						return bbs[4].RenderPS(ps77)
					}
					if !ps.General {
						ps.General = true
						return bbs[5].RenderPS(ps)
					}
					lbl18 := ctx.ReserveLabel()
					lbl19 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d75.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl18)
					ctx.EmitJmp(lbl19)
					ctx.MarkLabel(lbl18)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl19)
					ctx.EmitJmp(lbl5)
					ps78 := PhiState{General: true}
					ps78.OverlayValues = make([]JITValueDesc, 76)
					ps78.OverlayValues[1] = d1
					ps78.OverlayValues[2] = d2
					ps78.OverlayValues[3] = d3
					ps78.OverlayValues[4] = d4
					ps78.OverlayValues[5] = d5
					ps78.OverlayValues[6] = d6
					ps78.OverlayValues[7] = d7
					ps78.OverlayValues[8] = d8
					ps78.OverlayValues[9] = d9
					ps78.OverlayValues[10] = d10
					ps78.OverlayValues[11] = d11
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
					ps78.OverlayValues[71] = d71
					ps78.OverlayValues[72] = d72
					ps78.OverlayValues[73] = d73
					ps78.OverlayValues[74] = d74
					ps78.OverlayValues[75] = d75
					ps79 := PhiState{General: true}
					ps79.OverlayValues = make([]JITValueDesc, 76)
					ps79.OverlayValues[1] = d1
					ps79.OverlayValues[2] = d2
					ps79.OverlayValues[3] = d3
					ps79.OverlayValues[4] = d4
					ps79.OverlayValues[5] = d5
					ps79.OverlayValues[6] = d6
					ps79.OverlayValues[7] = d7
					ps79.OverlayValues[8] = d8
					ps79.OverlayValues[9] = d9
					ps79.OverlayValues[10] = d10
					ps79.OverlayValues[11] = d11
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
					ps79.OverlayValues[71] = d71
					ps79.OverlayValues[72] = d72
					ps79.OverlayValues[73] = d73
					ps79.OverlayValues[74] = d74
					ps79.OverlayValues[75] = d75
					snap80 := d1
					snap81 := d2
					snap82 := d3
					snap83 := d4
					snap84 := d5
					snap85 := d6
					snap86 := d7
					snap87 := d8
					snap88 := d9
					snap89 := d10
					snap90 := d11
					snap91 := d28
					snap92 := d29
					snap93 := d30
					snap94 := d31
					snap95 := d32
					snap96 := d33
					snap97 := d34
					snap98 := d35
					snap99 := d36
					snap100 := d37
					snap101 := d38
					snap102 := d39
					snap103 := d40
					snap104 := d71
					snap105 := d72
					snap106 := d73
					snap107 := d74
					snap108 := d75
					alloc109 := ctx.SnapshotAllocState()
					if !bbs[4].Rendered {
						bbs[4].RenderPS(ps79)
					}
					ctx.RestoreAllocState(alloc109)
					d1 = snap80
					d2 = snap81
					d3 = snap82
					d4 = snap83
					d5 = snap84
					d6 = snap85
					d7 = snap86
					d8 = snap87
					d9 = snap88
					d10 = snap89
					d11 = snap90
					d28 = snap91
					d29 = snap92
					d30 = snap93
					d31 = snap94
					d32 = snap95
					d33 = snap96
					d34 = snap97
					d35 = snap98
					d36 = snap99
					d37 = snap100
					d38 = snap101
					d39 = snap102
					d40 = snap103
					d71 = snap104
					d72 = snap105
					d73 = snap106
					d74 = snap107
					d75 = snap108
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps78)
					}
					return result
					ctx.FreeDesc(&d74)
					return result
				}
				bbs[6].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[6].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					ctx.ReclaimUntrackedRegs()
					var d110 JITValueDesc
					if d38.SliceSizeKnown {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.KnownSliceLen))}
					} else if d38.Loc == LocImm {
						d110 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.StackOff))}
					} else if d38.Loc == LocStackTriple {
						d110 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
							d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2, ID: 0}
						} else if d38.Loc == LocReg {
							d110 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d110)
					var d111 JITValueDesc
					if d110.Loc == LocImm {
						d111 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d110.Imm.Int() == 0)}
					} else {
						r9 := ctx.AllocReg()
						ctx.EmitCmpRegImm32(d110.Reg, 0)
						ctx.EmitSetcc(r9, CondEqual)
						d111 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r9}
						ctx.BindReg(r9, &d111)
					}
					ctx.FreeDesc(&d110)
					d112 = d111
					ctx.EnsureDesc(&d112)
					if d112.Loc != LocImm && d112.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d112.Loc == LocImm {
						if d112.Imm.Bool() {
							if ps.General {
							}
							ps113 := PhiState{General: ps.General}
							ps113.OverlayValues = make([]JITValueDesc, 113)
							ps113.OverlayValues[1] = d1
							ps113.OverlayValues[2] = d2
							ps113.OverlayValues[3] = d3
							ps113.OverlayValues[4] = d4
							ps113.OverlayValues[5] = d5
							ps113.OverlayValues[6] = d6
							ps113.OverlayValues[7] = d7
							ps113.OverlayValues[8] = d8
							ps113.OverlayValues[9] = d9
							ps113.OverlayValues[10] = d10
							ps113.OverlayValues[11] = d11
							ps113.OverlayValues[28] = d28
							ps113.OverlayValues[29] = d29
							ps113.OverlayValues[30] = d30
							ps113.OverlayValues[31] = d31
							ps113.OverlayValues[32] = d32
							ps113.OverlayValues[33] = d33
							ps113.OverlayValues[34] = d34
							ps113.OverlayValues[35] = d35
							ps113.OverlayValues[36] = d36
							ps113.OverlayValues[37] = d37
							ps113.OverlayValues[38] = d38
							ps113.OverlayValues[39] = d39
							ps113.OverlayValues[40] = d40
							ps113.OverlayValues[71] = d71
							ps113.OverlayValues[72] = d72
							ps113.OverlayValues[73] = d73
							ps113.OverlayValues[74] = d74
							ps113.OverlayValues[75] = d75
							ps113.OverlayValues[110] = d110
							ps113.OverlayValues[111] = d111
							ps113.OverlayValues[112] = d112
							return bbs[3].RenderPS(ps113)
						}
						if ps.General {
						}
						ps114 := PhiState{General: ps.General}
						ps114.OverlayValues = make([]JITValueDesc, 113)
						ps114.OverlayValues[1] = d1
						ps114.OverlayValues[2] = d2
						ps114.OverlayValues[3] = d3
						ps114.OverlayValues[4] = d4
						ps114.OverlayValues[5] = d5
						ps114.OverlayValues[6] = d6
						ps114.OverlayValues[7] = d7
						ps114.OverlayValues[8] = d8
						ps114.OverlayValues[9] = d9
						ps114.OverlayValues[10] = d10
						ps114.OverlayValues[11] = d11
						ps114.OverlayValues[28] = d28
						ps114.OverlayValues[29] = d29
						ps114.OverlayValues[30] = d30
						ps114.OverlayValues[31] = d31
						ps114.OverlayValues[32] = d32
						ps114.OverlayValues[33] = d33
						ps114.OverlayValues[34] = d34
						ps114.OverlayValues[35] = d35
						ps114.OverlayValues[36] = d36
						ps114.OverlayValues[37] = d37
						ps114.OverlayValues[38] = d38
						ps114.OverlayValues[39] = d39
						ps114.OverlayValues[40] = d40
						ps114.OverlayValues[71] = d71
						ps114.OverlayValues[72] = d72
						ps114.OverlayValues[73] = d73
						ps114.OverlayValues[74] = d74
						ps114.OverlayValues[75] = d75
						ps114.OverlayValues[110] = d110
						ps114.OverlayValues[111] = d111
						ps114.OverlayValues[112] = d112
						return bbs[5].RenderPS(ps114)
					}
					if !ps.General {
						ps.General = true
						return bbs[6].RenderPS(ps)
					}
					lbl20 := ctx.ReserveLabel()
					lbl21 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d112.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl20)
					ctx.EmitJmp(lbl21)
					ctx.MarkLabel(lbl20)
					ctx.EmitJmp(lbl4)
					ctx.MarkLabel(lbl21)
					ctx.EmitJmp(lbl6)
					ps115 := PhiState{General: true}
					ps115.OverlayValues = make([]JITValueDesc, 113)
					ps115.OverlayValues[1] = d1
					ps115.OverlayValues[2] = d2
					ps115.OverlayValues[3] = d3
					ps115.OverlayValues[4] = d4
					ps115.OverlayValues[5] = d5
					ps115.OverlayValues[6] = d6
					ps115.OverlayValues[7] = d7
					ps115.OverlayValues[8] = d8
					ps115.OverlayValues[9] = d9
					ps115.OverlayValues[10] = d10
					ps115.OverlayValues[11] = d11
					ps115.OverlayValues[28] = d28
					ps115.OverlayValues[29] = d29
					ps115.OverlayValues[30] = d30
					ps115.OverlayValues[31] = d31
					ps115.OverlayValues[32] = d32
					ps115.OverlayValues[33] = d33
					ps115.OverlayValues[34] = d34
					ps115.OverlayValues[35] = d35
					ps115.OverlayValues[36] = d36
					ps115.OverlayValues[37] = d37
					ps115.OverlayValues[38] = d38
					ps115.OverlayValues[39] = d39
					ps115.OverlayValues[40] = d40
					ps115.OverlayValues[71] = d71
					ps115.OverlayValues[72] = d72
					ps115.OverlayValues[73] = d73
					ps115.OverlayValues[74] = d74
					ps115.OverlayValues[75] = d75
					ps115.OverlayValues[110] = d110
					ps115.OverlayValues[111] = d111
					ps115.OverlayValues[112] = d112
					ps116 := PhiState{General: true}
					ps116.OverlayValues = make([]JITValueDesc, 113)
					ps116.OverlayValues[1] = d1
					ps116.OverlayValues[2] = d2
					ps116.OverlayValues[3] = d3
					ps116.OverlayValues[4] = d4
					ps116.OverlayValues[5] = d5
					ps116.OverlayValues[6] = d6
					ps116.OverlayValues[7] = d7
					ps116.OverlayValues[8] = d8
					ps116.OverlayValues[9] = d9
					ps116.OverlayValues[10] = d10
					ps116.OverlayValues[11] = d11
					ps116.OverlayValues[28] = d28
					ps116.OverlayValues[29] = d29
					ps116.OverlayValues[30] = d30
					ps116.OverlayValues[31] = d31
					ps116.OverlayValues[32] = d32
					ps116.OverlayValues[33] = d33
					ps116.OverlayValues[34] = d34
					ps116.OverlayValues[35] = d35
					ps116.OverlayValues[36] = d36
					ps116.OverlayValues[37] = d37
					ps116.OverlayValues[38] = d38
					ps116.OverlayValues[39] = d39
					ps116.OverlayValues[40] = d40
					ps116.OverlayValues[71] = d71
					ps116.OverlayValues[72] = d72
					ps116.OverlayValues[73] = d73
					ps116.OverlayValues[74] = d74
					ps116.OverlayValues[75] = d75
					ps116.OverlayValues[110] = d110
					ps116.OverlayValues[111] = d111
					ps116.OverlayValues[112] = d112
					snap117 := d1
					snap118 := d2
					snap119 := d3
					snap120 := d4
					snap121 := d5
					snap122 := d6
					snap123 := d7
					snap124 := d8
					snap125 := d9
					snap126 := d10
					snap127 := d11
					snap128 := d28
					snap129 := d29
					snap130 := d30
					snap131 := d31
					snap132 := d32
					snap133 := d33
					snap134 := d34
					snap135 := d35
					snap136 := d36
					snap137 := d37
					snap138 := d38
					snap139 := d39
					snap140 := d40
					snap141 := d71
					snap142 := d72
					snap143 := d73
					snap144 := d74
					snap145 := d75
					snap146 := d110
					snap147 := d111
					snap148 := d112
					alloc149 := ctx.SnapshotAllocState()
					if !bbs[5].Rendered {
						bbs[5].RenderPS(ps116)
					}
					ctx.RestoreAllocState(alloc149)
					d1 = snap117
					d2 = snap118
					d3 = snap119
					d4 = snap120
					d5 = snap121
					d6 = snap122
					d7 = snap123
					d8 = snap124
					d9 = snap125
					d10 = snap126
					d11 = snap127
					d28 = snap128
					d29 = snap129
					d30 = snap130
					d31 = snap131
					d32 = snap132
					d33 = snap133
					d34 = snap134
					d35 = snap135
					d36 = snap136
					d37 = snap137
					d38 = snap138
					d39 = snap139
					d40 = snap140
					d71 = snap141
					d72 = snap142
					d73 = snap143
					d74 = snap144
					d75 = snap145
					d110 = snap146
					d111 = snap147
					d112 = snap148
					if !bbs[3].Rendered {
						return bbs[3].RenderPS(ps115)
					}
					return result
					ctx.FreeDesc(&d111)
					return result
				}
				bbs[7].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d150 := ps.PhiValues[0]
							ctx.EnsureDesc(&d150)
							ctx.EmitStoreToStack(d150, int32(bbs[7].PhiBase)+int32(0))
						}
						if bbs[7].VisitCount >= 0 {
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
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d1 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d1)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d7)
					var d151 JITValueDesc
					if d1.Loc == LocImm && d7.Loc == LocImm {
						d151 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d1.Imm.Int() < d7.Imm.Int())}
					} else if d7.Loc == LocImm {
						r10 := ctx.AllocRegExcept(d1.Reg)
						if d7.Imm.Int() >= -2147483648 && d7.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d1.Reg, int32(d7.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d7.Imm.Int()))
							ctx.EmitCmpInt64(d1.Reg, RegR11)
						}
						ctx.EmitSetcc(r10, CondSignedLess)
						d151 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r10}
						ctx.BindReg(r10, &d151)
					} else if d1.Loc == LocImm {
						r11 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d1.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d7.Reg)
						ctx.EmitSetcc(r11, CondSignedLess)
						d151 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r11}
						ctx.BindReg(r11, &d151)
					} else {
						r12 := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitCmpInt64(d1.Reg, d7.Reg)
						ctx.EmitSetcc(r12, CondSignedLess)
						d151 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r12}
						ctx.BindReg(r12, &d151)
					}
					ctx.FreeDesc(&d7)
					d152 = d151
					ctx.EnsureDesc(&d152)
					if d152.Loc != LocImm && d152.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d152.Loc == LocImm {
						if d152.Imm.Bool() {
							if ps.General {
							}
							ps153 := PhiState{General: ps.General}
							ps153.OverlayValues = make([]JITValueDesc, 153)
							ps153.OverlayValues[1] = d1
							ps153.OverlayValues[2] = d2
							ps153.OverlayValues[3] = d3
							ps153.OverlayValues[4] = d4
							ps153.OverlayValues[5] = d5
							ps153.OverlayValues[6] = d6
							ps153.OverlayValues[7] = d7
							ps153.OverlayValues[8] = d8
							ps153.OverlayValues[9] = d9
							ps153.OverlayValues[10] = d10
							ps153.OverlayValues[11] = d11
							ps153.OverlayValues[28] = d28
							ps153.OverlayValues[29] = d29
							ps153.OverlayValues[30] = d30
							ps153.OverlayValues[31] = d31
							ps153.OverlayValues[32] = d32
							ps153.OverlayValues[33] = d33
							ps153.OverlayValues[34] = d34
							ps153.OverlayValues[35] = d35
							ps153.OverlayValues[36] = d36
							ps153.OverlayValues[37] = d37
							ps153.OverlayValues[38] = d38
							ps153.OverlayValues[39] = d39
							ps153.OverlayValues[40] = d40
							ps153.OverlayValues[71] = d71
							ps153.OverlayValues[72] = d72
							ps153.OverlayValues[73] = d73
							ps153.OverlayValues[74] = d74
							ps153.OverlayValues[75] = d75
							ps153.OverlayValues[110] = d110
							ps153.OverlayValues[111] = d111
							ps153.OverlayValues[112] = d112
							ps153.OverlayValues[150] = d150
							ps153.OverlayValues[151] = d151
							ps153.OverlayValues[152] = d152
							return bbs[8].RenderPS(ps153)
						}
						if ps.General {
						}
						ps154 := PhiState{General: ps.General}
						ps154.OverlayValues = make([]JITValueDesc, 153)
						ps154.OverlayValues[1] = d1
						ps154.OverlayValues[2] = d2
						ps154.OverlayValues[3] = d3
						ps154.OverlayValues[4] = d4
						ps154.OverlayValues[5] = d5
						ps154.OverlayValues[6] = d6
						ps154.OverlayValues[7] = d7
						ps154.OverlayValues[8] = d8
						ps154.OverlayValues[9] = d9
						ps154.OverlayValues[10] = d10
						ps154.OverlayValues[11] = d11
						ps154.OverlayValues[28] = d28
						ps154.OverlayValues[29] = d29
						ps154.OverlayValues[30] = d30
						ps154.OverlayValues[31] = d31
						ps154.OverlayValues[32] = d32
						ps154.OverlayValues[33] = d33
						ps154.OverlayValues[34] = d34
						ps154.OverlayValues[35] = d35
						ps154.OverlayValues[36] = d36
						ps154.OverlayValues[37] = d37
						ps154.OverlayValues[38] = d38
						ps154.OverlayValues[39] = d39
						ps154.OverlayValues[40] = d40
						ps154.OverlayValues[71] = d71
						ps154.OverlayValues[72] = d72
						ps154.OverlayValues[73] = d73
						ps154.OverlayValues[74] = d74
						ps154.OverlayValues[75] = d75
						ps154.OverlayValues[110] = d110
						ps154.OverlayValues[111] = d111
						ps154.OverlayValues[112] = d112
						ps154.OverlayValues[150] = d150
						ps154.OverlayValues[151] = d151
						ps154.OverlayValues[152] = d152
						return bbs[9].RenderPS(ps154)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d155 := ps.PhiValues[0]
							ctx.EnsureDesc(&d155)
							ctx.EmitStoreToStack(d155, int32(bbs[7].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[7].RenderPS(ps)
					}
					lbl22 := ctx.ReserveLabel()
					lbl23 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d152.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl22)
					ctx.EmitJmp(lbl23)
					ctx.MarkLabel(lbl22)
					ctx.EmitJmp(lbl9)
					ctx.MarkLabel(lbl23)
					ctx.EmitJmp(lbl10)
					ps156 := PhiState{General: true}
					ps156.OverlayValues = make([]JITValueDesc, 156)
					ps156.OverlayValues[1] = d1
					ps156.OverlayValues[2] = d2
					ps156.OverlayValues[3] = d3
					ps156.OverlayValues[4] = d4
					ps156.OverlayValues[5] = d5
					ps156.OverlayValues[6] = d6
					ps156.OverlayValues[7] = d7
					ps156.OverlayValues[8] = d8
					ps156.OverlayValues[9] = d9
					ps156.OverlayValues[10] = d10
					ps156.OverlayValues[11] = d11
					ps156.OverlayValues[28] = d28
					ps156.OverlayValues[29] = d29
					ps156.OverlayValues[30] = d30
					ps156.OverlayValues[31] = d31
					ps156.OverlayValues[32] = d32
					ps156.OverlayValues[33] = d33
					ps156.OverlayValues[34] = d34
					ps156.OverlayValues[35] = d35
					ps156.OverlayValues[36] = d36
					ps156.OverlayValues[37] = d37
					ps156.OverlayValues[38] = d38
					ps156.OverlayValues[39] = d39
					ps156.OverlayValues[40] = d40
					ps156.OverlayValues[71] = d71
					ps156.OverlayValues[72] = d72
					ps156.OverlayValues[73] = d73
					ps156.OverlayValues[74] = d74
					ps156.OverlayValues[75] = d75
					ps156.OverlayValues[110] = d110
					ps156.OverlayValues[111] = d111
					ps156.OverlayValues[112] = d112
					ps156.OverlayValues[150] = d150
					ps156.OverlayValues[151] = d151
					ps156.OverlayValues[152] = d152
					ps156.OverlayValues[155] = d155
					ps157 := PhiState{General: true}
					ps157.OverlayValues = make([]JITValueDesc, 156)
					ps157.OverlayValues[1] = d1
					ps157.OverlayValues[2] = d2
					ps157.OverlayValues[3] = d3
					ps157.OverlayValues[4] = d4
					ps157.OverlayValues[5] = d5
					ps157.OverlayValues[6] = d6
					ps157.OverlayValues[7] = d7
					ps157.OverlayValues[8] = d8
					ps157.OverlayValues[9] = d9
					ps157.OverlayValues[10] = d10
					ps157.OverlayValues[11] = d11
					ps157.OverlayValues[28] = d28
					ps157.OverlayValues[29] = d29
					ps157.OverlayValues[30] = d30
					ps157.OverlayValues[31] = d31
					ps157.OverlayValues[32] = d32
					ps157.OverlayValues[33] = d33
					ps157.OverlayValues[34] = d34
					ps157.OverlayValues[35] = d35
					ps157.OverlayValues[36] = d36
					ps157.OverlayValues[37] = d37
					ps157.OverlayValues[38] = d38
					ps157.OverlayValues[39] = d39
					ps157.OverlayValues[40] = d40
					ps157.OverlayValues[71] = d71
					ps157.OverlayValues[72] = d72
					ps157.OverlayValues[73] = d73
					ps157.OverlayValues[74] = d74
					ps157.OverlayValues[75] = d75
					ps157.OverlayValues[110] = d110
					ps157.OverlayValues[111] = d111
					ps157.OverlayValues[112] = d112
					ps157.OverlayValues[150] = d150
					ps157.OverlayValues[151] = d151
					ps157.OverlayValues[152] = d152
					ps157.OverlayValues[155] = d155
					snap158 := d1
					snap159 := d2
					snap160 := d3
					snap161 := d4
					snap162 := d5
					snap163 := d6
					snap164 := d7
					snap165 := d8
					snap166 := d9
					snap167 := d10
					snap168 := d11
					snap169 := d28
					snap170 := d29
					snap171 := d30
					snap172 := d31
					snap173 := d32
					snap174 := d33
					snap175 := d34
					snap176 := d35
					snap177 := d36
					snap178 := d37
					snap179 := d38
					snap180 := d39
					snap181 := d40
					snap182 := d71
					snap183 := d72
					snap184 := d73
					snap185 := d74
					snap186 := d75
					snap187 := d110
					snap188 := d111
					snap189 := d112
					snap190 := d150
					snap191 := d151
					snap192 := d152
					snap193 := d155
					alloc194 := ctx.SnapshotAllocState()
					if !bbs[9].Rendered {
						bbs[9].RenderPS(ps157)
					}
					ctx.RestoreAllocState(alloc194)
					d1 = snap158
					d2 = snap159
					d3 = snap160
					d4 = snap161
					d5 = snap162
					d6 = snap163
					d7 = snap164
					d8 = snap165
					d9 = snap166
					d10 = snap167
					d11 = snap168
					d28 = snap169
					d29 = snap170
					d30 = snap171
					d31 = snap172
					d32 = snap173
					d33 = snap174
					d34 = snap175
					d35 = snap176
					d36 = snap177
					d37 = snap178
					d38 = snap179
					d39 = snap180
					d40 = snap181
					d71 = snap182
					d72 = snap183
					d73 = snap184
					d74 = snap185
					d75 = snap186
					d110 = snap187
					d111 = snap188
					d112 = snap189
					d150 = snap190
					d151 = snap191
					d152 = snap192
					d155 = snap193
					if !bbs[8].Rendered {
						return bbs[8].RenderPS(ps156)
					}
					return result
					ctx.FreeDesc(&d151)
					return result
				}
				bbs[8].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[8].VisitCount >= 0 {
							ps.General = true
							return bbs[8].RenderPS(ps)
						}
					}
					bbs[8].VisitCount++
					if ps.General {
						if bbs[8].Rendered {
							ctx.EmitJmp(lbl9)
							return result
						}
						bbs[8].Rendered = true
						bbs[8].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_8 = bbs[8].Address
						ctx.MarkLabel(lbl9)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d31)
					var d195 JITValueDesc
					ctx.EnsureDesc(&d38)
					if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
						d195 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2}
						ctx.BindReg(d38.Reg2, &d195)
					} else {
						panic("Slice with omitted high requires descriptor with length in Reg2")
					}
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d195)
					var d197 JITValueDesc
					if d195.Loc == LocImm && d31.Loc == LocImm {
						d197 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d195.Imm.Int() - d31.Imm.Int())}
					} else {
						r13 := ctx.AllocReg()
						if d195.Loc == LocImm {
							ctx.EmitMovRegImm64(r13, uint64(d195.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r13, d195.Reg)
						}
						if d31.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
							ctx.EmitSubInt64(r13, RegR11)
						} else {
							ctx.EmitSubInt64(r13, d31.Reg)
						}
						d197 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r13}
						ctx.BindReg(r13, &d197)
					}
					var d198 JITValueDesc
					if d38.Loc == LocImm && d31.Loc == LocImm {
						d198 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d38.Imm.Int() + d31.Imm.Int()*16)}
					} else {
						r14 := ctx.AllocReg()
						if d38.Loc == LocImm {
							ctx.EmitMovRegImm64(r14, uint64(d38.Imm.Int()))
						} else {
							ctx.EmitMovRegReg(r14, d38.Reg)
						}
						if d31.Loc == LocImm {
							ctx.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()*16))
							ctx.EmitAddInt64(r14, RegR11)
						} else {
							offsetReg := ctx.AllocRegExcept(r14, d31.Reg)
							ctx.EmitMovRegReg(offsetReg, d31.Reg)
							ctx.EmitShlRegImm8(offsetReg, 4)
							ctx.EmitAddInt64(r14, offsetReg)
							ctx.FreeReg(offsetReg)
						}
						d198 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r14}
						ctx.BindReg(r14, &d198)
					}
					var d199 JITValueDesc
					var r15 Reg
					var r16 Reg
					ctx.SyncDesc(&d198)
					ctx.EnsureDesc(&d198)
					if d198.Loc == LocImm {
						r15 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r15, uint64(d198.Imm.Int()))
					} else {
						r15 = d198.Reg
					}
					ctx.ProtectReg(r15)
					ctx.SyncDesc(&d197)
					ctx.EnsureDesc(&d197)
					if d197.Loc == LocImm {
						r16 = ctx.AllocReg()
						ctx.EmitMovRegImm64(r16, uint64(d197.Imm.Int()))
					} else {
						r16 = d197.Reg
					}
					ctx.ProtectReg(r16)
					r17 := ctx.EmitSliceCapAfterLow(&d38, &d31, r15, r16)
					ctx.UnprotectReg(r16)
					ctx.UnprotectReg(r15)
					d199 = JITValueDesc{Loc: LocRegTriple, Reg: r15, Reg2: r16, Reg3: r17}
					ctx.BindReg(r15, &d199)
					ctx.BindReg(r16, &d199)
					ctx.BindReg(r17, &d199)
					ctx.BindReg(r15, &d199)
					ctx.BindReg(r16, &d199)
					ctx.BindReg(r17, &d199)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d199)
					ctx.EnsureDesc(&d38)
					ctx.EnsureDesc(&d199)
					callResults200 := JITEmitGoCallResults(ctx, GoFuncAddr(jitCopyScmerSlice), []JITValueDesc{d38, d199}, []uint8{1}, []uint8{0})
					d201 = callResults200[0]
					d201.Type = tagInt
					var d202 JITValueDesc
					if d38.SliceSizeKnown {
						d202 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.KnownSliceLen))}
					} else if d38.Loc == LocImm {
						d202 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.StackOff))}
					} else if d38.Loc == LocStackTriple {
						d202 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
							d202 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2, ID: 0}
						} else if d38.Loc == LocReg {
							d202 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d202)
					ctx.EnsureDesc(&d31)
					ctx.EnsureDesc(&d202)
					ctx.ProtectReg(d202.Reg)
					ctx.EnsureDesc(&d31)
					ctx.UnprotectReg(d202.Reg)
					var d203 JITValueDesc
					if d202.Loc == LocImm && d31.Loc == LocImm {
						d203 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d202.Imm.Int() - d31.Imm.Int())}
					} else if d31.Loc == LocImm && d31.Imm.Int() == 0 {
						r18 := ctx.AllocRegExcept(d202.Reg)
						ctx.EmitMovRegReg(r18, d202.Reg)
						d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r18}
						ctx.BindReg(r18, &d203)
					} else if d202.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d31.Reg)
						ctx.EmitMovRegImm64(scratch, uint64(d202.Imm.Int()))
						ctx.EmitSubInt64(scratch, d31.Reg)
						d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d203)
					} else if d31.Loc == LocImm {
						scratch := ctx.AllocRegExcept(d202.Reg)
						ctx.EmitMovRegReg(scratch, d202.Reg)
						if d31.Imm.Int() >= -2147483648 && d31.Imm.Int() <= 2147483647 {
							ctx.EmitSubRegImm32(scratch, int32(d31.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d31.Imm.Int()))
							ctx.EmitSubInt64(scratch, RegR11)
						}
						d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d203)
					} else {
						r19 := ctx.AllocRegExcept(d202.Reg, d31.Reg)
						ctx.EmitMovRegReg(r19, d202.Reg)
						ctx.EmitSubInt64(r19, d31.Reg)
						d203 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: r19}
						ctx.BindReg(r19, &d203)
					}
					if d203.Loc == LocReg && d202.Loc == LocReg && d203.Reg == d202.Reg {
						ctx.TransferReg(d202.Reg)
						d202.Loc = LocNone
					}
					ctx.EnsureDesc(&d203)
					ctx.EmitStoreToStack(d203, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d203)
					ctx.FreeDesc(&d202)
					ctx.FreeDesc(&d31)
					if ps.General {
					}
					ps204 := PhiState{General: ps.General}
					ps204.OverlayValues = make([]JITValueDesc, 204)
					ps204.OverlayValues[1] = d1
					ps204.OverlayValues[2] = d2
					ps204.OverlayValues[3] = d3
					ps204.OverlayValues[4] = d4
					ps204.OverlayValues[5] = d5
					ps204.OverlayValues[6] = d6
					ps204.OverlayValues[7] = d7
					ps204.OverlayValues[8] = d8
					ps204.OverlayValues[9] = d9
					ps204.OverlayValues[10] = d10
					ps204.OverlayValues[11] = d11
					ps204.OverlayValues[28] = d28
					ps204.OverlayValues[29] = d29
					ps204.OverlayValues[30] = d30
					ps204.OverlayValues[31] = d31
					ps204.OverlayValues[32] = d32
					ps204.OverlayValues[33] = d33
					ps204.OverlayValues[34] = d34
					ps204.OverlayValues[35] = d35
					ps204.OverlayValues[36] = d36
					ps204.OverlayValues[37] = d37
					ps204.OverlayValues[38] = d38
					ps204.OverlayValues[39] = d39
					ps204.OverlayValues[40] = d40
					ps204.OverlayValues[71] = d71
					ps204.OverlayValues[72] = d72
					ps204.OverlayValues[73] = d73
					ps204.OverlayValues[74] = d74
					ps204.OverlayValues[75] = d75
					ps204.OverlayValues[110] = d110
					ps204.OverlayValues[111] = d111
					ps204.OverlayValues[112] = d112
					ps204.OverlayValues[150] = d150
					ps204.OverlayValues[151] = d151
					ps204.OverlayValues[152] = d152
					ps204.OverlayValues[155] = d155
					ps204.OverlayValues[195] = d195
					ps204.OverlayValues[196] = d196
					ps204.OverlayValues[197] = d197
					ps204.OverlayValues[198] = d198
					ps204.OverlayValues[199] = d199
					ps204.OverlayValues[201] = d201
					ps204.OverlayValues[202] = d202
					ps204.OverlayValues[203] = d203
					ps204.PhiValues = make([]JITValueDesc, 1)
					if ps204.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps204)
					return result
				}
				bbs[9].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[9].VisitCount >= 0 {
							ps.General = true
							return bbs[9].RenderPS(ps)
						}
					}
					bbs[9].VisitCount++
					if ps.General {
						if bbs[9].Rendered {
							ctx.EmitJmp(lbl10)
							return result
						}
						bbs[9].Rendered = true
						bbs[9].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_9 = bbs[9].Address
						ctx.MarkLabel(lbl10)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					ctx.ReclaimUntrackedRegs()
					d205 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d205)
					if d205.Loc == LocRegPair {
						ctx.EmitMovPairToResult(&d205, &result)
						result.Type = d205.Type
					} else {
						switch d205.Type {
						case tagBool:
							ctx.EmitMakeBool(result, d205)
							result.Type = tagBool
						case tagInt:
							ctx.EmitMakeInt(result, d205)
							result.Type = tagInt
						case tagFloat:
							ctx.EmitMakeFloat(result, d205)
							result.Type = tagFloat
						case tagNil:
							ctx.EmitMakeNil(result)
							result.Type = tagNil
						default:
							ctx.EmitMovPairToResult(&d205, &result)
							result.Type = d205.Type
						}
					}
					ctx.EmitJmp(lbl0)
					return result
				}
				bbs[10].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d206 := ps.PhiValues[0]
							ctx.EnsureDesc(&d206)
							ctx.EmitStoreToStack(d206, int32(bbs[10].PhiBase)+int32(0))
						}
						if bbs[10].VisitCount >= 0 {
							ps.General = true
							return bbs[10].RenderPS(ps)
						}
					}
					bbs[10].VisitCount++
					if ps.General {
						if bbs[10].Rendered {
							ctx.EmitJmp(lbl11)
							return result
						}
						bbs[10].Rendered = true
						bbs[10].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_10 = bbs[10].Address
						ctx.MarkLabel(lbl11)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if !ps.General && len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
						d2 = ps.PhiValues[0]
					}
					ctx.ReclaimUntrackedRegs()
					ctx.StabilizeDescForControlFlow(&d2)
					var d207 JITValueDesc
					if d38.SliceSizeKnown {
						d207 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.KnownSliceLen))}
					} else if d38.Loc == LocImm {
						d207 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(d38.StackOff))}
					} else if d38.Loc == LocStackTriple {
						d207 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: d38.StackOff + 8, NoHeapPointer: true}
					} else {
						ctx.EnsureDesc(&d38)
						if d38.Loc == LocRegPair || d38.Loc == LocRegTriple {
							d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg2, ID: 0}
						} else if d38.Loc == LocReg {
							d207 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d38.Reg, ID: 0}
						} else {
							panic("len on unsupported descriptor location")
						}
					}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d207)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d207)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d207)
					var d208 JITValueDesc
					if d2.Loc == LocImm && d207.Loc == LocImm {
						d208 = JITValueDesc{Loc: LocImm, Type: tagBool, Imm: NewBool(d2.Imm.Int() < d207.Imm.Int())}
					} else if d207.Loc == LocImm {
						r20 := ctx.AllocRegExcept(d2.Reg)
						if d207.Imm.Int() >= -2147483648 && d207.Imm.Int() <= 2147483647 {
							ctx.EmitCmpRegImm32(d2.Reg, int32(d207.Imm.Int()))
						} else {
							ctx.EmitMovRegImm64(RegR11, uint64(d207.Imm.Int()))
							ctx.EmitCmpInt64(d2.Reg, RegR11)
						}
						ctx.EmitSetcc(r20, CondSignedLess)
						d208 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r20}
						ctx.BindReg(r20, &d208)
					} else if d2.Loc == LocImm {
						r21 := ctx.AllocReg()
						ctx.EmitMovRegImm64(RegR11, uint64(d2.Imm.Int()))
						ctx.EmitCmpInt64(RegR11, d207.Reg)
						ctx.EmitSetcc(r21, CondSignedLess)
						d208 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r21}
						ctx.BindReg(r21, &d208)
					} else {
						r22 := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitCmpInt64(d2.Reg, d207.Reg)
						ctx.EmitSetcc(r22, CondSignedLess)
						d208 = JITValueDesc{Loc: LocReg, Type: tagBool, Reg: r22}
						ctx.BindReg(r22, &d208)
					}
					ctx.FreeDesc(&d207)
					d209 = d208
					ctx.EnsureDesc(&d209)
					if d209.Loc != LocImm && d209.Loc != LocReg {
						panic("jit: If condition is neither LocImm nor LocReg")
					}
					if d209.Loc == LocImm {
						if d209.Imm.Bool() {
							if ps.General {
							}
							ps210 := PhiState{General: ps.General}
							ps210.OverlayValues = make([]JITValueDesc, 210)
							ps210.OverlayValues[1] = d1
							ps210.OverlayValues[2] = d2
							ps210.OverlayValues[3] = d3
							ps210.OverlayValues[4] = d4
							ps210.OverlayValues[5] = d5
							ps210.OverlayValues[6] = d6
							ps210.OverlayValues[7] = d7
							ps210.OverlayValues[8] = d8
							ps210.OverlayValues[9] = d9
							ps210.OverlayValues[10] = d10
							ps210.OverlayValues[11] = d11
							ps210.OverlayValues[28] = d28
							ps210.OverlayValues[29] = d29
							ps210.OverlayValues[30] = d30
							ps210.OverlayValues[31] = d31
							ps210.OverlayValues[32] = d32
							ps210.OverlayValues[33] = d33
							ps210.OverlayValues[34] = d34
							ps210.OverlayValues[35] = d35
							ps210.OverlayValues[36] = d36
							ps210.OverlayValues[37] = d37
							ps210.OverlayValues[38] = d38
							ps210.OverlayValues[39] = d39
							ps210.OverlayValues[40] = d40
							ps210.OverlayValues[71] = d71
							ps210.OverlayValues[72] = d72
							ps210.OverlayValues[73] = d73
							ps210.OverlayValues[74] = d74
							ps210.OverlayValues[75] = d75
							ps210.OverlayValues[110] = d110
							ps210.OverlayValues[111] = d111
							ps210.OverlayValues[112] = d112
							ps210.OverlayValues[150] = d150
							ps210.OverlayValues[151] = d151
							ps210.OverlayValues[152] = d152
							ps210.OverlayValues[155] = d155
							ps210.OverlayValues[195] = d195
							ps210.OverlayValues[196] = d196
							ps210.OverlayValues[197] = d197
							ps210.OverlayValues[198] = d198
							ps210.OverlayValues[199] = d199
							ps210.OverlayValues[201] = d201
							ps210.OverlayValues[202] = d202
							ps210.OverlayValues[203] = d203
							ps210.OverlayValues[205] = d205
							ps210.OverlayValues[206] = d206
							ps210.OverlayValues[207] = d207
							ps210.OverlayValues[208] = d208
							ps210.OverlayValues[209] = d209
							return bbs[11].RenderPS(ps210)
						}
						if ps.General {
						}
						ps211 := PhiState{General: ps.General}
						ps211.OverlayValues = make([]JITValueDesc, 210)
						ps211.OverlayValues[1] = d1
						ps211.OverlayValues[2] = d2
						ps211.OverlayValues[3] = d3
						ps211.OverlayValues[4] = d4
						ps211.OverlayValues[5] = d5
						ps211.OverlayValues[6] = d6
						ps211.OverlayValues[7] = d7
						ps211.OverlayValues[8] = d8
						ps211.OverlayValues[9] = d9
						ps211.OverlayValues[10] = d10
						ps211.OverlayValues[11] = d11
						ps211.OverlayValues[28] = d28
						ps211.OverlayValues[29] = d29
						ps211.OverlayValues[30] = d30
						ps211.OverlayValues[31] = d31
						ps211.OverlayValues[32] = d32
						ps211.OverlayValues[33] = d33
						ps211.OverlayValues[34] = d34
						ps211.OverlayValues[35] = d35
						ps211.OverlayValues[36] = d36
						ps211.OverlayValues[37] = d37
						ps211.OverlayValues[38] = d38
						ps211.OverlayValues[39] = d39
						ps211.OverlayValues[40] = d40
						ps211.OverlayValues[71] = d71
						ps211.OverlayValues[72] = d72
						ps211.OverlayValues[73] = d73
						ps211.OverlayValues[74] = d74
						ps211.OverlayValues[75] = d75
						ps211.OverlayValues[110] = d110
						ps211.OverlayValues[111] = d111
						ps211.OverlayValues[112] = d112
						ps211.OverlayValues[150] = d150
						ps211.OverlayValues[151] = d151
						ps211.OverlayValues[152] = d152
						ps211.OverlayValues[155] = d155
						ps211.OverlayValues[195] = d195
						ps211.OverlayValues[196] = d196
						ps211.OverlayValues[197] = d197
						ps211.OverlayValues[198] = d198
						ps211.OverlayValues[199] = d199
						ps211.OverlayValues[201] = d201
						ps211.OverlayValues[202] = d202
						ps211.OverlayValues[203] = d203
						ps211.OverlayValues[205] = d205
						ps211.OverlayValues[206] = d206
						ps211.OverlayValues[207] = d207
						ps211.OverlayValues[208] = d208
						ps211.OverlayValues[209] = d209
						return bbs[12].RenderPS(ps211)
					}
					if !ps.General {
						if len(ps.PhiValues) > 0 && ps.PhiValues[0].Loc != LocNone {
							d212 := ps.PhiValues[0]
							ctx.EnsureDesc(&d212)
							ctx.EmitStoreToStack(d212, int32(bbs[10].PhiBase)+int32(0))
						}
						ps.General = true
						return bbs[10].RenderPS(ps)
					}
					lbl24 := ctx.ReserveLabel()
					lbl25 := ctx.ReserveLabel()
					ctx.EmitCmpRegImm32(d209.Reg, 0)
					ctx.EmitJump(CondNotEqual, lbl24)
					ctx.EmitJmp(lbl25)
					ctx.MarkLabel(lbl24)
					ctx.EmitJmp(lbl12)
					ctx.MarkLabel(lbl25)
					ctx.EmitJmp(lbl13)
					ps213 := PhiState{General: true}
					ps213.OverlayValues = make([]JITValueDesc, 213)
					ps213.OverlayValues[1] = d1
					ps213.OverlayValues[2] = d2
					ps213.OverlayValues[3] = d3
					ps213.OverlayValues[4] = d4
					ps213.OverlayValues[5] = d5
					ps213.OverlayValues[6] = d6
					ps213.OverlayValues[7] = d7
					ps213.OverlayValues[8] = d8
					ps213.OverlayValues[9] = d9
					ps213.OverlayValues[10] = d10
					ps213.OverlayValues[11] = d11
					ps213.OverlayValues[28] = d28
					ps213.OverlayValues[29] = d29
					ps213.OverlayValues[30] = d30
					ps213.OverlayValues[31] = d31
					ps213.OverlayValues[32] = d32
					ps213.OverlayValues[33] = d33
					ps213.OverlayValues[34] = d34
					ps213.OverlayValues[35] = d35
					ps213.OverlayValues[36] = d36
					ps213.OverlayValues[37] = d37
					ps213.OverlayValues[38] = d38
					ps213.OverlayValues[39] = d39
					ps213.OverlayValues[40] = d40
					ps213.OverlayValues[71] = d71
					ps213.OverlayValues[72] = d72
					ps213.OverlayValues[73] = d73
					ps213.OverlayValues[74] = d74
					ps213.OverlayValues[75] = d75
					ps213.OverlayValues[110] = d110
					ps213.OverlayValues[111] = d111
					ps213.OverlayValues[112] = d112
					ps213.OverlayValues[150] = d150
					ps213.OverlayValues[151] = d151
					ps213.OverlayValues[152] = d152
					ps213.OverlayValues[155] = d155
					ps213.OverlayValues[195] = d195
					ps213.OverlayValues[196] = d196
					ps213.OverlayValues[197] = d197
					ps213.OverlayValues[198] = d198
					ps213.OverlayValues[199] = d199
					ps213.OverlayValues[201] = d201
					ps213.OverlayValues[202] = d202
					ps213.OverlayValues[203] = d203
					ps213.OverlayValues[205] = d205
					ps213.OverlayValues[206] = d206
					ps213.OverlayValues[207] = d207
					ps213.OverlayValues[208] = d208
					ps213.OverlayValues[209] = d209
					ps213.OverlayValues[212] = d212
					ps214 := PhiState{General: true}
					ps214.OverlayValues = make([]JITValueDesc, 213)
					ps214.OverlayValues[1] = d1
					ps214.OverlayValues[2] = d2
					ps214.OverlayValues[3] = d3
					ps214.OverlayValues[4] = d4
					ps214.OverlayValues[5] = d5
					ps214.OverlayValues[6] = d6
					ps214.OverlayValues[7] = d7
					ps214.OverlayValues[8] = d8
					ps214.OverlayValues[9] = d9
					ps214.OverlayValues[10] = d10
					ps214.OverlayValues[11] = d11
					ps214.OverlayValues[28] = d28
					ps214.OverlayValues[29] = d29
					ps214.OverlayValues[30] = d30
					ps214.OverlayValues[31] = d31
					ps214.OverlayValues[32] = d32
					ps214.OverlayValues[33] = d33
					ps214.OverlayValues[34] = d34
					ps214.OverlayValues[35] = d35
					ps214.OverlayValues[36] = d36
					ps214.OverlayValues[37] = d37
					ps214.OverlayValues[38] = d38
					ps214.OverlayValues[39] = d39
					ps214.OverlayValues[40] = d40
					ps214.OverlayValues[71] = d71
					ps214.OverlayValues[72] = d72
					ps214.OverlayValues[73] = d73
					ps214.OverlayValues[74] = d74
					ps214.OverlayValues[75] = d75
					ps214.OverlayValues[110] = d110
					ps214.OverlayValues[111] = d111
					ps214.OverlayValues[112] = d112
					ps214.OverlayValues[150] = d150
					ps214.OverlayValues[151] = d151
					ps214.OverlayValues[152] = d152
					ps214.OverlayValues[155] = d155
					ps214.OverlayValues[195] = d195
					ps214.OverlayValues[196] = d196
					ps214.OverlayValues[197] = d197
					ps214.OverlayValues[198] = d198
					ps214.OverlayValues[199] = d199
					ps214.OverlayValues[201] = d201
					ps214.OverlayValues[202] = d202
					ps214.OverlayValues[203] = d203
					ps214.OverlayValues[205] = d205
					ps214.OverlayValues[206] = d206
					ps214.OverlayValues[207] = d207
					ps214.OverlayValues[208] = d208
					ps214.OverlayValues[209] = d209
					ps214.OverlayValues[212] = d212
					snap215 := d1
					snap216 := d2
					snap217 := d3
					snap218 := d4
					snap219 := d5
					snap220 := d6
					snap221 := d7
					snap222 := d8
					snap223 := d9
					snap224 := d10
					snap225 := d11
					snap226 := d28
					snap227 := d29
					snap228 := d30
					snap229 := d31
					snap230 := d32
					snap231 := d33
					snap232 := d34
					snap233 := d35
					snap234 := d36
					snap235 := d37
					snap236 := d38
					snap237 := d39
					snap238 := d40
					snap239 := d71
					snap240 := d72
					snap241 := d73
					snap242 := d74
					snap243 := d75
					snap244 := d110
					snap245 := d111
					snap246 := d112
					snap247 := d150
					snap248 := d151
					snap249 := d152
					snap250 := d155
					snap251 := d195
					snap252 := d196
					snap253 := d197
					snap254 := d198
					snap255 := d199
					snap256 := d201
					snap257 := d202
					snap258 := d203
					snap259 := d205
					snap260 := d206
					snap261 := d207
					snap262 := d208
					snap263 := d209
					snap264 := d212
					alloc265 := ctx.SnapshotAllocState()
					if !bbs[12].Rendered {
						bbs[12].RenderPS(ps214)
					}
					ctx.RestoreAllocState(alloc265)
					d1 = snap215
					d2 = snap216
					d3 = snap217
					d4 = snap218
					d5 = snap219
					d6 = snap220
					d7 = snap221
					d8 = snap222
					d9 = snap223
					d10 = snap224
					d11 = snap225
					d28 = snap226
					d29 = snap227
					d30 = snap228
					d31 = snap229
					d32 = snap230
					d33 = snap231
					d34 = snap232
					d35 = snap233
					d36 = snap234
					d37 = snap235
					d38 = snap236
					d39 = snap237
					d40 = snap238
					d71 = snap239
					d72 = snap240
					d73 = snap241
					d74 = snap242
					d75 = snap243
					d110 = snap244
					d111 = snap245
					d112 = snap246
					d150 = snap247
					d151 = snap248
					d152 = snap249
					d155 = snap250
					d195 = snap251
					d196 = snap252
					d197 = snap253
					d198 = snap254
					d199 = snap255
					d201 = snap256
					d202 = snap257
					d203 = snap258
					d205 = snap259
					d206 = snap260
					d207 = snap261
					d208 = snap262
					d209 = snap263
					d212 = snap264
					if !bbs[11].Rendered {
						return bbs[11].RenderPS(ps213)
					}
					return result
					ctx.FreeDesc(&d208)
					return result
				}
				bbs[11].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[11].VisitCount >= 0 {
							ps.General = true
							return bbs[11].RenderPS(ps)
						}
					}
					bbs[11].VisitCount++
					if ps.General {
						if bbs[11].Rendered {
							ctx.EmitJmp(lbl12)
							return result
						}
						bbs[11].Rendered = true
						bbs[11].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_11 = bbs[11].Address
						ctx.MarkLabel(lbl12)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					ctx.ReclaimUntrackedRegs()
					d266 = JITValueDesc{Loc: LocImm, Type: tagNil, Imm: NewNil()}
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d266)
					d267 = ctx.EmitSliceElementAddress(&d38, &d2, int32(16))
					ctx.EmitStoreScmerAt(&d267, &d266)
					ctx.FreeDesc(&d267)
					ctx.FreeDesc(&d266)
					ctx.EnsureDesc(&d2)
					ctx.EnsureDesc(&d2)
					var d268 JITValueDesc
					if d2.Loc == LocImm {
						d268 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d2.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d2.Reg)
						ctx.EmitMovRegReg(scratch, d2.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d268 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d268)
					}
					if d268.Loc == LocReg && d2.Loc == LocReg && d268.Reg == d2.Reg {
						ctx.TransferReg(d2.Reg)
						d2.Loc = LocNone
					}
					ctx.EnsureDesc(&d268)
					ctx.EmitStoreToStack(d268, int32(bbs[10].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d268)
					ctx.FreeDesc(&d2)
					if ps.General {
					}
					ps269 := PhiState{General: ps.General}
					ps269.OverlayValues = make([]JITValueDesc, 269)
					ps269.OverlayValues[1] = d1
					ps269.OverlayValues[2] = d2
					ps269.OverlayValues[3] = d3
					ps269.OverlayValues[4] = d4
					ps269.OverlayValues[5] = d5
					ps269.OverlayValues[6] = d6
					ps269.OverlayValues[7] = d7
					ps269.OverlayValues[8] = d8
					ps269.OverlayValues[9] = d9
					ps269.OverlayValues[10] = d10
					ps269.OverlayValues[11] = d11
					ps269.OverlayValues[28] = d28
					ps269.OverlayValues[29] = d29
					ps269.OverlayValues[30] = d30
					ps269.OverlayValues[31] = d31
					ps269.OverlayValues[32] = d32
					ps269.OverlayValues[33] = d33
					ps269.OverlayValues[34] = d34
					ps269.OverlayValues[35] = d35
					ps269.OverlayValues[36] = d36
					ps269.OverlayValues[37] = d37
					ps269.OverlayValues[38] = d38
					ps269.OverlayValues[39] = d39
					ps269.OverlayValues[40] = d40
					ps269.OverlayValues[71] = d71
					ps269.OverlayValues[72] = d72
					ps269.OverlayValues[73] = d73
					ps269.OverlayValues[74] = d74
					ps269.OverlayValues[75] = d75
					ps269.OverlayValues[110] = d110
					ps269.OverlayValues[111] = d111
					ps269.OverlayValues[112] = d112
					ps269.OverlayValues[150] = d150
					ps269.OverlayValues[151] = d151
					ps269.OverlayValues[152] = d152
					ps269.OverlayValues[155] = d155
					ps269.OverlayValues[195] = d195
					ps269.OverlayValues[196] = d196
					ps269.OverlayValues[197] = d197
					ps269.OverlayValues[198] = d198
					ps269.OverlayValues[199] = d199
					ps269.OverlayValues[201] = d201
					ps269.OverlayValues[202] = d202
					ps269.OverlayValues[203] = d203
					ps269.OverlayValues[205] = d205
					ps269.OverlayValues[206] = d206
					ps269.OverlayValues[207] = d207
					ps269.OverlayValues[208] = d208
					ps269.OverlayValues[209] = d209
					ps269.OverlayValues[212] = d212
					ps269.OverlayValues[266] = d266
					ps269.OverlayValues[267] = d267
					ps269.OverlayValues[268] = d268
					ps269.PhiValues = make([]JITValueDesc, 1)
					if ps269.General && bbs[10].Rendered {
						ctx.EmitJmp(lbl11)
						return result
					}
					return bbs[10].RenderPS(ps269)
					return result
				}
				bbs[12].RenderPS = func(ps PhiState) JITValueDesc {
					if !ps.General {
						if bbs[12].VisitCount >= 0 {
							ps.General = true
							return bbs[12].RenderPS(ps)
						}
					}
					bbs[12].VisitCount++
					if ps.General {
						if bbs[12].Rendered {
							ctx.EmitJmp(lbl13)
							return result
						}
						bbs[12].Rendered = true
						bbs[12].Address = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
						bbpos_0_12 = bbs[12].Address
						ctx.MarkLabel(lbl13)
						ctx.ResolveFixups()
					}
					d1 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(0)}
					d2 = JITValueDesc{Loc: LocStack, Type: tagInt, StackOff: int32(phiBase0) + int32(16)}
					if !ps.General && len(ps.OverlayValues) > 1 && ps.OverlayValues[1].Loc != LocNone {
						d1 = ps.OverlayValues[1]
					}
					if !ps.General && len(ps.OverlayValues) > 2 && ps.OverlayValues[2].Loc != LocNone {
						d2 = ps.OverlayValues[2]
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
					if len(ps.OverlayValues) > 6 && ps.OverlayValues[6].Loc != LocNone {
						d6 = ps.OverlayValues[6]
					}
					if len(ps.OverlayValues) > 7 && ps.OverlayValues[7].Loc != LocNone {
						d7 = ps.OverlayValues[7]
					}
					if len(ps.OverlayValues) > 8 && ps.OverlayValues[8].Loc != LocNone {
						d8 = ps.OverlayValues[8]
					}
					if len(ps.OverlayValues) > 9 && ps.OverlayValues[9].Loc != LocNone {
						d9 = ps.OverlayValues[9]
					}
					if len(ps.OverlayValues) > 10 && ps.OverlayValues[10].Loc != LocNone {
						d10 = ps.OverlayValues[10]
					}
					if len(ps.OverlayValues) > 11 && ps.OverlayValues[11].Loc != LocNone {
						d11 = ps.OverlayValues[11]
					}
					if len(ps.OverlayValues) > 28 && ps.OverlayValues[28].Loc != LocNone {
						d28 = ps.OverlayValues[28]
					}
					if len(ps.OverlayValues) > 29 && ps.OverlayValues[29].Loc != LocNone {
						d29 = ps.OverlayValues[29]
					}
					if len(ps.OverlayValues) > 30 && ps.OverlayValues[30].Loc != LocNone {
						d30 = ps.OverlayValues[30]
					}
					if len(ps.OverlayValues) > 31 && ps.OverlayValues[31].Loc != LocNone {
						d31 = ps.OverlayValues[31]
					}
					if len(ps.OverlayValues) > 32 && ps.OverlayValues[32].Loc != LocNone {
						d32 = ps.OverlayValues[32]
					}
					if len(ps.OverlayValues) > 33 && ps.OverlayValues[33].Loc != LocNone {
						d33 = ps.OverlayValues[33]
					}
					if len(ps.OverlayValues) > 34 && ps.OverlayValues[34].Loc != LocNone {
						d34 = ps.OverlayValues[34]
					}
					if len(ps.OverlayValues) > 35 && ps.OverlayValues[35].Loc != LocNone {
						d35 = ps.OverlayValues[35]
					}
					if len(ps.OverlayValues) > 36 && ps.OverlayValues[36].Loc != LocNone {
						d36 = ps.OverlayValues[36]
					}
					if len(ps.OverlayValues) > 37 && ps.OverlayValues[37].Loc != LocNone {
						d37 = ps.OverlayValues[37]
					}
					if len(ps.OverlayValues) > 38 && ps.OverlayValues[38].Loc != LocNone {
						d38 = ps.OverlayValues[38]
					}
					if len(ps.OverlayValues) > 39 && ps.OverlayValues[39].Loc != LocNone {
						d39 = ps.OverlayValues[39]
					}
					if len(ps.OverlayValues) > 40 && ps.OverlayValues[40].Loc != LocNone {
						d40 = ps.OverlayValues[40]
					}
					if len(ps.OverlayValues) > 71 && ps.OverlayValues[71].Loc != LocNone {
						d71 = ps.OverlayValues[71]
					}
					if len(ps.OverlayValues) > 72 && ps.OverlayValues[72].Loc != LocNone {
						d72 = ps.OverlayValues[72]
					}
					if len(ps.OverlayValues) > 73 && ps.OverlayValues[73].Loc != LocNone {
						d73 = ps.OverlayValues[73]
					}
					if len(ps.OverlayValues) > 74 && ps.OverlayValues[74].Loc != LocNone {
						d74 = ps.OverlayValues[74]
					}
					if len(ps.OverlayValues) > 75 && ps.OverlayValues[75].Loc != LocNone {
						d75 = ps.OverlayValues[75]
					}
					if len(ps.OverlayValues) > 110 && ps.OverlayValues[110].Loc != LocNone {
						d110 = ps.OverlayValues[110]
					}
					if len(ps.OverlayValues) > 111 && ps.OverlayValues[111].Loc != LocNone {
						d111 = ps.OverlayValues[111]
					}
					if len(ps.OverlayValues) > 112 && ps.OverlayValues[112].Loc != LocNone {
						d112 = ps.OverlayValues[112]
					}
					if len(ps.OverlayValues) > 150 && ps.OverlayValues[150].Loc != LocNone {
						d150 = ps.OverlayValues[150]
					}
					if len(ps.OverlayValues) > 151 && ps.OverlayValues[151].Loc != LocNone {
						d151 = ps.OverlayValues[151]
					}
					if len(ps.OverlayValues) > 152 && ps.OverlayValues[152].Loc != LocNone {
						d152 = ps.OverlayValues[152]
					}
					if len(ps.OverlayValues) > 155 && ps.OverlayValues[155].Loc != LocNone {
						d155 = ps.OverlayValues[155]
					}
					if len(ps.OverlayValues) > 195 && ps.OverlayValues[195].Loc != LocNone {
						d195 = ps.OverlayValues[195]
					}
					if len(ps.OverlayValues) > 196 && ps.OverlayValues[196].Loc != LocNone {
						d196 = ps.OverlayValues[196]
					}
					if len(ps.OverlayValues) > 197 && ps.OverlayValues[197].Loc != LocNone {
						d197 = ps.OverlayValues[197]
					}
					if len(ps.OverlayValues) > 198 && ps.OverlayValues[198].Loc != LocNone {
						d198 = ps.OverlayValues[198]
					}
					if len(ps.OverlayValues) > 199 && ps.OverlayValues[199].Loc != LocNone {
						d199 = ps.OverlayValues[199]
					}
					if len(ps.OverlayValues) > 201 && ps.OverlayValues[201].Loc != LocNone {
						d201 = ps.OverlayValues[201]
					}
					if len(ps.OverlayValues) > 202 && ps.OverlayValues[202].Loc != LocNone {
						d202 = ps.OverlayValues[202]
					}
					if len(ps.OverlayValues) > 203 && ps.OverlayValues[203].Loc != LocNone {
						d203 = ps.OverlayValues[203]
					}
					if len(ps.OverlayValues) > 205 && ps.OverlayValues[205].Loc != LocNone {
						d205 = ps.OverlayValues[205]
					}
					if len(ps.OverlayValues) > 206 && ps.OverlayValues[206].Loc != LocNone {
						d206 = ps.OverlayValues[206]
					}
					if len(ps.OverlayValues) > 207 && ps.OverlayValues[207].Loc != LocNone {
						d207 = ps.OverlayValues[207]
					}
					if len(ps.OverlayValues) > 208 && ps.OverlayValues[208].Loc != LocNone {
						d208 = ps.OverlayValues[208]
					}
					if len(ps.OverlayValues) > 209 && ps.OverlayValues[209].Loc != LocNone {
						d209 = ps.OverlayValues[209]
					}
					if len(ps.OverlayValues) > 212 && ps.OverlayValues[212].Loc != LocNone {
						d212 = ps.OverlayValues[212]
					}
					if len(ps.OverlayValues) > 266 && ps.OverlayValues[266].Loc != LocNone {
						d266 = ps.OverlayValues[266]
					}
					if len(ps.OverlayValues) > 267 && ps.OverlayValues[267].Loc != LocNone {
						d267 = ps.OverlayValues[267]
					}
					if len(ps.OverlayValues) > 268 && ps.OverlayValues[268].Loc != LocNone {
						d268 = ps.OverlayValues[268]
					}
					ctx.ReclaimUntrackedRegs()
					d270 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					d272 = ctx.EmitSliceElementAddress(&d4, &d270, 16)
					ctx.EnsureDesc(&d272)
					r23 := ctx.AllocRegExcept(d272.Reg)
					ctx.EmitMovRegMem(r23, d272.Reg, 8)
					ctx.EmitMovRegMem(d272.Reg, d272.Reg, 0)
					d271 = JITValueDesc{Loc: LocRegPair, Type: JITTypeUnknown, Reg: d272.Reg, Reg2: r23}
					ctx.BindReg(d272.Reg, &d271)
					ctx.BindReg(r23, &d271)
					var d273 JITValueDesc
					if d271.Loc == LocImm {
						d273 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d271.Imm.Int())}
					} else if d271.Type == tagInt && d271.Loc == LocRegPair {
						ctx.FreeReg(d271.Reg)
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d271.Reg2}
						ctx.BindReg(d271.Reg2, &d273)
						ctx.BindReg(d271.Reg2, &d273)
					} else if d271.Type == tagInt && d271.Loc == LocReg {
						d273 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: d271.Reg}
						ctx.BindReg(d271.Reg, &d273)
						ctx.BindReg(d271.Reg, &d273)
					} else {
						d273 = ctx.EmitGoCallScalar(GoFuncAddr(Scmer.Int), []JITValueDesc{d271}, 1)
						d273.Type = tagInt
						ctx.BindReg(d273.Reg, &d273)
					}
					ctx.FreeDesc(&d271)
					ctx.EnsureDesc(&d273)
					ctx.EnsureDesc(&d273)
					var d274 JITValueDesc
					if d273.Loc == LocImm {
						d274 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d273.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d273.Reg)
						ctx.EmitMovRegReg(scratch, d273.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d274 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d274)
					}
					if d274.Loc == LocReg && d273.Loc == LocReg && d274.Reg == d273.Reg {
						ctx.TransferReg(d273.Reg)
						d273.Loc = LocNone
					}
					ctx.FreeDesc(&d273)
					ctx.EnsureDesc(&d274)
					d275 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(1)}
					ctx.EnsureDesc(&d274)
					d276 = ctx.EmitSliceElementAddress(&d4, &d275, int32(16))
					ctx.EmitStoreScmerAt(&d276, &d274)
					ctx.FreeDesc(&d276)
					ctx.EnsureDesc(&d5)
					ctx.EnsureDesc(&d38)
					d277 = d5
					_ = d277
					ctx.StabilizeDescForControlFlow(&d277)
					d278 = d38
					_ = d278
					ctx.StabilizeDescForControlFlow(&d278)
					r24 := d38.Loc == LocReg || d38.Loc == LocRegPair || d38.Loc == LocRegTriple
					r25 := d38.Reg
					if r24 {
						ctx.ProtectReg(r25)
					}
					r26 := d38.Loc == LocRegPair || d38.Loc == LocRegTriple
					r27 := d38.Reg2
					if r26 {
						ctx.ProtectReg(r27)
					}
					r28 := d38.Loc == LocRegTriple
					r29 := d38.Reg3
					if r28 {
						ctx.ProtectReg(r29)
					}
					bbpos_1_0 := int32(-1)
					_ = bbpos_1_0
					bbpos_1_0 = int32(uintptr(ctx.Ptr) - uintptr(ctx.Start))
					ctx.ReclaimUntrackedRegs()
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d277)
					ctx.EnsureDesc(&d277)
					ctx.EnsureDesc(&d277)
					if d277.Loc == LocImm {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d277.Type, Reg: ctx.AllocReg(), Reg2: ctx.AllocReg()}
						if d277.Imm.GetTag() == tagBool {
							ctx.EmitMakeBool(tmpPair, d277)
						} else if d277.Imm.GetTag() == tagInt {
							ctx.EmitMakeInt(tmpPair, d277)
						} else if d277.Imm.GetTag() == tagFloat {
							ctx.EmitMakeFloat(tmpPair, d277)
						} else if d277.Imm.GetTag() == tagNil {
							ctx.EmitMakeNil(tmpPair)
						} else {
							ptrWord, auxWord := d277.Imm.RawWords()
							ctx.EmitMovRegImm64(tmpPair.Reg, uint64(ptrWord))
							ctx.EmitMovRegImm64(tmpPair.Reg2, auxWord)
						}
						d277 = tmpPair
					} else if d277.Loc == LocReg {
						tmpPair := JITValueDesc{Loc: LocRegPair, Type: d277.Type, Reg: ctx.AllocRegExcept(d277.Reg), Reg2: ctx.AllocRegExcept(d277.Reg)}
						switch d277.Type {
						case tagBool:
							ctx.EmitMakeBool(tmpPair, d277)
						case tagInt:
							ctx.EmitMakeInt(tmpPair, d277)
						case tagFloat:
							ctx.EmitMakeFloat(tmpPair, d277)
						default:
							panic("jit: generic call arg scalar type unknown for 2-word value")
						}
						ctx.FreeDesc(&d277)
						d277 = tmpPair
					}
					if d277.Loc != LocRegPair && d277.Loc != LocStackPair {
						panic("jit: generic call arg expects 2-word value (ApplyEx arg0)")
					}
					ctx.EnsureDesc(&d278)
					ctx.EnsureDesc(&d278)
					ctx.EnsureDesc(&d278)
					if d278.Loc != LocRegTriple && d278.Loc != LocStackTriple {
						panic("jit: generic call arg expects 3-word Go slice (ApplyEx arg1)")
					}
					d279 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(int64(uintptr(unsafe.Pointer(&Globalenv)))), NoHeapPointer: true, Rooted: true}
					if d279.Loc == LocRegPair || d279.Loc == LocStackPair || d279.Loc == LocRegTriple || d279.Loc == LocStackTriple {
						panic("jit: generic call arg expects 1-word value")
					}
					ctx.SyncDesc(&d277)
					ctx.SyncDesc(&d278)
					ctx.SyncDesc(&d279)
					d280 = ctx.EmitGoCallScalar(GoFuncAddr(ApplyEx), []JITValueDesc{d277, d278, d279}, 2)
					ctx.BindReg(d280.Reg, &d280)
					ctx.BindReg(d280.Reg2, &d280)
					ctx.ReclaimUntrackedRegs()
					ctx.EnsureDesc(&d280)
					if r24 {
						ctx.UnprotectReg(r25)
					}
					if r26 {
						ctx.UnprotectReg(r27)
					}
					if r28 {
						ctx.UnprotectReg(r29)
					}
					ctx.FreeDesc(&d5)
					ctx.EnsureDesc(&d1)
					ctx.EnsureDesc(&d1)
					var d281 JITValueDesc
					if d1.Loc == LocImm {
						d281 = JITValueDesc{Loc: LocImm, Type: tagInt, Imm: NewInt(d1.Imm.Int() + 1)}
					} else {
						scratch := ctx.AllocRegExcept(d1.Reg)
						ctx.EmitMovRegReg(scratch, d1.Reg)
						ctx.EmitAddRegImm32(scratch, int32(1))
						d281 = JITValueDesc{Loc: LocReg, Type: tagInt, Reg: scratch}
						ctx.BindReg(scratch, &d281)
					}
					if d281.Loc == LocReg && d1.Loc == LocReg && d281.Reg == d1.Reg {
						ctx.TransferReg(d1.Reg)
						d1.Loc = LocNone
					}
					ctx.EnsureDesc(&d281)
					ctx.EmitStoreToStack(d281, int32(bbs[7].PhiBase)+int32(0))
					ctx.StabilizeDescForControlFlow(&d281)
					ctx.FreeDesc(&d1)
					if ps.General {
					}
					ps282 := PhiState{General: ps.General}
					ps282.OverlayValues = make([]JITValueDesc, 282)
					ps282.OverlayValues[1] = d1
					ps282.OverlayValues[2] = d2
					ps282.OverlayValues[3] = d3
					ps282.OverlayValues[4] = d4
					ps282.OverlayValues[5] = d5
					ps282.OverlayValues[6] = d6
					ps282.OverlayValues[7] = d7
					ps282.OverlayValues[8] = d8
					ps282.OverlayValues[9] = d9
					ps282.OverlayValues[10] = d10
					ps282.OverlayValues[11] = d11
					ps282.OverlayValues[28] = d28
					ps282.OverlayValues[29] = d29
					ps282.OverlayValues[30] = d30
					ps282.OverlayValues[31] = d31
					ps282.OverlayValues[32] = d32
					ps282.OverlayValues[33] = d33
					ps282.OverlayValues[34] = d34
					ps282.OverlayValues[35] = d35
					ps282.OverlayValues[36] = d36
					ps282.OverlayValues[37] = d37
					ps282.OverlayValues[38] = d38
					ps282.OverlayValues[39] = d39
					ps282.OverlayValues[40] = d40
					ps282.OverlayValues[71] = d71
					ps282.OverlayValues[72] = d72
					ps282.OverlayValues[73] = d73
					ps282.OverlayValues[74] = d74
					ps282.OverlayValues[75] = d75
					ps282.OverlayValues[110] = d110
					ps282.OverlayValues[111] = d111
					ps282.OverlayValues[112] = d112
					ps282.OverlayValues[150] = d150
					ps282.OverlayValues[151] = d151
					ps282.OverlayValues[152] = d152
					ps282.OverlayValues[155] = d155
					ps282.OverlayValues[195] = d195
					ps282.OverlayValues[196] = d196
					ps282.OverlayValues[197] = d197
					ps282.OverlayValues[198] = d198
					ps282.OverlayValues[199] = d199
					ps282.OverlayValues[201] = d201
					ps282.OverlayValues[202] = d202
					ps282.OverlayValues[203] = d203
					ps282.OverlayValues[205] = d205
					ps282.OverlayValues[206] = d206
					ps282.OverlayValues[207] = d207
					ps282.OverlayValues[208] = d208
					ps282.OverlayValues[209] = d209
					ps282.OverlayValues[212] = d212
					ps282.OverlayValues[266] = d266
					ps282.OverlayValues[267] = d267
					ps282.OverlayValues[268] = d268
					ps282.OverlayValues[270] = d270
					ps282.OverlayValues[271] = d271
					ps282.OverlayValues[272] = d272
					ps282.OverlayValues[273] = d273
					ps282.OverlayValues[274] = d274
					ps282.OverlayValues[275] = d275
					ps282.OverlayValues[276] = d276
					ps282.OverlayValues[277] = d277
					ps282.OverlayValues[278] = d278
					ps282.OverlayValues[279] = d279
					ps282.OverlayValues[280] = d280
					ps282.OverlayValues[281] = d281
					ps282.PhiValues = make([]JITValueDesc, 1)
					if ps282.General && bbs[7].Rendered {
						ctx.EmitJmp(lbl8)
						return result
					}
					return bbs[7].RenderPS(ps282)
					return result
				}
				for i := range args {
					ctx.StabilizeDescForControlFlow(&args[i])
				}
				ps283 := PhiState{General: false}
				_ = bbs[0].RenderPS(ps283)
				ctx.MarkLabel(lbl0)
				ctx.ResolveFixups()
				ctx.FreeStack(int32(32))
				return result
			},
			JITVirtualArgs: true,
			JITInlineCost:  62,
		},
	})
}
